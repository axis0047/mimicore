package engine

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Gateway struct {
	Registry *Registry
}

func apiFromHost(host string) string {
	host = strings.Split(host, ":")[0]
	parts := strings.Split(host, ".")
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}

func normalizeHost(host string) string {
	if i := strings.Index(host, ":"); i != -1 {
		return host[:i]
	}
	return host
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Host:", r.Host, "Method:", r.Method, "Path:", r.URL.Path)

	fmt.Print("Host is - ")
	fmt.Println(r.Host)

	host := normalizeHost(r.Host)
	api := apiFromHost(host)

	fmt.Println(api)

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
