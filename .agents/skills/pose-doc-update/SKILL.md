---
name: pose-doc-update
description: Use for POSE documentation changes such as AGENTS.md, POSE.md, workflows, rules, module READMEs, or editorial specs. Keeps style consistent and references valid. Trigger keywords - documentation, docs, doc-update, AGENTS, POSE.md, workflow, rule, README, editorial.
when_to_use: The task edits operational documentation rather than product code. Use before writing to align tone, avoid duplication, and preserve references under .pose, .agents/skills, and local paths.
---

# Skill: pose-doc-update

## Required reading

1. [`.pose/workflows/documentation-update.md`](../../../.pose/workflows/documentation-update.md).
2. [`.pose/rules/documentation-style.md`](../../../.pose/rules/documentation-style.md).
3. Root [AGENTS.md](../../../AGENTS.md) and [POSE.md](../../../POSE.md) for root-level scope.

## Steps

1. Choose the correct source: AGENTS for the short contract, POSE for the manual, workflows for procedures, and rules for cumulative domain constraints.
2. Use imperative language, short bullets, consistent terms, and links to a single source of truth.
3. Use [`.pose/templates/doc-audit-report.md`](../../../.pose/templates/doc-audit-report.md) for broad editorial audits.
4. Run `./pose check --strict` after reference changes.
5. Generate a `doc-audit` report for broad changes.

## Output requirements

- Cohesive, readable diff with one editorial intent per commit.
- No verbatim duplication across files.
- Green `./pose check --strict`.
- Optional doc-audit report for broad changes.
