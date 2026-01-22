# MCP Server Setup Guide

This guide covers setting up Wingmate as an MCP (Model Context Protocol) server for Claude Code integration.

## Overview

Wingmate can run as an MCP server, exposing its capabilities as tools that Claude Code can invoke directly. This enables Claude Code to:

- Send chat messages through Wingmate's LLM integration
- Check Wingmate's operational status
- Discover available Wingmate agents on the network

## Prerequisites

- Go 1.21 or later (for building from source)
- Claude Code CLI installed and authenticated
- Internet connection for Claude API access

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/wingmate/wingmate.git
cd wingmate

# Build the binary
go build -o wingmate ./cmd/wingmate

# Verify the build
./wingmate --help | grep mcp
```

You should see:
```
  mcp                 Start MCP server (stdio transport for Claude Code)
```

### Option 2: Download Pre-built Binary

Download the latest release for your platform from the [releases page](https://github.com/wingmate/wingmate/releases).

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/wingmate/wingmate/releases/latest/download/wingmate-darwin-arm64 -o wingmate
chmod +x wingmate
```

**macOS (Intel):**
```bash
curl -L https://github.com/wingmate/wingmate/releases/latest/download/wingmate-darwin-amd64 -o wingmate
chmod +x wingmate
```

**Linux (x64):**
```bash
curl -L https://github.com/wingmate/wingmate/releases/latest/download/wingmate-linux-amd64 -o wingmate
chmod +x wingmate
```

**Windows:**
Download `wingmate-windows-amd64.exe` from the releases page.

### Install to PATH

Move the binary to a directory in your PATH:

**macOS / Linux:**
```bash
sudo mv wingmate /usr/local/bin/
# Or for user-local installation:
mkdir -p ~/.local/bin
mv wingmate ~/.local/bin/
# Add to PATH if needed: export PATH="$HOME/.local/bin:$PATH"
```

**Windows:**
Move `wingmate.exe` to a directory in your PATH, or add its location to PATH.

## Configuration

### Claude Code Configuration

Add Wingmate as an MCP server in your Claude Code configuration file.

**Location:**
- macOS/Linux: `~/.claude.json`
- Windows: `%USERPROFILE%\.claude.json`

**Configuration:**
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

**With verbose logging (for debugging):**
```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp", "--verbose"],
      "type": "stdio"
    }
  }
}
```

