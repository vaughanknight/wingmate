# LLM Setup Guide

This guide covers setting up Claude CLI integration with Wingmate agents.

## Prerequisites

- Wingmate installed and working
- Internet connection for Claude CLI authentication

## Installation

### Step 1: Install Claude CLI

Download and install Claude CLI from the official source:

**macOS / Linux:**
```bash
# Download from https://claude.ai/download
# Or use the installer script (if available)
curl -fsSL https://claude.ai/install.sh | sh
```

**Windows:**
Download the installer from [claude.ai/download](https://claude.ai/download) and run it.

**Verify Installation:**
```bash
claude --version
```

### Step 2: Authenticate

Run Claude CLI once to authenticate:

```bash
claude
```

This opens a browser window for authentication. Sign in with your Anthropic account.

### Step 3: Verify Integration

Start a Wingmate agent and check the Agent Card:

```bash
./bin/wingmate --name test-agent --port 9000
```

In another terminal:
```bash
curl http://localhost:9000/.well-known/agent.json | jq '.skills'
```

You should see the `chat` skill listed:
```json
[
  {
    "id": "chat",
    "name": "Chat",
    "description": "Process messages using Claude"
  }
]
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `WINGMATE_LLM_MODEL` | Claude model to use | `claude-sonnet-4-20250514` |
| `WINGMATE_LLM_TIMEOUT` | Request timeout (Go duration) | `30s` |
| `WINGMATE_LLM_CLI_PATH` | Path to Claude CLI binary | Auto-detected from PATH |

**Example:**
```bash
export WINGMATE_LLM_MODEL="claude-sonnet-4-20250514"
export WINGMATE_LLM_TIMEOUT="60s"
./bin/wingmate --name my-agent --port 9000
```

### Config File

Add LLM settings to your agent config file:

```json
{
  "name": "my-agent",
  "port": 9000,
  "llm": {
    "model": "claude-sonnet-4-20250514",
    "timeout": "30s",
    "cliPath": "/usr/local/bin/claude"
  }
}
```

### Model Selection

Available models (check Claude documentation for current list):

| Model | Use Case |
|-------|----------|
| `claude-sonnet-4-20250514` | Balanced performance and cost (default) |
| `claude-opus-4-20250514` | Highest capability |
| `claude-haiku-3-5-20241022` | Fast responses, lower cost |

## How It Works

When Claude CLI is available:

1. **Agent Initialization**: `New()` checks if Claude CLI is installed
2. **Message Routing**: Non-ping messages route to `handleLLMMessage()`
3. **Session Management**: Conversation IDs map to Claude session IDs for continuity
4. **Flight Log**: All CLI requests and responses are logged

### Session Continuity

Wingmate maintains conversation context using session IDs:

- Each conversation gets a unique ID (from trace ID or generated)
- Session ID returned by Claude is stored in-memory
- Follow-up messages in the same conversation use the stored session ID

**Limitation**: Sessions are stored in-memory and lost on agent restart.

### Flight Log Entries

CLI interactions are logged to the Flight Log:

**Request (before CLI call):**
```json
{
  "type": "cli_request",
  "prompt": "What is Go?",
  "sessionID": ""
}
```

**Response (after CLI call):**
```json
{
  "type": "cli_response",
  "sessionID": "abc123",
  "inputTokens": 10,
  "outputTokens": 150,
  "model": "claude-sonnet-4-20250514",
  "durationMs": 2500
}
```

## Troubleshooting

### Error Codes

| Code | Name | Description | Solution |
|------|------|-------------|----------|
| 2001 | LLMUnavailable | Claude CLI not installed | Install CLI from [claude.ai/download](https://claude.ai/download) |
| 2002 | LLMExecutionFailed | CLI execution error | Check `claude --version`; verify authentication |
| 2003 | LLMResponseInvalid | Cannot parse CLI response | Update CLI to latest version |
| 2004 | LLMTimeout | Request timed out | Increase `WINGMATE_LLM_TIMEOUT` |
| 2005 | LLMAuthFailed | Authentication error | Run `claude` to re-authenticate |
| 2006 | LLMRateLimited | Rate limit exceeded | Wait and retry; check usage limits |

### Common Issues

**"Claude CLI not installed" (Error 2001)**

The agent couldn't find Claude CLI in the PATH.

1. Verify installation: `which claude` or `where claude` (Windows)
2. If installed elsewhere, set `WINGMATE_LLM_CLI_PATH`
3. Restart the agent after installing

**"Chat skill not appearing in Agent Card"**

The agent initializes LLM support at startup. If Claude CLI was installed after the agent started:

1. Stop the agent
2. Verify `claude --version` works
3. Restart the agent

**"Response parse error" (Error 2003)**

The CLI returned unexpected output format.

1. Update Claude CLI to the latest version
2. Verify `claude -p "test" --output-format json` returns valid JSON
3. Check for CLI error messages in the output

**Slow responses**

1. Check network connectivity
2. Increase timeout: `export WINGMATE_LLM_TIMEOUT="60s"`
3. Consider using a faster model like `claude-haiku-3-5-20241022`

**Session not maintaining context**

1. Ensure messages are sent within the same conversation (same trace ID)
2. Check Flight Log for session ID handling
3. Remember: sessions are lost on agent restart (in-memory storage)

## Security Considerations

- **Authentication**: Claude CLI handles authentication; no API keys in Wingmate config
- **Logging**: Prompts are logged to Flight Log; ensure log file permissions are appropriate
- **Network**: CLI communicates with Anthropic servers; ensure outbound HTTPS is allowed

## Related Documentation

- [README - LLM Support section](../../README.md#llm-support)
- [Architecture - Agent Component](../project-rules/architecture.md)
- [Claude CLI Documentation](https://docs.anthropic.com/claude-cli)
