---
slug: pose-verifier-extension-install-cwd
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-verifier-assets-variable-fix
priority: 1
components: release
delivers: governance:verifier-extension-install-cwd
---

# Spec: The verification installs the extension where it says it does

## 1. Intent

### Goal
Make the consumer-side extension-install check install into the instance it
created, by entering that directory first.

### Business value
The v0.20.2 verification proved the part that mattered most and had never been
proven: the published extension verifies against its real Sigstore signature,
and a tampered copy is rejected. Both ran with real cosign against a real
published artifact for the first time.

The install leg failed, and not because installing is broken: `pose extension
install` acts on the current directory, not on a target argument. Without a `cd`
it installed wherever the verifier happened to be standing, and the check then
asserted the rule was missing from the instance.

This is the third failure in a row in the same block I added, each a different
mistake of mine, and this particular one had already bitten me in a manual test
earlier the same day — it is recorded as finding F2 on
`pose-extension-reference-publication`. Writing the finding down did not stop me
repeating it in code.

### Constraints
- The fix must be proven by reproducing the gate's exact command locally, not by
  a syntax check. Syntax checking already failed to catch the previous defect.

### Non-goals
- Changing `pose extension install` to take a target directory. That is a real
  usability question, already tracked, but not a release-verification fix.

---

## 2. Requirements

### Functional
- R1: The extension-install check shall enter the created instance before
  installing, so the assertion and the installation refer to the same directory.
- R2: The check shall keep failing when the extension genuinely does not install.

### Non-functional
- No new dependency on the verifier's working directory.

### Security
- The install still verifies the signature first; nothing is relaxed.

### Compatibility
- No product change.

---

## 3. Technical Plan

### Affected areas
- `tests/release/independent-verify.sh` — the extension-install gate.

### Artifacts
- created: .pose/specs/pose-verifier-extension-install-cwd/spec.md
- modified: tests/release/independent-verify.sh
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: docs-site/docs/ci.md

### Delivery targets
- governance:verifier-extension-install-cwd module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- The block still has legs that only execute with cosign present, which is not
  the case locally. Those were exercised by the v0.20.2 run and passed.

---

## 4. Tasks

### Planning
- [x] Review the whole block for other mistakes of the same class

### Implementation
- [x] R1: cd into the instance before installing
- [x] R2: confirm the assertion still fails on a genuine miss

### Validation
- [x] Reproduce the gate's exact command locally
- [x] Re-audit every variable the script reads
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Reproduce the gate's command verbatim against a locally built binary and a real
package, which is what the two previous attempts skipped. Then re-audit the
script for unassigned reads, and re-read the whole block for the same class of
error — wrong sibling variable, implicit working directory, relative path.

### Deterministic checks

#### Security / Contract
- Command: the gate's `bash -c` line, reproduced locally against a built binary
- Scope: install into the created instance and assert the rule landed
- Expected: passes; asserted before cutting

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.20.3-dev; cosign absent locally.
- Notes: run 31211887058 reported `PASS: reference extension verifies against
  its published signature` and `PASS: a tampered extension is rejected`, with
  only the install leg failing. The block review also removed a dead assignment
  of `ext_pkg` that the `if` branch immediately overwrote.

### Results summary
- Successes: gate command reproduced locally; variable audit clean; strict gates
  green
- Failures: none
- Warnings: the cosign-dependent legs cannot run locally; they passed in the
  v0.20.2 verification

### Requirement trace
- R1 [satisfied] governance:verifier-extension-install-cwd evidence:integration check:delivery-integration check:verifier-install-gate — the gate's exact command was reproduced locally and passes; it enters the instance before installing
- R2 [satisfied] check:verifier-install-gate — the assertion is unchanged and still tests for the installed rule, so a genuine install failure still fails the gate

### Known gaps
- v0.20.2 stays `published` without `verified`, for the same immutable-tag
  reason as v0.20.1.
- Nothing prevents the next block from repeating this class of error; the
  shellcheck follow-up on `pose-verifier-assets-variable-fix` is the systemic
  answer.

---

## 7. Final Report

### Delivered scope
The extension-install check enters the instance it created before installing.
The block was also re-read end to end for the same class of defect, which
removed one dead assignment.

### Files and modules changed
- tests/release/independent-verify.sh

### Validation executed
- Command: the gate's `bash -c` line, reproduced locally
- Result: PASS

### Residual risks
- Three consecutive releases failed verification on this block, each for a
  different mistake. The block is now small and reviewed, but the pattern says
  the local proof was the weak link, not the code.

### Follow-ups

- [open] `pose extension install` takes no target directory and silently uses the current one. It caused a manual mis-install and then this gate defect. Consider an explicit target argument. (owner:@pose-maintainers crit:medium review:2026-10-02)
