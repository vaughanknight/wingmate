# Deep Research Prompt: Claude Code MCP Configuration

**Research Topic**: Configuring Claude Code to Use Local MCP Servers
**Priority**: High (validates end-to-end user experience)
**Created**: 2026-01-22

---

## 1. Clear Problem Definition

We are building an MCP server (`wingmate mcp`) that users will add to Claude Code. We need to understand:

- **How to configure Claude Code** to recognize our MCP server
- **The exact configuration file format** and location
- **How to specify a local binary** as an MCP server command
- **What happens when things go wrong** (errors, timeouts, crashes)

**User Story**:
> As a user, I want to add Wingmate as an MCP server in Claude Code so I can use `wingmate_chat` tool directly from my Claude Code sessions without manual configuration of ports or protocols.

**Current Knowledge Gap**: We don't know:
- Where Claude Code stores MCP server configuration
- The exact JSON/YAML format for adding servers
- Whether stdio is the expected transport for local servers
- How Claude Code handles MCP server lifecycle (start, restart, crash recovery)

---

## 2. Contextual Information

### Our Setup
- **MCP Server Binary**: `wingmate` (Go binary)
- **MCP Subcommand**: `wingmate mcp`
- **Transport**: stdio (stdin/stdout)
- **Tool Exposed**: `wingmate_chat`
- **Platforms**: macOS, Linux, Windows

### What We're Building
```bash
# User installs wingmate binary (e.g., to /usr/local/bin/wingmate)
# User runs Claude Code
# Claude Code spawns: wingmate mcp
# User can then invoke wingmate_chat tool in Claude Code
```

### Tool Schema (for reference)
```json
{
  "name": "wingmate_chat",
  "description": "Send a prompt to Claude CLI and get a response",
  "inputSchema": {
    "type": "object",
    "properties": {
      "prompt": {
        "type": "string",
        "description": "The prompt to send"
      },
      "session_id": {
        "type": "string",
        "description": "Optional session ID for conversation continuity"
      }
    },
    "required": ["prompt"]
  }
}
```

---

## 3. Key Research Questions

### Configuration Location & Format
1. **Where does Claude Code store MCP server configuration?**
   - File path on macOS
   - File path on Linux
   - File path on Windows
   - Is it per-user or per-project?

2. **What is the exact configuration file format?**
   - JSON or YAML?
   - Full schema/structure
   - Required vs optional fields

3. **How do you add a stdio-based MCP server?**
   - Command specification format
   - Arguments handling
   - Working directory
   - Environment variables

### Server Lifecycle
4. **When does Claude Code start MCP servers?**
   - On Claude Code startup?
   - On first tool use?
   - Lazy initialization?

5. **How does Claude Code handle server crashes?**
   - Automatic restart?
   - Error reporting to user?
   - Retry policies?

6. **How are server timeouts handled?**
   - Default timeout values
   - Configurable timeouts?
   - Behavior on timeout

### Authentication & Security
7. **Are there authentication requirements for local MCP servers?**
   - Token-based auth?
   - Certificate requirements?
   - Or is local stdio trusted by default?

8. **Any sandboxing or permission considerations?**
   - File system access
   - Network access
   - Environment variable access

### Error Handling
9. **How are MCP errors displayed to users?**
   - Error message formatting
   - Tool call failure UI
   - Debugging information

10. **How do you troubleshoot MCP server issues?**
    - Log file locations
    - Debug mode
    - Diagnostic commands

---

## 4. Recommended Tools and Resources

### Primary Sources
- Claude Code official documentation
- Anthropic's MCP documentation
- Claude Code GitHub issues (if public)

### Configuration Examples to Find
- Example MCP server configurations
- Third-party MCP servers and their setup instructions
- Claude Code settings/preferences documentation

### Related Projects
- Other MCP servers (filesystem, git, etc.) and their setup guides
- How they document Claude Code integration

---

## 5. Practical Examples Requested

### A. Complete Configuration File Example
```json
// or yaml - whatever Claude Code uses
// Show the full configuration for adding a local MCP server
// with command: wingmate mcp
// name: wingmate
// description: Wingmate LLM integration
```

### B. Step-by-Step Setup Instructions
```
1. Install wingmate binary to [location]
2. Open Claude Code settings at [location]
3. Add the following configuration:
   [exact config]
4. Restart Claude Code (if needed?)
5. Verify tool appears in [where]
```

### C. Platform-Specific Variations
- macOS-specific paths or settings
- Linux-specific paths or settings
- Windows-specific paths or settings (especially binary extension)

### D. Environment Variable Passing
```json
// How to pass env vars to the MCP server
// e.g., WINGMATE_LLM_MODEL, WINGMATE_LLM_TIMEOUT
```

### E. Multiple MCP Servers
```json
// How to configure multiple MCP servers
// (user might have wingmate + other MCP servers)
```

---

## 6. Pitfalls and Mitigation

### Configuration Pitfalls
1. **Common configuration mistakes**
   - Wrong file format
   - Missing required fields
   - Path quoting issues on Windows
   - Permission problems

2. **Binary path issues**
   - Relative vs absolute paths
   - PATH resolution
   - Spaces in paths

### Runtime Pitfalls
3. **Server startup failures**
   - Binary not found
   - Permission denied
   - Port conflicts (if any)

4. **Communication issues**
   - stdio buffering problems
   - Message framing mismatches
   - Encoding issues (UTF-8?)

### User Experience Pitfalls
5. **Silent failures**
   - Server crashes without user notification
   - Tool not appearing without explanation
   - Misleading error messages

---

## 7. Integration Considerations

### Our Documentation Needs
We need to write a setup guide (`docs/how/mcp-setup.md`) that covers:
- Prerequisites (Claude Code installed, wingmate installed)
- Configuration steps
- Verification steps
- Troubleshooting

### Cross-Platform Support
- Configuration works on all platforms
- Binary naming (`wingmate` vs `wingmate.exe`)
- Path separators

### Version Compatibility
- Minimum Claude Code version required
- MCP protocol version compatibility
- How to handle version mismatches

### Update Workflow
- What happens when user updates wingmate binary?
- Does Claude Code need restart?
- Any cache clearing needed?

---

## Desired Configuration Template

Please provide a template we can include in our documentation:

```markdown
## Adding Wingmate to Claude Code

### Prerequisites
- Claude Code version X.Y.Z or later
- Wingmate installed at `/usr/local/bin/wingmate`

### Configuration

1. Open Claude Code settings file at:
   - macOS: `[path]`
   - Linux: `[path]`
   - Windows: `[path]`

2. Add the following to the `mcpServers` section:

\`\`\`json
[exact configuration]
\`\`\`

3. Restart Claude Code

### Verification

[How to verify it works]

### Troubleshooting

[Common issues and solutions]
```

---

## Expected Output Format

1. **Configuration Guide** - Step-by-step with exact file paths and formats
2. **Configuration Schema** - Full schema documentation
3. **Lifecycle Documentation** - How Claude Code manages MCP servers
4. **Error Handling** - What happens when things go wrong
5. **Troubleshooting Guide** - Common issues and solutions
6. **Template** - Ready-to-use documentation template

---

## Success Criteria

This research is successful if we can:
- [ ] Provide exact configuration file path for each platform
- [ ] Show complete configuration JSON/YAML for wingmate
- [ ] Document the server lifecycle (start, run, stop, crash)
- [ ] List troubleshooting steps for common issues
- [ ] Create a user-facing setup guide template
- [ ] Understand version requirements and compatibility
