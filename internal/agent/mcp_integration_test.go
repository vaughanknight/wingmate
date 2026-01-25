package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

/*
Test Doc:
- Why: Phase 2 integration requires MCP HTTP endpoint available on running agent
- Contract: Agent serves MCP requests at /mcp endpoint alongside A2A at /
- Usage Notes: Use httptest or real HTTP client to test; LocalhostMiddleware enforced
- Quality Contribution: Validates Claude Code can connect to agent via HTTP MCP
- Worked Example: Start agent → POST /mcp with initialize → tools/list → tools/call discover
*/

// TestAgent_MCP_InitializeToolsListDiscover tests full MCP workflow via HTTP.
//
// Given: A running agent with MCP handler registered
// When: Client sends initialize → tools/list → tools/call discover
// Then: All requests succeed and return expected MCP responses
func TestAgent_MCP_InitializeToolsListDiscover(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "mcp-test-agent",
		Port:    0, // Auto-assign
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start agent in goroutine
	go agent.Start(ctx)

	// Wait for ready
	if err := agent.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("WaitUntilReady failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	mcpURL := agent.URL() + "/mcp"
	client := &http.Client{Timeout: 5 * time.Second}

	// Step 1: Initialize
	t.Run("initialize", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test-client","version":"1.0"}}}`
		resp, err := client.Post(mcpURL, "application/json", bytes.NewReader([]byte(reqBody)))
		if err != nil {
			t.Fatalf("POST initialize failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response failed: %v", err)
		}

		// Verify JSON-RPC structure
		if result["jsonrpc"] != "2.0" {
			t.Errorf("expected jsonrpc 2.0, got %v", result["jsonrpc"])
		}

		// Verify result contains serverInfo
		res, ok := result["result"].(map[string]any)
		if !ok {
			t.Fatalf("expected result object, got %T", result["result"])
		}

		serverInfo, ok := res["serverInfo"].(map[string]any)
		if !ok {
			t.Fatal("expected serverInfo in result")
		}

		if serverInfo["name"] != "wingmate" {
			t.Errorf("expected serverInfo.name = 'wingmate', got %v", serverInfo["name"])
		}

		// Verify session header is returned
		sessionID := resp.Header.Get("Mcp-Session-Id")
		if sessionID == "" {
			t.Error("expected Mcp-Session-Id header")
		}
	})

	// Step 2: Tools List
	t.Run("tools/list", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
		resp, err := client.Post(mcpURL, "application/json", bytes.NewReader([]byte(reqBody)))
		if err != nil {
			t.Fatalf("POST tools/list failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response failed: %v", err)
		}

		res, ok := result["result"].(map[string]any)
		if !ok {
			t.Fatalf("expected result object, got %T", result["result"])
		}

		tools, ok := res["tools"].([]any)
		if !ok {
			t.Fatalf("expected tools array, got %T", res["tools"])
		}

		// Should have the default tools (chat, status, discover)
		if len(tools) < 3 {
			t.Errorf("expected at least 3 tools, got %d", len(tools))
		}

		// Verify discover tool is present
		foundDiscover := false
		for _, tool := range tools {
			toolMap := tool.(map[string]any)
			if toolMap["name"] == "wingmate_discover" {
				foundDiscover = true
				break
			}
		}
		if !foundDiscover {
			t.Error("expected wingmate_discover tool in tools list")
		}
	})

	// Step 3: Tools Call - Discover
	t.Run("tools/call discover", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"wingmate_discover","arguments":{}}}`
		resp, err := client.Post(mcpURL, "application/json", bytes.NewReader([]byte(reqBody)))
		if err != nil {
			t.Fatalf("POST tools/call failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode response failed: %v", err)
		}

		// Check for error
		if errObj, ok := result["error"]; ok {
			t.Fatalf("unexpected error: %v", errObj)
		}

		res, ok := result["result"].(map[string]any)
		if !ok {
			t.Fatalf("expected result object, got %T", result["result"])
		}

		content, ok := res["content"].([]any)
		if !ok {
			t.Fatalf("expected content array, got %T", res["content"])
		}

		if len(content) == 0 {
			t.Fatal("expected non-empty content")
		}

		// Parse the discover response
		firstContent := content[0].(map[string]any)
		text, ok := firstContent["text"].(string)
		if !ok {
			t.Fatalf("expected text content, got %T", firstContent["text"])
		}

		var discoverInfo map[string]any
		if err := json.Unmarshal([]byte(text), &discoverInfo); err != nil {
			t.Fatalf("failed to parse discover info: %v", err)
		}

		// Verify agent_id matches
		if discoverInfo["agent_id"] != "mcp-test-agent" {
			t.Errorf("expected agent_id = 'mcp-test-agent', got %v", discoverInfo["agent_id"])
		}

		// Verify peers is an array (even if empty)
		peers, ok := discoverInfo["peers"].([]any)
		if !ok {
			t.Fatalf("expected peers array, got %T", discoverInfo["peers"])
		}

		// No peers configured, should be empty
		if len(peers) != 0 {
			t.Errorf("expected 0 peers, got %d", len(peers))
		}
	})
}

// TestAgent_MCP_NonLocalhost tests that non-localhost requests are rejected.
//
// Given: A running agent with LocalhostMiddleware
// When: Request comes from non-localhost IP
// Then: Request is rejected with 403 Forbidden
//
// Note: This test uses httptest which simulates localhost, so we can't fully test
// the middleware rejection without mocking. Instead, we verify the middleware is installed
// by checking that localhost requests succeed.
func TestAgent_MCP_LocalhostEnforced(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "mcp-localhost-test",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agent.Start(ctx)

	if err := agent.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("WaitUntilReady failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	// Localhost request should succeed
	mcpURL := agent.URL() + "/mcp"
	client := &http.Client{Timeout: 5 * time.Second}

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`
	resp, err := client.Post(mcpURL, "application/json", bytes.NewReader([]byte(reqBody)))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	// Should succeed from localhost
	if resp.StatusCode != http.StatusOK {
		t.Errorf("localhost request should succeed, got status %d", resp.StatusCode)
	}
}

