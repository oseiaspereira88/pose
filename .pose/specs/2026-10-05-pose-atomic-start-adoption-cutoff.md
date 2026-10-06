---
slug: pose-atomic-start-adoption-cutoff
status: in-progress
created_at: 2026-10-05
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: spec:pose-abm-atomic-start@defect-fix
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:atomic-start-adoption-cutoff
---

# Spec: Atomic start applies from its adoption date

## 1. Intent

### Goal

Let a project adopt atomic start without blocking specs that were already in progress: `atomic_start_adopted_at` limits the capability to specs created on or after that date.

### Business value

The shadow in pose-abm-capability-adoption measured 39 closeout-blocking `start-reconciliation` obligations on adoption, one per spec already in progress, with no command to remedy them. The maintainer chose, in action request act-fa1d72f029d567a0, to add an adoption cutoff and then adopt (option B).

### Constraints

Same pattern as the other contract cutoffs; nothing recorded is rewritten; without the date the capability keeps applying to every spec.

### Non-goals

Recording baselines for older specs (option C, not chosen).

## 2. Requirements

### Functional

- R1: With `atomic_start_adopted_at`, an in-progress spec created before the date and without a start record shall be reported as `legacy-unbaselined` with an explanatory note and no reconciliation.
- R2: A spec created on or after the date without a start record shall still need reconciliation; without the date every such spec shall, as before.
- R3: An unparsable date shall be refused when the policy is read.
- R4: Effective governance shall state whether a cutoff exists and what it means.
- R5: A start record under `.pose/starts/` shall be classified in the review subject as a governance record, so a spec started with `pose start --apply` can be sealed.

### Non-functional

- None beyond the shared constraints.

### Security

- None.

### Compatibility

- Additive optional key; existing policies unchanged.

## 3. Technical Plan

### Affected areas

Start status, review policy reader, effective governance, manuals.

### Artifacts

- created: .pose/specs/2026-10-05-pose-atomic-start-adoption-cutoff.md
- modified: pose-mcp/internal/pose/start.go
- created: pose-mcp/internal/pose/start_cutoff_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/effective_governance.go
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_bundle_start_record_test.go
- created: .pose/specs/2026-10-05-pose-atomic-start-adoption-cutoff.amendments.jsonl
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-atomic-start-adoption-cutoff.md

### Delivery targets

- capability:atomic-start-adoption-cutoff module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- The cutoff reads `created_at`: a spec created before the date and moved to in-progress by hand after it is also treated as older. Stated in the manual note's scope; a start record still governs any spec started with `--apply`.

## 6. Validation

### Strategy

A fixture with one spec created before and one after the cutoff, the policy with and without the date, and a malformed date.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose -run AtomicStartCutoff`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestAtomicStartCutoffReportsOlderSpecsWithoutBlocking check:atomic-start-cutoff-integration
- R2 [satisfied] test:TestAtomicStartCutoffReportsOlderSpecsWithoutBlocking check:atomic-start-cutoff-integration
- R3 [satisfied] test:TestAtomicStartCutoffReportsOlderSpecsWithoutBlocking check:atomic-start-cutoff-integration
- R4 [satisfied] <the atomic-start capability description names the cutoff> test:TestUnadoptedCapabilitiesAreSupportedAndNotInForce check:effective-governance-integration

### Known gaps

The `created_at` proxy described under Technical risks.

## 7. Final Report

### Delivered scope

`atomic_start_adopted_at` limits atomic start to specs created on or after it; older in-progress specs carry a note instead of a reconciliation, and effective governance describes the cutoff.

### Residual risks

The `created_at` proxy.

### Follow-ups

