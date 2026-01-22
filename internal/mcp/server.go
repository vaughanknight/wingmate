package mcp

import (
	"context"
	"errors"
	"io"
	"sync"
)

// ServerState represents the current state of the MCP server.
type ServerState int

const (
	// StateUninitialized indicates the server has not yet received initialize request.
	StateUninitialized ServerState = iota
	// StateInitializing indicates the server is processing the initialize request.
	StateInitializing
	// StateReady indicates the server has been initialized and is ready for requests.
	StateReady
	// StateShuttingDown indicates the server is shutting down.
	StateShuttingDown
	// StateStopped indicates the server has stopped.
	StateStopped
)

// ServerConfig holds configuration for creating an MCP server.
type ServerConfig struct {
	// Name is the server name reported in initialize response.
	Name string
	// Version is the server version reported in initialize response.
	Version string
}

// Logger defines the interface for Flight Log recording.
// This allows the server to log lifecycle events per Constitution P1.
type Logger interface {
	Record(entry any) error
	Close() error
}

// Server implements an MCP server that communicates over stdio.
type Server struct {
	config    ServerConfig
	transport *Transport
	logger    Logger
	state     ServerState
	mu        sync.RWMutex
	cancel    context.CancelFunc
	done      chan struct{}
	tools     []*ToolDefinition       // Registered MCP tools
	handlers  map[string]ToolHandler  // Tool handlers keyed by tool name
}

// NewServer creates a new MCP server with the given configuration and transport.
// This creates a server without logging (use NewServerWithLogger for observability).
func NewServer(config ServerConfig, transport *Transport) *Server {
	return &Server{
		config:    config,
		transport: transport,
		state:     StateStopped, // Servers start in stopped state
	}
}

// NewServerWithLogger creates a new MCP server with Flight Log integration.
// Per Constitution P1, all server lifecycle events are logged.
func NewServerWithLogger(config ServerConfig, transport *Transport, logger Logger) *Server {
	return &Server{
		config:    config,
		transport: transport,
		logger:    logger,
		state:     StateStopped,
	}
}

// Start starts the server. Returns an error if the server is already running.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateStopped {
		return errors.New("server already started")
	}

	s.state = StateUninitialized
	s.done = make(chan struct{})
	return nil
}

// Stop stops the server gracefully. Safe to call multiple times.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateStopped {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}

	s.state = StateStopped
	return nil
}

// Run runs the server message loop until the context is cancelled or EOF is received.
// Returns nil on graceful shutdown, or an error if something went wrong.
func (s *Server) Run(ctx context.Context) error {
	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.state = StateUninitialized // Server starts uninitialized when run begins
	s.mu.Unlock()

	// Log shutdown on exit
	defer func() {
		s.logEvent("MCP server stopped", "shutdown")
		s.mu.Lock()
		s.state = StateStopped
		s.mu.Unlock()
	}()

	// Message processing loop
	errChan := make(chan error, 1)
	go func() {
		for {
			var msg map[string]any
			if err := s.transport.Read(&msg); err != nil {
				if err == io.EOF {
					errChan <- nil
					return
				}
				errChan <- err
				return
			}

			// Handle the message (will be expanded in T009)
			if err := s.handleMessage(ctx, msg); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Wait for context cancellation or message loop exit
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// JSON-RPC error codes
const (
	// ErrCodeInvalidRequest is the JSON-RPC error code for invalid requests.
	ErrCodeInvalidRequest = -32600
)

// handleMessage processes a single JSON-RPC message.
func (s *Server) handleMessage(ctx context.Context, msg map[string]any) error {
	method, _ := msg["method"].(string)
	id := msg["id"]

	// Check if we need initialization first
	s.mu.RLock()
	state := s.state
	s.mu.RUnlock()

	if state == StateUninitialized && method != "initialize" {
		// Reject non-initialize requests before initialization
		return s.sendError(id, ErrCodeInvalidRequest, "Server not initialized")
	}

	switch method {
	case "initialize":
		return s.handleInitialize(ctx, msg)
	case "tools/list":
		return s.handleToolsList(ctx, msg)
	case "tools/call":
		return s.handleToolsCall(ctx, msg)
	default:
		// Unknown method - for now just ignore notifications (no id)
		// and return method not found for requests
		if id != nil {
			return s.sendError(id, -32601, "Method not found")
		}
		return nil
	}
}

// handleInitialize processes the MCP initialize request.
func (s *Server) handleInitialize(ctx context.Context, msg map[string]any) error {
	s.mu.Lock()
	s.state = StateInitializing
	s.mu.Unlock()

	// Log server startup per Constitution P1
	s.logEvent("MCP server started", "startup")

	id := msg["id"]

	// Build the initialize response per MCP spec
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]any{
				"name":    s.config.Name,
				"version": s.config.Version,
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
		},
	}

	if err := s.transport.Write(response); err != nil {
		return err
	}

	s.mu.Lock()
	s.state = StateReady
	s.mu.Unlock()

	return nil
}

// handleToolsList processes the tools/list request.
// Returns all registered tools with their schemas.
func (s *Server) handleToolsList(ctx context.Context, msg map[string]any) error {
	id := msg["id"]

	// Build tools array from registered tools
	s.mu.RLock()
	toolsArray := make([]any, 0, len(s.tools))
	for _, tool := range s.tools {
		toolsArray = append(toolsArray, map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}
	s.mu.RUnlock()

	// Build the tools/list response
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"tools": toolsArray,
		},
	}

	return s.transport.Write(response)
}

