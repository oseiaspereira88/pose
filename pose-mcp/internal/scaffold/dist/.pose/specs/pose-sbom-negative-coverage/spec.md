---
slug: pose-sbom-negative-coverage
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-sbom-license-coverage-gate
priority: 1
components: release
delivers: governance:sbom-negative-coverage
---

# Spec: The SBOM assertions are demonstrated, not just written

## 1. Intent

### Goal
Make every SBOM rejection path reachable by the negative harness, including the
license-coverage floor, and prove each one.

### Business value
`pose-release-signing-rejection` was written because a gate whose failure path
never runs is indistinguishable from a gate that cannot fail. The SBOM
assertions were in exactly that position, and the harness built to fix it could
not reach them: inside `verify.sh` they sit behind signature verification, and
the synthetic fixtures are unsigned by construction, so every run died before
any SBOM assertion was evaluated.

`pose-sbom-license-coverage-gate` recorded this as a known gap on the day it
shipped the floor — its rejection path had a real measurement behind it, but no
standing check. That is the gap this closes.

Splitting the assertions into their own script is what makes them testable.
It is also honest about the structure: signature identity and inventory
completeness are separate guarantees that happened to live in one file.

### Constraints
- No bypass in the production path. An `SBOM_SKIP_*` escape hatch would make
  the fixtures easier to write and would add a way to silence a release gate;
  the fixture is built from the real `go.mod` instead.
- `verify.sh` keeps its behaviour: the same assertions, in the same order,
  failing the release the same way.

### Non-goals
- Testing syft. The subject is this repository's assertions over an inventory,
  not the tool that produces one.

---

## 2. Requirements

### Functional
- R1: The SBOM assertions shall be invocable independently of signature
  verification.
- R2: The negative harness shall exercise the license-coverage floor at zero
  coverage and at partial coverage below the floor.
- R3: It shall exercise the zero-component case, which must not divide into a
  pass.
- R4: It shall include a positive control — a compliant inventory that must be
  accepted — so the rejections above are not the harness refusing everything.
- R5: `verify.sh` shall keep failing the release when the delegated assertions
  fail.

### Non-functional
- The fixtures are generated in a temp directory and removed on exit.

### Security
- No new bypass and no new authority: the split is read-only refactoring plus
  fixtures.

### Compatibility
- `SBOM_MIN_LICENSE_PCT` keeps its meaning and default.

---

## 3. Technical Plan

### Affected areas
- `tests/release/verify-sbom.sh` — the extracted assertions.
- `tests/release/verify.sh` — delegates to them.
- `tests/release/verify-negative.sh` — four new cases.

### Artifacts
- created: .pose/specs/pose-sbom-negative-coverage/spec.md
- created: tests/release/verify-sbom.sh
- modified: tests/release/verify.sh
- modified: tests/release/verify-negative.sh

### Delivery targets
- governance:sbom-negative-coverage module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- The positive control depends on parsing `pose-mcp/go.mod` for direct
  dependencies, duplicating the extraction `verify-sbom.sh` performs. If the two
  ever disagree, the control fails in a way that looks like a real defect. The
  duplication is deliberate — the alternative was a skip flag in the release
  path — but it is duplication.
- Reachability is not equivalence: the harness now exercises the assertions
  through `verify-sbom.sh`, while a release exercises them through `verify.sh`.
  R5 covers the wiring, but the two entry points could diverge.

---

## 4. Tasks

### Planning
- [x] Establish why the existing harness could not reach these assertions
- [x] Reject the skip-flag approach in favour of a truthful fixture

### Implementation
- [x] R1: extract the assertions into their own script
- [x] R2: zero-coverage and below-floor cases
- [x] R3: zero-component case
- [x] R4: positive control from the real go.mod
- [x] R5: `verify.sh` delegates and still fails on their failure

### Validation
- [x] Run the negative harness
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
A negative harness is only trustworthy with a positive control. Seven
rejections prove nothing on their own — a harness that rejects everything
produces the same output — so a compliant inventory must be accepted in the
same run, by the same script.

That control earned its place immediately: the first version of it failed,
because the synthetic components did not name the direct production
dependencies. The honest fix was to build the fixture from the real `go.mod`
rather than to add a skip flag to a release gate.

### Deterministic checks

#### Security / Contract
- Command: `bash tests/release/verify-negative.sh`
- Scope: four signature-path rejections, three SBOM rejections, one positive control
- Expected: all eight report PASS; exit 0

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64.
- Notes: the harness reports rejection for an archive without a signature
  bundle, without a CycloneDX SBOM, with no archives, with a malformed SBOM,
  with an SBOM resolving no licenses, with coverage at 50% against a 75% floor,
  and with zero components — then accepts the compliant inventory. Shellcheck
  passes at warning severity over the new and modified scripts.

### Results summary
- Successes: three previously unreachable rejection paths demonstrated, plus a
  positive control
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:sbom-negative-coverage evidence:integration check:delivery-integration test:tests/release/verify-sbom.sh — the assertions run standalone against an artifact directory, with no signature dependency
- R2 [satisfied] check:verify-negative — zero coverage and 50%-against-75% are both rejected, the second being the quiet degradation the floor exists for
- R3 [satisfied] check:verify-negative — a zero-component inventory is rejected with its own message rather than passing the ratio comparison
- R4 [satisfied] check:verify-negative — a compliant inventory built from the real `go.mod` is accepted in the same run
- R5 [satisfied] test:tests/release/verify.sh — `verify.sh` delegates to the extracted script and records a failure when it exits non-zero

### Known gaps
- The positive control re-implements the direct-dependency extraction, so the
  two parsers can drift apart.
- The harness reaches the assertions through `verify-sbom.sh` while a release
  reaches them through `verify.sh`; the wiring is asserted, the equivalence of
  the two paths is not.
- A valid signature over a tampered payload remains outside the harness, as it
  was before.

---

## 7. Final Report

### Delivered scope
The SBOM assertions are their own script, the negative harness exercises three
rejection paths it previously could not reach, and a positive control proves it
is not simply rejecting everything.

### Files and modules changed
- tests/release/verify-sbom.sh
- tests/release/verify.sh
- tests/release/verify-negative.sh

### Validation executed
- Command: `bash tests/release/verify-negative.sh`
- Result: eight cases, all as specified

### Residual risks
- Two entry points to the same assertions, with only the wiring asserted.

### Follow-ups

- [open] The positive control parses `pose-mcp/go.mod` for direct dependencies, duplicating what `verify-sbom.sh` does; the two can drift and the failure would look like a real defect rather than a fixture problem. Consider having the harness ask `verify-sbom.sh` for the dependency list it will require, so there is one parser. (owner:@pose-maintainers crit:low review:2026-11-06)
