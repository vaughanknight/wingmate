package protocol

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wingmate/wingmate/pkg/types"
)

// mockHandler is a simple MessageHandler for testing.
type mockHandler struct {
	response *types.A2AResponse
	err      error
	called   bool
	lastMsg  *types.Message
}

func (m *mockHandler) HandleMessage(ctx context.Context, msg *types.Message) (*types.A2AResponse, error) {
	m.called = true
	m.lastMsg = msg
	return m.response, m.err
}

// TestServer_ValidJSONRPC verifies that a valid JSON-RPC POST request is handled.
func TestServer_ValidJSONRPC(t *testing.T) {
	handler := &mockHandler{
		response: &types.A2AResponse{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Result:  json.RawMessage(`{"status":"ok"}`),
		},
	}

	srv := NewA2AServer()
	srv.SetHandler(handler)
	srv.SetAgentCard(&types.AgentCard{
		Name:    "test-agent",
		URL:     "http://localhost:9000",
		Version: "0.1.0",
	})

	// Create test request
	reqBody := `{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"message":{"role":"user","parts":[{"kind":"text","text":"hello"}]}}}`

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	if !handler.called {
		t.Error("Handler was not called")
	}

	// Verify response is valid JSON-RPC
	body, _ := io.ReadAll(resp.Body)
	var jsonResp Response
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	if jsonResp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", jsonResp.JSONRPC)
	}
}

// TestServer_ParseError verifies -32700 for invalid JSON.
func TestServer_ParseError(t *testing.T) {
	srv := NewA2AServer()
	srv.SetHandler(&mockHandler{})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var jsonResp Response
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	if jsonResp.Error == nil {
		t.Fatal("Expected error response")
	}

	if jsonResp.Error.Code != CodeParseError {
		t.Errorf("Error code = %d, want %d (Parse error)", jsonResp.Error.Code, CodeParseError)
	}
}

// TestServer_InvalidRequest verifies -32600 for missing method.
func TestServer_InvalidRequest(t *testing.T) {
	srv := NewA2AServer()
	srv.SetHandler(&mockHandler{})

	// Valid JSON but missing method field
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc":"2.0","id":1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var jsonResp Response
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	if jsonResp.Error == nil {
		t.Fatal("Expected error response")
	}

	if jsonResp.Error.Code != CodeInvalidRequest {
		t.Errorf("Error code = %d, want %d (Invalid Request)", jsonResp.Error.Code, CodeInvalidRequest)
	}
}

// TestServer_MethodNotFound verifies -32601 for unknown method.
func TestServer_MethodNotFound(t *testing.T) {
	srv := NewA2AServer()
	// No handler set, or handler returns method not found
	srv.SetHandler(&mockHandler{
		err: &ErrorObject{Code: CodeMethodNotFound, Message: MsgMethodNotFound},
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"unknown/method"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var jsonResp Response
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	if jsonResp.Error == nil {
		t.Fatal("Expected error response")
	}

	if jsonResp.Error.Code != CodeMethodNotFound {
		t.Errorf("Error code = %d, want %d (Method not found)", jsonResp.Error.Code, CodeMethodNotFound)
	}
}

// TestServer_AgentCard verifies GET /.well-known/agent.json returns the Agent Card.
func TestServer_AgentCard(t *testing.T) {
	srv := NewA2AServer()
	srv.SetAgentCard(&types.AgentCard{
		Name:        "test-agent",
		Description: "A test agent",
		URL:         "http://localhost:9000",
		Version:     "0.1.0",
		Capabilities: types.Capabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		Skills: []types.Skill{
			{Name: "ping", Description: "Respond with pong"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, types.AgentCardWellKnownPath, nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	var card types.AgentCard
	if err := json.Unmarshal(body, &card); err != nil {
		t.Fatalf("Response is not valid AgentCard JSON: %v", err)
	}

	if card.Name != "test-agent" {
		t.Errorf("Name = %q, want test-agent", card.Name)
	}

	if card.URL != "http://localhost:9000" {
		t.Errorf("URL = %q, want http://localhost:9000", card.URL)
	}

	if len(card.Skills) != 1 || card.Skills[0].Name != "ping" {
		t.Errorf("Skills = %v, want [{ping ...}]", card.Skills)
	}
}

// TestServer_AgentCard_NotSet verifies 404 when Agent Card is not set.
func TestServer_AgentCard_NotSet(t *testing.T) {
	srv := NewA2AServer()
	// Don't set agent card

	req := httptest.NewRequest(http.MethodGet, types.AgentCardWellKnownPath, nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// TestServer_MethodNotAllowed verifies 405 for non-POST to root.
func TestServer_MethodNotAllowed(t *testing.T) {
	srv := NewA2AServer()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

// TestServer_ListenAndShutdown verifies server can start and shutdown gracefully.
func TestServer_ListenAndShutdown(t *testing.T) {
	srv := NewA2AServer()
	srv.SetHandler(&mockHandler{
		response: &types.A2AResponse{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Result:  json.RawMessage(`{}`),
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe(ctx, "127.0.0.1:0")
	}()

	// Wait for ready
	select {
	case <-srv.Ready():
		// Good
	case err := <-errCh:
		t.Fatalf("Server failed to start: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for server to be ready")
	}

	// Verify we can get the address
	addr := srv.Addr()
	if addr == "" {
		t.Error("Addr() returned empty string after server started")
	}

	// Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

// TestServer_Ready verifies ReadyNotifier interface.
func TestServer_Ready(t *testing.T) {
	srv := NewA2AServer()

	// Ready channel should exist but not be closed yet
	readyCh := srv.Ready()
	if readyCh == nil {
		t.Fatal("Ready() returned nil")
	}

	select {
	case <-readyCh:
		t.Error("Ready channel should not be closed before ListenAndServe")
	default:
		// Expected - not closed yet
	}
}
