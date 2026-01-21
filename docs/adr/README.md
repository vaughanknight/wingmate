# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the Wingmate project.

## Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [000](./000-template.md) | Template | N/A | 2026-01-21 |
| [001](./001-language-choice.md) | Language Choice (Go) | DECIDED | 2026-01-21 |
| [002](./002-flight-log-storage.md) | Flight Log Storage (JSONL + stdout) | DECIDED | 2026-01-21 |

## Usage

When making a significant technical decision:

1. Copy `000-template.md` to `NNN-descriptive-title.md`
2. Fill in all sections including the machine-readable YAML at the bottom
3. Submit for review via PR
4. Update this index when ADR is approved

See [constitution.md](../project-rules/constitution.md#45-architecture-decision-records-adrs) for when to write an ADR.

## Quick Reference

Each ADR includes a **machine-readable YAML block** at the bottom for efficient AI assistant parsing. Look for the `MACHINE-READABLE CONTEXT` section to quickly understand:

- `decision_summary`: One-line decision description
- `constraints`: What this decision requires/forbids
- `affects`: Which components are impacted
- `implementation.status`: Whether it's implemented yet
