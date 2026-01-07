package engine

import (
	"github.com/axis0047/mockingGOD/internal/ir"
	"github.com/axis0047/mockingGOD/internal/utils"
)

func BuildResponse(rules []ir.ResponseRule, ctx map[string]any) map[string]any {
	resp := map[string]any{}

	// Wrap raw map in MapContext adapter
	safeCtx := MapContext(ctx)

	for _, rule := range rules {
		// Now it satisfies the interface
		val, _ := rule.Source.Resolve(safeCtx)
		utils.SetNested(resp, rule.Target, val)
	}

	return resp
}
