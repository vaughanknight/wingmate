// Package llm provides types and interfaces for LLM integration.
//
// Why: Tests verify that CLIExecutor correctly invokes the Claude CLI binary,
// handles responses, and manages errors.
//
// Contract: CLIExecutor must implement LLMExecutor, build correct command lines,
// handle all exit codes, and respect timeouts.
//
// Usage Notes: Run with `go test -v ./internal/llm/... -run CLI`
// Uses mock exec.Command pattern to avoid requiring CLI installed.
//
// Quality Contribution: Ensures CLI integration is reliable without requiring
// actual CLI installation in test environments.
//
// Worked Example: TestCLIExecutor_Execute_Success demonstrates a successful
// CLI invocation with JSON parsing.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Mock command execution for testing.
// This uses the TestHelperProcess pattern from Go's exec package.
var execCommand = exec.Command

// TestHelperProcess is used by tests to mock exec.Command.
// It's not a real test - it's invoked as a subprocess by mocked commands.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Get the mock behavior from environment
	behavior := os.Getenv("GO_HELPER_BEHAVIOR")
	exitCode := 0

	switch behavior {
	case "success":
		// Output valid JSON response
		resp := CLIResponse{
			Result:    "Hello from Claude",
			SessionID: "test-session-123",
			Usage: Usage{
				InputTokens:  10,
				OutputTokens: 20,
			},
			Metadata: Metadata{
				Model: "claude-sonnet-4",
			},
		}
		json.NewEncoder(os.Stdout).Encode(resp)

	case "success_with_session":
		// Verify --resume flag was passed
		args := os.Args
		hasResume := false
		for i, arg := range args {
			if arg == "--resume" && i+1 < len(args) {
				hasResume = true
				break
			}
		}
		if !hasResume {
			fmt.Fprintln(os.Stderr, "expected --resume flag")
			os.Exit(1)
		}
		resp := CLIResponse{
			Result:    "Continued conversation",
			SessionID: "test-session-123",
		}
		json.NewEncoder(os.Stdout).Encode(resp)

	case "exit_1":
		// Non-blocking error
		fmt.Fprintln(os.Stderr, "warning: non-blocking error")
		exitCode = 1

	case "exit_2":
		// Blocking error
		fmt.Fprintln(os.Stderr, "error: blocking error")
		exitCode = 2

	case "exit_127":
		// Command not found
		exitCode = 127

	case "invalid_json":
		// Invalid JSON response
		fmt.Println("not valid json {{{")

	case "slow":
		// Simulate slow command for timeout testing
		time.Sleep(5 * time.Second)
		resp := CLIResponse{Result: "slow response"}
		json.NewEncoder(os.Stdout).Encode(resp)

	case "capture_args":
		// Output the arguments for verification
		fmt.Println(strings.Join(os.Args[3:], " ")) // Skip test binary, -test.run, --

	default:
		fmt.Fprintf(os.Stderr, "unknown behavior: %s\n", behavior)
		exitCode = 1
	}

	os.Exit(exitCode)
}

// mockExecCommand returns a function that creates mock commands.
func mockExecCommand(behavior string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_BEHAVIOR=" + behavior,
		}
		return cmd
	}
}

func TestCLIExecutor_SatisfiesInterface(t *testing.T) {
	// Given: A CLIExecutor
	executor := NewCLIExecutor()

	// Then: It should satisfy the LLMExecutor interface
	var _ LLMExecutor = executor
}

func TestCLIExecutor_NewWithDefaults(t *testing.T) {
	// When: We create a new executor with defaults
	executor := NewCLIExecutor()

	// Then: It should have default values
	if executor.cliPath != "claude" {
		t.Errorf("cliPath = %q, want %q", executor.cliPath, "claude")
	}

	if executor.timeout != 120*time.Second {
		t.Errorf("timeout = %v, want %v", executor.timeout, 120*time.Second)
	}

	if executor.model != "" {
		t.Errorf("model = %q, want empty string", executor.model)
	}
}

func TestCLIExecutor_WithTimeout(t *testing.T) {
	// When: We create an executor with custom timeout
	executor := NewCLIExecutor(WithTimeout(60 * time.Second))

	// Then: Timeout should be set
	if executor.timeout != 60*time.Second {
		t.Errorf("timeout = %v, want %v", executor.timeout, 60*time.Second)
	}
}

