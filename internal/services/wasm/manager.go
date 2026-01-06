package wasm

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type Config struct {
	Binary        []byte
	Timeout       time.Duration
	MaxMemoryPage uint32
	PoolSize      int
}

type Manager struct {
	runtime      wazero.Runtime
	compiledCode wazero.CompiledModule
	config       Config
	pool         chan api.Module
	mu           sync.Mutex
	isClosed     bool
}

func NewManager(cfg Config) (*Manager, error) {
	ctx := context.Background()

	// 1. Create Runtime
	r := wazero.NewRuntime(ctx)

	// 2. Instantiate WASI (System Interface) - ONCE per Runtime
	// This registers the "wasi_snapshot_preview1" module that your TinyGo code needs.
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	if len(cfg.Binary) == 0 {
		r.Close(ctx)
		return nil, fmt.Errorf("wasm binary is empty")
	}

	// 3. Compile the module once (Performance)
	compiled, err := r.CompileModule(ctx, cfg.Binary)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("compile module: %w", err)
	}

	mgr := &Manager{
		runtime:      r,
		compiledCode: compiled,
		config:       cfg,
		pool:         make(chan api.Module, cfg.PoolSize),
	}

	// 4. Pre-warm the pool
	for i := 0; i < cfg.PoolSize; i++ {
		mod, err := mgr.instantiate(ctx)
		if err != nil {
			log.Printf("Failed to warm up wasm instance: %v", err)
			continue
		}
		mgr.pool <- mod
	}

	return mgr, nil
}

func (m *Manager) instantiate(ctx context.Context) (api.Module, error) {
	// REMOVED: wasi_snapshot_preview1.MustInstantiate(ctx, m.runtime)
	// It is now handled in NewManager once.

	// Create config with limits
	modConfig := wazero.NewModuleConfig().
		WithSysWalltime().
		WithSysNanotime().
		WithRandSource(nil)

	// Instantiate the user's module (links to the already loaded WASI)
	return m.runtime.InstantiateModule(ctx, m.compiledCode, modConfig)
}

// Call executes a function with Security limits (Timeout)
func (m *Manager) Call(funcName string, args ...uint64) (uint64, error) {
	var mod api.Module
	select {
	case mod = <-m.pool:
	default:
		// Pool exhausted, create temporary instance
		var err error
		mod, err = m.instantiate(context.Background())
		if err != nil {
			return 0, err
		}
	}

	defer func() {
		// Return to pool if not closed, otherwise close it
		select {
		case m.pool <- mod:
		default:
			mod.Close(context.Background())
		}
	}()

	// Security: Timeout enforcement
	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()

	f := mod.ExportedFunction(funcName)
	if f == nil {
		return 0, fmt.Errorf("function %s not exported", funcName)
	}

	results, err := f.Call(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("execution error: %w", err)
	}

	if len(results) > 0 {
		return results[0], nil
	}
	return 0, nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.isClosed {
		m.isClosed = true
		m.runtime.Close(context.Background())
	}
}
