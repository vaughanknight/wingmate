// Package mcp provides tests for the MCP tool definitions and registration.
//
// Why: Ensures tools/list returns registered tools per MCP specification.
// Contract: After RegisterTool is called, tools/list must return all registered tools.
// Usage Notes: Use TestTransport for bidirectional communication testing.
// Quality Contribution: Catches missing tool registration or schema mismatches.
package mcp

import (
	"context"
	"testing"

	"github.com/wingmate/wingmate/tests/helpers"
)

// TestToolsListIncludesRegisteredTools verifies that tools/list returns
// tools that have been registered with the server.
//
// Given: A server with wingmate_chat registered
// When: Client sends tools/list request
// Then: Response includes wingmate_chat tool with correct schema
func TestToolsListIncludesRegisteredTools(t *testing.T) {
	// Setup
	tt := helpers.NewTestTransport()
	t.Cleanup(func() { tt.Close() })

	transport := NewTransport(tt.ServerReader, tt.ServerWriter)
	config := ServerConfig{Name: "test-server", Version: "1.0.0"}
	server := NewServer(config, transport)

	// Register default tools
	for _, tool := range DefaultTools() {
		server.RegisterTool(tool)
	}

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = server.Run(ctx)
	}()

	// Send initialize request first (required before tools/list)
	err := tt.SendRequest(1, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
	})
	helpers.AssertNoError(t, err, "send initialize")

	_, err = tt.ReadResponse()
	helpers.AssertNoError(t, err, "read initialize response")

	// Send tools/list request
	err = tt.SendRequest(2, "tools/list", nil)
	helpers.AssertNoError(t, err, "send tools/list")

	// Read response
	resp, err := tt.ReadResponse()
	helpers.AssertNoError(t, err, "read tools/list response")

	// Assert no error
	if resp.Error != nil {
		t.Fatalf("tools/list error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("Expected result to be map, got %T", resp.Result)
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("Expected tools to be array, got %T", result["tools"])
	}

	// Should have all 3 default tools
	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}

	// Find wingmate_chat tool
	var chatTool map[string]any
	for _, tool := range tools {
		toolMap := tool.(map[string]any)
		if toolMap["name"] == ToolNameChat {
			chatTool = toolMap
			break
		}
	}

	if chatTool == nil {
		t.Fatal("Expected wingmate_chat tool to be registered")
	}

	// Verify schema
	if chatTool["description"] != "Send a prompt to Claude CLI and get a response" {
		t.Errorf("Unexpected description: %s", chatTool["description"])
	}

	schema, ok := chatTool["inputSchema"].(map[string]any)
	if !ok {
		t.Fatal("Expected inputSchema to be a map")
	}

	if schema["type"] != "object" {
		t.Errorf("Expected schema type to be object, got %s", schema["type"])
	}
}

// TestToolsListIncludesAllDefaultTools verifies all three default tools are present.
//
// Given: A server with default tools registered
// When: Client sends tools/list request
// Then: Response includes wingmate_chat, wingmate_status, wingmate_discover
func TestToolsListIncludesAllDefaultTools(t *testing.T) {
	// Setup
	tt := helpers.NewTestTransport()
	t.Cleanup(func() { tt.Close() })

	transport := NewTransport(tt.ServerReader, tt.ServerWriter)
	config := ServerConfig{Name: "test-server", Version: "1.0.0"}
	server := NewServer(config, transport)

	// Register default tools
	for _, tool := range DefaultTools() {
		server.RegisterTool(tool)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = server.Run(ctx)
	}()

	// Initialize first
	err := tt.SendRequest(1, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
	})
	helpers.AssertNoError(t, err, "send initialize")
	tt.ReadResponse()

	// Send tools/list
	err = tt.SendRequest(2, "tools/list", nil)
	helpers.AssertNoError(t, err, "send tools/list")

	resp, err := tt.ReadResponse()
	helpers.AssertNoError(t, err, "read response")

	if resp.Error != nil {
		t.Fatalf("tools/list error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	tools := result["tools"].([]any)

	// Collect tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]any)
		toolNames[toolMap["name"].(string)] = true
	}

	// Verify all default tools are present
	expectedTools := []string{ToolNameChat, ToolNameStatus, ToolNameDiscover}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool %q to be registered", name)
		}
	}
}

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
