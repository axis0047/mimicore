package utils

import (
	"regexp"
	"strings"
)

// ConvertPath converts OpenAPI style "{param}" to httprouter style ":param"
func ConvertPath(path string) string {
	// If it doesn't contain '{', return as is
	if !strings.Contains(path, "{") {
		return path
	}

	// Regex to find {param}
	re := regexp.MustCompile(`\{([^}]+)\}`)

	// Replace {param} with :param
	return re.ReplaceAllString(path, ":$1")
}
