# Wingmate Idioms

**Version:** 1.0.0
**Last Updated:** 2026-01-21
**Constitution Reference:** [constitution.md](./constitution.md)
**Rules Reference:** [rules.md](./rules.md)

---

## 1. Project Structure

### 1.1 Recommended Directory Layout

```
wingmate/
├── docs/
│   ├── project-rules/     # Constitution, rules, idioms, architecture
│   ├── research/          # Technical research and references
│   └── adr/               # Architecture Decision Records
├── src/
│   ├── pilot/             # Initiating agent code
│   ├── wingmate/          # Responding agent code
│   ├── common/            # Shared utilities
│   ├── protocol/          # A2A protocol implementation
│   └── observability/     # Flight Log and tracing
├── tests/
│   ├── scratch/           # Temporary exploration tests
│   ├── unit/              # Isolated component tests
│   ├── integration/       # Agent-to-agent tests
│   └── fixtures/          # Test data
├── config/                # Configuration templates
├── scripts/               # Development and deployment scripts
└── README.md
```

### 1.2 File Naming

| Type | Pattern | Example |
|------|---------|---------|
| Source files | lowercase-hyphenated | `agent-executor.py` |
| Test files | `test_<module>.py` or `<module>.test.ts` | `test_agent_executor.py` |
| Config files | lowercase-hyphenated | `agent-config.json` |
| Documentation | Title-Case or lowercase | `README.md`, `getting-started.md` |

<!-- USER CONTENT START -->
<!-- Add project-specific structure idioms here -->
<!-- USER CONTENT END -->

---

## 2. Wingmate Terminology in Code

### 2.1 Naming Map

| Concept | Code Name | Description |
|---------|-----------|-------------|
| Initiating agent | `Pilot` | The Claude instance that starts a conversation |
| Responding agent | `Wingmate` | The Claude instance that receives and responds |
| Conversation log | `FlightLog` | Observability record of agent exchanges |
| Debugging session | `Mission` | A task or investigation being executed |
| Configuration store | `Hangar` | Where agent cards and config live |
| Single exchange | `Sortie` | One request-response cycle |

### 2.2 Class/Type Names

```python
# Good - uses established terminology
class PilotAgent:
    def initiate_mission(self, target: WingmateAgent) -> Mission:
        ...

class FlightLogEntry:
    timestamp: datetime
    pilot_id: str
    wingmate_id: str
    summary: str
    payload: Any

# Avoid - generic or inconsistent
class ClientAgent:  # Use PilotAgent
class ServerAgent:  # Use WingmateAgent
class LogEntry:     # Use FlightLogEntry
```

### 2.3 When NOT to Use Themed Names

- Internal implementation details (use descriptive technical names)
- Third-party integrations (use their terminology)
- Standard protocol concepts (keep A2A terminology: Task, Message, Artifact)

**Example:**
```python
# A2A protocol concepts - keep standard names
class A2ATask:      # Not "Mission" - this is A2A's concept
class A2AMessage:   # Not "Transmission"
class AgentCard:    # Standard A2A term

# Wingmate concepts - use themed names
class Mission:      # Our debugging session concept
class FlightLog:    # Our observability concept
```

<!-- USER CONTENT START -->
<!-- Add project-specific terminology idioms here -->
<!-- USER CONTENT END -->

---

## 3. Observability Patterns

### 3.1 Flight Log Entry Structure

Every agent exchange SHOULD be logged with this structure:

```python
@dataclass
class FlightLogEntry:
    # Required fields
    timestamp: datetime
    trace_id: str
    pilot_id: str
    wingmate_id: str
    direction: Literal["outbound", "inbound"]

    # Human-readable summary (one sentence)
    summary: str

    # Raw payload - can be anything
    payload_type: str  # "json", "text", "binary_ref", etc.
    payload: Any       # The actual content

    # Optional context
    mission_id: Optional[str] = None
    parent_sortie_id: Optional[str] = None
```

### 3.2 Summary Writing Guidelines

The `summary` field is for humans reviewing logs. Write it as:

```
Good:
- "Pilot requested CPU metrics from Wingmate"
- "Wingmate returned 47 log entries from last hour"
- "Pilot asked for clarification on error code E-4521"

Avoid:
- "Message sent"  # Too vague
- "GetMetrics request with params cpu=true memory=false..." # Too detailed
- JSON in summary # Put that in payload
```

### 3.3 Payload Examples

```python
# Text payload
FlightLogEntry(
    summary="Wingmate provided analysis of memory leak",
    payload_type="text",
    payload="The memory leak appears to originate from..."
)

# JSON payload
FlightLogEntry(
    summary="Pilot requested system metrics",
    payload_type="json",
    payload={"metrics": ["cpu", "memory"], "duration": "1h"}
)

# Binary reference
FlightLogEntry(
    summary="Wingmate attached core dump file",
    payload_type="binary_ref",
    payload={"path": "/tmp/dumps/core.12345", "size_bytes": 1048576}
)
```

