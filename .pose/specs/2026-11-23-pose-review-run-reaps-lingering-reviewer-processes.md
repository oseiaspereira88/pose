---
slug: pose-review-run-reaps-lingering-reviewer-processes
status: in-progress        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-23
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:review-run-reaps-lingering-reviewer-processes
---

# Spec: pose-review-run-reaps-lingering-reviewer-processes

## 1. Intent

### Goal
Keep a delegated review run concludable when the reviewer ends while a process it started is still running, by terminating and reaping the run's remaining processes before the cleanup inventory.

### Business value
Round 2 of `pose-review-readiness-before-seal` (rrn-9d5816c780952f13) produced a complete draft with two valid findings, but its cleanup was `runtime cleanup inventory incomplete; resources retained` because a broad `go test` the reviewer started was still running. The run could not be concluded and its findings had to be carried by hand. On 2026-11-24 a dispatch stopped during setup left reservation `000002.json` of attempt `rat-cfa14ceb…` with no run: every later dispatch failed with `attempt … incomplete or unreadable`, and `pose delegate cancel` refused it as `stalled, not running`; the reservation had to be moved out of the repository by hand.

### Constraints
Cleanup keeps proving that the sealed checkout is unchanged and that no owned resource survives; only the order changes. A run whose resources truly cannot be reclaimed stays unconcludable.

### Non-goals
Changing review independence, round accounting or the attestation preflight.

## 2. Requirements

### Functional
- R1: When the adapter exits, the dispatch shall terminate every remaining process of the run's process group, wait for them within a bound, and only then take the cleanup inventory.
- R2: A run whose leftover processes were terminated and reaped, and whose inventory then completes, shall be concludable; the run record shall say which processes were terminated.
- R3: The review brief shall ask the reviewer to let every command it starts finish before ending.
- R4: A regression test shall dispatch an adapter that leaves a background process running and conclude the run.
- R5: An attempt reservation whose dispatch was interrupted before its run was recorded shall not block later dispatches: `pose delegate ls` shall show it as abandoned with its cause, and `pose delegate cancel` (or the next dispatch) shall release it with a record, never by deleting the reservation.
- R6: A regression test shall interrupt a dispatch between the reservation and the run record and dispatch again.

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp` (review dispatch and conclusion), the manual and its embedded copy.

### Artifacts
- created: .pose/specs/2026-11-23-pose-review-run-reaps-lingering-reviewer-processes.md
- created: pose-mcp/internal/pose/review_reap_linux.go
- created: pose-mcp/internal/pose/review_reap_other.go
- created: pose-mcp/internal/pose/delegation_release.go
- created: pose-mcp/internal/pose/delegation_release_test.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/pose/review_brief.go
- modified: pose-mcp/internal/pose/review_brief_test.go
- modified: pose-mcp/internal/pose/review_brief_conclusion_rules_test.go
- modified: pose-mcp/internal/pose/testdata/review_brief_template_digests.json
- modified: pose-mcp/internal/cli/delegate.go
- created: pose-mcp/internal/cli/review_reap_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md


### Delivery targets

- capability:review-run-reaps-lingering-reviewer-processes module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate.go


### API/contract changes
`pose delegate cancel` releases a stalled attempt that has no run, recording a failed run and its receipt. Brief template `review-brief/10`. The run record's `safety.limitations` names the processes the dispatch ended after the adapter.


### Data/storage changes
A released attempt gets an ordinary failed run record and attempt receipt; no schema changes.


### Technical risks
The leftover processes are found by the `POSE_SEALED_CHECKOUT=<checkout>` entry of their environment, which every process of the run inherits; a process that clears its environment escapes, and the existing cleanup inventory still fails that run (L1). Reading another process's environment needs Linux and the same user; elsewhere the group kill stays the only termination (L2).


## 4. Tasks

### Planning
- [ ] Confirm the cause against the current engine before starting.

### Implementation
- [ ] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R1, R2
Leftover processes are identified by the sealed-checkout entry the dispatch already sets in the run's environment, not by process group or session: adapters such as `codex exec` run each command in a session of its own, which the group kill misses, and the entry names this run's checkout only.

### Decision D2
- Basis: R5
An abandoned attempt is closed by the records a failed run writes (a run record and its receipt), released only when its marker is stalled and its owner proven dead. Nothing is deleted from the ledger, and older engines read the result as an ordinary failed run.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-24: implemented R1–R6. Each has a test that fails with its fix mutated out: the dispatch ends a process the adapter left in a session of its own (the test's leftover keeps writing into the runtime directory), the release records the failed run and receipt and removes the marker and its stall claim, a live owner is never released, and `review-brief/10` asks commands to finish while `/9` stays unchanged.


### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.


### Requirement trace
- R1 [satisfied] test:TestDispatchEndsWhatTheReviewerLeftRunning
- R2 [satisfied] test:TestDispatchEndsWhatTheReviewerLeftRunning
- R3 [satisfied] test:TestTheCurrentBriefAsksCommandsToFinish
- R4 [satisfied] test:TestDispatchEndsWhatTheReviewerLeftRunning
- R5 [satisfied] test:TestAnAbandonedAttemptIsReleasedWithARecord
- R6 [satisfied] test:TestAnAbandonedAttemptIsReleasedWithARecord


### Known gaps
The release is exercised through the Store; the `pose delegate cancel` branch that calls it is not driven end to end, because producing a stalled marker through the CLI needs a dispatch process killed and three silent heartbeat intervals.


## 7. Final Report

### Delivered scope
The dispatch ends the processes a run left running before its cleanup and names them in the run record; `pose delegate cancel` releases an attempt abandoned before its run was recorded; `review-brief/10` asks the reviewer to let its commands finish.


### Files and modules changed
`pose-mcp` (review dispatch, attempt ledger, delegation CLI, brief), the manual in both locales and its embedded copies.


### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.


### Residual risks
L1: a process that clears its environment is not found; the cleanup inventory still fails that run. L2: outside Linux only the process-group kill applies.


### Follow-ups
None.


