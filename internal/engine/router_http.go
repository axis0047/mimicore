package engine

import (
	"log"
	"net/http"

	"github.com/axis0047/mockingGOD/internal/utils" // <-- Import utils
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
		// Use the correct SetNested from the utils package
		utils.SetNested(resp, rule.Target, val)
	}

	// Use the helper function for consistency
	utils.WriteJSON(w, http.StatusOK, resp)
}