func TestCLIExecutor_WithCLIPath(t *testing.T) {
	// When: We create an executor with custom CLI path
	executor := NewCLIExecutor(WithCLIPath("/custom/path/claude"))

	// Then: CLI path should be set
	if executor.cliPath != "/custom/path/claude" {
		t.Errorf("cliPath = %q, want %q", executor.cliPath, "/custom/path/claude")
	}
}

func TestCLIExecutor_WithModel(t *testing.T) {
	// When: We create an executor with custom model
	executor := NewCLIExecutor(WithModel("opus"))

	// Then: Model should be set
	if executor.model != "opus" {
		t.Errorf("model = %q, want %q", executor.model, "opus")
	}
}

func TestCLIExecutor_Execute_Success(t *testing.T) {
	// Setup mock
	oldExecCommand := execCommand
	execCommand = mockExecCommand("success")
	defer func() { execCommand = oldExecCommand }()

	// Given: An executor
	executor := NewCLIExecutor()
	executor.execCommand = execCommand

	// When: We execute a prompt
	ctx := context.Background()
	resp, err := executor.Execute(ctx, "Hello", "")

	// Then: It should succeed
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Result != "Hello from Claude" {
		t.Errorf("Result = %q, want %q", resp.Result, "Hello from Claude")
	}

	if resp.SessionID != "test-session-123" {
		t.Errorf("SessionID = %q, want %q", resp.SessionID, "test-session-123")
	}
}

func TestCLIExecutor_Execute_WithSessionID(t *testing.T) {
	// Setup mock
	oldExecCommand := execCommand
	execCommand = mockExecCommand("success_with_session")
	defer func() { execCommand = oldExecCommand }()

	// Given: An executor
	executor := NewCLIExecutor()
	executor.execCommand = execCommand

	// When: We execute with a session ID
	ctx := context.Background()
	resp, err := executor.Execute(ctx, "Follow-up", "existing-session")

	// Then: It should succeed
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Result != "Continued conversation" {
		t.Errorf("Result = %q, want %q", resp.Result, "Continued conversation")
	}
}

func TestCLIExecutor_Execute_ExitCode1(t *testing.T) {
	// Setup mock
	oldExecCommand := execCommand
	execCommand = mockExecCommand("exit_1")
	defer func() { execCommand = oldExecCommand }()

	// Given: An executor
	executor := NewCLIExecutor()
	executor.execCommand = execCommand

	// When: CLI returns exit code 1 (non-blocking error)
	ctx := context.Background()
	_, err := executor.Execute(ctx, "test", "")

	// Then: It should return an error with code 2004
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	llmErr, ok := err.(*LLMError)
	if !ok {
		t.Fatalf("expected *LLMError, got %T", err)
	}

	if llmErr.Code != CodeLLMNonBlocking {
		t.Errorf("Code = %d, want %d", llmErr.Code, CodeLLMNonBlocking)
	}
}

func TestCLIExecutor_Execute_ExitCode2(t *testing.T) {
	// Setup mock
	oldExecCommand := execCommand
	execCommand = mockExecCommand("exit_2")
	defer func() { execCommand = oldExecCommand }()

	// Given: An executor
	executor := NewCLIExecutor()
	executor.execCommand = execCommand

	// When: CLI returns exit code 2 (blocking error)
	ctx := context.Background()
	_, err := executor.Execute(ctx, "test", "")

	// Then: It should return an error with code 2002
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	llmErr, ok := err.(*LLMError)
	if !ok {
		t.Fatalf("expected *LLMError, got %T", err)
	}

	if llmErr.Code != CodeLLMExecutionFailed {
		t.Errorf("Code = %d, want %d", llmErr.Code, CodeLLMExecutionFailed)
	}
}

func TestCLIExecutor_Execute_ExitCode127(t *testing.T) {
	// Setup mock
	oldExecCommand := execCommand
	execCommand = mockExecCommand("exit_127")
	defer func() { execCommand = oldExecCommand }()

	// Given: An executor
	executor := NewCLIExecutor()
	executor.execCommand = execCommand

	// When: CLI returns exit code 127 (not found)
	ctx := context.Background()
	_, err := executor.Execute(ctx, "test", "")

	// Then: It should return an error with code 2001
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	llmErr, ok := err.(*LLMError)
	if !ok {
		t.Fatalf("expected *LLMError, got %T", err)
	}

	if llmErr.Code != CodeLLMUnavailable {
		t.Errorf("Code = %d, want %d", llmErr.Code, CodeLLMUnavailable)
	}
}

