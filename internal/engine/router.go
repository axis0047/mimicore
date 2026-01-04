package engine

import (
	"net/http"
	"strings"

	"github.com/axis0047/mockingGOD/internal/ir"
)

type Router struct {
	Routes []ir.Route
}

func (r *Router) Match(req *http.Request) (*ir.Route, map[string]string) {
	pathSegs := strings.Split(strings.Trim(req.URL.Path, "/"), "/")

	for _, route := range r.Routes {
		if route.Method != req.Method {
			continue
		}

		if len(route.Path.Segments) != len(pathSegs) {
			continue
		}

		params := map[string]string{}
		matched := true

		for i, seg := range route.Path.Segments {
			if strings.HasPrefix(seg, "{") {
				key := strings.Trim(seg, "{}")
				params[key] = pathSegs[i]
			} else if seg != pathSegs[i] {
				matched = false
				break
			}
		}

		if matched {
			return &route, params
		}
	}

	return nil, nil
}
