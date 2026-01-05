package v2

// V2RouteConfig represents the enhanced route configuration
type V2RouteConfig struct {
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Validate  *ValidationConfig `json:"validate,omitempty"`
	Transform *TransformConfig  `json:"transform,omitempty"`
	Response  ResponseConfig    `json:"response"`
	Handler   *WASMHandler      `json:"handler,omitempty"`
}

type ValidationConfig struct {
	Headers map[string]ValidationRule `json:"headers,omitempty"`
	Query   map[string]ValidationRule `json:"query,omitempty"`
	Body    *BodyValidation           `json:"body,omitempty"`
}

type ValidationRule struct {
	Required bool     `json:"required"`
	Type     string   `json:"type"` // "string", "number", "email", etc.
	Pattern  string   `json:"pattern,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Enum     []string `json:"enum,omitempty"`
}

type BodyValidation struct {
	Schema string `json:"schema"` // JSON Schema reference or inline
}

type TransformConfig struct {
	Extract map[string]ExtractRule `json:"extract,omitempty"`
	HTTP    []HTTPCallConfig       `json:"http,omitempty"`
	WASM    []WASMCallConfig       `json:"wasm,omitempty"`
}

type ExtractRule struct {
	From string `json:"from"` // "header.X-User-ID", "body.user.name", "query.id"
	As   string `json:"as"`   // Variable name in context
	Type string `json:"type"` // Optional type conversion
}

type HTTPCallConfig struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
	Timeout int               `json:"timeout"` // milliseconds
}

type WASMCallConfig struct {
	Name     string                 `json:"name"`
	Module   string                 `json:"module"`
	Function string                 `json:"function"`
	Args     map[string]interface{} `json:"args"`
}

type ResponseConfig struct {
	Status  int                    `json:"status,omitempty"` // Default 200
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body"`
}

type WASMHandler struct {
	Module   string `json:"module"`
	Function string `json:"function"`
}
