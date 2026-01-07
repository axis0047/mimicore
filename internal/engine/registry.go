package engine

import (
	"sync"
)

type APIHandler interface {
	Serve(*Context)
}

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]APIHandler
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]APIHandler),
	}
}

// Get Read-Locking
func (r *Registry) Get(api string) (APIHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[api]
	return h, ok
}

// Register (Write-Locking): Updates or Adds a single handler
func (r *Registry) Register(api string, handler APIHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[api] = handler
}

// Remove (Write-Locking): Removes a single handler
func (r *Registry) Remove(api string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, api)
}

// ReplaceAll (Legacy support for initial load)
func (r *Registry) ReplaceAll(next map[string]APIHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = next
}
