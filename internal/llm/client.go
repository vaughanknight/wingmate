// Package llm provides types and interfaces for LLM integration via CLI.
package llm

import "context"

// LLMExecutor defines the interface for executing LLM requests.
// This abstraction allows different LLM providers (Claude CLI, GitHub Copilot, etc.)
// to be used interchangeably and enables mock implementations for testing.
type LLMExecutor interface {
	// Execute sends a prompt to the LLM and returns the response.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - prompt: The user's message to send to the LLM
	//   - sessionID: Optional session ID for conversation continuity.
	//     If empty, a new session is started.
	//     If provided, the conversation continues from that session.
	//
	// Returns:
	//   - *CLIResponse: The LLM's response with result text and session info
	//   - error: An LLMError if execution fails
	Execute(ctx context.Context, prompt string, sessionID string) (*CLIResponse, error)

	// IsInstalled reports whether the LLM provider is available on the system.
	// For CLI-based providers, this checks if the binary is in PATH.
	IsInstalled() bool
}
