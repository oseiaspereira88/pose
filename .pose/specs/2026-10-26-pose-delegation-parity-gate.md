---
slug: pose-delegation-parity-gate
status: in-progress
created_at: 2026-10-26
completed_at:
supersedes:
depends_on: pose-delegation-task, pose-delegated-review-capability, pose-delegation-task-detach-and-state
priority: 3
components: pose-mcp
task_type: feature
delivers:
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
- removed: pose-mcp/internal/cli/review_dispatch.go
- modified: pose-mcp/internal/cli/review_run_continuity.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/delegate.go
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

### API/contract changes

Parity matrix schema 1 (`.pose/parity/delegation-launchers.json`); removal of `pose review dispatch|resume|status`, which now exit 2 naming their replacement; `pose delegate review|adjudicate|smoke` discloses the run's containment (`delegate.safety`, `safety` in `--json`); the `launcher-parity` check of the validation matrix.

## 4. Tasks

- [ ] Matrix seeded from the ADR inventory
- [ ] Parity check, proven failing first
- [ ] Documentation check
- [ ] Operator retirement section
- [ ] Alias removal in the following release

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

## 7. Final Report

### Delivered scope

Not started: milestone `retirement` of roadmap `delegation`, after `pose-delegation-task-detach-and-state`.

### Residual risks

None yet.

### Follow-ups