**Windows example:**
```json
{
  "mcpServers": {
    "wingmate": {
      "command": "C:\\Users\\YourName\\bin\\wingmate.exe",
      "args": ["mcp"],
      "type": "stdio"
    }
  }
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `WINGMATE_LLM_MODEL` | Claude model for chat tool | CLI default |
| `WINGMATE_LLM_TIMEOUT` | Request timeout in seconds | `120` |

**Using environment variables in Claude Code config:**
```json
{
  "mcpServers": {
    "wingmate": {
      "command": "/usr/local/bin/wingmate",
      "args": ["mcp"],
      "type": "stdio",
      "env": {
        "WINGMATE_LLM_MODEL": "claude-sonnet-4-20250514",
        "WINGMATE_LLM_TIMEOUT": "180"
      }
    }
  }
}
```

## Available Tools

When connected, Wingmate exposes three MCP tools:

### wingmate_chat

Send a chat message through Wingmate's LLM integration.

**Schema:**
```json
{
  "name": "wingmate_chat",
  "description": "Send a chat message through Wingmate",
  "inputSchema": {
    "type": "object",
    "properties": {
      "message": {
        "type": "string",
        "description": "The message to send"
      },
      "session_id": {
        "type": "string",
        "description": "Optional session ID for conversation continuity"
      }
    },
    "required": ["message"]
  }
}
```

### wingmate_status

Get Wingmate's current operational status.

**Schema:**
```json
{
  "name": "wingmate_status",
  "description": "Get Wingmate status and health information",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

**Response includes:**
- Server uptime
- LLM availability
- Configuration details

### wingmate_discover

Discover available Wingmate agents on the network.

**Schema:**
```json
{
  "name": "wingmate_discover",
  "description": "Discover available Wingmate agents",
  "inputSchema": {
    "type": "object",
    "properties": {
      "timeout": {
        "type": "integer",
        "description": "Discovery timeout in seconds",
        "default": 5
      }
    }
  }
}
```

## Verification

### Step 1: Test Manual Startup

Before configuring Claude Code, verify Wingmate MCP works standalone:

```bash
# Start MCP server (will wait for input on stdin)
wingmate mcp --verbose
```

Press `Ctrl+C` to stop. You should see a clean shutdown with no errors.

### Step 2: Restart Claude Code

After updating `~/.claude.json`:

1. Completely quit Claude Code
2. Reopen Claude Code
3. The MCP server starts automatically

### Step 3: Verify Tool Availability

In Claude Code, the `wingmate_chat`, `wingmate_status`, and `wingmate_discover` tools should now be available. You can verify by asking Claude Code to use them:

> "Use the wingmate_status tool to check Wingmate's status"

## How It Works

### Protocol

Wingmate implements the MCP protocol over stdio transport:

1. **Transport**: Newline-delimited JSON (NDJSON) over stdin/stdout
2. **Protocol**: JSON-RPC 2.0
3. **Lifecycle**: Claude Code starts the process; process runs until terminated

### Message Flow

```
┌──────────────┐        stdin (JSON-RPC)       ┌──────────────┐
│              │ ───────────────────────────▶  │              │
│  Claude Code │                               │   Wingmate   │
│              │ ◀───────────────────────────  │   MCP Server │
└──────────────┘        stdout (JSON-RPC)      └──────────────┘
                                                     │
                                                     │ (for chat tool)
                                                     ▼
                                               ┌──────────────┐
                                               │  Claude CLI  │
                                               └──────────────┘
```

### Flight Log

All MCP interactions are logged to the Flight Log (`./flight.jsonl` by default):

```json
{
  "timestamp": "2026-01-22T10:30:00Z",
  "trace_id": "abc123",
  "agent": "wingmate-mcp",
  "direction": "outbound",
  "summary": "chat response",
  "payload": { "tool": "wingmate_chat", "status": "success" }
}
```

## Troubleshooting

### Error Codes

| Code | Name | Description | Solution |
|------|------|-------------|----------|
| 3001 | MCPInvalidRequest | Malformed JSON-RPC request | Check request format |
| 3002 | MCPMethodNotFound | Unknown method called | Verify tool name |
| 3003 | MCPInvalidParams | Invalid tool parameters | Check parameter schema |
| 3004 | MCPInternalError | Server internal error | Check logs, restart server |
| 3010 | MCPToolNotFound | Tool not registered | Restart MCP server |
| 3011 | MCPToolExecutionFailed | Tool handler error | Check specific tool error |
| 3020 | MCPTransportError | stdin/stdout error | Restart Claude Code |
| 3021 | MCPParseError | Cannot parse JSON | Check for malformed messages |
| 3022 | MCPShutdown | Server shutting down | Normal during shutdown |

### Common Issues

**Tool not appearing in Claude Code**

1. Verify config file location: `~/.claude.json`
2. Verify JSON syntax is valid
3. Verify binary path is correct and executable
4. Completely restart Claude Code (not just reload)

**"Command not found" error**

1. Use absolute path in config: `/usr/local/bin/wingmate`
2. Verify binary exists: `ls -la /usr/local/bin/wingmate`
3. Verify binary is executable: `chmod +x /usr/local/bin/wingmate`

**Tool calls timing out**

1. Increase `WINGMATE_LLM_TIMEOUT` environment variable
2. Check network connectivity to Claude API
3. Try verbose mode to see where it's hanging

**Flight Log not being written**

1. Check write permissions in current working directory
2. Specify log path: `wingmate mcp --log /path/to/flight.jsonl`
3. Check disk space

**Verbose output interfering with MCP**

Verbose output goes to stderr (not stdout), so it should not interfere with MCP transport. If you see issues:

1. Remove `--verbose` flag from production config
2. Use verbose only for debugging

### Debug Mode

For detailed debugging, run with verbose logging:

```bash
# In one terminal, run MCP manually
wingmate mcp --verbose 2>mcp-debug.log

# Check the debug log
tail -f mcp-debug.log
```

### Log Locations

| Log | Location | Purpose |
|-----|----------|---------|
| Flight Log | `./flight.jsonl` | MCP tool invocations |
| Verbose Log | stderr | Debug output (when `--verbose`) |
| Claude Code Logs | Platform-specific | MCP client errors |

## Security Considerations

- **No API keys stored**: Wingmate uses Claude CLI which handles authentication
- **Local execution**: MCP runs as a local process, no network exposure
- **Flight Log**: Tool invocations are logged; ensure appropriate file permissions
- **Process isolation**: Wingmate runs as a subprocess of Claude Code

## Platform-Specific Notes

### macOS

- Binary may need to be allowed in System Preferences > Security & Privacy
- Use absolute paths in `~/.claude.json`

### Linux

- Ensure binary has execute permissions: `chmod +x wingmate`
- For system-wide installation: `/usr/local/bin/wingmate`
- For user installation: `~/.local/bin/wingmate`

### Windows

- Use forward slashes or escaped backslashes in JSON paths
- Add `.exe` extension to command
- Example path: `C:/Users/Name/bin/wingmate.exe` or `C:\\Users\\Name\\bin\\wingmate.exe`

## Related Documentation

- [LLM Setup Guide](./llm-setup.md) - Claude CLI integration for agents
- [README - Quick Start](../../README.md)
- [CLAUDE.md - Project Context](../../CLAUDE.md)
- [MCP Protocol Specification](https://modelcontextprotocol.io/specification)
