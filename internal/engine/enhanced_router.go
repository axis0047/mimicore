package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/axis0047/mockingGOD/internal/ir"
	"github.com/axis0047/mockingGOD/internal/services/wasm"
	"github.com/axis0047/mockingGOD/internal/utils"
)

// EnhancedRouter handles v2 routes with validation, transformation, and WASM support
type EnhancedRouter struct {
	Routes []ir.EnhancedRoute
	Wasm   *wasm.Manager // <--- Added this field
}

func (r *EnhancedRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route, params := r.matchRoute(req)
	if route == nil {
		log.Printf("No route matched for %s %s", req.Method, req.URL.Path)
		http.NotFound(w, req)
		return
	}

	log.Printf("Matched route: %s %s", route.Method, strings.Join(route.Path.Segments, "/"))

	// Build context
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
		// Check if source is a template string "{{...}}"
		if static, ok := rule.Source.(ir.StaticValue); ok {
			if strVal, isStr := static.Value.(string); isStr && strings.Contains(strVal, "{{") {
				// Resolve template (includes WASM function calls)
				finalVal := r.resolveTemplate(strVal, ctx)
				utils.SetNested(resp, rule.Target, finalVal)
				continue
			}
		}

		// Fallback for standard resolution
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
	// Note: You can re-add the specific header/query validation logic here from previous steps
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

	ctx[extractCfg.To] = value
	return nil
}

func (r *EnhancedRouter) handleHTTP(step ir.TransformStep, ctx map[string]any) error {
	cfg, ok := step.Config.(ir.HTTPTransform)
	if !ok {
		return fmt.Errorf("invalid HTTP config")
	}

	finalURL := r.resolveTemplate(cfg.URL, ctx)
	req, err := http.NewRequest(cfg.Method, finalURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var result any
	json.Unmarshal(bodyBytes, &result)
	ctx[cfg.Name] = result

	return nil
}

// resolveTemplate resolves {{var}} and {{func(var)}}
func (r *EnhancedRouter) resolveTemplate(template string, ctx map[string]any) string {
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

				// Resolve arg (might be a variable name)
				resolved := r.resolveVariable(rawArg, ctx)

				// Convert to uint64
				intVal, err := strconv.ParseUint(fmt.Sprintf("%v", resolved), 10, 64)
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

func (r *EnhancedRouter) resolveVariable(key string, ctx map[string]any) any {
	// Direct
	if val, ok := ctx[key]; ok {
		return val
	}
	// Nested
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
	return key // Return literal if not found
}
