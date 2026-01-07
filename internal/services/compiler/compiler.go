package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/axis0047/mockingGOD/internal/services/storage"
)

var (
	cache   = make(map[string][]byte)
	cacheMu sync.RWMutex
)

const memoryHelpers = `
package main

//export _guest_alloc
func _guest_alloc(size uint32) *byte {
	buf := make([]byte, size)
	return &buf[0]
}

//export _guest_dealloc
func _guest_dealloc(ptr *byte) {
}
`

func CompileWASM(sourceCode string) ([]byte, error) {
	// 1. Calculate Hash
	hasher := sha256.New()
	hasher.Write([]byte(sourceCode))
	hasher.Write([]byte(memoryHelpers))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// 2. Check L1 Cache
	cacheMu.RLock()
	if wasmBytes, exists := cache[hash]; exists {
		cacheMu.RUnlock()
		return wasmBytes, nil
	}
	cacheMu.RUnlock()

	// 3. Check L2 Cache (S3)
	if storage.GlobalStorage != nil {
		if wasmBytes, found := storage.GlobalStorage.Get(hash); found {
			log.Printf("[Compiler] S3 Cache Hit for %s...", hash[:8])
			cacheMu.Lock()
			cache[hash] = wasmBytes
			cacheMu.Unlock()
			return wasmBytes, nil
		}
	}

	log.Printf("[Compiler] Cache Miss. Compiling %s...", hash[:8])

	// 4. Compilation Setup
	tmpDir, err := os.MkdirTemp("", "mockinggod_build_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write User Code
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(sourceCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source: %w", err)
	}

	// Write Helpers
	if err := os.WriteFile(filepath.Join(tmpDir, "host_helpers.go"), []byte(memoryHelpers), 0644); err != nil {
		return nil, fmt.Errorf("failed to write helpers: %w", err)
	}

	// --- NEW: Initialize Go Module ---
	// This ensures imports (like encoding/json) resolve correctly
	modCmd := exec.Command("go", "mod", "init", "dynamic_wasm_build")
	modCmd.Dir = tmpDir
	if out, err := modCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod init failed: %s", out)
	}
	// ---------------------------------

	outPath := filepath.Join(tmpDir, "main.wasm")

	// 5. Run TinyGo Build
	cmd := exec.Command("tinygo", "build",
		"-o", outPath,
		"-target=wasi",
		"-no-debug",
		"-scheduler=none",
		".", // Build current directory (tmpDir)
	)
	cmd.Dir = tmpDir // Execute inside the temp dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compilation failed:\n%s\nError: %w", string(output), err)
	}

	wasmBytes, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled wasm: %w", err)
	}

	// 6. Update Caches
	cacheMu.Lock()
	cache[hash] = wasmBytes
	cacheMu.Unlock()

	if storage.GlobalStorage != nil {
		go func() {
			storage.GlobalStorage.Put(hash, wasmBytes)
		}()
	}

	return wasmBytes, nil
}
