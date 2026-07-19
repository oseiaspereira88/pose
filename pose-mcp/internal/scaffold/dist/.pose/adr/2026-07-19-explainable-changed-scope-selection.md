# ADR: Explainable changed-scope selection

## Status
Accepted (2026-07-19) — spec `pose-changed-scope-validation`

## Context

Full-matrix validation in a polyglot monorepo wastes feedback time when a
change touches one module — but naive path filtering silently narrows
coverage, which the roadmap forbids: skipped work must stay visible. The
[Nx affected model](https://nx.dev/ci/features/affected) shows the shape:
changed files → owning projects → dependents → tasks.

Alternatives considered:

1. **Build-graph analysis per language** — precise but requires per-stack
   tooling; contradicts POSE's thin deterministic core and the spec's
   non-goal (no perfect semantic impact analysis).
2. **Path prefixes + declared dependency edges + policy widening, with
   reasons everywhere.**

## Decision

Option 2:

- **Inputs (R1):** `--changed-from <rev> [--changed-to <rev>]`. Revisions
  are confined to a safe grammar (no leading dash, no option injection)
  before Git ever sees them. Worktree comparisons include untracked files —
  invisible new code must never narrow validation.
- **Selection:** a module is selected when it contains a changed file
  (longest-prefix match), when it transitively depends on a selected module
  via `dependsOn` edges declared in `module-metadata.json`, or when policy
  widens it (`criticality: high` always runs). A changed file outside every
  module selects everything — uncertainty prefers safe execution (R2).
- **Explainability (R1/R3):** every selected module carries its reason
  (changed file, dependency chain, policy); every unselected check is
  recorded in the structured result as `skipped` with
  `changed-scope: module not affected by <base>..<head>`; `--explain`
  prints each decision. Nothing is silently dropped.
- **Determinism:** same base/head/config → same selection; no caches of
  mutable state (the only inputs are Git content and versioned indexes).
- **Compatibility:** without the flags, full validation is byte-identical
  to before.

## Consequences

- Positive: monorepo feedback shrinks to affected modules while the result
  contract proves exactly what was skipped and why.
- Positive: dependency edges live in versioned metadata the team already
  owns; no new index format.
- Trade-off: undeclared dependencies create false negatives — mitigated by
  the safe-execution fallback, policy widening and the visible skip
  reasons that make gaps reviewable.
- Residual: path-based selection cannot see semantic coupling (shared
  contracts, codegen); declaring `dependsOn` edges is the team's
  responsibility, and the non-goal stands.
