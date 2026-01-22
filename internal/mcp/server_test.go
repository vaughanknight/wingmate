package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Server Lifecycle Tests (T006)
// =============================================================================

// TestServerStart verifies that a server can be created and started without error.
func TestServerStart(t *testing.T) {
	// Arrange: Create server with test transport
	transport := NewTransport(strings.NewReader(""), io.Discard)
	server := NewServer(ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}, transport)

	// Act: Start the server
	err := server.Start()

	// Assert: Should start without error
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Cleanup
	server.Stop()
}

// TestServerStop verifies that a running server can be stopped cleanly.
func TestServerStop(t *testing.T) {
	// Arrange: Create and start server
	transport := NewTransport(strings.NewReader(""), io.Discard)
	server := NewServer(ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}, transport)
	if err := server.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Act: Stop the server
	err := server.Stop()

	// Assert: Should stop without error
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

// TestServerRunContext verifies that context cancellation stops the server.
func TestServerRunContext(t *testing.T) {
	// Arrange: Create server with test transport that blocks on read
	pr, pw := io.Pipe()
	defer pw.Close()
	transport := NewTransport(pr, io.Discard)
	server := NewServer(ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}, transport)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Act: Run server in goroutine, cancel after short delay
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	// Give server time to start, then cancel
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Assert: Server should exit within reasonable time
	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Errorf("Run returned unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Server did not exit after context cancellation")
	}
}

// TestServerDoubleStartReturnsError verifies that starting a server twice returns an error.
func TestServerDoubleStartReturnsError(t *testing.T) {
	// Arrange: Create and start server
	transport := NewTransport(strings.NewReader(""), io.Discard)
	server := NewServer(ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}, transport)
	if err := server.Start(); err != nil {
		t.Fatalf("First Start failed: %v", err)
	}
	defer server.Stop()

	// Act: Try to start again
	err := server.Start()

	// Assert: Should return error
	if err == nil {
		t.Error("Expected error on double start, got nil")
	}
}

// TestServerStopIdempotent verifies that Stop can be called multiple times safely.
func TestServerStopIdempotent(t *testing.T) {
	// Arrange: Create and start server
	transport := NewTransport(strings.NewReader(""), io.Discard)
	server := NewServer(ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}, transport)
	if err := server.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Act: Stop multiple times
	err1 := server.Stop()
	err2 := server.Stop()

	// Assert: Both should succeed without error
	if err1 != nil {
		t.Errorf("First Stop failed: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second Stop failed: %v", err2)
	}
}

// =============================================================================
// Initialize Handler Tests (T008)
// =============================================================================

// TestServerInitializeHandshake verifies the MCP initialize request is handled correctly.
func TestServerInitializeHandshake(t *testing.T) {
	// Arrange: Create initialize request
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
	requestJSON, _ := json.Marshal(initRequest)
	input := string(requestJSON) + "\n"

	var output bytes.Buffer
	transport := NewTransport(strings.NewReader(input), &output)
	server := NewServer(ServerConfig{
		Name:    "wingmate",
		Version: "0.1.0",
	}, transport)

	// Act: Run server (will exit after processing request due to EOF)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = server.Run(ctx)

	// Assert: Should have written a valid initialize response
	outputStr := output.String()
	if outputStr == "" {
		t.Fatal("Expected initialize response, got empty output")
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(outputStr)), &response); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	// Verify JSON-RPC fields
	if response["jsonrpc"] != "2.0" {
		t.Errorf("got jsonrpc %v, want '2.0'", response["jsonrpc"])
	}
	if response["id"].(float64) != 1 {
		t.Errorf("got id %v, want 1", response["id"])
	}

	// Verify result structure
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not an object: %v", response["result"])
	}

	// Verify protocol version
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("got protocolVersion %v, want '2024-11-05'", result["protocolVersion"])
	}

	// Verify server info
	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("serverInfo is not an object: %v", result["serverInfo"])
	}
	if serverInfo["name"] != "wingmate" {
		t.Errorf("got server name %v, want 'wingmate'", serverInfo["name"])
	}
	if serverInfo["version"] != "0.1.0" {
		t.Errorf("got server version %v, want '0.1.0'", serverInfo["version"])
	}

	// Verify capabilities include tools
	capabilities, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities is not an object: %v", result["capabilities"])
	}
	if _, hasTools := capabilities["tools"]; !hasTools {
		t.Error("capabilities should include 'tools' field")
	}
}

