package main

import (
	"log"
	"net/http"

	"github.com/axis0047/mockingGOD/internal/engine"
)

func main() {
	registry := engine.NewRegistry()

	// Initial load
	handlers, err := buildHandlers("configs")
	if err != nil {
		log.Fatal(err)
	}
	registry.ReplaceAll(handlers)

	// fsnotify hot reload
	go watchConfigs("configs", registry)

	gateway := &engine.Gateway{Registry: registry}

	log.Println("Gateway listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", gateway))
}
