package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wingmate/wingmate/internal/agent"
	"github.com/wingmate/wingmate/internal/flightlog"
)

func TestIntegration_PingPong(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create temp directories for logs
	tmpDir := t.TempDir()
	logA := filepath.Join(tmpDir, "agent-a.jsonl")
	logB := filepath.Join(tmpDir, "agent-b.jsonl")

	// Configure Agent A (will receive ping)
	configA := &agent.Config{
		Name:    "agent-a",
		Port:    19000, // Use high ports to avoid conflicts
		LogFile: logA,
		Verbose: false,
	}

	// Configure Agent B (will send ping)
	configB := &agent.Config{
		Name:    "agent-b",
		Port:    19001,
		LogFile: logB,
		Verbose: false,
	}

	// Start Agent A
	agentA, err := agent.New(configA)
	if err != nil {
		t.Fatalf("Failed to create agent A: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- agentA.Start(ctx)
	}()

	// Wait for Agent A to be ready
	if err := agentA.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("Agent A failed to start: %v", err)
	}
	defer agentA.Shutdown(context.Background())

	// Create Agent B (no server needed for ping)
	agentB, err := agent.New(configB)
	if err != nil {
		t.Fatalf("Failed to create agent B: %v", err)
	}
	defer agentB.Shutdown(context.Background())

	// Agent B pings Agent A
	if err := agentB.Ping(ctx, agentA.URL()); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Give time for logs to be written
	time.Sleep(100 * time.Millisecond)

	// Verify Agent A's Flight Log has an inbound ping
	entriesA := readLogEntries(t, logA)
	if len(entriesA) == 0 {
		t.Error("Agent A's flight log is empty")
	}

	foundInboundPing := false
	for _, entry := range entriesA {
		if entry.Direction == flightlog.Inbound && entry.Role == flightlog.RoleWingmate {
			foundInboundPing = true
			break
		}
	}
	if !foundInboundPing {
		t.Error("Agent A's flight log does not contain inbound ping as wingmate")
	}

	// Verify Agent B's Flight Log has an outbound ping
	entriesB := readLogEntries(t, logB)
	if len(entriesB) == 0 {
		t.Error("Agent B's flight log is empty")
	}

	foundOutboundPing := false
	for _, entry := range entriesB {
		if entry.Direction == flightlog.Outbound && entry.Role == flightlog.RolePilot {
			foundOutboundPing = true
			break
		}
	}
	if !foundOutboundPing {
		t.Error("Agent B's flight log does not contain outbound ping as pilot")
	}
}

func TestIntegration_AgentCard(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create temp directory for logs
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "agent.jsonl")

	// Configure and start agent
	cfg := &agent.Config{
		Name:    "test-agent",
		Port:    19002,
		LogFile: logFile,
		Verbose: false,
	}

	a, err := agent.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Start(ctx)
	}()

	if err := a.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("Agent failed to start: %v", err)
	}
	defer a.Shutdown(context.Background())

	// Fetch Agent Card via HTTP
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(a.URL() + "/.well-known/agent.json")
	if err != nil {
		t.Fatalf("Failed to fetch Agent Card: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify JSON structure
	var card map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatalf("Failed to decode Agent Card: %v", err)
	}

	if card["name"] != "test-agent" {
		t.Errorf("Expected name 'test-agent', got %v", card["name"])
	}

	if card["url"] != a.URL() {
		t.Errorf("Expected URL %s, got %v", a.URL(), card["url"])
	}
}

func TestIntegration_GetPeerCard(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	logA := filepath.Join(tmpDir, "agent-a.jsonl")
	logB := filepath.Join(tmpDir, "agent-b.jsonl")

	// Start Agent A
	configA := &agent.Config{
		Name:    "card-agent",
		Port:    19003,
		LogFile: logA,
		Verbose: false,
	}

	agentA, err := agent.New(configA)
	if err != nil {
		t.Fatalf("Failed to create agent A: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- agentA.Start(ctx)
	}()

	if err := agentA.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("Agent A failed to start: %v", err)
	}
	defer agentA.Shutdown(context.Background())

	// Create Agent B to fetch card
	configB := &agent.Config{
		Name:    "fetcher-agent",
		Port:    19004,
		LogFile: logB,
		Verbose: false,
	}

	agentB, err := agent.New(configB)
	if err != nil {
		t.Fatalf("Failed to create agent B: %v", err)
	}
	defer agentB.Shutdown(context.Background())

	// Agent B fetches Agent A's card
	card, err := agentB.GetPeerCard(ctx, agentA.URL())
	if err != nil {
		t.Fatalf("Failed to get peer card: %v", err)
	}

	if card.Name != "card-agent" {
		t.Errorf("Expected name 'card-agent', got %s", card.Name)
	}

	if card.URL != agentA.URL() {
		t.Errorf("Expected URL %s, got %s", agentA.URL(), card.URL)
	}
}

func TestIntegration_MultipleMessages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	logA := filepath.Join(tmpDir, "agent-a.jsonl")
	logB := filepath.Join(tmpDir, "agent-b.jsonl")

	// Start Agent A
	configA := &agent.Config{
		Name:    "multi-agent-a",
		Port:    19005,
		LogFile: logA,
		Verbose: false,
	}

	agentA, err := agent.New(configA)
	if err != nil {
		t.Fatalf("Failed to create agent A: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- agentA.Start(ctx)
	}()

	if err := agentA.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("Agent A failed to start: %v", err)
	}
	defer agentA.Shutdown(context.Background())

	// Create Agent B
	configB := &agent.Config{
		Name:    "multi-agent-b",
		Port:    19006,
		LogFile: logB,
		Verbose: false,
	}

	agentB, err := agent.New(configB)
	if err != nil {
		t.Fatalf("Failed to create agent B: %v", err)
	}
	defer agentB.Shutdown(context.Background())

	// Send multiple pings
	for i := 0; i < 5; i++ {
		if err := agentB.Ping(ctx, agentA.URL()); err != nil {
			t.Fatalf("Ping %d failed: %v", i+1, err)
		}
	}

	// Give time for logs to be written
	time.Sleep(100 * time.Millisecond)

	// Verify correct number of log entries
	entriesA := readLogEntries(t, logA)
	if len(entriesA) < 5 {
		t.Errorf("Expected at least 5 entries in Agent A log, got %d", len(entriesA))
	}

	entriesB := readLogEntries(t, logB)
	if len(entriesB) < 5 {
		t.Errorf("Expected at least 5 entries in Agent B log, got %d", len(entriesB))
	}
}

// readLogEntries reads all entries from a flight log file.
func readLogEntries(t *testing.T, path string) []*flightlog.Entry {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("Failed to read log file: %v", err)
	}

	var entries []*flightlog.Entry
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry flightlog.Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}
		entries = append(entries, &entry)
	}

	return entries
}
