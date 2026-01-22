package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// TestTransportReadNDJSON verifies that Transport.Read correctly parses
// newline-delimited JSON messages from the input stream.
func TestTransportReadNDJSON(t *testing.T) {
	// Arrange: Create a transport with test input
	input := `{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n"
	transport := NewTransport(strings.NewReader(input), io.Discard)

	// Act: Read a message
	var msg map[string]any
	err := transport.Read(&msg)

	// Assert: Message should be parsed correctly
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if msg["method"] != "test" {
		t.Errorf("got method %v, want 'test'", msg["method"])
	}
	if msg["id"].(float64) != 1 {
		t.Errorf("got id %v, want 1", msg["id"])
	}
}

// TestTransportWriteNDJSON verifies that Transport.Write correctly outputs
// JSON followed by a newline.
func TestTransportWriteNDJSON(t *testing.T) {
	// Arrange: Create a transport with test output buffer
	var output bytes.Buffer
	transport := NewTransport(strings.NewReader(""), &output)

	// Act: Write a message
	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  "ok",
	}
	err := transport.Write(msg)

	// Assert: Output should be valid NDJSON
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	written := output.String()
	if !strings.HasSuffix(written, "\n") {
		t.Error("output should end with newline")
	}
	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(written)), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
	if parsed["result"] != "ok" {
		t.Errorf("got result %v, want 'ok'", parsed["result"])
	}
}

// TestTransportHandlesMultipleMessages verifies that Transport can read
// multiple messages in sequence from the same stream.
func TestTransportHandlesMultipleMessages(t *testing.T) {
	// Arrange: Create input with 3 messages
	input := `{"id":1}` + "\n" + `{"id":2}` + "\n" + `{"id":3}` + "\n"
	transport := NewTransport(strings.NewReader(input), io.Discard)

	// Act & Assert: Read all 3 messages
	for i := 1; i <= 3; i++ {
		var msg map[string]any
		err := transport.Read(&msg)
		if err != nil {
			t.Fatalf("Read %d failed: %v", i, err)
		}
		gotID := int(msg["id"].(float64))
		if gotID != i {
			t.Errorf("message %d: got id %d, want %d", i, gotID, i)
		}
	}

	// Verify EOF after all messages
	var msg map[string]any
	err := transport.Read(&msg)
	if err != io.EOF {
		t.Errorf("expected EOF after all messages, got %v", err)
	}
}

// TestTransportRejectsInvalidJSON verifies that Transport.Read returns
// an error for malformed JSON input.
func TestTransportRejectsInvalidJSON(t *testing.T) {
	// Arrange: Create transport with invalid JSON
	input := `{not valid json}` + "\n"
	transport := NewTransport(strings.NewReader(input), io.Discard)

	// Act: Try to read
	var msg map[string]any
	err := transport.Read(&msg)

	// Assert: Should return an error
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// TestTransportReturnsEOFOnEmptyInput verifies that Transport.Read returns
// io.EOF when there's no more input to read.
func TestTransportReturnsEOFOnEmptyInput(t *testing.T) {
	// Arrange: Create transport with empty input
	transport := NewTransport(strings.NewReader(""), io.Discard)

	// Act: Try to read
	var msg map[string]any
	err := transport.Read(&msg)

	// Assert: Should return EOF
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}
