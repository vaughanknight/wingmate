// Package llm provides types and interfaces for LLM integration via CLI.
package llm

import "fmt"

// LLM-specific error codes.
// These are in the range 2001-2006 to avoid conflicts with protocol error codes.
const (
	// CodeLLMUnavailable indicates the LLM provider is not installed or accessible.
	// For CLI providers, this means the binary is not in PATH.
	CodeLLMUnavailable = 2001

	// CodeLLMExecutionFailed indicates the LLM command failed with a blocking error.
	// This maps to CLI exit code 2.
	CodeLLMExecutionFailed = 2002

	// CodeLLMTimeout indicates the LLM request timed out.
	// This occurs when the context deadline is exceeded.
	CodeLLMTimeout = 2003

	// CodeLLMNonBlocking indicates a non-blocking error occurred.
	// This maps to CLI exit code 1; the response may still contain partial data.
	CodeLLMNonBlocking = 2004

	// CodeLLMSessionNotFound indicates the requested session does not exist.
	// This can happen if the session expired or was never created.
	CodeLLMSessionNotFound = 2005

	// CodeLLMInvalidResponse indicates the LLM returned an invalid or unparseable response.
	// This typically means JSON parsing failed.
	CodeLLMInvalidResponse = 2006
)

// LLMError represents an error from LLM operations.
// It includes an error code for programmatic handling and a message for users.
type LLMError struct {
	// Code is one of the CodeLLM* constants.
	Code int

	// Message is a human-readable error description.
	Message string

	// Err is the underlying error, if any.
	Err error
}

// Error implements the error interface.
func (e *LLMError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("llm error %d: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("llm error %d: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Unwrap support.
func (e *LLMError) Unwrap() error {
	return e.Err
}

// NewLLMError creates a new LLMError with the given code and message.
func NewLLMError(code int, message string) *LLMError {
	return &LLMError{Code: code, Message: message}
}

// WrapLLMError creates a new LLMError that wraps an existing error.
func WrapLLMError(code int, message string, err error) *LLMError {
	return &LLMError{Code: code, Message: message, Err: err}
}

// Predefined errors for common failure scenarios.
var (
	// ErrLLMUnavailable is returned when the LLM provider is not installed.
	ErrLLMUnavailable = &LLMError{
		Code:    CodeLLMUnavailable,
		Message: "Claude CLI not installed - install from https://claude.ai/download",
	}

	// ErrLLMTimeout is returned when the LLM request times out.
	ErrLLMTimeout = &LLMError{
		Code:    CodeLLMTimeout,
		Message: "Claude CLI request timed out",
	}
)
