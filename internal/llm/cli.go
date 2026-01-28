// Package llm provides types and interfaces for LLM integration.
//
// Why: CLIExecutor invokes the Claude CLI binary to execute prompts,
// enabling Wingmate to use Claude's capabilities without direct API access.
//
// Contract: CLIExecutor implements LLMExecutor. It builds correct command lines,
// handles all exit codes per CLI spec, and respects configured timeouts.
//
// Usage Notes: Create with NewCLIExecutor(opts...). Configure with WithTimeout,
// WithCLIPath, WithModel. Call Execute with context, prompt, and optional sessionID.
//
// Quality Contribution: Provides a clean abstraction over CLI invocation,
// handling subprocess management, timeout enforcement, and error mapping.
//
// Worked Example:
//
//	executor := NewCLIExecutor(WithTimeout(60*time.Second))
//	resp, err := executor.Execute(ctx, "Hello", "")
//	// resp.Result contains Claude's response
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

// CLIExecutor implements LLMExecutor by invoking the Claude CLI binary.
type CLIExecutor struct {
	cliPath      string
	timeout      time.Duration
	model        string
	systemPrompt string
	execCommand  func(name string, args ...string) *exec.Cmd
	lookPath     func(file string) (string, error)
}

// Option configures a CLIExecutor.
type Option func(*CLIExecutor)

// NewCLIExecutor creates a new CLIExecutor with the given options.
// Default: cliPath="claude", timeout=120s, model="" (use CLI default).
func NewCLIExecutor(opts ...Option) *CLIExecutor {
	e := &CLIExecutor{
		cliPath:     "claude",
		timeout:     120 * time.Second,
		model:       "",
		execCommand: exec.Command,
		lookPath:    exec.LookPath,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// WithTimeout sets the command timeout.
func WithTimeout(d time.Duration) Option {
	return func(e *CLIExecutor) {
		e.timeout = d
	}
}

// WithCLIPath sets the path to the Claude CLI binary.
func WithCLIPath(path string) Option {
	return func(e *CLIExecutor) {
		e.cliPath = path
	}
}

// WithSystemPrompt sets the system prompt for CLI invocations.
func WithSystemPrompt(prompt string) Option {
	return func(e *CLIExecutor) {
		e.systemPrompt = prompt
	}
}

// WithModel sets the model to use.
func WithModel(model string) Option {
	return func(e *CLIExecutor) {
		e.model = model
	}
}

// Execute runs a prompt through the Claude CLI and returns the response.
// If sessionID is non-empty, continues an existing conversation.
func (e *CLIExecutor) Execute(ctx context.Context, prompt string, sessionID string) (*CLIResponse, error) {
	args := e.buildArgs(prompt, sessionID)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := e.execCommand(e.cliPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, WrapLLMError(CodeLLMUnavailable, "failed to start Claude CLI", err)
	}

	// Wait for completion with context
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Timeout - kill the process
		_ = cmd.Process.Kill()
		return nil, NewLLMError(CodeLLMTimeout, "Claude CLI timed out")

	case err := <-done:
		if err != nil {
			// Check exit code
			if exitErr, ok := err.(*exec.ExitError); ok {
				code := exitErr.ExitCode()
				stderrStr := strings.TrimSpace(stderr.String())

				switch code {
				case 1:
					// Non-blocking error
					return nil, WrapLLMError(CodeLLMNonBlocking, stderrStr, err)
				case 2:
					// Blocking error
					return nil, WrapLLMError(CodeLLMExecutionFailed, stderrStr, err)
				case 127:
					// Command not found
					return nil, WrapLLMError(CodeLLMUnavailable, "Claude CLI not found", err)
				default:
					return nil, WrapLLMError(CodeLLMExecutionFailed, stderrStr, err)
				}
			}
			return nil, WrapLLMError(CodeLLMExecutionFailed, "CLI execution failed", err)
		}
	}

	// Parse JSON response
	var resp CLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, WrapLLMError(CodeLLMInvalidResponse, "failed to parse CLI response", err)
	}

	return &resp, nil
}

// IsInstalled returns true if the Claude CLI is available in PATH.
func (e *CLIExecutor) IsInstalled() bool {
	_, err := e.lookPath(e.cliPath)
	return err == nil
}

// buildArgs constructs the command-line arguments for the CLI.
func (e *CLIExecutor) buildArgs(prompt string, sessionID string) []string {
	args := []string{"-p", prompt, "--output-format", "json"}

	if e.model != "" {
		args = append(args, "--model", e.model)
	}

	if e.systemPrompt != "" {
		args = append(args, "--system-prompt", e.systemPrompt)
	}

	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	}

	return args
}
