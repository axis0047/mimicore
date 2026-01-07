package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"golang.org/x/sync/errgroup"

	v2 "github.com/axis0047/mockingGOD/internal/adapters/v2"
	"github.com/axis0047/mockingGOD/internal/ir"
	"github.com/axis0047/mockingGOD/internal/services/wasm"
	"github.com/axis0047/mockingGOD/internal/utils"
)

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

	ctx := NewSafeContext()
	for k, v := range params {
		ctx.Set(k, v)
	}

	// Phase 1: Validation (Header + Schema)
	if err := r.runValidation(route, req); err != nil {
		log.Printf("Validation failed: %v", err)
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Phase 2: Transformation
	if err := r.runTransformations(route, req, ctx); err != nil {
		log.Printf("Transformation failed: %v", err)
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Phase 3: Response building
	resp := make(map[string]any)
	for _, rule := range route.Response {
		if static, ok := rule.Source.(ir.StaticValue); ok {
			if strVal, isStr := static.Value.(string); isStr && strings.Contains(strVal, "{{") {
				finalVal := r.resolveTemplate(strVal, ctx)
				utils.SetNested(resp, rule.Target, finalVal)
				continue
			}
		}
		val, err := rule.Source.Resolve(ctx)
		if err != nil {
			log.Printf("error resolving %s: %v", rule.Target, err)
			val = "error: " + err.Error()
		}
		utils.SetNested(resp, rule.Target, val)
	}

	// Phase 4: Latency Simulation
	if route.Delay.FixedMs > 0 || route.Delay.JitterMs > 0 {
		sleepTime := time.Duration(route.Delay.FixedMs) * time.Millisecond
		if route.Delay.JitterMs > 0 {
			randMs := time.Duration(rand.Intn(route.Delay.JitterMs)) * time.Millisecond
			sleepTime += randMs
		}
		time.Sleep(sleepTime)
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
				if err != nil || !matched {
					return fmt.Errorf("header %q validation failed", step.Field)
				}
			}

		case "body":
			// JSON Schema Validation
			schemaMap, ok := step.Rules.(map[string]interface{})
			if !ok {
				continue
			}

			// Read body
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return fmt.Errorf("failed to read body")
			}
			// Restore body
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// Compile Schema
			compiler := jsonschema.NewCompiler()
			schemaJSON, _ := json.Marshal(schemaMap)
			if err := compiler.AddResource("schema.json", bytes.NewReader(schemaJSON)); err != nil {
				return fmt.Errorf("invalid schema: %v", err)
			}
			schema, err := compiler.Compile("schema.json")
			if err != nil {
				return fmt.Errorf("schema compile error: %v", err)
			}

			// Validate
			var v interface{}
			if err := json.Unmarshal(bodyBytes, &v); err != nil {
				return fmt.Errorf("invalid json body")
			}
			if err := schema.Validate(v); err != nil {
				return fmt.Errorf("body schema validation failed: %v", err)
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
	source, key := "", ""
	if len(parts) == 2 {
		source, key = parts[0], parts[1]
	} else {
		// Handle "body" whole extraction
		source = parts[0]
	}

	var value any // Can be string or map/object
	var valStr string

	switch source {
	case "header":
		valStr = req.Header.Get(key)
		value = valStr
	case "path":
		if val, exists := ctx.Get(key); exists {
			value = val
			valStr = fmt.Sprintf("%v", val)
		}
	case "query":
		valStr = req.URL.Query().Get(key)
		value = valStr
	case "body":
		// Only support extracting full body as JSON for now
		if req.Body != nil {
			bodyBytes, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var jsonBody interface{}
			if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
				value = jsonBody
			} else {
				value = string(bodyBytes)
			}
			valStr = "<body>"
		}
	}

	if value != nil {
		ctx.Set(extractCfg.To, value)
		log.Printf("EXTRACT: %s -> ctx[%s]", extractCfg.From, extractCfg.To)
	}
	return nil
}

// handleHTTPBatch and performSingleHTTP remain mostly the same,
// ensuring they use goccy/go-json and sharedHTTPClient...
func (r *EnhancedRouter) handleHTTPBatch(step ir.TransformStep, ctx *SafeContext) error {
	batchCfg, ok := step.Config.(ir.ParallelHTTPConfig)
	if !ok {
		return fmt.Errorf("invalid http batch config")
	}

	g, _ := errgroup.WithContext(context.Background())
	for _, cfg := range batchCfg.Calls {
		c := cfg
		g.Go(func() error { return r.performSingleHTTP(c, ctx) })
	}
	return g.Wait()
}

func (r *EnhancedRouter) performSingleHTTP(cfg ir.HTTPTransform, ctx *SafeContext) error {
	finalURL := r.resolveTemplate(cfg.URL, ctx)
	req, err := http.NewRequest(cfg.Method, finalURL, nil)
	if err != nil {
		return err
	}

	for k, v := range cfg.Headers {
		req.Header.Set(k, r.resolveTemplate(v, ctx))
	}

	timeout := 5000
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	ctxReq, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctxReq)

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
	ctx.Set(cfg.Name, result)
	return nil
}

func (r *EnhancedRouter) resolveTemplate(template string, ctx *SafeContext) string {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	return re.ReplaceAllStringFunc(template, func(match string) string {
		content := strings.TrimSpace(match[2 : len(match)-2])

		funcRe := regexp.MustCompile(`^(\w+)\((.*)\)$`)
		funcMatch := funcRe.FindStringSubmatch(content)

		if len(funcMatch) == 3 {
			funcName := funcMatch[1]
			argVar := strings.TrimSpace(funcMatch[2])

			if r.Wasm == nil {
				return "[error: no user code]"
			}

			// Resolve argument
			val := r.resolveVariable(argVar, ctx)

			// Try Legacy Integer Strategy first (fastest)
			strVal := fmt.Sprintf("%v", val)
			if intVal, err := strconv.ParseUint(strVal, 10, 64); err == nil {
				res, err := r.Wasm.Call(funcName, intVal)
				if err != nil {
					return fmt.Sprintf("[error: %v]", err)
				}
				return fmt.Sprintf("%d", res)
			}

			// Fallback to JSON/String Strategy (malloc/free)
			// Marshal the input variable to JSON string
			jsonBytes, _ := json.Marshal(val)
			jsonStr := string(jsonBytes)

			// Remove quotes if it was just a string primitive to avoid double quoting "value"
			if s, ok := val.(string); ok {
				jsonStr = s
			}

			resStr, err := r.Wasm.CallJSON(funcName, jsonStr)
			if err != nil {
				return fmt.Sprintf("[error: %v]", err)
			}
			return resStr
		}

		val := r.resolveVariable(content, ctx)
		return fmt.Sprintf("%v", val)
	})
}

func (r *EnhancedRouter) resolveVariable(key string, ctx *SafeContext) any {
	if val, ok := ctx.Get(key); ok {
		return val
	}

	if strings.Contains(key, ".") {
		parts := strings.Split(key, ".")
		if len(parts) == 2 {
			if root, ok := ctx.Get(parts[0]); ok {
				if rootMap, ok := root.(map[string]any); ok {
					if subVal, ok := rootMap[parts[1]]; ok {
						return subVal
					}
				}
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
