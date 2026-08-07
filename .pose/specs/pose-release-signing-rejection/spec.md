---
slug: pose-release-signing-rejection
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-release-signing
priority: 1
components: release
delivers: governance:release-signing-rejection-proof
---

# Spec: The identity gate is proven to reject, not just asserted to

## 1. Intent

### Goal
Exercise the artifact-identity gate's rejection path on every CI run, so
`pose-release-signing` R3 rests on a demonstration instead of a waiver.

### Business value
`tests/release/verify.sh` has always contained the rejection logic: an archive
without a Sigstore bundle, without a CycloneDX SBOM, or with a malformed one
fails the release. Every run that ever executed it verified artifacts that were
correctly signed, so the failure path was never taken. R3 was waived at retrofit
for exactly that reason — "the gate's rejection behaviour is asserted rather
than demonstrated".

A gate whose failure path never runs is indistinguishable, from the evidence
available, from a gate that cannot fail. That is the whole value of the signing
chain: the guarantee is not that signed artifacts pass, it is that unsigned ones
do not.

### Constraints
- Shell is the harness, never the POSE runtime — the same rule the other
  `tests/release/` scripts follow.
- The proof must run on ordinary CI, not only during a release. A check that
  only executes at cut time is what left this path unexercised for four
  releases.

### Non-goals
- Testing Sigstore itself, or asserting on `cosign`'s own failure messages. The
  subject is this repository's gate, not the upstream verifier.
- Producing genuinely signed-then-tampered artifacts. Signing in CI needs the
  release environment; the rejection cases here need no valid signature to be
  meaningful.

---

## 2. Requirements

### Functional
- R1: A deterministic check shall build deliberately broken artifact sets and
  require `verify.sh` to reject each one.
- R2: The cases shall cover the distinct rejection reasons the gate claims:
  a missing signature bundle, a missing SBOM, an empty artifact set, and a
  malformed SBOM.
- R3: The check shall run in ordinary CI and fail the job when the gate accepts
  something it must reject.

### Non-functional
- The check runs in seconds against synthetic fixtures in a temporary directory
  and leaves nothing behind.

### Security
- No signing material, credentials or network access is involved: the fixtures
  are unsigned by construction, which is the point.

### Compatibility
- No product change; `verify.sh` itself is untouched.

---

## 3. Technical Plan

### Affected areas
- `tests/release/verify-negative.sh` — the new harness.
- `.github/workflows/ci.yml` — the step that runs it.
- `.pose/specs/pose-release-signing/spec.md` — R3 leaves the waived state.

### Artifacts
- created: .pose/specs/pose-release-signing-rejection/spec.md
- created: tests/release/verify-negative.sh
- modified: .github/workflows/ci.yml
- modified: .pose/specs/pose-release-signing/spec.md

### Delivery targets
- governance:release-signing-rejection-proof module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- An "empty artifact set" case passes for the wrong reason if `verify.sh` ever
  exits non-zero before reaching its own emptiness check. The four cases are
  distinguished by construction, not by exit code, but nothing enforces that a
  future refactor keeps them distinguishable.

---

## 4. Tasks

### Planning
- [x] Enumerate the rejection reasons verify.sh actually claims
- [x] Confirm each is reachable with synthetic, unsigned fixtures

### Implementation
- [x] R1: build the broken sets and require rejection
- [x] R2: cover missing bundle, missing SBOM, no archives, malformed SBOM
- [x] R3: wire the check into ci.yml so it runs on every push

### Validation
- [x] Run the check locally and confirm all four paths reject
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Assert on the gate's behaviour, not its source: construct an artifact directory
per rejection reason, run the real `verify.sh` against it, and require a
non-zero exit. A case that is *accepted* is reported by name and fails the run,
so a regression identifies which guarantee was lost.

### Deterministic checks

#### Security / Contract
- Command: `bash tests/release/verify-negative.sh`
- Scope: the artifact-identity gate's rejection path
- Expected: all four broken sets rejected; exit 0

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, development build `pose 0.20.3-dev`.
- Notes: output was `PASS` for an archive without a signature bundle, without a
  CycloneDX SBOM, with no archives at all, and with a malformed SBOM —
  "verify-negative: all rejection paths exercised", exit 0.

### Results summary
- Successes: four rejection paths demonstrated; R3 of pose-release-signing
  leaves the waived state
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:release-signing-rejection-proof evidence:integration check:delivery-integration test:tests/release/verify-negative.sh — the harness builds each broken set and requires `verify.sh` to reject it, reporting by name any that is accepted
- R2 [satisfied] check:verify-negative — the four cases are a missing signature bundle, a missing CycloneDX SBOM, an artifact set with no archives, and a malformed SBOM
- R3 [satisfied] report:.github/workflows/ci.yml — the step runs in the ordinary `ci` job, not only at release time, and its non-zero exit fails the build

### Known gaps
- A valid signature over a tampered payload is not among the cases: producing
  one needs the release environment's signing identity. What is proven is that
  the gate rejects absent and malformed provenance, which is the class that
  four releases left undemonstrated.

---

## 7. Final Report

### Delivered scope
The artifact-identity gate's rejection path is exercised on every CI run
against four deliberately broken artifact sets. `pose-release-signing` R3 is
recorded as satisfied by demonstration instead of waived.

### Files and modules changed
- tests/release/verify-negative.sh
- .github/workflows/ci.yml
- .pose/specs/pose-release-signing/spec.md

### Validation executed
- Command: `bash tests/release/verify-negative.sh`
- Result: all four rejection paths exercised; exit 0

### Residual risks
- The tamper case — valid signature, altered payload — remains outside CI's
  reach without the release signing identity.

### Follow-ups

- [open] Consider exercising the tamper case during a release run, where a signing identity exists: sign a fixture, alter it, and require the gate to reject it. That is the one rejection reason still asserted rather than demonstrated. (owner:@pose-maintainers crit:medium review:2026-10-02)