// TestServerRejectsPreInitializeRequests verifies that non-initialize requests
// before initialization are rejected with an error.
func TestServerRejectsPreInitializeRequests(t *testing.T) {
	// Arrange: Create a tools/list request without prior initialize
	toolsRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	requestJSON, _ := json.Marshal(toolsRequest)
	input := string(requestJSON) + "\n"

	var output bytes.Buffer
	transport := NewTransport(strings.NewReader(input), &output)
	server := NewServer(ServerConfig{
		Name:    "wingmate",
		Version: "0.1.0",
	}, transport)

	// Act: Run server
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = server.Run(ctx)

	// Assert: Should have written an error response
	outputStr := output.String()
	if outputStr == "" {
		t.Fatal("Expected error response, got empty output")
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(outputStr)), &response); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	// Verify it's an error response
	errObj, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("Expected error response, got: %v", response)
	}

	// Verify error code is -32600 (Invalid Request) per JSON-RPC spec
	errCode := int(errObj["code"].(float64))
	if errCode != -32600 {
		t.Errorf("got error code %d, want -32600", errCode)
	}
}

// =============================================================================
// Tools/List Handler Tests (T010)
// =============================================================================

// TestServerToolsListEmpty verifies that tools/list returns an empty tools array.
func TestServerToolsListEmpty(t *testing.T) {
	// Arrange: Create initialize request followed by tools/list
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
	toolsRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}

	initJSON, _ := json.Marshal(initRequest)
	toolsJSON, _ := json.Marshal(toolsRequest)
	input := string(initJSON) + "\n" + string(toolsJSON) + "\n"

	var output bytes.Buffer
	transport := NewTransport(strings.NewReader(input), &output)
	server := NewServer(ServerConfig{
		Name:    "wingmate",
		Version: "0.1.0",
	}, transport)

	// Act: Run server
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = server.Run(ctx)

	// Assert: Should have two responses (init and tools/list)
	outputStr := output.String()
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected 2 responses, got %d: %s", len(lines), outputStr)
	}

	// Parse the tools/list response (second line)
	var toolsResponse map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &toolsResponse); err != nil {
		t.Fatalf("tools/list response is not valid JSON: %v", err)
	}

	// Verify JSON-RPC fields
	if toolsResponse["jsonrpc"] != "2.0" {
		t.Errorf("got jsonrpc %v, want '2.0'", toolsResponse["jsonrpc"])
	}
	if toolsResponse["id"].(float64) != 2 {
		t.Errorf("got id %v, want 2", toolsResponse["id"])
	}

	// Verify result contains empty tools array
	result, ok := toolsResponse["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not an object: %v", toolsResponse["result"])
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools is not an array: %v", result["tools"])
	}
	if len(tools) != 0 {
		t.Errorf("expected empty tools array, got %d tools", len(tools))
	}
}

// =============================================================================
// Flight Log Integration Tests (T012)
// =============================================================================

// mockLogger implements flightlog.Logger for testing.
type mockLogger struct {
	entries []mockEntry
}

type mockEntry struct {
	summary string
}

func (m *mockLogger) Record(entry any) error {
	// Extract summary from the entry map or struct
	switch e := entry.(type) {
	case map[string]any:
		if summary, ok := e["summary"].(string); ok {
			m.entries = append(m.entries, mockEntry{summary: summary})
		}
	}
	return nil
}

func (m *mockLogger) Close() error {
	return nil
}

// TestServerLogsStartup verifies that server startup is logged to Flight Log.
func TestServerLogsStartup(t *testing.T) {
	// Arrange: Create server with mock logger
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
	requestJSON, _ := json.Marshal(initRequest)
	input := string(requestJSON) + "\n"

	var output bytes.Buffer
	transport := NewTransport(strings.NewReader(input), &output)
	logger := &mockLogger{}
	server := NewServerWithLogger(ServerConfig{
		Name:    "wingmate",
		Version: "0.1.0",
	}, transport, logger)

	// Act: Run server
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = server.Run(ctx)

	// Assert: Should have logged startup event
	found := false
	for _, entry := range logger.entries {
		if strings.Contains(entry.summary, "started") || strings.Contains(entry.summary, "startup") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected startup event to be logged, but it wasn't")
	}
}

// TestServerGracefulShutdownOnEOF verifies that shutdown is logged when EOF is received.
func TestServerGracefulShutdownOnEOF(t *testing.T) {
	// Arrange: Create server with mock logger and empty input (EOF immediately)
	var output bytes.Buffer
	transport := NewTransport(strings.NewReader(""), &output)
	logger := &mockLogger{}
	server := NewServerWithLogger(ServerConfig{
		Name:    "wingmate",
		Version: "0.1.0",
	}, transport, logger)

	// Act: Run server (will exit on EOF)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = server.Run(ctx)

	// Assert: Should have logged shutdown event
	found := false
	for _, entry := range logger.entries {
		if strings.Contains(entry.summary, "stopped") || strings.Contains(entry.summary, "shutdown") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected shutdown event to be logged, but it wasn't")
	}
}

// =============================================================================
// Tools/Call Handler Tests (T004)
// =============================================================================

