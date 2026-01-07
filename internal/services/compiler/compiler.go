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

func CompileWASM(sourceCode string) ([]byte, error) {
	// 1. Calculate Hash
	hasher := sha256.New()
	hasher.Write([]byte(sourceCode))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// 2. Check L1 Cache (RAM)
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
			// Populate RAM cache
			cacheMu.Lock()
			cache[hash] = wasmBytes
			cacheMu.Unlock()
			return wasmBytes, nil
		}
	}

	log.Printf("[Compiler] Cache Miss. Compiling %s...", hash[:8])

	// 4. Compilation (Expensive)
	tmpDir, err := os.MkdirTemp("", "mockinggod_build_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte(sourceCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source code: %w", err)
	}

	outPath := filepath.Join(tmpDir, "main.wasm")
	cmd := exec.Command("tinygo", "build", "-o", outPath, "-target=wasi", "-no-debug", srcPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compilation failed:\n%s\nError: %w", string(output), err)
	}

	wasmBytes, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled wasm: %w", err)
	}

	// 5. Update Caches
	cacheMu.Lock()
	cache[hash] = wasmBytes
	cacheMu.Unlock()

	// Upload to S3 asynchronously
	if storage.GlobalStorage != nil {
		go func() {
			if err := storage.GlobalStorage.Put(hash, wasmBytes); err != nil {
				log.Printf("[Compiler] Failed to upload to S3: %v", err)
			} else {
				log.Printf("[Compiler] Persisted %s to S3", hash[:8])
			}
		}()
	}

	return wasmBytes, nil
}
