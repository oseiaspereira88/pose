---
slug: pose-engine-reads-raw-history
status: in-progress
created_at: 2026-11-02
supersedes:
depends_on: pose-history-readers-read-raw
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:engine-reads-raw-history
completed_at:
---

# Spec: The engine reads raw history everywhere

## 1. Intent

### Goal

No reading of a repository by the engine can be redirected by a replace ref or a graft.

### Business value

`pose release check --strict` accepted a tag with no release snapshot in its tree once `git replace` pointed its commit at a valid release: the tag and its object did not move, and the check returned `valid:true` with no gaps. `pose-history-readers-read-raw` gave the engine one raw reader, and the release helpers and about forty other calls never went through it. The review of milestone `manifest-tag-integrity` found this with a falsifier, after the review of the spec had not.

### Constraints

- Behavior on a repository with no replace ref, graft or redirecting environment does not change.
- The rule is enforced by a test, since a rule by convention is what failed here.

### Non-goals

- Authenticating a tag: a signature is another concern.
- `.git/shallow`, handled by `pose-review-reads-raw-history-and-roots-diff`.

## 2. Requirements

### Functional

- R1: `pose release check` and `ReadReleaseSnapshot` shall read a tag's own tree when a replace ref redirects its commit.
- R2: Every Git subprocess of the engine shall go through the raw reader, except those a named exception lists with its reason.
- R3: A new direct Git subprocess in engine source shall fail a test.

### Non-functional

- One more environment variable per subprocess, already paid by the raw reader.

### Security

- A replace ref is a local, unversioned object: this closes a way for a checkout to change what a gate judges without moving a ref.

### Compatibility

- An inherited `GIT_DIR` or `GIT_WORK_TREE` no longer redirects these calls. `-C` decides, as it does for the raw reader.

## 3. Technical Plan

### Affected areas

Every non-test file that started `git` directly.

### Artifacts

- created: .pose/specs/2026-11-02-pose-engine-reads-raw-history.md
- created: .pose/starts/pose-engine-reads-raw-history.json
- created: .pose/changelogs/unreleased/pose-engine-reads-raw-history.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/cli/assess.go
- modified: pose-mcp/internal/cli/capability_triggers.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/historycheck.go
- modified: pose-mcp/internal/cli/identity.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/report.go
- modified: pose-mcp/internal/cli/setup.go
- modified: pose-mcp/internal/cli/state.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_scope.go
- modified: pose-mcp/internal/mcpserver/validate_orchestration.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/assessment_freshness.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/pose/docs.go
- modified: pose-mcp/internal/pose/docs_sync.go
- modified: pose-mcp/internal/pose/release_manifest_integrity.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/spec_transfer.go
- modified: pose-mcp/internal/pose/state.go
- modified: pose-mcp/internal/pose/test_catalog.go
- modified: pose-mcp/internal/pose/spec_trailers.go
- created: pose-mcp/internal/pose/engine_reads_raw_history_test.go
- created: pose-mcp/internal/cli/engine_reads_raw_history_test.go

### Delivery targets

- capability:engine-reads-raw-history module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None but the environment noted under Compatibility.

## 4. Tasks

- [ ] Route every Git subprocess through the raw reader
- [ ] Forbid a direct one by test
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R2, R3
- Minimal option: route only the release helpers
- Selected option: route every call and forbid new ones
- Rationale: `pose-history-readers-read-raw` fixed the readers it listed, and this defect lived in the ones it did not; each exception would need the judgment that was missed here
- Consequences: a large mechanical diff, with exceptions named in one test
- Falsifier: a read that needs replace refs honored

## 6. Validation

### Strategy

The falsifier of the milestone review as an integration test, domain and CLI, seen failing first, and a scan of the sources.

### Requirement trace

- R1 [planned] capability:engine-reads-raw-history evidence:integration check:engine-reads-raw-history-integration test:TestAReplacedTagTreeIsRefused test:TestReleaseCheckRefusesAReplacedTagTree
- R2 [planned] check:engine-reads-raw-history-integration test:TestEngineStartsGitOnlyThroughTheRawReader
- R3 [planned] check:engine-reads-raw-history-integration test:TestEngineStartsGitOnlyThroughTheRawReader

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
