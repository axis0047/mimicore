package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/axis0047/mockingGOD/internal/ir"
)

func TestRouterMatch(t *testing.T) {
	routes := []ir.Route{
		{
			Method: "GET",
			Path:   ir.PathTemplate{Segments: []string{"health"}},
			Response: []ir.ResponseRule{
				{Target: "status", Source: ir.StaticValue{Value: "ok"}},
			},
		},
		{
			Method: "GET",
			Path:   ir.PathTemplate{Segments: []string{"users", "{id}"}},
			Response: []ir.ResponseRule{
				{Target: "user_id", Source: ir.ContextValue{Key: "id"}},
			},
		},
	}

	router := &Router{Routes: routes}

	tests := []struct {
		name           string
		method         string
		path           string
		shouldMatch    bool
		expectedParams map[string]string
	}{
		{
			name:        "static path match",
			method:      "GET",
			path:        "/health",
			shouldMatch: true,
		},
		{
			name:        "dynamic path match",
			method:      "GET",
			path:        "/users/123",
			shouldMatch: true,
			expectedParams: map[string]string{
				"id": "123",
			},
		},
		{
			name:        "no match - wrong method",
			method:      "POST",
			path:        "/health",
			shouldMatch: false,
		},
		{
			name:        "no match - wrong path",
			method:      "GET",
			path:        "/nonexistent",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			route, params := router.Match(req)

			if tt.shouldMatch && route == nil {
				t.Error("Expected route to match but got nil")
			}
			if !tt.shouldMatch && route != nil {
				t.Error("Expected no match but got a route")
			}

			if tt.expectedParams != nil {
				for key, expected := range tt.expectedParams {
					if params[key] != expected {
						t.Errorf("Param %s: expected %s, got %s", key, expected, params[key])
					}
				}
			}
		})
	}
}

func TestRouterServeHTTP(t *testing.T) {
	routes := []ir.Route{
		{
			Method: "GET",
			Path:   ir.PathTemplate{Segments: []string{"health"}},
			Response: []ir.ResponseRule{
				{Target: "status", Source: ir.StaticValue{Value: "ok"}},
			},
		},
	}

	router := &Router{Routes: routes}

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}
