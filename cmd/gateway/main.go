package main

import (
	"log"
	"net/http"

	v1 "github.com/axis0047/mockingGOD/internal/adapters/v1"
	"github.com/axis0047/mockingGOD/internal/config"
	"github.com/axis0047/mockingGOD/internal/engine"
)

func main() {
	adapter := &v1.Adapter{
		Path: "configs/api_v1.json",
	}

	routes, err := adapter.Compile()
	if err != nil {
		log.Fatal(err)
	}

	gateway := engine.NewGateway(&engine.Router{Routes: routes})

	manager := &config.Manager{
		Adapter: adapter,
		Gateway: gateway,
	}

	if err := config.Watch("configs/api_v1.json", manager); err != nil {
		log.Fatal(err)
	}

	log.Println("Gateway listening on :8080")
	http.ListenAndServe(":8080", gateway)
}
