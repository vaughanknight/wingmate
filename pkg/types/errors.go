package types

import "fmt"

// Standard JSON-RPC 2.0 error codes (reserved range).
// We use these for protocol-level errors.
const (
	// ParseError indicates invalid JSON was received.
	ParseError = -32700

	// InvalidRequest indicates the JSON sent is not a valid Request object.
	InvalidRequest = -32600

	// MethodNotFound indicates the method does not exist or is not available.
	MethodNotFound = -32601

	// InvalidParams indicates invalid method parameters.
	InvalidParams = -32602

	// InternalError indicates an internal JSON-RPC error.
	InternalError = -32603
)

// Application-specific error codes (1000+ range).
// Per A2A guidance, we use 1000+ to avoid reserved -32000 range.
const (
	// ErrCodeAgentUnavailable indicates the target agent is not reachable.
	ErrCodeAgentUnavailable = 1001

	// ErrCodeTaskNotFound indicates the requested task does not exist.
	ErrCodeTaskNotFound = 1002

	// ErrCodeTaskFailed indicates a task execution failure.
	ErrCodeTaskFailed = 1003

	// ErrCodeAuthRequired indicates authentication is required.
	ErrCodeAuthRequired = 1004

	// ErrCodeAuthFailed indicates authentication failed.
	ErrCodeAuthFailed = 1005

	// ErrCodeRateLimited indicates rate limit exceeded.
	ErrCodeRateLimited = 1006

	// ErrCodeTimeout indicates operation timed out.
	ErrCodeTimeout = 1007
)

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	// Code is the error code.
	Code int `json:"code"`

	// Message is a short error description.
	Message string `json:"message"`

	// Data contains additional error information (optional).
	Data interface{} `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *JSONRPCError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("JSON-RPC error %d: %s (data: %v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// NewJSONRPCError creates a new JSONRPCError.
func NewJSONRPCError(code int, message string, data interface{}) *JSONRPCError {
	return &JSONRPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// Standard error constructors for common cases.

// ErrParseError creates a parse error response.
func ErrParseError(detail string) *JSONRPCError {
	return NewJSONRPCError(ParseError, "Parse error", detail)
}

// ErrInvalidRequest creates an invalid request error.
func ErrInvalidRequest(detail string) *JSONRPCError {
	return NewJSONRPCError(InvalidRequest, "Invalid request", detail)
}

// ErrMethodNotFound creates a method not found error.
func ErrMethodNotFound(method string) *JSONRPCError {
	return NewJSONRPCError(MethodNotFound, "Method not found", method)
}

// ErrInvalidParams creates an invalid params error.
func ErrInvalidParams(detail string) *JSONRPCError {
	return NewJSONRPCError(InvalidParams, "Invalid params", detail)
}

// ErrInternalError creates an internal error.
func ErrInternalError(detail string) *JSONRPCError {
	return NewJSONRPCError(InternalError, "Internal error", detail)
}

// Application error constructors.

// ErrAgentUnavailable creates an agent unavailable error.
func ErrAgentUnavailable(url string) *JSONRPCError {
	return NewJSONRPCError(ErrCodeAgentUnavailable, "Agent unavailable", url)
}

// ErrTaskNotFound creates a task not found error.
func ErrTaskNotFound(taskID string) *JSONRPCError {
	return NewJSONRPCError(ErrCodeTaskNotFound, "Task not found", taskID)
}

// ErrTaskFailed creates a task failed error.
func ErrTaskFailed(detail string) *JSONRPCError {
	return NewJSONRPCError(ErrCodeTaskFailed, "Task failed", detail)
}

// ErrTimeout creates a timeout error.
func ErrTimeout(operation string) *JSONRPCError {
	return NewJSONRPCError(ErrCodeTimeout, "Operation timed out", operation)
}
