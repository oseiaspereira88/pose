---
slug: pose-v7-4-0-version-alignment
status: in-progress
created_at: 2026-10-10
completed_at:
priority: 0
components: pose-mcp
task_type: feature
changelog: none
delivers: governance:pose-v7-4-0-version-alignment
---

# Spec: POSE 7.4.0 version alignment

## 1. Intent

### Goal

Align authoritative version metadata on 7.4.0 and authenticate the 7.3.0 upgrade source before preparing the next release snapshot.

### Business value

Six closed specs wait for release: the answer's reason covered by its signature, the delegated-review brief and dispatch, the release guard on claimed fragments, an answerable maintainer in setup and resolve, and the lowest independence level admitting a different actor. Harne8 needs the last one to close its open specs with a cross-vendor reviewer. Two fragments are `added` and none is breaking, so this is a minor release. The copy-based dry-run (`pose-update-dry-run-reports-the-whole-update`) is left out by the maintainer's decision; its fragment was withdrawn.

### Constraints

Prior assets, tags and upgrade pins are preserved; this scope does not publish a release.

## 2. Requirements

### Functional

- R1: CLI version, compatibility matrix, MCP registry manifest, README install snippets in both locales (POSIX and PowerShell) and the CI guide pin shall agree on 7.4.0.
- R2: The matrix shall retain every existing supported upgrade and add 7.3.0, authenticated by its published checksums.txt digest and verified release evidence.
- R3: The compatibility gate shall pass every populated-instance upgrade, the contract checks and the installer E2E using the candidate tree.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-10-pose-v7-4-0-version-alignment.md
- created: .pose/starts/pose-v7-4-0-version-alignment.json
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v7-4-0-version-alignment module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

## 4. Tasks

- [x] Pin the 7.3.0 upgrade by its verified checksums.txt digest
- [x] Align public metadata on 7.4.0
- [x] Pass the compatibility gate and the canonical validation

## 5. Decisions

Minor version 7.4.0: the unreleased fragments add or fix behaviour and none is breaking.

## 6. Validation

### Strategy

`bash tests/release/compat.sh v7.4.0`, `pose public-claims --strict`, `pose validate --tolerant --json-out .pose/results/delivery-validation.json`.

### Execution log

2026-10-10: the 7.3.0 pin is `8d550cb64b9bd15d99a4edfb588475407cc2875f46b943432a8182b8b2749d68`, the `checksums.txt` digest in `.pose/releases/v7.3.0/verified-evidence.json`. `public-claims --strict` passed. The compatibility gate passed every contract check, the installer E2E and all 16 authenticated upgrades (0.18.2 through 7.3.0 → 7.4.0): COMPATIBLE.

### Requirement trace

- R1 [satisfied] governance:pose-v7-4-0-version-alignment evidence:unit check:public-claims
- R2 [satisfied] governance:pose-v7-4-0-version-alignment report:.pose/releases/v7.3.0/verified-evidence.json
- R3 [satisfied] governance:pose-v7-4-0-version-alignment evidence:manual <tests/release/compat.sh v7.4.0: COMPATIBLE, 16 of 16 upgrades>

## 7. Final Report

### Delivered scope

Version metadata agrees on 7.4.0, and 7.3.0 is a supported, authenticated upgrade source.

### Residual risks

- None beyond the release's own verification.

### Follow-ups