// TestServerToolsCallDispatchesToHandler verifies that tools/call invokes
// the registered handler and returns the result.
//
// Given: A server with a mock tool handler registered
// When: Client sends tools/call for that tool
// Then: Handler is invoked and result is returned
func TestServerToolsCallDispatchesToHandler(t *testing.T) {
	// Arrange: Create requests
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
		},
	}
	callRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "test_tool",
			"arguments": map[string]any{"input": "hello"},
		},
	}

	initJSON, _ := json.Marshal(initRequest)
	callJSON, _ := json.Marshal(callRequest)
	input := string(initJSON) + "\n" + string(callJSON) + "\n"

	var output bytes.Buffer
	transport := NewTransport(strings.NewReader(input), &output)
	server := NewServer(ServerConfig{Name: "test", Version: "1.0"}, transport)

	// Register a test tool with handler
	testTool := &ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]any{"type": "object"},
	}
	server.RegisterTool(testTool)

	// Register handler for the tool
	handlerCalled := false
	server.RegisterHandler("test_tool", func(ctx context.Context, req *ToolRequest) (*ToolResult, error) {
		handlerCalled = true
		if req.Arguments["input"] != "hello" {
			t.Errorf("Expected input 'hello', got %v", req.Arguments["input"])
		}
		return &ToolResult{
			Content: []Content{&TextContent{Text: "world"}},
		}, nil
	})

	// Act: Run server
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = server.Run(ctx)

	// Assert: Handler was called
	if !handlerCalled {
		t.Error("Handler was not called")
	}

	// Assert: Verify response
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected 2 responses, got %d", len(lines))
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &response); err != nil {
		t.Fatalf("Invalid JSON response: %v", err)
	}

	// Should have result with content
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object, got: %v", response)
	}

	content, ok := result["content"].([]any)
	if !ok {
		t.Fatalf("Expected content array, got: %v", result)
	}

	if len(content) == 0 {
		t.Fatal("Expected at least one content item")
	}

	firstContent := content[0].(map[string]any)
	if firstContent["text"] != "world" {
		t.Errorf("Expected text 'world', got %v", firstContent["text"])
	}
}

// TestServerToolsCallUnknownToolReturnsError verifies that calling an
// unregistered tool returns an error.
//
// Given: A server with no handlers registered for "unknown_tool"
// When: Client sends tools/call for "unknown_tool"
// Then: Server returns error with appropriate code
func TestServerToolsCallUnknownToolReturnsError(t *testing.T) {
	// Arrange
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
		},
	}
	callRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "unknown_tool",
			"arguments": map[string]any{},
		},
	}

	initJSON, _ := json.Marshal(initRequest)
	callJSON, _ := json.Marshal(callRequest)
	input := string(initJSON) + "\n" + string(callJSON) + "\n"

	var output bytes.Buffer
	transport := NewTransport(strings.NewReader(input), &output)
	server := NewServer(ServerConfig{Name: "test", Version: "1.0"}, transport)

	// Act: Run server (no tools registered)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = server.Run(ctx)

	// Assert: Should return error for unknown tool
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected 2 responses, got %d", len(lines))
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &response); err != nil {
		t.Fatalf("Invalid JSON response: %v", err)
	}

	// Should have error
	errObj, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("Expected error response, got: %v", response)
	}

	// Error code should indicate tool not found (per JSON-RPC -32601 Method not found)
	errCode := int(errObj["code"].(float64))
	if errCode != -32601 {
		t.Errorf("Expected error code -32601 (Method not found), got %d", errCode)
	}
}

// TestServerToolsCallMissingNameReturnsError verifies that calling
// tools/call without a tool name returns an error.
//
// Given: A server with tools registered
// When: Client sends tools/call without name parameter
// Then: Server returns error with appropriate code
func TestServerToolsCallMissingNameReturnsError(t *testing.T) {
	// Arrange
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
		},
	}
	callRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			// Missing "name" field
			"arguments": map[string]any{},
		},
	}

	initJSON, _ := json.Marshal(initRequest)
	callJSON, _ := json.Marshal(callRequest)
	input := string(initJSON) + "\n" + string(callJSON) + "\n"

	var output bytes.Buffer
	transport := NewTransport(strings.NewReader(input), &output)
	server := NewServer(ServerConfig{Name: "test", Version: "1.0"}, transport)

	// Act
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = server.Run(ctx)

	// Assert: Should return error
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected 2 responses, got %d", len(lines))
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &response); err != nil {
		t.Fatalf("Invalid JSON response: %v", err)
	}

	// Should have error
	errObj, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("Expected error response, got: %v", response)
	}

	// Error code should indicate invalid params (-32602)
	errCode := int(errObj["code"].(float64))
	if errCode != -32602 {
		t.Errorf("Expected error code -32602 (Invalid params), got %d", errCode)
	}
}
