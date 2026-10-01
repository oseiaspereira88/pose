---
slug: pose-live-policy-reads-through-aliases
status: in-progress
created_at: 2026-10-01
completed_at:
delivers: capability:sealed-policy-read-guard
components: pose-mcp
task_type: bugfix
changelog: none
---

# Spec: Live policy reads through aliases

## 1. Intent

Close the regex blind spot in the sealed-review security regression guard.

## 2. Requirements

- R1: The live-read guard shall detect policy field reads through local aliases.
- R2: Comments, string literals and shadowed variables shall not create false
  policy reads. Existing signing-only live access remains the required contract.

## 3. Technical Plan

### Artifacts

- modified: pose-mcp/internal/pose/live_policy_reads_test.go
- created: .pose/specs/2026-10-01-pose-live-policy-reads-through-aliases.md

### Delivery targets

- capability:sealed-policy-read-guard module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

These targets identify local engine behavior exercised by this scope. Test-only
changes strengthen its regression coverage; they do not introduce a new runtime
endpoint or claim composition in Harne8.

## 4. Tasks

- [x] Parse the selected validator/helper bodies with standard Go AST tooling.
- [x] Follow local assignment aliases with lexical identifier binding.
- [x] Test chained aliases, literals, comments and shadowing.

## 5. Decisions

Use Go's standard parser; add no dependency and change no production contract.
The guard covers direct local aliases, not interprocedural or reflective flow.

## 6. Validation

Synthetic snippets must expose an obligation-changing field through two aliases
and avoid reporting text or a locally shadowed unrelated policy variable.
Run the actual signing-only and sealed-gate guards against production source.

### Execution log

- Strict module validation passed 45/45 on 2026-10-01; full canonical validation
  is regenerated before sealing the governed review.

- 2026-10-01: targeted synthetic guards and production-source/workflow guards
  passed. No native runner or release execution was required.

### Requirement trace

- R1 [satisfied] capability:sealed-policy-read-guard evidence:integration test:TestLivePolicyReadsFollowAliasesAndIgnoreText
- R2 [satisfied] test:TestLivePolicyReadsFollowAliasesAndIgnoreText test:TestOnlyTheSigningGateIsReadFromLivePolicy

## 7. Final Report

Implementation validated; governed review and closeout pending.
