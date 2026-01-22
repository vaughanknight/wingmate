package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/wingmate/wingmate/internal/llm"
	"github.com/wingmate/wingmate/internal/mcp"
)

// testLogger is a minimal logger for testing that captures log entries.
type testLogger struct {
	entries []map[string]any
}

func (l *testLogger) Record(entry any) error {
	if m, ok := entry.(map[string]any); ok {
		l.entries = append(l.entries, m)
	}
	return nil
}

func (l *testLogger) Close() error {
	return nil
}

// mockLLMExecutor is a mock LLM executor for testing.
type mockLLMExecutor struct {
	response    string
	sessionID   string
	err         error
	isInstalled bool
}

func (m *mockLLMExecutor) Execute(ctx context.Context, prompt string, sessionID string) (*llm.CLIResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.CLIResponse{
		Result:    m.response,
		SessionID: m.sessionID,
		Usage: llm.Usage{
			InputTokens:  10,
			OutputTokens: 50,
		},
		Metadata: llm.Metadata{
			Model: "test-model",
		},
	}, nil
}

func (m *mockLLMExecutor) IsInstalled() bool {
	return m.isInstalled
}

// sendJSONRPC writes a JSON-RPC request to the writer and returns error if any.
func sendJSONRPC(w io.Writer, method string, id any, params any) error {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if id != nil {
		msg["id"] = id
	}
	if params != nil {
		msg["params"] = params
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

// readJSONRPC reads a JSON-RPC response from the reader.
func readJSONRPC(r *bufio.Reader) (map[string]any, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func TestIntegration_MCPServerInitialize(t *testing.T) {
	// Create io.Pipe for stdin/stdout simulation
	// Server reads from stdinReader, writes to stdoutWriter
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// Create transport with pipes
	transport := mcp.NewTransport(stdinReader, stdoutWriter)

	// Create server with test logger
	logger := &testLogger{}
	server := mcp.NewServerWithLogger(
		mcp.ServerConfig{
			Name:    "test-server",
			Version: "1.0.0",
		},
		transport,
		logger,
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(ctx)
	}()

	// Create buffered reader for responses
	reader := bufio.NewReader(stdoutReader)

	// Send initialize request
	err := sendJSONRPC(stdinWriter, "initialize", 1, map[string]any{
		"protocolVersion": "2024-11-05",
		"clientInfo": map[string]any{
			"name":    "test-client",
			"version": "1.0.0",
		},
	})
	if err != nil {
		t.Fatalf("Failed to send initialize: %v", err)
	}

	// Read response
	resp, err := readJSONRPC(reader)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify response structure
	if resp["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc 2.0, got %v", resp["jsonrpc"])
	}

	if resp["id"].(float64) != 1 {
		t.Errorf("Expected id 1, got %v", resp["id"])
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object, got %T", resp["result"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("Expected serverInfo object, got %T", result["serverInfo"])
	}

	if serverInfo["name"] != "test-server" {
		t.Errorf("Expected server name 'test-server', got %v", serverInfo["name"])
	}

	if serverInfo["version"] != "1.0.0" {
		t.Errorf("Expected server version '1.0.0', got %v", serverInfo["version"])
	}

	// Cancel to stop server
	cancel()

	// Close pipes to ensure server exits
	stdinWriter.Close()
	stdinReader.Close()

	// Wait for server to stop
	select {
	case err := <-serverErr:
		if err != nil && err != context.Canceled {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Server did not stop in time")
	}
}

func TestIntegration_MCPServerToolsList(t *testing.T) {
	// Create io.Pipe for stdin/stdout simulation
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// Create transport
	transport := mcp.NewTransport(stdinReader, stdoutWriter)

	// Create server and register tools
	server := mcp.NewServer(
		mcp.ServerConfig{
			Name:    "test-server",
			Version: "1.0.0",
		},
		transport,
	)

	// Register default tools
	for _, tool := range mcp.DefaultTools() {
		server.RegisterTool(tool)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(ctx)
	}()

	// Create buffered reader for responses
	reader := bufio.NewReader(stdoutReader)

	// Send initialize request first
	err := sendJSONRPC(stdinWriter, "initialize", 1, map[string]any{
		"protocolVersion": "2024-11-05",
	})
	if err != nil {
		t.Fatalf("Failed to send initialize: %v", err)
	}

	// Read initialize response
	_, err = readJSONRPC(reader)
	if err != nil {
		t.Fatalf("Failed to read initialize response: %v", err)
	}

	// Send tools/list request
	err = sendJSONRPC(stdinWriter, "tools/list", 2, nil)
	if err != nil {
		t.Fatalf("Failed to send tools/list: %v", err)
	}

	// Read response
	resp, err := readJSONRPC(reader)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify response
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object, got %T", resp["result"])
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("Expected tools array, got %T", result["tools"])
	}

	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]any)
		toolNames[toolMap["name"].(string)] = true
	}

	expectedTools := []string{"wingmate_chat", "wingmate_status", "wingmate_discover"}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool %s not found", name)
		}
	}

	// Cleanup
	cancel()
	stdinWriter.Close()
	stdinReader.Close()

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
	}
}

