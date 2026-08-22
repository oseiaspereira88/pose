---
name: pose-feature
description: Use to implement a non-trivial feature under POSE when scope affects at least one module and requires a spec, incremental planning, deterministic validation, and cross-execution handoff. Trigger keywords - feature, implement, new functionality, scope change, new spec, behavior-preserving refactor.
when_to_use: The task adds or extends observable functionality rather than fixing a bug, editing docs, or reviewing. Use before coding to establish the spec, consult knowledge, plan increments, and select proportional validation.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, spec-write, validate
---

# Skill: pose-feature

## Before anything

Read `pose_project_state` (MCP tool) or run `pose state` — when the artifact exists
and is not stale, it answers "what is the current state of this project?" in one
call (specs/roadmaps, follow-ups, capabilities, decisions/knowledge, validation
evidence) instead of scanning the repo from scratch every session. When absent or
stale, go straight to the reading below — the artifact is additive, never blocking.

## Required reading

1. [AGENTS.md](../../../AGENTS.md).
2. [`.pose/workflows/feature.md`](../../../.pose/workflows/feature.md).
3. The affected module's nearest `AGENTS.md`, when present.
4. Cumulative rules returned by `pose suggest feature --path <affected-dir>`.

## Steps

1. Identify a short slug and create the spec with `pose new-spec <slug>` (creating `.pose/specs/YYYY-MM-DD-<slug>.md` by default, or with `--folder` when companion amends/assets are needed), or locate the existing spec.
2. Run `pose assess discover [--component <dir>]` / `pose_component_discover` to obtain LOC metrics, module structure, and debts before modifying code.
3. Search `.pose/knowledge/` for related handoffs and decision logs; cite each one used as `knowledge:<slug>` in the spec, the form `pose knowledge-usage` counts.
4. Complete Intent, Requirements, Technical Plan, and Tasks before coding.
5. Implement incrementally, commit changes with a `POSE-Spec: <slug>` trailer in the commit message (e.g. `POSE-Spec: <slug>`) to attribute file modifications to the spec, and run `pose validate --strict --module <affected-path> --report`.
6. Record executed commands and results in Validation.
7. Create a handoff with `pose new-knowledge handoff <slug>` when another execution needs partial state, follow-ups, or owner transition.
8. Complete the Final Report with delivered scope and residual risk.
9. Use [pose-spec-closeout](../pose-spec-closeout/SKILL.md). When review bundles are enabled, seal the validated subject (`pose review bundle spec:<slug> --seal`), attach the independent attestation (`pose review auto-attest <bundle-id> --reviewer agent:<id> --apply` or `pose review attest`) and require `pose review verify spec:<slug>` before closeout. Disposition follow-ups from `pose followups --all` and pass `pose lint-spec <slug> --strict`.
10. Run `pose assess discover --update-state` upon delivery completion to refresh dynamic platform metrics.
11. When Contributor Mode is active and scope reveals missing POSE stack rules or reusable engine capabilities, stage a contribution proposal with `pose contribute stage --type enhancement --title "<summary>"`.

## Output requirements

- Complete spec without required placeholders.
- Successful strict validation for affected modules.
- Closed frontmatter and dispositioned follow-ups.
- Successful strict spec lint.
- Handoff when reusable cross-execution context exists.
