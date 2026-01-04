package v1

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/axis0047/mockingGOD/internal/ir"
)

type RawConfig struct {
	Method   string         `json:"method"`
	Path     string         `json:"path"`
	Response map[string]any `json:"response"`
}

func Compile(path string) ([]ir.Route, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfgs []RawConfig
	if err := json.Unmarshal(raw, &cfgs); err != nil {
		return nil, err
	}

	var routes []ir.Route

	for _, c := range cfgs {
		segs := strings.Split(strings.Trim(c.Path, "/"), "/")

		var rules []ir.ResponseRule
		for k, v := range c.Response {
			rules = append(rules, ir.ResponseRule{
				Target: k,
				Source: ir.StaticValue{Value: v},
			})
		}

		routes = append(routes, ir.Route{
			Method:     c.Method,
			Path:       ir.PathTemplate{Segments: segs},
			Validators: nil,
			Response:   rules,
		})
	}

	return routes, nil
}
