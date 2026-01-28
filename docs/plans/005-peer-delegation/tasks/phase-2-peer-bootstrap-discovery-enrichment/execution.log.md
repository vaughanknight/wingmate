# Phase 2: Peer Bootstrap & Discovery Enrichment — Execution Log

**Plan**: 005 - Peer Delegation
**Phase**: 2 of 5
**Started**: 2026-01-28

---

## Task T001: Write tests for PeerInfo struct and extended PeerProvider
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
Wrote 3 tests in `handlers_test.go`: TestPeerInfo_Fields, TestPeerInfo_JSON (with omitempty verification), TestMockRichPeerProvider_GetPeerInfo. Also created `mockRichPeerProvider` that implements extended PeerProvider.

### Evidence
Tests fail to compile (RED) — `undefined: PeerInfo`, `undefined: PeerSkill`. Confirmed no false passes.

### Files Changed
- `internal/mcp/handlers_test.go` — Added PeerInfo tests and mockRichPeerProvider

**Completed**: 2026-01-28

---

## Task T002: Define PeerInfo struct and update PeerProvider interface
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
1. Added `PeerSkill` struct (Name, Description)
2. Added `PeerInfo` struct (Name, URL, Description, Skills, Available)
3. Added `GetPeerInfo() []PeerInfo` to `PeerProvider` interface
4. Updated existing `mockPeerProvider` to satisfy extended interface

### Evidence
```
=== RUN   TestPeerInfo_Fields
--- PASS: TestPeerInfo_Fields (0.00s)
=== RUN   TestPeerInfo_JSON
--- PASS: TestPeerInfo_JSON (0.00s)
=== RUN   TestMockRichPeerProvider_GetPeerInfo
--- PASS: TestMockRichPeerProvider_GetPeerInfo (0.00s)
PASS
```

### Files Changed
- `internal/mcp/handlers.go` — Added PeerSkill, PeerInfo, GetPeerInfo to PeerProvider
- `internal/mcp/handlers_test.go` — Updated mockPeerProvider with GetPeerInfo

**Completed**: 2026-01-28

---

## Task T003: Write tests for Agent.GetPeerInfo()
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
Wrote 3 tests: TestAgent_GetPeerInfo_Empty, TestAgent_GetPeerInfo_Populated (verifies AgentCard → PeerInfo conversion), TestAgent_GetPeerInfo_ThreadSafe (concurrent read/write with race detector).

### Evidence
All tests pass immediately (GREEN) because GetPeerInfo was implemented alongside T002 to satisfy the PeerProvider interface compile check.

### Files Changed
- `internal/agent/agent_test.go` — Added 3 GetPeerInfo tests

**Completed**: 2026-01-28

---

## Task T004: Implement Agent.GetPeerInfo()
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
Added `GetPeerInfo() []mcp.PeerInfo` method to Agent. Reads from knownPeers under RLock, converts AgentCard fields (Name, URL, Description, Skills) to PeerInfo. All known peers marked Available=true.

### Evidence
```
=== RUN   TestAgent_GetPeerInfo_Empty
--- PASS
=== RUN   TestAgent_GetPeerInfo_Populated
--- PASS
=== RUN   TestAgent_GetPeerInfo_ThreadSafe
--- PASS (race detector clean)
```

### Files Changed
- `internal/agent/agent.go` — Added GetPeerInfo() method

**Completed**: 2026-01-28

---

## Task T005: Write tests for probePeers()
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
Wrote 3 tests: TestAgent_ProbePeers_PopulatesKnownPeers (two agents, peer probed on startup), TestAgent_ProbePeers_UnreachablePeer (bad URL, no crash), TestAgent_ProbePeers_NoPeers (empty peers list).

### Evidence
```
=== RUN   TestAgent_ProbePeers_PopulatesKnownPeers
    agent_test.go:577: GetPeerInfo() len = 0, want 1
--- FAIL
```
RED confirmed — knownPeers empty because probePeers not called yet.

### Files Changed
- `internal/agent/agent_test.go` — Added 3 probePeers tests

**Completed**: 2026-01-28

---

## Task T006: Implement probePeers() in Agent.Start()
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
1. Added `probePeers(ctx)` method: iterates config.Peers (under mu.RLock for thread safety), calls GetPeerCard for each, logs warnings for failures
2. Added `go a.probePeers(ctx)` call after `close(a.ready)` in Start()

### Evidence
```
=== RUN   TestAgent_ProbePeers_PopulatesKnownPeers
--- PASS (0.52s)
=== RUN   TestAgent_ProbePeers_UnreachablePeer
--- PASS (0.53s)
=== RUN   TestAgent_ProbePeers_NoPeers
--- PASS (0.07s)
```

