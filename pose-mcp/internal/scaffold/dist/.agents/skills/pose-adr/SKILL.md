---
name: pose-adr
description: Use to record an architectural decision under POSE when choosing between structurally significant options, changing a public contract, or accepting a trade-off that may need later review. Trigger keywords - ADR, architecture decision, structural contract, technical decision, trade-off, design choice.
when_to_use: A technical decision's rationale must outlive the original author. Typical cases include stack or library changes, HTTP or schema contracts, cross-module organization patterns, and recurring rejected alternatives.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, adr-write
---

# Skill: pose-adr

Record architectural decisions so their rationale remains reviewable.

## Required reading

1. [AGENTS.md](../../../AGENTS.md) for ADR requirements.
2. Existing [`.pose/adr/`](../../../.pose/adr/) entries to avoid duplication and identify superseded decisions.
3. Applicable rules under `.pose/rules/`.

## Steps

1. Confirm this is architectural rather than tactical: someone will reasonably ask why in six months.
2. Search existing ADRs for the topic.
3. Run `pose new-adr "<decision title>"`.
4. Fill Status, Context, Decision, and Consequences in the generated file.
5. Link affected modules and explain rejected trade-offs.
6. Create a decision log with `pose new-knowledge decision-log adr-<slug>-review --owner @<team> --ttl-days 90` when a future review trigger exists.
7. Reference the ADR from the related spec's Decisions section.

## Output requirements

- Complete ADR under `.pose/adr/<date>-<slug>.md`.
- One-line rationale for each rejected trade-off.
- Decision log when a review trigger exists.
- Cross-reference from an active implementation spec when applicable.
