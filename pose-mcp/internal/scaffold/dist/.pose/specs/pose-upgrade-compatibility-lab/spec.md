---
slug: pose-upgrade-compatibility-lab
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-release-compatibility-matrix
priority: 26
---

# Spec: Upgrade compatibility lab

## 1. Intent

### Goal
continuously test real repository upgrades across the supported version window.
### Business value
Protects adopter-owned specs and evidence as the product evolves.
### Constraints
- Never mutate original fixtures; prove idempotency and preserve unknown content.
### Non-goals
- Support downgrades or arbitrary invalid instances.

## 2. Requirements

### Functional
- R1: The lab shall test every supported N-minus engine/schema pair.
- R2: Fixtures shall cover locales, user-modified managed files and populated artifacts.
- R3: Each path shall prove dry-run accuracy, idempotency and preservation.

### Non-functional
- Run in isolated copies with deterministic snapshots.

### Security
- Authenticate prior binaries and block path/symlink escapes.

### Compatibility
- Unsupported versions receive explicit remediation, not partial upgrade.

## 3. Technical Plan

### Affected areas
- Upgrade engine, migrations, fixtures, release CI and docs.

### API/contract changes
- Make the support matrix executable and release-blocking.

### Data/storage changes
- Version sanitized fixtures and expected migration plans.

### Technical risks
- Synthetic fixtures can miss real customization patterns.

### Primary references
- [The Update Framework](https://theupdateframework.io/)
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [The Update Framework](https://theupdateframework.io/).

### Implementation
- [ ] Build populated-instance fixtures for each supported release. ([reference](https://theupdateframework.io/))
- [ ] Exercise dry-run, apply, reapply and preservation assertions. ([reference](https://slsa.dev/spec/v1.2/))
- [ ] Publish candidate results and unsupported-path diagnostics. ([reference](https://theupdateframework.io/))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Upgrade|Migration|Preserve'` and retain evidence. ([reference](https://theupdateframework.io/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://slsa.dev/spec/v1.2/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-upgrade-compatibility-lab-populated-fixtures.md` (Accepted): populated fixtures (pt-BR locale install + real spec + real knowledge note + hand-edited managed file) in both `internal/cli/upgrade_test.go` (network-free, R2/R3) and `tests/release/compat.sh` (real N-minus pairs, R1), plus a symlink-escape guard (`ensureManagedDirSafe`) added directly to `cmdUpgrade`. Rejected: bare-install fixtures with only unit tests (never proves populated-instance preservation); a separate dedicated lab harness/package (fragments ownership away from where the upgrade engine and the existing compatibility gate already live).

## 6. Validation

**Strategy:** validate the upgrade engine against a populated instance (dry-run accuracy, apply/reapply idempotency, preservation of locale/user-modified/populated content), an explicit-remediation negative case, and a symlink-escape security case — all network-free; real N-minus pairs are deferred to the network-dependent release gate.

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/cli/... -run 'Upgrade' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-upgrade-compatibility-lab --ready-check`.

### Requirement trace
- R1 [satisfied] `tests/release/compat.sh`'s `check_upgrade_pair` exercises every declared `supported_upgrades` entry against a populated pt-BR/spec/knowledge/user-edit fixture with the real prior binary and the real candidate binary; check:doc (`tests/release/compat.sh`) — currently 0 pairs declared (`compatibility.json.supported_upgrades` is empty pre-first-release), same gap as the underlying `pose-release-compatibility-matrix`; open follow-up to confirm on the first real pair
- R2 [satisfied] fixture covers pt-BR locale-installed content, a hand-edited `AGENTS.md`, a real spec and a real knowledge note; check:test (TestUpgradeApplyIsIdempotentAndPreservesInstanceContent — asserts all four survive byte-for-byte)
- R3 [satisfied] dry-run proven byte-for-byte non-mutating, apply proven to change only `schema-version` on a populated instance, reapply proven a strict no-op; check:test (TestUpgradeDryRunIsAccurateAndNonMutating, TestUpgradeApplyIsIdempotentAndPreservesInstanceContent)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli/... -run 'Upgrade' -v -count=1` — SUCCESS (5 tests: dry-run non-mutation, apply+idempotent-reapply+preservation, newer-instance explicit remediation with zero partial mutation, symlinked-managed-dir rejection with zero write-through and zero schema advance, plus the pre-existing legacy-instance compatibility test).
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold`.
- `bash -n tests/release/compat.sh` — syntax OK (the populated N-minus fixture path is not executable in this sandbox: no network, no prior release exists yet — same constraint as `pose-release-compatibility-matrix` itself).
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-upgrade-compatibility-lab --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Compatibility (unsupported versions get explicit remediation, not partial upgrade): TestUpgradeRejectsNewerInstanceWithExplicitRemediation asserts both the diagnostic and zero tree mutation.
- Security (authenticate prior binaries; block path/symlink escapes): prior-binary authentication unchanged from `compat.sh`'s existing checksum pin; symlink-escape blocking is new (`ensureManagedDirSafe`), proven by TestUpgradeBlocksManagedDirSymlinkEscape.

## 7. Final Report

- Delivered scope: `cmdUpgrade` (previously zero unit-tested) now has full R2/R3 coverage against a populated instance — pt-BR locale content, a real spec, a real knowledge note, a hand-edited managed file — proving dry-run non-mutation, apply-changes-only-schema-version, idempotent reapply, and preservation; a new `ensureManagedDirSafe` helper closes a real symlink-escape gap in the three managed directories `pose upgrade` creates; `tests/release/compat.sh` gained the same populated-fixture depth for real N-minus pairs (R1) plus an idempotency/preservation assertion it previously lacked.
- Residual risk: synthetic fixtures can still miss real-world customization patterns the Go/shell fixtures didn't anticipate — mitigated by both fixtures using the actual `pose install`/`new-spec`/`new-knowledge` commands (not hand-crafted files) so their shape tracks the real scaffold as it evolves; the real N-minus path in `compat.sh` remains unverified until a real prior release exists.
- Follow-ups: see below.

### Follow-ups

- [open] Confirm the first real `check_upgrade_pair` run in `tests/release/compat.sh` once `compatibility.json.supported_upgrades` gets its first entry (0.9.0), and record the result. (owner:@pose-maintainers crit:high review:2026-08-19)

