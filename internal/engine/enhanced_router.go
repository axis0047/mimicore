package engine

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	// Import the V2 adapter config for type assertion and utils for helpers
	v2 "github.com/axis0047/mockingGOD/internal/adapters/v2"
	"github.com/axis0047/mockingGOD/internal/ir"
	"github.com/axis0047/mockingGOD/internal/utils"
)

// EnhancedRouter handles v2 routes with validation and transformation
type EnhancedRouter struct {
	Routes []ir.EnhancedRoute
}

func (r *EnhancedRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route, params := r.matchRoute(req)
	if route == nil {
		log.Printf("No route matched for %s %s", req.Method, req.URL.Path)
		http.NotFound(w, req)
		return
	}

	log.Printf("Matched route: %s %s", route.Method, strings.Join(route.Path.Segments, "/"))

	// Build execution context, starting with path parameters
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

	// Phase 3: Response building (uses the responder utility)
	resp := BuildResponse(route.Response, ctx)

	// Use the helper function to write the response
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

// runValidation checks the request against the route's validation rules.
func (r *EnhancedRouter) runValidation(route *ir.EnhancedRoute, req *http.Request) error {
	if len(route.ValidationSteps) == 0 {
		return nil
	}
	log.Printf("Running %d validation steps", len(route.ValidationSteps))

	for _, step := range route.ValidationSteps {
		switch step.Type {
		case "header":
			// The rules are stored as an interface{}, so we need to assert the type.
			// This couples the engine to the v2 config struct, which is acceptable here.
			rules, ok := step.Rules.(v2.ValidationRule)
			if !ok {
				return fmt.Errorf("invalid validation rule type for header %s", step.Field)
			}

			headerValue := req.Header.Get(step.Field)

			// Check for required headers
			if rules.Required && headerValue == "" {
				return fmt.Errorf("header %q is required", step.Field)
			}

			// Check regex pattern if value is present
			if rules.Pattern != "" && headerValue != "" {
				matched, err := regexp.MatchString(rules.Pattern, headerValue)
				if err != nil {
					return fmt.Errorf("invalid pattern for header %q: %w", step.Field, err)
				}
				if !matched {
					return fmt.Errorf("header %q with value %q does not match pattern %s", step.Field, headerValue, rules.Pattern)
				}
			}
			// TODO: Add cases for "query" and "body" validation
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
		case "wasm":
			if err := r.handleWASM(step, ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// handleExtract pulls data from the request and puts it into the context.
func (r *EnhancedRouter) handleExtract(step ir.TransformStep, req *http.Request, ctx map[string]any) error {
	extractCfg, ok := step.Config.(ir.ExtractTransform)
	if !ok {
		return fmt.Errorf("invalid config for extract step")
	}

	log.Printf("Extracting from %q into variable %q", extractCfg.From, extractCfg.To)

	parts := strings.SplitN(extractCfg.From, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid 'from' format in extract: %s", extractCfg.From)
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
	// TODO: Add case for "body" extraction, which requires parsing the request body.
	default:
		return fmt.Errorf("unknown source in extract: %s", source)
	}

	// Store the extracted value in the context
	ctx[extractCfg.To] = value
	log.Printf("  ✓ Extracted value %q into context key %q", value, extractCfg.To)

	return nil
}

func (r *EnhancedRouter) handleHTTP(step ir.TransformStep, ctx map[string]any) error {
	log.Printf("HTTP step: %+v", step.Config)
	// TODO: Implement HTTP call logic
	// 1. Create http.Client with timeout.
	// 2. Create http.Request with method, url, headers, and body from step.Config.
	// 3. You may need to resolve variables in the URL/headers/body from the context `ctx`.
	// 4. Execute the request.
	// 5. Read the response body, unmarshal it if it's JSON.
	// 6. Store the result in the context, e.g., `ctx["http_call_name"] = responseData`.
	return nil
}

func (r *EnhancedRouter) handleWASM(step ir.TransformStep, ctx map[string]any) error {
	log.Printf("WASM step: %+v", step.Config)
	// TODO: Implement WASM call logic
	// 1. Use a WASM runtime like Wazero or Wasmtime.
	// 2. Load the .wasm module specified in step.Config.
	// 3. Call the specified function, passing arguments from `ctx`.
	// 4. Get the result and store it back into `ctx`.
	return nil
}
