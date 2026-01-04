package engine

import (
	"encoding/json"
	"net/http"
)

type Gateway struct {
	Router *Router
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, params := g.Router.Match(r)
	if route == nil {
		http.NotFound(w, r)
		return
	}

	var payload map[string]any
	_ = json.NewDecoder(r.Body).Decode(&payload)

	ctx := map[string]any{}
	for k, v := range params {
		ctx[k] = v
	}

	if err := RunValidators(route.Validators, payload, ctx); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := BuildResponse(route.Response, ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
