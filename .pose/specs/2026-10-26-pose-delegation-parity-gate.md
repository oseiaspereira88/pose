---
slug: pose-delegation-parity-gate
status: done
created_at: 2026-10-26
completed_at: 2026-10-30
supersedes:
depends_on: pose-delegation-task, pose-delegated-review-capability, pose-delegation-task-detach-and-state
priority: 3
components: pose-mcp
task_type: feature
delivers: capability:delegation-task
---

# Spec: Personal launchers retire only at proven parity

## 1. Intent

### Goal

Make the retirement of the personal launchers a gate: a versioned matrix maps each of their capabilities to a native command and a test, and a check fails while any capability is uncovered or any active document still points at the replaced commands.

### Business value

Decided by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md). The launchers carry behaviour learned in operation — removing the parent's session variables, replaying resumed chains, hinting the resume command on a usage limit — that a rewrite can silently drop. A matrix that must fail before it passes keeps "nothing lost" checkable instead of remembered.

### Constraints

POSE cannot modify files outside the repository; retiring a person's launcher, skills and notes is an operator step that this spec documents and does not perform. Historical records (ADRs, reports, roadmaps, changelogs, done specs) keep naming the launchers as written.

### Non-goals

Implementing any capability; the specs of roadmap `delegation` do that.

## 2. Requirements

### Functional

- R1: A versioned parity matrix shall list one row per launcher capability — stable identifier, launcher, capability, native command, covering test — seeded with every capability the ADR's "Retirement by parity" section enumerates.
- R2: The parity check shall fail when a row has no native command or no covering test, when a named test does not exist, or when a capability enumerated by the ADR has no row.
- R3: The parity check shall be shown to fail against a matrix with an uncovered row before it is accepted as passing, and that failing result shall be kept as evidence.
- R4: A documentation check shall fail when an active document, skill or help text of the distribution names `pose review dispatch`, `pose review resume`, `pose review status` or a personal launcher, outside deprecation notes and historical records.
- R5: The operating manual shall document how an operator retires a personal launcher once the parity check passes: what to remove, what to keep, and that historical records stay as written.
- R6: The release that follows a passing parity check shall remove the deprecated aliases named by `pose-delegation-surface` R5.

### Non-functional

- Both checks run offline and in the ordinary validation matrix.

### Security

- None beyond the checks' read-only access to the repository.

### Compatibility

- Removing the aliases is announced in the changelog of the release that removes them.

## 3. Technical Plan

### Affected areas

Parity matrix and its check, documentation check, operating manual section, validation matrix entries.

### Artifacts

- modified: .pose/specs/2026-10-26-pose-delegation-parity-gate.md
- created: .pose/starts/pose-delegation-parity-gate.json
- created: .pose/changelogs/unreleased/pose-delegation-parity-gate.md
- created: .pose/parity/delegation-launchers.json
- created: .pose/reports/2026-10-29-launcher-parity-failing-first.md
- modified: .pose/indexes/validation-matrix.json
- created: pose-mcp/internal/cli/launcher_parity_test.go
- modified: pose-mcp/internal/cli/review_dispatch.go
- modified: pose-mcp/internal/cli/review_run_continuity.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/delegate_task.go
- modified: pose-mcp/internal/cli/delegate_task_detach_test.go
- modified: pose-mcp/internal/pose/delegation_task.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_subject_capabilities_test.go
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/b07_independent_probe_test.go
- modified: pose-mcp/internal/cli/b07_r2_probe_test.go
- modified: pose-mcp/internal/cli/b07_r3_probe_test.go
- modified: pose-mcp/internal/cli/b07_r4b_probe_test.go
- modified: pose-mcp/internal/cli/delegated_review_request_test.go
- modified: pose-mcp/internal/cli/delegated_verdicts_test.go
- modified: pose-mcp/internal/cli/delegate_routing_test.go
- modified: pose-mcp/internal/cli/delegate_test.go
- modified: pose-mcp/internal/cli/delegate_watch_test.go
- modified: pose-mcp/internal/cli/git_snapshot_integrity_test.go
- modified: pose-mcp/internal/cli/review_adapter_containment_test.go
- modified: pose-mcp/internal/cli/review_attempt_ledger_test.go
- modified: pose-mcp/internal/cli/review_dispatch_test.go
- modified: pose-mcp/internal/cli/review_run_continuity_test.go
- modified: pose-mcp/internal/cli/review_run_provenance_test.go
- modified: docs-site/docs/cli.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegation-task module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate_task.go