func TestIntegration_MCPServerToolsCallStatus(t *testing.T) {
	// Create io.Pipe for stdin/stdout simulation
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// Create transport
	transport := mcp.NewTransport(stdinReader, stdoutWriter)

	// Create server and register tools
	server := mcp.NewServer(
		mcp.ServerConfig{
			Name:    "test-server",
			Version: "1.0.0",
		},
		transport,
	)

	// Register default tools
	for _, tool := range mcp.DefaultTools() {
		server.RegisterTool(tool)
	}

	// Create mock executor for status handler
	mockExecutor := &mockLLMExecutor{
		response:    "test response",
		sessionID:   "test-session",
		isInstalled: true,
	}

	// Register handlers
	startTime := time.Now()
	server.RegisterHandler(mcp.ToolNameStatus, mcp.NewStatusHandler(mockExecutor, startTime))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(ctx)
	}()

	// Create buffered reader for responses
	reader := bufio.NewReader(stdoutReader)

	// Initialize first
	err := sendJSONRPC(stdinWriter, "initialize", 1, map[string]any{
		"protocolVersion": "2024-11-05",
	})
	if err != nil {
		t.Fatalf("Failed to send initialize: %v", err)
	}
	_, err = readJSONRPC(reader)
	if err != nil {
		t.Fatalf("Failed to read initialize response: %v", err)
	}

	// Call wingmate_status tool
	err = sendJSONRPC(stdinWriter, "tools/call", 2, map[string]any{
		"name":      "wingmate_status",
		"arguments": map[string]any{},
	})
	if err != nil {
		t.Fatalf("Failed to send tools/call: %v", err)
	}

	// Read response
	resp, err := readJSONRPC(reader)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify response
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object, got %T", resp["result"])
	}

	content, ok := result["content"].([]any)
	if !ok {
		t.Fatalf("Expected content array, got %T", result["content"])
	}

	if len(content) == 0 {
		t.Fatal("Expected at least one content item")
	}

	// Verify content contains status info
	textContent := content[0].(map[string]any)
	if textContent["type"] != "text" {
		t.Errorf("Expected text content type, got %v", textContent["type"])
	}

	text := textContent["text"].(string)
	if !strings.Contains(text, "uptime_ms") && !strings.Contains(text, "cli_available") {
		t.Errorf("Expected status response to contain status info, got: %s", text)
	}

	// Cleanup
	cancel()
	stdinWriter.Close()
	stdinReader.Close()

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
	}
}

