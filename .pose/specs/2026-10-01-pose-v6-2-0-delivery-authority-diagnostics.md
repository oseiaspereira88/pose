---
slug: pose-v6-2-0-delivery-authority-diagnostics
status: in-progress
created_at: 2026-10-01
completed_at:
delivers: surface:delivery-authority-diagnostics, capability:delivery-authority-diagnostics
priority: 1
components: pose-mcp
task_type: bugfix
---

# Spec: Delivery and authority diagnostics

## 1. Intent

### Goal

Correct duplicated delivery failures and authority validation inconsistencies.

### Constraints

Keep strict failures blocking. Preserve signed-authority fail-closed semantics.
Native platform verification is independent and deferred.

## 2. Requirements

### Functional

- R1: A failure to build the shared delivery graph shall be reported once,
  preserving the responsible spec in target parsing/validation diagnostics.
- R2: Authority claims shall use the envelope schema constant rather than the
  unrelated review profile schema constant.
- R3: Malformed human authority pins shall be rejected with a policy diagnostic;
  valid issuer and sha256 digest pins shall remain accepted.

- R4: Excluded lifecycle inputs shall use repository-relative paths so review
  diagnostics are portable across checkouts.

## 3. Technical Plan

### Artifacts

- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/surface_check.go
- created: pose-mcp/internal/cli/check_delivery_diagnostics_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_authority_test.go
- created: .pose/specs/2026-10-01-pose-v6-2-0-delivery-authority-diagnostics.md
- created: .pose/changelogs/unreleased/pose-v6-2-0-delivery-authority-diagnostics.md
- modified: .pose/reports/2026-10-01-standard-validate-native.md
- modified: .pose/reports/history/standard-validate-native.jsonl

### Technical risks

Malformed pins previously failed at authority matching; earlier rejection must
not accidentally authorize them or reject existing valid pins.

### Delivery targets

- surface:delivery-authority-diagnostics module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
- capability:delivery-authority-diagnostics module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

These targets identify local engine behavior exercised by this scope. Test-only
changes strengthen its regression coverage; they do not introduce a new runtime
endpoint or claim composition in Harne8.

## 4. Tasks

- [x] Reproduce duplicate findings using malformed draft plus completed specs.
- [x] Preserve source identity and report shared failure once.
- [x] Correct schema comparison and validate issuer pins.
- [x] Run targeted negative/positive tests and full module validation.

## 5. Decisions

Report a shared graph error once and stop that delivery check: dependent target
checks cannot establish acceptance without the graph. Other check domains continue.

## 6. Validation

Before implementation: run the malformed-draft regression in strict and tolerant
modes; test malformed and valid pins, signed authority acceptance and invalid
schema rejection. Run full Go tests/vet/build through module validation.

### Execution log

- Strict module validation passed 45/45 on 2026-10-01; full canonical validation
  is regenerated before sealing the governed review.

- 2026-10-01: malformed draft reproduced three findings for three unrelated
  completed specs in both modes; corrected regression now emits one finding
  naming spec:broken, preserving strict error versus tolerant warning.
- Targeted authority positive/negative tests, portable lifecycle test,
  gitignored/submodule discovery tests and native validation fixture passed.
- Discovery gitignore handling and the old self-exec fixture concern are already
  implemented in current code; no duplicate changes are needed.

- Full strict module validation passed 45/45 checks (build, all tests, vet and
  integration checks). Follow-up portability/schema regressions also passed.

### Requirement trace

- R1 [satisfied] surface:delivery-authority-diagnostics capability:delivery-authority-diagnostics evidence:integration check:delivery-integration test:TestCheckDeliveryGraphFailureReportedOnce
- R2 [satisfied] test:TestABMReviewAuthorityValid test:TestABMReviewAuthorityRejectsUnsupportedClaimSchema
- R3 [satisfied] test:TestHumanAuthorityIssuerPinsValidated
- R4 [satisfied] test:TestReviewBundleExcludedLifecyclePathIsPortable

## 7. Final Report

### Delivered scope

Four bounded corrections implemented; governed review and closeout pending.

### Follow-ups

None introduced yet.
