package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func LoadAll(dir string) ([]APIConfig, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	var apis []APIConfig

	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		var cfg APIConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			continue
		}

		apis = append(apis, cfg)
	}

	return apis, nil
}
