---
slug: pose-v7-1-0-version-alignment
status: done
created_at: 2026-10-09
completed_at: 2026-10-09
priority: 0
components: pose-mcp
task_type: feature
changelog: none
delivers: governance:pose-v7-1-0-version-alignment
---

# Spec: POSE 7.1.0 version alignment

## 1. Intent

Align authoritative version metadata on 7.1.0 and authenticate the 7.0.0 upgrade source before preparing the next immutable release snapshot.

Origin: the maintainer's plan of 2026-10-09 — close the pose-dist backlog, release, then install and adopt in the projects. Nine unreleased fragments, none breaking, make this a minor release. Prior assets, tags and upgrade pins are preserved; this scope does not publish a release.

## 2. Requirements

- R1: CLI version, compatibility matrix, MCP registry manifest, README install snippets in both locales and the CI guide pin shall agree on 7.1.0.
- R2: The matrix shall retain every existing supported upgrade and add 7.0.0, authenticated by the published checksums.txt digest and verified release evidence.
- R3: The compatibility gate shall pass every populated-instance upgrade, the contract checks and the installer E2E using the candidate tree.

## 3. Technical Plan

Update the six metadata and documentation files and the matrix. The 7.0.0 publication and verification were recorded first (`250ea5a9`), so the pin rests on verified evidence.

### Artifacts

- created: .pose/specs/2026-10-09-pose-v7-1-0-version-alignment.md
- modified: compatibility.json
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/ci.md

### Delivery targets

- governance:pose-v7-1-0-version-alignment module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

## 4. Tasks

- [x] Record the v7.0.0 publication and verification evidence
- [x] Align public metadata and add the authenticated 7.0.0 upgrade pin
- [x] Pass the compatibility gate and the canonical validation

## 5. Decisions

Minor version 7.1.0: the unreleased fragments add and change behaviour, and none is marked breaking. Tag creation requires a frozen candidate and the owner's explicit confirmation, following the release precedent.

## 6. Validation

`bash tests/release/compat.sh v7.1.0`, `pose public-claims --strict`, `pose validate --tolerant --json-out .pose/results/delivery-validation.json`.

### Execution log

2026-10-09: v7.0.0 checksums.txt downloaded and hashed locally: `aeece2ca1e46f9341a553b27316d742940ec6618807711cf082ef4aaf96d5712`, equal to the GitHub asset digest and to the recorded verified evidence. The compatibility gate passed every contract check, the installer E2E and all 13 authenticated upgrades (0.18.2 through 7.0.0 → 7.1.0): COMPATIBLE. A first run failed the installer E2E on a full /tmp quota; 200 leftover test directories were removed and the gate rerun.

### Requirement trace

- R1 [satisfied] governance:pose-v7-1-0-version-alignment evidence:unit check:public-claims
- R2 [satisfied] governance:pose-v7-1-0-version-alignment report:.pose/releases/v7.0.0/verified-evidence.json
- R3 [satisfied] governance:pose-v7-1-0-version-alignment evidence:manual <tests/release/compat.sh v7.1.0: COMPATIBLE, 13 of 13 upgrades>

## 7. Final Report

### Delivered scope

Version metadata agrees on 7.1.0, and 7.0.0 is a supported, authenticated upgrade source.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups

- [open] Clean up the temporary directories journey and compatibility tests leave in /tmp when a run is interrupted: 200 were found on 2026-10-09, about 5 GB, and they filled the user quota twice (owner:@pose-maintainers crit:medium review:2026-10-23)
