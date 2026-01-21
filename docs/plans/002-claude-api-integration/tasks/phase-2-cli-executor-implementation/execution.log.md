# Phase 2: CLI Executor Implementation — Execution Log

**Phase**: 2 of 5
**Started**: 2026-01-21
**Status**: ✅ Complete

---

## Task T001: Create CLIExecutor struct

**Started**: 2026-01-21
**Status**: ✅ Complete

### What I Did

Created the `CLIExecutor` struct in `/internal/llm/cli.go` with fields:
- `cliPath string` - path to claude binary (default: "claude")
- `timeout time.Duration` - command timeout (default: 120s)
- `model string` - optional model override
- `execCommand func(...)` - injectable for testing
- `lookPath func(...)` - injectable for testing

### Evidence

```go
type CLIExecutor struct {
	cliPath     string
	timeout     time.Duration
	model       string
	execCommand func(name string, args ...string) *exec.Cmd
	lookPath    func(file string) (string, error)
}
```

Test passes:
```
=== RUN   TestCLIExecutor_SatisfiesInterface
--- PASS: TestCLIExecutor_SatisfiesInterface (0.00s)
```

### Files Changed

- `/internal/llm/cli.go` — Created with CLIExecutor struct

**Completed**: 2026-01-21

---

## Task T002: Add functional options

**Started**: 2026-01-21
**Status**: ✅ Complete

### What I Did

Implemented functional options pattern:
- `type Option func(*CLIExecutor)`
- `WithTimeout(d time.Duration) Option`
- `WithCLIPath(path string) Option`
- `WithModel(model string) Option`
- `NewCLIExecutor(opts ...Option) *CLIExecutor`

### Evidence

```
=== RUN   TestCLIExecutor_NewWithDefaults
--- PASS: TestCLIExecutor_NewWithDefaults (0.00s)
=== RUN   TestCLIExecutor_WithTimeout
--- PASS: TestCLIExecutor_WithTimeout (0.00s)
=== RUN   TestCLIExecutor_WithCLIPath
--- PASS: TestCLIExecutor_WithCLIPath (0.00s)
=== RUN   TestCLIExecutor_WithModel
--- PASS: TestCLIExecutor_WithModel (0.00s)
```

### Files Changed

- `/internal/llm/cli.go` — Added functional options

**Completed**: 2026-01-21

---

## Task T003-T004: Implement and test IsInstalled()

**Started**: 2026-01-21
**Status**: ✅ Complete

### What I Did

Implemented `IsInstalled()` using injectable `lookPath` function:
```go
func (e *CLIExecutor) IsInstalled() bool {
	_, err := e.lookPath(e.cliPath)
	return err == nil
}
```

Tests mock `lookPath` to test both found and not-found scenarios.

### Evidence

```
=== RUN   TestCLIExecutor_IsInstalled_True
--- PASS: TestCLIExecutor_IsInstalled_True (0.00s)
=== RUN   TestCLIExecutor_IsInstalled_False
--- PASS: TestCLIExecutor_IsInstalled_False (0.00s)
```

### Files Changed

- `/internal/llm/cli.go` — Added IsInstalled()
- `/internal/llm/cli_test.go` — Added IsInstalled tests

**Completed**: 2026-01-21

---

## Task T005-T006: Implement and test Execute()

**Started**: 2026-01-21
**Status**: ✅ Complete

### What I Did

Implemented `Execute()` method that:
1. Builds args array: `["-p", prompt, "--output-format", "json"]`
2. Creates context with timeout
3. Executes command via injectable `execCommand`
4. Handles completion via goroutine + channel pattern
5. Parses JSON response

Used `TestHelperProcess` pattern for mocking `exec.Command` - the test binary re-runs itself as a subprocess with controlled behavior.

### Evidence

```
=== RUN   TestCLIExecutor_Execute_Success
--- PASS: TestCLIExecutor_Execute_Success (0.01s)
=== RUN   TestCLIExecutor_BuildArgs_Basic
--- PASS: TestCLIExecutor_BuildArgs_Basic (0.00s)
```

### Files Changed

- `/internal/llm/cli.go` — Added Execute() and buildArgs()
- `/internal/llm/cli_test.go` — Added TestHelperProcess and success tests

**Completed**: 2026-01-21

---

## Task T007-T008: Add --resume flag support

