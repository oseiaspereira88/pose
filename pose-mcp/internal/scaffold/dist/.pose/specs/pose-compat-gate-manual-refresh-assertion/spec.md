---
slug: pose-compat-gate-manual-refresh-assertion
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-manual-distribution-merge, pose-compat-gate-candidate-integrity
priority: 1
components: release
delivers: governance:compat-gate-manual-assertion
---

# Spec: The upgrade gate asserts preservation, not a frozen manual

## 1. Intent

### Goal
Make the compatibility gate's upgrade check assert what the upgrade contract
actually promises — that an instance keeps what it wrote — instead of asserting
that `AGENTS.md` never changes.

### Business value
The v0.20.0 cut is blocked by both of its upgrade pairs, and the upgrade is not
at fault: it refreshes the manual, preserves the instance's customization, keeps
a `.pose-backup`, and the upgraded instance passes `pose check --strict`. The
gate fails it anyway because `check_upgrade_pair` compares `AGENTS.md` byte for
byte before and after.

That assertion was written when a plain `pose upgrade` did not touch the
manuals. `pose-manual-distribution-merge` changed that deliberately — the whole
point was that canonical manual content should reach installed repositories. The
two contracts have contradicted each other ever since; it stayed invisible only
because the support window was empty, so no pair ever ran. The first release
whose manual actually changed exposes it.

### Constraints
- The replacement must not be weaker. Dropping the assertion entirely would let
  a genuine content-losing regression through.

### Non-goals
- Changing what `pose upgrade` does to the manuals. It behaves as specified.

---

## 2. Requirements

### Functional
- R1: The upgrade pair check shall require the instance's own customization to
  survive the upgrade, in the manual itself or in its `.pose-backup`.
- R2: The check shall keep failing when an upgrade loses that customization from
  both places.

### Non-functional
- The gate stays runnable offline once artifacts are cached.

### Security
- No change to what the gate executes or trusts.

### Compatibility
- No product change: this corrects a test assertion.

---

## 3. Technical Plan

### Affected areas
- `tests/release/compat.sh` — the `check_upgrade_pair` manual assertion.

### Artifacts
- created: .pose/specs/pose-compat-gate-manual-refresh-assertion/spec.md
- modified: tests/release/compat.sh

### Delivery targets
- governance:compat-gate-manual-assertion module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- Accepting "present in the backup" is weaker than "present in the manual". It
  is the correct bar: the engine is allowed to refresh an engine-owned section,
  and the backup is exactly the guarantee that nothing is lost when it does.

---

## 4. Tasks

### Planning
- [x] Confirm the upgrade itself is correct before touching the assertion

### Implementation
- [x] R1: assert customization survival instead of a frozen manual
- [x] R2: keep the failing case failing

### Validation
- [x] Both upgrade pairs pass against published artifacts
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Reproduce the pair by hand first, to establish that the upgrade preserves
content and the instance stays valid — only then change what the gate asserts.
Then run the real gate over both published pairs.

### Deterministic checks

#### Security / Contract
- Command: `bash tests/release/compat.sh v0.20.0`
- Scope: compatibility gate over the 0.19.0 and 0.18.2 upgrade pairs
- Expected: `Result: COMPATIBLE — release gate passed.`

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.20.0-dev.
- Notes: reproduced 0.19.0 → candidate manually. `AGENTS.md` changed as designed,
  the instance's customization survived in the file, a `.pose-backup` was
  written, and `pose check --strict` passed. Only then was the assertion changed.

### Results summary
- Successes: both upgrade pairs, full test suite, strict gates
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:compat-gate-manual-assertion evidence:integration check:delivery-integration check:compat-gate — the gate reports PASS for 0.19.0 → 0.20.0 and 0.18.2 → 0.20.0, whose manuals legitimately changed
- R2 [satisfied] check:compat-gate — the check still fails when the customization is in neither the manual nor its backup; only the "identical file" condition was replaced

### Known gaps
- The assertion covers `AGENTS.md`, the file the fixture customizes. `POSE.md`
  goes through the same merge and is not separately asserted.

---

## 7. Final Report

### Delivered scope
`check_upgrade_pair` now requires the instance's customization to survive an
upgrade, in the manual or in its backup, instead of requiring the manual to be
unchanged. That is the guarantee `pose-manual-distribution-merge` actually
makes.

### Files and modules changed
- tests/release/compat.sh

### Validation executed
- Command: `bash tests/release/compat.sh v0.20.0`
- Result: COMPATIBLE, both pairs exercised

### Residual risks
- A regression that loses content from the manual *and* fails to write a backup
  is still caught; one that writes a backup but corrupts the manual body is not
  distinguished from a legitimate refresh.

### Follow-ups

- [open] Assert the same preservation property for POSE.md, which takes the same merge path but is not customized by the upgrade-lab fixture. (owner:@pose-maintainers crit:low review:2026-11-20)
