# Phase 4: System Prompt & Specialization — Task Dossier

**Plan**: [005 - Peer Delegation](../../peer-delegation-plan.md)
**Phase**: 4 of 5
**Status**: COMPLETE
**Prior Phase**: Phase 3 (Delegation Tool) — COMPLETE
**Created**: 2026-01-28

---

## Executive Briefing

Phase 4 wires `Config.Claude.SystemPrompt` into the Claude CLI invocation so that specialized agents respond in character when receiving A2A messages. The system prompt field already exists in config with env var support (`WINGMATE_CLAUDE_SYSTEM_PROMPT`) and defaults, but is never passed to the CLI executor. This phase adds `WithSystemPrompt` option to `CLIExecutor`, passes it during agent construction, and adds `--system-prompt` to the CLI args when non-empty.

**Key Risks**:
- Claude CLI `--system-prompt` flag may not exist or may use a different name. Verify via `claude --help`.
- Changing `buildArgs()` affects all CLI invocations — regression risk is real.

---

## Objectives & Scope

### Objectives

1. Add `WithSystemPrompt` option to `CLIExecutor`
2. Wire `--system-prompt` flag into `buildArgs()` when system prompt is non-empty
3. Pass `config.Claude.SystemPrompt` to executor during `Agent.New()` construction
4. Verify empty system prompt preserves existing behavior (no extra flag)

### Out of Scope

- Changing the `LLMExecutor` interface signature
- Per-message system prompt override
- System prompt for MCP chat tool (already handled separately if needed)

---

## Tasks

| ID | Status | Task | CS | File(s) | Depends | Success Criteria |
|----|--------|------|----|---------|---------|------------------|
| T001 | [x] | Write tests for WithSystemPrompt option | 1 | `internal/llm/cli_test.go` | — | Option sets systemPrompt field on executor |
| T002 | [x] | Add WithSystemPrompt option | 1 | `internal/llm/cli.go` | T001 | Option compiles and sets field |
| T003 | [x] | Write tests for buildArgs with system prompt | 2 | `internal/llm/cli_test.go` | T002 | Args include `--system-prompt` when set, omit when empty |
| T004 | [x] | Wire system prompt into buildArgs | 1 | `internal/llm/cli.go` | T003 | `--system-prompt` flag appended when systemPrompt non-empty |
| T005 | [x] | Wire config.Claude.SystemPrompt into Agent.New() | 1 | `internal/agent/agent.go` | T004 | SystemPrompt option passed to NewCLIExecutor |
| T006 | [x] | Write agent test verifying system prompt wiring | 2 | `internal/agent/agent_test.go` | T005 | Agent constructed with system prompt passes it to executor |
| T007 | [x] | Run full test suite | 1 | — | T006 | `go test ./... -race` passes |

---

## Alignment Brief

### Relevant Findings

| Finding | Summary | How This Phase Addresses It |
|---------|---------|---------------------------|
| 12 (MEDIUM) | SystemPrompt not used in A2A handling | T004/T005 wire system prompt from config into CLI invocation |

### Constitution Alignment

| Principle | Compliance |
|-----------|-----------|
| P1 (Observability) | System prompt already logged in CLI request entry (prompt field) |
| P4 (Protocol Compliance) | No protocol changes; internal CLI flag addition |
| P5 (Themed but Tasteful) | Uses existing config field name |

### Doctrine Compliance

| Rule | Compliance |
|------|-----------|
| TDD | Tests first (T001, T003, T006) before implementation |
| ADR-001 | Go, single binary — no new dependencies |

---

## Verification Commands

```bash
# Unit tests
go test ./internal/llm/ -run "TestCLIExecutor_BuildArgs|TestCLIExecutor_WithSystemPrompt" -v
go test ./internal/agent/ -run "TestAgent.*SystemPrompt" -v

# Full regression
go test ./... -race
```