func TestIntegration_MCPServerToolsCallChat(t *testing.T) {
	// Create io.Pipe for stdin/stdout simulation
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// Create transport
	transport := mcp.NewTransport(stdinReader, stdoutWriter)

	// Create test logger
	logger := &testLogger{}

	// Create server and register tools
	server := mcp.NewServerWithLogger(
		mcp.ServerConfig{
			Name:    "test-server",
			Version: "1.0.0",
		},
		transport,
		logger,
	)

	// Register default tools
	for _, tool := range mcp.DefaultTools() {
		server.RegisterTool(tool)
	}

	// Create mock executor that returns a predictable response
	mockExecutor := &mockLLMExecutor{
		response:    "Hello from mock LLM!",
		sessionID:   "test-session",
		isInstalled: true,
	}

	// Register chat handler with mock executor
	server.RegisterHandler(mcp.ToolNameChat, mcp.NewChatHandler(mockExecutor, nil, logger))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(ctx)
	}()

	// Create buffered reader for responses
	reader := bufio.NewReader(stdoutReader)

	// Initialize first
	err := sendJSONRPC(stdinWriter, "initialize", 1, map[string]any{
		"protocolVersion": "2024-11-05",
	})
	if err != nil {
		t.Fatalf("Failed to send initialize: %v", err)
	}
	_, err = readJSONRPC(reader)
	if err != nil {
		t.Fatalf("Failed to read initialize response: %v", err)
	}

	// Call wingmate_chat tool
	err = sendJSONRPC(stdinWriter, "tools/call", 2, map[string]any{
		"name": "wingmate_chat",
		"arguments": map[string]any{
			"prompt": "Hello, Wingmate!",
		},
	})
	if err != nil {
		t.Fatalf("Failed to send tools/call: %v", err)
	}

	// Read response
	resp, err := readJSONRPC(reader)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify response
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object, got %T", resp["result"])
	}

	content, ok := result["content"].([]any)
	if !ok {
		t.Fatalf("Expected content array, got %T", result["content"])
	}

	if len(content) == 0 {
		t.Fatal("Expected at least one content item")
	}

	// Verify content contains mock response
	textContent := content[0].(map[string]any)
	text := textContent["text"].(string)
	if !strings.Contains(text, "Hello from mock LLM!") {
		t.Errorf("Expected response to contain mock LLM response, got: %s", text)
	}

	// Cleanup
	cancel()
	stdinWriter.Close()
	stdinReader.Close()

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
	}
}

func TestIntegration_MCPServerGracefulShutdown(t *testing.T) {
	// Create io.Pipe for stdin/stdout simulation
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// Create transport
	transport := mcp.NewTransport(stdinReader, stdoutWriter)

	// Create server
	server := mcp.NewServer(
		mcp.ServerConfig{
			Name:    "test-server",
			Version: "1.0.0",
		},
		transport,
	)

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Run server in goroutine
	serverErr := make(chan error, 1)
	serverDone := make(chan struct{})
	go func() {
		serverErr <- server.Run(ctx)
		close(serverDone)
	}()

	// Create buffered reader
	reader := bufio.NewReader(stdoutReader)

	// Initialize
	err := sendJSONRPC(stdinWriter, "initialize", 1, map[string]any{
		"protocolVersion": "2024-11-05",
	})
	if err != nil {
		t.Fatalf("Failed to send initialize: %v", err)
	}
	_, err = readJSONRPC(reader)
	if err != nil {
		t.Fatalf("Failed to read initialize response: %v", err)
	}

	// Verify server is ready
	if server.State() != mcp.StateReady {
		t.Errorf("Expected server state Ready, got %v", server.State())
	}

	// Cancel context to trigger shutdown
	cancel()

	// Close pipes to help server detect shutdown
	stdinWriter.Close()
	stdinReader.Close()

	// Wait for server to stop
	select {
	case err := <-serverErr:
		if err != nil && err != context.Canceled {
			t.Errorf("Unexpected server error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Server did not stop gracefully in time")
	}

	// Verify server stopped
	select {
	case <-serverDone:
		// Good - server stopped
	case <-time.After(1 * time.Second):
		t.Error("Server goroutine did not exit")
	}
}

func TestIntegration_MCPServerRejectsBeforeInitialize(t *testing.T) {
	// Create io.Pipe for stdin/stdout simulation
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// Create transport
	transport := mcp.NewTransport(stdinReader, stdoutWriter)

	// Create server
	server := mcp.NewServer(
		mcp.ServerConfig{
			Name:    "test-server",
			Version: "1.0.0",
		},
		transport,
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(ctx)
	}()

	// Create buffered reader
	reader := bufio.NewReader(stdoutReader)

	// Send tools/list WITHOUT initializing first - should be rejected
	err := sendJSONRPC(stdinWriter, "tools/list", 1, nil)
	if err != nil {
		t.Fatalf("Failed to send tools/list: %v", err)
	}

	// Read response - should be an error
	resp, err := readJSONRPC(reader)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify error response
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("Expected error response, got: %v", resp)
	}

	errMsg := errObj["message"].(string)
	if !strings.Contains(errMsg, "not initialized") {
		t.Errorf("Expected 'not initialized' error, got: %s", errMsg)
	}

	// Cleanup
	cancel()
	stdinWriter.Close()
	stdinReader.Close()

	select {
	case <-serverErr:
	case <-time.After(2 * time.Second):
	}
}
