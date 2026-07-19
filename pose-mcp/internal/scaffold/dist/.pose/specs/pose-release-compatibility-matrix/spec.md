---
slug: pose-release-compatibility-matrix
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-version-contract, pose-public-install-contract
priority: 4
---

# Spec: Release compatibility matrix

## 1. Intent

### Goal
prove engine, instance schema, scaffold, MCP metadata, docs and upgrades for each release candidate.
### Business value
Prevents a nominal release from distributing mutually incompatible parts.
### Constraints
- Separate SemVer compatibility from repository schema compatibility and test both.
### Non-goals
- Promise downgrade support.

## 2. Requirements

### Functional
- R1: A machine-readable matrix shall declare supported engine, schema and upgrade pairs.
- R2: Release CI shall test fresh install and every supported prior-version upgrade.
- R3: Documentation commands and MCP metadata shall be validated against the same candidate artifact.

### Non-functional
- Run fixtures with pinned, authenticated prior artifacts.

### Security
- Verify prior artifacts before executing compatibility tests.

### Compatibility
- Unsupported pairs shall fail with actionable diagnostics.

## 3. Technical Plan

### Affected areas
- Release workflow, migrations, scaffold fixtures, docs checks and MCP metadata.

### API/contract changes
- Publish a compatibility artifact and support policy.

### Data/storage changes
- Version the matrix and retain candidate results.

### Technical risks
- An unbounded version matrix can make release latency unacceptable.

### Primary references
- [Semantic Versioning](https://semver.org/)
- [The Update Framework](https://theupdateframework.io/)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [Semantic Versioning](https://semver.org/): no compatibility artifact existed; historical tags `v0.1.x` predate the version/catalog/install contracts and cannot be honest upgrade sources; SemVer and `.pose/schema-version` confirmed as independent axes.

### Implementation
- [x] Define the support window and compatibility schema: `compatibility.json` (matrix_version, engine_version, schema_version, support_policy, supported_upgrades with SHA-256 pins); window starts at 0.9.0 (ADR `2026-07-19-bounded-release-compatibility-policy`). ([reference](https://semver.org/))
- [x] Build fresh-install and N-minus upgrade fixtures from verified releases: `tests/release/compat.sh` exercises every `supported_upgrades` entry (download, pinned-checksum verification before execution, prior install → candidate `pose upgrade` → `pose check --strict`); Go fixtures cover the schema axis (`TestCompatibilityUpgradeFromLegacyInstance`, `TestCompatibilityDowngradeRejected` with actionable diagnostics). ([reference](https://theupdateframework.io/))
- [x] Gate release notes and docs on the candidate compatibility report: `release.yml` runs the compatibility gate before GoReleaser publishes; a tag that diverges from the matrix aborts the release; `compatibility.json` and `compatibility-report.md` ship as release assets; report retained 400 days as CI artifact. ([reference](https://semver.org/))

### Validation
- [x] Run `go test ./pose-mcp/internal/cli/... -run 'Upgrade|Install|Schema'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://semver.org/))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://theupdateframework.io/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-bounded-release-compatibility-policy.md` (Accepted): versioned matrix with a bounded window gated in release CI, over ad-hoc release-note claims and over unbounded historical support; pre-0.9.0 tags excluded from the window; prior artifacts must be checksum-authenticated before execution; downgrade remains unsupported by contract.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Upgrade|Install|Schema'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-release-compatibility-matrix --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli -run 'Compat|Upgrade|Install|Schema' -count=1` — SUCCESS.
- `bash tests/release/compat.sh` — Result: COMPATIBLE (candidate stamped 0.9.0; version, catalog, matrix, scaffold and installer gates all PASS; empty upgrade window reported honestly).
- `bash tests/release/compat.sh v0.8.0` — FAIL as designed (tag/matrix divergence aborts with both values named).
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-release-compatibility-matrix --ready-check` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Versioned `compatibility.json` (engine/schema/upgrade pairs, support policy,
checksum-pinned prior artifacts) published with each release; release
compatibility gate in CI generating `compatibility-report.md` from the same
candidate tree (version, catalog, install, scaffold, installer E2E and
upgrade fixtures); schema-axis fixtures including negative downgrade path;
tag/matrix consistency enforcement; support-window ADR; README compatibility
policy section.

### Residual risks

- The upgrade window is empty until the next release exists; the first real
  prior-version upgrade execution happens in the 0.9.x → next release gate.
- The matrix and report are not yet cryptographically signed.

### Follow-ups

- [open] After the first post-0.9.0 release, add 0.9.0 to supported_upgrades with its checksums.txt SHA-256 pin and verify the gate exercises it.
- [covered: pose-release-signing] Sign the compatibility artifacts with the release.
- [covered: pose-upgrade-compatibility-lab] Broaden upgrade coverage across OS/arch and released-version pairs.
