# ADR-XXX: [Title]

**Status**: PROPOSED | DECIDED | SUPERSEDED | DEPRECATED
**Date**: YYYY-MM-DD
**Deciders**: [List stakeholders]
**Supersedes**: [ADR-NNN if applicable]
**Superseded by**: [ADR-NNN if applicable]

---

## Context

What is the issue that we're seeing that is motivating this decision or change?

Describe the forces at play (technical, political, social, project constraints). Include any assumptions or constraints that influenced the decision.

---

## Decision Drivers

- Driver 1: [e.g., Must compile to single binary]
- Driver 2: [e.g., Cross-platform support required]
- Driver 3: [e.g., Minimize external dependencies]

---

## Options Considered

### Option A: [Name]

**Description**: Brief explanation of this approach.

**Pros**:
- Pro 1
- Pro 2

**Cons**:
- Con 1
- Con 2

### Option B: [Name]

**Description**: Brief explanation of this approach.

**Pros**:
- Pro 1
- Pro 2

**Cons**:
- Con 1
- Con 2

### Option C: [Name]

**Description**: Brief explanation of this approach.

**Pros**:
- Pro 1
- Pro 2

**Cons**:
- Con 1
- Con 2

---

## Decision

**Chosen Option**: [Option X]

[Explain why this option was selected. Reference the decision drivers and explain how this option best addresses them.]

---

## Consequences

### Positive

- Consequence 1
- Consequence 2

### Negative

- Consequence 1 (with mitigation if applicable)
- Consequence 2

### Neutral

- Consequence that is neither good nor bad but worth noting

---

## Implementation Notes

[Optional: Any specific implementation guidance, code patterns, or considerations for developers implementing this decision.]

---

## Related Decisions

- [ADR-NNN](./NNN-title.md): Related decision description
- Constitution Principle: [P1, P2, etc.] - How this aligns with project principles

---

## References

- [Link to relevant documentation]
- [Link to relevant discussion/issue]

---

<!--
MACHINE-READABLE CONTEXT
========================
This section provides structured metadata for AI assistants to quickly parse ADR context.
Format: YAML for easy parsing, JSON-compatible where possible.

```yaml
adr:
  id: XXX
  title: "[Title]"
  status: "proposed|decided|superseded|deprecated"
  date: "YYYY-MM-DD"

  # Quick decision summary (one line)
  decision_summary: "[One sentence describing the decision]"

  # What this ADR affects
  affects:
    - component: "[component name]"
      impact: "high|medium|low"
    - component: "[another component]"
      impact: "high|medium|low"

  # Key constraints this decision imposes
  constraints:
    - "[Constraint 1: e.g., Must use X library]"
    - "[Constraint 2: e.g., Cannot use Y pattern]"

  # Dependencies on other ADRs
  depends_on: []  # List of ADR IDs this builds upon

  # ADRs that depend on this one
  dependents: []  # List of ADR IDs that reference this

  # Constitution principles this implements
  principles: []  # e.g., ["P1", "P4"]

  # Implementation status
  implementation:
    status: "not_started|in_progress|complete"
    location: "[file path or module if implemented]"

  # Tags for categorization
  tags: []  # e.g., ["infrastructure", "protocol", "testing"]

  # Complexity impact (if this changes project complexity)
  complexity_impact:
    score_delta: 0  # How much this adds to overall project complexity
    reason: "[Why this affects complexity]"
```
-->
