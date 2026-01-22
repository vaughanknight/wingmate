package mcp

import (
	"errors"
	"fmt"
)

// MCP error codes per ADR-004 allocation (3001-3099).
// See plan § 7 Cross-Cutting Concerns for full allocation.
const (
	// Tool execution errors (3001-3009)

	// ErrCodeCLIUnavailable indicates Claude CLI is not installed or not found.
	ErrCodeCLIUnavailable = 3001

	// ErrCodeExecutionFailed indicates a tool execution failure.
	ErrCodeExecutionFailed = 3002

	// ErrCodeTimeout indicates a tool execution timeout.
	ErrCodeTimeout = 3003

	// ErrCodeInvalidInput indicates invalid tool input.
	ErrCodeInvalidInput = 3004

	// ErrCodeSessionNotFound indicates the requested session doesn't exist.
	ErrCodeSessionNotFound = 3005

	// Server lifecycle errors (3010-3019)

	// ErrCodeInitializationFailed indicates server initialization failed.
	ErrCodeInitializationFailed = 3010

	// ErrCodeNotInitialized indicates a request was made before initialization.
	ErrCodeNotInitialized = 3011

	// ErrCodeShutdownInProgress indicates the server is shutting down.
	ErrCodeShutdownInProgress = 3012

	// Transport errors (3020-3029)

	// ErrCodeInvalidJSON indicates invalid JSON in request/response.
	ErrCodeInvalidJSON = 3020

	// ErrCodeMessageTooLarge indicates the message exceeds size limits.
	ErrCodeMessageTooLarge = 3021

	// ErrCodeTransportError indicates a transport-level error.
	ErrCodeTransportError = 3022
)

// MCPError represents an MCP-specific error with a code and message.
type MCPError struct {
	// Code is the MCP error code (3001-3099).
	Code int

	// Message is a human-readable error description.
	Message string

	// Cause is the underlying error, if any.
	Cause error
}

// Error implements the error interface.
func (e *MCPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("MCP error %d: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *MCPError) Unwrap() error {
	return e.Cause
}

// NewMCPError creates a new MCPError with the given code and message.
func NewMCPError(code int, message string) *MCPError {
	return &MCPError{
		Code:    code,
		Message: message,
	}
}

// WrapMCPError creates a new MCPError wrapping an underlying error.
func WrapMCPError(code int, message string, cause error) *MCPError {
	return &MCPError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Predefined errors for common conditions.
var (
	// ErrCLIUnavailable indicates Claude CLI is not installed or not found.
	ErrCLIUnavailable = NewMCPError(ErrCodeCLIUnavailable, "Claude CLI not available")

	// ErrNotInitialized indicates a request was made before server initialization.
	ErrNotInitialized = NewMCPError(ErrCodeNotInitialized, "server not initialized")

	// ErrShutdownInProgress indicates the server is shutting down.
	ErrShutdownInProgress = NewMCPError(ErrCodeShutdownInProgress, "server shutdown in progress")
)

// IsMCPError checks if an error is an MCPError with a specific code.
func IsMCPError(err error, code int) bool {
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		return mcpErr.Code == code
	}
	return false
}

// GetMCPErrorCode returns the MCP error code from an error, or 0 if not an MCPError.
func GetMCPErrorCode(err error) int {
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		return mcpErr.Code
	}
	return 0
}
