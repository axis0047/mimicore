package main

import (
	"log"
	"net/http"

	v1 "github.com/axis0047/mockingGOD/internal/adapters/v1"
	"github.com/axis0047/mockingGOD/internal/engine"
)

func main() {
	routes, err := v1.Compile("configs/api_v1.json")
	if err != nil {
		log.Fatal(err)
	}

	gw := &engine.Gateway{
		Router: &engine.Router{Routes: routes},
	}

	log.Println("Gateway listening on :8080")
	http.ListenAndServe(":8080", gw)
}
