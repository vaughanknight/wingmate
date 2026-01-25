package mcp

import (
	"sync"
	"testing"
	"time"
)

/*
Test Doc:
- Why: MCP HTTP transport needs session continuity across requests (spec Q7, Discovery 05)
- Contract: MCPSessionManager provides thread-safe CRUD with 30-min idle TTL and background cleanup
- Usage Notes: Create via NewMCPSessionManager; call Stop() on shutdown to halt cleanup goroutine
- Quality Contribution: Ensures session state persists across HTTP requests; prevents memory leaks
- Worked Example: CreateSession() → returns ID; GetSession(id) → returns data; after 30min idle → session expires
*/

func TestMCPSessionManager_CreateSession(t *testing.T) {
	/*
	Test Doc:
	- Why: Sessions must be created with unique IDs for Claude Code conversation tracking
	- Contract: CreateSession returns unique ID; session is immediately retrievable
	- Usage Notes: IDs should be UUIDs or similar unique format
	- Quality Contribution: Foundation for all session operations
	- Worked Example: CreateSession() → "sess-abc123"; GetSession("sess-abc123") → (data, true)
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	defer mgr.Stop()

	id := mgr.CreateSession()
	if id == "" {
		t.Fatal("CreateSession returned empty ID")
	}

	// Session should be immediately retrievable
	_, exists := mgr.GetSession(id)
	if !exists {
		t.Errorf("newly created session %q not found", id)
	}
}

func TestMCPSessionManager_CreateSession_UniqueIDs(t *testing.T) {
	/*
	Test Doc:
	- Why: Each Claude Code connection needs its own session to avoid state collision
	- Contract: Repeated CreateSession calls return different IDs
	- Usage Notes: IDs must be unique for concurrent multi-client scenarios
	- Quality Contribution: Prevents session ID collision in multi-client scenarios
	- Worked Example: CreateSession() → "a"; CreateSession() → "b"; a != b
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	defer mgr.Stop()

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := mgr.CreateSession()
		if seen[id] {
			t.Fatalf("duplicate session ID generated: %q on iteration %d", id, i)
		}
		seen[id] = true
	}
}

func TestMCPSessionManager_GetSession_NotFound(t *testing.T) {
	/*
	Test Doc:
	- Why: Client may send invalid or expired session ID
	- Contract: GetSession returns (nil, false) for non-existent session
	- Usage Notes: Callers should check exists bool before using data
	- Quality Contribution: Clear signal for session miss vs error
	- Worked Example: GetSession("nonexistent") → (nil, false)
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	defer mgr.Stop()

	data, exists := mgr.GetSession("nonexistent-id")
	if exists {
		t.Errorf("expected false for non-existent session, got true with data: %v", data)
	}
	if data != nil {
		t.Errorf("expected nil data for non-existent session, got: %v", data)
	}
}

func TestMCPSessionManager_SetSession(t *testing.T) {
	/*
	Test Doc:
	- Why: Chat handler stores conversation context in session
	- Contract: SetSession stores data; GetSession retrieves same data
	- Usage Notes: Data is a map[string]any for flexibility
	- Quality Contribution: Enables conversation context persistence
	- Worked Example: SetSession("id", {"history": [...]}) → GetSession("id") returns {"history": [...]}
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	defer mgr.Stop()

	id := mgr.CreateSession()
	testData := map[string]any{
		"conversation": []string{"hello", "world"},
		"turn":         2,
	}

	mgr.SetSession(id, testData)

	retrieved, exists := mgr.GetSession(id)
	if !exists {
		t.Fatal("session not found after SetSession")
	}

	// Check data integrity
	if conv, ok := retrieved["conversation"].([]string); !ok || len(conv) != 2 {
		t.Errorf("conversation data not preserved: %v", retrieved)
	}
	if turn, ok := retrieved["turn"].(int); !ok || turn != 2 {
		t.Errorf("turn data not preserved: %v", retrieved)
	}
}

func TestMCPSessionManager_SetSession_UpdatesExisting(t *testing.T) {
	/*
	Test Doc:
	- Why: Each chat request updates session with new conversation state
	- Contract: SetSession replaces existing data; original data is gone
	- Usage Notes: Always merge data client-side before calling SetSession
	- Quality Contribution: Simple replace semantics, no merge complexity
	- Worked Example: SetSession("id", {"v": 1}); SetSession("id", {"v": 2}) → GetSession("id") returns {"v": 2}
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	defer mgr.Stop()

	id := mgr.CreateSession()

	mgr.SetSession(id, map[string]any{"version": 1})
	mgr.SetSession(id, map[string]any{"version": 2, "new_field": "added"})

	retrieved, _ := mgr.GetSession(id)
	if version, ok := retrieved["version"].(int); !ok || version != 2 {
		t.Errorf("expected version 2, got: %v", retrieved)
	}
	if _, ok := retrieved["new_field"]; !ok {
		t.Error("new_field should be present after update")
	}
}

func TestMCPSessionManager_TTLExpiry(t *testing.T) {
	/*
	Test Doc:
	- Why: Spec Q8 defines 30-minute idle TTL to prevent unbounded memory growth
	- Contract: Sessions expire after idle TTL; expired sessions return (nil, false)
	- Usage Notes: TTL resets on GetSession and SetSession calls
	- Quality Contribution: Prevents memory leaks from abandoned sessions
	- Worked Example: Create session → wait > TTL → GetSession returns (nil, false)
	*/

	// Use very short TTL for testing
	mgr := NewMCPSessionManager(50*time.Millisecond, 10*time.Millisecond)
	defer mgr.Stop()

	id := mgr.CreateSession()

	// Verify session exists
	_, exists := mgr.GetSession(id)
	if !exists {
		t.Fatal("session should exist immediately after creation")
	}

	// Wait for TTL + cleanup interval
	time.Sleep(100 * time.Millisecond)

	// Session should be expired
	_, exists = mgr.GetSession(id)
	if exists {
		t.Error("session should have expired after TTL")
	}
}

