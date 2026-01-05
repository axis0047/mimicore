package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/axis0047/mockingGOD/internal/engine"
)

func TestFullIntegration(t *testing.T) {
	// Setup temporary config directory
	tmpDir := t.TempDir()

	configContent := `[
		{
			"method": "GET",
			"path": "/health",
			"response": {
				"status": "ok",
				"service": "test_api"
			}
		}
	]`

	configPath := filepath.Join(tmpDir, "test_api.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Build handlers
	handlers, err := buildHandlers(tmpDir)
	if err != nil {
		t.Fatalf("buildHandlers failed: %v", err)
	}

	if len(handlers) != 1 {
		t.Fatalf("Expected 1 handler, got %d", len(handlers))
	}

	// Setup gateway
	registry := engine.NewRegistry()
	registry.ReplaceAll(handlers)
	gateway := &engine.Gateway{Registry: registry}

	// Make request
	req := httptest.NewRequest("GET", "/health", nil)
	req.Host = "test_api.example.com"
	rr := httptest.NewRecorder()

	gateway.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response["status"])
	}
	if response["service"] != "test_api" {
		t.Errorf("Expected service 'test_api', got %v", response["service"])
	}
}
