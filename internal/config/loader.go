package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func LoadAll(dir string) ([]APIConfig, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	var apis []APIConfig

	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			continue
		}

		// Derive API name from filename (without extension)
		apiName := strings.TrimSuffix(filepath.Base(f), ".json")

		// Detect version by peeking at the file structure
		version := detectVersion(f)

		apis = append(apis, APIConfig{
			API:        apiName,
			Mode:       ModeIR,
			Version:    version,
			RoutesFile: f,
		})
	}

	return apis, nil
}

// detectVersion peeks at the route file to determine if it's v1 or v2
func detectVersion(filepath string) string {
	raw, err := os.ReadFile(filepath)
	if err != nil {
		return "v1" // Default to v1 on error
	}

	// Parse as generic array
	var routes []map[string]interface{}
	if err := json.Unmarshal(raw, &routes); err != nil {
		return "v1"
	}

	if len(routes) == 0 {
		return "v1"
	}

	// Check if first route has v2-specific fields
	firstRoute := routes[0]
	if _, hasValidate := firstRoute["validate"]; hasValidate {
		return "v2"
	}
	if _, hasTransform := firstRoute["transform"]; hasTransform {
		return "v2"
	}

	return "v1"
}
