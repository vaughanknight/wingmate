// Package mcp provides the MCP (Model Context Protocol) server implementation
// for Wingmate. It enables Claude Code and other MCP-compatible clients to
// invoke Wingmate's LLM capabilities through a standardized interface.
//
// This package uses the official modelcontextprotocol/go-sdk for protocol
// handling per ADR-004.
package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerInfo contains information about the MCP server.
// This wraps mcp.Implementation to isolate internal code from SDK changes.
type ServerInfo struct {
	// Name is the server name (e.g., "wingmate").
	Name string

	// Version is the server version.
	Version string
}

// toSDK converts ServerInfo to SDK's Implementation type.
func (s *ServerInfo) toSDK() *mcp.Implementation {
	return &mcp.Implementation{
		Name:    s.Name,
		Version: s.Version,
	}
}

// ToolDefinition defines an MCP tool that can be registered with the server.
// This wraps mcp.Tool to provide a stable internal API.
type ToolDefinition struct {
	// Name is the tool identifier (e.g., "wingmate_chat").
	Name string

	// Description describes what the tool does.
	Description string

	// InputSchema is the JSON Schema for tool input validation.
	// This is optional when using AddTool with typed handlers.
	InputSchema map[string]any
}

// toSDK converts ToolDefinition to SDK's Tool type.
func (t *ToolDefinition) toSDK() *mcp.Tool {
	tool := &mcp.Tool{
		Name:        t.Name,
		Description: t.Description,
	}
	if t.InputSchema != nil {
		tool.InputSchema = t.InputSchema
	}
	return tool
}

// ToolResult represents the result of a tool invocation.
// This wraps mcp.CallToolResult to provide a stable internal API.
type ToolResult struct {
	// Content is the tool output content.
	Content []Content

	// IsError indicates if the result represents an error.
	IsError bool
}

// toSDK converts ToolResult to SDK's CallToolResult type.
func (r *ToolResult) toSDK() *mcp.CallToolResult {
	result := &mcp.CallToolResult{
		IsError: r.IsError,
	}
	for _, c := range r.Content {
		result.Content = append(result.Content, c.toSDK())
	}
	return result
}

// Content represents content in a tool result.
// This is an interface to support different content types.
type Content interface {
	toSDK() mcp.Content
}

// TextContent represents text content in a tool result.
type TextContent struct {
	Text string
}

// toSDK converts TextContent to SDK's TextContent type.
func (c *TextContent) toSDK() mcp.Content {
	return &mcp.TextContent{Text: c.Text}
}

// ToolRequest represents a tool invocation request.
// This wraps mcp.CallToolRequest for internal use.
type ToolRequest struct {
	// Name is the tool being invoked.
	Name string

	// Arguments are the tool input arguments.
	Arguments map[string]any

	// ctx is the request context.
	ctx context.Context
}

// Context returns the request context.
func (r *ToolRequest) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

// ToolHandler is the function signature for tool handlers.
// It receives a context and tool request, and returns a result or error.
type ToolHandler func(ctx context.Context, req *ToolRequest) (*ToolResult, error)

// ServerCapabilities describes what the MCP server supports.
// This wraps mcp.ServerCapabilities for internal use.
type ServerCapabilities struct {
	// Tools indicates the server provides tools.
	Tools bool

	// Resources indicates the server provides resources (not used in Phase 1).
	Resources bool

	// Prompts indicates the server provides prompts (not used in Phase 1).
	Prompts bool
}

// Ensure imports are used
var (
	_ = mcp.NewServer
	_ = context.Background
)