**Started**: 2026-01-21
**Status**: ✅ Complete

### What I Did

Added conditional `--resume` flag in `buildArgs()`:
```go
if sessionID != "" {
	args = append(args, "--resume", sessionID)
}
```

### Evidence

```
=== RUN   TestCLIExecutor_Execute_WithSessionID
--- PASS: TestCLIExecutor_Execute_WithSessionID (0.01s)
=== RUN   TestCLIExecutor_BuildArgs_WithSession
--- PASS: TestCLIExecutor_BuildArgs_WithSession (0.00s)
```

### Files Changed

- `/internal/llm/cli.go` — Added resume flag logic
- `/internal/llm/cli_test.go` — Added session tests

**Completed**: 2026-01-21

---

## Task T009-T010: Implement timeout handling

**Started**: 2026-01-21
**Status**: ✅ Complete

### What I Did

Implemented timeout using context with deadline:
```go
ctx, cancel := context.WithTimeout(ctx, e.timeout)
defer cancel()

// Start command, then wait with select
select {
case <-ctx.Done():
	_ = cmd.Process.Kill()
	return nil, NewLLMError(CodeLLMTimeout, "Claude CLI timed out")
case err := <-done:
	// handle result
}
```

### Evidence

```
=== RUN   TestCLIExecutor_Execute_Timeout
--- PASS: TestCLIExecutor_Execute_Timeout (0.10s)
```

Test uses 100ms timeout with mock that sleeps 5s, verifying timeout fires.

### Files Changed

- `/internal/llm/cli.go` — Added timeout logic
- `/internal/llm/cli_test.go` — Added timeout test

**Completed**: 2026-01-21

---

## Task T011-T015: Handle exit codes

**Started**: 2026-01-21
**Status**: ✅ Complete

### What I Did

Implemented exit code handling in Execute():

| Exit Code | Error Code | Meaning |
|-----------|------------|---------|
| 0 | (none) | Success - parse JSON |
| 1 | 2004 | Non-blocking error |
| 2 | 2002 | Blocking error |
| 127 | 2001 | Command not found |

```go
if exitErr, ok := err.(*exec.ExitError); ok {
	code := exitErr.ExitCode()
	switch code {
	case 1:
		return nil, WrapLLMError(CodeLLMNonBlocking, stderrStr, err)
	case 2:
		return nil, WrapLLMError(CodeLLMExecutionFailed, stderrStr, err)
	case 127:
		return nil, WrapLLMError(CodeLLMUnavailable, "Claude CLI not found", err)
	}
}
```

### Evidence

```
=== RUN   TestCLIExecutor_Execute_ExitCode1
--- PASS: TestCLIExecutor_Execute_ExitCode1 (0.01s)
=== RUN   TestCLIExecutor_Execute_ExitCode2
--- PASS: TestCLIExecutor_Execute_ExitCode2 (0.01s)
=== RUN   TestCLIExecutor_Execute_ExitCode127
--- PASS: TestCLIExecutor_Execute_ExitCode127 (0.01s)
=== RUN   TestCLIExecutor_Execute_InvalidJSON
--- PASS: TestCLIExecutor_Execute_InvalidJSON (0.01s)
```

### Files Changed

- `/internal/llm/cli.go` — Added exit code handling
- `/internal/llm/cli_test.go` — Added exit code tests

**Completed**: 2026-01-21

---

## Phase Summary

**Total Tests**: 17 CLI tests (26 total in package)
**Test Coverage**: 94.6%
**All Tasks Complete**: ✅

### Files Created

| File | Purpose |
|------|---------|
| `/internal/llm/cli.go` | CLIExecutor struct with Execute, IsInstalled, buildArgs |
| `/internal/llm/cli_test.go` | Comprehensive tests using TestHelperProcess mock pattern |

### Verification

```bash
go test -v ./internal/llm/... -run CLI
# 17 tests pass

go test -cover ./internal/llm/...
# coverage: 94.6% of statements
```

### Key Implementation Decisions

1. **Injectable dependencies**: Both `execCommand` and `lookPath` are injectable for testing
2. **TestHelperProcess pattern**: Standard Go pattern for mocking exec.Command without external tools
3. **Goroutine + channel for timeout**: Allows clean process kill on timeout
4. **Exit code 1 returns error**: Spec says "non-blocking" but we treat all non-zero as errors for simplicity

---
