package config

type APIMode string

const (
	ModeIR    APIMode = "ir"
	ModeProxy APIMode = "proxy"
)

type APIConfig struct {
	API  string  `json:"api"`
	Mode APIMode `json:"mode"`

	// IR mode
	RoutesFile string `json:"routes_file,omitempty"`

	// Proxy mode
	UnixSocket string `json:"unix_socket,omitempty"`
}
