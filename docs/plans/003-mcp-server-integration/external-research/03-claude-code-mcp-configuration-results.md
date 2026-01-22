# Research Results: Claude Code MCP Configuration

**Source**: Perplexity Deep Research
**Date**: 2026-01-22
**Status**: Complete

---

# Configuring Claude Code to Use Local MCP Servers

## Configuration File Locations

### Claude Code CLI Configuration

| Platform | Path | Notes |
|----------|------|-------|
| **macOS** | `~/.claude.json` | User's home directory |
| **Linux** | `~/.claude.json` | Same as macOS |
| **Windows** | `%USERPROFILE%\.claude.json` | User profile directory |
| **WSL** | `~/.claude.json` | Linux filesystem within WSL |

### Project-Scoped Configuration

- **File**: `.mcp.json` at project root
- **Purpose**: Team-shared MCP server configurations
- **Designed to be**: Committed to version control

### Managed Configuration (Enterprise)

| Platform | Path |
|----------|------|
| macOS | `/Library/Application Support/ClaudeCode/managed-settings.json` |
| Linux | `/etc/claude-code/managed-settings.json` |
| Windows | `C:\Program Files\ClaudeCode\managed-settings.json` |

---

## Configuration Format

### Complete Schema Structure

```json
{
  "mcpServers": {
    "server-identifier": {
      "command": "executable-name-or-path",
      "args": ["argument1", "argument2"],
      "env": {
        "ENVIRONMENT_VAR": "value"
      },
      "type": "stdio"
    }
  }
}
```

### Field Descriptions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | Yes | Executable path (absolute recommended) or command name in PATH |
| `args` | array | No | Command-line arguments passed in order |
| `env` | object | No | Environment variables for the server process |
| `type` | string | No | Transport type: `"stdio"` for local binaries |

### Environment Variable Expansion

Supported syntax in `command`, `args`, `env`, and `url` fields:
- `${VAR}` - expands to value of environment variable `VAR`
- `${VAR:-default}` - expands to `VAR` if set, otherwise uses `default`

---

## Complete Wingmate Configuration Examples

### Basic Configuration (in ~/.claude.json)

```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "type": "stdio"
    }
  }
}
```

### With Environment Variables

```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "env": {
        "WINGMATE_LLM_MODEL": "claude-sonnet-4-20250514",
        "WINGMATE_LLM_TIMEOUT": "120",
        "WINGMATE_DEBUG": "true"
      },
      "type": "stdio"
    }
  }
}
```

### Using Environment Variable Expansion

```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "env": {
        "WINGMATE_CONFIG": "${HOME}/.config/wingmate/settings.json",
        "API_KEY": "${ANTHROPIC_API_KEY}"
      },
      "type": "stdio"
    }
  }
}
```

### Project-Scoped (.mcp.json at project root)

```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "env": {
        "WINGMATE_PROJECT_PATH": "${PWD}"
      }
    }
  }
}
```

---

## Adding via CLI Command

You can also add MCP servers using the `claude mcp add` command:

```bash
# Add wingmate globally
claude mcp add wingmate /usr/local/bin/wingmate -- mcp

# Add with scope
claude mcp add --scope user wingmate /usr/local/bin/wingmate -- mcp
claude mcp add --scope project wingmate /usr/local/bin/wingmate -- mcp
```

---

## Server Lifecycle

### When Servers Start

- Claude Code starts MCP servers **during initialization** (on startup)
- Maintains persistent connections throughout the session
- Does NOT lazy-load servers on first tool use

### Initialization Sequence

1. Claude Code reads configuration from `~/.claude.json` and `.mcp.json`
2. Spawns each configured server process
3. Sends MCP `initialize` request with protocol version and capabilities
4. Server responds with its capabilities (tools, resources, prompts)
5. If successful, tools become available in the session

### Timeout Behavior

- **Default timeout**: 5 seconds for initialization
- If server doesn't respond within timeout: `Connection to MCP server 'wingmate' timed out after 5000ms`

### Crash Handling

- If server crashes during session: Claude Code detects disconnection and reports error
- **No automatic restart** within a session
- User must restart Claude Code to re-establish connection

---

## Error Handling and Debugging

### Common Error Messages

| Error | Cause |
|-------|-------|
| "Connection to MCP server 'wingmate' timed out after 5000ms" | Binary not found, can't execute, or slow startup |
| "spawn ... ENOENT" | Command path doesn't exist |
| "Permission denied" | Binary not executable |

### Log File Locations

