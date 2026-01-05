package engine

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/axis0047/mockingGOD/internal/ir"
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

	// Build execution context
	ctx := map[string]any{}
	for k, v := range params {
		ctx[k] = v
	}

	// Phase 1: Validation
	if err := r.runValidation(route, req, ctx); err != nil {
		log.Printf("Validation failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Phase 2: Transformation
	if err := r.runTransformations(route, req, ctx); err != nil {
		log.Printf("Transformation failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Phase 3: Response building (same as v1)
	resp := make(map[string]any)
	for _, rule := range route.Response {
		val, err := rule.Source.Resolve(ctx)
		if err != nil {
			log.Printf("error resolving %s: %v", rule.Target, err)
			val = "error: " + err.Error()
		}
		SetNested(resp, rule.Target, val)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Println("failed to write response:", err)
	}
}

func (r *EnhancedRouter) matchRoute(req *http.Request) (*ir.EnhancedRoute, map[string]string) {
	path := strings.Trim(req.URL.Path, "/")
	segments := strings.Split(path, "/")

	log.Printf("Matching: %s %s (segments: %v)", req.Method, req.URL.Path, segments)

	for i := range r.Routes {
		route := &r.Routes[i]

		log.Printf("  Checking route: %s %v", route.Method, route.Path.Segments)

		if route.Method != req.Method {
			log.Printf("    Method mismatch: %s != %s", route.Method, req.Method)
			continue
		}

		params := make(map[string]string)
		if matchSegments(route.Path.Segments, segments, params) {
			log.Printf("    ✓ Matched! Params: %v", params)
			return route, params
		}
		log.Printf("    Segments don't match")
	}

	log.Printf("  No routes matched")
	return nil, nil
}

func (r *EnhancedRouter) runValidation(route *ir.EnhancedRoute, req *http.Request, ctx map[string]any) error {
	if len(route.ValidationSteps) > 0 {
		log.Printf("Running %d validation steps", len(route.ValidationSteps))
	}
	// TODO: Implement validation logic
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

func (r *EnhancedRouter) handleExtract(step ir.TransformStep, req *http.Request, ctx map[string]any) error {
	log.Printf("Extract step: %+v", step.Config)
	// TODO: Implement extraction logic
	return nil
}

func (r *EnhancedRouter) handleHTTP(step ir.TransformStep, ctx map[string]any) error {
	log.Printf("HTTP step: %+v", step.Config)
	// TODO: Implement HTTP call logic
	return nil
}

func (r *EnhancedRouter) handleWASM(step ir.TransformStep, ctx map[string]any) error {
	log.Printf("WASM step: %+v", step.Config)
	// TODO: Implement WASM call logic
	return nil
}
