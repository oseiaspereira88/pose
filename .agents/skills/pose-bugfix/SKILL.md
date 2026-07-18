---
name: pose-bugfix
description: Use to correct an observable defect under POSE by reproducing the failure, isolating root cause, applying the smallest cohesive fix, covering regression, and recording systemic debt. Trigger keywords - bugfix, bug, defect, regression, hotfix, correction, root cause, fix.
when_to_use: The task corrects observable broken behavior rather than adding a feature. Use before editing code so reproduction, root-cause isolation, and regression coverage are explicit.
---

# Skill: pose-bugfix

## Required reading

1. [AGENTS.md](../../../AGENTS.md).
2. [`.pose/workflows/bugfix.md`](../../../.pose/workflows/bugfix.md).
3. The affected module's nearest `AGENTS.md`, when present.
4. Cumulative rules returned by `./pose suggest bugfix --path <affected-dir>`.

## Steps

1. Reproduce the defect and record expected and actual observable output.
2. Search `.pose/knowledge/` for earlier incidents and handoffs in the same module.
3. Isolate root cause and map collateral impact.
4. Implement the smallest cohesive fix without parallel refactoring.
5. Add or update a regression test.
6. Run `./pose validate --tolerant --module <affected-path> --report`.
7. Create a decision log when root cause exposes systemic debt or a significant trade-off.

## Output requirements

- Root-cause description and correction approach.
- Surgical diff without unrelated changes.
- Regression-test evidence.
- Successful applicable validation evidence.
- Decision log under `.pose/knowledge/` when needed.
