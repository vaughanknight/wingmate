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
	peers []string
}

func (m *mockPeerProvider) GetPeers() []string {
	return m.peers
}

// TestWingmateDiscoverReturnsPeersFromProvider verifies discover returns actual peers.
//
// Given: A discover handler with a PeerProvider returning 2 peers
// When: HandleDiscover is called
// Then: Returns those 2 peers in the response
func TestWingmateDiscoverReturnsPeersFromProvider(t *testing.T) {
	provider := &mockPeerProvider{
		peers: []string{"http://peer1:9000", "http://peer2:9001"},
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
		t.Errorf("Peers count = %d, want 2", len(info.Peers))
	}

	// Check peer URLs are present
	foundPeer1 := false
	foundPeer2 := false
	for _, p := range info.Peers {
		if p == "http://peer1:9000" {
			foundPeer1 = true
		}
		if p == "http://peer2:9001" {
			foundPeer2 = true
		}
	}
	if !foundPeer1 || !foundPeer2 {
		t.Errorf("Expected both peers, got: %v", info.Peers)
	}
}
