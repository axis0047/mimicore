package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiFromHost(t *testing.T) {
	tests := []struct {
		host     string
		expected string
	}{
		{"api_config_1.example.com", "api_config_1"},
		{"api_config_2.example.com:8080", "api_config_2"},
		{"localhost:8080", "localhost"},
		{"example.com", "example"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := apiFromHost(tt.host)
			if result != tt.expected {
				t.Errorf("apiFromHost(%s) = %s, want %s", tt.host, result, tt.expected)
			}
		})
	}
}

func TestGatewayServeHTTP(t *testing.T) {
	registry := NewRegistry()
	mockHandler := &MockHandler{}

	registry.ReplaceAll(map[string]APIHandler{
		"api_test": mockHandler,
	})

	gateway := &Gateway{Registry: registry}

	tests := []struct {
		name              string
		host              string
		expectedStatus    int
		shouldCallHandler bool
	}{
		{
			name:              "valid API",
			host:              "api_test.example.com",
			expectedStatus:    http.StatusOK,
			shouldCallHandler: true,
		},
		{
			name:              "unknown API",
			host:              "unknown.example.com",
			expectedStatus:    http.StatusNotFound,
			shouldCallHandler: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler.Called = false // Reset

			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			rr := httptest.NewRecorder()

			gateway.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if mockHandler.Called != tt.shouldCallHandler {
				t.Errorf("Handler called = %v, want %v", mockHandler.Called, tt.shouldCallHandler)
			}
		})
	}
}
