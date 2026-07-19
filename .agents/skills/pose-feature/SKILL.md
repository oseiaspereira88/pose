---
name: pose-feature
description: Use to implement a non-trivial feature under POSE when scope affects at least one module and requires a spec, incremental planning, deterministic validation, and cross-execution handoff. Trigger keywords - feature, implement, new functionality, scope change, new spec, behavior-preserving refactor.
when_to_use: The task adds or extends observable functionality rather than fixing a bug, editing docs, or reviewing. Use before coding to establish the spec, consult knowledge, plan increments, and select proportional validation.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, spec-write, validate
---

# Skill: pose-feature

## Required reading

1. [AGENTS.md](../../../AGENTS.md).
2. [`.pose/workflows/feature.md`](../../../.pose/workflows/feature.md).
3. The affected module's nearest `AGENTS.md`, when present.
4. Cumulative rules returned by `pose suggest feature --path <affected-dir>`.

## Steps

1. Identify a short slug and create or locate `.pose/specs/<slug>/spec.md`.
2. Search `.pose/knowledge/` for related handoffs and decision logs.
3. Complete Intent, Requirements, Technical Plan, and Tasks before coding.
4. Implement incrementally and run `pose validate --strict --module <affected-path> --report`.
5. Record executed commands and results in Validation.
6. Create a handoff when another execution needs partial state, follow-ups, or owner transition.
7. Complete the Final Report with delivered scope and residual risk.
8. Use [pose-spec-closeout](../pose-spec-closeout/SKILL.md), disposition follow-ups, and pass `pose lint-spec <slug> --strict`.

## Output requirements

- Complete spec without required placeholders.
- Successful strict validation for affected modules.
- Closed frontmatter and dispositioned follow-ups.
- Successful strict spec lint.
- Handoff when reusable cross-execution context exists.
