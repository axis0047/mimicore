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
	"github.com/axis0047/mockingGOD/internal/middleware"
	"github.com/axis0047/mockingGOD/internal/services/compiler"
	"github.com/axis0047/mockingGOD/internal/services/wasm"
)

// buildHandlers loads all configs from a directory (Startup)
func buildHandlers(configDir string) (map[string]engine.APIHandler, error) {
	next := make(map[string]engine.APIHandler)

	apis, err := config.LoadAll(configDir)
	if err != nil {
		return nil, err
	}

	log.Printf("Loaded %d API configs", len(apis))

	for _, api := range apis {
		handler, err := buildSingleHandler(api)
		if err != nil {
			log.Printf("Skipping %s: %v", api.API, err)
			continue
		}
		next[api.API] = handler
		log.Printf("✓ Registered API: %s (version: %s)", api.API, api.Version)
	}

	return next, nil
}

// buildFromPath loads a specific file (Hot Reload)
func buildFromPath(path string) (string, engine.APIHandler, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("file not found")
	}

	apiName := strings.TrimSuffix(filepath.Base(path), ".json")

	// Detect version dynamically
	version := config.DetectVersion(path)

	api := config.APIConfig{
		API:        apiName,
		Mode:       config.ModeIR, // Defaulting to IR for reload simplicity
		Version:    version,
		RoutesFile: path,
	}

	handler, err := buildSingleHandler(api)
	return apiName, handler, err
}

func buildSingleHandler(api config.APIConfig) (engine.APIHandler, error) {
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
	// 1. Read the config file
	raw, err := os.ReadFile(api.RoutesFile)
	if err != nil {
		return nil, err
	}

	// 2. Unmarshal strictly into the new Object format
	var fullCfg v2.APIFileConfig
	if err := json.Unmarshal(raw, &fullCfg); err != nil {
		return nil, fmt.Errorf("invalid v2 config (must be object with 'routes'): %w", err)
	}

	// 3. Register Rate Limits (Middleware)
	if fullCfg.RateLimit != nil {
		// Register both standard localhost and .local for testing flexibility
		hostKey := api.API + ".localhost"
		hostKeyLocal := api.API + ".local"

		middleware.GlobalLimitRegistry.UpdateConfig(
			hostKey,
			fullCfg.RateLimit.RequestsPerSecond,
			fullCfg.RateLimit.Burst,
		)

		// Alias
		middleware.GlobalLimitRegistry.UpdateConfig(
			hostKeyLocal,
			fullCfg.RateLimit.RequestsPerSecond,
			fullCfg.RateLimit.Burst,
		)
	} else {
		// If RateLimit was removed from config, clear it from registry
		middleware.GlobalLimitRegistry.RemoveConfig(api.API + ".localhost")
		middleware.GlobalLimitRegistry.RemoveConfig(api.API + ".local")
	}

	var wasmMgr *wasm.Manager

	// 4. Handle User Code (WASM)
	if fullCfg.UserCode != nil {
		var wasmBytes []byte

		// Option A: Inline Source (Compiles dynamically)
		if fullCfg.UserCode.InlineSource != "" {
			log.Printf("[%s] Checking/Compiling code...", api.API)
			startTime := time.Now()

			wasmBytes, err = compiler.CompileWASM(fullCfg.UserCode.InlineSource)
			if err != nil {
				return nil, fmt.Errorf("user code compilation failed: %w", err)
			}
			log.Printf("[%s] Code ready in %s", api.API, time.Since(startTime))

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

	// 5. Compile Routes
	enhancedRoutes, err := v2.Compile(fullCfg.Routes)
	if err != nil {
		return nil, err
	}

	// 6. Create Router
	router := engine.NewEnhancedRouter(enhancedRoutes, wasmMgr)

	return &engine.EnhancedIRHandler{Router: router}, nil
}
