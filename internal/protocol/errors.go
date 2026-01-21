package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Standard JSON-RPC 2.0 error codes.
// These are in the reserved range -32700 to -32600.
const (
	// CodeParseError indicates invalid JSON was received by the server.
	CodeParseError = -32700

	// CodeInvalidRequest indicates the JSON sent is not a valid Request object.
	CodeInvalidRequest = -32600

	// CodeMethodNotFound indicates the method does not exist or is not available.
	CodeMethodNotFound = -32601

	// CodeInvalidParams indicates invalid method parameter(s).
	CodeInvalidParams = -32602

	// CodeInternalError indicates an internal JSON-RPC error.
	CodeInternalError = -32603
)

// Standard error messages for JSON-RPC errors.
const (
	MsgParseError     = "Parse error"
	MsgInvalidRequest = "Invalid Request"
	MsgMethodNotFound = "Method not found"
	MsgInvalidParams  = "Invalid params"
	MsgInternalError  = "Internal error"
)

// Sentinel errors for protocol-level failures.
// These are not JSON-RPC errors but Go errors for client-side handling.
var (
	// ErrConnectionRefused indicates the server refused the connection.
	ErrConnectionRefused = errors.New("connection refused")

	// ErrTimeout indicates the request timed out.
	ErrTimeout = errors.New("request timeout")

	// ErrInvalidResponse indicates the server returned an invalid response.
	ErrInvalidResponse = errors.New("invalid response")

	// ErrNotA2AAgent indicates the server is not an A2A agent.
	ErrNotA2AAgent = errors.New("not an A2A agent")

	// ErrServerClosed indicates the server has been closed.
	ErrServerClosed = errors.New("server closed")
)

// ProtocolError wraps an error with additional context.
type ProtocolError struct {
	Op  string // Operation that failed (e.g., "connect", "send", "recv")
	URL string // URL involved, if any
	Err error  // Underlying error
}

func (e *ProtocolError) Error() string {
	if e.URL != "" {
		return fmt.Sprintf("%s %s: %v", e.Op, e.URL, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *ProtocolError) Unwrap() error {
	return e.Err
}

// NewProtocolError creates a new ProtocolError.
func NewProtocolError(op, url string, err error) *ProtocolError {
	return &ProtocolError{Op: op, URL: url, Err: err}
}

// NewParseErrorResponse creates a JSON-RPC parse error response (-32700).
func NewParseErrorResponse(id json.RawMessage) *Response {
	return NewErrorResponse(id, CodeParseError, MsgParseError, nil)
}

// NewInvalidRequestResponse creates a JSON-RPC invalid request error response (-32600).
func NewInvalidRequestResponse(id json.RawMessage, details string) *Response {
	var data interface{}
	if details != "" {
		data = map[string]string{"details": details}
	}
	return NewErrorResponse(id, CodeInvalidRequest, MsgInvalidRequest, data)
}

// NewMethodNotFoundResponse creates a JSON-RPC method not found error response (-32601).
func NewMethodNotFoundResponse(id json.RawMessage, method string) *Response {
	var data interface{}
	if method != "" {
		data = map[string]string{"method": method}
	}
	return NewErrorResponse(id, CodeMethodNotFound, MsgMethodNotFound, data)
}

// NewInvalidParamsResponse creates a JSON-RPC invalid params error response (-32602).
func NewInvalidParamsResponse(id json.RawMessage, details string) *Response {
	var data interface{}
	if details != "" {
		data = map[string]string{"details": details}
	}
	return NewErrorResponse(id, CodeInvalidParams, MsgInvalidParams, data)
}

// NewInternalErrorResponse creates a JSON-RPC internal error response (-32603).
func NewInternalErrorResponse(id json.RawMessage, details string) *Response {
	var data interface{}
	if details != "" {
		data = map[string]string{"details": details}
	}
	return NewErrorResponse(id, CodeInternalError, MsgInternalError, data)
}

// IsJSONRPCError checks if the response is an error response.
func IsJSONRPCError(resp *Response) bool {
	return resp != nil && resp.Error != nil
}

// IsParseError checks if the error is a parse error.
func IsParseError(resp *Response) bool {
	return resp != nil && resp.Error != nil && resp.Error.Code == CodeParseError
}

// IsMethodNotFound checks if the error is a method not found error.
func IsMethodNotFound(resp *Response) bool {
	return resp != nil && resp.Error != nil && resp.Error.Code == CodeMethodNotFound
}
