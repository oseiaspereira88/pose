---
slug: pose-engine-package-stays-mirrorable
status: done
created_at: 2026-10-16
completed_at: 2026-10-16
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
- [x] Prove a Harne8 mirror sync builds and passes; strict validation, review and closeout.

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

Kept production files of `internal/pose` free of sibling internal imports, guarded that boundary, and preserved `version.Version` in binary assessment bindings through CLI and MCP startup initialization. D1 is proportionate: it preserves the consumer's byte-for-byte mirror contract without import rewriting or an additional mirrored package. No CLI, MCP or binding schema changed.

Independent reviewer `agent:independent-gpt-6.1-sol-review` approved bundle `rvb-e9c5e6e54616df45` in attestation `rva-bdd5a9d2b672d719`. This was a maintainer-authorized Codex/OpenAI execution distinct from Claude/Anthropic implementation session `296d1fcb-e989-40ae-9362-cf3514db8b26` (`agent:claude-opus-5-5`). The `agent:independent-` prefix is an engine-required declaration until `pose-delegated-review-run-provenance` replaces it; identity is declared and these attestations are unsigned.

On 2026-10-16, the reviewer independently demonstrated the guard failing with only `assessment_freshness.go` reverted to `origin/main`, restored the exact implementation bytes, and observed the guard passing. `go vet ./...` and `go test ./... -count=1` passed in `pose-mcp`. Auditing `cmd/pose`, CLI hooks, bootstrap and MCP startup found no binary binding path outside version initialization.

The prepared Harne8 `cirepro` mirror at engine commit `4dcf70b6` passed Harness build, vet and tests. Its parity self-test passed all six drift cases, overlaid 98 upstream test files successfully and confirmed mirror parity. That clone was not edited, committed or pushed by this review.

Native full validation passed 133/133 checks, and explicit module validation passed 130/130, without failures, errors or skips. Strict `surface-check` reported one composed capability and zero findings. `review verify` reported fresh and approved; guarded `close`, strict spec lint and strict structure check passed. Evidence is recorded in `.pose/results/delivery-validation.json`, `.pose/results/pose-engine-package-stays-mirrorable-module-validation.json` and `.pose/results/pose-engine-package-stays-mirrorable-independent-review.json`, plus the sealed bundle and attestation. Discovery with `--update-state` was executed; its generated output was excluded as directed by the maintainer.

### Residual risks

External embedders must set `EngineVersion` to distinguish engine revisions. An unset embedder records `unknown`; two such bindings can match despite an engine revision change. The current binary initializes its version before runtime binding calls, so this limitation does not weaken R2.

The review identity and vendor separation are declared, without signed provenance. The repository-wide strict check retains 29 warnings, including legacy changelog and capability-assessment age diagnostics. No policy, capability assessment, adoption setting, production code or test was changed by the reviewer. Dedicated vulnerability and secret-scanner binaries were unavailable; the changed diff was inspected and a bounded secret-pattern scan found no matches.

### Follow-ups

This closeout creates no follow-up work. The Harne8 archive-script adjustment remains outside this delivery and is reserved for its own spec, as directed by the maintainer.
