package ir

type Validator interface {
	Validate(input map[string]any, ctx map[string]any) error
}
