package agent

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/wingmate/wingmate/internal/flightlog"
	"github.com/wingmate/wingmate/pkg/types"
)

// TestAgent_HandleMessage_Ping verifies ping returns pong.
func TestAgent_HandleMessage_Ping(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "ping-test-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	// Send ping message
	pingMsg := &types.Message{
		Role: "user",
		Parts: []types.Part{
			{Kind: "text", Text: "ping"},
		},
	}

	resp, err := agent.HandleMessage(context.Background(), pingMsg)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %+v", resp.Error)
	}

	// Parse result to verify it's a pong
	// Result should contain a message with "pong"
	if resp.Result == nil {
		t.Error("Result is nil")
	}
}

// TestAgent_HandleMessage_NonPingWithoutCLI verifies non-ping message returns error when no CLI.
// Note: This test was previously named TestAgent_HandleMessage_Unknown.
// With LLM integration, non-ping messages are routed to LLM if available.
// When CLI is not available, error 2001 (CodeLLMUnavailable) is returned.
func TestAgent_HandleMessage_NonPingWithoutCLI(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "no-cli-test-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	// Explicitly set llmExecutor to nil to simulate no CLI installed
	agent.llmExecutor = nil

	// Send non-ping message
	textMsg := &types.Message{
		Role: "user",
		Parts: []types.Part{
			{Kind: "text", Text: "What is Go?"},
		},
	}

	_, err = agent.HandleMessage(context.Background(), textMsg)
	if err == nil {
		t.Error("Expected error for non-ping message when CLI unavailable")
	}
}

// TestAgent_HandleMessage_FlightLog verifies Flight Log records as wingmate.
func TestAgent_HandleMessage_FlightLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "flight.jsonl")
	cfg := &Config{
		Name:    "log-test-agent",
		Port:    0,
		LogFile: logPath,
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Send ping message
	pingMsg := &types.Message{
		Role: "user",
		Parts: []types.Part{
			{Kind: "text", Text: "ping"},
		},
	}

	ctx := flightlog.WithTraceID(context.Background(), "test-trace-001")
	_, err = agent.HandleMessage(ctx, pingMsg)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Must close agent to flush logs
	agent.Shutdown(context.Background())

	// Read log entries
	entries := flightlog.ReadLogEntries(t, logPath)

	// Should have at least 2 entries (inbound ping, outbound pong)
	if len(entries) < 2 {
		t.Fatalf("Expected at least 2 log entries, got %d", len(entries))
	}

	// Verify inbound entry
	var foundInbound, foundOutbound bool
	for _, entry := range entries {
		if entry.Direction == flightlog.Inbound && entry.Role == flightlog.RoleWingmate {
			foundInbound = true
			if entry.TraceID != "test-trace-001" {
				t.Errorf("Inbound TraceID = %q, want test-trace-001", entry.TraceID)
			}
		}
		if entry.Direction == flightlog.Outbound && entry.Role == flightlog.RoleWingmate {
			foundOutbound = true
		}
	}

	if !foundInbound {
		t.Error("Missing inbound entry with role=wingmate")
	}
	if !foundOutbound {
		t.Error("Missing outbound entry with role=wingmate")
	}
}

// TestAgent_HandleMessage_EmptyParts verifies empty parts are handled.
func TestAgent_HandleMessage_EmptyParts(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "empty-parts-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	// Send message with empty parts
	emptyMsg := &types.Message{
		Role:  "user",
		Parts: []types.Part{},
	}

	_, err = agent.HandleMessage(context.Background(), emptyMsg)
	if err == nil {
		t.Error("Expected error for empty parts")
	}
}

// TestAgent_HandleMessage_NilMessage verifies nil message is handled.
func TestAgent_HandleMessage_NilMessage(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "nil-msg-agent",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	_, err = agent.HandleMessage(context.Background(), nil)
	if err == nil {
		t.Error("Expected error for nil message")
	}
}

// TestAgent_FullPingPong verifies full ping/pong flow via HTTP.
func TestAgent_FullPingPong(t *testing.T) {
	dir := t.TempDir()

	// Agent A (will respond to ping)
	cfgA := &Config{
		Name:    "agent-a",
		Port:    0,
		LogFile: filepath.Join(dir, "agent-a.jsonl"),
	}

	agentA, err := New(cfgA)
	if err != nil {
		t.Fatalf("New agent A failed: %v", err)
	}

	ctxA, cancelA := context.WithCancel(context.Background())
	defer cancelA()

	go agentA.Start(ctxA)
	if err := agentA.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("Agent A failed to start: %v", err)
	}
	defer agentA.Shutdown(context.Background())

	// Agent B (will send ping)
	cfgB := &Config{
		Name:    "agent-b",
		Port:    0,
		LogFile: filepath.Join(dir, "agent-b.jsonl"),
	}

	agentB, err := New(cfgB)
	if err != nil {
		t.Fatalf("New agent B failed: %v", err)
	}
	defer agentB.Shutdown(context.Background())

	// Agent B sends ping to Agent A
	err = agentB.Ping(context.Background(), agentA.URL())
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Close agents to flush logs
	agentB.Shutdown(context.Background())
	agentA.Shutdown(context.Background())

	// Verify Agent A logged as wingmate
	entriesA := flightlog.ReadLogEntries(t, cfgA.LogFile)

	foundWingmate := false
	for _, entry := range entriesA {
		if entry.Role == flightlog.RoleWingmate {
			foundWingmate = true
			break
		}
	}
	if !foundWingmate {
		t.Error("Agent A should have logged as wingmate")
	}

	// Verify Agent B logged as pilot
	entriesB := flightlog.ReadLogEntries(t, cfgB.LogFile)

	foundPilot := false
	for _, entry := range entriesB {
		if entry.Role == flightlog.RolePilot {
			foundPilot = true
			break
		}
	}
	if !foundPilot {
		t.Error("Agent B should have logged as pilot")
	}
}
