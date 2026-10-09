---
slug: pose-v7-2-0-version-alignment
status: in-progress
created_at: 2026-10-09
completed_at:
priority: 0
components: pose-mcp
task_type: feature
changelog: none
delivers: governance:pose-v7-2-0-version-alignment
---

# Spec: POSE 7.2.0 version alignment

## 1. Intent

### Goal

Align authoritative version metadata on 7.2.0 and authenticate the 7.1.0 upgrade source before preparing the next release snapshot.

### Business value

The native attestation issuer (`pose-native-attestation-issuer`, closed after five independent reviews) lets a project with POSE alone adopt signed attestations and verified identity. Harne8 deferred both until such a release exists; this release delivers it. One unreleased fragment, `added` and not breaking, makes it a minor release.

### Constraints

Prior assets, tags and upgrade pins are preserved; this scope does not publish a release.

## 2. Requirements

### Functional

- R1: CLI version, compatibility matrix, MCP registry manifest, README install snippets in both locales (POSIX and PowerShell) and the CI guide pin shall agree on 7.2.0.
- R2: The matrix shall retain every existing supported upgrade and add 7.1.0, authenticated by its published checksums.txt digest and verified release evidence.
- R3: The compatibility gate shall pass every populated-instance upgrade, the contract checks and the installer E2E using the candidate tree.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-v7-2-0-version-alignment.md
- created: .pose/starts/pose-v7-2-0-version-alignment.json
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v7-2-0-version-alignment module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

## 4. Tasks

- [x] Pin the 7.1.0 upgrade by its verified checksums.txt digest
- [x] Align public metadata on 7.2.0
- [x] Pass the compatibility gate and the canonical validation

## 5. Decisions

Minor version 7.2.0: the only unreleased fragment adds behaviour and is not breaking.

## 6. Validation

### Strategy

`bash tests/release/compat.sh v7.2.0`, `pose public-claims --strict`, `pose validate --tolerant --json-out .pose/results/delivery-validation.json`.

### Execution log

2026-10-09: the 7.1.0 pin is `451e962361f2dce23fb1ce19b42531eb8490e3ae39533e35666a9799a4e7c182`, the SHA-256 computed locally from the downloaded checksums.txt when 7.1.0 was installed, equal to the `checksums.txt` digest in `.pose/releases/v7.1.0/verified-evidence.json`. `public-claims --strict` passed. The compatibility gate passed every contract check, the installer E2E and all 14 authenticated upgrades (0.18.2 through 7.1.0 → 7.2.0): COMPATIBLE.

### Requirement trace

- R1 [satisfied] governance:pose-v7-2-0-version-alignment evidence:unit check:public-claims
- R2 [satisfied] governance:pose-v7-2-0-version-alignment report:.pose/releases/v7.1.0/verified-evidence.json
- R3 [satisfied] governance:pose-v7-2-0-version-alignment evidence:manual <tests/release/compat.sh v7.2.0: COMPATIBLE, 14 of 14 upgrades>

## 7. Final Report

### Delivered scope

Version metadata agrees on 7.2.0, and 7.1.0 is a supported, authenticated upgrade source.

### Residual risks

- None beyond the release's own verification.

### Follow-ups
