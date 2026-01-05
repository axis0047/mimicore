package config

type APIMode string

const (
	ModeIR    APIMode = "ir"
	ModeProxy APIMode = "proxy"
)

type APIConfig struct {
	API     string  `json:"api"`
	Mode    APIMode `json:"mode"`
	Version string  `json:"version"` // ← NEW: "v1" or "v2", defaults to "v1"

	// V1 mode - simple route file
	RoutesFile string `json:"routes_file,omitempty"`

	// V2 mode - inline routes with advanced features
	Routes []map[string]interface{} `json:"routes,omitempty"` // ← NEW: raw JSON for v2

	// Proxy mode
	UnixSocket string `json:"unix_socket,omitempty"`

	// V2 features
	Middleware []map[string]interface{} `json:"middleware,omitempty"` // ← NEW
}
