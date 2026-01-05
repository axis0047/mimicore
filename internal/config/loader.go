package config

import (
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
		// Skip if the file doesn't exist or can't be read
		if _, err := os.Stat(f); os.IsNotExist(err) {
			continue
		}

		// Derive API name from filename (without extension)
		apiName := strings.TrimSuffix(filepath.Base(f), ".json")

		apis = append(apis, APIConfig{
			API:        apiName,
			Mode:       ModeIR,
			RoutesFile: f,
		})
	}

	return apis, nil
}
