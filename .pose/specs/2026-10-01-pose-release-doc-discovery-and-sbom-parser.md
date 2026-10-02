---
slug: pose-release-doc-discovery-and-sbom-parser
status: done
created_at: 2026-10-01
completed_at: 2026-10-01
components: pose-mcp
task_type: bugfix
priority: 2
---

# Spec: Release documentation discovery and shared SBOM dependency parsing

## 1. Intent

Eliminate manual documentation source lists and duplicate dependency parsing in
Linux release verification. Native package checks remain deferred.

## 2. Requirements

- R1: Download URL parity shall inspect README and all Markdown documents under
  docs-site/docs, including nested paths and names containing spaces.
- R2: An asset promised only by a newly added document shall fail parity when
  absent and pass when the published asset inventory contains it.
- R3: SBOM verification and its positive control shall consume one direct Go
  dependency parser; indirect dependencies shall remain excluded.

- R4: The local verification script shall cover the current CI command gates,
  declaring expensive or online omissions explicitly in fast mode.

## 3. Technical Plan

### Artifacts

- modified: scripts/verify.sh
- modified: tests/release/docs-asset-parity.sh
- modified: tests/release/verify-sbom.sh
- modified: tests/release/verify-negative.sh
- created: pose-mcp/internal/version/release_harness_discovery_test.go
- created: .pose/specs/2026-10-01-pose-release-doc-discovery-and-sbom-parser.md
- created: .pose/changelogs/unreleased/pose-release-doc-discovery-and-sbom-parser.md

## 4. Tasks

- [x] Replace manual docs enumeration with null-delimited recursive discovery.
- [x] Expose the verifier dependency list to the fixture generator.
- [x] Execute positive/negative synthetic parity and SBOM harness cases.

## 5. Decisions

Keep shell as the release harness and preserve the verifier's existing parsing
semantics; centralize the parser instead of introducing a new implementation.

## 6. Validation

Before implementation: exercise the actual parity script with stubbed read-only
Git/GitHub inventory providers and a nested document containing a missing asset.
Run the actual SBOM negative harness and shellcheck if installed. No remote
release, signing identity or platform runner is required.

### Execution log

- Nested document parity positive/negative and local CI parity tests passed.
- Actual SBOM negative harness passed all seven rejection scenarios and its
  positive control. Shell syntax validation passed. Local shellcheck is absent;
  remote CI retains its required shellcheck gate.

### Requirement trace

- R1 [satisfied] test:TestReleaseDocsParityDiscoversNestedDocument
- R2 [satisfied] test:TestReleaseDocsParityDiscoversNestedDocument
- R3 [satisfied] report:tests/release/verify-negative.sh
- R4 [satisfied] test:TestLocalVerifyCoversCurrentCIGates

## 7. Final Report

Implementation validated and closed through the governed review gate on 2026-10-01. Canonical strict validation passed 48/48 checks at f41b5aa; review bundle `rvb-3b6f52e24b22f0d3` was fresh and approved before `pose close`.
