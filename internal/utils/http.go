package utils

import (
	"net/http"

	json "github.com/goccy/go-json"
)

// WriteJSON writes a JSON response with status code
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// goccy/go-json is much faster at encoding
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteError writes a standard JSON error response
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]any{
		"error": message,
	})
}
