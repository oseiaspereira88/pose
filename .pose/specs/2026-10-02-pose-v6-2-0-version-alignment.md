---
slug: pose-v6-2-0-version-alignment
status: done
created_at: 2026-10-02
completed_at: 2026-10-02
priority: 0
components: pose-mcp
task_type: feature
changelog: none
delivers: governance:pose-v6-2-0-version-alignment
---

# Spec: Align POSE 6.2.0 candidate metadata

## 1. Intent

Align authoritative version metadata and authenticate the 6.1.0 upgrade source
before preparing the next immutable release snapshot.

Preserve all prior published assets, tags and upgrade pins. This scope does not
publish a release, change public schemas or run the deferred native package round.

## 2. Requirements

- R1: CLI version, compatibility matrix, MCP registry manifest, README install
  snippets in both locales and the CI guide pin shall agree on 6.2.0.
- R2: The matrix shall retain all ten existing supported upgrades and add 6.1.0
  authenticated by the published checksums.txt digest and verified release evidence.
- R3: The compatibility gate shall pass all eleven populated-instance upgrades,
  contract checks and installer E2E using the candidate tree.

## 3. Technical Plan

Update the six existing metadata/documentation files and the compatibility matrix.
Verify the checksum pin before running any downloaded prior binary. Reuse the
existing compatibility harness; no additional runtime or test framework is needed.

### Artifacts

- created: .pose/specs/2026-10-02-pose-v6-2-0-version-alignment.md
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v6-2-0-version-alignment module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

## 4. Tasks

- [x] Align public metadata and add authenticated 6.1.0 upgrade pin.
- [x] Pass strict canonical validation and compatibility gates; artifact/surface gates run before sealing.
- [x] Record separate governed review and close this bounded scope.

## 5. Decisions

Keep the requested minor version 6.2.0. Release machinery remains separate from
version alignment. Tag creation requires a reviewable frozen candidate and the
owner's explicit confirmation, following the established release precedent.

## 6. Validation

Run `pose validate --strict --json-out .pose/results/delivery-validation.json`,
`pose artifact-check --spec pose-v6-2-0-version-alignment --strict`,
`pose surface-check --spec pose-v6-2-0-version-alignment --strict`,
`pose public-claims --strict` and `bash tests/release/compat.sh v6.2.0`.

### Execution log

2026-10-02: canonical strict matrix passed 48/48 at 552542f. Compatibility
harness passed all five contract/installer checks and all eleven authenticated
populated-instance upgrades. Fixture boundary checks passed five negative/positive
cases: empty field, commented empty field, nonempty refs, another slug and a
commented example. Public-claims checked 17 surfaces with zero findings.

2026-10-02: published 6.1.0 checksums.txt SHA256 is
840ce3cfc5ef8a6bb4b3b161165fa919269d55a3ade396754e340b0bcdeb2ad5,
matching .pose/releases/v6.1.0/verified-evidence.json. Component discovery ran
before changes; tech-debt found zero markers.

### Requirement trace

- R1 [satisfied] governance:pose-v6-2-0-version-alignment evidence:unit check:public-claims
- R2 [satisfied] governance:pose-v6-2-0-version-alignment report:compatibility-report.md
- R3 [satisfied] governance:pose-v6-2-0-version-alignment report:compatibility-report.md

## 7. Final Report

Closed on 2026-10-02 with fresh approved bundle `rvb-917d129fb07d50f8`,
explicit attestation and guarded lifecycle transition. All 48 canonical checks
and all eleven supported upgrades passed. No follow-ups introduced.
