package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

// HTTPHandler implements http.Handler for MCP over HTTP transport.
// It processes JSON-RPC requests at the /mcp endpoint, replacing the
// stdio-based transport for integration with the agent's HTTP server.
//
// HTTPHandler manages server state, session management, and tool registration
// internally, delegating JSON-RPC message processing to the same handlers
// used by the stdio transport.
type HTTPHandler struct {
	config     ServerConfig
	sessionMgr *MCPSessionManager
	state      ServerState
	mu         sync.RWMutex

	tools    []*ToolDefinition
	handlers map[string]ToolHandler
}

// NewHTTPHandler creates a new HTTP handler for MCP requests.
//
// Parameters:
//   - config: Server configuration (name, version)
//   - sessionMgr: Session manager for conversation continuity
//
// The handler starts in uninitialized state and transitions to ready
// after receiving an initialize request.
func NewHTTPHandler(config ServerConfig, sessionMgr *MCPSessionManager) *HTTPHandler {
	return &HTTPHandler{
		config:     config,
		sessionMgr: sessionMgr,
		state:      StateUninitialized,
		handlers:   make(map[string]ToolHandler),
	}
}

// RegisterTool adds a tool definition to the handler.
// Registered tools will be returned in tools/list responses.
func (h *HTTPHandler) RegisterTool(tool *ToolDefinition) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tools = append(h.tools, tool)
}

// RegisterHandler registers a handler function for a tool.
// The handler will be invoked when tools/call is received for this tool.
func (h *HTTPHandler) RegisterHandler(name string, handler ToolHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[name] = handler
}

// ServeHTTP implements http.Handler interface.
// It processes POST requests with JSON-RPC bodies and returns JSON-RPC responses.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeJSONRPCError(w, nil, -32700, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Check for empty body
	if len(body) == 0 {
		h.writeJSONRPCError(w, nil, -32700, "Empty request body", http.StatusBadRequest)
		return
	}

	// Parse JSON-RPC request
	var msg map[string]any
	if err := json.Unmarshal(body, &msg); err != nil {
		h.writeJSONRPCError(w, nil, -32700, "Parse error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get or create session
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		sessionID = h.sessionMgr.CreateSession()
	} else {
		// Verify session exists
		if _, exists := h.sessionMgr.GetSession(sessionID); !exists {
			// Session expired or invalid - create new one
			sessionID = h.sessionMgr.CreateSession()
		}
	}

	// Process the message
	ctx := r.Context()
	response, httpStatus := h.handleMessage(ctx, msg)

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Mcp-Session-Id", sessionID)
	w.WriteHeader(httpStatus)

	// Write response
	if response != nil {
		json.NewEncoder(w).Encode(response)
	}
}

// handleMessage processes a single JSON-RPC message and returns the response.
func (h *HTTPHandler) handleMessage(ctx context.Context, msg map[string]any) (map[string]any, int) {
	method, _ := msg["method"].(string)
	id := msg["id"]

	// Check if we need initialization first
	h.mu.RLock()
	state := h.state
	h.mu.RUnlock()

	if state == StateUninitialized && method != "initialize" {
		return h.errorResponse(id, -32600, "Server not initialized"), http.StatusOK
	}

	switch method {
	case "initialize":
		return h.handleInitialize(ctx, msg), http.StatusOK
	case "tools/list":
		return h.handleToolsList(ctx, msg), http.StatusOK
	case "tools/call":
		return h.handleToolsCall(ctx, msg), http.StatusOK
	default:
		// Unknown method - return method not found for requests with id
		if id != nil {
			return h.errorResponse(id, -32601, "Method not found"), http.StatusOK
		}
		// Notifications (no id) - just return empty success
		return map[string]any{"jsonrpc": "2.0"}, http.StatusOK
	}
}

// handleInitialize processes the MCP initialize request.
func (h *HTTPHandler) handleInitialize(ctx context.Context, msg map[string]any) map[string]any {
	h.mu.Lock()
	h.state = StateReady
	h.mu.Unlock()

	id := msg["id"]

	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]any{
				"name":    h.config.Name,
				"version": h.config.Version,
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
		},
	}
}

// handleToolsList processes the tools/list request.
func (h *HTTPHandler) handleToolsList(ctx context.Context, msg map[string]any) map[string]any {
	id := msg["id"]

	h.mu.RLock()
	toolsArray := make([]any, 0, len(h.tools))
	for _, tool := range h.tools {
		toolsArray = append(toolsArray, map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}
	h.mu.RUnlock()

	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"tools": toolsArray,
		},
	}
}

// handleToolsCall processes the tools/call request.
func (h *HTTPHandler) handleToolsCall(ctx context.Context, msg map[string]any) map[string]any {
	id := msg["id"]

	// Extract params
	params, ok := msg["params"].(map[string]any)
	if !ok {
		return h.errorResponse(id, -32602, "Invalid params: expected object")
	}

	// Extract tool name
	toolName, ok := params["name"].(string)
	if !ok || toolName == "" {
		return h.errorResponse(id, -32602, "Invalid params: missing tool name")
	}

	// Extract arguments (optional)
	arguments, _ := params["arguments"].(map[string]any)
	if arguments == nil {
		arguments = make(map[string]any)
	}

	// Look up handler
	h.mu.RLock()
	handler := h.handlers[toolName]
	h.mu.RUnlock()

	if handler == nil {
		return h.errorResponse(id, -32601, "Method not found: unknown tool "+toolName)
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
		return h.toolResultResponse(id, &ToolResult{
			Content: []Content{&TextContent{Text: err.Error()}},
			IsError: true,
		})
	}

	return h.toolResultResponse(id, result)
}

// toolResultResponse builds a JSON-RPC response for a tool result.
func (h *HTTPHandler) toolResultResponse(id any, result *ToolResult) map[string]any {
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

	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"content": contentArray,
			"isError": result.IsError,
		},
	}
}

// errorResponse builds a JSON-RPC error response.
func (h *HTTPHandler) errorResponse(id any, code int, message string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

// writeJSONRPCError writes a JSON-RPC error response with the given HTTP status.
func (h *HTTPHandler) writeJSONRPCError(w http.ResponseWriter, id any, code int, message string, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(h.errorResponse(id, code, message))
}
