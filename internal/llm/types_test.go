// Package llm provides types and interfaces for LLM integration.
//
// Why: Tests verify that CLIResponse types correctly parse Claude CLI JSON output,
// ensuring reliable communication with the CLI.
//
// Contract: CLIResponse must unmarshal valid CLI JSON, handle missing fields
// gracefully, and support round-trip marshaling.
//
// Usage Notes: Run with `go test -v ./internal/llm/...`
// No external dependencies required.
//
// Quality Contribution: These tests catch JSON tag mismatches, missing fields,
// and serialization issues before they cause runtime failures.
//
// Worked Example: TestCLIResponse_UnmarshalJSON demonstrates parsing a complete
// CLI response with all fields populated.
package llm

import (
	"encoding/json"
	"testing"
)

func TestCLIResponse_UnmarshalJSON(t *testing.T) {
	// Given: A complete JSON response from Claude CLI
	input := `{
		"result": "Hello! How can I help you today?",
		"session_id": "550e8400-e29b-41d4-a716-446655440000",
		"usage": {
			"input_tokens": 5,
			"output_tokens": 12
		},
		"metadata": {
			"model": "claude-sonnet-4-20250514"
		}
	}`

	// When: We unmarshal the JSON
	var resp CLIResponse
	err := json.Unmarshal([]byte(input), &resp)

	// Then: All fields should be populated correctly
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Result != "Hello! How can I help you today?" {
		t.Errorf("Result = %q, want %q", resp.Result, "Hello! How can I help you today?")
	}

	if resp.SessionID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("SessionID = %q, want %q", resp.SessionID, "550e8400-e29b-41d4-a716-446655440000")
	}

	if resp.Usage.InputTokens != 5 {
		t.Errorf("Usage.InputTokens = %d, want %d", resp.Usage.InputTokens, 5)
	}

	if resp.Usage.OutputTokens != 12 {
		t.Errorf("Usage.OutputTokens = %d, want %d", resp.Usage.OutputTokens, 12)
	}

	if resp.Metadata.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Metadata.Model = %q, want %q", resp.Metadata.Model, "claude-sonnet-4-20250514")
	}
}

func TestCLIResponse_UnmarshalJSON_Empty(t *testing.T) {
	// Given: A response with empty result
	input := `{"result": "", "session_id": "abc"}`

	// When: We unmarshal the JSON
	var resp CLIResponse
	err := json.Unmarshal([]byte(input), &resp)

	// Then: Empty result should be preserved
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Result != "" {
		t.Errorf("Result = %q, want empty string", resp.Result)
	}

	if resp.SessionID != "abc" {
		t.Errorf("SessionID = %q, want %q", resp.SessionID, "abc")
	}
}

func TestCLIResponse_UnmarshalJSON_MissingFields(t *testing.T) {
	// Given: A partial JSON response (missing usage and metadata)
	input := `{"result": "Hello"}`

	// When: We unmarshal the JSON
	var resp CLIResponse
	err := json.Unmarshal([]byte(input), &resp)

	// Then: Should succeed with zero values for missing fields
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Result != "Hello" {
		t.Errorf("Result = %q, want %q", resp.Result, "Hello")
	}

	if resp.SessionID != "" {
		t.Errorf("SessionID = %q, want empty string", resp.SessionID)
	}

	if resp.Usage.InputTokens != 0 {
		t.Errorf("Usage.InputTokens = %d, want 0", resp.Usage.InputTokens)
	}

	if resp.Metadata.Model != "" {
		t.Errorf("Metadata.Model = %q, want empty string", resp.Metadata.Model)
	}
}

func TestUsage_MarshalJSON(t *testing.T) {
	// Given: A Usage struct
	usage := Usage{
		InputTokens:  100,
		OutputTokens: 200,
	}

	// When: We marshal and unmarshal
	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var result Usage
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Then: Values should round-trip correctly
	if result.InputTokens != usage.InputTokens {
		t.Errorf("InputTokens = %d, want %d", result.InputTokens, usage.InputTokens)
	}

	if result.OutputTokens != usage.OutputTokens {
		t.Errorf("OutputTokens = %d, want %d", result.OutputTokens, usage.OutputTokens)
	}
}

func TestCLIResponse_MarshalJSON_RoundTrip(t *testing.T) {
	// Given: A complete CLIResponse
	original := CLIResponse{
		Result:    "Test response",
		SessionID: "session-123",
		Usage: Usage{
			InputTokens:  50,
			OutputTokens: 100,
		},
		Metadata: Metadata{
			Model: "claude-opus-4-20250514",
		},
	}

	// When: We marshal and unmarshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var result CLIResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Then: All values should round-trip correctly
	if result.Result != original.Result {
		t.Errorf("Result = %q, want %q", result.Result, original.Result)
	}

	if result.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", result.SessionID, original.SessionID)
	}

	if result.Usage.InputTokens != original.Usage.InputTokens {
		t.Errorf("Usage.InputTokens = %d, want %d", result.Usage.InputTokens, original.Usage.InputTokens)
	}

	if result.Usage.OutputTokens != original.Usage.OutputTokens {
		t.Errorf("Usage.OutputTokens = %d, want %d", result.Usage.OutputTokens, original.Usage.OutputTokens)
	}

	if result.Metadata.Model != original.Metadata.Model {
		t.Errorf("Metadata.Model = %q, want %q", result.Metadata.Model, original.Metadata.Model)
	}
}
