package flightlog

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestFlightLog_Verbose verifies verbose mode writes to stdout.
//
// Test Doc:
// - Why: Engineers need to watch agent communication in real-time
// - Contract: Verbose mode writes entry to both file and stdout
// - Usage Notes: Use WithStdout option to capture output in tests
// - Quality Contribution: Catches missing stdout mirroring
// - Worked Example: Record(entry) with verbose → entry in file AND buffer
func TestFlightLog_Verbose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	var buf bytes.Buffer
	fl, err := New(path, WithVerbose(true), WithStdout(&buf))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer fl.Close()

	ctx := WithTraceID(context.Background(), "test-trace")
	entry := NewEntry(ctx, "test-agent", Outbound, "Test message", map[string]string{"key": "value"})

	if err := fl.Record(entry); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Verify stdout output
	output := buf.String()
	if !strings.Contains(output, "test-agent") {
		t.Error("Stdout does not contain agent name")
	}
	if !strings.Contains(output, "Test message") {
		t.Error("Stdout does not contain summary")
	}
}

// TestFlightLog_FileOnly verifies non-verbose writes file only.
//
// Test Doc:
// - Why: Default mode should not spam stdout
// - Contract: Without verbose, stdout remains empty
// - Usage Notes: File still receives all entries
// - Quality Contribution: Catches unwanted stdout writes
func TestFlightLog_FileOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	var buf bytes.Buffer
	fl, err := New(path, WithVerbose(false), WithStdout(&buf))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer fl.Close()

	ctx := context.Background()
	entry := NewEntry(ctx, "test-agent", Outbound, "Test message", nil)

	if err := fl.Record(entry); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Verify stdout is empty
	if buf.Len() > 0 {
		t.Errorf("Expected empty stdout, got: %s", buf.String())
	}

	// Verify file has content using test helper
	entries := ReadLogEntries(t, path)
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry in file, got %d", len(entries))
	}
}

// TestFlightLog_NewEntry verifies constructor auto-fills fields.
//
// Test Doc:
// - Why: Callers shouldn't need to remember to set timestamp and trace
// - Contract: NewEntry auto-fills Timestamp (UTC) and TraceID from context
// - Usage Notes: Per Critical Insight #3 - handles time.Time gotcha
// - Quality Contribution: Catches missing auto-fill bugs
func TestFlightLog_NewEntry(t *testing.T) {
	traceID := "my-trace-id"
	ctx := WithTraceID(context.Background(), traceID)

	entry := NewEntry(ctx, "test-agent", Inbound, "Test summary", nil)

	// Timestamp should be auto-filled (not zero)
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp was not auto-filled")
	}

	// TraceID should come from context
	if entry.TraceID != traceID {
		t.Errorf("TraceID = %q, want %q", entry.TraceID, traceID)
	}

	// Other fields should be set
	if entry.Agent != "test-agent" {
		t.Errorf("Agent = %q, want %q", entry.Agent, "test-agent")
	}
	if entry.Direction != Inbound {
		t.Errorf("Direction = %v, want %v", entry.Direction, Inbound)
	}
	if entry.Summary != "Test summary" {
		t.Errorf("Summary = %q, want %q", entry.Summary, "Test summary")
	}
}

// TestFlightLog_NewEntryNoTrace verifies behavior without trace in context.
//
// Test Doc:
// - Why: Must handle case where no trace ID is in context
// - Contract: Missing trace ID results in empty TraceID field
// - Usage Notes: Consider generating trace ID if empty
func TestFlightLog_NewEntryNoTrace(t *testing.T) {
	ctx := context.Background() // No trace ID

	entry := NewEntry(ctx, "test-agent", Outbound, "No trace", nil)

	// Should not panic, TraceID should be empty
	if entry.TraceID != "" {
		t.Errorf("TraceID = %q, expected empty", entry.TraceID)
	}
}

// TestFlightLog_DirectionString verifies Direction string representation.
//
// Test Doc:
// - Why: Direction appears in JSON output
// - Contract: Direction constants serialize to expected strings
func TestFlightLog_DirectionString(t *testing.T) {
	tests := []struct {
		dir  Direction
		want string
	}{
		{Outbound, "out"},
		{Inbound, "in"},
		{ErrorDir, "error"},
	}

	for _, tt := range tests {
		if string(tt.dir) != tt.want {
			t.Errorf("Direction %v = %q, want %q", tt.dir, string(tt.dir), tt.want)
		}
	}
}
