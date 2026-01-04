package engine

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type Gateway struct {
	router atomic.Value // holds *Router
}

func NewGateway(initial *Router) *Gateway {
	g := &Gateway{}
	g.router.Store(initial)
	return g
}

// SwapRouter atomically replaces the active router
func (g *Gateway) SwapRouter(r *Router) {
	g.router.Store(r)
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := g.router.Load().(*Router)

	route, params := router.Match(r)
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
