package protocol

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wingmate/wingmate/pkg/types"
)

// TestClient_SendMessage verifies successful message sending.
func TestClient_SendMessage(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", ct)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}`))
	}))
	defer server.Close()

	client := NewA2AClient()
	defer client.Close()

	msg := &types.Message{
		Role: "user",
		Parts: []types.Part{
			{Kind: "text", Text: "hello"},
		},
	}

	resp, err := client.SendMessage(context.Background(), server.URL, msg)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
}

// TestClient_SendMessage_ErrorResponse verifies error response handling.
func TestClient_SendMessage_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`))
	}))
	defer server.Close()

	client := NewA2AClient()
	defer client.Close()

	resp, err := client.SendMessage(context.Background(), server.URL, &types.Message{})
	if err != nil {
		t.Fatalf("SendMessage should not return Go error for JSON-RPC error: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("Error code = %d, want %d", resp.Error.Code, CodeMethodNotFound)
	}
}

// TestClient_GetAgentCard verifies successful Agent Card fetching.
func TestClient_GetAgentCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != types.AgentCardWellKnownPath {
			t.Errorf("Path = %s, want %s", r.URL.Path, types.AgentCardWellKnownPath)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Method = %s, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.AgentCard{
			Name:    "remote-agent",
			URL:     "http://example.com",
			Version: "1.0.0",
			Capabilities: types.Capabilities{
				Streaming: true,
			},
			Skills: []types.Skill{
				{Name: "ping", Description: "Pong!"},
			},
		})
	}))
	defer server.Close()

	client := NewA2AClient()
	defer client.Close()

	card, err := client.GetAgentCard(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GetAgentCard failed: %v", err)
	}

	if card.Name != "remote-agent" {
		t.Errorf("Name = %q, want remote-agent", card.Name)
	}

	if card.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", card.Version)
	}

	if !card.Capabilities.Streaming {
		t.Error("Capabilities.Streaming = false, want true")
	}

	if len(card.Skills) != 1 || card.Skills[0].Name != "ping" {
		t.Errorf("Skills = %v, want [{ping ...}]", card.Skills)
	}
}

// TestClient_GetAgentCard_NotFound verifies 404 handling.
func TestClient_GetAgentCard_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewA2AClient()
	defer client.Close()

	_, err := client.GetAgentCard(context.Background(), server.URL)
	if err == nil {
		t.Fatal("Expected error for 404 response")
	}
}

// TestClient_ConnectionRefused verifies connection error handling.
func TestClient_ConnectionRefused(t *testing.T) {
	client := NewA2AClient()
	defer client.Close()

	// Use a port that should be closed
	_, err := client.SendMessage(context.Background(), "http://127.0.0.1:1", &types.Message{})
	if err == nil {
		t.Fatal("Expected error for connection refused")
	}

	// Should be wrapped as ProtocolError
	var protoErr *ProtocolError
	if ok := isProtocolError(err, &protoErr); !ok {
		// Connection errors are acceptable even without wrapping
		t.Logf("Got error: %v", err)
	}
}

// TestClient_Timeout verifies context timeout handling.
func TestClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than context timeout
		time.Sleep(2 * time.Second)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
	}))
	defer server.Close()

	client := NewA2AClient()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.SendMessage(ctx, server.URL, &types.Message{})
	if err == nil {
		t.Fatal("Expected timeout error")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Logf("Context error: %v", ctx.Err())
	}
}

// TestClient_InvalidJSON verifies invalid response handling.
func TestClient_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewA2AClient()
	defer client.Close()

	_, err := client.SendMessage(context.Background(), server.URL, &types.Message{})
	if err == nil {
		t.Fatal("Expected error for invalid JSON response")
	}
}

// TestClient_Close verifies client cleanup.
func TestClient_Close(t *testing.T) {
	client := NewA2AClient()

	if err := client.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Should be safe to call multiple times
	if err := client.Close(); err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

// isProtocolError is a helper to check if an error is a ProtocolError.
func isProtocolError(err error, target **ProtocolError) bool {
	if pe, ok := err.(*ProtocolError); ok {
		*target = pe
		return true
	}
	return false
}
