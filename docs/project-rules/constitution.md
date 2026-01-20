# Wingmate Project Constitution

<!--
Sync Impact Report
==================
Mode: CREATE
Version: 1.0.0
Created: 2026-01-21
Rationale: Initial establishment of project doctrine for Wingmate A2A agent communication system
Outstanding TODOs:
  - TODO(CI_PIPELINE): Define CI/CD pipeline once initial implementation exists
  - TODO(COVERAGE_THRESHOLD): Set test coverage thresholds after baseline established
  - TODO(SECURITY_AUDIT): Complete security audit checklist before first release
Supporting docs created: rules.md, idioms.md, architecture.md
-->

**Version:** 1.0.0
**Ratification Date:** 2026-01-21
**Last Amended:** 2026-01-21

---

## 1. Guiding Principles

### 1.1 Mission Statement

Wingmate enables Claude instances running on different machines to communicate bidirectionally for collaborative debugging and diagnostics. The system prioritizes **observability**, **rapid iteration**, and **engineer autonomy**.

### 1.2 Core Principles

| ID | Principle | Rationale |
|----|-----------|-----------|
| P1 | **Observability First** | Every agent conversation MUST be traceable. Logs capture human-readable summaries alongside raw payloads (JSON, binary, etc.) to enable post-hoc analysis of what agents discussed and why. |
| P2 | **Deploy Early, Deploy Often** | Installation and upgrade mechanisms take priority. Engineers MUST be able to rapidly iterate on both "pilot" (initiating) and "wingmate" (responding) agents across machines. |
| P3 | **Engineer Autonomy** | Target users are engineers doing serious debugging. Manual installation from git is acceptable; simplicity over magic. Configuration is explicit, not inferred. |
| P4 | **Protocol Compliance** | Wingmate implements the A2A (Agent-to-Agent) protocol. MCP may be used for local tool integration. Adherence to these standards ensures interoperability. |
| P5 | **Themed but Tasteful** | Use "wingmate" nomenclature for significant features (Pilot/Wingmate agents, Flight Log, etc.) but avoid excessive aviation metaphors. Clarity trumps cleverness. |
| P6 | **Security by Design** | Agent communication involves potentially sensitive debugging data. Authentication, encryption (TLS), and audit logging are non-negotiable. |

<!-- USER CONTENT START -->
<!-- Add project-specific principles here -->
<!-- USER CONTENT END -->

---

## 2. Quality & Verification Strategy

### 2.1 Philosophy

Quality is proven through **automated tests**, **observability**, and **real-world validation** across multiple machines. Given the distributed nature of Wingmate, integration testing between agents is as important as unit testing.

### 2.2 Testing Approach

- **Unit Tests**: Validate individual components (message formatting, protocol handling, log parsing)
- **Integration Tests**: Verify agent-to-agent communication flows
- **Manual Verification**: Cross-machine testing during development (the "can Claude A talk to Claude B?" test)
- **Scratch Tests**: Exploratory tests in `tests/scratch/` for rapid iteration (excluded from CI)

### 2.3 Tools & Automation

| Tool Category | Purpose | Notes |
|---------------|---------|-------|
| Test Runner | Execute test suites | Language-specific (pytest, jest, etc.) |
| Linting | Code style enforcement | Pre-commit hooks recommended |
| Type Checking | Static type validation | When language supports it |
| Tracing | Distributed observability | OpenTelemetry integration via A2A |

### 2.4 Quality Gates

- All tests MUST pass before merge
- New features MUST include relevant tests
- Protocol changes MUST include integration tests
- TODO(COVERAGE_THRESHOLD): Coverage thresholds to be established

<!-- USER CONTENT START -->
<!-- Add project-specific quality requirements here -->
<!-- USER CONTENT END -->

---

## 3. Delivery Practices

### 3.1 Development Workflow

1. **Feature branches** for all changes
2. **Small, focused commits** with descriptive messages
3. **Pull requests** for code review
4. **Rapid iteration**: Deploy to test environments frequently

### 3.2 Documentation Expectations

- README with installation and quick-start
- API documentation for agent capabilities (Agent Cards)
- Architecture decision records (ADRs) for significant choices
- Inline code comments for non-obvious logic only

### 3.3 Definition of Done

A feature is complete when:
- [ ] Code is implemented and passes linting
- [ ] Tests are written and passing
- [ ] Documentation is updated (if user-facing)
- [ ] Cross-machine testing performed (for agent communication features)
- [ ] Observability: Feature actions appear in logs appropriately

### 3.4 Release Process

- Git-based installation: Users clone and run
- Version tags for releases
- Upgrade path: Pull latest, restart agents
- TODO(CI_PIPELINE): Automated release pipeline

<!-- USER CONTENT START -->
<!-- Add project-specific delivery requirements here -->
<!-- USER CONTENT END -->

---

## 4. Governance

### 4.1 Amendment Procedure

1. Propose changes via pull request to this constitution
2. Document rationale in PR description
3. Version bump follows semantic versioning:
   - MAJOR: Breaking changes to principles or governance
   - MINOR: New principles/sections or materially expanded guidance
   - PATCH: Clarifications or formatting adjustments
4. Update "Last Amended" date upon merge

### 4.2 Review Cadence

- Constitution review: Quarterly or upon significant project milestones
- Rules/Idioms review: As needed, typically with major features

### 4.3 Compliance Tracking

- Code reviews verify adherence to rules
- Automated checks enforce style and test requirements
- Architecture decisions traceable to constitution principles

### 4.4 Decision Authority

- **Technical decisions**: Maintainers with input from contributors
- **Doctrine changes**: Require explicit review and approval
- **Security matters**: Elevated review; no shortcuts

<!-- USER CONTENT START -->
<!-- Add project-specific governance requirements here -->
<!-- USER CONTENT END -->

---

## 5. Complexity-First Estimation

### 5.1 No-Time Policy

Wingmate prohibits time-based estimates. All effort quantification uses the **Complexity Score (CS 1-5)** system.

### 5.2 Scoring Factors (0-2 points each)

| Factor | 0 | 1 | 2 |
|--------|---|---|---|
| **S**urface Area | One file | Multiple files | Many files/cross-cutting |
| **I**ntegration Breadth | Internal only | One external | Multiple externals |
| **D**ata & State | None | Minor tweaks | Migration/concurrency |
| **N**ovelty & Ambiguity | Well-specified | Some ambiguity | Significant discovery |
| **N**on-Functional | Standard gates | Moderate constraints | Critical constraints |
| **T**esting & Rollout | Unit only | Integration/e2e | Flags/staged rollout |

### 5.3 Score Mapping

| Total Points | CS | Label | Description |
|--------------|-----|-------|-------------|
| 0-2 | 1 | Trivial | Isolated tweak, no new deps |
| 3-4 | 2 | Small | Few files, familiar code |
| 5-7 | 3 | Medium | Multiple modules, integration tests |
| 8-9 | 4 | Large | Cross-component, new dependencies |
| 10-12 | 5 | Epic | Architectural change, phased rollout |

---

## Appendix A: Terminology

| Term | Definition |
|------|------------|
| **Pilot** | The Claude agent initiating a conversation (A2A "client agent") |
| **Wingmate** | The Claude agent responding to requests (A2A "remote agent") |
| **Flight Log** | The observability log capturing agent conversations |
| **Mission** | A debugging session or task being executed |
| **Hangar** | Configuration and agent card storage |

---

*This constitution establishes the foundational principles for Wingmate development. All rules, idioms, and architectural decisions should trace back to these principles.*
