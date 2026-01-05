package v2

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/axis0047/mockingGOD/internal/ir"
)

// Compile converts V2 config to enhanced IR
func Compile(routes []map[string]interface{}) ([]ir.EnhancedRoute, error) {
	var enhancedRoutes []ir.EnhancedRoute

	for _, routeData := range routes {
		// Parse into structured config
		routeJSON, err := json.Marshal(routeData)
		if err != nil {
			return nil, fmt.Errorf("marshal route: %w", err)
		}

		var cfg V2RouteConfig
		if err := json.Unmarshal(routeJSON, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshal route: %w", err)
		}

		// Build base route (same as v1)
		baseRoute := buildBaseRoute(cfg)

		// Build validation steps
		validationSteps := buildValidationSteps(cfg.Validate)

		// Build transformation steps
		transformSteps := buildTransformSteps(cfg.Transform)

		enhancedRoutes = append(enhancedRoutes, ir.EnhancedRoute{
			Route:           baseRoute,
			ValidationSteps: validationSteps,
			TransformSteps:  transformSteps,
		})
	}

	return enhancedRoutes, nil
}

// CompileFile reads v2 config from file (for backward compat with routes_file)
func CompileFile(path string) ([]ir.EnhancedRoute, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var routes []map[string]interface{}
	if err := json.Unmarshal(raw, &routes); err != nil {
		return nil, err
	}

	return Compile(routes)
}

func buildBaseRoute(cfg V2RouteConfig) ir.Route {
	segs := strings.Split(strings.Trim(cfg.Path, "/"), "/")

	log.Printf("  Building base route: %s %s -> segments: %v", cfg.Method, cfg.Path, segs)

	// Build response rules from v2 response config
	var rules []ir.ResponseRule
	for k, v := range cfg.Response.Body {
		// Check if value contains template syntax {{...}}
		if str, ok := v.(string); ok && strings.Contains(str, "{{") {
			// This is a dynamic value - use ContextValue
			varName := extractVarName(str)
			rules = append(rules, ir.ResponseRule{
				Target: k,
				Source: ir.ContextValue{Key: varName},
			})
		} else {
			// Static value
			rules = append(rules, ir.ResponseRule{
				Target: k,
				Source: ir.StaticValue{Value: v},
			})
		}
	}

	return ir.Route{
		Method:     cfg.Method,
		Path:       ir.PathTemplate{Segments: segs},
		Validators: nil,
		Response:   rules,
	}
}

func buildValidationSteps(validate *ValidationConfig) []ir.ValidationStep {
	if validate == nil {
		return nil
	}

	var steps []ir.ValidationStep

	// Header validations
	for field, rule := range validate.Headers {
		steps = append(steps, ir.ValidationStep{
			Type:  "header",
			Field: field,
			Rules: rule,
		})
	}

	// Query validations
	for field, rule := range validate.Query {
		steps = append(steps, ir.ValidationStep{
			Type:  "query",
			Field: field,
			Rules: rule,
		})
	}

	// Body validation
	if validate.Body != nil {
		steps = append(steps, ir.ValidationStep{
			Type:  "body",
			Field: "",
			Rules: validate.Body,
		})
	}

	return steps
}

func buildTransformSteps(transform *TransformConfig) []ir.TransformStep {
	if transform == nil {
		return nil
	}

	var steps []ir.TransformStep

	// Extract steps
	for varName, rule := range transform.Extract {
		steps = append(steps, ir.TransformStep{
			Type: "extract",
			Config: ir.ExtractTransform{
				From: rule.From,
				To:   varName,
			},
		})
	}

	// HTTP call steps
	for _, httpCall := range transform.HTTP {
		steps = append(steps, ir.TransformStep{
			Type: "http",
			Config: ir.HTTPTransform{
				Name:    httpCall.Name,
				URL:     httpCall.URL,
				Method:  httpCall.Method,
				Headers: httpCall.Headers,
				Body:    httpCall.Body,
				Timeout: httpCall.Timeout,
			},
		})
	}

	// WASM call steps
	for _, wasmCall := range transform.WASM {
		steps = append(steps, ir.TransformStep{
			Type: "wasm",
			Config: ir.WASMTransform{
				Name:     wasmCall.Name,
				Module:   wasmCall.Module,
				Function: wasmCall.Function,
				Args:     wasmCall.Args,
			},
		})
	}

	return steps
}

// extractVarName extracts variable name from template string
// "{{user_id}}" -> "user_id"
func extractVarName(template string) string {
	start := strings.Index(template, "{{")
	end := strings.Index(template, "}}")
	if start == -1 || end == -1 {
		return template
	}
	return strings.TrimSpace(template[start+2 : end])
}
