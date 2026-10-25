---
slug: pose-delegated-review-run-continuity
status: in-progress
created_at: 2026-10-18
completed_at:
supersedes:
depends_on: pose-delegated-review-attempt-ledger
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers: capability:delegated-run-continuity
---

# Spec: A delegated review that is interrupted is resumed, not restarted

## 1. Intent

### Goal

Let a delegated review that stopped before it finished (the reviewer's usage limit was reached, the process timed out or died, the host was lost) continue from where it stopped, on the same sealed commit and brief, as one attempt, instead of being restarted from nothing or mistaken for a completed review.

### Business value

Origin: the B06 consolidation reviews (2026-10-17). A review of a large diff takes tens of minutes of reviewer time. The reviewer's usage limit was reached during the programme, and the work was picked up by hand: the session was resumed with a personal launcher flag once the maintainer reported that the limit had reset, and nobody could say whether the half-finished run counted against the attempt budget or as a rerun. The engine knows none of this: `pose-delegated-review-dispatch` records an interruption as a failed run, and `pose-delegated-review-attempt-ledger` would refuse "rerun without reason" on a bundle whose only run was interrupted. The cost of an interruption should be the time to the reset, not the review.

### Constraints

Reuses the run record, the attempt reservation and the exclusive publisher of the ledger and state-consistency work; adds no runner. Vendor-specific behaviour (how a session is named, how a limit is worded) lives in the adapter declaration, never in the engine. Nothing here makes an interrupted run a review. Implementation authorized by continuation on 2026-10-25, baseline `3fa69b6`.

### Non-goals

Resuming across a changed sealed bundle (that is a new attempt, carrying prior findings: `pose-delegated-review-round-protocol`); choosing a different vendor on the maintainer's behalf; waiting for a limit that has no stated reset.

## 2. Requirements

### Functional

- R1: A run shall end in exactly one state of `completed`, `failed` or `interrupted`; an `interrupted` run shall record its cause (`usage-limit`, `timeout`, `process-lost`, `cancelled`, or `unknown`), whether it is resumable, and, when the adapter reports one, the instant the limit resets.
- R2: An adapter shall declare, in `.pose/policy/reviewers.json`, how its limit is recognised (exit codes and output patterns, and how the reset instant is read), how its session identifier is captured, and its resume command template. An adapter that declares none shall never have an interruption classified as `usage-limit`, and its interrupted runs shall be reported `unknown` and not resumable.
- R3: `pose review resume <run>` shall continue a resumable run on a worktree at the same sealed commit, with the same brief (digest compared), the same adapter and the vendor session captured by the interrupted run, and shall record the continuation as a new run linked `resumes: <run>` under the same attempt.
- R4: Resume shall be refused, with the reason, when the bundle or the brief digest changed, when the run already reached `completed` or `failed`, when the recreated worktree's tree digest differs from the one the run recorded, or when the resume budget (default 3 per attempt) or the attempt's accumulated wall-clock budget is spent.
- R5: A resume shall consume the reservation of the attempt it continues and shall not be a rerun: the ledger's rerun rule, its quota and `pose insights` count one attempt, and report the resumes of that attempt separately.
- R6: The draft and findings of an interrupted run shall be preserved in the run record by digest and disclosed as partial, and shall never be accepted as the review or counted as a rejection; only the draft of the run that reaches `completed` is the review.
- R7: `pose review resume <run> --wait-until-reset[=<max>]` shall sleep until the recorded reset instant (plus a bounded jitter) and then resume, shall never poll the vendor, and shall refuse when there is no recorded reset or the reset is later than `<max>`.
- R8: After a `usage-limit` interruption, dispatching the same bundle through a different adapter shall be allowed only with an explicit `--via` and shall be recorded as a new attempt with reason `limit-of:<run>`; the independence evidence shall name both runs, and the engine shall never switch adapter by itself.
- R9: `pose review status` shall show, for every interrupted run, its cause, whether it is resumable, the reset instant and the time left to it; `pose doctor` shall report an attempt whose last run is `interrupted` and not resumed.

### Non-functional

- A resume is idempotent under a crash: a second `resume` of the same run while a continuation exists shall return that continuation, not start another.
- Every transition is published through the exclusive publisher; a torn state is never visible.

### Security

- The captured session identifier is a reference to a vendor session and shall not be written to the transcript or to published reports; the resume template runs under the same containment as the first run.
- A run record is evidence: a resume that finds its predecessor rewritten or removed shall refuse.

### Compatibility

- Adapters without the new declarations keep working as they do now; their interruptions are classified `unknown`.

## 3. Technical Plan

### Affected areas

Run-state model and the supervisor's classification, adapter policy reader, `review resume`, `review status`, ledger accounting of resumes, doctor and insights.

### Artifacts

- created: .pose/adr/2026-10-25-continue-delegated-runs-under-one-immutable-attempt.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- created: .pose/changelogs/unreleased/pose-delegated-review-run-continuity.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/reports/2026-10-25-b16-run-continuity.md
- modified: .pose/specs/2026-10-18-pose-delegated-review-run-continuity.md
- created: .pose/starts/pose-delegated-review-run-continuity.json
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/git_snapshot_integrity_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/insights.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_dispatch.go
- modified: pose-mcp/internal/cli/review_dispatch_test.go
- created: pose-mcp/internal/cli/review_run_continuity.go
- created: pose-mcp/internal/cli/review_run_continuity_test.go
- created: pose-mcp/internal/mcpserver/review_run_continuity_test.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- created: pose-mcp/internal/pose/review_run_continuity.go
- created: pose-mcp/internal/pose/review_run_continuity_test.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegated-run-continuity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_dispatch.go

### API/contract changes

New run states and fields; new `review resume` subcommand; additions to `reviewers.json` (`limit`, `session`, `resume`).

## 4. Tasks

- [x] Run states, interruption causes and adapter declarations
- [x] Resume with worktree and brief verification
- [x] Attempt accounting and ledger interaction
- [x] Wait-until-reset and explicit vendor fallback
- [x] Status, doctor and insights

## 5. Decisions

### Decision D1
- Basis: R3, R5, R8
- ADR: [Continue under one immutable attempt](../adr/2026-10-25-continue-delegated-runs-under-one-immutable-attempt.md)
- Knowledge: knowledge:pose-b07-implementation, knowledge:adr-sealed-review-bundles-review
- Minimal option: treat a resume as a new dispatch with `--reason`
- Selected option: a resume is a continuation of one attempt, linked by `resumes:`
- Rationale: a reason-bearing rerun is exactly what the ledger exists to make rare and visible; an interruption is not the reviewer's verdict and must not be priced like one
- Consequences: attempts and runs become distinct counts; insights report both
- Falsifier: an interrupted run whose resume is counted as a rerun, or a rerun that is disguised as a resume

### Decision D2
- Basis: R8
- Minimal option: fall back automatically to the other vendor when a limit is reached
- Selected option: never; the maintainer chooses, and the choice is recorded
- Rationale: independence is evidence about who reviewed; changing reviewer silently changes the evidence
- Consequences: a limit can stop a review until the reset or a human decision
- Falsifier: an attestation whose run chain names an adapter nobody selected

## 6. Validation

### Strategy

A fake adapter that is interrupted at chosen points (limit with and without a reset, timeout, killed process) and resumes: the resumed run reaches the same draft as an uninterrupted run; a changed bundle, a changed brief, a rewritten predecessor and an exhausted budget are refused; the ledger counts one attempt; a fallback requires `--via` and is linked to the interrupted run.

### Requirement trace

- R1 [satisfied] test:TestRunContinuityClassificationAndWait test:TestRunContinuityObservedProcessLoss test:TestRunContinuityLaunchFailureAndSessionRotation
- R2 [satisfied] test:TestRunContinuityPolicyAndPartialIntegrity test:TestRunContinuityOutputCaptureAndConflicts
- R3 [satisfied] test:TestRunContinuityInterruptedResumeAndAccounting test:TestRunContinuityConcurrentResumeLaunchesOnce
- R4 [satisfied] test:TestRunContinuityBudgetsAndChangedInputs test:TestRunContinuityRefusalAndExplicitFallback
- R5 [satisfied] test:TestRunContinuityInterruptedResumeAndAccounting test:TestRunContinuityConcurrentResumeLaunchesOnce
- R6 [satisfied] test:TestRunContinuityMCPPartialNeverBecomesVerdict test:TestRunContinuityOutputCaptureAndConflicts
- R7 [satisfied] test:TestRunContinuityClassificationAndWait test:TestRunContinuityRefusalAndExplicitFallback
- R8 [satisfied] test:TestRunContinuityRefusalAndExplicitFallback
- R9 [satisfied] test:TestRunContinuityDoctorStatusAndPending

### Required execution plan

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Classification, redaction, budgets and chain integrity (required) | `cd pose-mcp && go test ./internal/pose ./internal/cli ./internal/mcpserver -run RunContinuity -count=1` | Interrupted partials never become verdicts; same-attempt continuation; refusal on changed evidence, duplicate claim and spent budget |
| Concurrent resume and publication (required) | `cd pose-mcp && go test -race ./internal/pose ./internal/cli -run RunContinuity -count=1` | One continuation claim and no duplicate adapter launch |
| Full composition and regression (required) | `pose validate --strict --json-out .pose/results/delivery-validation.json` with TMPDIR=/var/tmp | All registered checks pass with current provenance |
| Security (required) | `cd pose-mcp && govulncheck ./...` and pinned CI gitleaks | No new vulnerabilities or secrets; session absent from persisted transcript, output and summaries |

## 7. Final Report

### Delivered scope

Implemented interruption classification, private session capture, same-attempt continuations, bounded wait and explicit fallback, with CLI/doctor/insights/MCP disclosure. Canonical validation, native review and closeout determine delivery; see report `2026-10-25-b16-run-continuity.md`.

### Residual risks

Filesystem/Git anchors remain trusted. Private session loss prevents recovery. A claim without a captured run/result remains explicit pending evidence, not an automatic host-loss recovery or permission to relaunch. No review authority, vendor identity authentication, new runner, policy adoption or adapter containment is introduced.

### Follow-ups

No new residual task. Round protocol, producer authentication and adapter containment remain in their canonical specs; this delivery does not change their disposition.
