# Wingmate Rules

**Version:** 1.0.0
**Last Updated:** 2026-01-21
**Constitution Reference:** [constitution.md](./constitution.md)

---

## 1. Source Control

### 1.1 Branching

- MUST use feature branches for all changes
- MUST name branches descriptively: `feature/<name>`, `fix/<name>`, `docs/<name>`
- SHOULD keep branches short-lived (merge within days, not weeks)
- MUST NOT commit directly to `main`

### 1.2 Commits

- MUST write descriptive commit messages
- SHOULD use conventional commit format: `type(scope): description`
- MUST NOT include secrets, credentials, or sensitive data
- SHOULD keep commits atomic and focused

### 1.3 Pull Requests

- MUST include description of changes and rationale
- MUST pass all automated checks before merge
- SHOULD reference related issues or decisions
- MUST be reviewed before merge (when team size > 1)

<!-- USER CONTENT START -->
<!-- Add project-specific source control rules here -->
<!-- USER CONTENT END -->

---

## 2. Coding Standards

### 2.1 General

- MUST use consistent formatting (enforced by linter/formatter)
- MUST handle errors explicitly; no silent failures
- SHOULD prefer explicit over implicit
- MUST NOT use abbreviations in public APIs (except well-known: `id`, `url`, etc.)

### 2.2 Naming Conventions

- **Files**: lowercase with hyphens (`agent-executor.py`, `flight-log.ts`)
- **Classes/Types**: PascalCase (`AgentCard`, `FlightLogEntry`)
- **Functions/Methods**: camelCase or snake_case per language convention
- **Constants**: UPPER_SNAKE_CASE (`DEFAULT_TIMEOUT`, `MAX_RETRIES`)
- **Wingmate Terms**: Use established terminology (Pilot, Wingmate, Flight Log, etc.)

### 2.3 Documentation

- MUST document public APIs
- SHOULD explain "why" not "what" in comments
- MUST NOT leave TODO comments without associated issue/tracking
- SHOULD include usage examples in docstrings

### 2.4 Security

- MUST NOT log sensitive data (tokens, credentials, PII)
- MUST use TLS for all agent-to-agent communication
- MUST validate inputs at system boundaries
- SHOULD sanitize data before passing to Claude to prevent prompt injection

<!-- USER CONTENT START -->
<!-- Add project-specific coding standards here -->
<!-- USER CONTENT END -->

---

## 3. Testing

### 3.1 Philosophy (Constitution P1, P2)

Tests serve as **executable documentation** following TAD (Test-Assisted Development) principles. Tests MUST "pay rent" through comprehension value, not just coverage metrics.

### 3.2 Test Quality Standards

Every test MUST include these elements (in docstring or comment):

| Field | Requirement | Purpose |
|-------|-------------|---------|
| **Why** | MUST explain why this test exists | Business/bug/regression reason |
| **Contract** | MUST document the invariant being asserted | What behavior is guaranteed |
| **Usage Notes** | MUST describe how to call the API | Gotchas, required setup |
| **Quality Contribution** | MUST state what failures it catches | Value proposition |
| **Worked Example** | SHOULD include inputs/outputs | Concrete illustration |

### 3.3 Scratch-to-Promote Workflow

```
tests/scratch/    <-- Fast exploration, NOT in CI
    |
    v (promote if valuable)
    |
tests/unit/       <-- Promoted, documented tests
tests/integration/<-- Multi-component tests
```

**Promotion Heuristic (CROP)**:
- **C**ritical path: Does it test a core feature?
- **R**egression-prone: Has this broken before?
- **O**paque behavior: Is the logic non-obvious?
- **P**otential edge case: Does it cover boundary conditions?

If none apply, delete the scratch test (keep learning notes in PR).

### 3.4 Test-Driven Development Guidance

- TDD SHOULD be used for: complex logic, A2A protocol handling, critical paths
- TDD MAY be skipped for: configuration, simple wrappers, trivial operations
- When using TDD, follow RED-GREEN-REFACTOR cycles
- Tests written first MUST document expected behavior clearly

### 3.5 Test Reliability

- MUST NOT use network calls (use fixtures/mocks for external dependencies)
- MUST NOT use sleep/timers (use time mocking if needed)
- MUST be deterministic (no flaky tests in main suite)
- SHOULD be reasonably fast for quick feedback

### 3.6 Test Organization

```
tests/
├── scratch/       # Temporary exploration (gitignored from CI)
├── unit/          # Isolated component tests
├── integration/   # Multi-component, agent-to-agent tests
├── fixtures/      # Shared test data, realistic examples
└── conftest.py    # (or equivalent) Shared test configuration
```

### 3.7 Mock Usage Policy

