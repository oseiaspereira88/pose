# ADR: Unified Review Convergence and Auto Attestation

## Status
Accepted (2026-08-14) — implemented by spec `pose-unified-review-convergence`

## Context
POSE v1.1.0 introduced opt-in `ReviewBundles` and `ReviewAttestations` (`review.json` schema 2) to eliminate the self-invalidating closeout loops of legacy Markdown review attempts (`rvw-*.md`). However, having two coexisting review paths created friction:
1. Operators and subagents had to choose between legacy `pose review record` and `pose review bundle/attest`.
2. `pose review attest` required specifying verbose and brittle tool dispositions manually via CLI flags, hindering automated orchestration.
3. CI workflows and subagents lacked a single automated reconciliation command to attach fresh deterministic validation results (`delivery-validation.json`) directly to a sealed bundle attestation.

## Decision
1. **Unify into a Single Review Track:** Deprecate the legacy Markdown review attempt track (`pose review record`) and establish `ReviewBundle` + `ReviewAttestation` as the single canonical review mechanism for all POSE scopes (specs, milestones, roadmaps).
2. **Automated Evidence & Tool Reconciliation (`pose review auto-attest`):** Add an automated attestation command that inspects the effective review plan for a sealed bundle, matches executed deterministic checks from structured validation artifacts (e.g., `.pose/results/delivery-validation.json` and tool execution logs), validates required evidence classes, and emits the canonical attestation without manual flag formatting.
3. **Subagent & CI Workflow Integration:** Provide built-in support for subagents (`same-actor-separate-execution`) and GitHub Actions to seal bundles, run validation, and reconcile attestations in one deterministic step.

## Consequences
- **Positive:** A single, clean review pipeline without dual tracks; no more self-invalidation cycles; seamless subagent delegation and CI automation.
- **Negative:** Legacy `pose review record` invocations are sunsetted in favor of bundle-based reviews.
