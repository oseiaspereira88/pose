---
slug: pose-attest-refuses-what-verify-rejects
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
delivers: capability:attestation-preflight
---

# Spec: An approving attestation the verifier would reject is refused before it is written

## 1. Intent

### Goal

Run the verifier's attestation rules when an approving attestation is previewed or applied, so `pose review attest` and `pose review auto-attest` refuse, with the verifier's own reasons, what `pose review verify` would reject.

### Business value

In the measurement for act-08a9fd2d9a0e50bb every malformed attestation — a structural fact left unmapped, a basis the scope does not declare, a mapping without a reason, an integrity fact accepted as a risk, evidence absent from the bundle — was previewed as `plan=record` and written with `--apply`; the refusal only appeared at `verify`. Each mistake left an immutable attestation behind and cost a second one. The maintainer asked for the refusal to come first.

### Constraints

The rules are the verifier's, not a copy: one function decides both. A non-approving decision (`changes-requested`, `rejected`) is an audit record and is still written, because the verifier rejects it for closeout by definition.

### Non-goals

Changing any verification rule.

## 2. Requirements

### Functional

- R1: When an approving attestation would fail verification against its sealed bundle, `pose review attest` shall refuse it with each reason the verifier gives and exit non-zero, both in preview and with `--apply`, and shall write nothing.
- R2: `pose review auto-attest --apply` shall refuse the same way an assembled attestation the verifier would reject.
- R3: A `changes-requested` or `rejected` attestation shall still be recorded.
- R4: An approving attestation the verifier accepts shall be previewed and recorded as before.
- R5: The manual shall state that attest refuses what verify would reject.

### Non-functional

- No extra reads beyond the bundle and policy the verifier already reads.

### Security

- None beyond the shared constraints.

### Compatibility

- A caller that recorded an approving attestation only to watch verify reject it now gets the rejection from attest.

## 3. Technical Plan

### Affected areas

Review attestation recording in the CLI, the store's verification entry point, manuals.

### Artifacts

- created: .pose/specs/2026-10-05-pose-attest-refuses-what-verify-rejects.md
- created: .pose/starts/pose-attest-refuses-what-verify-rejects.json
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/cli/review_closeout.go
- created: pose-mcp/internal/cli/review_attest_preflight_test.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- renamed: .pose/changelogs/unreleased/pose-attest-refuses-what-verify-rejects.md -> .pose/changelogs/v7.0.0/pose-attest-refuses-what-verify-rejects.md

### Delivery targets

- capability:attestation-preflight module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Verification rules that read live state (a stored signature) run at attest time against the state of that moment; the verifier still runs again at closeout.

## 6. Validation

### Strategy

CLI fixtures with a sealed bundle: an approving attestation citing evidence the bundle does not seal, a negative decision, and a valid approval, previewed and applied.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run AttestPreflight`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestAttestPreflightRefusesWhatVerifyWouldReject test:TestReviewAttestationWontFixCannotApproveThroughCLI check:attest-preflight-integration
- R2 [satisfied] <auto-attest records only a complete attestation and now runs the same preflight before RecordReviewAttestation> test:TestAttestPreflightRefusesWhatVerifyWouldReject check:attest-preflight-integration
- R3 [satisfied] test:TestAttestPreflightStillRecordsNegativeDecisions check:attest-preflight-integration
- R4 [satisfied] test:TestAttestPreflightLetsAValidApprovalThrough test:TestReviewRecordDelegatesToBundleAttestationWhenAdopted check:attest-preflight-integration
- R5 [satisfied] test:TestAttestPreflightIsDocumented check:attest-preflight-integration

### Known gaps

The auto-attest path is covered by the shared preflight call, not by a dedicated fixture: auto-attest only records a complete attestation assembled from sealed results, and no fixture yet assembles one the verifier rejects.

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None beyond the technical risk above.

### Follow-ups

None.
