package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/wingmate/wingmate/pkg/types"
)

// A2AServer implements the Server interface for A2A protocol.
// It serves JSON-RPC requests at POST / and Agent Card at GET /.well-known/agent.json.
type A2AServer struct {
	handler   MessageHandler
	agentCard *types.AgentCard

	server   *http.Server
	listener net.Listener
	ready    chan struct{}
	addr     string

	mu sync.RWMutex
	wg sync.WaitGroup // Tracks in-flight requests
}

// NewA2AServer creates a new A2A server.
func NewA2AServer() *A2AServer {
	return &A2AServer{
		ready: make(chan struct{}),
	}
}

// SetHandler registers a MessageHandler for processing incoming messages.
func (s *A2AServer) SetHandler(handler MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
}

// SetAgentCard sets the Agent Card to serve at /.well-known/agent.json.
func (s *A2AServer) SetAgentCard(card *types.AgentCard) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentCard = card
}

// ListenAndServe starts the HTTP server on the given address.
// Blocks until the server is shutdown or encounters a fatal error.
func (s *A2AServer) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.listener = ln
	s.addr = ln.Addr().String()
	s.server = &http.Server{
		Handler: s,
	}
	s.mu.Unlock()

	// Signal ready
	close(s.ready)

	// Run server
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Serve(ln)
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// Addr returns the address the server is listening on.
func (s *A2AServer) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

// Shutdown gracefully shuts down the server.
func (s *A2AServer) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	server := s.server
	s.mu.RUnlock()

	if server == nil {
		return nil
	}

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	// Wait for in-flight requests
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Ready returns a channel that is closed when the server is ready.
func (s *A2AServer) Ready() <-chan struct{} {
	return s.ready
}

// ServeHTTP implements http.Handler.
func (s *A2AServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.wg.Add(1)
	defer s.wg.Done()

	// Route based on path
	switch r.URL.Path {
	case types.AgentCardWellKnownPath:
		s.handleAgentCard(w, r)
	case "/", "":
		s.handleJSONRPC(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleAgentCard serves the Agent Card.
func (s *A2AServer) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	card := s.agentCard
	s.mu.RUnlock()

	if card == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

// handleJSONRPC processes JSON-RPC requests.
func (s *A2AServer) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeResponse(w, NewInternalErrorResponse(nil, "failed to read request"))
		return
	}

	// Parse JSON-RPC request
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeResponse(w, NewParseErrorResponse(nil))
		return
	}

	// Validate request
	if req.Method == "" {
		s.writeResponse(w, NewInvalidRequestResponse(req.ID, "missing method"))
		return
	}

	// Get handler
	s.mu.RLock()
	handler := s.handler
	s.mu.RUnlock()

	if handler == nil {
		s.writeResponse(w, NewInternalErrorResponse(req.ID, "no handler configured"))
		return
	}

	// Parse message from params
	var msg types.Message
	if len(req.Params) > 0 {
		// Try to extract message from params
		var params struct {
			Message types.Message `json:"message"`
		}
		if err := json.Unmarshal(req.Params, &params); err == nil {
			msg = params.Message
		}
	}

	// Handle message
	resp, err := handler.HandleMessage(r.Context(), &msg)
	if err != nil {
		// Check if error is an ErrorObject
		var errObj *ErrorObject
		if errors.As(err, &errObj) {
			s.writeResponse(w, NewErrorResponse(req.ID, errObj.Code, errObj.Message, errObj.Data))
			return
		}
		// Generic internal error
		s.writeResponse(w, NewInternalErrorResponse(req.ID, err.Error()))
		return
	}

	// Success response
	if resp == nil {
		resp = &types.A2AResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{}`),
		}
	}

	// Convert types.A2AResponse to protocol.Response
	protoResp := &Response{
		JSONRPC: resp.JSONRPC,
	}

	// Marshal ID to json.RawMessage
	if resp.ID != nil {
		if idBytes, err := json.Marshal(resp.ID); err == nil {
			protoResp.ID = idBytes
		}
	}

	// Marshal Result to json.RawMessage
	if resp.Result != nil {
		if resultBytes, err := json.Marshal(resp.Result); err == nil {
			protoResp.Result = resultBytes
		}
	}

	// Convert error if present
	if resp.Error != nil {
		protoResp.Error = &ErrorObject{
			Code:    resp.Error.Code,
			Message: resp.Error.Message,
		}
		// Marshal error data if present
		if resp.Error.Data != nil {
			if dataBytes, err := json.Marshal(resp.Error.Data); err == nil {
				protoResp.Error.Data = dataBytes
			}
		}
	}

	s.writeResponse(w, protoResp)
}

// writeResponse writes a JSON-RPC response.
func (s *A2AServer) writeResponse(w http.ResponseWriter, resp *Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Ensure A2AServer implements required interfaces.
var (
	_ Server        = (*A2AServer)(nil)
	_ ReadyNotifier = (*A2AServer)(nil)
	_ http.Handler  = (*A2AServer)(nil)
)