### API/contract changes

Parity matrix schema 1 (`.pose/parity/delegation-launchers.json`); removal of `pose review dispatch|resume|status`, which now exit 2 naming their replacement; `pose delegate review|adjudicate|smoke` discloses the run's containment (`delegate.safety`, `safety` in `--json`); the `launcher-parity` check of the validation matrix.

## 4. Tasks

- [x] Matrix seeded from the ADR inventory
- [x] Parity check, proven failing first
- [x] Documentation check
- [x] Operator retirement section
- [x] Alias removal in the following release

## 5. Decisions

### Decision D1
- Basis: R2, R3
- Minimal option: a checklist in the closeout of the last delegation spec
- Selected option: a matrix with a check that must fail before it passes
- Rationale: a checklist is read once and forgotten; the capabilities most likely to be lost are the ones nobody remembers writing
- Consequences: each new launcher capability discovered later is a new row, not a note
- Falsifier: a launcher capability found missing after retirement whose row the matrix accepted

### Decision D2
- Basis: R6
- Minimal option: keep the aliases for the minor version `pose-delegation-surface` promised, and remove them in a later spec
- Selected option: remove `pose review dispatch`, `resume` and `status` in this spec, so the release that carries the passing parity check is the one without them (the maintainer's decision of 2026-10-29)
- Rationale: the deprecation and its replacement ship together; keeping aliases one more release keeps three commands documented only to say they are gone, and the launchers' retirement is the moment the vocabulary changes
- Consequences: a user of 7.4.x who runs `pose review dispatch|resume|status` after the upgrade gets an unknown-command error with no release that printed a deprecation first; the changelog marks it breaking and names each replacement
- Falsifier: an installed instance or a skill that still calls a removed alias after the upgrade, found without the changelog naming it

## 6. Validation

### Strategy

The check fails on a matrix with an empty test, a missing test and a missing ADR capability, and passes on the complete matrix; the documentation check fails on a fixture that names a replaced command outside the allowed places.

### Requirement trace

- R1 [satisfied] check:launcher-parity test:TestLauncherParityMatrixCoversEveryLauncherCapability report:.pose/results/delivery-validation.json commit:badde66753299900be9f6d8f56b5ef8e346700a7 — `.pose/parity/delegation-launchers.json` schema 1 has one row per capability of the ADR's "Retirement by parity" section, 21 in all, each with its `pose delegate` command and covering tests.
- R2 [satisfied] check:launcher-parity test:TestLauncherParityCheckFailsOnEveryUncoveredRow test:TestLauncherParityFollowsTheADRsText report:.pose/results/delivery-validation.json commit:badde66753299900be9f6d8f56b5ef8e346700a7 — The check fails on a row with no command or test, a test or a documented flag that does not exist, an ADR capability with no row, and inventory text it cannot read.
- R3 [satisfied] check:launcher-parity test:TestLauncherParityCheckFailsOnEveryUncoveredRow report:.pose/reports/2026-10-29-launcher-parity-failing-first.md commit:badde66753299900be9f6d8f56b5ef8e346700a7 — Its first run failed on five real gaps, kept verbatim, before they were closed.
- R4 [satisfied] check:launcher-parity test:TestActiveDocumentsNameOnlyPoseDelegate test:TestActiveDocumentCheckFailsOnAReplacedCommand report:.pose/results/delivery-validation.json commit:badde66753299900be9f6d8f56b5ef8e346700a7 — Manuals, docs-site, skills, workflows, rules, scaffold, help and the Go strings of every release configuration name a replaced command only in a listed removal note.
- R5 [satisfied] check:test test:TestManualLocaleParity test:TestActiveDocumentsNameOnlyPoseDelegate report:.pose/results/delivery-validation.json commit:badde66753299900be9f6d8f56b5ef8e346700a7 — "Retiring a personal launcher" in both manuals and their scaffold copies: what to declare, replace, remove and keep, raw sessions included, historical records as written.
- R6 [satisfied] check:delegation-surface-integration test:TestTheRemovedReviewAliasesNameTheirReplacement test:TestAdapterContainmentCLIDisclosesItBeforeTheRun test:TestRunContinuityDoctorStatusAndPending report:.pose/results/delivery-validation.json commit:badde66753299900be9f6d8f56b5ef8e346700a7 — `pose review dispatch|resume|status` are removed in this spec, ahead of the release that carries it (D2); `pose delegate` keeps what they gave: the brief without an agent, containment before the run, a pending continuation answered with exit 0.

## 7. Final Report

### Delivered scope

Milestone `retirement` of roadmap `delegation`, on implementation head `badde66753299900be9f6d8f56b5ef8e346700a7`: the versioned parity matrix and its check (R1–R3), the active-document check (R4), the operator's retirement section (R5) and the removal of `pose review dispatch`, `resume` and `status` (R6, D2, the maintainer's decision of 2026-10-29: removed in this spec although no release printed their deprecation; the changelog marks it breaking). Closing the matrix took capabilities the launchers had and POSE lacked: `pose delegate task --session <id> --agent <name>` continues a session POSE never recorded, and the alias removal kept three guarantees of `dispatch` and `review resume` in `pose delegate` — the brief when no agent can run, the containment disclosed before a run starts, a reserved continuation answered as pending with exit 0. `.pose/parity/` is classified as review governance. `pose-mcp/internal/cli/review_dispatch.go` is kept as the home of the removed aliases' answer rather than deleted: five done specs declare it as a current artifact, and POSE has no record of a later spec removing a file an earlier one created (a sealed first closeout, `rvb-b02c94d7b79d4088` and `rva-0b2e83cc88087b2b`, failed `check --strict` on its deletion and is kept as history). The personal launchers outside the repository are not touched: retiring them is the operator's step the manual describes.

