# Deep Research Prompt: Go MCP Libraries Evaluation

**Research Topic**: Evaluating Go Libraries for MCP Server Implementation
**Priority**: High (determines build vs buy decision)
**Created**: 2026-01-22

---

## 1. Clear Problem Definition

We need to implement an MCP (Model Context Protocol) server in Go. We have two main options:

1. **Use an existing Go MCP library** (e.g., `mark3labs/mcp-go`)
2. **Implement the protocol from specification** (zero external dependencies)

**Decision Needed**: Which approach is best for our project given:
- Current zero external dependencies in go.mod
- Need for stability and maintainability
- Desire for idiomatic Go patterns
- Single binary distribution requirement

**Current Knowledge Gap**: We don't know:
- What Go MCP libraries exist
- How mature/stable they are
- What their API looks like
- Whether they support our use case (stdio transport, tool definition)

---

## 2. Contextual Information

### Technology Stack
- **Language**: Go 1.21+
- **Target Protocol**: MCP (Model Context Protocol)
- **Transport**: stdio (stdin/stdout)
- **Build Constraint**: Single binary, cross-platform compilation

### Current go.mod (zero external deps)
```go
module github.com/wingmate/wingmate

go 1.21

// No external dependencies currently
```

### Our Requirements
- Define custom tools (specifically `wingmate_chat`)
- Handle tool invocations with context and cancellation
- Support session IDs for conversation continuity
- Log all operations to Flight Log
- Graceful shutdown support

### Existing Code We'll Integrate With
```go
// Interface to expose via MCP tool:
type LLMExecutor interface {
    Execute(ctx context.Context, prompt string, sessionID string) (*CLIResponse, error)
    IsInstalled() bool
}

// Pattern we use for servers:
type Server interface {
    Start(ctx context.Context) error
    Shutdown(ctx context.Context) error
}
```

---

## 3. Key Research Questions

### Library Discovery
1. **What Go MCP libraries currently exist?**
   - `mark3labs/mcp-go` - what is it?
   - Any other implementations?
   - Official vs community libraries?

2. **What is the maturity level of each library?**
   - GitHub stars, forks, watchers
   - Last commit date, commit frequency
   - Open issues count and types
   - Release versioning (stable releases?)
   - Documentation quality

### API Evaluation
3. **What does the tool definition API look like?**
   - How do you define a tool with input schema?
   - How do you register tool handlers?
   - How is tool invocation routed to handlers?

4. **Does the library support stdio transport?**
   - Is stdio built-in or must be implemented?
   - How is message framing handled?
   - Can you plug in custom transports?

5. **How does error handling work?**
   - MCP error code support
   - Custom error types
   - Error response formatting

### Integration Concerns
6. **How does it integrate with context.Context?**
   - Cancellation propagation
   - Timeout handling
   - Request-scoped values

7. **What is the testing story?**
   - Test helpers provided?
   - Mocking support?
   - Example test patterns?

8. **What are the dependencies?**
   - Transitive dependency count
   - Any problematic dependencies?
   - License compatibility (we're MIT-compatible)

---

## 4. Recommended Tools and Resources

### Libraries to Research
- `github.com/mark3labs/mcp-go` (primary candidate)
- Any forks or alternatives
- Official Anthropic Go SDK (if exists)

### Evaluation Criteria
| Criterion | Weight | Notes |
|-----------|--------|-------|
| Stability | High | Must not break in production |
| API Quality | High | Idiomatic Go, easy to use |
| Dependencies | Medium | Prefer minimal |
| Documentation | Medium | Examples, godoc |
| Maintenance | High | Active development |
| Community | Low | Nice to have |

### Sources to Check
- GitHub repository (stars, issues, PRs)
- pkg.go.dev documentation
- GitHub issues for known problems
- Any blog posts or tutorials

---

## 5. Practical Examples Requested

### For each library found, please provide:

#### A. Basic Server Setup
```go
// How do you create and start an MCP server?
// Show the minimal code to get a server running on stdio
```

#### B. Tool Definition
```go
// How do you define a tool with:
// - Name: "wingmate_chat"
// - Description: "Send a prompt to Claude CLI"
// - Input schema: { prompt: string (required), session_id: string (optional) }
// - Output: { result: string, session_id: string, model: string, duration_ms: int }
```

#### C. Tool Handler Implementation
```go
// How do you implement a tool handler that:
// - Receives the tool call
// - Extracts parameters
// - Calls our LLMExecutor
// - Returns the response
// - Handles errors
```

#### D. Graceful Shutdown
```go
// How do you:
// - Listen for shutdown signals
// - Wait for in-flight requests
// - Clean up resources
```

#### E. Testing
```go
// How do you test an MCP tool:
// - Unit test the tool handler
// - Integration test the full server
// - Mock the MCP client
```

---

## 6. Pitfalls and Mitigation

### Library-Specific Pitfalls
1. **What are known issues with each library?**
   - Open bugs
   - Missing features
   - Performance problems

2. **What are common integration mistakes?**
   - Misconfiguration
   - Incorrect tool schemas
   - Response format errors

### Build vs Buy Pitfalls
3. **Risks of using a library**
   - Dependency on maintainer
   - Breaking changes
   - Bloat from unused features

4. **Risks of implementing from scratch**
   - Protocol compliance bugs
   - Maintenance burden
   - Missing edge cases

### Go-Specific Pitfalls
5. **stdio handling in Go**
   - Buffering issues with os.Stdin/os.Stdout
   - Race conditions
   - Proper EOF handling

---

## 7. Integration Considerations

### Our Constraints
- **Single binary**: Library must compile into single executable
- **Cross-platform**: Must work on macOS, Linux, Windows
- **Zero-dependency preference**: Adding deps must be justified
- **Go 1.21+**: Can use modern Go features

### CI/CD Impact
- How does adding this dependency affect build times?
- Any special build flags needed?
- Vulnerability scanning considerations

### Future Maintenance
- If library is abandoned, can we fork/maintain?
- Is the codebase readable/modifiable?
- License allows modification?

---

## Comparison Framework

Please provide a comparison table:

| Aspect | mark3labs/mcp-go | [Other Lib] | From Scratch |
|--------|------------------|-------------|--------------|
| Maturity | ? | ? | N/A |
| Dependencies | ? | ? | 0 |
| API Quality | ? | ? | Custom |
| stdio Support | ? | ? | Implement |
| Documentation | ? | ? | N/A |
| Maintenance | ? | ? | Us |
| Effort to Integrate | ? | ? | High |

---

## Expected Output Format

1. **Library Inventory** - List of all Go MCP libraries found
2. **Detailed Evaluation** - For each library:
   - Repository stats
   - API overview with code examples
   - Pros and cons
   - Known issues
3. **Recommendation** - Which approach to take and why
4. **Implementation Guide** - How to integrate the recommended choice

---

## Success Criteria

This research is successful if we can:
- [ ] List all viable Go MCP libraries
- [ ] Evaluate each against our criteria
- [ ] See code examples for our specific use case
- [ ] Make an informed build vs buy decision
- [ ] Understand integration effort for chosen approach
