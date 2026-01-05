package engine

import (
	"encoding/json"
	"log"
	"net/http"
)

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route, params := r.Match(req)
	if route == nil {
		http.NotFound(w, req)
		return
	}

	// Build context for IR execution
	ctx := map[string]any{}
	for k, v := range params {
		ctx[k] = v
	}

	// Execute response rules
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
