package hm

import (
	"encoding/json"
	"net/http"
)

// SuccessResponse define el envelope para respuestas exitosas.
type SuccessResponse struct {
	Data interface{} `json:"data"`
	Meta interface{} `json:"meta,omitempty"`
}

// ErrorPayload define la estructura interna del objeto de error.
type ErrorPayload struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details []ValidationError `json:"details,omitempty"`
}

// ErrorResponse define el envelope para respuestas de error.
type ErrorResponse struct {
	Error ErrorPayload `json:"error"`
}

// Respond envía una respuesta JSON exitosa.
func Respond(w http.ResponseWriter, code int, data interface{}, meta interface{}) {
	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(SuccessResponse{Data: data, Meta: meta})
}

// Error envía una respuesta JSON de error.
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
