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

		apiName := strings.TrimSuffix(filepath.Base(f), ".json")
		version := detectVersion(f) // <--- Logic updated below

		apis = append(apis, APIConfig{
			API:        apiName,
			Mode:       ModeIR,
			Version:    version,
			RoutesFile: f,
		})
	}

	return apis, nil
}

func detectVersion(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "v1"
	}

	// 1. Try parsing as a Generic Map (New Object Format)
	// If the file starts with '{', it's likely an Object.
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		// Check explicit version field
		if v, ok := obj["version"].(string); ok {
			return v
		}
		// If it's an object but has "routes" key, assume v2
		if _, ok := obj["routes"]; ok {
			return "v2"
		}
		// Fallback for objects
		return "v1"
	}

	// 2. Try parsing as Array (Legacy V1/V2 Format)
	var routes []map[string]interface{}
	if err := json.Unmarshal(raw, &routes); err == nil {
		if len(routes) == 0 {
			return "v1"
		}
		// Check for V2-specific fields in the first route
		if _, ok := routes[0]["validate"]; ok {
			return "v2"
		}
		if _, ok := routes[0]["transform"]; ok {
			return "v2"
		}
		return "v1"
	}

	return "v1"
}
