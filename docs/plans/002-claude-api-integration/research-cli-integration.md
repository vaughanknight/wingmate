# Claude CLI Integration Research Dossier

**Created**: 2026-01-21
**Purpose**: Document findings for CLI-based Claude integration (replacing API approach)

---

## Executive Summary

Wingmate will integrate with Claude via the **Claude CLI** (Claude Code) rather than direct API calls. This approach:
- Leverages CLI's built-in session management
- Doesn't require API keys in Wingmate config (CLI handles auth)
- Supports future extensibility to other CLI tools (GitHub Copilot, etc.)

---

## 1. CLI Invocation

### Basic Syntax
```bash
claude [flags] [prompt]
```

### Programmatic (Non-Interactive) Mode
```bash
# Single prompt, JSON output
claude -p "What does this code do?" --output-format json

# With specific session
claude -p "Follow-up question" --resume "session-id" --output-format json

# Continue most recent session
claude -p "Another question" --continue --output-format json
```

### Key Flags for Wingmate

| Flag | Short | Purpose |
|------|-------|---------|
| `--print` | `-p` | Non-interactive mode (required for programmatic use) |
| `--output-format json` | | Structured JSON response with session_id |
| `--resume` | `-r` | Resume specific session by ID |
| `--continue` | `-c` | Continue most recent session |
| `--max-turns` | | Limit agentic turns |
| `--model` | | Select model (sonnet, opus, haiku) |

---

## 2. Session Management

### How Sessions Work
- Each conversation has a unique `session_id` (UUID)
- Sessions stored in SQLite at `~/.claude/`
- Sessions persist full message history and tool state

### Session Flow for Wingmate
```
Conversation Start:
  1. Wingmate receives A2A message
  2. Execute: claude -p "message" --output-format json
  3. Parse JSON response, extract session_id
  4. Store session_id mapped to conversation
  5. Return response to peer

Conversation Continue:
  1. Wingmate receives follow-up A2A message
  2. Look up session_id for this conversation
  3. Execute: claude -p "message" --resume "session-id" --output-format json
  4. Return response to peer
```

### Session ID Storage
Wingmate needs to maintain a mapping:
```
conversation_id -> claude_session_id
```

Options:
- In-memory map (lost on restart)
- Flight Log entries (recoverable)
- Dedicated session store file

---

## 3. JSON Response Format

### Response Structure
```json
{
  "result": "The assistant's text response",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "usage": {
    "input_tokens": 150,
    "output_tokens": 523
  },
  "metadata": {
    "model": "claude-sonnet-4-20250514"
  }
}
```

### Parsing with jq (reference)
```bash
# Extract text result
claude -p "question" --output-format json | jq -r '.result'

# Extract session ID
claude -p "question" --output-format json | jq -r '.session_id'
```

---

## 4. Installation Detection

### Check Methods
```bash
# Check if installed
which claude        # Returns path or empty
command -v claude   # Returns path or empty

# Check version (also confirms working)
claude --version

# Full diagnostic
claude doctor
```

### Wingmate Behavior
- On startup or first LLM request: check if `claude` is in PATH
- If not found: log warning, return error code to peer
- Do NOT attempt auto-install

---

## 5. Error Handling

### Exit Codes

| Code | Meaning | Wingmate Action |
|------|---------|-----------------|
| 0 | Success | Parse JSON, return response |
| 1 | Non-blocking error | Log warning, return partial if available |
| 2 | Blocking error | Log error, return error to peer |
| 127 | Not found | Return "Claude CLI not installed" error |

### Timeout Handling
- CLI operations can be long-running
- Use Go's `context.Context` with deadline
- Default timeout: 120s (matches previous API default)

### Example Error Response
```go
// CLI not installed
{
    Code:    2001,  // CodeLLMUnavailable
    Message: "Claude CLI not installed - install from https://claude.ai/download",
}

// CLI execution failed
{
    Code:    2001,
    Message: "Claude CLI error: <stderr content>",
}
```

---

