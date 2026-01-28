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

// TestIntegration_DelegationRoundTrip verifies full MCP → A2A delegation.
// Alpha asks bravo a question via wingmate_ask, bravo responds with pong.
func TestIntegration_DelegationRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	// Start bravo (responds to ping)
	cfg2 := &agent.Config{
		Name:        "bravo",
		Description: "Backend dev",
		Port:        0,
		LogFile:     filepath.Join(tmpDir, "bravo.jsonl"),
	}
	agent2, err := agent.New(cfg2)
	if err != nil {
		t.Fatalf("Failed to create bravo: %v", err)
	}
	go agent2.Start(ctx)
	if err := agent2.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("bravo not ready: %v", err)
	}
	defer agent2.Shutdown(context.Background())

	// Start alpha with bravo as peer
	cfg1 := &agent.Config{
		Name:        "alpha",
		Description: "iOS dev",
		Port:        0,
		Peers:       []string{agent2.URL()},
		LogFile:     filepath.Join(tmpDir, "alpha.jsonl"),
	}
	agent1, err := agent.New(cfg1)
	if err != nil {
		t.Fatalf("Failed to create alpha: %v", err)
	}
	go agent1.Start(ctx)
	if err := agent1.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("alpha not ready: %v", err)
	}
	defer agent1.Shutdown(context.Background())

	// Wait for peer probing
	time.Sleep(1 * time.Second)

	// MCP handshake
	mcpURL := agent1.URL() + "/mcp"

	// Initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	resp, err := http.Post(mcpURL, "application/json", strings.NewReader(initReq))
	if err != nil {
		t.Fatalf("MCP initialize failed: %v", err)
	}
	sessionID := resp.Header.Get("Mcp-Session-Id")
	resp.Body.Close()

	// Initialized notification
	notifReq := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	req2, _ := http.NewRequest("POST", mcpURL, strings.NewReader(notifReq))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Mcp-Session-Id", sessionID)
	resp2, _ := http.DefaultClient.Do(req2)
	resp2.Body.Close()

	// Call wingmate_ask: ask bravo to ping
	callReq := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"%s","arguments":{"peer":"bravo","message":"ping"}}}`, mcp.ToolNameAsk)
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

	// Parse response
	var rpcResp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&rpcResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if rpcResp.Result.IsError {
		t.Fatalf("wingmate_ask returned error: %v", rpcResp.Result.Content)
	}

	if len(rpcResp.Result.Content) == 0 {
		t.Fatal("response has no content")
	}

	responseText := rpcResp.Result.Content[0].Text
	if responseText != "pong" {
		t.Errorf("response = %q, want %q", responseText, "pong")
	}
}

// TestIntegration_DelegationUnknownPeer verifies error for unknown peer via MCP.
func TestIntegration_DelegationUnknownPeer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	cfg := &agent.Config{
		Name:    "alpha",
		Port:    0,
		LogFile: filepath.Join(tmpDir, "alpha.jsonl"),
	}
	agent1, err := agent.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create alpha: %v", err)
	}
	go agent1.Start(ctx)
	if err := agent1.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("alpha not ready: %v", err)
	}
	defer agent1.Shutdown(context.Background())

	// MCP handshake
	mcpURL := agent1.URL() + "/mcp"

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	resp, err := http.Post(mcpURL, "application/json", strings.NewReader(initReq))
	if err != nil {
		t.Fatalf("MCP initialize failed: %v", err)
	}
	sessionID := resp.Header.Get("Mcp-Session-Id")
	resp.Body.Close()

	notifReq := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	req2, _ := http.NewRequest("POST", mcpURL, strings.NewReader(notifReq))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Mcp-Session-Id", sessionID)
	resp2, _ := http.DefaultClient.Do(req2)
	resp2.Body.Close()

	// Ask unknown peer
	callReq := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"%s","arguments":{"peer":"ghost","message":"hello"}}}`, mcp.ToolNameAsk)
	req3, _ := http.NewRequest("POST", mcpURL, strings.NewReader(callReq))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Mcp-Session-Id", sessionID)

	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("MCP tools/call failed: %v", err)
	}
	defer resp3.Body.Close()

	var rpcResp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&rpcResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should be an error
	if !rpcResp.Result.IsError {
		t.Error("expected IsError=true for unknown peer")
	}

	if len(rpcResp.Result.Content) > 0 {
		text := rpcResp.Result.Content[0].Text
		if !strings.Contains(text, "peer not found") {
			t.Errorf("error text = %q, want to contain 'peer not found'", text)
		}
	}
}
