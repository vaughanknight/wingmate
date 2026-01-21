package agent

import (
	"encoding/json"
	"strings"

	"github.com/wingmate/wingmate/pkg/types"
)

// isPing checks if a message is a ping request.
func isPing(msg *types.Message) bool {
	if msg == nil || len(msg.Parts) == 0 {
		return false
	}

	// Check for "ping" in text parts
	for _, part := range msg.Parts {
		if part.Kind == "text" {
			text := strings.TrimSpace(strings.ToLower(part.Text))
			if text == "ping" {
				return true
			}
		}
	}

	return false
}

// pongResponse creates a pong response message.
func pongResponse() *types.Message {
	return &types.Message{
		Role: "assistant",
		Parts: []types.Part{
			{Kind: "text", Text: "pong"},
		},
	}
}

// mustMarshal marshals a value to JSON, panicking on error.
// Only use for types that are known to be JSON-serializable.
func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic("mustMarshal: " + err.Error())
	}
	return data
}

// parseMessageFromResult parses a Message from a JSON-RPC result.
func parseMessageFromResult(result json.RawMessage) (*types.Message, error) {
	var msg types.Message
	if err := json.Unmarshal(result, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// isPong checks if a message is a pong response.
func isPong(msg *types.Message) bool {
	if msg == nil || len(msg.Parts) == 0 {
		return false
	}

	for _, part := range msg.Parts {
		if part.Kind == "text" {
			text := strings.TrimSpace(strings.ToLower(part.Text))
			if text == "pong" {
				return true
			}
		}
	}

	return false
}
