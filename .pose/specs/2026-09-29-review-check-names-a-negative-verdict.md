---
slug: review-check-names-a-negative-verdict
status: done
created_at: 2026-09-29
completed_at: 2026-09-29
supersedes:
depends_on: retained-review-survives-an-invalid-newer-approval
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:completed-review-verdict-reporting
---

# Spec: `review-check` names a negative verdict on a closed scope

## 1. Intent

### Goal

For a closed scope whose newest review rejected it or requested changes,
`pose review-check` and `pose check --strict` report that review, its decision and
its open findings.

### Business value

After a changes-requested review of two closed specs in audio-relay, `review-check`
and `check --strict` said `no review attempt exists` for both, while `review verify`
showed `changes-requested` with the findings. An operator reading the gate goes
looking for a missing review record instead of the findings that block the scope.

### Constraints

The approval outcome is unchanged: such a scope is still not approved. Only the
report changes. Scopes that predate review bundles keep the legacy path.

### Non-goals

Changing when a closed scope's approval is retained.

## 2. Requirements

- R1: When a closed scope's newest attested bundle carries a `rejected` or
  `changes-requested` attestation, `review-check` reports that attestation as the
  current review, a blocker naming its decision, and one blocker per open or
  changes-requested finding.
- R2: The scope stays unapproved.

## 3. Technical Plan

In `ReviewCheck`, after no approval is retained for a closed scope, look up the
newest attested bundle; when its latest attestation is negative, report it instead
of falling through to the legacy lookup, which only knows review attempts.

### Artifacts

- created: .pose/specs/2026-09-29-review-check-names-a-negative-verdict.md
- created: .pose/changelogs/unreleased/review-check-names-a-negative-verdict.md
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/completed_review_retention_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- capability:completed-review-verdict-reporting module:pose-mcp/internal/pose profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

Reporting only. Reverting restores the misleading message.

## 4. Tasks

- [x] Reproduce the message on audio-relay's two reviewed specs and compare with `review verify`.
- [x] Report the negative verdict and its findings for a closed scope.
- [x] Pin the report in a test that fails without the fix.

## 5. Decisions

### Decision D1

- Status: active
- Report from the newest attested bundle rather than from `VerifyReviewBundle`,
  whose result for a closed scope is `superseded` as soon as routine drift moves the
  preparation, and which then carries no attestation to report.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Negative verdict report | `go test ./internal/pose -run TestReviewCheckNamesANegativeVerdict -count=1` | decision and findings reported; not approved |
| Full suite | `go test ./... -count=1` | pass |

### Execution log

2026-09-29: in audio-relay, after changes-requested attestations were recorded for
`stream-session-reconnect-recovery`, `review-check` and `check --strict` reported
`no review attempt exists`, while `review verify` reported `changes-requested` with
findings f4 and f5. With the fix, the new test passes; with the lookup disabled it
fails with `[no review attempt exists for spec:backend]`. The full suite passes.

### Closeout

2026-09-29 UTC. Full matrix 42/42 into the results path; `surface-check --strict` with
0 findings; bundle `rvb-a96a5aafe8131eba`, 39 evidence items; attestation
`rva-ab0bdeae3026626c`, `agent:claude-opus-5-5`, approved with five explicit judgments.

### Requirement trace

- R1 [satisfied] capability:completed-review-verdict-reporting evidence:integration check:completed-review-retention-integration test:TestReviewCheckNamesANegativeVerdictOnAClosedScope — the changes-requested attestation is reported as current,
  with a blocker naming its decision and one per changes-requested finding
- R2 [satisfied] capability:completed-review-verdict-reporting evidence:integration check:completed-review-retention-integration test:TestReviewCheckNamesANegativeVerdictOnAClosedScope — the scope stays unapproved

## 7. Final Report

### Scope delivered

`review-check` and `check --strict` name the negative verdict and its open findings
for a closed scope instead of reporting that no review exists.

### Residual risks

None beyond the message text, which the test pins by substring.

### Follow-ups

None.
