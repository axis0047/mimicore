package engine

import (
	"testing"
)

// Mock handler for testing
type MockHandler struct {
	Called bool
}

func (m *MockHandler) Serve(ctx *Context) {
	m.Called = true
}

func TestRegistryGetSet(t *testing.T) {
	registry := NewRegistry()

	handler := &MockHandler{}
	handlers := map[string]APIHandler{
		"test_api": handler,
	}

	registry.ReplaceAll(handlers)

	// Test successful retrieval
	retrieved, ok := registry.Get("test_api")
	if !ok {
		t.Error("Expected to find test_api")
	}
	if retrieved != handler {
		t.Error("Retrieved handler doesn't match stored handler")
	}

	// Test missing key
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("Expected not to find nonexistent key")
	}
}

func TestRegistrySwap(t *testing.T) {
	registry := NewRegistry()

	handler1 := &MockHandler{}
	registry.ReplaceAll(map[string]APIHandler{
		"api1": handler1,
	})

	// Swap with new handlers
	handler2 := &MockHandler{}
	registry.Swap(map[string]APIHandler{
		"api2": handler2,
	})

	// Old handler should be gone
	_, ok := registry.Get("api1")
	if ok {
		t.Error("api1 should not exist after swap")
	}

	// New handler should exist
	_, ok = registry.Get("api2")
	if !ok {
		t.Error("api2 should exist after swap")
	}
}
