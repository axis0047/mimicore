package ir

type ResponseRule struct {
	Target string
	Source ValueSource
}

type ValueSource interface {
	Resolve(ctx map[string]any) (any, error)
}

// Static value
type StaticValue struct {
	Value any
}

func (s StaticValue) Resolve(_ map[string]any) (any, error) {
	return s.Value, nil
}

// Context lookup
type ContextValue struct {
	Key string
}

func (c ContextValue) Resolve(ctx map[string]any) (any, error) {
	return ctx[c.Key], nil
}
