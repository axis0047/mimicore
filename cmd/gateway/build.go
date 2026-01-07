package main

import (
	"fmt"
	"log"
	"os"
	"time"

	// Replace standard json with go-json
	json "github.com/goccy/go-json"

	v1 "github.com/axis0047/mockingGOD/internal/adapters/v1"
	v2 "github.com/axis0047/mockingGOD/internal/adapters/v2"
	"github.com/axis0047/mockingGOD/internal/config"
	"github.com/axis0047/mockingGOD/internal/engine"
	"github.com/axis0047/mockingGOD/internal/services/compiler"
	"github.com/axis0047/mockingGOD/internal/services/wasm"
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
	// 1. Read the config file
	raw, err := os.ReadFile(api.RoutesFile)
	if err != nil {
		return nil, err
	}

	// 2. Unmarshal strictly into the new Object format
	// This drops support for root-level arrays.
	var fullCfg v2.APIFileConfig
	if err := json.Unmarshal(raw, &fullCfg); err != nil {
		return nil, fmt.Errorf("invalid v2 config (must be object with 'routes'): %w", err)
	}

	var wasmMgr *wasm.Manager

	// 3. Handle User Code (WASM)
	if fullCfg.UserCode != nil {
		var wasmBytes []byte

		// Option A: Inline Source (Compiles dynamically)
		if fullCfg.UserCode.InlineSource != "" {
			log.Printf("[%s] Compiling inline user code...", api.API)
			startTime := time.Now()

			wasmBytes, err = compiler.CompileWASM(fullCfg.UserCode.InlineSource)
			if err != nil {
				return nil, fmt.Errorf("user code compilation failed: %w", err)
			}
			log.Printf("[%s] Compilation success (%s)", api.API, time.Since(startTime))

		} else if fullCfg.UserCode.Filepath != "" {
			// Option B: Pre-compiled file
			wasmBytes, err = os.ReadFile(fullCfg.UserCode.Filepath)
			if err != nil {
				return nil, fmt.Errorf("failed to read wasm file: %w", err)
			}
		}

		// Initialize WASM Manager if we have binary data
		if len(wasmBytes) > 0 {
			poolSize := fullCfg.UserCode.MinInstances
			if poolSize < 1 {
				poolSize = 1
			}

			wasmMgr, err = wasm.NewManager(wasm.Config{
				Binary:   wasmBytes,
				Timeout:  time.Duration(fullCfg.UserCode.TimeoutMs) * time.Millisecond,
				PoolSize: poolSize,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to init wasm manager: %w", err)
			}
		}
	}

	// 4. Compile Routes
	enhancedRoutes, err := v2.Compile(fullCfg.Routes)
	if err != nil {
		return nil, err
	}

	// 5. Create Router
	router := &engine.EnhancedRouter{
		Routes: enhancedRoutes,
		Wasm:   wasmMgr,
	}
	return &engine.EnhancedIRHandler{Router: router}, nil
}