func TestMCPSessionManager_TTL_RefreshOnAccess(t *testing.T) {
	/*
	Test Doc:
	- Why: Active sessions should not expire; only idle sessions should
	- Contract: GetSession resets TTL; session survives if accessed within TTL
	- Usage Notes: Each MCP request touching session keeps it alive
	- Quality Contribution: Active Claude Code sessions persist; idle ones expire
	- Worked Example: Create → access at T+20ms → access at T+40ms → still alive at T+50ms (TTL=50ms)
	*/

	mgr := NewMCPSessionManager(50*time.Millisecond, 10*time.Millisecond)
	defer mgr.Stop()

	id := mgr.CreateSession()

	// Access session repeatedly within TTL
	for i := 0; i < 5; i++ {
		time.Sleep(20 * time.Millisecond)
		_, exists := mgr.GetSession(id)
		if !exists {
			t.Fatalf("session should still exist on access %d", i)
		}
	}

	// Total elapsed: ~100ms, but session should be alive because we kept accessing it
	_, exists := mgr.GetSession(id)
	if !exists {
		t.Error("session should survive due to access refreshing TTL")
	}
}

func TestMCPSessionManager_ConcurrentAccess(t *testing.T) {
	/*
	Test Doc:
	- Why: Multiple Claude Code instances may use same Wingmate agent (Critical Discovery 03)
	- Contract: Concurrent Create/Get/Set operations are thread-safe; no data races
	- Usage Notes: Run with -race flag to detect issues
	- Quality Contribution: Ensures AC-9 concurrent request handling
	- Worked Example: 20 goroutines doing CRUD simultaneously → no races, no panics
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	defer mgr.Stop()

	var wg sync.WaitGroup
	const numGoroutines = 20
	const opsPerGoroutine = 50

	// Mix of operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			var mySessionID string

			for j := 0; j < opsPerGoroutine; j++ {
				switch j % 3 {
				case 0: // Create
					mySessionID = mgr.CreateSession()
				case 1: // Set
					if mySessionID != "" {
						mgr.SetSession(mySessionID, map[string]any{
							"worker": workerID,
							"op":     j,
						})
					}
				case 2: // Get
					if mySessionID != "" {
						mgr.GetSession(mySessionID)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	// If we get here without panics or race detector complaints, test passes
}

func TestMCPSessionManager_ConcurrentAccess_SameSession(t *testing.T) {
	/*
	Test Doc:
	- Why: Same session may be accessed by multiple requests in flight
	- Contract: Concurrent access to same session ID is safe; last write wins
	- Usage Notes: No transactions; use for non-critical metadata
	- Quality Contribution: Prevents panics and corruption under load
	- Worked Example: 10 goroutines reading/writing same session → no races
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	defer mgr.Stop()

	// Create a shared session
	sharedID := mgr.CreateSession()

	var wg sync.WaitGroup
	const numGoroutines = 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				// Alternating read/write
				if j%2 == 0 {
					mgr.SetSession(sharedID, map[string]any{
						"writer": workerID,
						"seq":    j,
					})
				} else {
					mgr.GetSession(sharedID)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestMCPSessionManager_DeleteSession(t *testing.T) {
	/*
	Test Doc:
	- Why: Explicit cleanup when Claude Code disconnects
	- Contract: DeleteSession removes session; subsequent GetSession returns false
	- Usage Notes: Optional; TTL cleanup handles abandoned sessions
	- Quality Contribution: Immediate resource reclamation on disconnect
	- Worked Example: DeleteSession("id") → GetSession("id") returns (nil, false)
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	defer mgr.Stop()

	id := mgr.CreateSession()
	mgr.SetSession(id, map[string]any{"data": "test"})

	// Verify exists
	_, exists := mgr.GetSession(id)
	if !exists {
		t.Fatal("session should exist before delete")
	}

	mgr.DeleteSession(id)

	// Should be gone
	_, exists = mgr.GetSession(id)
	if exists {
		t.Error("session should not exist after delete")
	}
}

func TestMCPSessionManager_ActiveCount(t *testing.T) {
	/*
	Test Doc:
	- Why: Status handler needs to report sessions_active count
	- Contract: ActiveCount returns number of non-expired sessions
	- Usage Notes: Count may decrease between calls due to TTL expiry
	- Quality Contribution: Enables monitoring and status reporting
	- Worked Example: Create 3 sessions → ActiveCount() returns 3
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	defer mgr.Stop()

	if mgr.ActiveCount() != 0 {
		t.Errorf("expected 0 active sessions initially, got %d", mgr.ActiveCount())
	}

	mgr.CreateSession()
	mgr.CreateSession()
	mgr.CreateSession()

	if mgr.ActiveCount() != 3 {
		t.Errorf("expected 3 active sessions, got %d", mgr.ActiveCount())
	}
}

func TestMCPSessionManager_Stop(t *testing.T) {
	/*
	Test Doc:
	- Why: Clean shutdown of background cleanup goroutine
	- Contract: Stop() halts cleanup goroutine; safe to call multiple times
	- Usage Notes: Always call Stop() on server shutdown
	- Quality Contribution: Prevents goroutine leaks; clean shutdown
	- Worked Example: mgr.Stop() → cleanup goroutine exits
	*/

	mgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)

	// Should not panic on multiple stops
	mgr.Stop()
	mgr.Stop()
}

func TestMCPSessionManager_ImplementsInterface(t *testing.T) {
	/*
	Test Doc:
	- Why: MCPSessionManager must implement existing SessionManager interface
	- Contract: Type assertion to SessionManager succeeds
	- Usage Notes: Allows use with existing handlers expecting SessionManager
	- Quality Contribution: Backward compatibility with handlers.go interface
	- Worked Example: var _ SessionManager = (*MCPSessionManager)(nil) compiles
	*/

	var _ SessionManager = (*MCPSessionManager)(nil)
}
