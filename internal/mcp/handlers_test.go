// Package mcp provides tests for the MCP tool handlers.
//
// Why: Ensures tool handlers correctly invoke LLMExecutor and return proper results.
// Contract: Handlers must invoke the executor, handle errors, and log to Flight Log.
// Usage Notes: Use MockLLMExecutor for unit tests to avoid CLI dependency.
// Quality Contribution: Catches handler logic errors without requiring Claude CLI.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/wingmate/wingmate/internal/llm"
)

// =============================================================================
// Mock LLMExecutor
// =============================================================================

// MockLLMExecutor implements llm.LLMExecutor for testing.
type MockLLMExecutor struct {
	// Response to return from Execute
	Response *llm.CLIResponse
	// Error to return from Execute
	Err error
	// Whether CLI is installed
	Installed bool
	// Capture the prompt and session for verification
	LastPrompt    string
	LastSessionID string
	// Call count for verification
	CallCount int
}

// Execute implements llm.LLMExecutor.
func (m *MockLLMExecutor) Execute(ctx context.Context, prompt string, sessionID string) (*llm.CLIResponse, error) {
	m.CallCount++
	m.LastPrompt = prompt
	m.LastSessionID = sessionID

	if m.Err != nil {
		return nil, m.Err
	}
	return m.Response, nil
}

// IsInstalled implements llm.LLMExecutor.
func (m *MockLLMExecutor) IsInstalled() bool {
	return m.Installed
}

// =============================================================================
// Mock Logger
// =============================================================================

// MockLogger captures log entries for testing.
type MockLogger struct {
	Entries []map[string]any
}

// Record implements Logger.
func (m *MockLogger) Record(entry any) error {
	if e, ok := entry.(map[string]any); ok {
		m.Entries = append(m.Entries, e)
	}
	return nil
}

// Close implements Logger.
func (m *MockLogger) Close() error {
	return nil
}

// =============================================================================
// wingmate_chat Handler Tests (T006)
// =============================================================================

// TestWingmateChatSuccess verifies the happy path for wingmate_chat.
//
// Given: A mock executor that returns a valid response
// When: HandleChat is called with a valid prompt
// Then: Returns ToolResult with the response text
func TestWingmateChatSuccess(t *testing.T) {
	// Setup mock
	executor := &MockLLMExecutor{
		Installed: true,
		Response: &llm.CLIResponse{
			Result:    "Hello from Claude!",
			SessionID: "session-123",
			Usage: llm.Usage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		},
	}

	// Create handler
	handler := NewChatHandler(executor, nil, nil)

	// Call handler
	req := &ToolRequest{
		Name:      ToolNameChat,
		Arguments: map[string]any{"prompt": "Hello!"},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)

	// Assert no error
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Assert result
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.IsError {
		t.Error("Expected IsError=false")
	}
	if len(result.Content) == 0 {
		t.Fatal("Expected content, got empty")
	}

	// Assert content text
	textContent, ok := result.Content[0].(*TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", result.Content[0])
	}
	if textContent.Text != "Hello from Claude!" {
		t.Errorf("Expected 'Hello from Claude!', got '%s'", textContent.Text)
	}

	// Verify executor was called correctly
	if executor.CallCount != 1 {
		t.Errorf("Expected 1 call, got %d", executor.CallCount)
	}
	if executor.LastPrompt != "Hello!" {
		t.Errorf("Expected prompt 'Hello!', got '%s'", executor.LastPrompt)
	}
}