<!-- USER CONTENT START -->
<!-- Add project-specific observability idioms here -->
<!-- USER CONTENT END -->

---

## 4. A2A Integration Patterns

### 4.1 Agent Initialization

```python
# Pattern: Create agent with explicit configuration
def create_pilot_agent(config: PilotConfig) -> PilotAgent:
    """
    Create a Pilot agent ready to initiate missions.

    Configuration is explicit, not inferred from environment.
    """
    agent_card = AgentCard(
        name=config.name,
        description=config.description,
        url=config.endpoint_url,
        capabilities=config.capabilities,
        authentication=config.auth_config,
    )

    return PilotAgent(
        card=agent_card,
        flight_log=FlightLog(config.log_path),
        client=A2AClient(config.client_options),
    )
```

### 4.2 Message Exchange Pattern

```python
async def execute_sortie(
    pilot: PilotAgent,
    wingmate_url: str,
    request: SortieRequest
) -> SortieResponse:
    """
    Execute a single request-response exchange (sortie).

    Always logs both outbound and inbound messages.
    """
    # Log outbound
    pilot.flight_log.record(
        direction="outbound",
        summary=f"Requesting {request.operation} from Wingmate",
        payload=request.to_dict(),
    )

    # Execute A2A call
    response = await pilot.client.send_task(
        url=wingmate_url,
        task=request.to_a2a_task(),
    )

    # Log inbound
    pilot.flight_log.record(
        direction="inbound",
        summary=f"Received response: {response.summary}",
        payload=response.to_dict(),
    )

    return SortieResponse.from_a2a(response)
```

### 4.3 Error Handling Pattern

```python
async def safe_sortie(pilot: PilotAgent, request: SortieRequest) -> SortieResult:
    """
    Execute sortie with comprehensive error handling.
    """
    try:
        response = await execute_sortie(pilot, request)
        return SortieResult.success(response)

    except A2AAuthError as e:
        pilot.flight_log.record(
            direction="error",
            summary=f"Authentication failed: {e.message}",
            payload={"error_type": "auth", "details": str(e)},
        )
        return SortieResult.auth_error(e)

    except A2AConnectionError as e:
        pilot.flight_log.record(
            direction="error",
            summary=f"Connection failed to Wingmate",
            payload={"error_type": "connection", "details": str(e)},
        )
        return SortieResult.connection_error(e)

    except A2AProtocolError as e:
        pilot.flight_log.record(
            direction="error",
            summary=f"Protocol error: {e.message}",
            payload={"error_type": "protocol", "details": str(e)},
        )
        return SortieResult.protocol_error(e)
```

<!-- USER CONTENT START -->
<!-- Add project-specific A2A patterns here -->
<!-- USER CONTENT END -->

---

## 5. Testing Idioms

### 5.1 Integration Test Setup

```python
@pytest.fixture
async def pilot_wingmate_pair():
    """
    Create a connected Pilot-Wingmate pair for integration testing.

    Both agents run locally on different ports.
    """
    wingmate = await start_test_wingmate(port=9001)
    pilot = create_test_pilot(wingmate_url=f"http://localhost:9001")

    yield pilot, wingmate

    await wingmate.shutdown()
    await pilot.shutdown()


async def test_basic_communication(pilot_wingmate_pair):
    """
    Test Doc:
    - Why: Verify fundamental agent-to-agent communication works
    - Contract: Pilot can send message and receive response from Wingmate
    - Usage Notes: Uses local test agents; no external dependencies
    - Quality Contribution: Catches protocol, serialization, routing bugs
    """
    pilot, wingmate = pilot_wingmate_pair

    response = await pilot.send_message("ping")

    assert response.status == "acknowledged"
    assert "pong" in response.content.lower()
```

### 5.2 Flight Log Assertions

```python
def assert_flight_log_contains(
    log: FlightLog,
    summary_pattern: str,
    direction: Optional[str] = None
):
    """
    Assert that flight log contains an entry matching the pattern.

    Useful for verifying observability in integration tests.
    """
    entries = log.get_entries()
    matching = [
        e for e in entries
        if summary_pattern in e.summary
        and (direction is None or e.direction == direction)
    ]
    assert matching, f"No log entry matching '{summary_pattern}'"
    return matching[0]


async def test_sortie_logged(pilot_wingmate_pair):
    """
    Test Doc:
    - Why: Verify observability requirement (Constitution P1)
    - Contract: Every exchange appears in Flight Log with summary + payload
    - Usage Notes: Check both outbound and inbound entries
    """
    pilot, wingmate = pilot_wingmate_pair

    await pilot.request_metrics()

    assert_flight_log_contains(pilot.flight_log, "Requesting", "outbound")
    assert_flight_log_contains(pilot.flight_log, "Received", "inbound")
```