// sendError sends a JSON-RPC error response.
func (s *Server) sendError(id any, code int, message string) error {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	return s.transport.Write(response)
}

// State returns the current server state.
func (s *Server) State() ServerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// RegisterTool adds a tool definition to the server.
// Registered tools will be returned in tools/list responses.
// Must be called before Run() to ensure tools are available.
func (s *Server) RegisterTool(tool *ToolDefinition) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, tool)
}

// RegisterHandler registers a handler function for a tool.
// The handler will be invoked when tools/call is received for this tool.
func (s *Server) RegisterHandler(name string, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handlers == nil {
		s.handlers = make(map[string]ToolHandler)
	}
	s.handlers[name] = handler
}

// handleToolsCall processes the tools/call request.
// Dispatches to the registered handler for the requested tool.
func (s *Server) handleToolsCall(ctx context.Context, msg map[string]any) error {
	id := msg["id"]

	// Extract params
	params, ok := msg["params"].(map[string]any)
	if !ok {
		return s.sendError(id, -32602, "Invalid params: expected object")
	}

	// Extract tool name
	toolName, ok := params["name"].(string)
	if !ok || toolName == "" {
		return s.sendError(id, -32602, "Invalid params: missing tool name")
	}

	// Extract arguments (optional)
	arguments, _ := params["arguments"].(map[string]any)
	if arguments == nil {
		arguments = make(map[string]any)
	}

	// Look up handler
	s.mu.RLock()
	handler := s.handlers[toolName]
	s.mu.RUnlock()

	if handler == nil {
		return s.sendError(id, -32601, "Method not found: unknown tool "+toolName)
	}

	// Build request and invoke handler
	req := &ToolRequest{
		Name:      toolName,
		Arguments: arguments,
		ctx:       ctx,
	}

	result, err := handler(ctx, req)
	if err != nil {
		// Handler returned an error - wrap it in a tool result
		return s.sendToolResult(id, &ToolResult{
			Content: []Content{&TextContent{Text: err.Error()}},
			IsError: true,
		})
	}

	return s.sendToolResult(id, result)
}

// sendToolResult sends a tools/call response with the given result.
func (s *Server) sendToolResult(id any, result *ToolResult) error {
	// Convert content to JSON-serializable format
	contentArray := make([]any, 0, len(result.Content))
	for _, c := range result.Content {
		switch content := c.(type) {
		case *TextContent:
			contentArray = append(contentArray, map[string]any{
				"type": "text",
				"text": content.Text,
			})
		}
	}

	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"content": contentArray,
			"isError": result.IsError,
		},
	}

	return s.transport.Write(response)
}

// logEvent logs a server lifecycle event to the Flight Log.
// This is a no-op if no logger is configured.
func (s *Server) logEvent(summary, event string) {
	if s.logger == nil {
		return
	}

	entry := map[string]any{
		"summary": summary,
		"event":   event,
		"server":  s.config.Name,
		"version": s.config.Version,
	}
	_ = s.logger.Record(entry)
}
