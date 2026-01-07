package engine

// MapContext wraps a standard map to satisfy the ir.ContextAccessor interface.
// This is used by V1 routes which don't need thread-safety (MapContext is NOT thread-safe).
type MapContext map[string]any

// Get implements ir.ContextAccessor
func (m MapContext) Get(key string) (any, bool) {
	val, ok := m[key]
	return val, ok
}
