package view

import (
	"encoding/json"
	"fmt"

	"github.com/Noooste/fhttp"

	"github.com/Noooste/azuretls-api/protocol"
)

type ResponseWriter struct{}

func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{}
}

// WriteResponse writes a response using the specified encoder
func (rw *ResponseWriter) WriteResponse(w http.ResponseWriter, data any, statusCode int, encoder protocol.MessageEncoder) {
	if encoder == nil {
		encoder = protocol.GetJSONEncoder()
	}

	responseBytes, err := encoder.Encode(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", encoder.ContentType())
	w.WriteHeader(statusCode)
	w.Write(responseBytes)
}

// WriteErrorResponse writes an error response
func (rw *ResponseWriter) WriteErrorResponse(w http.ResponseWriter, message string, statusCode int, encoder protocol.MessageEncoder) {
	if encoder == nil {
		encoder = protocol.GetJSONEncoder()
	}

	errorResponse := map[string]any{
		"error":  message,
		"status": statusCode,
	}

	rw.WriteResponse(w, errorResponse, statusCode, encoder)
}

// WriteJSONResponse writes a JSON response directly
func (rw *ResponseWriter) WriteJSONResponse(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// WriteSuccessResponse writes a success response
func (rw *ResponseWriter) WriteSuccessResponse(w http.ResponseWriter) {
	rw.WriteJSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

// WriteCreatedResponse writes a creation success response
func (rw *ResponseWriter) WriteCreatedResponse(w http.ResponseWriter, data any, encoder protocol.MessageEncoder) {
	rw.WriteResponse(w, data, http.StatusCreated, encoder)
}