| Platform | Path |
|----------|------|
| macOS | `~/Library/Logs/Claude/mcp-server-wingmate.log` |
| Linux | `~/.cache/claude-cli/mcp-server-wingmate.log` |
| Windows | `%APPDATA%\Claude\logs\mcp-server-wingmate.log` |

### Debug Mode

```bash
claude --mcp-debug
```

Shows detailed MCP initialization information.

### Diagnostic Commands

Within a Claude Code session:

| Command | Purpose |
|---------|---------|
| `/mcp` | List all configured servers and their connection status |
| `/doctor` | Comprehensive configuration validation |

From terminal:

```bash
claude mcp list          # List all configured MCP servers
claude mcp get wingmate  # Show specific server configuration
```

---

## Authentication

### Local MCP Servers

- **No authentication required** for local stdio-based servers
- Local servers are trusted by default (same-user process)
- Sensitive data should be passed via environment variables (not in config files)

### Project-Scoped Security

- Claude Code prompts for **user approval** before using servers from `.mcp.json`
- This prevents accidental execution of untrusted configurations

---

## Best Practices

### Path Specification

- **Always use absolute paths** for `command` field
- Avoid relative paths (`./wingmate`) - behavior depends on working directory
- Examples:
  - Good: `/usr/local/bin/wingmate`
  - Bad: `./wingmate`, `wingmate` (unless in PATH)

### macOS PATH Issues

GUI applications may not inherit shell PATH. Solutions:
1. Use absolute path in config: `/usr/local/bin/wingmate`
2. Symlink to PATH location: `ln -s ~/.local/bin/wingmate /usr/local/bin/wingmate`

### stdout/stderr Rules

**Critical for stdio servers**:
- **NEVER write to stdout** except for JSON-RPC protocol messages
- All logging MUST go to **stderr** or log files
- Extraneous stdout output breaks protocol communication

```go
// WRONG - breaks MCP protocol
fmt.Println("Debug message")

// CORRECT - log to stderr
log.SetOutput(os.Stderr)
log.Println("Debug message")
```

### Secrets Management

For sensitive values (API keys, passwords):
1. Don't hardcode in config files
2. Use environment variable expansion: `"${API_KEY}"`
3. Set actual values in shell profile (`.bashrc`, `.zshrc`)

---

## Complete Setup Workflow

### Step 1: Install Binary

```bash
# Build and install wingmate
go build -o /usr/local/bin/wingmate ./cmd/wingmate

# Verify it works
/usr/local/bin/wingmate mcp
# Should start MCP server (Ctrl+C to exit)
```

### Step 2: Add Configuration

Edit `~/.claude.json`:

```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "type": "stdio"
    }
  }
}
```

### Step 3: Verify Configuration

```bash
# Check configuration is recognized
claude mcp list
# Should show: wingmate

claude mcp get wingmate
# Should show the configuration details
```

### Step 4: Test Connection

```bash
# Start Claude Code
claude

# Within session, run:
/mcp
# Should show wingmate with "connected" status
```

### Step 5: Use the Tool

Ask Claude Code to use the wingmate_chat tool:
> "Use wingmate_chat to ask about Go programming"

---

## User Documentation Template

Use this template in your `docs/how/mcp-setup.md`:

```markdown
## Adding Wingmate to Claude Code

### Prerequisites
- Claude Code installed and configured
- Wingmate binary installed at `/usr/local/bin/wingmate`

### Configuration

1. Open your Claude Code configuration file:
   - macOS/Linux: `~/.claude.json`
   - Windows: `%USERPROFILE%\.claude.json`

2. Add the following to the `mcpServers` section:

```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "type": "stdio"
    }
  }
}
```

3. Restart Claude Code if it's running

### Verification

1. Run `claude mcp list` - wingmate should appear
2. Start Claude Code: `claude`
3. Run `/mcp` command - wingmate should show "connected"

### Troubleshooting

**Server not appearing:**
- Verify JSON syntax is valid
- Check path exists: `ls -la /usr/local/bin/wingmate`
- Test binary: `/usr/local/bin/wingmate mcp`

**Connection timeout:**
- Check logs: `~/.cache/claude-cli/mcp-server-wingmate.log`
- Run with debug: `claude --mcp-debug`

**Tool not working:**
- Ensure Claude CLI is installed: `claude --version`
- Check environment variables are set
```

---

## Citations

Key sources:
- https://code.claude.com/docs/en/mcp
- https://code.claude.com/docs/en/settings
- https://modelcontextprotocol.io/docs/develop/connect-local-servers
- https://scottspence.com/posts/configuring-mcp-tools-in-claude-code
- https://modelcontextprotocol.io/docs/develop/build-server
