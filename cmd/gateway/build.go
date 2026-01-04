package main

import (
	"log"

	v1 "github.com/axis0047/mockingGOD/internal/adapters/v1"
	"github.com/axis0047/mockingGOD/internal/config"
	"github.com/axis0047/mockingGOD/internal/engine"
)

func buildHandlers(configDir string) (map[string]engine.APIHandler, error) {
	next := make(map[string]engine.APIHandler)

	apis, err := config.LoadAll(configDir)
	if err != nil {
		return nil, err
	}

	for _, api := range apis {
		switch api.Mode {

		case config.ModeIR:
			routes, err := v1.Compile(api.RoutesFile)
			if err != nil {
				log.Println("IR compile failed:", err)
				continue
			}

			router := &engine.Router{Routes: routes}
			next[api.API] = &engine.IRHandler{Router: router}

		case config.ModeProxy:
			next[api.API] = engine.NewUnixProxy(api.UnixSocket)
		}
	}

	return next, nil
}
