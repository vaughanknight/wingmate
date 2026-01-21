// Package protocol implements the A2A JSON-RPC 2.0 protocol layer.
//
// This package provides HTTP server and client implementations for agent-to-agent
// communication following the A2A specification. It handles:
//   - JSON-RPC 2.0 request/response serialization
//   - Agent Card serving at /.well-known/agent.json
//   - Connection pooling and timeout management
//   - Graceful shutdown with in-flight request handling
package protocol

import (
	"context"

	"github.com/wingmate/wingmate/pkg/types"
)

// MessageHandler processes incoming A2A messages.
// Implementations handle the actual message logic (e.g., ping/pong).
// The handler receives a parsed message and returns a response or error.
type MessageHandler interface {
	// HandleMessage processes an incoming A2A message and returns a response.
	// The handler should use the provided context for cancellation and timeouts.
	// Returns an A2AResponse on success, or an error that will be converted
	// to a JSON-RPC error response.
	HandleMessage(ctx context.Context, msg *types.Message) (*types.A2AResponse, error)
}

// Server defines the A2A HTTP server interface.
// The server listens for incoming JSON-RPC requests and serves Agent Cards.
type Server interface {
	// SetHandler registers a MessageHandler for processing incoming messages.
	// Must be called before ListenAndServe.
	SetHandler(handler MessageHandler)

	// SetAgentCard sets the Agent Card to serve at /.well-known/agent.json.
	// Must be called before ListenAndServe.
	SetAgentCard(card *types.AgentCard)

	// ListenAndServe starts the HTTP server on the given address.
	// Blocks until the server is shutdown or encounters a fatal error.
	// The context can be used to trigger shutdown.
	ListenAndServe(ctx context.Context, addr string) error

	// Addr returns the address the server is listening on.
	// Returns empty string if not yet listening.
	Addr() string

	// Shutdown gracefully shuts down the server.
	// Waits for in-flight requests to complete up to the context deadline.
	Shutdown(ctx context.Context) error
}

// Client defines the A2A HTTP client interface.
// The client makes outbound requests to other A2A agents.
type Client interface {
	// SendMessage sends a message to a remote agent and waits for a response.
	// The URL should be the root endpoint of the remote agent (e.g., "http://agent:9001").
	// Returns the parsed A2AResponse or an error.
	SendMessage(ctx context.Context, url string, msg *types.Message) (*types.A2AResponse, error)

	// GetAgentCard fetches and parses the Agent Card from a remote agent.
	// The URL should be the root endpoint (the method appends /.well-known/agent.json).
	GetAgentCard(ctx context.Context, url string) (*types.AgentCard, error)

	// Close releases any resources held by the client.
	Close() error
}

// ReadyNotifier is implemented by servers that support ready signaling.
// This is used for testing to avoid race conditions when starting servers.
type ReadyNotifier interface {
	// Ready returns a channel that is closed when the server is ready to accept connections.
	Ready() <-chan struct{}
}