Independent review: eight rounds by Codex `gpt-6.1-sol` (OpenAI, reasoning medium), a different vendor than the implementer Claude Opus 5.5 (Anthropic). Rounds 1–7 requested changes (PG-R1-01..10, R2-01..07, R3-01..03, R4-01..02, R5-01..03, R6-01..03, R7-01..03), each fixed in its own commit; most hardened the active-document reader, which ended reading Go strings through the type checker, under each release configuration's files as `go list` selects them. Round 8 approved. Reports are local review evidence under `/tmp/pose-roadmaps-evidence/parity/` (untracked); the native bundle and attestation are the sealed record.

### Validation executed

- `go test ./...` in pose-mcp without agent session markers (`GOMAXPROCS=2 GOFLAGS=-p=1`) passed on the implementation head; the `launcher-parity` lane passes in about 6 s.
- Mutations: 30 production and check mutants, each failing an assertion after a passing baseline (`mutate.py`); two were dropped as equivalent when the code they targeted was replaced.
- The native strict validation, index, seal, attestation and verify of the closeout are recorded by the bundle and attestation of the closeout commit.

### Residual risks

The check proves that a row's command and tests exist, not that they exercise the capability; review of the cited tests is the defence (corpus known-gap `parity-row-cites-an-unrelated-test`). The Go reader resolves constants within a package: a fragment imported from another package, text in a cgo preamble, and configurations the release does not build are not read (known gaps `imported-fragment-completes-a-replaced-command`, `cgo-preamble-text-reaches-usage`). Exact removal notes are an allowlist kept by hand.

### Follow-ups

- [open] F1: harden the release-definition reader before the release adopts another representation or a source-selecting setting: reject every unmodelled line, multiline lists and empty dimensions, and model or refuse source-selecting environment variables and build directories (PG-R8-01, consolidating PG-R7-03). (owner:@oseiaspereira88 crit:low review:2026-11-27)
- [open] F2: detect incomplete package enumeration beyond what the gate refuses today, document vendor, nested-module and generated-source boundaries, and pin or validate the toolchain whose version tags select files (PG-R8-02). (owner:@oseiaspereira88 crit:low review:2026-11-27)
- [open] F3: read configurations the release does not build (other GOOS/GOARCH, custom tags) before shipping one, and check generated operator output (PG-R6-02). (owner:@oseiaspereira88 crit:low review:2026-11-27)
- [open] F5: let a done spec's `removed:` claim satisfy the existence of a path an earlier spec declared current, so removing a file does not require keeping it (`review_dispatch.go` here). (owner:@pose-maintainers crit:low review:2026-11-27)
- [open] F4: resolve string fragments imported from other packages of the module and add a reader or runtime check before a cgo preamble supplies operator text (PG-R5-02, PG-R5-03; corpus known gaps). (owner:@oseiaspereira88 crit:low review:2026-11-27)
