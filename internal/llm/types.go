// Package llm provides types and interfaces for LLM integration via CLI.
//
// This package defines the data structures for communicating with Claude CLI
// and converting responses to A2A message format. It follows the CLI-based
// integration approach (not direct API) as specified in the project plan.
package llm

// CLIResponse represents the JSON output from Claude CLI.
// This structure matches the output of: claude -p "prompt" --output-format json
type CLIResponse struct {
	// Result contains the assistant's text response.
	Result string `json:"result"`

	// SessionID is the UUID identifying the conversation session.
	// Used with --resume flag for follow-up messages.
	SessionID string `json:"session_id"`

	// Usage contains token count information.
	Usage Usage `json:"usage"`

	// Metadata contains model information.
	Metadata Metadata `json:"metadata"`
}

// Usage represents token usage statistics from the CLI response.
type Usage struct {
	// InputTokens is the number of tokens in the prompt.
	InputTokens int `json:"input_tokens"`

	// OutputTokens is the number of tokens in the response.
	OutputTokens int `json:"output_tokens"`
}

// Metadata contains additional information about the CLI response.
type Metadata struct {
	// Model is the Claude model that generated the response.
	// Example: "claude-sonnet-4-20250514"
	Model string `json:"model"`
}
