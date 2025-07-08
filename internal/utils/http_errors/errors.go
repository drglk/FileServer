package utils

import (
	"encoding/json"
	"net/http"
)

func WriteJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code": statusCode,
			"text": message,
		},
	})
}

func WriteStatusError(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)
}
