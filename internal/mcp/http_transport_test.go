package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

/*
Test Doc:
- Why: MCP HTTP transport needs to serve JSON-RPC requests at /mcp endpoint (Critical Discovery 01)
- Contract: HTTPHandler implements http.Handler, accepts POST with JSON-RPC, returns JSON-RPC response
- Usage Notes: Use NewHTTPHandler(server, sessionMgr) to create handler; register on router
- Quality Contribution: Enables Claude Code connection via HTTP instead of stdio
- Worked Example: POST /mcp with initialize request → returns initialize response with server info
*/

// TestHTTPHandler_InitializeRequest tests the MCP initialize handshake via HTTP.
func TestHTTPHandler_InitializeRequest(t *testing.T) {
	/*
	Test Doc:
	- Why: AC-1 requires HTTP endpoint available with valid MCP initialize response
	- Contract: POST /mcp with initialize request returns server info and capabilities
	- Usage Notes: First request after connection must be initialize
	- Quality Contribution: Validates Claude Code can establish MCP session
	- Worked Example: POST {method:"initialize"} → {result:{protocolVersion, serverInfo, capabilities}}
	*/

	handler := createTestHTTPHandler(t)

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Parse response
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify JSON-RPC structure
	if resp["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %v", resp["jsonrpc"])
	}
	if resp["id"] != float64(1) { // JSON numbers are float64
		t.Errorf("expected id 1, got %v", resp["id"])
	}

	// Verify result contains required fields
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %T", resp["result"])
	}

	if _, ok := result["protocolVersion"]; !ok {
		t.Error("result missing protocolVersion")
	}
	if _, ok := result["serverInfo"]; !ok {
		t.Error("result missing serverInfo")
	}
	if _, ok := result["capabilities"]; !ok {
		t.Error("result missing capabilities")
	}
}

// TestHTTPHandler_ToolsList tests the tools/list request via HTTP.
func TestHTTPHandler_ToolsList(t *testing.T) {
	/*
	Test Doc:
	- Why: AC-2 requires Claude Code can list available tools
	- Contract: POST /mcp with tools/list returns array of tool definitions
	- Usage Notes: Server must be initialized first
	- Quality Contribution: Validates tool discovery works over HTTP
	- Worked Example: POST {method:"tools/list"} → {result:{tools:[...]}}
	*/

	handler := createTestHTTPHandler(t)

	// Initialize first
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`
	doRequest(t, handler, initReq)

	// Now request tools list
	reqBody := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %T", resp["result"])
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("expected tools array, got %T", result["tools"])
	}

	// We should have at least one tool registered by createTestHTTPHandler
	if len(tools) == 0 {
		t.Error("expected at least one tool in tools/list response")
	}
}

// TestHTTPHandler_ToolsCall tests tool invocation via HTTP.
func TestHTTPHandler_ToolsCall(t *testing.T) {
	/*
	Test Doc:
	- Why: AC-3 requires tool invocation works via HTTP
	- Contract: POST /mcp with tools/call returns tool result
	- Usage Notes: Tool must be registered; server must be initialized
	- Quality Contribution: Validates full MCP workflow over HTTP
	- Worked Example: POST {method:"tools/call",params:{name:"test_tool"}} → {result:{content:[...]}}
	*/

	handler := createTestHTTPHandler(t)

	// Initialize
	doRequest(t, handler, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)

	// Call the test tool
	reqBody := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"test_tool","arguments":{}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %T", resp["result"])
	}

	content, ok := result["content"].([]any)
	if !ok {
		t.Fatalf("expected content array, got %T", result["content"])
	}

	if len(content) == 0 {
		t.Error("expected non-empty content in tool result")
	}
}

// TestHTTPHandler_GETMethodRejected tests that GET requests return 405.
func TestHTTPHandler_GETMethodRejected(t *testing.T) {
	/*
	Test Doc:
	- Why: MCP Streamable HTTP spec requires POST for requests (GET is for SSE, not implemented)
	- Contract: GET /mcp returns 405 Method Not Allowed
	- Usage Notes: Only POST is supported in initial implementation
	- Quality Contribution: Prevents accidental misuse and follows spec
	- Worked Example: GET /mcp → 405 Method Not Allowed
	*/

	handler := createTestHTTPHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

// TestHTTPHandler_InvalidJSON tests error handling for malformed JSON.
func TestHTTPHandler_InvalidJSON(t *testing.T) {
	/*
	Test Doc:
	- Why: AC-10 requires graceful error handling for malformed requests
	- Contract: Invalid JSON returns 400 Bad Request with JSON-RPC error
	- Usage Notes: Error response is JSON-RPC compliant
	- Quality Contribution: Server doesn't crash on bad input
	- Worked Example: POST "not json" → 400 with parse error
	*/

	handler := createTestHTTPHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("not valid json{"))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	// Should still return JSON-RPC error
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("error response should be valid JSON: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}

// TestHTTPHandler_EmptyBody tests error handling for empty request body.
func TestHTTPHandler_EmptyBody(t *testing.T) {
	/*
	Test Doc:
	- Why: Edge case - client sends POST with no body
	- Contract: Empty body returns 400 Bad Request
	- Usage Notes: A valid MCP request always has a JSON body
	- Quality Contribution: Clear error for misconfigured clients
	- Worked Example: POST /mcp with empty body → 400
	*/

	handler := createTestHTTPHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// TestHTTPHandler_SessionHeader tests Mcp-Session-Id header handling.
func TestHTTPHandler_SessionHeader(t *testing.T) {
	/*
	Test Doc:
	- Why: AC-7 requires session continuity via session ID
	- Contract: Response includes Mcp-Session-Id header for session tracking
	- Usage Notes: Client should include this header in subsequent requests
	- Quality Contribution: Enables chat conversation continuity
	- Worked Example: Initialize response includes Mcp-Session-Id header
	*/

	handler := createTestHTTPHandler(t)

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	sessionID := rr.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Error("expected Mcp-Session-Id header in response")
	}
}

// TestHTTPHandler_SessionContinuity tests that session ID persists state.
func TestHTTPHandler_SessionContinuity(t *testing.T) {
	/*
	Test Doc:
	- Why: AC-7 requires conversation context maintained via session ID
	- Contract: Same session ID returns same session state across requests
	- Usage Notes: Session ID in request header links to server-side session
	- Quality Contribution: Enables multi-turn conversations
	- Worked Example: Request with session ID → accesses same session state
	*/

	handler := createTestHTTPHandler(t)

	// First request gets a session
	reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`
	req1 := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(reqBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.RemoteAddr = "127.0.0.1:54321"
	rr1 := httptest.NewRecorder()

	handler.ServeHTTP(rr1, req1)
	sessionID := rr1.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("expected Mcp-Session-Id in first response")
	}

	// Second request uses same session
	req2 := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Mcp-Session-Id", sessionID)
	req2.RemoteAddr = "127.0.0.1:54321"
	rr2 := httptest.NewRecorder()

	handler.ServeHTTP(rr2, req2)

	// Should succeed and return same session ID
	if rr2.Code != http.StatusOK {
		t.Errorf("expected status 200 with existing session, got %d", rr2.Code)
	}

	returnedSession := rr2.Header().Get("Mcp-Session-Id")
	if returnedSession != sessionID {
		t.Errorf("expected same session ID %q, got %q", sessionID, returnedSession)
	}
}

