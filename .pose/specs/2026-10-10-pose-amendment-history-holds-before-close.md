---
slug: pose-amendment-history-holds-before-close
status: done
created_at: 2026-10-10
completed_at: 2026-10-25
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers: capability:amendment-history-before-close
---

# Spec: The amendment history holds before a spec closes

## 1. Intent

### Goal

A started spec's first amendment shall not leave its earlier nodes unacknowledged, and a spec whose amendment history would fail once done shall be refused before `pose close`, not after.

### Business value

On 2026-10-10 two specs (pose-dist's `pose-review-subject-classifies-engine-records` and Harne8's `harne8-adopt-pose-v7-2-0`) recorded their first `pose amend` event as `semantic` or `added` with no baseline. Their atomic start records held a baseline, but the amendment log did not, so every node present since the start read as "added without an amendment event". The amendment gate in `lint-spec` runs only on done specs, so the failure appeared only after `pose close`, and an independent reviewer had to unwind an approved close.

### Constraints

The amendment log stays append-only; nothing rewrites recorded events.

## 2. Requirements

### Functional

- R1: When `pose amend` records the first event of a spec that has a recorded atomic start, the engine shall seed the log with a baseline taken from the start record before the change event.
- R2: `pose close` shall refuse a spec whose amendment history would fail `lint-spec --strict` once done, naming the findings.
- R3: `lint-spec` on an in-progress spec shall report, as a warning, the amendment findings it would raise once done.

## 3. Technical Plan

### Artifacts

- created: .pose/adr/2026-10-24-seed-first-amendment-from-recorded-start.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/technical-debt.md
- created: .pose/changelogs/unreleased/pose-amendment-history-holds-before-close.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/reports/2026-10-24-amendment-history-before-close.md
- created: .pose/specs/2026-10-10-pose-amendment-history-holds-before-close.md
- created: .pose/starts/pose-amendment-history-holds-before-close.json
- modified: .pose/state/technical-debt.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/amend.go
- created: pose-mcp/internal/cli/amendment_history_close_test.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json
- created: pose-mcp/internal/pose/amendment_history_close_test.go
- modified: pose-mcp/internal/pose/amendments.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:amendment-history-before-close module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/amend.go

## 4. Tasks

- [x] Resolve canonical authority, assess freshness and preserve R1–R3.
- [x] Reproduce first amendment and premature close failures.
- [x] Seed only from a valid recorded start; share history gate and warning.
- [x] Execute domain/CLI regressions and canonical full validation.
- [x] Separate review and native closeout.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3
- ADR: [First amendment baseline](../adr/2026-10-24-seed-first-amendment-from-recorded-start.md)
- Knowledge: knowledge:adr-sealed-review-bundles-review, knowledge:pose-b07-implementation
- Decision: Seed the first non-baseline amendment from the validated start record, and reuse the same history findings in lint and pre-transition close.
- Rationale: Start records preserve the old nodes; current text cannot reconstruct that baseline. Keep opt-in journals, schemas and append-only history.

## 6. Validation

### Strategy

Reproduce both 2026-10-10 cases in fixtures: a started spec amended once fails the done lint today and passes with R1; a close over a failing history is refused with R2.

The module has high criticality. Use `pose-test-plan` before code. Required checks:

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| First semantic/added/transition amendment, schema 1/2, flat/folder, invalid or absent start | `cd pose-mcp && go test ./internal/pose ./internal/cli -run AmendmentHistory -count=1` | start nodes acknowledged; changed/new nodes use explicit event; invalid input writes zero bytes; no legacy auto-baseline |
| Close with invalid/malformed history and in-progress lint warning; valid positive close | same registered integration command | refuse before lifecycle write, actionable finding; warning is nonfatal; valid history closes |
| Full suite/build/vet and composition | `pose validate --strict --json-out .pose/results/delivery-validation.json` | all checks pass with current provenance |
| Concurrency | `cd pose-mcp && go test -race ./internal/pose ./internal/cli` | shared lifecycle lock serializes amend/close; no races |
| Security | `cd pose-mcp && govulncheck ./...` plus pinned CI gitleaks | no vulnerabilities or secrets |

### Requirement trace

- R1 [satisfied] test:TestAmendmentHistoryFirstChangeSeedsRecordedStart test:TestAmendmentHistoryFlatAddedAndTransitionKeepEarlierNodes
- R2 [satisfied] test:TestAmendmentHistoryCloseRefusesBeforeTransition test:TestAmendmentHistoryCloseRechecksUnderLockAndValidControl
- R3 [satisfied] test:TestAmendmentHistoryInProgressWarnsBeforeDone

## 7. Final Report

### Delivered scope

Implemented and natively closed on 2026-10-24 the existing R1–R3 as the prerequisite of B11, under current continuation authorization. Readonly context resolved `proj.pose-dist`; assessment freshness reused. The first amendment copies only recorded start state; existing journals stay append-only. Lint warns during execution and close checks before transition and again under the shared lifecycle lock. No release, policy adoption, parent gitlink update or remote publication. Final strict validation passed 151/151 at HEAD `8a35140`; sealed review and guarded closeout are recorded in [implementation evidence](../reports/2026-10-24-amendment-history-before-close.md).

### Residual risks

Explicit `--baseline` retains its existing current-state semantics; review must judge materiality. Cooperative locking cannot prevent arbitrary filesystem edits. Append/Sync failure is surfaced without truncation; inspect partial journals explicitly. No journal remains opt-out. B11 still owns semantic terminal verification and plan/transport parity; this prerequisite does not claim that delivery.

### Follow-ups

- [open] Extend the generic adversarial corpus with first-amendment/start and preclose-history cases; permanent regression producer already covers R1–R3. (owner:@pose-maintainers crit:low review:2026-11-08)
