---
slug: pose-red-signals-reach-a-person
status: done
created_at: 2026-10-09
completed_at: 2026-10-09
supersedes:
depends_on: pose-release-security-gate-integrity
priority: 0
components: ci, release
task_type: feature
changelog: none
delivers: governance:red-signals-reach-a-person
---

# Spec: Red signals reach a person

## 1. Intent

### Goal

Make a failed gating workflow on `main`, on a release tag or on a schedule reach the maintainer as a notification, once per workflow, and clear itself when the workflow passes again.

### Business value

Origin: the two open follow-ups of `pose-release-security-gate-integrity` (crit high and medium), prioritized by the maintainer on 2026-10-09 for the release. Ten consecutive releases failed and CI stayed red on `main` for weeks: the workflows failed correctly and nobody was told. On 2026-10-08 the Security workflow went red on `main` for new Go advisories and was found only because a session happened to look.

### Constraints

No third-party action and no checkout: the workflow uses the runner's `gh` with the job token, raised to `issues: write` for that job only. Every value the `workflow_run` event carries is bound through `env` and validated before use, as the event-ref contract requires. Pull requests and forks never alert.

### Non-goals

Paging outside GitHub (email, chat); the same alert in Harne8, which is that repository's own follow-up.

## 2. Requirements

### Functional

- R1: When a watched workflow ends in failure or timeout on `main`, on a release tag or on a schedule, the alert workflow shall open one issue per workflow, labelled `red-signal` and assigned to the repository owner, or comment on the open one.
- R2: When that workflow next succeeds there, the alert workflow shall close its issue.
- R3: Every workflow that runs on push, tag, release, schedule or `workflow_run` shall be watched or exempt with a stated reason, and a test shall fail otherwise.

### Non-functional

- None.

### Security

- Event-supplied names, refs, SHAs and URLs are validated against fixed patterns; an unexpected value stops the step without calling `gh`.

### Compatibility

- Additive: one workflow and one test; existing workflows are unchanged.

## 3. Technical Plan

### Affected areas

`.github/workflows`, the workflow contract tests and the validation matrix.

### Artifacts

- created: .pose/specs/2026-10-09-pose-red-signals-reach-a-person.md
- created: .github/workflows/failure-alert.yml
- created: pose-mcp/internal/version/failure_alert_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-09-06-pose-release-security-gate-integrity.md

### Delivery targets

- governance:red-signals-reach-a-person module:. profile:release-governance entrypoint:.github/workflows/failure-alert.yml

### Technical risks

- The first real alert is only observable after a red run on `main`; the step's logic was exercised locally against a recording `gh`.

## 4. Tasks

### Implementation
- [x] Alert workflow with validated event values
- [x] Contract test that fails on an unwatched workflow, registered as `red-signal-contract`

### Validation
- [x] actionlint, the workflow contract tests, and a local run of the step for each case

## 5. Decisions

### Decision D1
- Date: 2026-10-09
- Context: GitHub notifies an assignee by default; an issue persists and deduplicates.
- Options considered: (a) an issue per workflow, commented and auto-closed; (b) a third-party notification action; (c) email from the runner.
- Decision: (a).
- Rationale: no new secret or third-party action, one place per red workflow, and the history stays in the repository.
- Consequences: the owner must keep GitHub notifications for assignments enabled.

## 6. Validation

### Strategy

`TestFailureAlertWatchesEveryWorkflowOutsidePullRequests` reads every workflow and fails when one that runs outside pull requests is neither watched nor exempt, when the watched list names a missing workflow, or when the name check in the step does not accept a watched one; it failed with the workflow absent. `TestFailureAlertContractRejectsAnUnwatchedWorkflow` proves the check rejects an unwatched scheduled workflow. The step was run locally with a recording `gh`: a failed Release on a tag opened an issue, a failed CI with one open commented on it, a success closed it (on `main` and on a tag), a feature branch did nothing, and an unknown workflow name or a ref with shell characters stopped the step before any call. `actionlint` v1.7.7 reports nothing.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/version -run FailureAlert`
- Scope: workflows
- Expected: pass

### Requirement trace

- R1 [satisfied] governance:red-signals-reach-a-person check:red-signal-contract evidence:integration test:TestFailureAlertWatchesEveryWorkflowOutsidePullRequests
- R2 [satisfied] governance:red-signals-reach-a-person check:red-signal-contract evidence:integration test:TestFailureAlertWatchesEveryWorkflowOutsidePullRequests
- R3 [satisfied] governance:red-signals-reach-a-person check:red-signal-contract evidence:integration test:TestFailureAlertContractRejectsAnUnwatchedWorkflow

### Known gaps

The alert itself is exercised on GitHub only when a watched workflow next fails.

## 7. Final Report

### Delivered scope

`failure-alert.yml` watches CI, POSE docs, Governance audit, Release, Release liveness, Scorecard, Security and Verify release. A failure or timeout on `main`, a release tag or a schedule opens or comments on one `red-signal` issue per workflow, assigned to the owner; the next success closes it. Package channels, the Dependabot repair and the alert itself are exempt with reasons, and a contract test keeps the list complete.

### Residual risks

- The first real alert happens on the next red run.
- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
