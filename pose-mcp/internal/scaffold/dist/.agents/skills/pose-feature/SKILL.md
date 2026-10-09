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

1. Resolve the task before creating or starting work: run `pose context --task <typed-or-qualified-artifact-ref> --json`. Reuse the canonical spec when it resolves; stop on ambiguity, unsupported metadata or `transfer-in-progress`. Do not infer an external task from a bare slug or matching checkout path. Use `pose new-spec <slug>` for a local draft. For a new cross-project authority, pass the exact `xref:<project>/spec:<slug>` and the fresh `context_revision` to `pose new-spec-qualified <slug> --task <xref> --expect-context <digest>`; the target must have an explicit `POSE_PROJECT_ROOTS` binding. The distinct verb makes pre-contract engines reject the operation. Keep requirements in the authority spec and link coordinator composition through qualified references.
2. Run `pose assess discover --if-stale [--component <dir>]` / `pose_component_discover` before modifying code; it reuses an assessment bound to the same committed content, engine and matrix, and refreshes a stale one.
3. Search `.pose/knowledge/` for related handoffs and decision logs; cite each one used as `knowledge:<slug>` in the spec, the form `pose knowledge-usage` counts.
4. Complete Intent, Requirements, Technical Plan, and Tasks before coding.
5. When a decision, approval, input, external operation or acceptance genuinely belongs to someone else, record it with `pose action open` instead of stopping the session or hiding it in a follow-up — but only when a different answer would materially change execution, scope, authority, risk acceptance, closeout or publication and no authorization already given covers it. Give it targets and the phases it restricts (`--effect closeout:block` when it does not stop implementation), keep working on what it does not restrict, and present the open requests together with `pose state --attention`. A cosmetic or reversible choice you can make within the authorized scope is not a request.
6. Implement incrementally, commit changes with a `POSE-Spec: <slug>` trailer in the commit message (e.g. `POSE-Spec: <slug>`) to attribute file modifications to the spec, and run `pose validate --strict --module <affected-path> --report`.
7. Record executed commands and results in Validation.
8. Create a handoff with `pose new-knowledge handoff <slug>` when another execution needs partial state, follow-ups, or owner transition.
9. Complete the Final Report with delivered scope and residual risk.
10. Use [pose-spec-closeout](../pose-spec-closeout/SKILL.md). When review bundles are enabled, seal the validated subject (`pose review bundle spec:<slug> --seal`), attach the independent attestation (`pose review auto-attest <bundle-id> --reviewer agent:<id>` to collect the mechanical half, then `pose review attest spec:<slug> ... --apply` to answer what it reports as pending) and require `pose review verify spec:<slug>` before closeout. Disposition follow-ups from `pose followups --all` and pass `pose lint-spec <slug> --strict`.
11. Run `pose assess discover --if-stale --update-state` upon delivery completion to refresh the metrics of the components the delivery changed.
12. When Contributor Mode is active and scope reveals missing POSE stack rules or reusable engine capabilities, stage a contribution proposal with `pose contribute stage --type enhancement --title "<summary>"`.

## Output requirements

- Complete spec without required placeholders.
- Successful strict validation for affected modules.
- Closed frontmatter and dispositioned follow-ups.
- Successful strict spec lint.
- Handoff when reusable cross-execution context exists.
