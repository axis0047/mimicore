package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CompileWASM takes raw Go source code, writes it to a temp file,
// compiles it using TinyGo, and returns the WASM binary bytes.
func CompileWASM(sourceCode string) ([]byte, error) {
	// 1. Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "mockinggod_build_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) // Clean up source and artifacts after build

	// 2. Write the source code to main.go
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte(sourceCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source code: %w", err)
	}

	// 3. Define output path
	outPath := filepath.Join(tmpDir, "main.wasm")

	// 4. Run TinyGo command
	// Command: tinygo build -o <outPath> -target=wasi <srcPath>
	cmd := exec.Command("tinygo", "build", "-o", outPath, "-target=wasi", "-no-debug", srcPath)

	// Capture output for debugging compilation errors
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compilation failed:\n%s\nError: %w", string(output), err)
	}

	// 5. Read the compiled binary
	wasmBytes, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled wasm: %w", err)
	}

	return wasmBytes, nil
}
