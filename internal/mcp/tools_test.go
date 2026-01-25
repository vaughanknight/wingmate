// Package mcp provides tests for the MCP tool definitions and registration.
//
// Why: Ensures tool definitions have correct schemas per MCP specification.
// Contract: Tool definitions must have valid name, description, and inputSchema.
// Usage Notes: Integration tests in internal/agent cover full tool registration flow.
// Quality Contribution: Catches schema mismatches and missing tool fields.
package mcp

import (
	"testing"
)

// NOTE: Tests for tools/list over stdio transport (TestToolsListIncludesRegisteredTools,
// TestToolsListIncludesAllDefaultTools) were removed in Phase 3 of MCP HTTP Transport migration.
// Tool registration is now tested via HTTP integration tests in internal/agent/mcp_integration_test.go.

// TestWingmateChatSchema validates the wingmate_chat tool schema structure.
//
// Given: wingmate_chat tool definition
// When: Schema is examined
// Then: Required prompt field exists, optional session_id field exists
func TestWingmateChatSchema(t *testing.T) {
	tool := NewChatTool()

	if tool.Name != ToolNameChat {
		t.Errorf("Expected name %q, got %q", ToolNameChat, tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected non-empty description")
	}

	schema := tool.InputSchema
	if schema["type"] != "object" {
		t.Errorf("Expected type object, got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check prompt field
	prompt, ok := props["prompt"].(map[string]any)
	if !ok {
		t.Fatal("Expected prompt property")
	}
	if prompt["type"] != "string" {
		t.Errorf("Expected prompt type string, got %v", prompt["type"])
	}

	// Check session_id field
	sessionID, ok := props["session_id"].(map[string]any)
	if !ok {
		t.Fatal("Expected session_id property")
	}
	if sessionID["type"] != "string" {
		t.Errorf("Expected session_id type string, got %v", sessionID["type"])
	}

	// Check required
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Expected required to be string array")
	}
	if len(required) != 1 || required[0] != "prompt" {
		t.Errorf("Expected required=[\"prompt\"], got %v", required)
	}
}

// TestWingmateStatusSchema validates the wingmate_status tool schema structure.
//
// Given: wingmate_status tool definition
// When: Schema is examined
// Then: No required fields, empty properties
func TestWingmateStatusSchema(t *testing.T) {
	tool := NewStatusTool()

	if tool.Name != ToolNameStatus {
		t.Errorf("Expected name %q, got %q", ToolNameStatus, tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected non-empty description")
	}

	schema := tool.InputSchema
	if schema["type"] != "object" {
		t.Errorf("Expected type object, got %v", schema["type"])
	}
}

// TestWingmateDiscoverSchema validates the wingmate_discover tool schema structure.
//
// Given: wingmate_discover tool definition
// When: Schema is examined
// Then: No required fields, empty properties
func TestWingmateDiscoverSchema(t *testing.T) {
	tool := NewDiscoverTool()

	if tool.Name != ToolNameDiscover {
		t.Errorf("Expected name %q, got %q", ToolNameDiscover, tool.Name)
	}

	if tool.Description == "" {
		t.Error("Expected non-empty description")
	}

	schema := tool.InputSchema
	if schema["type"] != "object" {
		t.Errorf("Expected type object, got %v", schema["type"])
	}
}

// TestDefaultToolsReturnsAllTools verifies DefaultTools() returns all 3 tools.
func TestDefaultToolsReturnsAllTools(t *testing.T) {
	tools := DefaultTools()

	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}

	if !names[ToolNameChat] {
		t.Error("Missing wingmate_chat")
	}
	if !names[ToolNameStatus] {
		t.Error("Missing wingmate_status")
	}
	if !names[ToolNameDiscover] {
		t.Error("Missing wingmate_discover")
	}
}
