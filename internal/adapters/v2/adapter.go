package v2

import (
	"fmt"
	"log"
	"os"
	"strings"

	json "github.com/goccy/go-json"

	"github.com/axis0047/mockingGOD/internal/ir"
)

func Compile(routes []V2RouteConfig) ([]ir.EnhancedRoute, error) {
	var enhancedRoutes []ir.EnhancedRoute

	for _, cfg := range routes {
		baseRoute := buildBaseRoute(cfg)
		validationSteps := buildValidationSteps(cfg.Validate)
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

	var fileCfg APIFileConfig
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
		if str, ok := v.(string); ok && strings.Contains(str, "{{") {
			rules = append(rules, ir.ResponseRule{
				Target: k,
				Source: ir.StaticValue{Value: str},
			})
		} else {
			rules = append(rules, ir.ResponseRule{
				Target: k,
				Source: ir.StaticValue{Value: v},
			})
		}
	}

	return ir.Route{
		Method:   cfg.Method,
		Path:     ir.PathTemplate{Segments: segs},
		Response: rules,
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
	return steps
}

func buildTransformSteps(transform *TransformConfig) []ir.TransformStep {
	if transform == nil {
		return nil
	}

	var steps []ir.TransformStep

	// 1. Extract Steps (Sequential)
	for mapKey, rule := range transform.Extract {
		targetVar := rule.To
		if targetVar == "" {
			targetVar = mapKey
		}
		steps = append(steps, ir.TransformStep{
			Type: "extract",
			Config: ir.ExtractTransform{
				From: rule.From,
				To:   targetVar,
			},
		})
	}

	// 2. HTTP Steps (BATCHED for Parallelism)
	if len(transform.HTTP) > 0 {
		var batch []ir.HTTPTransform

		for _, httpCall := range transform.HTTP {
			batch = append(batch, ir.HTTPTransform{
				Name:    httpCall.Name,
				URL:     httpCall.URL,
				Method:  httpCall.Method,
				Headers: httpCall.Headers,
				Body:    httpCall.Body,
				Timeout: httpCall.Timeout,
			})
		}

		// Add as a single step
		steps = append(steps, ir.TransformStep{
			Type: "http_batch", // New Type
			Config: ir.ParallelHTTPConfig{
				Calls: batch,
			},
		})
	}

	// 3. WASM (Currently sequential, could be batched similarly if needed)
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
