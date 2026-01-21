// Package llm provides types and interfaces for LLM integration.
//
// Why: Tests verify that CLI responses are correctly converted to A2A message
// format for use in agent communication.
//
// Contract: FromCLIResponse must create a valid types.Message with the response
// text, preserving session information and handling edge cases.
//
// Usage Notes: Run with `go test -v ./internal/llm/... -run Convert`
//
// Quality Contribution: Ensures CLI integration produces valid A2A messages,
// catching format mismatches that would break agent communication.
//
// Worked Example: TestFromCLIResponse_Basic demonstrates conversion of a
// typical CLI response to an A2A message.
package llm

import (
	"testing"
)

func TestFromCLIResponse_Basic(t *testing.T) {
	// Given: A CLI response with result text
	resp := &CLIResponse{
		Result:    "Hello! How can I help you?",
		SessionID: "session-123",
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 20,
		},
		Metadata: Metadata{
			Model: "claude-sonnet-4",
		},
	}

	// When: We convert to A2A message
	msg, err := FromCLIResponse(resp)

	// Then: It should succeed
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// And: The message should have correct role
	if msg.Role != "assistant" {
		t.Errorf("Role = %q, want %q", msg.Role, "assistant")
	}

	// And: The message should have one text part
	if len(msg.Parts) != 1 {
		t.Fatalf("len(Parts) = %d, want 1", len(msg.Parts))
	}

	part := msg.Parts[0]
	if part.Kind != "text" {
		t.Errorf("Parts[0].Kind = %q, want %q", part.Kind, "text")
	}

	if part.Text != "Hello! How can I help you?" {
		t.Errorf("Parts[0].Text = %q, want %q", part.Text, "Hello! How can I help you?")
	}
}

func TestFromCLIResponse_EmptyResult(t *testing.T) {
	// Given: A CLI response with empty result
	resp := &CLIResponse{
		Result:    "",
		SessionID: "session-456",
	}

	// When: We convert to A2A message
	msg, err := FromCLIResponse(resp)

	// Then: It should succeed
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// And: The message should have an empty text part
	if len(msg.Parts) != 1 {
		t.Fatalf("len(Parts) = %d, want 1", len(msg.Parts))
	}

	if msg.Parts[0].Text != "" {
		t.Errorf("Parts[0].Text = %q, want empty string", msg.Parts[0].Text)
	}
}

func TestFromCLIResponse_NilInput(t *testing.T) {
	// Given: A nil response
	var resp *CLIResponse = nil

	// When: We try to convert
	msg, err := FromCLIResponse(resp)

	// Then: It should return an error
	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}

	if msg != nil {
		t.Error("expected nil message when error occurs")
	}

	// And: The error should be an LLMError
	llmErr, ok := err.(*LLMError)
	if !ok {
		t.Fatalf("expected *LLMError, got %T", err)
	}

	if llmErr.Code != CodeLLMInvalidResponse {
		t.Errorf("Code = %d, want %d", llmErr.Code, CodeLLMInvalidResponse)
	}
}

func TestFromCLIResponse_PreservesWhitespace(t *testing.T) {
	// Given: A response with whitespace in the result
	resp := &CLIResponse{
		Result: "  \n  Hello  \n  World  \n  ",
	}

	// When: We convert to A2A message
	msg, err := FromCLIResponse(resp)

	// Then: Whitespace should be preserved
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "  \n  Hello  \n  World  \n  "
	if msg.Parts[0].Text != expected {
		t.Errorf("Parts[0].Text = %q, want %q", msg.Parts[0].Text, expected)
	}
}

func TestFromCLIResponse_MultilineResult(t *testing.T) {
	// Given: A response with multiple lines
	resp := &CLIResponse{
		Result: "Line 1\nLine 2\nLine 3",
	}

	// When: We convert to A2A message
	msg, err := FromCLIResponse(resp)

	// Then: All lines should be preserved
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Line 1\nLine 2\nLine 3"
	if msg.Parts[0].Text != expected {
		t.Errorf("Parts[0].Text = %q, want %q", msg.Parts[0].Text, expected)
	}
}