### Files Changed
- `internal/agent/agent.go` — Added probePeers(), called from Start()

**Completed**: 2026-01-28

---

## Task T007: Write tests for enhanced DiscoverHandler
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
Wrote 3 tests: TestWingmateDiscoverReturnsPeerInfoRich (full PeerInfo with skills), TestWingmateDiscoverEmptyPeerInfo (empty array not null), updated existing TestWingmateDiscoverReturnsPeersFromProvider to use mockRichPeerProvider.

### Files Changed
- `internal/mcp/handlers_test.go` — Added/updated discover tests for PeerInfo

**Completed**: 2026-01-28

---

## Task T008: Update NewDiscoverHandler to use PeerInfo
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
1. Changed `DiscoverInfo.Peers` from `[]string` to `[]PeerInfo`
2. Updated `NewDiscoverHandler` to call `peers.GetPeerInfo()` instead of `peers.GetPeers()`
3. Nil-safe: returns empty `[]PeerInfo{}` when provider is nil

### Evidence
All 4 discover tests pass (including existing nil-provider test).

### Files Changed
- `internal/mcp/handlers.go` — Updated DiscoverInfo, NewDiscoverHandler

**Completed**: 2026-01-28

---

## Task T009: Write tests for background health re-probe
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
Wrote 2 tests: TestAgent_BackgroundProbe_CleanShutdown (no goroutine leak), TestAgent_BackgroundProbe_PeerRecovery (peer comes online, detected by re-probe).

### Evidence
Initially RED — `PeerProbeInterval` undefined on Config struct. After T010 implementation, both pass with race detector.

### Files Changed
- `internal/agent/agent_test.go` — Added 2 background probe tests

**Completed**: 2026-01-28

---

## Task T010: Implement background health re-probe goroutine
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
1. Config: Added `DefaultPeerProbeInterval` (30s), `EnvPeerProbeInterval`, `PeerProbeInterval` field, updated WithEnv/WithDefaults/Clone
2. Agent: Added `backgroundProbe(ctx)` method with ticker + select on ctx.Done()
3. Called `go a.backgroundProbe(ctx)` in Start() after probePeers

### Discovery
Race condition found: test modified `config.Peers` while `probePeers` read it. Fixed by copying peers under `mu.RLock` in `probePeers()`.

### Evidence
```
=== RUN   TestAgent_BackgroundProbe_CleanShutdown
--- PASS (0.36s)
=== RUN   TestAgent_BackgroundProbe_PeerRecovery
--- PASS (0.52s)
```

### Files Changed
- `internal/agent/config.go` — PeerProbeInterval field, env var, defaults
- `internal/agent/agent.go` — backgroundProbe(), probePeers() thread-safe read

**Completed**: 2026-01-28

---

## Task T011: Integration test
**Started**: 2026-01-28
**Status**: ✅ Complete

### What I Did
Created `tests/integration/discover_test.go` with TestIntegration_DiscoverReturnsPeerInfo. Starts two agents (alpha with purpose "iOS dev", bravo with "Backend dev"), alpha has bravo as peer. Sends full MCP JSON-RPC sequence (initialize → notifications/initialized → tools/call wingmate_discover). Verifies response contains bravo with description "Backend dev".

### Evidence
```
=== RUN   TestIntegration_DiscoverReturnsPeerInfo
--- PASS (1.02s)
```

### Files Changed
- `tests/integration/discover_test.go` — New file

**Completed**: 2026-01-28

---

## Task T012: Full regression suite
**Started**: 2026-01-28
**Status**: ✅ Complete

### Evidence
```
ok  	github.com/wingmate/wingmate/internal/agent	3.657s
ok  	github.com/wingmate/wingmate/internal/flightlog	1.977s
ok  	github.com/wingmate/wingmate/internal/llm	4.768s
ok  	github.com/wingmate/wingmate/internal/mcp	2.273s
ok  	github.com/wingmate/wingmate/internal/protocol	4.967s
ok  	github.com/wingmate/wingmate/pkg/types	2.554s
ok  	github.com/wingmate/wingmate/tests/integration	3.507s
```
All packages pass. Zero failures. No race conditions.

**Completed**: 2026-01-28

---

## Discoveries & Learnings

| ID | Discovery | Impact | Action |
|----|-----------|--------|--------|
| D1 | `config.Peers` read by `probePeers()` races with test writes | Race detector failure | Fixed: copy peers under `mu.RLock` in `probePeers()` |
| D2 | MCP integration test requires full JSON-RPC handshake (initialize → initialized → tools/call) | Test complexity | Documented pattern in discover_test.go for reuse in Phase 3 |