## 6. Configuration Changes

### Removed (API-specific)
- `WINGMATE_CLAUDE_API_KEY` - not needed, CLI handles auth
- `WINGMATE_CLAUDE_MODEL` - can use CLI's default or pass via flag
- `WINGMATE_CLAUDE_MAX_TOKENS` - controlled by CLI

### New/Modified
- `WINGMATE_CLAUDE_TIMEOUT` - keep for execution timeout
- `WINGMATE_CLAUDE_CLI_PATH` - optional custom path to claude binary
- `WINGMATE_CLAUDE_MODEL` - optional, passed as --model flag

### ClaudeConfig Struct (Revised)
```go
type ClaudeConfig struct {
    CLIPath      string        `json:"cliPath,omitempty"`      // Custom claude binary path
    Model        string        `json:"model,omitempty"`        // Model override (sonnet/opus/haiku)
    Timeout      time.Duration `json:"timeout,omitempty"`      // Execution timeout (default: 120s)
    SystemPrompt string        `json:"systemPrompt,omitempty"` // Append to system prompt
}
```

---

## 7. Implementation Architecture

### Package Structure
```
internal/llm/
├── cli.go              # Claude CLI executor
├── cli_test.go         # CLI tests (with mock)
├── session.go          # Session ID management
├── session_test.go
├── types.go            # Request/Response types (revised for CLI)
├── types_test.go
├── errors.go           # Error types (keep existing)
└── errors_test.go
```

### Key Components

**CLIExecutor** (replaces ClaudeClient):
```go
type CLIExecutor struct {
    cliPath string
    timeout time.Duration
    model   string
}

func (e *CLIExecutor) Execute(ctx context.Context, prompt string, sessionID string) (*CLIResponse, error)
func (e *CLIExecutor) IsInstalled() bool
```

**SessionManager**:
```go
type SessionManager struct {
    sessions map[string]string // conversationID -> claudeSessionID
}

func (m *SessionManager) GetSession(conversationID string) (string, bool)
func (m *SessionManager) SetSession(conversationID, claudeSessionID string)
```

---

## 8. Testing Strategy

### Unit Tests
- Mock `exec.Command` for CLI invocation
- Test JSON parsing
- Test session management
- Test error handling for various exit codes

### Integration Tests
- Requires Claude CLI installed (skip if not available)
- Test actual session creation and resume
- Use `t.Skip` pattern for CI environments

### Test Markers
```go
func TestCLI_Integration(t *testing.T) {
    if os.Getenv("WINGMATE_TEST_CLI") == "" {
        t.Skip("Skipping CLI integration test - set WINGMATE_TEST_CLI=1")
    }
    // ... actual CLI tests
}
```

---

## 9. Migration from API Approach

### Files to Replace
| Current (API) | New (CLI) |
|---------------|-----------|
| `claude.go` | `cli.go` |
| `claude_test.go` | `cli_test.go` |

### Files to Modify
| File | Changes |
|------|---------|
| `types.go` | Simplify for CLI response format |
| `convert.go` | May be simplified or removed |
| `config.go` | Remove API key, add CLI path |

### Files to Keep
| File | Reason |
|------|--------|
| `errors.go` | Error codes still applicable |
| `client.go` | Interface may be adapted |

---

## 10. Future Extensibility

### Multi-CLI Support
Design should allow for:
- GitHub Copilot CLI
- Other LLM CLI tools

### Suggested Interface
```go
type LLMExecutor interface {
    Execute(ctx context.Context, prompt string, sessionID string) (*Response, error)
    IsInstalled() bool
    Name() string
}

// Implementations
type ClaudeExecutor struct { ... }
type CopilotExecutor struct { ... }  // Future
```

---

## Sources

- [CLI reference - Claude Code Docs](https://code.claude.com/docs/en/cli-reference)
- [Run Claude Code programmatically](https://code.claude.com/docs/en/headless)
- [Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices)
- [Agent SDK reference - Python](https://platform.claude.com/docs/en/agent-sdk/python)
