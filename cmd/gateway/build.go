package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	json "github.com/goccy/go-json"

	v1 "github.com/axis0047/mockingGOD/internal/adapters/v1"
	v2 "github.com/axis0047/mockingGOD/internal/adapters/v2"
	"github.com/axis0047/mockingGOD/internal/config"
	"github.com/axis0047/mockingGOD/internal/engine"
	"github.com/axis0047/mockingGOD/internal/services/compiler"
	"github.com/axis0047/mockingGOD/internal/services/wasm"
)

// BuildAll (Startup)
func buildHandlers(configDir string) (map[string]engine.APIHandler, error) {
	next := make(map[string]engine.APIHandler)
	apis, err := config.LoadAll(configDir)
	if err != nil {
		return nil, err
	}

	log.Printf("Initial load: found %d configs", len(apis))
	for _, api := range apis {
		handler, err := buildSingleHandler(api)
		if err != nil {
			log.Printf("Skipping %s: %v", api.API, err)
			continue
		}
		next[api.API] = handler
	}
	return next, nil
}

// BuildOne (Hot Reload Helper)
// Takes a filepath (e.g., configs/api_abc.json) and returns the API Name and Handler
func buildFromPath(path string) (string, engine.APIHandler, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("file not found")
	}

	// 1. Derive Config Object manually
	apiName := strings.TrimSuffix(filepath.Base(path), ".json")
	version := config.DetectVersion(path) // We need to move/export detectVersion or copy logic

	api := config.APIConfig{
		API:        apiName,
		Mode:       config.ModeIR, // Defaulting to IR for reload simplicity
		Version:    version,
		RoutesFile: path,
	}

	handler, err := buildSingleHandler(api)
	return apiName, handler, err
}

// Logic extracted from the loop
func buildSingleHandler(api config.APIConfig) (engine.APIHandler, error) {
	log.Printf("Building: %s (%s)", api.API, api.Version)

	if api.Mode == config.ModeProxy {
		return engine.NewUnixProxy(api.UnixSocket), nil
	}

	switch api.Version {
	case "v1":
		return buildV1Handler(api)
	case "v2":
		return buildV2Handler(api)
	default:
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
	raw, err := os.ReadFile(api.RoutesFile)
	if err != nil {
		return nil, err
	}

	var fullCfg v2.APIFileConfig
	if err := json.Unmarshal(raw, &fullCfg); err != nil {
		return nil, fmt.Errorf("invalid v2 config: %w", err)
	}

	var wasmMgr *wasm.Manager

	if fullCfg.UserCode != nil {
		var wasmBytes []byte

		if fullCfg.UserCode.InlineSource != "" {
			// USES NEW CACHING COMPILER
			log.Printf("[%s] Checking/Compiling code...", api.API)
			start := time.Now()

			// This call is now cached!
			wasmBytes, err = compiler.CompileWASM(fullCfg.UserCode.InlineSource)

			if err != nil {
				return nil, err
			}
			log.Printf("[%s] Code ready in %v", api.API, time.Since(start))

		} else if fullCfg.UserCode.Filepath != "" {
			wasmBytes, err = os.ReadFile(fullCfg.UserCode.Filepath)
			if err != nil {
				return nil, err
			}
		}

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
				return nil, err
			}
		}
	}

	enhancedRoutes, err := v2.Compile(fullCfg.Routes)
	if err != nil {
		return nil, err
	}

	router := &engine.EnhancedRouter{
		Routes: enhancedRoutes,
		Wasm:   wasmMgr,
	}
	return &engine.EnhancedIRHandler{Router: router}, nil
}