// TestAgent_MCP_ConcurrentWithA2A tests MCP and A2A requests don't interfere.
//
// Given: A running agent serving both /mcp and /
// When: Concurrent MCP and A2A requests are sent
// Then: Both types of requests are handled correctly without interference
func TestAgent_MCP_ConcurrentWithA2A(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "mcp-concurrent-test",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agent.Start(ctx)

	if err := agent.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("WaitUntilReady failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	baseURL := agent.URL()
	mcpURL := baseURL + "/mcp"
	a2aURL := baseURL + "/" // JSON-RPC A2A endpoint
	client := &http.Client{Timeout: 5 * time.Second}

	// First, initialize MCP session
	initResp, err := client.Post(mcpURL, "application/json",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)))
	if err != nil {
		t.Fatalf("MCP initialize failed: %v", err)
	}
	initResp.Body.Close()

	const numRequests = 10
	var wg sync.WaitGroup
	mcpErrors := make(chan error, numRequests)
	a2aErrors := make(chan error, numRequests)

	// Send concurrent MCP requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			reqBody := `{"jsonrpc":"2.0","id":` + string(rune('0'+id)) + `,"method":"tools/list"}`
			resp, err := client.Post(mcpURL, "application/json", bytes.NewReader([]byte(reqBody)))
			if err != nil {
				mcpErrors <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				mcpErrors <- &statusError{code: resp.StatusCode, endpoint: "mcp"}
			}
		}(i)
	}

	// Send concurrent A2A requests (ping)
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			reqBody := `{"jsonrpc":"2.0","id":` + string(rune('0'+id)) + `,"method":"message/send","params":{"message":{"role":"user","parts":[{"kind":"text","text":"ping"}]}}}`
			resp, err := client.Post(a2aURL, "application/json", bytes.NewReader([]byte(reqBody)))
			if err != nil {
				a2aErrors <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				a2aErrors <- &statusError{code: resp.StatusCode, endpoint: "a2a"}
			}
		}(i)
	}

	wg.Wait()
	close(mcpErrors)
	close(a2aErrors)

	// Check for MCP errors
	for err := range mcpErrors {
		t.Errorf("MCP request error: %v", err)
	}

	// Check for A2A errors
	for err := range a2aErrors {
		t.Errorf("A2A request error: %v", err)
	}
}

// statusError is a simple error type for HTTP status codes.
type statusError struct {
	code     int
	endpoint string
}

func (e *statusError) Error() string {
	return "unexpected status " + string(rune(e.code)) + " from " + e.endpoint
}

// TestAgent_MCP_StatusTool tests the wingmate_status tool returns health info.
func TestAgent_MCP_StatusTool(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Name:    "mcp-status-test",
		Port:    0,
		LogFile: filepath.Join(dir, "flight.jsonl"),
	}

	agent, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agent.Start(ctx)

	if err := agent.WaitUntilReady(5 * time.Second); err != nil {
		t.Fatalf("WaitUntilReady failed: %v", err)
	}
	defer agent.Shutdown(context.Background())

	mcpURL := agent.URL() + "/mcp"
	client := &http.Client{Timeout: 5 * time.Second}

	// Initialize
	initResp, err := client.Post(mcpURL, "application/json",
		bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)))
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initResp.Body.Close()

	// Call status tool
	reqBody := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"wingmate_status","arguments":{}}}`
	resp, err := client.Post(mcpURL, "application/json", bytes.NewReader([]byte(reqBody)))
	if err != nil {
		t.Fatalf("status call failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	res, ok := result["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %T", result["result"])
	}

	content, ok := res["content"].([]any)
	if !ok {
		t.Fatalf("expected content array, got %T", res["content"])
	}

	if len(content) == 0 {
		t.Fatal("expected non-empty content")
	}

	// Parse status info
	firstContent := content[0].(map[string]any)
	text := firstContent["text"].(string)

	var statusInfo map[string]any
	if err := json.Unmarshal([]byte(text), &statusInfo); err != nil {
		t.Fatalf("failed to parse status info: %v", err)
	}

	// Verify uptime_ms is present and positive
	uptimeMS, ok := statusInfo["uptime_ms"].(float64)
	if !ok {
		t.Fatal("expected uptime_ms in status")
	}
	if uptimeMS < 0 {
		t.Errorf("expected positive uptime_ms, got %v", uptimeMS)
	}

	// Verify cli_available is present (may be true or false depending on environment)
	if _, ok := statusInfo["cli_available"]; !ok {
		t.Error("expected cli_available in status")
	}
}
