package engine

import (
	"strings"
)

// SetNested sets a value in a nested map[string]any using dot notation.
// Example: SetNested(m, "user.name", "John") => m["user"]["name"] = "John"
func SetNested(m map[string]any, dottedKey string, value any) {
	keys := strings.Split(dottedKey, ".")

	curr := m
	for i, key := range keys {
		if i == len(keys)-1 {
			curr[key] = value
			return
		}

		// If next level does not exist or is not a map, create it
		if next, ok := curr[key]; ok {
			if nextMap, ok := next.(map[string]any); ok {
				curr = nextMap
			} else {
				newMap := make(map[string]any)
				curr[key] = newMap
				curr = newMap
			}
		} else {
			newMap := make(map[string]any)
			curr[key] = newMap
			curr = newMap
		}
	}
}
