package ir

import (
	"fmt"
	"strings"
)

// ContextAccessor defines what the IR needs from the context
type ContextAccessor interface {
	Get(key string) (any, bool)
}

type ResponseRule struct {
	Target string
	Source ValueSource
}

type ValueSource interface {
	// UPDATED: Now accepts an interface, not a raw map
	Resolve(ctx ContextAccessor) (any, error)
}

// Static value
type StaticValue struct {
	Value any
}

func (s StaticValue) Resolve(_ ContextAccessor) (any, error) {
	return s.Value, nil
}

// Context lookup
type ContextValue struct {
	Key string
}

func (c ContextValue) Resolve(ctx ContextAccessor) (any, error) {
	// Support nested keys like "upstream.data.id"
	if strings.Contains(c.Key, ".") {
		parts := strings.Split(c.Key, ".")

		// Get root object safely
		rootVal, ok := ctx.Get(parts[0])
		if !ok {
			return nil, fmt.Errorf("key %s not found in context", parts[0])
		}

		current := rootVal

		// Traverse the rest (standard map traversal, assumes nested maps are not being concurrently written to)
		for i := 1; i < len(parts); i++ {
			part := parts[i]

			if nextMap, ok := current.(map[string]any); ok {
				var exists bool
				current, exists = nextMap[part]
				if !exists {
					return nil, fmt.Errorf("key %s not found in nested object", part)
				}
			} else if nextMap, ok := current.(map[string]interface{}); ok {
				var exists bool
				current, exists = nextMap[part]
				if !exists {
					return nil, fmt.Errorf("key %s not found in nested object", part)
				}
			} else {
				return nil, fmt.Errorf("cannot traverse path %s: %s is not a map", c.Key, part)
			}
		}
		return current, nil
	}

	// Direct lookup
	if val, ok := ctx.Get(c.Key); ok {
		return val, nil
	}
	return nil, fmt.Errorf("key %s not found", c.Key)
}
