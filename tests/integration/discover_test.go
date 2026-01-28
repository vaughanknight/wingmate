package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wingmate/wingmate/internal/agent"
	"github.com/wingmate/wingmate/internal/mcp"
)

// TestIntegration_DiscoverReturnsPeerInfo verifies that wingmate_discover returns
// rich peer information after startup probing.
func TestIntegration_DiscoverReturnsPeerInfo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	// Start agent2 (bravo) first — the peer to be discovered
	cfg2 := &agent.Config{
		Name:        "bravo",
		Description: "Backend dev",
		Port:        0,
		LogFile:     filepath.Join(tmpDir, "bravo.jsonl"),
	}
	agent2, err := agent.New(cfg2)
	if err != nil {
		t.Fatalf("Failed to create agent2: %v", err)
	}
	go agent2.Start(ctx)
	if err := agent2.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("agent2 not ready: %v", err)
	}
	defer agent2.Shutdown(context.Background())

	// Start agent1 (alpha) with bravo as a peer
	cfg1 := &agent.Config{
		Name:        "alpha",
		Description: "iOS dev",
		Port:        0,
		Peers:       []string{agent2.URL()},
		LogFile:     filepath.Join(tmpDir, "alpha.jsonl"),
	}
	agent1, err := agent.New(cfg1)
	if err != nil {
		t.Fatalf("Failed to create agent1: %v", err)
	}
	go agent1.Start(ctx)
	if err := agent1.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("agent1 not ready: %v", err)
	}
	defer agent1.Shutdown(context.Background())

	// Wait for peer probing to complete
	time.Sleep(1 * time.Second)

	// Call wingmate_discover via MCP HTTP endpoint
	mcpURL := agent1.URL() + "/mcp"

	// Step 1: Initialize MCP session
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	resp, err := http.Post(mcpURL, "application/json", strings.NewReader(initReq))
	if err != nil {
		t.Fatalf("MCP initialize failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("MCP initialize status = %d, want 200", resp.StatusCode)
	}

	// Extract session ID from response header
	sessionID := resp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("No Mcp-Session-Id header in initialize response")
	}

	// Step 2: Send initialized notification
	notifReq := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	req2, _ := http.NewRequest("POST", mcpURL, strings.NewReader(notifReq))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Mcp-Session-Id", sessionID)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("MCP initialized notification failed: %v", err)
	}
	resp2.Body.Close()

	// Step 3: Call tools/call with wingmate_discover
	callReq := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"%s","arguments":{}}}`, mcp.ToolNameDiscover)
	req3, _ := http.NewRequest("POST", mcpURL, strings.NewReader(callReq))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Mcp-Session-Id", sessionID)

	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("MCP tools/call failed: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("MCP tools/call status = %d, want 200", resp3.StatusCode)
	}

	// Parse JSON-RPC response
	var rpcResp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&rpcResp); err != nil {
		t.Fatalf("Failed to decode MCP response: %v", err)
	}

	if len(rpcResp.Result.Content) == 0 {
		t.Fatal("MCP response has no content")
	}

	// Parse the discover info from the text content
	var discoverInfo mcp.DiscoverInfo
	if err := json.Unmarshal([]byte(rpcResp.Result.Content[0].Text), &discoverInfo); err != nil {
		t.Fatalf("Failed to parse discover info: %v", err)
	}

	// Verify agent identity
	if discoverInfo.AgentID != "alpha" {
		t.Errorf("AgentID = %q, want %q", discoverInfo.AgentID, "alpha")
	}

	// Verify peer info
	if len(discoverInfo.Peers) != 1 {
		t.Fatalf("Peers count = %d, want 1", len(discoverInfo.Peers))
	}

	peer := discoverInfo.Peers[0]
	if peer.Name != "bravo" {
		t.Errorf("peer.Name = %q, want %q", peer.Name, "bravo")
	}
	if peer.Description != "Backend dev" {
		t.Errorf("peer.Description = %q, want %q", peer.Description, "Backend dev")
	}
	if !peer.Available {
		t.Error("peer.Available = false, want true")
	}
	if peer.URL == "" {
		t.Error("peer.URL is empty")
	}
}
