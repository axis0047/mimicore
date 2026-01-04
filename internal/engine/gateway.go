package engine

import (
	"net/http"
	"strings"
)

type Gateway struct {
	Registry *Registry
}

func apiFromHost(host string) string {
	host = strings.Split(host, ":")[0]
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return ""
	}
	return parts[0]
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api := apiFromHost(r.Host)
	if api == "" {
		http.NotFound(w, r)
		return
	}

	handler, ok := g.Registry.Get(api)
	if !ok {
		http.NotFound(w, r)
		return
	}

	handler.Serve(&Context{W: w, R: r})
}
