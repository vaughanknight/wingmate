package llm

import (
	"sync"
	"testing"
)

// Package llm provides session management for CLI conversation continuity.
//
// Why: Session manager tests verify thread-safe storage and retrieval of CLI session IDs.
// Contract: SessionManager maintains conversation-to-session mappings correctly under concurrent access.
// Usage Notes: Run with `go test -v ./internal/llm/... -run Session`.
// Quality Contribution: Ensures session continuity works correctly in multi-threaded agent scenarios.
// Worked Example: TestSessionManager_GetSet demonstrates basic CRUD operations.

func TestSessionManager_New(t *testing.T) {
	sm := NewSessionManager()
	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}
	if sm.sessions == nil {
		t.Error("sessions map is nil")
	}
}

func TestSessionManager_GetSet(t *testing.T) {
	sm := NewSessionManager()

	// Get non-existent returns empty
	got := sm.Get("conv-1")
	if got != "" {
		t.Errorf("Get non-existent = %q, want empty", got)
	}

	// Set and get
	sm.Set("conv-1", "session-abc")
	got = sm.Get("conv-1")
	if got != "session-abc" {
		t.Errorf("Get after Set = %q, want session-abc", got)
	}

	// Update existing
	sm.Set("conv-1", "session-xyz")
	got = sm.Get("conv-1")
	if got != "session-xyz" {
		t.Errorf("Get after update = %q, want session-xyz", got)
	}
}

func TestSessionManager_Clear(t *testing.T) {
	sm := NewSessionManager()
	sm.Set("conv-1", "session-abc")
	sm.Set("conv-2", "session-def")

	// Clear one
	sm.Clear("conv-1")
	if got := sm.Get("conv-1"); got != "" {
		t.Errorf("Get after Clear = %q, want empty", got)
	}

	// Other still exists
	if got := sm.Get("conv-2"); got != "session-def" {
		t.Errorf("Other session = %q, want session-def", got)
	}
}

func TestSessionManager_ClearAll(t *testing.T) {
	sm := NewSessionManager()
	sm.Set("conv-1", "session-abc")
	sm.Set("conv-2", "session-def")

	sm.ClearAll()

	if got := sm.Get("conv-1"); got != "" {
		t.Errorf("Get conv-1 after ClearAll = %q, want empty", got)
	}
	if got := sm.Get("conv-2"); got != "" {
		t.Errorf("Get conv-2 after ClearAll = %q, want empty", got)
	}
}

func TestSessionManager_ThreadSafe(t *testing.T) {
	sm := NewSessionManager()
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Concurrent writers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sm.Set("conv", "session")
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = sm.Get("conv")
			}
		}(i)
	}

	// Concurrent clearers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sm.Clear("conv")
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race condition panic
}

func TestSessionManager_MultipleConversations(t *testing.T) {
	sm := NewSessionManager()

	sm.Set("conv-1", "session-1")
	sm.Set("conv-2", "session-2")
	sm.Set("conv-3", "session-3")

	tests := []struct {
		convID   string
		wantSess string
	}{
		{"conv-1", "session-1"},
		{"conv-2", "session-2"},
		{"conv-3", "session-3"},
		{"conv-4", ""}, // non-existent
	}

	for _, tt := range tests {
		got := sm.Get(tt.convID)
		if got != tt.wantSess {
			t.Errorf("Get(%q) = %q, want %q", tt.convID, got, tt.wantSess)
		}
	}
}
