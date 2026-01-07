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
	r := wazero.NewRuntime(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	if len(cfg.Binary) == 0 {
		r.Close(ctx)
		return nil, fmt.Errorf("wasm binary is empty")
	}

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

// Call executes a function with simple integer arguments
func (m *Manager) Call(funcName string, args ...uint64) (uint64, error) {
	mod, release, err := m.acquire()
	if err != nil {
		return 0, err
	}
	defer release()

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

func (m *Manager) instantiate(ctx context.Context) (api.Module, error) {
	// Create config with limits
	modConfig := wazero.NewModuleConfig().
		WithSysWalltime().
		WithSysNanotime().
		WithRandSource(nil).
		// FIX: Prevent _start from running automatically.
		// This keeps the module "alive" so we can call functions on it repeatedly.
		WithStartFunctions()

	return m.runtime.InstantiateModule(ctx, m.compiledCode, modConfig)
}

func (m *Manager) CallJSON(funcName string, input string) (string, error) {
	mod, release, err := m.acquire()
	if err != nil {
		return "", err
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()

	// 1. Allocate Memory using INJECTED helper
	inputSize := uint64(len(input))

	// FIX: Use the internal name we injected in compiler.go
	fnAlloc := mod.ExportedFunction("_guest_alloc")
	if fnAlloc == nil {
		return "", fmt.Errorf("module initialization error: _guest_alloc not found")
	}

	results, err := fnAlloc.Call(ctx, inputSize)
	if err != nil {
		return "", fmt.Errorf("alloc failed: %w", err)
	}
	inputPtr := results[0]

	// 2. Write to Memory
	if !mod.Memory().Write(uint32(inputPtr), []byte(input)) {
		return "", fmt.Errorf("failed to write memory")
	}

	// 3. Call User Function
	f := mod.ExportedFunction(funcName)
	if f == nil {
		return "", fmt.Errorf("function %s not exported by user", funcName)
	}

	res, err := f.Call(ctx, inputPtr, inputSize)
	if err != nil {
		return "", fmt.Errorf("wasm call failed: %w", err)
	}

	// 4. Read Result
	packed := res[0]
	resPtr := uint32(packed >> 32)
	resLen := uint32(packed)

	bytes, ok := mod.Memory().Read(resPtr, resLen)
	if !ok {
		return "", fmt.Errorf("failed to read result memory")
	}

	return string(bytes), nil
}

func (m *Manager) acquire() (api.Module, func(), error) {
	var mod api.Module
	select {
	case mod = <-m.pool:
	default:
		var err error
		mod, err = m.instantiate(context.Background())
		if err != nil {
			return nil, nil, err
		}
	}

	release := func() {
		select {
		case m.pool <- mod:
		default:
			mod.Close(context.Background())
		}
	}
	return mod, release, nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.isClosed {
		m.isClosed = true
		m.runtime.Close(context.Background())
	}
}
