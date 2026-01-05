package engine

import (
	"github.com/axis0047/mockingGOD/internal/ir"
	"github.com/axis0047/mockingGOD/internal/utils" // <-- Ensure this import is correct
)

func BuildResponse(rules []ir.ResponseRule, ctx map[string]any) map[string]any {
	resp := map[string]any{}

	for _, rule := range rules {
		val, _ := rule.Source.Resolve(ctx)
		// Use the correct SetNested from the utils package
		utils.SetNested(resp, rule.Target, val)
	}

	return resp
}
