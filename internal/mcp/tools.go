// Package mcp provides the MCP (Model Context Protocol) server implementation.
// This file defines tool schemas for Wingmate's MCP tools.
package mcp

// Tool names as constants for consistent reference.
const (
	// ToolNameChat is the name of the wingmate_chat tool.
	ToolNameChat = "wingmate_chat"

	// ToolNameStatus is the name of the wingmate_status tool.
	ToolNameStatus = "wingmate_status"

	// ToolNameDiscover is the name of the wingmate_discover tool.
	ToolNameDiscover = "wingmate_discover"

	// ToolNameAsk is the name of the wingmate_ask tool.
	ToolNameAsk = "wingmate_ask"
)

// NewChatTool creates the wingmate_chat tool definition.
// This tool sends prompts to Claude CLI and returns responses.
func NewChatTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        ToolNameChat,
		Description: "Send a prompt to Claude CLI and get a response",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "The prompt to send to Claude CLI",
				},
				"session_id": map[string]any{
					"type":        "string",
					"description": "Optional session ID for conversation continuity",
				},
			},
			"required": []string{"prompt"},
		},
	}
}

// NewStatusTool creates the wingmate_status tool definition.
// This tool returns server health status information.
func NewStatusTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        ToolNameStatus,
		Description: "Get Wingmate server health status",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// NewDiscoverTool creates the wingmate_discover tool definition.
// This tool lists known peer agents.
func NewDiscoverTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        ToolNameDiscover,
		Description: "List known peer agents",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// NewAskTool creates the wingmate_ask tool definition.
// This tool delegates a message to a named peer agent via A2A protocol.
func NewAskTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        ToolNameAsk,
		Description: "Send a message to a peer agent and get their response",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"peer": map[string]any{
					"type":        "string",
					"description": "Peer agent name or URL to delegate to",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "Message to send to the peer agent",
				},
				"session_id": map[string]any{
					"type":        "string",
					"description": "Optional session ID for multi-turn conversation continuity",
				},
			},
			"required": []string{"peer", "message"},
		},
	}
}

// DefaultTools returns all default Wingmate MCP tools.
func DefaultTools() []*ToolDefinition {
	return []*ToolDefinition{
		NewChatTool(),
		NewStatusTool(),
		NewDiscoverTool(),
		NewAskTool(),
	}
}
