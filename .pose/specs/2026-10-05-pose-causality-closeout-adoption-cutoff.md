---
slug: pose-causality-closeout-adoption-cutoff
status: in-progress
created_at: 2026-10-05
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: spec:pose-abm-causality-attestation@defect-fix
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:causality-closeout-adoption-cutoff
---

# Spec: Causality closeout and a late-adopted overlay apply from their adoption date

## 1. Intent

### Goal

Let a project adopt causality closeout and the structural overlay without holding work that predates them: `causality_closeout_adopted_at` limits the contract to scopes with a spec created on or after the date, and `overlay_adopted_at` does the same for each overlay profile adopted later.

### Business value

Today the contract is stamped on every bundle sealed after adoption, so a spec already in progress is held to it at its next seal; in the measurement for act-08a9fd2d9a0e50bb the agency-readiness pilot, created the day before, would have owed a causal mapping it was never planned for. Selecting `structural-materiality@1` has the same reach: the structural-causality contract is already in force here, so adding the overlay alone would charge mappings to every in-flight spec. The maintainer chose an adoption cutoff, as for atomic start.

### Constraints

Same pattern as the other adoption dates: without a date the behaviour is unchanged; nothing sealed is rewritten; a scope whose specs cannot be dated is held to the contract, never exempted.

### Non-goals

Changing what the contract checks.

## 2. Requirements

### Functional

- R1: With `causality_closeout_adopted_at`, a bundle for a scope whose specs were all created before the date shall not be stamped with causality-closeout and shall say why; a scope with a spec created on or after the date, or with a spec that has no creation date, shall be stamped as before; without the date every new bundle shall be stamped, as before.
- R2: An overlay listed in `overlay_adopted_at` with a date shall not be selected for a scope whose specs were all created before that date, and the plan's explain trail shall say so; a structural overlay skipped this way shall not resolve the structure; an overlay without a date shall be selected as before.
- R3: The policy reader shall refuse an unparsable date in either key and an `overlay_adopted_at` entry for an overlay that `overlay_profiles` does not list.
- R4: Effective governance shall state the causality cutoff and each dated overlay.
- R5: The manual shall describe both cutoffs.

### Non-functional

- A skipped structural overlay costs no Git read.

### Security

- An undated spec never escapes the contract.

### Compatibility

- Additive optional keys; existing policies unchanged.

## 3. Technical Plan

### Affected areas

Review policy reader, bundle sealing, review plan overlay selection, effective governance, manuals.

### Artifacts

- created: .pose/specs/2026-10-05-pose-causality-closeout-adoption-cutoff.md
- created: .pose/starts/pose-causality-closeout-adoption-cutoff.json
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/causality_closeout.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/effective_governance.go
- created: pose-mcp/internal/pose/causality_cutoff_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-causality-closeout-adoption-cutoff.md

### Delivery targets

- capability:causality-closeout-adoption-cutoff module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- The cutoff reads `created_at`, as the atomic-start cutoff does: a spec created before the date and only worked on after it is treated as older.

## 6. Validation

### Strategy

A fixture with one spec created before and one after each date, sealed and planned with and without the keys, plus malformed and orphaned keys.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose -run CausalityCutoff`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestCausalityCutoffStampsOnlyScopesCreatedOnOrAfterTheDate check:causality-cutoff-integration
- R2 [satisfied] test:TestCausalityCutoffSkipsALateOverlayForOlderScopes check:causality-cutoff-integration
- R3 [satisfied] test:TestCausalityCutoffRefusesMalformedOrOrphanedDates check:causality-cutoff-integration
- R4 [satisfied] test:TestCausalityCutoffIsStatedByEffectiveGovernance check:causality-cutoff-integration
- R5 [satisfied] test:TestCausalityCutoffIsDocumented check:causality-cutoff-integration

### Known gaps

The `created_at` proxy described under Technical risks.

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

The `created_at` proxy.

### Follow-ups

None.
