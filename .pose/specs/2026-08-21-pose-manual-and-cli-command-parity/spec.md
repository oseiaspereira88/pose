---
slug: pose-manual-and-cli-command-parity
status: done
created_at: 2026-08-21
completed_at: 2026-08-21
supersedes:
depends_on: pose-command-reference-parity
priority: 25
components: docs, scaffold, locales, cli, version
delivers: surface:cli-manual-parity
---

# Spec: Manual Overview and Command Reference Bidirectional Parity

## 1. Intent

### Goal
Ensure 100% bidirectional parity between Section 6 CLI overview examples and the full Command Reference across English and Brazilian Portuguese (`pt-BR`) documentation manuals, and keep embedded distribution scaffolds perfectly synchronized.

### Business value
Prevents developer and AI agent confusion caused by missing or unlisted CLI commands, subcommands, and flags in documentation.

### Constraints
- English and Portuguese manuals must maintain identical command coverage.
- Embedded scaffold copies must match root documentation.

### Non-goals
- Modifying underlying CLI execution semantics.

---

## 2. Requirements

### Functional
- R1: Section 6 CLI synopsis shall list every top-level tool with short descriptions.
- R2: The Command Reference section shall provide detailed documentation for all tools.
- R3: Brazilian Portuguese (`locales/pt-BR/POSE.md`) shall mirror English (`POSE.md`).
- R4: Embedded scaffold distribution files shall be synced automatically.
- R5: Release lifecycle tooling shall recognize hybrid and date-prefixed spec structures during release cuts.
- R6: Version metadata, compatibility matrix, and installation quickstarts shall align to the active release base.

### Non-functional
- Zero drift between root docs and embedded scaffolds.

### Security
- Documentation shall not leak private environment or authorization secrets.

### Compatibility
- Preserve all existing command links and headings.

---

## 3. Technical Plan

### Affected areas
- `POSE.md`
- `locales/pt-BR/POSE.md`
- `pose-mcp/internal/scaffold/dist/POSE.md`
- `pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md`
- `pose-mcp/internal/cli/release_lifecycle.go`
- `pose-mcp/internal/version/version.go`
- `pose-mcp/server.json`
- `compatibility.json`
- `README.md`
- `docs-site/docs/ci.md`

### Artifacts
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/version/version.go
- modified: pose-mcp/server.json
- modified: compatibility.json
- modified: README.md
- modified: docs-site/docs/ci.md

### Delivery targets
- surface:cli-manual-parity module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Artifacts

### Code
- `POSE.md` (action: modify)
- `locales/pt-BR/POSE.md` (action: modify)
- `pose-mcp/internal/scaffold/dist/POSE.md` (action: modify)
- `pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md` (action: modify)
- `pose-mcp/internal/cli/release_lifecycle.go` (action: modify)
- `pose-mcp/internal/version/version.go` (action: modify)
- `pose-mcp/server.json` (action: modify)
- `compatibility.json` (action: modify)
- `README.md` (action: modify)
- `docs-site/docs/ci.md` (action: modify)

---

## 5. Verification Plan

### Automated
- `pose check --strict`
- `pose validate --strict`
- `go test ./internal/scaffold ./internal/cli ./internal/version`

---

## 6. Delivery Evidence

### Artifact claims
- path: POSE.md action: modified sha256:fb193f6d05ebd48c81cdb83215c4ac3544dc5d4e0683b497d7f07623771dfa69
- path: locales/pt-BR/POSE.md action: modified sha256:bd127f803ac6613e367d3cbbe8c45a994e16e431b4a6417e92fc46a231efa378
- path: pose-mcp/internal/scaffold/dist/POSE.md action: modified sha256:fb193f6d05ebd48c81cdb83215c4ac3544dc5d4e0683b497d7f07623771dfa69
- path: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md action: modified sha256:bd127f803ac6613e367d3cbbe8c45a994e16e431b4a6417e92fc46a231efa378
- path: pose-mcp/internal/cli/release_lifecycle.go action: modified sha256:27e0594195b901aa0ffbb54252a3fae2a8cfe0f387c196daa2772840cb258a0a
- path: pose-mcp/internal/version/version.go action: modified sha256:b84e1fcbdf73c517191a1277db88e972d82b2c0d8a74b513c9c3201faa2a9441
- path: pose-mcp/server.json action: modified sha256:95754d5c95101de423eac67a004b4fed12340e5d030a94af6654df3b157eff35
- path: compatibility.json action: modified sha256:7b9bb8a8e8c31848a3a06218fc5801838e9412c315ee3e39cf103d130e116274
- path: README.md action: modified sha256:5b0d0e484b8ca4dcf6883c3de3703a95b201243dee352946dc04900220361123
- path: docs-site/docs/ci.md action: modified sha256:b15388c0c2140a2d2ba6deb22b821f2e208512c257efc10dca6aa2f1253850dd

### Requirement trace
- R1 [satisfied] surface:cli-manual-parity check:unit test:TestEmbeddedDistMatchesPoseDist evidence:integration
- R2 [satisfied] surface:cli-manual-parity check:unit test:TestEmbeddedDistMatchesPoseDist evidence:integration
- R3 [satisfied] surface:cli-manual-parity check:unit test:TestEmbeddedDistMatchesPoseDist evidence:integration
- R4 [satisfied] surface:cli-manual-parity check:unit test:TestEmbeddedDistMatchesPoseDist evidence:integration
- R5 [satisfied] surface:cli-manual-parity check:unit test:TestReleaseInputs evidence:integration
- R6 [satisfied] surface:cli-manual-parity check:unit test:TestPublicInstallContract evidence:integration

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Reconciled Section 6 CLI synopsis and Command Reference in English and pt-BR.
- Ensured embedded scaffold distribution files match repository manuals.
- Hardened release lifecycle spec resolution for date-prefixed structures.
- Synchronized version metadata, compatibility matrix, and installation docs for v1.6.0.

### Files and modules changed
- `POSE.md`
- `locales/pt-BR/POSE.md`
- `pose-mcp/internal/scaffold/dist/POSE.md`
- `pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md`
- `pose-mcp/internal/cli/release_lifecycle.go`
- `pose-mcp/internal/version/version.go`
- `pose-mcp/server.json`
- `compatibility.json`
- `README.md`
- `docs-site/docs/ci.md`

### Follow-ups
- [done] Reconciled Section 6 CLI overview and Command Reference with full bilingual parity.
