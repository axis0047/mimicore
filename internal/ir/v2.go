package ir

// Enhanced route with v2 features
type EnhancedRoute struct {
	Route // Embed base route

	// V2 additions
	ValidationSteps []ValidationStep
	TransformSteps  []TransformStep
}

type ValidationStep struct {
	Type  string      // "header", "query", "body"
	Field string      // Field name
	Rules interface{} // Validation rules
}

type TransformStep struct {
	Type   string      // "extract", "http", "wasm"
	Config interface{} // Specific config
}

// Extract transformation
type ExtractTransform struct {
	From   string // Source path
	To     string // Target variable
	Parser string // Optional parser
}

// HTTP call transformation
type HTTPTransform struct {
	Name    string
	URL     string
	Method  string
	Headers map[string]string
	Body    interface{}
	Timeout int
}

// WASM call transformation
type WASMTransform struct {
	Name     string
	Module   string
	Function string
	Args     map[string]interface{}
}
