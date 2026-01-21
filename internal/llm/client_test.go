// Package llm provides types and interfaces for LLM integration.
//
// Why: Tests verify that the LLMExecutor interface is implementable and that
// mock implementations can satisfy the contract.
//
// Contract: LLMExecutor defines Execute() and IsInstalled() methods that any
// LLM provider implementation must satisfy.
//
// Usage Notes: Run with `go test -v ./internal/llm/...`
// These tests use a mock implementation to verify the interface.
//
// Quality Contribution: Proves the interface is practical to implement,
// catching design issues before real implementations are built.
//
// Worked Example: TestMockExecutor_SatisfiesInterface demonstrates that
// a simple mock can satisfy the LLMExecutor interface.
package llm

import (
	"context"
	"testing"
)

// MockExecutor is a test implementation of LLMExecutor.
type MockExecutor struct {
	// Installed controls what IsInstalled returns.
	Installed bool

	// Response is returned by Execute on success.
	Response *CLIResponse

	// Err is returned by Execute when set.
	Err error

	// LastPrompt records the prompt passed to Execute.
	LastPrompt string

	// LastSessionID records the sessionID passed to Execute.
	LastSessionID string
}

// Execute implements LLMExecutor.Execute.
func (m *MockExecutor) Execute(ctx context.Context, prompt string, sessionID string) (*CLIResponse, error) {
	m.LastPrompt = prompt
	m.LastSessionID = sessionID

	if m.Err != nil {
		return nil, m.Err
	}
	return m.Response, nil
}

// IsInstalled implements LLMExecutor.IsInstalled.
func (m *MockExecutor) IsInstalled() bool {
	return m.Installed
}

func TestMockExecutor_SatisfiesInterface(t *testing.T) {
	// Given: A MockExecutor
	mock := &MockExecutor{}

	// Then: It should satisfy the LLMExecutor interface
	var _ LLMExecutor = mock
}

func TestMockExecutor_Execute_ReturnsResponse(t *testing.T) {
	// Given: A mock with a canned response
	mock := &MockExecutor{
		Response: &CLIResponse{
			Result:    "Hello from mock",
			SessionID: "mock-session-123",
		},
	}

	// When: We call Execute
	ctx := context.Background()
	resp, err := mock.Execute(ctx, "test prompt", "")

	// Then: It should return the configured response
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Result != "Hello from mock" {
		t.Errorf("Result = %q, want %q", resp.Result, "Hello from mock")
	}

	if resp.SessionID != "mock-session-123" {
		t.Errorf("SessionID = %q, want %q", resp.SessionID, "mock-session-123")
	}

	// And: It should record the prompt
	if mock.LastPrompt != "test prompt" {
		t.Errorf("LastPrompt = %q, want %q", mock.LastPrompt, "test prompt")
	}
}

func TestMockExecutor_Execute_WithSessionID(t *testing.T) {
	// Given: A mock executor
	mock := &MockExecutor{
		Response: &CLIResponse{Result: "ok"},
	}

	// When: We call Execute with a session ID
	ctx := context.Background()
	_, err := mock.Execute(ctx, "follow-up", "existing-session")

	// Then: It should record the session ID
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.LastSessionID != "existing-session" {
		t.Errorf("LastSessionID = %q, want %q", mock.LastSessionID, "existing-session")
	}
}

func TestMockExecutor_Execute_ReturnsError(t *testing.T) {
	// Given: A mock configured to return an error
	mockErr := &LLMError{Code: CodeLLMUnavailable, Message: "mock error"}
	mock := &MockExecutor{
		Err: mockErr,
	}

	// When: We call Execute
	ctx := context.Background()
	resp, err := mock.Execute(ctx, "test", "")

	// Then: It should return the error
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if resp != nil {
		t.Error("expected nil response when error occurs")
	}

	llmErr, ok := err.(*LLMError)
	if !ok {
		t.Fatalf("expected *LLMError, got %T", err)
	}

	if llmErr.Code != CodeLLMUnavailable {
		t.Errorf("Code = %d, want %d", llmErr.Code, CodeLLMUnavailable)
	}
}

func TestMockExecutor_IsInstalled_True(t *testing.T) {
	// Given: A mock with Installed = true
	mock := &MockExecutor{Installed: true}

	// Then: IsInstalled should return true
	if !mock.IsInstalled() {
		t.Error("IsInstalled() = false, want true")
	}
}

func TestMockExecutor_IsInstalled_False(t *testing.T) {
	// Given: A mock with Installed = false
	mock := &MockExecutor{Installed: false}

	// Then: IsInstalled should return false
	if mock.IsInstalled() {
		t.Error("IsInstalled() = true, want false")
	}
}
