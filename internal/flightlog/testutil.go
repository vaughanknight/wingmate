package flightlog

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

// ReadLogEntries reads all entries from a Flight Log file.
// Per Critical Insight #2, this helper enables agent tests to verify log contents
// using real file I/O instead of mocks.
//
// Usage:
//
//	dir := t.TempDir()
//	path := filepath.Join(dir, "test.jsonl")
//	// ... write entries ...
//	entries := flightlog.ReadLogEntries(t, path)
//	if len(entries) != expected {
//	    t.Errorf("got %d entries, want %d", len(entries), expected)
//	}
func ReadLogEntries(t *testing.T, path string) []Entry {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Failed to open log file %s: %v", path, err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("Failed to parse line %d: %v\nLine: %s", lineNum, err, line)
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading log file: %v", err)
	}

	return entries
}

// AssertEntryExists checks that at least one entry matches the given criteria.
// This is a convenience helper for common test assertions.
func AssertEntryExists(t *testing.T, entries []Entry, agent string, dir Direction, summaryContains string) {
	t.Helper()

	for _, e := range entries {
		if e.Agent == agent && e.Direction == dir {
			if summaryContains == "" || contains(e.Summary, summaryContains) {
				return // Found matching entry
			}
		}
	}

	t.Errorf("No entry found matching agent=%q dir=%v summary contains %q", agent, dir, summaryContains)
}

// contains checks if s contains substr (simple substring check).
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