func TestCLIExecutor_Execute_InvalidJSON(t *testing.T) {
	// Setup mock
	oldExecCommand := execCommand
	execCommand = mockExecCommand("invalid_json")
	defer func() { execCommand = oldExecCommand }()

	// Given: An executor
	executor := NewCLIExecutor()
	executor.execCommand = execCommand

	// When: CLI returns invalid JSON
	ctx := context.Background()
	_, err := executor.Execute(ctx, "test", "")

	// Then: It should return an error with code 2006
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	llmErr, ok := err.(*LLMError)
	if !ok {
		t.Fatalf("expected *LLMError, got %T", err)
	}

	if llmErr.Code != CodeLLMInvalidResponse {
		t.Errorf("Code = %d, want %d", llmErr.Code, CodeLLMInvalidResponse)
	}
}

func TestCLIExecutor_Execute_Timeout(t *testing.T) {
	// Setup mock
	oldExecCommand := execCommand
	execCommand = mockExecCommand("slow")
	defer func() { execCommand = oldExecCommand }()

	// Given: An executor with short timeout
	executor := NewCLIExecutor(WithTimeout(100 * time.Millisecond))
	executor.execCommand = execCommand

	// When: CLI takes too long
	ctx := context.Background()
	_, err := executor.Execute(ctx, "test", "")

	// Then: It should return a timeout error
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	llmErr, ok := err.(*LLMError)
	if !ok {
		t.Fatalf("expected *LLMError, got %T", err)
	}

	if llmErr.Code != CodeLLMTimeout {
		t.Errorf("Code = %d, want %d", llmErr.Code, CodeLLMTimeout)
	}
}

func TestCLIExecutor_BuildArgs_Basic(t *testing.T) {
	// Given: An executor
	executor := NewCLIExecutor()

	// When: We build args without session
	args := executor.buildArgs("Hello world", "")

	// Then: Should have correct flags
	expected := []string{"-p", "Hello world", "--output-format", "json"}

	if len(args) != len(expected) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(expected))
	}

	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestCLIExecutor_BuildArgs_WithSession(t *testing.T) {
	// Given: An executor
	executor := NewCLIExecutor()

	// When: We build args with session
	args := executor.buildArgs("Follow-up", "session-123")

	// Then: Should include --resume flag
	expected := []string{"-p", "Follow-up", "--output-format", "json", "--resume", "session-123"}

	if len(args) != len(expected) {
		t.Fatalf("len(args) = %d, want %d\nargs: %v\nexpected: %v", len(args), len(expected), args, expected)
	}

	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestCLIExecutor_BuildArgs_WithModel(t *testing.T) {
	// Given: An executor with model
	executor := NewCLIExecutor(WithModel("opus"))

	// When: We build args
	args := executor.buildArgs("Test", "")

	// Then: Should include --model flag
	hasModel := false
	for i, arg := range args {
		if arg == "--model" && i+1 < len(args) && args[i+1] == "opus" {
			hasModel = true
			break
		}
	}

	if !hasModel {
		t.Errorf("expected --model opus in args: %v", args)
	}
}

func TestCLIExecutor_IsInstalled_True(t *testing.T) {
	// Given: An executor with a path that exists (use this test binary)
	executor := NewCLIExecutor(WithCLIPath(os.Args[0]))

	// Override lookPath to simulate finding the binary
	executor.lookPath = func(file string) (string, error) {
		return file, nil
	}

	// Then: IsInstalled should return true
	if !executor.IsInstalled() {
		t.Error("IsInstalled() = false, want true")
	}
}

func TestCLIExecutor_IsInstalled_False(t *testing.T) {
	// Given: An executor
	executor := NewCLIExecutor()

	// Override lookPath to simulate not finding the binary
	executor.lookPath = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}

	// Then: IsInstalled should return false
	if executor.IsInstalled() {
		t.Error("IsInstalled() = true, want false")
	}
}
