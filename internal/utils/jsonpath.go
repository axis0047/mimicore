package utils

import "strings"

func SetNested(m map[string]any, path string, value any) {
	keys := strings.Split(path, ".")
	cur := m
	for _, k := range keys[:len(keys)-1] {
		if _, ok := cur[k]; !ok {
			cur[k] = map[string]any{}
		}
		cur = cur[k].(map[string]any)
	}
	cur[keys[len(keys)-1]] = value
}
