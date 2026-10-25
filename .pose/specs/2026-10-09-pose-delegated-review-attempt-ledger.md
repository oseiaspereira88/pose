---
slug: pose-delegated-review-attempt-ledger
status: done
created_at: 2026-10-09
completed_at: 2026-10-25
supersedes:
depends_on: pose-delegated-review-dispatch
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers: capability:delegated-attempt-ledger
---

# Spec: Every delegated review attempt stays on the record

## 1. Intent

### Goal

Keep every delegated review run, rejections included, so a review cannot be retried until it approves.

### Business value

Nothing on 2026-10-09 would have stopped the implementer from discarding the reviewer's "changes required" and running another reviewer. The three rejections that found real defects were valuable precisely because they stood.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

### Constraints

Append-only; no command deletes or rewrites a run. The delegated review contract is done and the ADR's maintainer decisions are accepted. This scope does not adopt delegated-review, authenticate adapter identity or apply a draft as an attestation. Implementation authorized by continuation on 2026-10-25, baseline `2c0bb3e`.

## 2. Requirements

### Functional

- R1: Runs under `.pose/review-runs/` shall be append-only and content-addressed, and `pose history-check` shall treat a rewritten or deleted run as a violation.
- R2: A new dispatch on a bundle that already has a completed run shall be refused unless the bundle changed or `--reason` is recorded with the new run.
- R3: `pose review verify` shall disclose earlier rejecting runs for the same scope next to the approving one.
- R4: `pose insights` shall report runs per bundle, approval rate per adapter, and reruns without a bundle change.

## 3. Technical Plan

### Affected areas

Ledger layout, the rerun rule in dispatch, verify disclosure and insights metrics.

### Artifacts

- created: .pose/adr/2026-10-25-reserve-delegated-attempts-before-launch-and-witness-immutable-runs.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- created: .pose/changelogs/unreleased/pose-delegated-review-attempt-ledger.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/reports/2026-10-25-b16-attempt-ledger.md
- created: .pose/specs/2026-10-09-pose-delegated-review-attempt-ledger.md
- created: .pose/starts/pose-delegated-review-attempt-ledger.json
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/adversarial_corpus_gates_test.go
- modified: pose-mcp/internal/cli/b07_r3_probe_test.go
- modified: pose-mcp/internal/cli/b07_r4b_probe_test.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/historycheck.go
- modified: pose-mcp/internal/cli/insights.go
- created: pose-mcp/internal/cli/review_attempt_ledger_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_dispatch.go
- modified: pose-mcp/internal/cli/review_dispatch_test.go
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- created: pose-mcp/internal/mcpserver/review_attempt_ledger_test.go
- modified: pose-mcp/internal/pose/insights.go
- created: pose-mcp/internal/pose/review_attempt_ledger.go
- created: pose-mcp/internal/pose/review_attempt_ledger_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegated-attempt-ledger module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_dispatch.go

## 4. Tasks

- [x] Append-only run ledger
- [x] Rerun rule
- [x] Verify disclosure
- [x] Insights metrics

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4
- ADR: [Durable attempts and immutable witnesses](../adr/2026-10-25-reserve-delegated-attempts-before-launch-and-witness-immutable-runs.md)
- Knowledge: knowledge:pose-b07-implementation, knowledge:adr-sealed-review-bundles-review
- Decision: Reserve one immutable ordinal slot before adapter launch, witness each content-addressed run before publication, and expose declared draft outcomes without granting authority.
- Rationale: A scan of surviving run files alone cannot detect deletion or serialize concurrent quota decisions.

## 6. Validation

### Strategy

Fixture with a rejecting run: a rerun on the same bundle without a reason is refused; verify shows the rejection; deleting a run fails history-check.

### Required execution plan

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Reservation race, quota, rerun reason, loss/tampering, failed publication and legacy compatibility | `cd pose-mcp && go test ./internal/pose ./internal/cli ./internal/mcpserver -run AttemptLedger -count=1` | one winner per slot; no adapter starts on refusal; missing/rewritten records visible; metadata parity |
| Concurrent reservation safety | `cd pose-mcp && go test -race ./internal/pose -run AttemptLedger -count=1` | no race or double reservation |
| Full deterministic composition | `pose validate --strict --json-out .pose/results/delivery-validation.json` with TMPDIR=/var/tmp | all required checks and registered integration pass |
| Security | `cd pose-mcp && govulncheck ./...` and pinned CI gitleaks | no vulnerabilities or secrets |

### Requirement trace

- R1 [satisfied] test:TestAttemptLedgerRunAndWitnessLossAndRewrite test:TestAttemptLedgerHistoryCheckRejectsDeletedCommittedEvidence
- R2 [satisfied] test:TestAttemptLedgerReasonQuotaAndReservationLoss test:TestAttemptLedgerConcurrentReservationsAndCrashStayVisible test:TestAttemptLedgerDispatchReasonBeforeLaunchAndDisclosure
- R3 [satisfied] test:TestAttemptLedgerMCPDisclosesPriorRejectionAndDeclaredMetrics test:TestAttemptLedgerDispatchReasonBeforeLaunchAndDisclosure
- R4 [satisfied] test:TestAttemptLedgerDeclaredOutcomesAndLegacyIdentity test:TestAttemptLedgerScopedWitnessAndHistoricalWindow

## 7. Final Report

### Delivered scope

Implemented durable ordinal reservations, immutable run witnesses, rerun admission, verify disclosure and declared draft metrics across text/JSON/HTML/MCP. No delegated-review policy, vendor identity proof, automatic attestation, run continuation or adapter containment is adopted. Native close completed on 2026-10-25 after 153/153 canonical checks, artifact reconciliation (35/35), composition with zero findings, and approved attestation `rva-83cfa3925739793c` on bundle `rvb-da193121009a577e`.

### Residual risks

New writers must be upgraded together; old writers cannot coordinate reservations. Filesystem/Git anchors are trusted; content hashes do not authenticate the producer or prove erased unanchored history existed. Incomplete reservations intentionally block further dispatch and remain explicit; the canonical continuity spec owns supported resume, without deleting or rewriting ledger artifacts. Read bounds return incomplete coverage and never a clean truncated history. Legacy unreserved runs are counted individually without fabricating a reservation. Declared draft rates are not attested approval rates.

### Follow-ups

No new residual task. Run continuity, authenticated provenance and containment remain in their existing canonical specs; this delivery does not change their disposition.
