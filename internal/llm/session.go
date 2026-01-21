package llm

import "sync"

// SessionManager manages CLI session IDs for conversation continuity.
// It maps conversation IDs to CLI session IDs in a thread-safe manner.
type SessionManager struct {
	sessions map[string]string
	mu       sync.RWMutex
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]string),
	}
}

// Get retrieves the session ID for a conversation.
// Returns empty string if no session exists.
func (sm *SessionManager) Get(conversationID string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[conversationID]
}

// Set stores the session ID for a conversation.
func (sm *SessionManager) Set(conversationID, sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[conversationID] = sessionID
}

// Clear removes the session for a conversation.
func (sm *SessionManager) Clear(conversationID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, conversationID)
}

// ClearAll removes all sessions.
func (sm *SessionManager) ClearAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions = make(map[string]string)
}
