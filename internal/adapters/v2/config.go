package v2

type V2RouteConfig struct {
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Validate  *ValidationConfig `json:"validate,omitempty"`
	Transform *TransformConfig  `json:"transform,omitempty"`
	Response  ResponseConfig    `json:"response"`
}

type ValidationConfig struct {
	Headers map[string]ValidationRule `json:"headers,omitempty"`
	Query   map[string]ValidationRule `json:"query,omitempty"`
	Body    *BodyValidation           `json:"body,omitempty"`
}

type ValidationRule struct {
	Required bool     `json:"required"`
	Pattern  string   `json:"pattern,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
}

type BodyValidation struct {
	Schema string `json:"schema"`
}

type TransformConfig struct {
	Extract map[string]ExtractRule `json:"extract,omitempty"`
	HTTP    []HTTPCallConfig       `json:"http,omitempty"`
	WASM    []WASMCallConfig       `json:"wasm,omitempty"`
}

type ExtractRule struct {
	From string `json:"from"`
	To   string `json:"as"`
	Type string `json:"type"`
}

type HTTPCallConfig struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
	Timeout int               `json:"timeout"`
}

type WASMCallConfig struct {
	Name     string                 `json:"name"`
	Module   string                 `json:"module"`
	Function string                 `json:"function"`
	Args     map[string]interface{} `json:"args"`
}

type ResponseConfig struct {
	Status  int                    `json:"status,omitempty"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body"`
}

// UserCodeConfig defines dynamic user code settings
type UserCodeConfig struct {
	InlineSource string `json:"inline_source,omitempty"` // Raw Go code
	Filepath     string `json:"source,omitempty"`        // Fallback file path
	TimeoutMs    int    `json:"timeout_ms"`
	MaxMemoryMB  int    `json:"max_memory_mb"`
	MinInstances int    `json:"min_instances"`
	MaxInstances int    `json:"max_instances"`
}

// APIFileConfig represents the full JSON file structure
type APIFileConfig struct {
	API      string          `json:"api"`
	Mode     string          `json:"mode"`
	Version  string          `json:"version"`
	UserCode *UserCodeConfig `json:"user_code,omitempty"`
	Routes   []V2RouteConfig `json:"routes"`
}
