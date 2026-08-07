---
slug: pose-verifier-assets-variable-fix
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-extension-reference-publication
priority: 1
components: release
delivers: governance:verifier-assets-variable
---

# Spec: The extension verification reads the variable it actually has

## 1. Intent

### Goal
Fix the consumer-side extension check in `independent-verify.sh`, which reads
`$dir` — a variable that script does not define — and aborts the whole
verification under `set -u`.

### Business value
v0.20.1 published correctly: 31 assets including the signed reference extension.
The verification job then died at line 96 before checking anything, so the
release cannot reach `verified` and the extension chain's consumer half remains
unproven on a real artifact.

The variable is `$assets` there; `$dir` is what `verify.sh` calls it. Two
sibling scripts, two names for the same thing, and the block was written against
the wrong one.

v0.20.1 can never be verified: `verify-release.yml` checks out the tag being
verified, and the tag carries the broken script. The fix only takes effect for a
release whose tag already contains it.

### Constraints
- The fix must be provable before cutting, since the previous attempt passed a
  syntax check and still failed at runtime.

### Non-goals
- Changing where `verify-release.yml` gets its script from. Verifying a release
  with the code that release shipped is a defensible design; whether it should
  instead use the newest verifier is a separate question.

---

## 2. Requirements

### Functional
- R1: The consumer-side extension check shall read the artifact directory
  variable that `independent-verify.sh` defines.
- R2: No variable read by that script shall be unassigned, so `set -u` cannot
  abort it again.

### Non-functional
- The check keeps skipping cleanly when a release carries no packaged extension.

### Security
- No change to what is verified or trusted.

### Compatibility
- No product change.

---

## 3. Technical Plan

### Affected areas
- `tests/release/independent-verify.sh` — the extension block.

### Artifacts
- created: .pose/specs/pose-verifier-assets-variable-fix/spec.md
- modified: tests/release/independent-verify.sh
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: docs-site/docs/ci.md

### Delivery targets
- governance:verifier-assets-variable module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- `bash -n` does not catch an unbound variable, so syntax checking is not
  sufficient evidence here — that is exactly how the defect shipped.

---

## 4. Tasks

### Planning
- [x] Confirm the tag carries the broken script, so v0.20.1 cannot be verified

### Implementation
- [x] R1: read `$assets`
- [x] R2: confirm every variable the script reads is assigned

### Validation
- [x] Static check over all variables the script reads
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
A syntax check already proved insufficient once. The check here extracts every
variable the script reads and asserts each one is assigned somewhere in it —
which is the class of defect that shipped. Full runtime proof needs cosign and a
published artifact, so it lands on the next cut.

### Deterministic checks

#### Security / Contract
- Command: static extraction of read-vs-assigned variables over `tests/release/independent-verify.sh`
- Scope: every `$var` the script reads
- Expected: no variable read without an assignment

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.20.2-dev; cosign absent locally.
- Notes: runs 31210599090 and 31210767642 both aborted at
  `line 96: dir: unbound variable`. The second was dispatched after the fix was
  on main, which is how the tag-checkout behaviour was confirmed.

### Results summary
- Successes: variable audit clean, strict gates green
- Failures: none
- Warnings: the consumer-side extension verification runs for the first time on
  this cut

### Requirement trace
- R1 [satisfied] governance:verifier-assets-variable evidence:integration check:delivery-integration report:tests/release/independent-verify.sh — the block reads `$assets`, the variable the script assigns for the downloaded release directory
- R2 [satisfied] check:verifier-variable-audit — every variable the script reads has an assignment; the audit reports none missing

### Known gaps
- v0.20.1 stays `published` and can never become `verified`: its tag carries the
  broken verifier and tags are immutable.
- The consumer-side extension legs still have not executed against a published
  artifact. This cut is their first opportunity.

---

## 7. Final Report

### Delivered scope
The consumer-side extension check reads `$assets` instead of `$dir`, and the
script has no unassigned reads left.

### Files and modules changed
- tests/release/independent-verify.sh

### Validation executed
- Command: static read-vs-assigned variable audit
- Result: no unassigned reads

### Residual risks
- Nothing enforces the audit: the next block written against the wrong sibling
  script fails the same way, at release time.

### Follow-ups

- [open] Add shellcheck to CI over tests/release/*.sh and tests/install/*.sh: it catches unbound variables that `bash -n` cannot, which is precisely how this reached a release. (owner:@pose-maintainers crit:medium review:2026-09-18)
- [open] Decide whether verify-release.yml should verify published artifacts with the verifier from the tag or from the default branch. The tag-based checkout made a fixable verifier defect permanent for v0.20.1. (owner:@pose-maintainers crit:medium review:2026-10-02)
