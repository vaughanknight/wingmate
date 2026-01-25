// Package mcp provides tool handlers for the MCP server.
//
// Why: Tool handlers implement the business logic for each MCP tool,
// bridging the MCP protocol to Wingmate's LLM capabilities.
//
// Contract: Handlers must validate inputs, invoke the executor, handle errors,
// and log to Flight Log. They return ToolResult, never Go errors for expected failures.
//
// Usage Notes: Create handlers via New*Handler functions. Pass an LLMExecutor
// for chat operations, and optionally a Logger for Flight Log integration.
//
// Quality Contribution: Isolates tool-specific logic from MCP protocol handling,
// enabling focused testing and clean separation of concerns.
package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wingmate/wingmate/internal/llm"
)

// SessionManager manages conversation sessions for continuity.
type SessionManager interface {
	// GetSession retrieves session data for a given ID.
	GetSession(id string) (map[string]any, bool)
	// SetSession stores session data.
	SetSession(id string, data map[string]any)
}

// PeerProvider supplies peer information for the discover tool.
// This interface enables dependency injection from the agent package
// without creating circular imports (agent imports mcp, so mcp cannot import agent).
type PeerProvider interface {
	// GetPeers returns URLs of known peer agents.
	GetPeers() []string
}

// =============================================================================
// wingmate_chat Handler
// =============================================================================

// NewChatHandler creates a handler for the wingmate_chat tool.
// It invokes the LLM executor to process prompts and return responses.
//
// Parameters:
//   - executor: The LLM executor (e.g., Claude CLI) to invoke
//   - sessions: Optional session manager for conversation continuity
//   - logger: Optional Flight Log logger
func NewChatHandler(executor llm.LLMExecutor, sessions SessionManager, logger Logger) ToolHandler {
	return func(ctx context.Context, req *ToolRequest) (*ToolResult, error) {
		// Log invocation
		if logger != nil {
			_ = logger.Record(map[string]any{
				"tool":      ToolNameChat,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"event":     "invoked",
			})
		}

		// Check if CLI is available
		if !executor.IsInstalled() {
			return &ToolResult{
				Content: []Content{&TextContent{Text: "Claude CLI is not installed or not in PATH"}},
				IsError: true,
			}, nil
		}

		// Extract and validate prompt
		prompt, ok := req.Arguments["prompt"].(string)
		if !ok || prompt == "" {
			return &ToolResult{
				Content: []Content{&TextContent{Text: "missing required argument: prompt"}},
				IsError: true,
			}, nil
		}

		// Extract optional session_id
		sessionID, _ := req.Arguments["session_id"].(string)

		// Execute the prompt
		resp, err := executor.Execute(ctx, prompt, sessionID)
		if err != nil {
			return &ToolResult{
				Content: []Content{&TextContent{Text: err.Error()}},
				IsError: true,
			}, nil
		}

		// Return successful result
		return &ToolResult{
			Content: []Content{&TextContent{Text: resp.Result}},
			IsError: false,
		}, nil
	}
}

// =============================================================================
// wingmate_status Handler
// =============================================================================

// StatusInfo contains server health information.
type StatusInfo struct {
	UptimeMS       int64  `json:"uptime_ms"`
	SessionsActive int    `json:"sessions_active"`
	CLIAvailable   bool   `json:"cli_available"`
	ServerName     string `json:"server_name,omitempty"`
	ServerVersion  string `json:"server_version,omitempty"`
}

// NewStatusHandler creates a handler for the wingmate_status tool.
// It returns health information about the Wingmate server.
//
// Parameters:
//   - executor: The LLM executor to check CLI availability
//   - startTime: The server start time for uptime calculation
func NewStatusHandler(executor llm.LLMExecutor, startTime time.Time) ToolHandler {
	return func(ctx context.Context, req *ToolRequest) (*ToolResult, error) {
		status := StatusInfo{
			UptimeMS:       time.Since(startTime).Milliseconds(),
			SessionsActive: 0, // Will be populated when session manager is integrated
			CLIAvailable:   executor.IsInstalled(),
		}

		// Serialize to JSON
		data, err := json.Marshal(status)
		if err != nil {
			return &ToolResult{
				Content: []Content{&TextContent{Text: "failed to serialize status"}},
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Content: []Content{&TextContent{Text: string(data)}},
			IsError: false,
		}, nil
	}
}

// =============================================================================
// wingmate_discover Handler
// =============================================================================

// DiscoverInfo contains peer discovery information.
type DiscoverInfo struct {
	AgentID string   `json:"agent_id"`
	Peers   []string `json:"peers"`
}

// NewDiscoverHandler creates a handler for the wingmate_discover tool.
// It returns information about peer agents and this agent's identity.
//
// Parameters:
//   - agentID: This agent's unique identifier
//   - peers: PeerProvider for retrieving known peer URLs (nil returns empty list)
func NewDiscoverHandler(agentID string, peers PeerProvider) ToolHandler {
	return func(ctx context.Context, req *ToolRequest) (*ToolResult, error) {
		// Get peers from provider (nil-safe)
		var peerList []string
		if peers != nil {
			peerList = peers.GetPeers()
		}
		if peerList == nil {
			peerList = []string{}
		}

		info := DiscoverInfo{
			AgentID: agentID,
			Peers:   peerList,
		}

		// Serialize to JSON
		data, err := json.Marshal(info)
		if err != nil {
			return &ToolResult{
				Content: []Content{&TextContent{Text: "failed to serialize discover info"}},
				IsError: true,
			}, nil
		}

		return &ToolResult{
			Content: []Content{&TextContent{Text: string(data)}},
			IsError: false,
		}, nil
	}
}
