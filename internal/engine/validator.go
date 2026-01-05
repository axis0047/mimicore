package engine

import "github.com/axis0047/mockingGOD/internal/ir"

func RunValidators(validators []ir.Validator, input map[string]any, ctx map[string]any) error {
	for _, v := range validators {
		if err := v.Validate(input, ctx); err != nil {
			return err
		}
	}
	return nil
}
