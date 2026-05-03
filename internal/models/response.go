package models

import (
	"bytes"
	"encoding/json"
)

// CommonResponse defines common response structure with status and message
type CommonResponse struct {
	// Status response status
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// NewErrorResponseBuffer creates byte buffer for error response
// to be passed in Write method of ResponseWriter
func NewErrorResponseBuffer(message string) []byte {
	result := CommonResponse{
		Status:  "error",
		Message: message,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(result); err != nil {
		return []byte{}
	}
	return buf.Bytes()
}

// NewSuccessResponseBuffer creates byte buffer for success response
// to be passed in Write method of ResponseWriter
func NewSuccessResponseBuffer(message string) []byte {
	result := CommonResponse{
		Status:  "ok",
		Message: message,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(result); err != nil {
		return []byte{}
	}
	return buf.Bytes()
}
