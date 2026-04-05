package utils

import (
	"encoding/json"
	"net/http"
)

// RespondJSON writes a JSON response
func RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// RespondError writes an error response
func RespondError(w http.ResponseWriter, statusCode int, message string) error {
	return RespondJSON(w, statusCode, map[string]string{"error": message})
}

// ParseJSON parses JSON from request body
func ParseJSON(r *http.Request, dest interface{}) error {
	return json.NewDecoder(r.Body).Decode(dest)
}

// Contains checks if a string slice contains a string
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
