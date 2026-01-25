package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// MCPSessionManager manages conversation sessions for HTTP-based MCP transport.
// It provides thread-safe CRUD operations with automatic TTL-based expiration
// and background cleanup of expired sessions.
//
// Sessions enable chat continuity across HTTP requests: each request can include
// a session ID (via Mcp-Session-Id header) to maintain conversation context.
type MCPSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*session
	ttl      time.Duration
	stopCh   chan struct{}
	stopped  bool
}

// session holds the data and metadata for a single MCP session.
type session struct {
	data       map[string]any
	lastAccess time.Time
}

// NewMCPSessionManager creates a new session manager with the specified TTL
// and cleanup interval.
//
// Parameters:
//   - ttl: How long a session survives without being accessed
//   - cleanupInterval: How often the cleanup goroutine runs
//
// The cleanup goroutine runs in the background and removes expired sessions.
// Call Stop() to halt the cleanup goroutine on server shutdown.
func NewMCPSessionManager(ttl, cleanupInterval time.Duration) *MCPSessionManager {
	mgr := &MCPSessionManager{
		sessions: make(map[string]*session),
		ttl:      ttl,
		stopCh:   make(chan struct{}),
	}

	// Start background cleanup goroutine
	go mgr.cleanupLoop(cleanupInterval)

	return mgr
}

// cleanupLoop runs periodically to remove expired sessions.
// It uses copy-on-write pattern: collect expired IDs, then delete them.
func (m *MCPSessionManager) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpired()
		case <-m.stopCh:
			return
		}
	}
}

// cleanupExpired removes sessions that have exceeded their TTL.
func (m *MCPSessionManager) cleanupExpired() {
	now := time.Now()
	var expired []string

	// First pass: collect expired session IDs (read lock)
	m.mu.RLock()
	for id, sess := range m.sessions {
		if now.Sub(sess.lastAccess) > m.ttl {
			expired = append(expired, id)
		}
	}
	m.mu.RUnlock()

	// Second pass: delete expired sessions (write lock)
	if len(expired) > 0 {
		m.mu.Lock()
		for _, id := range expired {
			// Re-check in case session was accessed between passes
			if sess, exists := m.sessions[id]; exists {
				if now.Sub(sess.lastAccess) > m.ttl {
					delete(m.sessions, id)
				}
			}
		}
		m.mu.Unlock()
	}
}

// CreateSession creates a new session with a unique ID.
// The session is immediately retrievable via GetSession.
func (m *MCPSessionManager) CreateSession() string {
	id := generateSessionID()

	m.mu.Lock()
	m.sessions[id] = &session{
		data:       make(map[string]any),
		lastAccess: time.Now(),
	}
	m.mu.Unlock()

	return id
}

// GetSession retrieves session data by ID.
// Returns (data, true) if found, (nil, false) if not found or expired.
// Accessing a session refreshes its TTL.
func (m *MCPSessionManager) GetSession(id string) (map[string]any, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, exists := m.sessions[id]
	if !exists {
		return nil, false
	}

	// Refresh TTL on access
	sess.lastAccess = time.Now()
	return sess.data, true
}

// SetSession stores data for a session.
// If the session exists, it replaces the existing data and refreshes TTL.
// If the session doesn't exist, this is a no-op (create session first).
func (m *MCPSessionManager) SetSession(id string, data map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, exists := m.sessions[id]
	if !exists {
		return
	}

	sess.data = data
	sess.lastAccess = time.Now()
}

// DeleteSession explicitly removes a session.
// This is optional; TTL cleanup handles abandoned sessions automatically.
func (m *MCPSessionManager) DeleteSession(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

// ActiveCount returns the number of non-expired sessions.
// Used by status handler to report sessions_active.
func (m *MCPSessionManager) ActiveCount() int {
	m.mu.RLock()
	count := len(m.sessions)
	m.mu.RUnlock()
	return count
}

// Stop halts the background cleanup goroutine.
// Safe to call multiple times.
func (m *MCPSessionManager) Stop() {
	m.mu.Lock()
	if !m.stopped {
		m.stopped = true
		close(m.stopCh)
	}
	m.mu.Unlock()
}

// generateSessionID creates a random session ID.
// Format: "mcp-" prefix + 16 random hex characters
func generateSessionID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return "mcp-" + hex.EncodeToString([]byte(time.Now().String()[:16]))
	}
	return "mcp-" + hex.EncodeToString(b)
}