// TestHTTPHandler_ContentTypeHeader tests that response has correct Content-Type.
func TestHTTPHandler_ContentTypeHeader(t *testing.T) {
	/*
	Test Doc:
	- Why: Proper HTTP semantics require Content-Type header
	- Contract: Response Content-Type is application/json
	- Usage Notes: Clients may validate Content-Type before parsing
	- Quality Contribution: HTTP compliance
	- Worked Example: Response has Content-Type: application/json
	*/

	handler := createTestHTTPHandler(t)

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// TestHTTPHandler_UnknownMethod tests error handling for unknown JSON-RPC methods.
func TestHTTPHandler_UnknownMethod(t *testing.T) {
	/*
	Test Doc:
	- Why: Clients may send unsupported methods
	- Contract: Unknown method returns JSON-RPC method not found error
	- Usage Notes: Error code should be -32601 per JSON-RPC spec
	- Quality Contribution: Clear error feedback for debugging
	- Worked Example: POST {method:"unknown"} → error code -32601
	*/

	handler := createTestHTTPHandler(t)

	// Initialize first
	doRequest(t, handler, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)

	// Call unknown method
	reqBody := `{"jsonrpc":"2.0","id":2,"method":"some/unknown/method"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should still be 200 OK (error is in JSON-RPC body)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for JSON-RPC error, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error in response")
	}

	code := errObj["code"].(float64)
	if code != -32601 {
		t.Errorf("expected error code -32601 (method not found), got %v", code)
	}
}

// TestHTTPHandler_ConcurrentRequests tests handling of simultaneous requests.
func TestHTTPHandler_ConcurrentRequests(t *testing.T) {
	/*
	Test Doc:
	- Why: AC-9 requires concurrent requests handled correctly
	- Contract: Multiple simultaneous requests don't interfere with each other
	- Usage Notes: Run with -race flag to detect data races
	- Quality Contribution: Multi-client support
	- Worked Example: 10 goroutines sending requests → all get correct responses
	*/

	handler := createTestHTTPHandler(t)

	// Initialize the handler (shared across requests)
	doRequest(t, handler, `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)

	const numRequests = 10
	results := make(chan int, numRequests)

	for i := 1; i <= numRequests; i++ {
		go func(reqID int) {
			reqBody := `{"jsonrpc":"2.0","id":` + strings.Repeat("1", reqID) + `,"method":"tools/list"}`
			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "127.0.0.1:54321"
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			results <- rr.Code
		}(i)
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		code := <-results
		if code != http.StatusOK {
			t.Errorf("concurrent request failed with status %d", code)
		}
	}
}

// TestHTTPHandler_ImplementsHTTPHandler tests interface compliance.
func TestHTTPHandler_ImplementsHTTPHandler(t *testing.T) {
	/*
	Test Doc:
	- Why: HTTPHandler must implement http.Handler for use with net/http
	- Contract: Type assertion to http.Handler succeeds
	- Usage Notes: Allows use with http.Handle, mux.Handle, etc.
	- Quality Contribution: Standard Go HTTP integration
	- Worked Example: var _ http.Handler = (*HTTPHandler)(nil) compiles
	*/

	var _ http.Handler = (*HTTPHandler)(nil)
}

// === Test Helpers ===

// createTestHTTPHandler creates an HTTPHandler with a test server and session manager.
func createTestHTTPHandler(t *testing.T) *HTTPHandler {
	t.Helper()

	sessionMgr := NewMCPSessionManager(30*time.Minute, 5*time.Minute)
	t.Cleanup(sessionMgr.Stop)

	handler := NewHTTPHandler(ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}, sessionMgr)

	// Register a test tool
	handler.RegisterTool(&ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	})
	handler.RegisterHandler("test_tool", func(ctx context.Context, req *ToolRequest) (*ToolResult, error) {
		return &ToolResult{
			Content: []Content{&TextContent{Text: "test result"}},
			IsError: false,
		}, nil
	})

	return handler
}

// doRequest is a helper to send a request and ignore the response.
func doRequest(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// Ensure io import is used (for any future read operations)
var _ = io.EOF
var _ = bytes.Buffer{}
