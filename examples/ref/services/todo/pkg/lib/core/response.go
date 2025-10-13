package core

import (
	"encoding/json"
	"net/http"
)

// SuccessResponse defines the envelope for successful responses.
type SuccessResponse struct {
	Data interface{} `json:"data"`
	Meta interface{} `json:"meta,omitempty"`
}

// ErrorPayload defines the internal structure of the error object.
type ErrorPayload struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details []ValidationError `json:"details,omitempty"`
}

// ErrorResponse defines the envelope for error responses.
type ErrorResponse struct {
	Error ErrorPayload `json:"error"`
}

// Respond sends a successful JSON response.
func Respond(w http.ResponseWriter, code int, data interface{}, meta interface{}) {
	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(SuccessResponse{Data: data, Meta: meta})
}

// Error sends a JSON error response.
func Error(w http.ResponseWriter, code int, errorCode string, message string, details ...ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorPayload{
			Code:    errorCode,
			Message: message,
			Details: details,
		},
	})
}
