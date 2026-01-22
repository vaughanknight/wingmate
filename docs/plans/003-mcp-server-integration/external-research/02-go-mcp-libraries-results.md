# Research Results: Go MCP Libraries Evaluation

**Source**: Perplexity Deep Research
**Date**: 2026-01-22
**Status**: Complete

---

# Go Libraries for Model Context Protocol Server Implementation

## Executive Summary

Three primary Go MCP implementations exist:

| Library | Stars | Status | Dependencies | Recommendation |
|---------|-------|--------|--------------|----------------|
| **modelcontextprotocol/go-sdk** (Official) | 3,100 | v1.1.0 stable | **Zero** (core package) | Best for minimal-deps projects |
| **mark3labs/mcp-go** | 7,500 | v0.43.0-beta | Minimal, bundled | Best for rapid development |
| **MegaGrindStone/go-mcp** | Lower | Pre-v1.0 | Minimal | Alternative architecture |

**Recommendation for Wingmate**: Use the **official modelcontextprotocol/go-sdk** - zero external dependencies in core package, type-safe API, and backed by Google/MCP team.

---

## 1. Official SDK: modelcontextprotocol/go-sdk

**Repository**: https://github.com/modelcontextprotocol/go-sdk

### Stats
- **Stars**: 3,100
- **Forks**: 277
- **Contributors**: 64
- **Used by**: 383 packages
- **Latest Release**: v1.1.0 (October 30, 2025)
- **Go Version**: 1.19+

### Key Features
- **Zero external dependencies** in core `mcp` package
- Type-safe API with automatic JSON Schema generation from Go structs
- Stdio and Streamable HTTP transport support
- Session-based client-server architecture
- Maintained by MCP team in collaboration with Google

### Code Example: wingmate_chat Tool

```go
package main

import (
    "context"
    "log"

    "github.com/modelcontextprotocol/go-sdk/mcp"
)

type WingmateChatInput struct {
    Prompt    string `json:"prompt" jsonschema:"The chat prompt to send"`
    SessionID string `json:"session_id,omitempty" jsonschema:"Optional session ID for conversation context"`
}

type WingmateChatOutput struct {
    Response  string `json:"response" jsonschema:"The chat response"`
    SessionID string `json:"session_id" jsonschema:"Session identifier"`
}

func WingmateChat(ctx context.Context, req *mcp.CallToolRequest, input WingmateChatInput) (
    *mcp.CallToolResult,
    WingmateChatOutput,
    error,
) {
    // Call your LLMExecutor here
    response := "Response to: " + input.Prompt

    return nil, WingmateChatOutput{
        Response:  response,
        SessionID: input.SessionID,
    }, nil
}

func main() {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "wingmate",
        Version: "v1.0.0",
    }, nil)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "wingmate_chat",
        Description: "Send a chat prompt to Wingmate and get a response",
    }, WingmateChat)

    if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
        log.Fatal(err)
    }
}
```

### Pros
- Zero dependencies (perfect for Wingmate)
- Type-safe with automatic schema generation
- Official, backed by MCP team + Google
- Clean separation of concerns (Server vs ServerSession)

### Cons
- Smaller community than mark3labs
- Less documentation/examples currently
- Some advanced features still evolving

---

## 2. Community SDK: mark3labs/mcp-go

**Repository**: https://github.com/mark3labs/mcp-go

### Stats
- **Stars**: 7,500
- **Forks**: 702
- **Contributors**: 141
- **Used by**: 2,200+ packages
- **Latest Release**: v0.43.0-beta.2 (October 25, 2025)
- **Go Version**: 1.16+

### Key Features
- Fluent builder API for tool definition
- More transports: stdio, SSE, Streamable HTTP
- Per-session tool customization
- Connection lost handler for reconnection logic
- Larger community and more examples

### Code Example: wingmate_chat Tool

```go
package main

import (
    "context"
    "fmt"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    s := server.NewMCPServer(
        "Wingmate",
        "1.0.0",
        server.WithToolCapabilities(true),
    )

    tool := mcp.NewTool("wingmate_chat",
        mcp.WithDescription("Send a chat prompt to Wingmate"),
        mcp.WithString("prompt",
            mcp.Required(),
            mcp.Description("The chat prompt to send"),
        ),
        mcp.WithString("session_id",
            mcp.Description("Optional session ID for conversation context"),
        ),
    )

    s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        prompt, err := request.RequireString("prompt")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        sessionID, _ := request.GetString("session_id")

        // Call your LLMExecutor here
        response := fmt.Sprintf("Response to: %s", prompt)

        return mcp.NewToolResultText(response), nil
    })

    if err := server.ServeStdio(s); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

### Pros
- Larger community (7,500 stars vs 3,100)
- More examples and documentation
- Fluent API reduces boilerplate
- Rich transport support

### Cons
- Beta status (v0.43.0-beta)
- Breaking changes possible (v0.29.0 had breaking changes)
- Slightly higher dependency footprint
- Pre-v1.0 stability concerns

---

## 3. Alternative: MegaGrindStone/go-mcp

**Repository**: https://github.com/MegaGrindStone/go-mcp

### Key Features
- Interface-based architecture
- Complete MCP protocol support
- Distinct tool/resource/prompt server interfaces
- Pre-v1.0, breaking changes in minor versions

### Code Example (Interface-based)

```go
type MyToolServer struct{}

