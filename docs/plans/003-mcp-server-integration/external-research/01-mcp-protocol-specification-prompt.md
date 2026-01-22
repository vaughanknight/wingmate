# Deep Research Prompt: MCP Protocol Specification

**Research Topic**: Model Context Protocol (MCP) Server Implementation Details
**Priority**: High (blocks architecture decisions)
**Created**: 2026-01-22

---

## 1. Clear Problem Definition

We are implementing an MCP (Model Context Protocol) server in Go as part of the Wingmate project. The server needs to:

- Run over **stdio transport** (not HTTP)
- Expose a `wingmate_chat` tool that invokes Claude CLI
- Handle the complete MCP lifecycle (initialize → list tools → invoke → shutdown)
- Follow the JSON-RPC 2.0 message format that MCP uses

**Current Gap**: We understand MCP conceptually but lack precise knowledge of:
- The exact wire protocol format for tool registration
- Required capabilities to declare during initialization
- The complete request/response lifecycle for tool invocation
- How stdio transport specifically works (message framing, delimiters)

**No error messages yet** - this is pre-implementation research.

---

## 2. Contextual Information

### Technology Stack
- **Language**: Go 1.21+
- **Target Protocol**: Model Context Protocol (MCP)
- **Transport**: stdio (stdin/stdout)
- **Wire Format**: JSON-RPC 2.0
- **Client**: Claude Code (Anthropic's CLI tool)

### Project Context
- Wingmate is an A2A (Agent-to-Agent) communication tool
- Currently has zero external dependencies (we prefer to maintain this if possible)
- Existing patterns: HTTP server with handler registration, graceful shutdown, error code ranges
- Constitution principle P4 sanctions MCP for local tool integration

### Relevant Existing Code Patterns
```go
// We have this interface we want to expose via MCP:
type LLMExecutor interface {
    Execute(ctx context.Context, prompt string, sessionID string) (*CLIResponse, error)
    IsInstalled() bool
}

// Response structure:
type CLIResponse struct {
    Result    string `json:"result"`
    SessionID string `json:"session_id"`
    Model     string `json:"model"`
    Duration  int64  `json:"duration_ms"`
}
```

---

## 3. Key Research Questions

### Protocol Fundamentals
1. **What is the exact JSON-RPC message format for MCP?**
   - Request structure (method names, params format)
   - Response structure (result format, error format)
   - Notification vs request distinction

2. **How does stdio transport work for MCP servers?**
   - Message framing (newline-delimited? length-prefixed?)
   - Reading from stdin / writing to stdout
   - Handling partial reads and buffering
   - Concurrent request handling

3. **What is the MCP server initialization handshake?**
   - What methods are called during `initialize`?
   - What capabilities must a server declare?
   - What client info is received?
   - Version negotiation process

### Tool Registration
4. **How do MCP servers register tools?**
   - Format of `tools/list` response
   - Tool schema definition (JSON Schema for inputs?)
   - Tool metadata (name, description, annotations)

5. **What is the tool invocation protocol?**
   - Request format for `tools/call`
   - Response format (success vs error)
   - Progress reporting (if supported)
   - Cancellation handling

### Lifecycle Management
6. **What is the complete MCP server lifecycle?**
   - Startup sequence
   - Steady-state operation
   - Shutdown sequence (graceful vs immediate)
   - Error recovery

7. **How should servers handle protocol errors?**
   - Invalid JSON
   - Unknown methods
   - Invalid parameters
   - Timeout handling

---

## 4. Recommended Tools and Resources

### Primary Sources to Research
- **MCP Specification**: https://spec.modelcontextprotocol.io/ (official spec)
- **MCP GitHub**: https://github.com/modelcontextprotocol (reference implementations)
- **JSON-RPC 2.0 Spec**: https://www.jsonrpc.org/specification

### Reference Implementations to Study
- TypeScript/JavaScript MCP SDK (official)
- Python MCP SDK (official)
- Any Go implementations (community)

### Specific Areas to Investigate
- `@modelcontextprotocol/sdk` package structure
- Example MCP servers (filesystem, git, etc.)
- stdio transport implementation details

---

## 5. Practical Examples Requested

Please provide:

### A. Complete Initialization Exchange
```
CLIENT → SERVER: initialize request
SERVER → CLIENT: initialize response
CLIENT → SERVER: initialized notification
```
With exact JSON payloads.

### B. Tool Registration Example
```
CLIENT → SERVER: tools/list request
SERVER → CLIENT: tools/list response
```
Showing a tool with:
- Name: `wingmate_chat`
- Description: "Send a prompt to Claude CLI"
- Input schema with `prompt` (required) and `session_id` (optional)

### C. Tool Invocation Example
```
CLIENT → SERVER: tools/call request
SERVER → CLIENT: tools/call response
```
For both success and error cases.

### D. Stdio Message Framing
Show exactly how messages are delimited on stdio:
- Is it newline-delimited JSON?
- Is there a Content-Length header like LSP?
- How are multiple concurrent requests handled?

---

## 6. Pitfalls and Mitigation

Please identify:

1. **Common MCP server implementation mistakes**
   - Protocol violations that cause client rejection
   - Capability declaration errors
   - Response format issues

2. **stdio transport gotchas**
   - Buffering issues
   - Blocking read problems
   - Mixed stdout/stderr handling
   - Signal handling during I/O

3. **JSON-RPC pitfalls**
   - ID handling (string vs number)
   - Null vs missing fields
   - Error code conventions

4. **Testing challenges**
   - How to test stdio servers
   - Mocking MCP clients
   - Integration testing approaches

---

## 7. Integration Considerations

### Claude Code Compatibility
- What MCP version does Claude Code expect?
- Are there Claude Code-specific requirements?
- How does Claude Code handle server errors/crashes?

### Go-Specific Considerations
- Recommended Go patterns for stdio servers
- Goroutine management for concurrent requests
- Context cancellation propagation
- Graceful shutdown with pending requests

### Our Existing Architecture
- We use context.Context for cancellation and tracing
- We have a Flight Log that needs MCP entries
- We use error code ranges (3xxx reserved for MCP)
- We prefer interface-first design

---

## Expected Output Format

Please structure your response as:

1. **Protocol Overview** - High-level MCP architecture
2. **Wire Format** - Exact JSON structures with examples
3. **Lifecycle** - Step-by-step server operation
4. **Code Examples** - Go-idiomatic implementation patterns
5. **Pitfalls** - Common mistakes and how to avoid them
6. **Testing** - How to verify correct implementation

---

## Success Criteria

This research is successful if we can:
- [ ] Implement MCP server initialization handshake
- [ ] Register the `wingmate_chat` tool correctly
- [ ] Handle tool invocations with proper response format
- [ ] Implement stdio transport correctly
- [ ] Handle errors according to MCP spec
- [ ] Gracefully shut down the server
