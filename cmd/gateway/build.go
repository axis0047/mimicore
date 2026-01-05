package main

import (
	"encoding/json"
	"log"
	"os"

	v1 "github.com/axis0047/mockingGOD/internal/adapters/v1"
	v2 "github.com/axis0047/mockingGOD/internal/adapters/v2"
	"github.com/axis0047/mockingGOD/internal/config"
	"github.com/axis0047/mockingGOD/internal/engine"
)

func buildHandlers(configDir string) (map[string]engine.APIHandler, error) {
	next := make(map[string]engine.APIHandler)

	apis, err := config.LoadAll(configDir)
	if err != nil {
		return nil, err
	}

	log.Printf("Loaded %d API configs", len(apis))

	for _, api := range apis {
		log.Printf("Building handler for API: %q, Mode: %q, Version: %q",
			api.API, api.Mode, api.Version)

		switch api.Mode {
		case config.ModeIR:
			handler, err := buildIRHandler(api)
			if err != nil {
				log.Printf("Failed to build IR handler for %s: %v", api.API, err)
				continue
			}
			next[api.API] = handler
			log.Printf("✓ Registered API: %s (version: %s)", api.API, api.Version)

		case config.ModeProxy:
			next[api.API] = engine.NewUnixProxy(api.UnixSocket)
			log.Printf("✓ Registered proxy: %s", api.API)
		}
	}

	log.Printf("Built %d handlers", len(next))
	return next, nil
}

func buildIRHandler(api config.APIConfig) (engine.APIHandler, error) {
	// Determine version (default to v1 for backward compatibility)
	version := api.Version
	if version == "" {
		version = "v1"
	}

	switch version {
	case "v1":
		return buildV1Handler(api)
	case "v2":
		return buildV2Handler(api)
	default:
		log.Printf("Unknown version %s, falling back to v1", version)
		return buildV1Handler(api)
	}
}

func buildV1Handler(api config.APIConfig) (engine.APIHandler, error) {
	routes, err := v1.Compile(api.RoutesFile)
	if err != nil {
		return nil, err
	}

	router := &engine.Router{Routes: routes}
	return &engine.IRHandler{Router: router}, nil
}

func buildV2Handler(api config.APIConfig) (engine.APIHandler, error) {
	var routes []map[string]interface{}

	// V2 can load from file or inline routes
	if api.RoutesFile != "" {
		// Load from file
		raw, err := os.ReadFile(api.RoutesFile)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &routes); err != nil {
			return nil, err
		}
	} else {
		// Use inline routes
		routes = api.Routes
	}

	enhancedRoutes, err := v2.Compile(routes)
	if err != nil {
		return nil, err
	}

	router := &engine.EnhancedRouter{Routes: enhancedRoutes}
	return &engine.EnhancedIRHandler{Router: router}, nil
}
