package ir

type EnhancedRoute struct {
	Route
	ValidationSteps []ValidationStep
	TransformSteps  []TransformStep
	Delay           DelayConfig // NEW
}

type DelayConfig struct {
	FixedMs  int
	JitterMs int
}

type ValidationStep struct {
	Type  string      // "header", "query", "body"
	Field string      // Field name (empty for body)
	Rules interface{} // ValidationRule or map[string]any (schema)
}

type TransformStep struct {
	Type   string // "extract", "http_batch", "wasm"
	Config interface{}
}

type ExtractTransform struct {
	From string
	To   string
}

type HTTPTransform struct {
	Name    string
	URL     string
	Method  string
	Headers map[string]string
	Body    interface{}
	Timeout int
}

type ParallelHTTPConfig struct {
	Calls []HTTPTransform
}

type WASMTransform struct {
	Name     string
	Module   string
	Function string
	Args     map[string]interface{}
}
