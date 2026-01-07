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

		// Map Delay Config
		var delayCfg ir.DelayConfig
		if cfg.Delay != nil {
			delayCfg = ir.DelayConfig{
				FixedMs:  cfg.Delay.FixedMs,
				JitterMs: cfg.Delay.JitterMs,
			}
		}

		enhancedRoutes = append(enhancedRoutes, ir.EnhancedRoute{
			Route:           baseRoute,
			ValidationSteps: validationSteps,
			TransformSteps:  transformSteps,
			Delay:           delayCfg,
		})
	}

	return enhancedRoutes, nil
}

func CompileFile(path string) ([]ir.EnhancedRoute, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var fileCfg APIFileConfig
	if err := json.Unmarshal(raw, &fileCfg); err != nil {
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

	// Also map Status Code
	status := cfg.Response.Status
	if status == 0 {
		status = 200
	}
	// We can store status as a specific rule or handle it in the response builder.
	// For now, let's just stick to body rules and rely on default 200 in engine if not dynamic.
	// (To fully support dynamic status, we'd need a ResponseRule for status too).

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

	for field, rule := range validate.Query {
		steps = append(steps, ir.ValidationStep{
			Type:  "query",
			Field: field,
			Rules: rule,
		})
	}

	if validate.Body != nil {
		steps = append(steps, ir.ValidationStep{
			Type:  "body",
			Rules: validate.Body.Schema, // Pass the schema map directly
		})
	}

	return steps
}

func buildTransformSteps(transform *TransformConfig) []ir.TransformStep {
	if transform == nil {
		return nil
	}

	var steps []ir.TransformStep

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
		steps = append(steps, ir.TransformStep{
			Type:   "http_batch",
			Config: ir.ParallelHTTPConfig{Calls: batch},
		})
	}

	return steps
}