### 5.3 Mock Wingmate Pattern

```python
class MockWingmate:
    """
    A programmable mock Wingmate for testing Pilot behavior.

    Use when you need to test specific Pilot reactions to
    various Wingmate responses without running a real agent.
    """

    def __init__(self):
        self.responses = []
        self.received_requests = []

    def queue_response(self, response: A2AResponse):
        """Queue a response to return for the next request."""
        self.responses.append(response)

    async def handle_request(self, request: A2ARequest) -> A2AResponse:
        self.received_requests.append(request)
        if self.responses:
            return self.responses.pop(0)
        return A2AResponse.default_ack()
```

<!-- USER CONTENT START -->
<!-- Add project-specific testing idioms here -->
<!-- USER CONTENT END -->

---

## 6. Complexity Estimation Examples

### 6.1 CS-1 (Trivial)

**Example:** Add a new log level to Flight Log

```json
{
  "complexity": {
    "score": 1,
    "label": "trivial",
    "breakdown": {"surface": 0, "integration": 0, "data_state": 0, "novelty": 0, "nfr": 0, "testing_rollout": 1},
    "confidence": 0.95
  },
  "assumptions": ["Log levels are already enumerated"],
  "phases": ["Implementation", "Unit test"]
}
```

### 6.2 CS-3 (Medium)

**Example:** Add streaming support to message exchange

```json
{
  "complexity": {
    "score": 3,
    "label": "medium",
    "breakdown": {"surface": 1, "integration": 1, "data_state": 0, "novelty": 1, "nfr": 0, "testing_rollout": 0},
    "confidence": 0.75
  },
  "assumptions": ["A2A SDK supports SSE", "Target infrastructure allows long-lived connections"],
  "dependencies": ["A2A Python SDK streaming module"],
  "risks": ["Proxy/firewall SSE compatibility"],
  "phases": ["Research SDK streaming API", "Implementation", "Integration tests"]
}
```

### 6.3 CS-5 (Epic)

**Example:** Add multi-agent orchestration (one Pilot, multiple Wingmates)

```json
{
  "complexity": {
    "score": 5,
    "label": "epic",
    "breakdown": {"surface": 2, "integration": 2, "data_state": 1, "novelty": 2, "nfr": 1, "testing_rollout": 2},
    "confidence": 0.5
  },
  "assumptions": ["Single Pilot architecture is acceptable"],
  "dependencies": ["Agent discovery mechanism", "Load balancing strategy"],
  "risks": ["Coordination complexity", "Partial failure handling", "Log correlation"],
  "phases": [
    "Design: Multi-agent topology",
    "Implementation: Discovery and routing",
    "Implementation: Coordinated logging",
    "Testing: Multi-agent integration",
    "Rollout: Feature flag + gradual enable"
  ]
}
```

<!-- USER CONTENT START -->
<!-- Add project-specific complexity examples here -->
<!-- USER CONTENT END -->

---

## 7. Common Anti-Patterns

### 7.1 Avoid: Silent Failures

```python
# Bad - swallows error silently
try:
    response = await pilot.send_message(request)
except Exception:
    pass  # Lost context, no observability

# Good - log and handle explicitly
try:
    response = await pilot.send_message(request)
except A2AError as e:
    pilot.flight_log.record_error(e)
    raise MissionAborted(f"Communication failed: {e}")
```

### 7.2 Avoid: Credential Exposure

```python
# Bad - logs contain secrets
logger.info(f"Connecting with token: {auth_token}")

# Good - redact sensitive data
logger.info(f"Connecting with token: {auth_token[:4]}...{auth_token[-4:]}")
# Or better:
logger.info("Connecting with authentication configured")
```

### 7.3 Avoid: Time-Based Estimates

```python
# Bad
# "This should be quick, maybe 30 minutes"
# "ETA: 2 hours"

# Good
# "CS-2: Few files, familiar patterns, integration test needed"
```

### 7.4 Avoid: Over-Themed Naming

```python
# Bad - over the top
class TakeoffSequencer:  # Just "Initializer" or "Startup"
class RunwayAllocator:   # Huh?
class FuelGauge:         # What does this even do?

# Good - themed but clear
class PilotAgent:        # Clear: the initiating agent
class FlightLog:         # Clear: the conversation log
class Mission:           # Clear: a debugging session
```

<!-- USER CONTENT START -->
<!-- Add project-specific anti-patterns here -->
<!-- USER CONTENT END -->

---

*These idioms illustrate patterns that align with the [constitution](./constitution.md) and [rules](./rules.md). Use them as starting points, not rigid templates.*
