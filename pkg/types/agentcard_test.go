package types

import (
	"encoding/json"
	"testing"
)

// TestAgentCard_JSONRoundTrip verifies that AgentCard serializes and deserializes correctly.
//
// Test Doc:
// - Why: AgentCard is the foundation of A2A discovery; incorrect serialization breaks interop
// - Contract: Marshal → Unmarshal produces identical struct
// - Usage Notes: All required fields must be present for valid A2A compliance
// - Quality Contribution: Catches JSON tag errors, field omission bugs
// - Worked Example: AgentCard{Name:"test"} → JSON → AgentCard{Name:"test"}
func TestAgentCard_JSONRoundTrip(t *testing.T) {
	original := AgentCard{
		Name:        "wingmate-001",
		Description: "Test wingmate agent",
		URL:         "http://localhost:9001",
		Version:     "0.1.0",
		Capabilities: Capabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		Skills: []Skill{
			{Name: "ping", Description: "Respond to ping with pong"},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal AgentCard: %v", err)
	}

	// Unmarshal back
	var decoded AgentCard
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AgentCard: %v", err)
	}

	// Verify fields
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description mismatch: got %q, want %q", decoded.Description, original.Description)
	}
	if decoded.URL != original.URL {
		t.Errorf("URL mismatch: got %q, want %q", decoded.URL, original.URL)
	}
	if decoded.Version != original.Version {
		t.Errorf("Version mismatch: got %q, want %q", decoded.Version, original.Version)
	}
	if decoded.Capabilities.Streaming != original.Capabilities.Streaming {
		t.Errorf("Capabilities.Streaming mismatch")
	}
	if len(decoded.Skills) != len(original.Skills) {
		t.Errorf("Skills length mismatch: got %d, want %d", len(decoded.Skills), len(original.Skills))
	}
}

// TestAgentCard_RequiredFields verifies that required fields are present in JSON output.
//
// Test Doc:
// - Why: A2A spec requires name, url, version, capabilities, skills fields
// - Contract: JSON output must contain all required keys
// - Usage Notes: Even empty arrays/objects must be present (not null)
// - Quality Contribution: Catches missing required fields early
// - Worked Example: AgentCard{} → JSON must have "skills":[] not missing
func TestAgentCard_RequiredFields(t *testing.T) {
	card := AgentCard{
		Name:    "test-agent",
		URL:     "http://localhost:9000",
		Version: "1.0.0",
		Capabilities: Capabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		Skills: []Skill{}, // Empty but must be present
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Parse as generic map to check field presence
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	requiredFields := []string{"name", "url", "version", "capabilities", "skills"}
	for _, field := range requiredFields {
		if _, ok := m[field]; !ok {
			t.Errorf("Required field %q missing from JSON output", field)
		}
	}
}

// TestAgentCard_WellKnownPath verifies the well-known path constant.
//
// Test Doc:
// - Why: A2A spec requires AgentCard at /.well-known/agent.json (not agent-card.json)
// - Contract: WellKnownPath constant is correct
// - Quality Contribution: Catches common mistake of using wrong path
func TestAgentCard_WellKnownPath(t *testing.T) {
	expected := "/.well-known/agent.json"
	if AgentCardWellKnownPath != expected {
		t.Errorf("WellKnownPath = %q, want %q", AgentCardWellKnownPath, expected)
	}
}
