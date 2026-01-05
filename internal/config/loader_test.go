package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAll(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test config files
	testConfigs := map[string]string{
		"api_test_1.json": `[{"method":"GET","path":"/health","response":{"status":"ok"}}]`,
		"api_test_2.json": `[{"method":"POST","path":"/data","response":{"result":"success"}}]`,
	}

	for name, content := range testConfigs {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test LoadAll
	apis, err := LoadAll(tmpDir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	// Verify results
	if len(apis) != 2 {
		t.Errorf("Expected 2 APIs, got %d", len(apis))
	}

	// Check API names are derived from filenames
	apiNames := make(map[string]bool)
	for _, api := range apis {
		apiNames[api.API] = true
	}

	if !apiNames["api_test_1"] {
		t.Error("Expected api_test_1 to be loaded")
	}
	if !apiNames["api_test_2"] {
		t.Error("Expected api_test_2 to be loaded")
	}
}

func TestLoadAllEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	apis, err := LoadAll(tmpDir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(apis) != 0 {
		t.Errorf("Expected 0 APIs in empty directory, got %d", len(apis))
	}
}