- SHOULD use "Targeted" mocking: mock external dependencies, not internal logic
- MUST document WHY the real dependency isn't used when mocking
- SHOULD prefer real data/fixtures over mocks when practical
- Mocks MUST be behavior-focused, not implementation-focused

### 3.8 Test Documentation Format

**Python Example:**
```python
def test_given_valid_message_when_sending_then_receives_acknowledgment():
    """
    Test Doc:
    - Why: Verify basic A2A message exchange works (core functionality)
    - Contract: send_message() returns MessageResponse with status='acknowledged'
    - Usage Notes: Requires agent to be running; use fixture for mock agent
    - Quality Contribution: Catches protocol violations, serialization bugs
    - Worked Example: send_message("ping") -> MessageResponse(status="acknowledged")
    """
    # Arrange
    agent = create_mock_agent()

    # Act
    response = agent.send_message("ping")

    # Assert
    assert response.status == "acknowledged"
```

**TypeScript Example:**
```typescript
test('given valid message when sending then receives acknowledgment', () => {
  /*
  Test Doc:
  - Why: Verify basic A2A message exchange works (core functionality)
  - Contract: sendMessage() returns MessageResponse with status='acknowledged'
  - Usage Notes: Requires agent fixture; async operation
  - Quality Contribution: Catches protocol violations, serialization bugs
  - Worked Example: sendMessage("ping") -> { status: "acknowledged" }
  */
  // Arrange-Act-Assert
});
```

<!-- USER CONTENT START -->
<!-- Add project-specific testing rules here -->
<!-- USER CONTENT END -->

---

## 4. Tooling & Automation

### 4.1 Required Tools

- MUST have linter configured and passing
- MUST have formatter configured (format on save recommended)
- SHOULD have pre-commit hooks for lint/format
- MUST have CI running tests on PRs

### 4.2 Local Development

- MUST be able to run full test suite locally
- SHOULD document environment setup in README
- MUST support major platforms (macOS, Linux; Windows/WSL acceptable)

### 4.3 Observability (Constitution P1)

- MUST log agent conversations to Flight Log
- Log entries MUST include:
  - Timestamp
  - Agent identifiers (Pilot/Wingmate)
  - Human-readable summary
  - Raw payload (may be JSON, binary reference, etc.)
- SHOULD support OpenTelemetry trace propagation
- MUST NOT log sensitive data (redact as needed)

### 4.4 CI/CD

- TODO(CI_PIPELINE): Define specific CI requirements
- MUST run tests on all PRs
- SHOULD run linting on all PRs
- SHOULD build/validate on multiple platforms if applicable

<!-- USER CONTENT START -->
<!-- Add project-specific tooling requirements here -->
<!-- USER CONTENT END -->

---

## 5. A2A Protocol Compliance

### 5.1 Agent Cards

- MUST expose Agent Card at well-known URL (`/.well-known/agent.json`)
- MUST declare capabilities accurately
- MUST specify supported authentication methods
- SHOULD include meaningful description

### 5.2 Communication

- MUST use HTTPS for production deployments
- MUST implement proper error responses per A2A spec
- SHOULD support streaming (SSE) for long-running operations
- MUST handle connection failures gracefully

### 5.3 Authentication

- MUST implement at least one auth method (API key, OAuth, mTLS)
- SHOULD use short-lived tokens when possible
- MUST validate tokens/credentials on every request
- MUST NOT expose credentials in logs or Agent Cards

<!-- USER CONTENT START -->
<!-- Add project-specific A2A rules here -->
<!-- USER CONTENT END -->

---

## 6. Complexity Estimation

### 6.1 Prohibition

- MUST NOT use time-based estimates (hours, days, "quick", "soon")
- MUST use Complexity Score (CS 1-5) system per constitution

### 6.2 Required Output Format

When estimating work, MUST include:

```json
{
  "complexity": {
    "score": 3,
    "label": "medium",
    "breakdown": {
      "surface": 1,
      "integration": 1,
      "data_state": 0,
      "novelty": 1,
      "nfr": 0,
      "testing_rollout": 0
    },
    "confidence": 0.8
  },
  "assumptions": ["Agent Card schema is finalized"],
  "dependencies": ["A2A Python SDK"],
  "risks": ["Protocol version compatibility"],
  "phases": ["Design", "Implementation", "Testing"]
}
```

### 6.3 High Complexity Requirements

For CS >= 4:
- MUST include phased rollout plan
- SHOULD consider feature flags
- MUST document rollback strategy

<!-- USER CONTENT START -->
<!-- Add project-specific estimation rules here -->
<!-- USER CONTENT END -->

---

*These rules enforce the principles defined in the [constitution](./constitution.md). When in doubt, refer to the guiding principles.*
