package v1

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/axis0047/mockingGOD/internal/ir"
)

type Adapter struct {
	Path string
}

func (a *Adapter) Compile() ([]ir.Route, error) {
	raw, err := os.ReadFile(a.Path)
	if err != nil {
		return nil, err
	}

	var cfgs []struct {
		Method   string         `json:"method"`
		Path     string         `json:"path"`
		Response map[string]any `json:"response"`
	}

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
