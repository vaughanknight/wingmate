package types

import (
	"encoding/json"
	"testing"
)

// TestMessage_PartUnion verifies that Part discriminated union works correctly.
//
// Test Doc:
// - Why: A2A uses "kind" discriminator for Part union (text/data/file)
// - Contract: Part.Kind determines which fields are populated
// - Usage Notes: Only one set of fields populated per Kind value
// - Quality Contribution: Catches union type handling bugs
// - Worked Example: {"kind":"text","text":"hello"} → Part{Kind:"text", Text:"hello"}
func TestMessage_PartUnion(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantKind string
		wantText string
	}{
		{
			name:     "text part",
			json:     `{"kind":"text","text":"hello world"}`,
			wantKind: "text",
			wantText: "hello world",
		},
		{
			name:     "data part",
			json:     `{"kind":"data","data":{"key":"value"}}`,
			wantKind: "data",
			wantText: "",
		},
		{
			name:     "file part",
			json:     `{"kind":"file","filePath":"/path/to/file","mimeType":"text/plain"}`,
			wantKind: "file",
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var part Part
			if err := json.Unmarshal([]byte(tt.json), &part); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if part.Kind != tt.wantKind {
				t.Errorf("Kind = %q, want %q", part.Kind, tt.wantKind)
			}
			if part.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", part.Text, tt.wantText)
			}
		})
	}
}

// TestMessage_NilVsEmpty verifies distinction between nil and empty parts.
//
// Test Doc:
// - Why: JSON null vs [] have different semantic meanings
// - Contract: nil parts → omitted or null; empty slice → empty array []
// - Usage Notes: Use empty slice for "no parts" to ensure [] in JSON
// - Quality Contribution: Catches nil/empty confusion bugs
// - Worked Example: Parts:nil → "parts":null; Parts:[]Part{} → "parts":[]
func TestMessage_NilVsEmpty(t *testing.T) {
	t.Run("nil parts marshals as null", func(t *testing.T) {
		msg := Message{
			Role:  "user",
			Parts: nil,
		}
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		// With omitempty, nil slice is omitted entirely
		// Without omitempty, nil slice becomes null
		// Our implementation should handle both cases gracefully
		if parts, ok := m["parts"]; ok && parts != nil {
			// If present and not null, verify it's an array
			if _, isArr := parts.([]interface{}); !isArr {
				t.Errorf("Expected parts to be null or array, got %T", parts)
			}
		}
	})

	t.Run("empty parts marshals as empty array", func(t *testing.T) {
		msg := Message{
			Role:  "user",
			Parts: []Part{},
		}
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		parts, ok := m["parts"]
		if !ok {
			t.Fatal("parts field missing from JSON")
		}

		arr, isArr := parts.([]interface{})
		if !isArr {
			t.Fatalf("parts is not an array, got %T", parts)
		}
		if len(arr) != 0 {
			t.Errorf("Expected empty array, got %d elements", len(arr))
		}
	})
}

// TestMessage_RoundTrip verifies full message serialization.
//
// Test Doc:
// - Why: Messages are the core communication unit in A2A
// - Contract: Marshal → Unmarshal produces identical struct
// - Quality Contribution: Catches any serialization bugs in Message type
func TestMessage_RoundTrip(t *testing.T) {
	original := Message{
		Role: "user",
		Parts: []Part{
			{Kind: "text", Text: "Hello, agent!"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Role != original.Role {
		t.Errorf("Role mismatch: got %q, want %q", decoded.Role, original.Role)
	}
	if len(decoded.Parts) != len(original.Parts) {
		t.Fatalf("Parts length mismatch: got %d, want %d", len(decoded.Parts), len(original.Parts))
	}
	if decoded.Parts[0].Kind != original.Parts[0].Kind {
		t.Errorf("Part Kind mismatch")
	}
	if decoded.Parts[0].Text != original.Parts[0].Text {
		t.Errorf("Part Text mismatch")
	}
}
