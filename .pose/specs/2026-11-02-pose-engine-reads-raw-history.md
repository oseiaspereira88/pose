---
slug: pose-engine-reads-raw-history
status: done
created_at: 2026-11-02
supersedes:
depends_on: pose-history-readers-read-raw
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:engine-reads-raw-history
completed_at: 2026-11-02
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
- An adversary inside the source: the test guards against a call written by habit, and reflection or a generated program would pass it.
- Executable Git configuration of the repository, such as `core.fsmonitor`, which the raw reader does not disarm.
- Shell scripts the repository ships, and declared external checks: the rule covers the Go sources of the engine.
- `.git/shallow`, handled by `pose-review-reads-raw-history-and-roots-diff`.

## 2. Requirements

### Functional

- R1: `pose release check` and `ReadReleaseSnapshot` shall read a tag's own tree when a replace ref redirects its commit.
- R2: Every Git reading the Go sources of the engine start shall go through the raw reader, except those a named exception lists with its reason; a write keeps the index the operator selected.
- R3: A new process start in engine source that names git, looks it up, runs it through a shell, or starts a program the source does not name without saying what it is, shall fail a test.

### Non-functional

- One more environment variable per subprocess, already paid by the raw reader.

### Security

- A replace ref is a local, unversioned object: this closes a way for a checkout to change what a gate judges without moving a ref.

### Compatibility

- An inherited `GIT_DIR` or `GIT_WORK_TREE` no longer redirects a reading. `-C` decides, as it does for the raw reader. `pose report --git-stage` still stages into `GIT_INDEX_FILE`.

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
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- created: pose-mcp/internal/pose/engine_reads_raw_history_test.go
- modified: pose-mcp/internal/pose/b07_r4b_probe_test.go
- modified: pose-mcp/internal/cli/b07_r4b_probe_test.go
- created: pose-mcp/internal/cli/engine_reads_raw_history_test.go

### Delivery targets

- capability:engine-reads-raw-history module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None but the environment noted under Compatibility.

## 4. Tasks

- [x] Route every Git subprocess through the raw reader
- [x] Forbid a direct one by test
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R2, R3
- Minimal option: route only the release helpers
- Selected option: route every call and forbid new ones
- Rationale: `pose-history-readers-read-raw` fixed the readers it listed, and this defect lived in the ones it did not; each exception would need the judgment that was missed here
- Consequences: a large mechanical diff, with exceptions and dynamic launches named in one test, which reads the syntax tree because a pattern over the text missed two calls
- Falsifier: a read that needs replace refs honored

## 6. Validation

### Strategy

The falsifier of the milestone review as an integration test, domain and CLI, seen failing first, and a scan of the sources.

### Requirement trace

- R1 [satisfied] capability:engine-reads-raw-history evidence:integration check:engine-reads-raw-history-integration test:TestAReplacedTagTreeIsRefused test:TestReleaseCheckRefusesAReplacedTagTree report:.pose/results/delivery-validation.json commit:270b6a82e7e1c8a5084305fc407357d161bfa5f5
- R2 [satisfied] check:engine-reads-raw-history-integration test:TestEngineStartsGitOnlyThroughTheRawReader test:TestReadyTransitionsReadRawHistory test:TestReportStagesIntoTheSelectedIndex report:.pose/results/delivery-validation.json commit:270b6a82e7e1c8a5084305fc407357d161bfa5f5
- R3 [satisfied] check:engine-reads-raw-history-integration test:TestEngineStartsGitOnlyThroughTheRawReader test:TestTheLaunchScanSeesEveryForm report:.pose/results/delivery-validation.json commit:270b6a82e7e1c8a5084305fc407357d161bfa5f5

## 7. Final Report

### Delivered scope

A replace ref can no longer supply the release snapshot of a tag: `pose release check` refuses it in the domain and in the CLI. Every Git reading in the Go sources of the engine goes through the raw reader, and a test that reads the syntax tree of the whole module refuses a new direct start, a starter taken as a value and a program the source does not name, with each exception bound to one start in one function. `pose report --git-stage` keeps writing to the index the operator selected. Independently reviewed by Codex in four rounds: the first found two calls in the ready check that a text pattern had missed and a write sent to the wrong index, the next two tightened the rule.

### Residual risks

The rule guards against a call written by habit; reflection or a generated program would pass it. Executable Git configuration of the repository still runs under the raw reader. Shell scripts the repository ships and declared external checks are outside the rule.

### Follow-ups

- [open] F1: Disarm executable Git configuration on observational gate calls, starting with `core.fsmonitor`, which still runs on a raw `git status` (review round 1 F2). (owner:@oseiaspereira88 crit:medium review:2026-11-27)
- [open] F3: Add a raw-history row to the adversarial corpus, with the launch forms and the operation of an exception. (owner:@oseiaspereira88 crit:low review:2026-11-27)
- [open] F2: Decide which shipped scripts a raw-history rule should reach, starting with `tests/hygiene/temp-cleanup.sh` (review round 1 F4). (owner:@oseiaspereira88 crit:low review:2026-11-27)
