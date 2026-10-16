---
slug: pose-engine-package-stays-mirrorable
status: in-progress
created_at: 2026-10-16
completed_at:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
changelog: none
delivers: capability:engine-package-stays-mirrorable
---

# Spec: Keep internal/pose free of sibling internal imports

## 1. Intent

### Goal

Let the engine package `pose-mcp/internal/pose` compile on its own again, so a consumer that vendors its production files (the Harne8 Harness mirror) can follow the engine.

### Business value

Harne8's Harness vendors `internal/pose` as generated copies and its CI gate `pose-mirror-parity` requires the copy to match the pinned engine byte for byte. Since `pose-adaptive-assessment-freshness` (2026-10-04), `assessment_freshness.go` imports `github.com/harne8/pose-mcp/internal/version`; Go forbids importing another module's internal package, so a synced mirror stops compiling and Harne8 has been unable to sync since 6.1.0, keeping its `main` red on that gate.

### Constraints

The binary must keep recording its own version in assessment bindings, so stored assessments keep their freshness verdicts. No change to bindings, schemas or CLI/MCP output.

## 2. Requirements

- R1: Production files of `internal/pose` shall import no other `github.com/harne8/pose-mcp/internal/...` package, and a test shall fail when one does.
- R2: The `pose` binary (CLI and MCP server) shall record `internal/version.Version` as the engine version of assessment bindings, as before.
- R3: A copy of the production files of `internal/pose`, as Harne8's `scripts/sync-pose-sources.sh` produces it, shall build and pass the Harness suite.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-16-pose-engine-package-stays-mirrorable.md
- created: .pose/starts/pose-engine-package-stays-mirrorable.json
- modified: pose-mcp/internal/pose/assessment_freshness.go
- created: pose-mcp/internal/pose/package_boundary_test.go
- modified: pose-mcp/internal/cli/cli.go
- created: pose-mcp/internal/cli/engine_version_binding_test.go
- modified: pose-mcp/internal/mcpserver/server.go

### Delivery targets

- capability:engine-package-stays-mirrorable module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/assessment_freshness.go

### Approach

`internal/pose` exposes `var EngineVersion = "unknown"` and uses it in `CurrentAssessmentBinding`. Packages `cli` and `mcpserver`, which already import `internal/version`, set it in `init`, so every entrypoint of the binary records `version.Version` exactly as before. An embedder that leaves it unset records `unknown`, and its freshness check reads stored bindings as another engine, which is the conservative direction. The only sibling import in the package was this one.

## 4. Tasks

- [x] Guard test first: it fails on `assessment_freshness.go` importing `internal/version`.
- [x] Move the version behind `EngineVersion`, set by cli and mcpserver.
- [ ] Prove a Harne8 mirror sync builds and passes; strict validation, review and closeout.

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: Harne8 mirrors `internal/version` too and rewrites the import, changing its byte-parity contract
- Selected option: the engine package takes the version as a variable set by the binary
- Rationale: one consumer-neutral change in the engine keeps the consumer's byte-for-byte parity contract intact
- Consequences: every new entrypoint embedding the package sets `EngineVersion` or records `unknown`
- Falsifier: a production entrypoint of the binary that records `unknown` in a binding

## 6. Validation

### Strategy

Prove the guard fails on the defect, then the full module suite, then a real Harne8 mirror sync from this commit.

### Deterministic checks

`go test ./internal/pose -run TestProductionFilesImportNoSiblingInternalPackage` failed before the change (`assessment_freshness.go imports github.com/harne8/pose-mcp/internal/version`) and passes after. `go vet ./...` and `go test ./... -count=1` pass in `pose-mcp`.

### Requirement trace

- R1 [satisfied] test:TestProductionFilesImportNoSiblingInternalPackage
- R2 [satisfied] test:TestBinaryRecordsItsVersionInAssessmentBindings
- R3 [satisfied] check:test — Harne8 `scripts/sync-pose-sources.sh` from this commit, then `go build`, `go vet` and `go test ./...` in `harness`

## 7. Final Report

### Delivered scope

Pending review.

### Residual risks

Embedders outside the binary must set `EngineVersion` to get matching bindings.

### Follow-ups
