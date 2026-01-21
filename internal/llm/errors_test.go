// Package llm provides types and interfaces for LLM integration.
//
// Why: Tests verify that LLM error codes are correctly defined and that
// the LLMError type properly formats error messages.
//
// Contract: Each error code must be unique and non-zero. LLMError.Error()
// must produce a readable string. Unwrap() must return the wrapped error.
//
// Usage Notes: Run with `go test -v ./internal/llm/... -run Error`
//
// Quality Contribution: Catches incorrect error code values, duplicate codes,
// and formatting issues that would confuse debugging.
//
// Worked Example: TestLLMError_Error demonstrates the expected error string format.
package llm

import (
	"errors"
	"testing"
)

func TestLLMError_Codes_NonZero(t *testing.T) {
	// All error codes should be non-zero
	codes := []struct {
		name string
		code int
	}{
		{"CodeLLMUnavailable", CodeLLMUnavailable},
		{"CodeLLMExecutionFailed", CodeLLMExecutionFailed},
		{"CodeLLMTimeout", CodeLLMTimeout},
		{"CodeLLMNonBlocking", CodeLLMNonBlocking},
		{"CodeLLMSessionNotFound", CodeLLMSessionNotFound},
		{"CodeLLMInvalidResponse", CodeLLMInvalidResponse},
	}

	for _, tc := range codes {
		if tc.code == 0 {
			t.Errorf("%s is zero, want non-zero", tc.name)
		}
	}
}

func TestLLMError_Codes_Unique(t *testing.T) {
	// All error codes should be unique
	codes := map[int]string{
		CodeLLMUnavailable:     "CodeLLMUnavailable",
		CodeLLMExecutionFailed: "CodeLLMExecutionFailed",
		CodeLLMTimeout:         "CodeLLMTimeout",
		CodeLLMNonBlocking:     "CodeLLMNonBlocking",
		CodeLLMSessionNotFound: "CodeLLMSessionNotFound",
		CodeLLMInvalidResponse: "CodeLLMInvalidResponse",
	}

	// If any codes are duplicates, the map will have fewer entries
	if len(codes) != 6 {
		t.Errorf("expected 6 unique codes, got %d (duplicates exist)", len(codes))
	}
}

func TestLLMError_Codes_InRange(t *testing.T) {
	// All error codes should be in the 2001-2006 range
	codes := []struct {
		name string
		code int
	}{
		{"CodeLLMUnavailable", CodeLLMUnavailable},
		{"CodeLLMExecutionFailed", CodeLLMExecutionFailed},
		{"CodeLLMTimeout", CodeLLMTimeout},
		{"CodeLLMNonBlocking", CodeLLMNonBlocking},
		{"CodeLLMSessionNotFound", CodeLLMSessionNotFound},
		{"CodeLLMInvalidResponse", CodeLLMInvalidResponse},
	}

	for _, tc := range codes {
		if tc.code < 2001 || tc.code > 2006 {
			t.Errorf("%s = %d, want in range [2001, 2006]", tc.name, tc.code)
		}
	}
}

func TestLLMError_Error_WithoutWrapped(t *testing.T) {
	// Given: An LLMError without a wrapped error
	err := &LLMError{
		Code:    CodeLLMUnavailable,
		Message: "test message",
	}

	// When: We call Error()
	result := err.Error()

	// Then: It should include code and message
	expected := "llm error 2001: test message"
	if result != expected {
		t.Errorf("Error() = %q, want %q", result, expected)
	}
}

func TestLLMError_Error_WithWrapped(t *testing.T) {
	// Given: An LLMError with a wrapped error
	underlying := errors.New("underlying cause")
	err := &LLMError{
		Code:    CodeLLMExecutionFailed,
		Message: "execution failed",
		Err:     underlying,
	}

	// When: We call Error()
	result := err.Error()

	// Then: It should include code, message, and wrapped error
	expected := "llm error 2002: execution failed: underlying cause"
	if result != expected {
		t.Errorf("Error() = %q, want %q", result, expected)
	}
}

func TestLLMError_Unwrap(t *testing.T) {
	// Given: An LLMError with a wrapped error
	underlying := errors.New("wrapped error")
	err := &LLMError{
		Code:    CodeLLMTimeout,
		Message: "timeout",
		Err:     underlying,
	}

	// When: We call Unwrap()
	result := err.Unwrap()

	// Then: It should return the wrapped error
	if result != underlying {
		t.Errorf("Unwrap() = %v, want %v", result, underlying)
	}
}

func TestLLMError_Unwrap_Nil(t *testing.T) {
	// Given: An LLMError without a wrapped error
	err := &LLMError{
		Code:    CodeLLMNonBlocking,
		Message: "no wrap",
	}

	// When: We call Unwrap()
	result := err.Unwrap()

	// Then: It should return nil
	if result != nil {
		t.Errorf("Unwrap() = %v, want nil", result)
	}
}

func TestNewLLMError(t *testing.T) {
	// When: We create an error with NewLLMError
	err := NewLLMError(CodeLLMSessionNotFound, "session not found")

	// Then: Fields should be set correctly
	if err.Code != CodeLLMSessionNotFound {
		t.Errorf("Code = %d, want %d", err.Code, CodeLLMSessionNotFound)
	}

	if err.Message != "session not found" {
		t.Errorf("Message = %q, want %q", err.Message, "session not found")
	}

	if err.Err != nil {
		t.Errorf("Err = %v, want nil", err.Err)
	}
}

func TestWrapLLMError(t *testing.T) {
	// Given: An underlying error
	underlying := errors.New("parse failed")

	// When: We wrap it with WrapLLMError
	err := WrapLLMError(CodeLLMInvalidResponse, "invalid JSON", underlying)

	// Then: All fields should be set correctly
	if err.Code != CodeLLMInvalidResponse {
		t.Errorf("Code = %d, want %d", err.Code, CodeLLMInvalidResponse)
	}

	if err.Message != "invalid JSON" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid JSON")
	}

	if err.Err != underlying {
		t.Errorf("Err = %v, want %v", err.Err, underlying)
	}
}

func TestPredefinedErrors(t *testing.T) {
	// Verify predefined errors have correct codes
	tests := []struct {
		name     string
		err      *LLMError
		wantCode int
	}{
		{"ErrLLMUnavailable", ErrLLMUnavailable, CodeLLMUnavailable},
		{"ErrLLMTimeout", ErrLLMTimeout, CodeLLMTimeout},
	}

	for _, tc := range tests {
		if tc.err.Code != tc.wantCode {
			t.Errorf("%s.Code = %d, want %d", tc.name, tc.err.Code, tc.wantCode)
		}

		if tc.err.Message == "" {
			t.Errorf("%s.Message is empty", tc.name)
		}
	}
}
