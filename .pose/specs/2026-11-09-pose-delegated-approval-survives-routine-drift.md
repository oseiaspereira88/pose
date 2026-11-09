---
slug: pose-delegated-approval-survives-routine-drift
status: in-progress
created_at: 2026-11-09
completed_at:
supersedes:
depends_on: pose-done-spec-keeps-its-sealed-plan
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:delegated-approval-survives-routine-drift
---

# Spec: The approval of a delegated review survives routine drift of a done scope

## 1. Intent

### Goal

A scope that is done keeps the approval a delegated review gave it when the repository moves on, as it keeps the approval a person attested.

### Business value

Every spec closed with a delegated review turned non-terminal as soon as another spec changed a file the closed one had sealed. `validation-matrix.json` is sealed by nearly every spec and `POSE.md` by many, so five specs closed in one day (B31, B33, B36, the material-binding spec and the sealed-plan spec) read "no review attempt exists" a day later, and `pose closeout-check` named each as unreviewed. The retention rule for a done scope exists (`retainedCompletedReview`): a person's attestation is kept over drift. It never applied to a run-backed one, because verifying a run proof asks whether the run still governs the repository as it is now, which no completed scope can answer after anything moved. Measured on a real tree: the proof of B31 failed with "the bundle was superseded or its inputs changed while the run was going" and that of B36 with "sealed bytes changed for .pose/indexes/validation-matrix.json".

### Constraints

- The mechanisms are the existing ones: the retention of a completed scope, the signed run proof and its record-level verification (`verifyRunProofCore`, already used by action fulfillment for records kept after the scope moved on).
- A live scope keeps every check it has, and issuing a proof is unchanged.
- Nothing about who may sign, or what a proof binds, is relaxed.

### Non-goals

- Retaining an approval for an open scope.
- Changing what a reviewer must judge or how a run is dispatched.
- Re-judging an approval recorded before this change.

## 2. Requirements

### Functional

- R1: When a scope is done, the approval of a delegated run attested over its sealed bundle shall remain retained after a later change to a file of that bundle's subject, with the same disclosure as any retained approval.
- R2: The verification of a run-backed approval of a done scope shall check what the sealed record says: the proof's signature under an issuer that still holds its pin, the project and audience, the exact run record, output, transcript and provenance it binds, the order of the run, its proof and the attestation, a completed and complete draft, no superseding run, the persisted record equal to the copy, an intact transcript, and the bundle the run names readable with the digest, scope and sealed commit it recorded.
- R3: That verification shall not ask whether the run still governs the repository as it is now: not that the bundle equals what the scope prepares today, not that the persisted inputs materialize in the working tree, not that the policy and adapter are the ones of the run, and not that the brief renders to the same bytes under today's templates.
- R4: An open scope shall be verified as before: a delegated approval of an open scope is not retained over a change to what it sealed.
- R5: A run record whose transcript was altered, or whose issuer no longer holds its pin, shall not keep its approval, drift or none.
- R6: Issuing a run proof and verifying one for a live scope shall keep every check they have.
- R7: The assurance read for a retained delegated approval shall be the one of the sealed run, verified, so that `review verify`, closeout and the assurance disclosure agree.

### Non-functional

- No new file, field or command; the verification of a done scope reads the same records as before.

### Security

- Retention never creates an approval: it keeps one the sealed record proves was given. The pin of the issuer is checked at every read, so revoking trust ends the retention.

### Compatibility

- Scopes closed with a delegated review that read non-terminal read terminal again, with no re-review. A scope whose record was altered still does not.

## 3. Technical Plan

### Affected areas