// TestWingmateChatMissingPrompt verifies error when prompt is missing.
//
// Given: A request without a prompt argument
// When: HandleChat is called
// Then: Returns error result with appropriate message
func TestWingmateChatMissingPrompt(t *testing.T) {
	executor := &MockLLMExecutor{Installed: true}
	handler := NewChatHandler(executor, nil, nil)

	req := &ToolRequest{
		Name:      ToolNameChat,
		Arguments: map[string]any{}, // No prompt
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)

	// Should return a result with IsError=true, not a Go error
	if err != nil {
		t.Fatalf("Expected no Go error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.IsError {
		t.Error("Expected IsError=true for missing prompt")
	}
}

// TestWingmateChatCLINotAvailable verifies error when CLI is not installed.
//
// Given: An executor that reports IsInstalled=false
// When: HandleChat is called
// Then: Returns error result with CLI unavailable message
func TestWingmateChatCLINotAvailable(t *testing.T) {
	executor := &MockLLMExecutor{Installed: false}
	handler := NewChatHandler(executor, nil, nil)

	req := &ToolRequest{
		Name:      ToolNameChat,
		Arguments: map[string]any{"prompt": "Hello!"},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no Go error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.IsError {
		t.Error("Expected IsError=true for CLI not available")
	}

	// Check error message mentions CLI
	textContent := result.Content[0].(*TextContent)
	if textContent.Text == "" {
		t.Error("Expected error message, got empty")
	}
}

// TestWingmateChatExecutionError verifies error handling when executor fails.
//
// Given: An executor that returns an error
// When: HandleChat is called
// Then: Returns error result with the error message
func TestWingmateChatExecutionError(t *testing.T) {
	executor := &MockLLMExecutor{
		Installed: true,
		Err:       errors.New("execution failed"),
	}
	handler := NewChatHandler(executor, nil, nil)

	req := &ToolRequest{
		Name:      ToolNameChat,
		Arguments: map[string]any{"prompt": "Hello!"},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no Go error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.IsError {
		t.Error("Expected IsError=true for execution error")
	}

	textContent := result.Content[0].(*TextContent)
	if textContent.Text != "execution failed" {
		t.Errorf("Expected 'execution failed', got '%s'", textContent.Text)
	}
}

// TestWingmateChatTimeout verifies error handling on context timeout.
//
// Given: A context that times out
// When: HandleChat is called
// Then: Returns error result with timeout message
func TestWingmateChatTimeout(t *testing.T) {
	executor := &MockLLMExecutor{
		Installed: true,
		Err:       context.DeadlineExceeded,
	}
	handler := NewChatHandler(executor, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	req := &ToolRequest{
		Name:      ToolNameChat,
		Arguments: map[string]any{"prompt": "Hello!"},
		ctx:       ctx,
	}

	result, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no Go error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.IsError {
		t.Error("Expected IsError=true for timeout")
	}
}

// =============================================================================
// Session Continuity Tests (T008)
// =============================================================================

// TestWingmateChatSessionContinuity verifies session_id is passed to executor.
//
// Given: A request with session_id argument
// When: HandleChat is called
// Then: The session_id is passed to the executor
func TestWingmateChatSessionContinuity(t *testing.T) {
	executor := &MockLLMExecutor{
		Installed: true,
		Response: &llm.CLIResponse{
			Result:    "Continued response",
			SessionID: "session-456",
		},
	}
	handler := NewChatHandler(executor, nil, nil)

	req := &ToolRequest{
		Name: ToolNameChat,
		Arguments: map[string]any{
			"prompt":     "Continue the conversation",
			"session_id": "existing-session",
		},
		ctx: context.Background(),
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.IsError {
		t.Error("Expected no error in result")
	}

	// Verify session was passed
	if executor.LastSessionID != "existing-session" {
		t.Errorf("Expected session 'existing-session', got '%s'", executor.LastSessionID)
	}
}

// =============================================================================
// Flight Log Tests (T012)
// =============================================================================

// TestWingmateChatLogsToFlightLog verifies tool invocations are logged.
//
// Given: A handler with a logger configured
// When: HandleChat is called
// Then: An entry is recorded in the Flight Log
func TestWingmateChatLogsToFlightLog(t *testing.T) {
	executor := &MockLLMExecutor{
		Installed: true,
		Response: &llm.CLIResponse{
			Result:    "Logged response",
			SessionID: "session-789",
		},
	}
	logger := &MockLogger{}
	handler := NewChatHandler(executor, nil, logger)

	req := &ToolRequest{
		Name:      ToolNameChat,
		Arguments: map[string]any{"prompt": "Test logging"},
		ctx:       context.Background(),
	}

	_, _ = handler(context.Background(), req)

	// Verify logging occurred
	if len(logger.Entries) == 0 {
		t.Error("Expected log entry, got none")
	}

	// Check log contains tool name
	found := false
	for _, entry := range logger.Entries {
		if entry["tool"] == ToolNameChat {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected log entry with tool name")
	}
}

// =============================================================================
// wingmate_status Handler Tests (T014)
// =============================================================================

// TestWingmateStatusReturnsHealth verifies status returns health info.
//
// Given: A server status handler
// When: HandleStatus is called
// Then: Returns uptime_ms, sessions_active, cli_available
func TestWingmateStatusReturnsHealth(t *testing.T) {
	executor := &MockLLMExecutor{Installed: true}
	handler := NewStatusHandler(executor, time.Now())

	req := &ToolRequest{
		Name:      ToolNameStatus,
		Arguments: map[string]any{},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.IsError {
		t.Error("Expected IsError=false")
	}
	if len(result.Content) == 0 {
		t.Fatal("Expected content, got empty")
	}

	// Content should contain JSON with health info
	textContent := result.Content[0].(*TextContent)
	if textContent.Text == "" {
		t.Error("Expected status JSON, got empty")
	}
}

// =============================================================================
// wingmate_discover Handler Tests (T016)
// =============================================================================

// TestWingmateDiscoverReturnsEmpty verifies discover returns empty peers when no provider.
//
// Given: A discover handler with nil PeerProvider
// When: HandleDiscover is called
// Then: Returns peers=[], agent_id
func TestWingmateDiscoverReturnsEmpty(t *testing.T) {
	handler := NewDiscoverHandler("agent-001", nil) // nil PeerProvider

	req := &ToolRequest{
		Name:      ToolNameDiscover,
		Arguments: map[string]any{},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.IsError {
		t.Error("Expected IsError=false")
	}
	if len(result.Content) == 0 {
		t.Fatal("Expected content, got empty")
	}

	// Content should contain JSON with peers and agent_id
	textContent := result.Content[0].(*TextContent)
	if textContent.Text == "" {
		t.Error("Expected discover JSON, got empty")
	}
}

// mockPeerProvider is a test implementation of PeerProvider.
type mockPeerProvider struct {
	peers    []string
	peerInfo []PeerInfo
}

func (m *mockPeerProvider) GetPeers() []string {
	return m.peers
}

func (m *mockPeerProvider) GetPeerInfo() []PeerInfo {
	if m.peerInfo != nil {
		return m.peerInfo
	}
	return nil
}

// =============================================================================
// PeerInfo & Extended PeerProvider Tests (Plan 005, Phase 2)
// =============================================================================

// TestPeerInfo_Fields verifies PeerInfo struct has all required fields.
func TestPeerInfo_Fields(t *testing.T) {
	info := PeerInfo{
		Name:        "bravo",
		URL:         "http://localhost:9001",
		Description: "Backend dev",
		Skills:      []PeerSkill{{Name: "chat", Description: "Process messages"}},
		Available:   true,
	}

	if info.Name != "bravo" {
		t.Errorf("Name = %q, want %q", info.Name, "bravo")
	}
	if info.URL != "http://localhost:9001" {
		t.Errorf("URL = %q, want %q", info.URL, "http://localhost:9001")
	}
	if info.Description != "Backend dev" {
		t.Errorf("Description = %q, want %q", info.Description, "Backend dev")
	}
	if len(info.Skills) != 1 || info.Skills[0].Name != "chat" {
		t.Errorf("Skills = %v, want [{chat, Process messages}]", info.Skills)
	}
	if !info.Available {
		t.Error("Available = false, want true")
	}
}

// TestPeerInfo_JSON verifies PeerInfo serializes correctly.
func TestPeerInfo_JSON(t *testing.T) {
	info := PeerInfo{
		Name:      "bravo",
		URL:       "http://localhost:9001",
		Available: true,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed PeerInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Name != info.Name {
		t.Errorf("Name = %q, want %q", parsed.Name, info.Name)
	}
	if parsed.URL != info.URL {
		t.Errorf("URL = %q, want %q", parsed.URL, info.URL)
	}
	if parsed.Available != info.Available {
		t.Errorf("Available = %v, want %v", parsed.Available, info.Available)
	}
	// Description should be omitted (empty)
	if parsed.Description != "" {
		t.Errorf("Description = %q, want empty (omitempty)", parsed.Description)
	}
}

// mockRichPeerProvider implements PeerProvider with GetPeerInfo support.
type mockRichPeerProvider struct {
	peers    []string
	peerInfo []PeerInfo
}

func (m *mockRichPeerProvider) GetPeers() []string {
	return m.peers
}

func (m *mockRichPeerProvider) GetPeerInfo() []PeerInfo {
	return m.peerInfo
}

// TestMockRichPeerProvider_GetPeerInfo verifies mock implements extended interface.
func TestMockRichPeerProvider_GetPeerInfo(t *testing.T) {
	provider := &mockRichPeerProvider{
		peerInfo: []PeerInfo{
			{Name: "alpha", URL: "http://localhost:9000", Available: true},
			{Name: "bravo", URL: "http://localhost:9001", Description: "Backend", Available: false},
		},
	}

	info := provider.GetPeerInfo()
	if len(info) != 2 {
		t.Fatalf("GetPeerInfo() len = %d, want 2", len(info))
	}
	if info[0].Name != "alpha" {
		t.Errorf("info[0].Name = %q, want %q", info[0].Name, "alpha")
	}
	if info[1].Description != "Backend" {
		t.Errorf("info[1].Description = %q, want %q", info[1].Description, "Backend")
	}
}

// TestWingmateDiscoverReturnsPeersFromProvider verifies discover returns actual peers.
//
// Given: A discover handler with a PeerProvider returning 2 peers
// When: HandleDiscover is called
// Then: Returns those 2 peers in the response
func TestWingmateDiscoverReturnsPeersFromProvider(t *testing.T) {
	provider := &mockRichPeerProvider{
		peers: []string{"http://peer1:9000", "http://peer2:9001"},
		peerInfo: []PeerInfo{
			{Name: "peer1", URL: "http://peer1:9000", Available: true},
			{Name: "peer2", URL: "http://peer2:9001", Description: "Backend", Available: true},
		},
	}
	handler := NewDiscoverHandler("agent-001", provider)

	req := &ToolRequest{
		Name:      ToolNameDiscover,
		Arguments: map[string]any{},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.IsError {
		t.Error("Expected IsError=false")
	}

	// Parse the JSON response
	textContent := result.Content[0].(*TextContent)
	var info DiscoverInfo
	if err := json.Unmarshal([]byte(textContent.Text), &info); err != nil {
		t.Fatalf("Failed to parse discover response: %v", err)
	}

	if info.AgentID != "agent-001" {
		t.Errorf("AgentID = %q, want %q", info.AgentID, "agent-001")
	}
	if len(info.Peers) != 2 {
		t.Fatalf("Peers count = %d, want 2", len(info.Peers))
	}

	// Check rich peer data
	if info.Peers[0].Name != "peer1" && info.Peers[1].Name != "peer1" {
		t.Errorf("Expected peer1 in response, got: %v", info.Peers)
	}
}

// TestWingmateDiscoverReturnsPeerInfoRich verifies rich peer data in response.
func TestWingmateDiscoverReturnsPeerInfoRich(t *testing.T) {
	provider := &mockRichPeerProvider{
		peerInfo: []PeerInfo{
			{
				Name:        "bravo",
				URL:         "http://localhost:9001",
				Description: "Backend dev",
				Skills:      []PeerSkill{{Name: "chat", Description: "Process messages"}},
				Available:   true,
			},
		},
	}
	handler := NewDiscoverHandler("alpha", provider)

	req := &ToolRequest{
		Name:      ToolNameDiscover,
		Arguments: map[string]any{},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	textContent := result.Content[0].(*TextContent)
	var info DiscoverInfo
	if err := json.Unmarshal([]byte(textContent.Text), &info); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(info.Peers) != 1 {
		t.Fatalf("Peers len = %d, want 1", len(info.Peers))
	}
	peer := info.Peers[0]
	if peer.Name != "bravo" {
		t.Errorf("Name = %q, want %q", peer.Name, "bravo")
	}
	if peer.Description != "Backend dev" {
		t.Errorf("Description = %q, want %q", peer.Description, "Backend dev")
	}
	if !peer.Available {
		t.Error("Available = false, want true")
	}
	if len(peer.Skills) != 1 || peer.Skills[0].Name != "chat" {
		t.Errorf("Skills = %v, want [{chat ...}]", peer.Skills)
	}
}

// TestWingmateDiscoverEmptyPeerInfo verifies empty peers returns empty array.
func TestWingmateDiscoverEmptyPeerInfo(t *testing.T) {
	provider := &mockRichPeerProvider{
		peerInfo: []PeerInfo{},
	}
	handler := NewDiscoverHandler("alpha", provider)

	req := &ToolRequest{
		Name:      ToolNameDiscover,
		Arguments: map[string]any{},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	textContent := result.Content[0].(*TextContent)
	var info DiscoverInfo
	if err := json.Unmarshal([]byte(textContent.Text), &info); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if info.Peers == nil {
		t.Error("Peers is nil, want empty array")
	}
	if len(info.Peers) != 0 {
		t.Errorf("Peers len = %d, want 0", len(info.Peers))
	}
}

// =============================================================================
// PeerDelegator & wingmate_ask Tests (Plan 005, Phase 3)
// =============================================================================

// mockPeerDelegator implements PeerDelegator for testing.
type mockPeerDelegator struct {
	response    string
	err         error
	lastPeer    string
	lastMessage string
	lastSession string
	callCount   int
}

func (m *mockPeerDelegator) DelegateMessage(ctx context.Context, peer string, message string, sessionID string) (string, error) {
	m.callCount++
	m.lastPeer = peer
	m.lastMessage = message
	m.lastSession = sessionID
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

// TestMockPeerDelegator_DelegateByName verifies mock handles name-based delegation.
func TestMockPeerDelegator_DelegateByName(t *testing.T) {
	delegator := &mockPeerDelegator{response: "pong"}
	resp, err := delegator.DelegateMessage(context.Background(), "bravo", "ping", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "pong" {
		t.Errorf("response = %q, want %q", resp, "pong")
	}
	if delegator.lastPeer != "bravo" {
		t.Errorf("lastPeer = %q, want %q", delegator.lastPeer, "bravo")
	}
}

// TestMockPeerDelegator_DelegateByURL verifies mock handles URL-based delegation.
func TestMockPeerDelegator_DelegateByURL(t *testing.T) {
	delegator := &mockPeerDelegator{response: "hello back"}
	resp, err := delegator.DelegateMessage(context.Background(), "http://localhost:9001", "hello", "session-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "hello back" {
		t.Errorf("response = %q, want %q", resp, "hello back")
	}
	if delegator.lastSession != "session-1" {
		t.Errorf("lastSession = %q, want %q", delegator.lastSession, "session-1")
	}
}

// TestMockPeerDelegator_UnknownPeer verifies mock returns error for unknown peer.
func TestMockPeerDelegator_UnknownPeer(t *testing.T) {
	delegator := &mockPeerDelegator{err: NewMCPError(ErrCodePeerNotFound, "peer not found: unknown")}
	_, err := delegator.DelegateMessage(context.Background(), "unknown", "hello", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsMCPError(err, ErrCodePeerNotFound) {
		t.Errorf("expected error code %d, got: %v", ErrCodePeerNotFound, err)
	}
}

// TestWingmateAskSuccess verifies successful delegation via ask handler.
func TestWingmateAskSuccess(t *testing.T) {
	delegator := &mockPeerDelegator{response: "Go is great!"}
	handler := NewAskHandler(delegator, nil)

	req := &ToolRequest{
		Name:      ToolNameAsk,
		Arguments: map[string]any{"peer": "bravo", "message": "What is Go?"},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected IsError=false")
	}

	text := result.Content[0].(*TextContent).Text
	if text != "Go is great!" {
		t.Errorf("response = %q, want %q", text, "Go is great!")
	}
	if delegator.lastPeer != "bravo" {
		t.Errorf("lastPeer = %q, want %q", delegator.lastPeer, "bravo")
	}
	if delegator.lastMessage != "What is Go?" {
		t.Errorf("lastMessage = %q, want %q", delegator.lastMessage, "What is Go?")
	}
}

// TestWingmateAskMissingPeer verifies error when peer is missing.
func TestWingmateAskMissingPeer(t *testing.T) {
	delegator := &mockPeerDelegator{response: "ok"}
	handler := NewAskHandler(delegator, nil)

	req := &ToolRequest{
		Name:      ToolNameAsk,
		Arguments: map[string]any{"message": "hello"},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for missing peer")
	}
}

// TestWingmateAskMissingMessage verifies error when message is missing.
func TestWingmateAskMissingMessage(t *testing.T) {
	delegator := &mockPeerDelegator{response: "ok"}
	handler := NewAskHandler(delegator, nil)

	req := &ToolRequest{
		Name:      ToolNameAsk,
		Arguments: map[string]any{"peer": "bravo"},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for missing message")
	}
}

// TestWingmateAskUnknownPeer verifies error for unknown peer.
func TestWingmateAskUnknownPeer(t *testing.T) {
	delegator := &mockPeerDelegator{err: NewMCPError(ErrCodePeerNotFound, "peer not found: ghost")}
	handler := NewAskHandler(delegator, nil)

	req := &ToolRequest{
		Name:      ToolNameAsk,
		Arguments: map[string]any{"peer": "ghost", "message": "hello"},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for unknown peer")
	}
}

// TestWingmateAskSessionPassthrough verifies session_id is passed to delegator.
func TestWingmateAskSessionPassthrough(t *testing.T) {
	delegator := &mockPeerDelegator{response: "continued"}
	handler := NewAskHandler(delegator, nil)

	req := &ToolRequest{
		Name:      ToolNameAsk,
		Arguments: map[string]any{"peer": "bravo", "message": "continue", "session_id": "sess-42"},
		ctx:       context.Background(),
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected IsError=false")
	}
	if delegator.lastSession != "sess-42" {
		t.Errorf("lastSession = %q, want %q", delegator.lastSession, "sess-42")
	}
}

// TestWingmateAskWithLogger verifies logging when logger is provided.
func TestWingmateAskWithLogger(t *testing.T) {
	delegator := &mockPeerDelegator{response: "logged response"}
	logger := &MockLogger{}
	handler := NewAskHandler(delegator, logger)

	req := &ToolRequest{
		Name:      ToolNameAsk,
		Arguments: map[string]any{"peer": "bravo", "message": "test logging"},
		ctx:       context.Background(),
	}

	_, _ = handler(context.Background(), req)

	if len(logger.Entries) == 0 {
		t.Error("expected log entry, got none")
	}

	found := false
	for _, entry := range logger.Entries {
		if entry["tool"] == ToolNameAsk {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log entry with tool name wingmate_ask")
	}
}
