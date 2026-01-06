package v2

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/axis0047/mockingGOD/internal/ir"
)

// Compile now accepts []V2RouteConfig directly
func Compile(routes []V2RouteConfig) ([]ir.EnhancedRoute, error) {
	var enhancedRoutes []ir.EnhancedRoute

	for _, cfg := range routes {
		// No need to unmarshal/marshal anymore, we have the struct

		// Build base route
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

// CompileFile reads v2 config from file (Legacy helper)
func CompileFile(path string) ([]ir.EnhancedRoute, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// We unmarshal into the struct now, not map[string]interface{}
	var fileCfg APIFileConfig // Using the struct from config.go
	if err := json.Unmarshal(raw, &fileCfg); err != nil {
		// Fallback: try unmarshalling just the array if it's not the full object
		var routes []V2RouteConfig
		if err2 := json.Unmarshal(raw, &routes); err2 != nil {
			return nil, fmt.Errorf("unmarshal failed: %w", err)
		}
		return Compile(routes)
	}

	return Compile(fileCfg.Routes)
}

func buildBaseRoute(cfg V2RouteConfig) ir.Route {
	segs := strings.Split(strings.Trim(cfg.Path, "/"), "/")

	log.Printf("  Building base route: %s %s -> segments: %v", cfg.Method, cfg.Path, segs)

	var rules []ir.ResponseRule
	for k, v := range cfg.Response.Body {
		// Check if value contains template syntax {{...}}
		if str, ok := v.(string); ok && strings.Contains(str, "{{") {
			// This is a dynamic value - store as StaticValue containing the template string.
			// The EnhancedRouter will detect the "{{" and resolve it at runtime.
			rules = append(rules, ir.ResponseRule{
				Target: k,
				Source: ir.StaticValue{Value: str},
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
		Validators: nil, // Validators are handled in EnhancedRouter via ValidationSteps
		Response:   rules,
	}
}

func buildValidationSteps(validate *ValidationConfig) []ir.ValidationStep {
	if validate == nil {
		return nil
	}

	var steps []ir.ValidationStep

	for field, rule := range validate.Headers {
		steps = append(steps, ir.ValidationStep{
			Type:  "header",
			Field: field,
			Rules: rule,
		})
	}

	// Add other validations (Query, Body) here if needed in future
	return steps
}

func buildTransformSteps(transform *TransformConfig) []ir.TransformStep {
	if transform == nil {
		return nil
	}

	var steps []ir.TransformStep

	for varName, rule := range transform.Extract {
		steps = append(steps, ir.TransformStep{
			Type: "extract",
			Config: ir.ExtractTransform{
				From: rule.From,
				To:   varName,
			},
		})
	}

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

	return steps
}
