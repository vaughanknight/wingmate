package flightlog

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestWriter_Create verifies that Writer creates a new file if it doesn't exist.
//
// Test Doc:
// - Why: Flight Log must create files on first write
// - Contract: Write creates file if missing, succeeds on first entry
// - Usage Notes: Uses t.TempDir() for isolated file I/O
// - Quality Contribution: Catches file creation permission issues
// - Worked Example: Writer.Write(entry) → file exists with entry
func TestWriter_Create(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer w.Close()

	entry := Entry{
		Agent:     "test-agent",
		Direction: Outbound,
		Summary:   "Test entry",
		Payload:   map[string]string{"test": "data"},
	}

	if err := w.Write(entry); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("File was not created")
	}

	// Verify content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "test-agent") {
		t.Error("File does not contain expected agent name")
	}
}

// TestWriter_Append verifies that Writer appends entries without overwriting.
//
// Test Doc:
// - Why: Flight Log must preserve all entries (append-only)
// - Contract: Multiple writes result in multiple lines
// - Usage Notes: Each entry is one JSON line
// - Quality Contribution: Catches overwrite bugs
// - Worked Example: Write(A), Write(B) → file has 2 lines
func TestWriter_Append(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer w.Close()

	// Write multiple entries
	for i := 0; i < 3; i++ {
		entry := Entry{
			Agent:     "test-agent",
			Direction: Outbound,
			Summary:   "Entry",
			Payload:   map[string]int{"index": i},
		}
		if err := w.Write(entry); err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	// Verify line count
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

// TestWriter_Concurrent verifies thread safety of Writer.
//
// Test Doc:
// - Why: Multiple goroutines may write simultaneously (agent handlers)
// - Contract: Concurrent writes don't corrupt data or cause race conditions
// - Usage Notes: Run with -race flag to detect races
// - Quality Contribution: Catches race conditions, data corruption
// - Worked Example: 10 goroutines × 100 writes → 1000 valid lines
func TestWriter_Concurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}
	defer w.Close()

	const goroutines = 10
	const writesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < writesPerGoroutine; i++ {
				entry := Entry{
					Agent:     "test-agent",
					Direction: Outbound,
					Summary:   "Concurrent write",
					Payload:   map[string]int{"goroutine": gid, "index": i},
				}
				if err := w.Write(entry); err != nil {
					t.Errorf("Write failed in goroutine %d: %v", gid, err)
				}
			}
		}(g)
	}

	wg.Wait()

	// Verify total line count
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	expectedLines := goroutines * writesPerGoroutine
	if len(lines) != expectedLines {
		t.Errorf("Expected %d lines, got %d", expectedLines, len(lines))
	}
}
