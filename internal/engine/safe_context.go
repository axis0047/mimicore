package engine

import (
	"sync"
)

// SafeContext wraps a map with a RWMutex for thread-safe access.
type SafeContext struct {
	mu   sync.RWMutex
	data map[string]any
}

func NewSafeContext() *SafeContext {
	return &SafeContext{
		data: make(map[string]any),
	}
}

// Set writes a value safely.
func (c *SafeContext) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Get reads a value safely.
func (c *SafeContext) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

// GetAll returns a copy of the map (useful for final response building).
// We return a copy so the responder reads a snapshot without holding the lock.
func (c *SafeContext) GetAll() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	copy := make(map[string]any, len(c.data))
	for k, v := range c.data {
		copy[k] = v
	}
	return copy
}
