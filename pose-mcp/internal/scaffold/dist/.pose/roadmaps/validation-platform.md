---
slug: validation-platform
status: done
created_at: 2026-07-18
depends_on:
---

# Roadmap: Validation platform

**Portfolio order:** 4 of 7
**Outcome:** make deterministic validation portable, machine-consumable, resource-bounded and useful in polyglot monorepositories.

The result contract comes before optimization. Changed-scope selection, timeouts and broader stacks are valuable only when skipped and partial results remain explicit and reproducible.

## Milestone: result-contract
- after:
- target_start: 2026-08-24
- target_due: 2026-09-11
- specs: pose-structured-validation-results

**Exit gate:** JSON, JUnit and SARIF projections share stable identities and outcome semantics.

## Milestone: safe-selection
- after: result-contract
- target_start: 2026-09-14
- target_due: 2026-10-16
- specs: pose-changed-scope-validation, pose-validation-runtime-guardrails

**Exit gate:** selection is explainable and execution is bounded locally or delegated safely.

## Milestone: ecosystem-breadth
- after: safe-selection
- target_start: 2026-10-19
- target_due: 2026-11-20
- specs: pose-stack-catalog-expansion, pose-monorepo-validation-recipes

**Exit gate:** representative polyglot and monorepo fixtures pass under the same result contract.

## Risk controls

- Never evaluate shell text or expand untrusted paths outside the project root.
- Record every skipped check and its deterministic selection reason.
- Keep sandbox execution in Harness rather than weakening the local CLI boundary.
