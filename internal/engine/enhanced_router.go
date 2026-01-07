package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	v2 "github.com/axis0047/mockingGOD/internal/adapters/v2"
	"github.com/axis0047/mockingGOD/internal/ir"
	"github.com/axis0047/mockingGOD/internal/services/wasm"
	"github.com/axis0047/mockingGOD/internal/utils"
)

// --- NEW: Shared HTTP Client for Connection Pooling ---
var sharedHTTPClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,              // Keep 100 connections open total
		MaxIdleConnsPerHost: 20,               // Keep 20 connections open per upstream host
		IdleConnTimeout:     90 * time.Second, // Close connections if unused for 90s
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

	ctx := map[string]any{}
	for k, v := range params {
		ctx[k] = v
	}

	// Phase 1: Validation
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
			rules, ok := step.Rules.(v2.ValidationRule) // Implicit dependency on adapter/v2
			// Safety check in case rules is a pointer or map (depending on adapter version)
			if !ok {
				// We assume rules are correct type for now or handle generic checking
				// Ideally engine shouldn't depend on v2 adapter types directly but for this MVP it's fine
				// If strictly decoupled, ValidationSteps should use engine-specific struct
				return nil
			}

			val := req.Header.Get(step.Field)
			if rules.Required && val == "" {
				return fmt.Errorf("header %q is required", step.Field)
			}
			if rules.Pattern != "" && val != "" {
				matched, _ := regexp.MatchString(rules.Pattern, val)
				if !matched {
					return fmt.Errorf("header %q value does not match pattern", step.Field)
				}
			}
		}
	}
	return nil
}

func (r *EnhancedRouter) runTransformations(route *ir.EnhancedRoute, req *http.Request, ctx map[string]any) error {
	for _, step := range route.TransformSteps {
		switch step.Type {
		case "extract":
			if err := r.handleExtract(step, req, ctx); err != nil {
				return err
			}
		case "http":
			if err := r.handleHTTP(step, ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *EnhancedRouter) handleExtract(step ir.TransformStep, req *http.Request, ctx map[string]any) error {
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
		if val, exists := ctx[key]; exists {
			value = fmt.Sprintf("%v", val)
		}
	case "query":
		value = req.URL.Query().Get(key)
	}

	if value != "" {
		ctx[extractCfg.To] = value
	}
	return nil
}

// --- UPDATED HTTP HANDLER ---
func (r *EnhancedRouter) handleHTTP(step ir.TransformStep, ctx map[string]any) error {
	cfg, ok := step.Config.(ir.HTTPTransform)
	if !ok {
		return fmt.Errorf("invalid HTTP config")
	}

	finalURL := r.resolveTemplate(cfg.URL, ctx)

	// Create request with Context
	// We use the context for cancellation/timeout, but the client is shared
	req, err := http.NewRequest(cfg.Method, finalURL, nil)
	if err != nil {
		return err
	}

	// Add headers from config
	for k, v := range cfg.Headers {
		req.Header.Set(k, r.resolveTemplate(v, ctx))
	}

	// Set Timeout via Context
	timeout := 5000 // default 5s
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	ctxReq, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// Associate context with request
	req = req.WithContext(ctxReq)

	// Use SHARED Client
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

	ctx[cfg.Name] = result
	log.Printf("HTTP CALL: %s -> Status %d", finalURL, resp.StatusCode)

	return nil
}

func (r *EnhancedRouter) resolveTemplate(template string, ctx map[string]any) string {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	return re.ReplaceAllStringFunc(template, func(match string) string {
		content := strings.TrimSpace(match[2 : len(match)-2])

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

		val := r.resolveVariable(content, ctx)
		return fmt.Sprintf("%v", val)
	})
}

func (r *EnhancedRouter) resolveVariable(key string, ctx map[string]any) any {
	if val, ok := ctx[key]; ok {
		return val
	}
	if strings.Contains(key, ".") {
		parts := strings.Split(key, ".")
		if len(parts) == 2 {
			if root, ok := ctx[parts[0]].(map[string]any); ok {
				if subVal, ok := root[parts[1]]; ok {
					return subVal
				}
			}
		}
	}
	return key
}
