package engine

import (
	"net/http"
	"strings"

	"github.com/axis0047/mockingGOD/internal/ir"
)

// Match matches an incoming HTTP request to a route in the router.
// Returns the matched route and path parameters.
func (r *Router) Match(req *http.Request) (*ir.Route, map[string]string) {
	path := strings.Trim(req.URL.Path, "/")
	segments := strings.Split(path, "/")

	for _, route := range r.Routes {
		if route.Method != req.Method {
			continue
		}

		params := make(map[string]string)
		if matchSegments(route.Path.Segments, segments, params) {
			return &route, params
		}
	}

	return nil, nil
}

// matchSegments checks if the route segments match the request path segments
// and extracts dynamic parameters.
func matchSegments(routeSegs, reqSegs []string, params map[string]string) bool {
	if len(routeSegs) != len(reqSegs) {
		return false
	}

	for i := range routeSegs {
		rSeg := routeSegs[i]
		sSeg := reqSegs[i]

		// Dynamic segments like {id}
		if strings.HasPrefix(rSeg, "{") && strings.HasSuffix(rSeg, "}") {
			paramName := rSeg[1 : len(rSeg)-1]
			params[paramName] = sSeg
		} else if rSeg != sSeg {
			return false
		}
	}
	return true
}
