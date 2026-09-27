---
slug: pose-spec-transfer-reconcile-terminal
status: done
created_at: 2026-09-26
supersedes:
depends_on: pose-spec-authority-transfer
priority: 0
components: pose-mcp
task_type: feature
delivers: contract:spec-transfer-reconcile-terminal
completed_at: 2026-09-27
---

# Spec: Terminal reconciliation of a coordinator onto a closed executor

## 1. Intent

### Goal
Retire a coordinator spec onto an executor that is already `done` in another
project, with every original requirement disposed, without reopening or
rewriting the executor.

### Business value
Harne8 holds seven draft coordinators whose executors in `pose-dist` are done
and reviewed. `spec-transfer preview` refuses them with
`destination-spec-not-reconcilable`, because the transfer only binds an open
destination and forces it to `blocked` on any reformulated or pending
requirement. The duplicates therefore stay executable in Harne8 and keep
blocking its ABM roadmap as local copies.

### Constraints
Reuse the transfer journal, redirect, archive, receipts, compare-and-swap and
reference rewriting. Keep ordinary transfer plans byte-identical. The
[transfer ADR](../adr/2026-09-21-qualified-artifact-authority-and-explicit-spec-transfer.md)
already requires explicit requirement reconciliation before binding an
existing executor and preservation of its approved evidence.

### Non-goals
Do not reopen, restage or re-review a done executor; do not add a policy
field; do not change ordinary transfer semantics.

## 2. Requirements

### Functional
- R1: `--mode reconcile-terminal` accepts only a destination that is `done`
  with a terminal closeout in its own project.
- R2: The executor file is never written; apply and resume verify its digest
  and refuse, retiring nothing, when it changed after preview.
- R3: The map is N:M: every source requirement has at least one disposition;
  a target is an executor R-ID or a qualified spec (optionally `#R<n>`) that
  resolves; `withdrawn` needs a rationale; `pending` needs a rationale and an
  open target spec. Executor requirements need not all be targeted.
- R4: The source is retired with the existing archive, stub and redirect, and
  consumer references are rewritten; the redirect records the mode and the
  owed requirements, and the source resolves to the executor.
- R5: Plans and redirects of this mode carry schema version 2; a terminal
  plan does not validate under version 1, and mapping fields of this mode are
  refused on an ordinary transfer. Unknown modes fail.
- R6: An interrupted reconciliation resumes from every phase without a second
  authority or an executor write.
- R7: The CLI accepts `--mode` and `--map-file`; help and the manual describe
  the mode.

### Security and compatibility
Authorization, capability, compare-and-swap and path confinement are those of
the transfer. Engines without the mode refuse schema-2 plans and redirects
instead of misreading them.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/spec_transfer.go`, `internal/cli/spec_transfer.go`,
`internal/cli/help_catalog.go`, the manual and its locale and scaffold copies.

### Artifacts
- created: .pose/specs/2026-09-26-pose-spec-transfer-reconcile-terminal.md
- modified: .pose/adr/2026-09-21-qualified-artifact-authority-and-explicit-spec-transfer.md
- modified: pose-mcp/internal/pose/spec_transfer.go
- created: pose-mcp/internal/pose/spec_transfer_reconcile_test.go
- modified: pose-mcp/internal/cli/spec_transfer.go
- modified: pose-mcp/internal/cli/spec_transfer_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/changelogs/unreleased/pose-spec-transfer-reconcile-terminal.md

### Delivery targets
- contract:spec-transfer-reconcile-terminal module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Preview selects schema 2 for the mode, checks the executor's status and
closeout, validates the N:M map with target resolution, and sets the stage
and final digests to the executor's own digest, so the existing apply phases
verify instead of writing. The redirect adds `mode` and `open_obligations`
only in this mode. Rollback is a revert; an applied reconciliation is undone
by restoring the archived source and removing the redirect, as for a
transfer.

## 4. Tasks

- [x] Reproduce the refusal on the Harne8 coordinators.
- [x] Write positive, negative and interruption tests.
- [x] Implement the mode, map validation, versioned redirect and CLI.
- [x] Document the mode in help and the manual.
- [x] Run the matrix, review and close.

## 5. Decisions

Amends the transfer ADR with a terminal-reconciliation section. A distinct
mode was chosen over relaxing `validTransferLifecycle`: relaxing it would let
an ordinary transfer restage and block a closed executor. Owed requirements
must target an open spec, so retiring the coordinator never drops them.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Positive, resume and redirect, R1-R4/R6 | matrix check `spec-authority-transfer-integration` | Executor unchanged; stub, schema-2 redirect with obligations; consumer rewritten; resume idempotent from every phase. |
| Negative gates, R1-R3/R5 | matrix check `spec-authority-transfer-negative-gates` | Open executor, unreviewed executor, uncovered or unresolvable targets, owed requirement in a closed spec, changed executor and schema downgrade refused. |
| CLI, R7 | `go test ./internal/cli -run TestSpecTransferCLIReconcileTerminal -count=1` | `--mode` and `--map-file` reach the engine; unknown map fields refused. |
| Module matrix | `pose validate --strict --module pose-mcp --report` | Required checks pass. |

### Execution log
2026-09-26: `spec-transfer preview` of `pose-abm-subject-evidence` from Harne8
onto its done executor, with all seven requirements `equivalent`, refused with
`destination-spec-not-reconcilable`. The new tests cover the positive path,
eight contract negatives, a changed executor, a schema downgrade and
interruption after `planned`, `prepared` and `source-retired`; ordinary
transfer tests pass unchanged. `go test ./...` and `go vet ./...` pass.
At `41b9a31` the full matrix passed 29/29 and `artifact-check` reported 12
claims and 12 observed paths. Bundle `rvb-30702a8e74e78051` was approved by
attestation `rva-6d6918968354d3cc`, recorded by the agent under explicit
authorization from the user to self-attest. The change touched the manual and
the transfer engine, so the foundation, transfer and seven ABM executor
reviews were resealed and reattested.

### Requirement trace
- R1 [satisfied] test:TestSpecTransferNegativeReconcileTerminalGates
- R2 [satisfied] test:TestSpecTransferReconcileTerminalRetiresCoordinatorWithoutTouchingExecutor test:TestSpecTransferNegativeReconcileTerminalChangedExecutorAndOldSchema
- R3 [satisfied] test:TestSpecTransferNegativeReconcileTerminalGates
- R4 [satisfied] test:TestSpecTransferReconcileTerminalRetiresCoordinatorWithoutTouchingExecutor
- R5 [satisfied] test:TestSpecTransferNegativeReconcileTerminalChangedExecutorAndOldSchema test:TestSpecTransferNegativeReconcileTerminalGates
- R6 [satisfied] test:TestSpecTransferReconcileTerminalResumesAfterInterruption
- R7 [satisfied] test:TestSpecTransferCLIReconcileTerminalReadsModeAndMapFile

## 7. Final Report

### Delivered scope
A coordinator can be retired onto a done, closed executor in another project
with every requirement disposed and owed work kept in an open spec, without
writing the executor.

### Residual risks
Rollback of an applied reconciliation is manual (restore the archived source,
remove the redirect), as for a transfer.

### Follow-ups
- [open] Apply the reconciliation to Harne8's seven ABM coordinators after a rehearsal on populated copies. (owner:@harne8-platform crit:high review:2026-10-03)
