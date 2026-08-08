---
slug: pose-sbom-license-coverage-gate
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-sbom-license-inventory
priority: 1
components: release
delivers: governance:sbom-license-coverage-gate
---

# Spec: An SBOM that stops resolving licenses fails the release

## 1. Intent

### Goal
Assert a minimum license coverage in the artifact-identity gate, so a silent
regression to an empty inventory blocks the release instead of shipping.

### Business value
`pose-sbom-license-inventory` raised coverage from 1 component of 27 to 24 by
enabling one syft setting. Nothing holds it there. The resolution depends on
that setting surviving in `.goreleaser.yaml` and on the module cache being
populated when syft runs — either can stop being true without any step failing,
which is exactly how four releases shipped an SBOM advertised as carrying
licenses and carrying one.

The gate already inspects the SBOM: schema validity, and the presence of every
direct production dependency. Coverage is the property it was missing, and the
one that actually degraded.

### Constraints
- The floor must sit below normal variation and far above collapse. A floor set
  at the current value converts every added dependency without a resolvable
  license into a release blocker.
- It belongs in `verify.sh`, beside the other SBOM assertions, rather than in a
  new gate: the artifact-identity check is where SBOM claims are already made.

### Non-goals
- Requiring a license for every component. Three are unresolvable by
  construction — the two project modules and the binary itself — and covered by
  the repository LICENSE.
- Validating that a resolved license is *correct*. This asserts presence, which
  is what regressed.

---

## 2. Requirements

### Functional
- R1: The artifact-identity gate shall compute the fraction of SBOM components
  carrying a non-empty `licenses` field and fail below a declared floor.
- R2: The failure shall report the observed ratio and the floor.
- R3: An SBOM with zero components shall fail rather than divide by zero or
  pass vacuously.

### Non-functional
- One `jq` pass per SBOM, alongside the passes already made.

### Security
- Read-only inspection of a file the gate already parses.

### Compatibility
- The floor is overridable through `SBOM_MIN_LICENSE_PCT` for a release that
  legitimately needs to go lower, so the decision is explicit rather than a
  code edit under pressure.

---

## 3. Technical Plan

### Affected areas
- `tests/release/verify.sh` — the SBOM assertion block.

### Artifacts
- created: .pose/specs/pose-sbom-license-coverage-gate/spec.md
- modified: tests/release/verify.sh

### Delivery targets
- governance:sbom-license-coverage-gate module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- **The floor is a judgement, and it will age.** 75% was chosen against a
  measured 88.9% with 27 components; a project with 200 components and a
  different resolution rate would need a different number. Nothing revisits it.
- The rejection path is not exercised by `verify-negative.sh`. Reaching the
  coverage check requires an artifact set that clears signature verification
  first, which the synthetic fixtures deliberately cannot do. It was instead
  proven against two real SBOMs, which is stronger evidence but not a standing
  check — recorded as a known gap rather than glossed.

---

## 4. Tasks

### Planning
- [x] Measure real coverage with and without the syft setting
- [x] Choose a floor from the measurement, not from intuition

### Implementation
- [x] R1: compute coverage and fail below the floor
- [x] R2: report the ratio and the floor
- [x] R3: reject an empty component list

### Validation
- [x] Accept the resolved SBOM, reject the degraded one
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Both sides of the threshold are proven against real SBOMs rather than fixtures.
Scanning the built binary twice — once with the syft setting the release
config sets, once without — produces exactly the two states that matter: the
current one and the one that shipped for four releases. A floor that does not
separate those two is the wrong floor, and this is the only way to know.

### Deterministic checks

#### Security / Contract
- Command: `bash tests/release/verify-negative.sh`
- Scope: the artifact-identity gate's existing rejection paths, unaffected
- Expected: all four still rejected

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64, syft 1.50.0 against a locally built `pose` binary.
- Notes: scanning without `SYFT_GOLANG_SEARCH_LOCAL_MOD_CACHE_LICENSES` yields
  1/27 components with a license; scanning with it yields 24/27 — reproducing
  independently the measurement `pose-sbom-license-inventory` recorded. Applied
  to those two SBOMs, the 75% floor accepts 24/27 and rejects 1/27.
  Shellcheck passes at warning severity and `verify-negative.sh` still exercises
  all four rejection paths.

### Results summary
- Successes: floor separates the resolved SBOM from the degraded one
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:sbom-license-coverage-gate evidence:integration check:delivery-integration test:tests/release/verify.sh — the gate counts components with a non-empty `licenses` field and fails below `SBOM_MIN_LICENSE_PCT`, default 75
- R2 [satisfied] check:sbom-license-coverage — the failure names the observed ratio and the floor, so the message says how far off it is rather than only that it failed
- R3 [satisfied] check:sbom-license-coverage — a zero-component SBOM fails with its own message instead of reaching the ratio comparison

### Known gaps
- The rejection path has no standing check: `verify-negative.sh` cannot reach
  it, because its fixtures fail signature verification first by design. The
  proof is a real measurement recorded here, not a repeating gate — the same
  category of gap `pose-release-signing-rejection` was written to close for the
  signature path.
- 75% is a judgement calibrated for 27 components and nothing revisits it.

---

## 7. Final Report

### Delivered scope
The artifact-identity gate fails a release whose SBOM resolves licenses for
fewer than 75% of components, reporting the ratio and the floor.

### Files and modules changed
- tests/release/verify.sh

### Validation executed
- Command: syft scans with and without the license setting, floor applied to both
- Result: 24/27 accepted, 1/27 rejected; existing rejection paths unaffected

### Residual risks
- The check runs only during a release, so a regression is caught at cut time
  rather than at the commit that caused it.

### Follow-ups

- [open] The coverage gate's rejection path is not exercised by any standing check, because verify-negative.sh's fixtures fail signature verification before reaching it. Splitting verify.sh's SBOM assertions into a separately callable path would let the negative harness cover them, and would close the same gap for any future SBOM assertion. (owner:@pose-maintainers crit:medium review:2026-10-02)
