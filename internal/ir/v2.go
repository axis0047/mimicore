package ir

type EnhancedRoute struct {
	Route
	ValidationSteps []ValidationStep
	TransformSteps  []TransformStep
}

type ValidationStep struct {
	Type  string
	Field string
	Rules interface{}
}

type TransformStep struct {
	Type   string // "extract", "http_batch" (UPDATED), "wasm"
	Config interface{}
}

// ExtractTransform remains the same...
type ExtractTransform struct {
	From string
	To   string
}

// HTTPTransform remains the same...
type HTTPTransform struct {
	Name    string
	URL     string
	Method  string
	Headers map[string]string
	Body    interface{}
	Timeout int
}

// NEW: Wrapper for parallel calls
type ParallelHTTPConfig struct {
	Calls []HTTPTransform
}

// WASMTransform remains the same...
type WASMTransform struct {
	Name     string
	Module   string
	Function string
	Args     map[string]interface{}
}
