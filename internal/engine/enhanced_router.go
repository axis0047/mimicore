package engine

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	// High-performance JSON library
	json "github.com/goccy/go-json"
	// For parallel execution
	"golang.org/x/sync/errgroup"

	v2 "github.com/axis0047/mockingGOD/internal/adapters/v2"
	"github.com/axis0047/mockingGOD/internal/ir"
	"github.com/axis0047/mockingGOD/internal/services/wasm"
	"github.com/axis0047/mockingGOD/internal/utils"
)

// Shared Client for Connection Pooling
var sharedHTTPClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
}

type EnhancedRouter struct {
	Routes []ir.EnhancedRoute
	Wasm   *wasm.Manager
}

func (r *EnhancedRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route, params := r.matchRoute(req)
	if route == nil {
		log.Printf("No route matched for %s %s", req.Method, req.URL.Path)
		http.NotFound(w, req)
		return
	}

	log.Printf("Matched route: %s %s", route.Method, strings.Join(route.Path.Segments, "/"))

	// Initialize Thread-Safe Context
	ctx := NewSafeContext()

	// Seed context with path parameters
	for k, v := range params {
		ctx.Set(k, v)
	}

	// Phase 1: Validation
	if err := r.runValidation(route, req); err != nil {
		log.Printf("Validation failed: %v", err)
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Phase 2: Transformation (Passes SafeContext)
	if err := r.runTransformations(route, req, ctx); err != nil {
		log.Printf("Transformation failed: %v", err)
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Phase 3: Response building
	resp := make(map[string]any)
	for _, rule := range route.Response {
		// Check for Template Strings {{...}}
		if static, ok := rule.Source.(ir.StaticValue); ok {
			if strVal, isStr := static.Value.(string); isStr && strings.Contains(strVal, "{{") {
				// Resolve template using SafeContext
				finalVal := r.resolveTemplate(strVal, ctx)
				utils.SetNested(resp, rule.Target, finalVal)
				continue
			}
		}

		// Standard Resolution (SafeContext implements ContextAccessor interface)
		val, err := rule.Source.Resolve(ctx)
		if err != nil {
			log.Printf("error resolving %s: %v", rule.Target, err)
			val = "error: " + err.Error()
		}
		utils.SetNested(resp, rule.Target, val)
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (r *EnhancedRouter) matchRoute(req *http.Request) (*ir.EnhancedRoute, map[string]string) {
	path := strings.Trim(req.URL.Path, "/")
	segments := strings.Split(path, "/")

	for i := range r.Routes {
		route := &r.Routes[i]
		if route.Method != req.Method {
			continue
		}
		params := make(map[string]string)
		if matchSegments(route.Path.Segments, segments, params) {
			return route, params
		}
	}
	return nil, nil
}

func (r *EnhancedRouter) runValidation(route *ir.EnhancedRoute, req *http.Request) error {
	if len(route.ValidationSteps) == 0 {
		return nil
	}

	for _, step := range route.ValidationSteps {
		switch step.Type {
		case "header":
			rules, ok := step.Rules.(v2.ValidationRule)
			if !ok {
				if ptr, okPtr := step.Rules.(*v2.ValidationRule); okPtr {
					rules = *ptr
				} else {
					return fmt.Errorf("invalid rule type for header %s", step.Field)
				}
			}

			val := req.Header.Get(step.Field)

			if rules.Required && val == "" {
				return fmt.Errorf("header %q is required", step.Field)
			}

			if rules.Pattern != "" && val != "" {
				matched, err := regexp.MatchString(rules.Pattern, val)
				if err != nil {
					return fmt.Errorf("invalid regex for %s", step.Field)
				}
				if !matched {
					return fmt.Errorf("header %q value does not match pattern", step.Field)
				}
			}
		}
	}
	return nil
}

func (r *EnhancedRouter) runTransformations(route *ir.EnhancedRoute, req *http.Request, ctx *SafeContext) error {
	for _, step := range route.TransformSteps {
		switch step.Type {
		case "extract":
			if err := r.handleExtract(step, req, ctx); err != nil {
				return err
			}
		case "http_batch":
			// Execute HTTP calls in parallel
			if err := r.handleHTTPBatch(step, ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *EnhancedRouter) handleExtract(step ir.TransformStep, req *http.Request, ctx *SafeContext) error {
	extractCfg, ok := step.Config.(ir.ExtractTransform)
	if !ok {
		return fmt.Errorf("invalid config for extract step")
	}

	parts := strings.SplitN(extractCfg.From, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid 'from' format: %s", extractCfg.From)
	}
	source, key := parts[0], parts[1]

	var value string
	switch source {
	case "header":
		value = req.Header.Get(key)
	case "path":
		if val, exists := ctx.Get(key); exists {
			value = fmt.Sprintf("%v", val)
		}
	case "query":
		value = req.URL.Query().Get(key)
	}

	if value != "" {
		ctx.Set(extractCfg.To, value)
		log.Printf("EXTRACT: %s -> ctx[%s] = %s", extractCfg.From, extractCfg.To, value)
	}
	return nil
}

func (r *EnhancedRouter) handleHTTPBatch(step ir.TransformStep, ctx *SafeContext) error {
	batchCfg, ok := step.Config.(ir.ParallelHTTPConfig)
	if !ok {
		return fmt.Errorf("invalid http batch config")
	}

	// Create ErrorGroup to manage goroutines
	g, _ := errgroup.WithContext(context.Background())

	for _, cfg := range batchCfg.Calls {
		// Capture loop variable for closure
		httpConfig := cfg

		g.Go(func() error {
			return r.performSingleHTTP(httpConfig, ctx)
		})
	}

	// Wait for all to finish
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func (r *EnhancedRouter) performSingleHTTP(cfg ir.HTTPTransform, ctx *SafeContext) error {
	// 1. Resolve Templates safely
	finalURL := r.resolveTemplate(cfg.URL, ctx)

	req, err := http.NewRequest(cfg.Method, finalURL, nil)
	if err != nil {
		return err
	}

	for k, v := range cfg.Headers {
		req.Header.Set(k, r.resolveTemplate(v, ctx))
	}

	// 2. Setup Timeout
	timeout := 5000
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	ctxReq, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctxReq)

	// 3. Execute with Shared Client
	start := time.Now()
	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var result any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		result = string(bodyBytes)
	}

	// 4. Write Result SAFELY
	ctx.Set(cfg.Name, result)
	log.Printf("HTTP CALL (Async): %s -> Status %d (took %v)", finalURL, resp.StatusCode, time.Since(start))

	return nil
}

func (r *EnhancedRouter) resolveTemplate(template string, ctx *SafeContext) string {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	return re.ReplaceAllStringFunc(template, func(match string) string {
		content := strings.TrimSpace(match[2 : len(match)-2])

		// 1. Function Call Detection: name(args)
		funcRe := regexp.MustCompile(`^(\w+)\((.*)\)$`)
		funcMatch := funcRe.FindStringSubmatch(content)

		if len(funcMatch) == 3 {
			funcName := funcMatch[1]
			argsStr := funcMatch[2]

			if r.Wasm == nil {
				return "[error: no user code]"
			}

			var wasmArgs []uint64
			rawArgs := strings.Split(argsStr, ",")

			for _, rawArg := range rawArgs {
				rawArg = strings.TrimSpace(rawArg)
				if rawArg == "" {
					continue
				}

				resolved := r.resolveVariable(rawArg, ctx)
				strVal := fmt.Sprintf("%v", resolved)

				intVal, err := strconv.ParseUint(strVal, 10, 64)
				if err != nil {
					return "[error: arg not int]"
				}
				wasmArgs = append(wasmArgs, intVal)
			}

			result, err := r.Wasm.Call(funcName, wasmArgs...)
			if err != nil {
				return fmt.Sprintf("[error: %v]", err)
			}
			return fmt.Sprintf("%d", result)
		}

		// 2. Variable Lookup
		val := r.resolveVariable(content, ctx)
		return fmt.Sprintf("%v", val)
	})
}

func (r *EnhancedRouter) resolveVariable(key string, ctx *SafeContext) any {
	// Direct safe lookup
	if val, ok := ctx.Get(key); ok {
		return val
	}

	// Nested lookup
	if strings.Contains(key, ".") {
		parts := strings.Split(key, ".")
		if len(parts) == 2 {
			if root, ok := ctx.Get(parts[0]); ok {
				// We expect root to be a map
				if rootMap, ok := root.(map[string]any); ok {
					if subVal, ok := rootMap[parts[1]]; ok {
						return subVal
					}
				}
				// Handle generic interface{} map from JSON unmarshal
				if rootMap, ok := root.(map[string]interface{}); ok {
					if subVal, ok := rootMap[parts[1]]; ok {
						return subVal
					}
				}
			}
		}
	}
	return key
}
