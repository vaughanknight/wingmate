// Package llm provides types and interfaces for LLM integration via CLI.
package llm

import (
	"github.com/wingmate/wingmate/pkg/types"
)

// FromCLIResponse converts a Claude CLI response to an A2A Message.
//
// The resulting message has:
//   - Role: "assistant" (indicating the message is from Claude)
//   - Parts: A single text part containing the response result
//
// Parameters:
//   - resp: The CLI response to convert
//
// Returns:
//   - *types.Message: The converted message
//   - error: An LLMError if conversion fails (e.g., nil input)
func FromCLIResponse(resp *CLIResponse) (*types.Message, error) {
	if resp == nil {
		return nil, NewLLMError(CodeLLMInvalidResponse, "cannot convert nil CLIResponse")
	}

	return &types.Message{
		Role: "assistant",
		Parts: []types.Part{
			types.NewTextPart(resp.Result),
		},
	}, nil
}
