package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Memory Cache: SourceHash -> WasmBytes
var (
	cache   = make(map[string][]byte)
	cacheMu sync.RWMutex
)

// CompileWASM takes raw Go source code.
// It checks the cache first. If missed, it compiles using TinyGo.
func CompileWASM(sourceCode string) ([]byte, error) {
	// 1. Calculate SHA256 Hash of the source
	hasher := sha256.New()
	hasher.Write([]byte(sourceCode))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// 2. Check Cache
	cacheMu.RLock()
	if wasmBytes, exists := cache[hash]; exists {
		cacheMu.RUnlock()
		return wasmBytes, nil // Instant return!
	}
	cacheMu.RUnlock()

	// 3. Cache Miss - Start Compilation
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

	// Use -no-debug for smaller binaries and faster execution
	cmd := exec.Command("tinygo", "build", "-o", outPath, "-target=wasi", "-no-debug", srcPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compilation failed:\n%s\nError: %w", string(output), err)
	}

	wasmBytes, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled wasm: %w", err)
	}

	// 4. Update Cache
	cacheMu.Lock()
	cache[hash] = wasmBytes
	cacheMu.Unlock()

	return wasmBytes, nil
}
