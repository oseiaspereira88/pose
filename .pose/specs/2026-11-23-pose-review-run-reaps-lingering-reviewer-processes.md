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
- created: pose-mcp/internal/pose/review_reap_linux_test.go
- created: pose-mcp/internal/pose/review_shim.go
- created: pose-mcp/internal/pose/review_shim_linux.go
- created: pose-mcp/internal/pose/review_shim_test.go
- modified: pose-mcp/internal/pose/main_test.go
- modified: pose-mcp/internal/pose/engine_reads_raw_history_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/cmd/pose/main.go
- modified: pose-mcp/internal/pose/delegation_list.go
- created: pose-mcp/internal/pose/delegation_release.go
- created: pose-mcp/internal/pose/delegation_release_test.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/pose/review_adapter_containment.go
- modified: pose-mcp/internal/pose/review_adapter_containment_regression_test.go
- modified: pose-mcp/internal/pose/review_brief.go
- modified: pose-mcp/internal/pose/review_brief_test.go
- modified: pose-mcp/internal/pose/review_brief_conclusion_rules_test.go
- modified: pose-mcp/internal/pose/testdata/review_brief_template_digests.json
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/help_catalog.go
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
On Linux the pose command runs the adapter under the review run shim, which is the subreaper of the run's subtree and ends what the run leaves in it after the adapter exits, on a timeout and when the dispatcher ends; the subtree is the run's by construction. An embedder that does not enable the shim falls back to the search by the sealed-checkout environment entry, which a reviewer process can clear or set (L6). Outside Linux only the process-group kill applies (L2).

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
The run's leftovers are identified by the process tree, not by the environment: on Linux the adapter runs under a shim that is the subreaper of the run's subtree, so every process the run starts, including one that opens its own session or clears its environment, stays the shim's to end. A subreaper in the dispatcher itself was rejected because it is process-wide there and adopted children other code started in the same process. The sealed-checkout entry the dispatch already sets remains a fallback for an embedder without the shim.

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
- 2026-11-24: the containment record disclosed escaped-descendant termination as `unavailable`, and a regression test pinned that limitation. On Linux it is now `environment-marked` (ended while the descendant keeps the run's sealed-checkout entry, never `available`, so a profile that requires the full guarantee is still refused), the limitation text says so, and the test now proves the escaped child is ended with the run.
- 2026-11-24: review round 1 (rrn-c5c082703d83c834) requested changes. N1: the reservation was published before the active marker, so a dispatch ended in between left an attempt nothing could release; the dispatch now marks the attempt active first and publishes the reservation after, and a stalled marker without a reservation is removed by the release. When marking itself fails, the dispatch still publishes the reservation and records the failed run and its receipt, keeping the contract of `pose-delegation-result-before-marker`. N2: `delegate ls` now names the abandonment cause and the release command, in text and JSON. N3: each process is pinned by a pidfd and its environment read again through it before SIGKILL, so a reused pid is never signalled. N4: a scan that cannot list `/proc` or read a process state fails; a process of this user whose environment cannot be read is counted and disclosed in the run record rather than skipped (L3). N5: the manual says the reaping is Linux-only. Each fix has a test that fails with it mutated out (N4's disclosure is not reproducible in a test: it needs a non-inspectable process).
- 2026-11-24: review round 2 (rrn-2c9fa8ac47c2fbab) requested changes. N1: a reservation an earlier engine left without any marker is now listed by `delegate ls` and released by `cancel` once it has no marker, receipt or run and is older than ten minutes (an earlier engine marked within moments of reserving). N2: a non-inspectable process of this user that did not exist before the adapter started may be the run's, so the scan fails and the run stays unconcludable; ones that predate the run are skipped and disclosed. N3: a release claims the attempt exclusively, so two cancels never record two runs. N4: `TestAnInterruptedDispatchIsReleasedAndTheNextOneRuns` interrupts a real dispatch after its reservation, sees the next dispatch refused with the release command, releases it and dispatches again. N5: a receipt that cannot be inspected is reported with its error. Each fix has a test that fails with it mutated out, except N5.
- 2026-11-24: review round 3 (rrn-667b69e8c01c345c) requested changes. N1: the release claim records its owner; a claim whose owner is gone is taken over, and a release interrupted after recording its run completes with that run instead of recording another. N2: age proves nothing about a markerless reservation, so it is released only with `--confirm-abandoned`, the operator's assertion that no dispatch of it runs, which the recorded run states. N3: reservations and receipts that cannot be read are reported as losses by `delegate ls`. Each fix has a test that fails with it mutated out.
- 2026-11-24: review round 4 (rrn-97acfcc2809e32c2) requested changes. N1: `pose delegate cancel` now asks the release for every attempt that is not running, so a release interrupted after recording its run is completed even when the attempt already shows that run; an abandonment run no longer counts as the attempt's run when looking for orphans. N2: a stale claim is taken over by renaming it, which only one taker wins, then an exclusive publish. N3: with the run history incomplete, the release refuses to rule a run out and the listing reports why. Each fix has a test that fails with it mutated out.
- 2026-11-24: review round 5 (rrn-146bc70da7364cde) requested changes. N1: the published claim and its takeover protocol are replaced by an operating-system lock (`flock`) on a per-attempt lock file, which the kernel releases when its holder ends, so there is nothing to take over and no window between two takers. N2: a release that finds the receipt already published and the marker still stalled completes by removing the marker. N3: the release completes with any run the attempt already recorded (by its dispatch or by an earlier release), never writing a second one. Each fix has a test that fails with it mutated out.
- 2026-11-24: review round 6 (rrn-adf91bb46841bd66) requested changes. N1: an unreadable active marker may be a live attempt's, so no reservation is then taken as markerless. N2: a descendant that leaves the group, starts its own session and clears its environment is now ended too on Linux: the dispatcher makes itself a child subreaper, and after the adapter exits it ends every child reparented to it in another session, started after the adapter, that does not name another run's checkout, pinning each by pidfd and reaping it. N3: the disclosed count holds only processes that predate the run. Each fix has a test that fails with it mutated out.
- 2026-11-24: review round 7 (rrn-9abc2fe5dedc7478) requested changes. N1, N2: a subreaper that cannot be enabled or a run start that cannot be read now fails the cleanup instead of skipping the orphan scan. N3: a process start that cannot be read, in the baseline or the orphan scan, is an error, and a process absent from the baseline is never taken as preexisting. N4: the run record names the ended processes by pid only; their names, which the reviewer chooses, are not recorded. N4 has a test; N1–N3 are error paths the host cannot be made to produce in a test.
- 2026-11-24: the matrix of round 8 failed `TestReviewDispatchRunsAndRecordsAdapters`, and the cause was the design, not the test: a subreaper is process-wide, so the dispatcher also adopted and ended children other code started in the same process (a detached delegation) and could reap a process another goroutine waited for. The subreaper moved into a review run shim: the dispatcher starts the adapter as `<pose> __pose-review-run-shim <adapter path> <argv>`, the shim makes itself the subreaper, runs the adapter as its only child, ends and reaps everything left in its subtree, reports the ended pids in a file the dispatcher reads, and returns the adapter's status, re-raising its signal so a lost process still reads as lost. The shim has no other children, so all it adopts is the run's. The pose command and the test binaries that dispatch enable it; another embedder starts the adapter directly. `TestAnOrphanThatClearedItsEnvironmentIsEnded` fails with the shim's reaping removed.
- 2026-11-24: review round 8 (rrn-f19179878d867e9a) requested changes. N1: the environment entry is the reviewer's to set, so with the shim the dispatcher no longer searches by it; the shim's subtree is the run's by construction, and the search remains only for an embedder without the shim (L6). N2: the shim's report goes through a pipe passed as file descriptor 3, which the shim closes on exec, never through a path the reviewer can write. N3: a timeout sends the shim SIGTERM, and the shim ends the run's subtree before it exits; the group kill after the run stays the backstop. N4: the shim receives SIGTERM when its dispatcher ends (`PR_SET_PDEATHSIG`, checked against the dispatcher's pid at start) and ends the run, so a dead dispatcher never leaves a reviewer running. Each fix has a test that fails with it mutated out.
- 2026-11-24: review round 9 (rrn-5bbd458c7701af31) requested changes. N1 (follow-up): the manual and the recorded limitation now describe the shim as the Linux path and the environment search as the fallback of an embedder without it. N2: the shim's report is bounded (at most 1000 pids, with the total counted, and a bounded error) and the dispatcher drains whatever exceeds its read, so a large report never blocks the shim. N3: a process the shim cannot inspect or end no longer stops the reaping of the others; the problems are reported when the bound expires. N1 and N2 have tests that fail with them mutated out; N3 needs an uninspectable child, which the test host cannot produce.
- 2026-11-24: review round 10 (rrn-ecfc61a1c2b009f6) requested changes. N1: the shim no longer counts a process of another user whose stat it cannot read (procfs mounted hidepid) as a possible run child; only an unreadable process of this user is. N2 (follow-up): the orphan scan lists each directory with a bound and reports a cut listing as a loss. N3 (follow-up): the shim reports a pid only once its signal was delivered, and its error is recorded in the run's limitations. N4 (follow-up): Technical risks, D1 and the fallback helper's comment describe the shim as the Linux path. N2 has a test; N1 and N3 need a hidepid procfs or an unkillable child.


### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.


### Requirement trace
- R1 [satisfied] test:TestDispatchEndsWhatTheReviewerLeftRunning, test:TestATimeoutEndsTheRunsOrphans, test:TestTheShimEndsTheRunWhenItsDispatcherEnds, test:TestTheAdapterDoesNotInheritTheShimReport, test:TestTheShimReportIsBounded, test:TestAnOrphanThatClearedItsEnvironmentIsEnded, test:TestAPidThatLostTheRunsEntryIsNotKilled, test:TestANonInspectableProcessStartedDuringTheRunFailsTheScan
- R2 [satisfied] test:TestDispatchEndsWhatTheReviewerLeftRunning
- R3 [satisfied] test:TestTheCurrentBriefAsksCommandsToFinish
- R4 [satisfied] test:TestDispatchEndsWhatTheReviewerLeftRunning
- R5 [satisfied] test:TestAnAbandonedAttemptIsReleasedWithARecord, test:TestTheAttemptIsMarkedActiveBeforeItIsReserved, test:TestAStalledMarkerWithoutReservationIsListedAndRemoved
- R6 [satisfied] test:TestAnInterruptedDispatchIsReleasedAndTheNextOneRuns, test:TestAMarkerlessReservationOfAnEarlierEngineIsReleased, test:TestOneReleaseRunsAtATime, test:TestARecordedRunCompletesItsAttempt, test:TestAnInterruptedReleaseCompletesWithItsRun, test:TestAnUnreadableReservationIsReportedAsALoss, test:TestAnIncompleteRunHistoryRulesNothingOut, test:TestAnUnreadableMarkerRulesNothingOut, test:TestTheOrphanScanListsBoundedDirectories


### Known gaps
The release is driven through the Store, including after a real interrupted dispatch; the thin `pose delegate cancel` branch that calls it is not driven through the CLI, because a stalled marker there needs a dispatch process killed and three silent heartbeat intervals. N5 (the receipt inspection error) has no test: making the receipt path uninspectable also makes the reservation unreadable first.


## 7. Final Report

### Delivered scope
The dispatch ends the processes a run left running before its cleanup and names them in the run record; `pose delegate cancel` releases an attempt abandoned before its run was recorded; `review-brief/10` asks the reviewer to let its commands finish.


### Files and modules changed
`pose-mcp` (review dispatch, attempt ledger, delegation CLI, brief), the manual in both locales and its embedded copies.


### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.


### Residual risks
L1: on Linux a run's process escapes only while it keeps a live parent outside the run and neither the run's group nor its environment entry, for example one handed to a service manager; the cleanup inventory still fails a run whose files it keeps changing. L2: outside Linux only the process-group kill applies. L6: an embedder that does not enable the shim falls back to the search by the sealed-checkout entry, which a reviewer process can set. L4: a reservation an earlier engine left without a marker records no owner, so its release rests on the operator's confirmation, not on proof. L3: a process of this user that made itself non-inspectable cannot be matched: one that predates the run is skipped and counted in the run record, and one started during the run fails the cleanup, so such a run is not concludable.


### Follow-ups
None.


