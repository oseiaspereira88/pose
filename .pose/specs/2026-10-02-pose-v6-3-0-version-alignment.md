---
slug: pose-v6-3-0-version-alignment
status: done
created_at: 2026-10-02
completed_at: 2026-10-02
priority: 0
components: pose-mcp
task_type: feature
changelog: none
delivers: governance:pose-v6-3-0-version-alignment
---

# Spec: Align POSE 6.3.0 candidate metadata

## 1. Intent

Align authoritative version metadata and authenticate the 6.2.0 upgrade source
before preparing the next immutable release snapshot.

Preserve all prior published assets, tags and upgrade pins. This scope does not
publish a release, change public schemas or run the deferred native package round.

## 2. Requirements

- R1: CLI version, compatibility matrix, MCP registry manifest, README install
  snippets in both locales and the CI guide pin shall agree on 6.3.0.
- R2: The matrix shall retain all eleven existing supported upgrades and add 6.2.0
  authenticated by the published checksums.txt digest and verified release evidence.
- R3: The compatibility gate shall pass all twelve populated-instance upgrades,
  contract checks and installer E2E using the candidate tree.

## 3. Technical Plan

Update the six existing metadata and documentation files, adding the 6.2.0 pin to the
compatibility matrix. Verify the checksum pin against the published release and the
retained evidence before running any downloaded prior binary. Reuse the existing
compatibility harness; no additional runtime or test framework is needed.

### Artifacts

- created: .pose/specs/2026-10-02-pose-v6-3-0-version-alignment.md
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v6-3-0-version-alignment module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

## 4. Tasks

- [x] Align public metadata and add the authenticated 6.2.0 upgrade pin.
- [x] Pass strict canonical validation and compatibility gates; artifact and surface gates run before sealing.
- [ ] Record a separate governed review and close this bounded scope.

## 5. Decisions

The version is minor, 6.3.0, as approved by the owner on 2026-10-02: the governed
change set adds a release version source and clean-activation evidence, and changes
the index schema, the CI release gates and the renderers, with no breaking fragment.
Release machinery stays separate from version alignment. Tag creation requires a
reviewable frozen candidate and the owner's explicit confirmation.

## 6. Validation

Run `pose validate --strict --json-out .pose/results/delivery-validation.json`,
`pose artifact-check --spec pose-v6-3-0-version-alignment --strict`,
`pose surface-check --spec pose-v6-3-0-version-alignment --strict`,
`pose public-claims --strict` and `bash tests/release/compat.sh v6.3.0`.

### Execution log

2026-10-02: the published 6.2.0 checksums.txt was downloaded with `gh release
download` and hashed: SHA256 2e937ae3a7ac12fca99148bd71179e4adc052b0005e7bea7d430deb061fe5b29,
equal to the digest in `.pose/releases/v6.2.0/publication-evidence.json` and
`verified-evidence.json`.

2026-10-02: `bash tests/release/compat.sh v6.3.0` at commit 5fbcb21 passed all five
contract and installer gates and all twelve authenticated populated-instance
upgrades, 6.2.0 through 0.18.2, with the candidate built as the release pipeline
stamps it; the resulting report is the untracked `compatibility-report.md`.
`pose public-claims --strict` checked 17 surfaces with zero errors and reported
the released version as 6.3.0.

### Requirement trace

- R1 [satisfied] governance:pose-v6-3-0-version-alignment evidence:unit check:public-claims
- R2 [satisfied] governance:pose-v6-3-0-version-alignment report:compatibility-report.md
- R3 [satisfied] governance:pose-v6-3-0-version-alignment report:compatibility-report.md
### Known gaps

The native macOS and Windows package round remains deferred.

## 7. Final Report

### Delivered scope
Version, registry manifest, README snippets in both locales and the CI guide agree on 6.3.0, and the compatibility matrix adds 6.2.0 pinned to the published checksums.txt digest. The compatibility gate passed five contract gates and twelve populated-instance upgrades.
### Follow-ups
No follow-ups introduced. The native macOS and Windows package round remains deferred.