`internal/pose/review_dispatch.go` (the run's record checks split from its present-state checks), `internal/pose/review_run_provenance.go` (the proof verified as sealed), the retained path of `computeReviewVerification`, the ADR on native signed run proofs and the manual.

### Artifacts

- created: .pose/specs/2026-11-09-pose-delegated-approval-survives-routine-drift.md
- created: .pose/starts/pose-delegated-approval-survives-routine-drift.json
- created: .pose/changelogs/unreleased/pose-delegated-approval-survives-routine-drift.md
- modified: .pose/adr/2026-10-25-bind-delegated-review-authority-to-native-signed-run-proofs.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- created: pose-mcp/internal/pose/retained_run_approval_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegated-approval-survives-routine-drift module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None to a schema. `ReviewRunRecordStands` and `VerifyReviewRunProofAsSealed` are added; `ReviewRunStillGoverns` is the first composed with the present-state checks.

## 4. Tasks

- [x] Reproduce the loss on a real run-backed approval and on the five specs of this repository
- [x] Split what a run's record says from whether it still governs
- [x] Verify a done scope's run-backed approval as sealed
- [x] Test the negatives: an open scope, an altered transcript, a revoked issuer
- [x] Measure the real repository, old engine against new
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3
- Minimal option: keep verifying the run as governing the present and lift only the materialization check
- Selected option: verify a done scope's approval as sealed, with the record's own checks, and keep the live checks for live scopes
- Rationale: three different present-state checks failed in turn (the prepared bundle, the materialization of the sealed bytes, the brief re-rendered under today's template), and each is a question about a repository that has moved on; the record, the signature and the pin are what a closed approval stands on, as they are for a person's attestation
- Consequences: the retention of a delegated approval ends only by revoking the issuer's pin, altering the record, or reopening the scope
- Falsifier: a run-backed approval retained over drift whose record was altered, whose issuer lost its pin, or whose scope is open

### Decision D2
- Basis: R3
- Minimal option: re-verify the brief against the template version recorded in the run
- Selected option: do not re-render the brief for a done scope
- Rationale: the run records the digest of the brief it was sent and the proof signs it; templates changed text under the same version in this repository (a mistake, recorded as a follow-up), and a re-render would have refused honest approvals; the immutability of a template version is its own guard
- Consequences: the brief a delegated approval was given on is trusted as the signed record says
- Falsifier: an approval whose signed brief digest names bytes the run was not sent

## 6. Validation

### Strategy

A real run-backed approval is produced by the fixture, the scope is marked done and a file of its subject changes: the approval must stay, and must go when the transcript is altered or the issuer loses its pin. Then the five specs of this repository that lost their closeout are read by the old and the new engine.

### Deterministic checks

`go test -race` of the tests named in the trace; the strict matrix in a clean clone.

### Requirement trace
- R1 [satisfied] test:TestADelegatedApprovalSurvivesRoutineDriftOfADoneScope check:delegated-approval-survives-routine-drift-integration
- R2 [satisfied] test:TestARetainedDelegatedApprovalRefusesATamperedTranscript test:TestARetainedDelegatedApprovalRefusesARevokedIssuer
- R3 [satisfied] test:TestADelegatedApprovalSurvivesRoutineDriftOfADoneScope
- R4 [satisfied] test:TestADelegatedApprovalOfAnOpenScopeDoesNotSurviveDrift
- R5 [satisfied] test:TestARetainedDelegatedApprovalRefusesATamperedTranscript test:TestARetainedDelegatedApprovalRefusesARevokedIssuer
- R6 [satisfied] test:TestConcludeReviewRunRecordsTheAttestation
- R7 [satisfied] test:TestTheAssuranceOfARetainedDelegatedApprovalIsVerified

### Known gaps
- A run's brief is not re-rendered for a done scope. The text of `review-brief/4` changed three times under one version in this repository, so the runs briefed with its earlier texts could not re-render; the template version is not guarded against that. It is a follow-up.
- A run superseded by a later run is refused at the record check; the test covers the altered transcript and the revoked issuer, not a superseding run, which the record check already performs for a live scope.
- Measured on this repository: five specs that read non-terminal under the previous engine (B31, B33, B36, material-binding and sealed-plan) read terminal under this one.

---

## 7. Final Report

### Delivered scope
A done scope keeps the approval a delegated run backed over later drift of the files it sealed (R1); the verification of that approval is the sealed record's own (R2, R3); an open scope, an altered transcript and a revoked issuer keep their refusal (R4, R5); live verification and issuing are unchanged (R6); the assurance of a retained delegated approval is the sealed run's (R7). Left out: guarding the immutability of a template version, and a superseding-run test of its own.

### Files and modules changed
- pose-mcp/internal/pose (review_dispatch.go, review_run_provenance.go, retained_run_approval_test.go)
- ADR native signed run proofs (amendment); POSE.md and its pt-BR and scaffold copies; the validation matrix gains `delegated-approval-survives-routine-drift-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head, and the suites of `internal/pose`, `internal/cli` and `internal/mcpserver`
- Result: recorded by the sealed validation evidence; independent delegated review (Claude Opus 5.5, independent by model) concluded with `pose review conclude`.

### Residual risks
- The brief of a retained delegated approval is trusted as signed, not re-derived.
- The retention ends only through revocation, alteration or reopening, never by age.

### Follow-ups

- [open] Guard the immutability of every review brief template version by a golden digest of its rendering, so a text change must take a new version, and name the versions in use. (owner:@pose-maintainers crit:medium review:2027-01-08)
