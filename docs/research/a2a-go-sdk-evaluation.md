# A2A Go SDK Evaluation

**Date**: 2026-01-21
**SDK**: `github.com/a2aproject/a2a-go`
**Version**: v0.3.4 (latest as of evaluation)

---

## Executive Summary

**Decision**: Implement custom types with SDK as reference

**Rationale**: The SDK provides types and utilities, but requires Go 1.24.4 minimum (we target Go 1.21+ per ADR-001). Additionally, implementing our own types gives us full control over the JSON serialization and allows us to match the exact wire format needed without SDK version coupling.

---

## SDK Overview

### Types Provided

| Type | Description | Usability |
|------|-------------|-----------|
| `AgentCard` | Agent capability advertisement | Good reference, but tightly coupled to SDK |
| `Message` | Communication unit with roles/parts | Good reference |
| `MessageSendParams` | Parameters for sending | Useful pattern |
| `TextPart` | Message content component | Simple, we can implement |

### Utilities Offered

| Utility | Description | Verdict |
|---------|-------------|---------|
| `a2asrv.NewHandler()` | Transport-agnostic request handler | Overkill for MVP |
| `a2asrv.NewJSONRPCHandler()` | JSON-RPC transport | Could use, but simple to implement |
| `a2agrpc.NewHandler()` | gRPC transport | Not needed for MVP |
| `a2aclient.NewFromCard()` | Client from AgentCard | Useful pattern |

### Requirements

- **Go Version**: 1.24.4 minimum (BLOCKER: we target Go 1.21+)
- **Dependencies**: Pulls in gRPC and other heavy dependencies

---

## Evaluation Criteria

| Criterion | Score | Notes |
|-----------|-------|-------|
| Go version compatibility | ❌ | Requires 1.24.4, we target 1.21+ |
| Type completeness | ✅ | Has all A2A types we need |
| Minimal dependencies | ❌ | Pulls gRPC, heavy for MVP |
| Wire format control | ⚠️ | Good, but coupled to SDK updates |
| Learning/reference value | ✅ | Excellent reference for correct types |

---

## Decision

**Proceed with custom implementation** using SDK as reference material.

### Reasons

1. **Go Version**: SDK requires Go 1.24.4; we want Go 1.21+ compatibility (per ADR-001)
2. **Dependency Weight**: SDK pulls gRPC and other deps we don't need for HTTP-only MVP
3. **Control**: Custom types let us control JSON tags, omitempty behavior, validation
4. **Simplicity**: A2A is JSON-RPC 2.0 over HTTP - straightforward to implement
5. **Decoupling**: No SDK version drift concerns

### What We'll Use from SDK

- **Type structure reference**: Field names, JSON tags, required vs optional
- **Protocol patterns**: Request/response shapes, error codes
- **Agent Card schema**: Exact fields and their purposes

---

## Type Implementation Plan

### AgentCard (from SDK reference)

```go
type AgentCard struct {
    Name         string       `json:"name"`
    Description  string       `json:"description,omitempty"`
    URL          string       `json:"url"`
    Version      string       `json:"version"`
    Capabilities Capabilities `json:"capabilities"`
    Skills       []Skill      `json:"skills"`
    // Authentication omitted for MVP (per DEV-002)
}

type Capabilities struct {
    Streaming         bool `json:"streaming"`
    PushNotifications bool `json:"pushNotifications"`
}

type Skill struct {
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}
```

### Message (from SDK reference)

```go
type Message struct {
    Role  string `json:"role"`
    Parts []Part `json:"parts"`
}

// Part uses discriminated union via "kind" field
type Part struct {
    Kind string `json:"kind"` // "text", "data", "file"
    // Union fields - only one populated based on Kind
    Text     string                 `json:"text,omitempty"`
    Data     map[string]interface{} `json:"data,omitempty"`
    FilePath string                 `json:"filePath,omitempty"`
    MimeType string                 `json:"mimeType,omitempty"`
}
```

---

## References

- SDK Repository: https://github.com/a2aproject/a2a-go
- A2A Specification: https://github.com/a2aproject/A2A/blob/main/specification/json/a2a.json
- ADR-001: Language Choice (Go 1.21+ requirement)
