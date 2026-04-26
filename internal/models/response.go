package models

import (
	"bytes"
	"encoding/json"
)

type CommonResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func NewErrorResponseBuffer() []byte {
	result := CommonResponse{
		Status: "error",
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(result); err != nil {
		return []byte{}
	}
	return buf.Bytes()
}
