package protocol

import (
	"encoding/json"
)

// JSON-RPC 2.0 version constant.
const JSONRPCVersion = "2.0"

// Request represents a JSON-RPC 2.0 request.
// Per JSON-RPC spec: https://www.jsonrpc.org/specification
type Request struct {
	// JSONRPC specifies the JSON-RPC version. MUST be "2.0".
	JSONRPC string `json:"jsonrpc"`

	// ID is the request identifier. Can be string, number, or null.
	// Notifications omit this field entirely.
	ID json.RawMessage `json:"id,omitempty"`

	// Method is the name of the method to be invoked.
	Method string `json:"method"`

	// Params holds the parameter values to be used during method invocation.
	// Can be omitted if no parameters.
	Params json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
// Either Result or Error will be set, never both.
type Response struct {
	// JSONRPC specifies the JSON-RPC version. MUST be "2.0".
	JSONRPC string `json:"jsonrpc"`

	// ID matches the request ID. Null for parse errors.
	ID json.RawMessage `json:"id"`

	// Result contains the result on success. Omitted on error.
	Result json.RawMessage `json:"result,omitempty"`

	// Error contains the error on failure. Omitted on success.
	Error *ErrorObject `json:"error,omitempty"`
}

// ErrorObject represents a JSON-RPC 2.0 error object.
type ErrorObject struct {
	// Code is the error code (integer).
	Code int `json:"code"`

	// Message is a short description of the error.
	Message string `json:"message"`

	// Data contains additional information about the error.
	// This may be omitted.
	Data json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *ErrorObject) Error() string {
	return e.Message
}

// IsNotification returns true if this is a notification (no ID).
func (r *Request) IsNotification() bool {
	return len(r.ID) == 0
}

// NewRequest creates a new JSON-RPC 2.0 request.
// The id can be any type that can be marshaled to JSON (typically int or string).
// The params can be any type that can be marshaled to JSON.
func NewRequest(id interface{}, method string, params interface{}) (*Request, error) {
	req := &Request{
		JSONRPC: JSONRPCVersion,
		Method:  method,
	}

	// Marshal ID if provided
	if id != nil {
		idBytes, err := json.Marshal(id)
		if err != nil {
			return nil, err
		}
		req.ID = idBytes
	}

	// Marshal params if provided
	if params != nil {
		paramsBytes, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		req.Params = paramsBytes
	}

	return req, nil
}

// NewSuccessResponse creates a new successful JSON-RPC 2.0 response.
// The result can be any type that can be marshaled to JSON.
func NewSuccessResponse(id json.RawMessage, result interface{}) (*Response, error) {
	resp := &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
	}

	if result != nil {
		resultBytes, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		resp.Result = resultBytes
	}

	return resp, nil
}

// NewErrorResponse creates a new error JSON-RPC 2.0 response.
// The data parameter can be nil if no additional data is needed.
func NewErrorResponse(id json.RawMessage, code int, message string, data interface{}) *Response {
	resp := &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &ErrorObject{
			Code:    code,
			Message: message,
		},
	}

	if data != nil {
		// Best effort marshal, ignore errors for error response creation
		if dataBytes, err := json.Marshal(data); err == nil {
			resp.Error.Data = dataBytes
		}
	}

	return resp
}

// ParseRequest parses a JSON-RPC request from raw bytes.
func ParseRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// ParseResponse parses a JSON-RPC response from raw bytes.
func ParseResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
