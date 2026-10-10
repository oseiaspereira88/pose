---
slug: pose-v7-4-1-version-alignment
status: done
created_at: 2026-10-10
completed_at: 2026-10-10
priority: 0
components: pose-mcp
task_type: feature
changelog: none
delivers: governance:pose-v7-4-1-version-alignment
---

# Spec: POSE 7.4.1 version alignment

## 1. Intent

### Goal

Align authoritative version metadata on 7.4.1 and authenticate the 7.4.0 upgrade source before preparing the next release snapshot.

### Business value

`pose-review-subject-classifies-engine-records` lets a scope that carries the signed legacy ledger `pose adopt` writes be sealed; Harne8 needs it to close `harne8-adopt-pose-v7-2-0`. One unreleased fragment, `fixed` and not breaking, makes it a patch release.

### Constraints

Prior assets, tags and upgrade pins are preserved; this scope does not publish a release.

## 2. Requirements

### Functional

- R1: CLI version, compatibility matrix, MCP registry manifest, README install snippets in both locales (POSIX and PowerShell) and the CI guide pin shall agree on 7.4.1.
- R2: The matrix shall retain every existing supported upgrade and add 7.4.0, authenticated by its published checksums.txt digest and verified release evidence.
- R3: The compatibility gate shall pass every populated-instance upgrade, the contract checks and the installer E2E using the candidate tree.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-10-pose-v7-4-1-version-alignment.md
- created: .pose/starts/pose-v7-4-1-version-alignment.json
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v7-4-1-version-alignment module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

## 4. Tasks

- [x] Pin the 7.4.0 upgrade by its verified checksums.txt digest
- [x] Align public metadata on 7.4.1
- [x] Pass the compatibility gate and the canonical validation

## 5. Decisions

Patch version 7.4.1: the only unreleased fragment fixes behaviour and is not breaking.

## 6. Validation

### Strategy

`bash tests/release/compat.sh v7.4.1`, `pose public-claims --strict`, `pose validate --tolerant --json-out .pose/results/delivery-validation.json`.

### Execution log

2026-10-10: the 7.4.0 pin is `bce6822db971275e7c7d700fa52edecf3905d09278e2d820e5bae24134f7449c`, the `checksums.txt` digest in `.pose/releases/v7.4.0/verified-evidence.json`. `public-claims --strict` passed. The compatibility gate passed every contract check, the installer E2E and all 17 authenticated upgrades (0.18.2 through 7.4.0 → 7.4.1): COMPATIBLE.

### Requirement trace

- R1 [satisfied] governance:pose-v7-4-1-version-alignment evidence:unit check:public-claims
- R2 [satisfied] governance:pose-v7-4-1-version-alignment report:.pose/releases/v7.4.0/verified-evidence.json
- R3 [satisfied] governance:pose-v7-4-1-version-alignment evidence:manual <tests/release/compat.sh v7.4.1: COMPATIBLE, 17 of 17 upgrades>

## 7. Final Report

### Delivered scope

Version metadata agrees on 7.4.1, and 7.4.0 is a supported, authenticated upgrade source.

### Residual risks

- None beyond the release's own verification.

### Follow-ups