func (s *MyToolServer) ListTools(ctx context.Context, params mcp.ListToolsParams,
    progress mcp.ProgressReporter, requestClient mcp.RequestClientFunc) (mcp.ListToolsResult, error) {
    return mcp.ListToolsResult{
        Tools: []mcp.Tool{
            {
                Name:        "wingmate_chat",
                Description: "Send a chat prompt",
            },
        },
    }, nil
}

func (s *MyToolServer) CallTool(ctx context.Context, params mcp.CallToolParams,
    progress mcp.ProgressReporter, requestClient mcp.RequestClientFunc) (mcp.CallToolResult, error) {
    return mcp.CallToolResult{
        Content: []mcp.Content{
            {Type: mcp.ContentTypeText, Text: "Response"},
        },
    }, nil
}
```

### Pros
- Flexible interface-based design
- Good for complex scenarios

### Cons
- Smaller community
- Pre-v1.0 stability
- Less documentation

---

## Build vs Buy Analysis

### Effort Comparison

| Approach | Lines of Code | Development Time |
|----------|---------------|------------------|
| From Scratch | 5,000-10,000 | Months |
| Official SDK | 50-100 | Hours |
| mark3labs | 50-100 | Hours |

**Conclusion**: Building from scratch provides **no meaningful benefit** given available libraries. The 50-100x complexity reduction justifies library adoption.

### What Building From Scratch Requires

1. **JSON-RPC 2.0 layer**: 500-1000 lines
2. **MCP protocol layer**: 2000-4000 lines
   - Initialization/capability negotiation
   - Tool discovery and invocation
   - Resource management
   - Notification support
3. **Transport layer**: 500-1000 lines
   - stdio message framing
   - Buffering and synchronization
4. **Testing**: Thousands of test cases
5. **Bug fixes**: Protocol subtleties discovered through usage

---

## Decision Matrix

| Criterion | Official SDK | mark3labs | From Scratch |
|-----------|-------------|-----------|--------------|
| Dependencies | **0** | Few | 0 |
| API Quality | Excellent | Excellent | Custom |
| Type Safety | **Strong** | Weak | Custom |
| Community | 3.1K stars | **7.5K stars** | N/A |
| Stability | **v1.1.0 stable** | v0.43 beta | Custom |
| stdio Support | Yes | Yes | Implement |
| Documentation | Good | **Better** | N/A |
| Effort | **Low** | **Low** | Very High |

---

## Recommendation for Wingmate

### Primary Choice: Official modelcontextprotocol/go-sdk

**Reasons**:
1. **Zero dependencies** - matches Wingmate's philosophy
2. **Type-safe API** - automatic schema from Go structs
3. **Stable release** - v1.1.0 (not beta)
4. **Official backing** - MCP team + Google
5. **Specification alignment** - canonical implementation

### Installation

```bash
go get github.com/modelcontextprotocol/go-sdk/mcp
```

### Integration with Wingmate's LLMExecutor

```go
package main

import (
    "context"
    "log"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/wingmate/wingmate/internal/llm"
)

type ChatInput struct {
    Prompt    string `json:"prompt" jsonschema:"The prompt to send to Claude CLI"`
    SessionID string `json:"session_id,omitempty" jsonschema:"Optional session ID"`
}

type ChatOutput struct {
    Result    string `json:"result"`
    SessionID string `json:"session_id"`
    Model     string `json:"model"`
    Duration  int64  `json:"duration_ms"`
}

func makeWingmateChat(executor llm.LLMExecutor) func(context.Context, *mcp.CallToolRequest, ChatInput) (*mcp.CallToolResult, ChatOutput, error) {
    return func(ctx context.Context, req *mcp.CallToolRequest, input ChatInput) (*mcp.CallToolResult, ChatOutput, error) {
        resp, err := executor.Execute(ctx, input.Prompt, input.SessionID)
        if err != nil {
            return &mcp.CallToolResult{IsError: true}, ChatOutput{}, err
        }

        return nil, ChatOutput{
            Result:    resp.Result,
            SessionID: resp.SessionID,
            Model:     resp.Model,
            Duration:  resp.Duration,
        }, nil
    }
}

func main() {
    executor := llm.NewCLIExecutor(llm.DefaultConfig())

    server := mcp.NewServer(&mcp.Implementation{
        Name:    "wingmate",
        Version: "1.0.0",
    }, nil)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "wingmate_chat",
        Description: "Send a prompt to Claude CLI and get a response",
    }, makeWingmateChat(executor))

    if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
        log.Fatal(err)
    }
}
```

---

## Citations

Key sources:
- https://github.com/modelcontextprotocol/go-sdk
- https://github.com/mark3labs/mcp-go
- https://github.com/MegaGrindStone/go-mcp
- https://modelcontextprotocol.io/specification/2025-06-18/basic/transports
