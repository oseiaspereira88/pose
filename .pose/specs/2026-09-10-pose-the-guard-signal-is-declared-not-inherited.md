---
slug: pose-the-guard-signal-is-declared-not-inherited
status: in-progress
created_at: 2026-09-10
completed_at:
supersedes:
depends_on: pose-cross-version-guard-is-this-repositorys
priority: 0
components: pose-mcp
delivers:
task_type: refactor
---

# Spec: The guard reads a signal the workflow declares

## 1. Intent

### Goal
Have the cross-version guard key on a variable this repository's workflows
declare, and assert in the workflow contract that every job running the suite
declares it.

### Business value
The guard has now been wrong twice, in opposite directions, and both times the
cause was reading a signal someone else owns.

Keying on `CI` broke every repository that vendors the engine and runs its
suite: those are CI too, their submodule checkouts have no tags, and there was
nothing for them to configure. Keying on `GITHUB_REPOSITORY` fixed that and left
the other half open — the variable is the provider's to set, so on a provider
that does not set it this repository's own CI skips in silence, which is exactly
the hole the guard exists to close.

A variable the workflow declares has neither problem. It is set only where a
checkout was prepared with the release history, it cannot be inherited by a
consumer, and — unlike a provider variable — a static test can assert that the
jobs which need it declare it.

### Constraints
- Every job that runs the Go suite with full history must declare it, or the
  test it protects skips there.

### Non-goals
- Removing the `fetch-depth: 0` assertion. Full history is the promise; the
  variable is the job saying so where the test can read it, and both are needed.

---

## 2. Requirements

### Functional
- R1: The guard shall fail when the workflow declares the release history is
  available, and skip otherwise.
- R2: Every workflow job that runs the Go suite shall declare it, enforced by
  the workflow contract test.

### Non-functional
- A consumer that vendors the engine skips, with nothing to configure.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/release_compatibility_test.go` — the guard
- `pose-mcp/internal/version/workflow_history_depth_test.go` — the contract
- `.github/workflows/ci.yml`, `security.yml`, `release.yml` — the declaration

### Artifacts
- created: .pose/specs/2026-09-10-pose-the-guard-signal-is-declared-not-inherited.md
- created: .pose/changelogs/unreleased/pose-the-guard-signal-is-declared-not-inherited.md
- modified: pose-mcp/internal/cli/release_compatibility_test.go
- modified: pose-mcp/internal/version/workflow_history_depth_test.go
- modified: .github/workflows/ci.yml
- modified: .github/workflows/security.yml
- modified: .github/workflows/release.yml

### Technical risks
- A job added later that runs the suite and forgets the variable would skip the
  test silently. That is what the contract test exists to catch, and it caught a
  job on its first run.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The guard reads the declared variable (R1)
- [x] Increment 2: The contract requires every suite job to declare it (R2)

### Validation
- [x] All three environments exercised, and the contract shown catching a job

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the guard could keep reading `GITHUB_REPOSITORY`, or read something
  this repository declares.
- Decision: declare it.
- Rationale: a provider variable is right about *which* repository and silent
  about whether the checkout has what the test needs — and a static test cannot
  assert a provider will set it. A declared variable says the thing that
  actually matters, is impossible for a consumer to inherit, and is exactly what
  the workflow contract can check.

### Decision 2
- Date: 2026-09-10
- Context: the contract test found a third job running the suite —
  `release.yml`'s `Tests + installer E2E` — that neither of the previous two
  changes had touched.
- Decision: declare it there too.
- Rationale: that is the run whose verdict cuts the release. It is the one place
  the cross-version test must never skip in silence, and it was the one place
  nobody had thought to look. Both earlier fixes enumerated the jobs by hand and
  both missed it.

---

## 6. Validation

### Strategy
Run the guard under each environment it can meet, and require the contract test
to name a job that runs the suite without declaring the variable.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-10
- Environment: local, Go 1.26
- Notes: with the variable set and the resolved tag forced to one that does not
  exist, the guard fails naming the declaration. Without it, the same forced tag
  skips — including under `CI=1` with a consumer's `GITHUB_REPOSITORY`, which is
  the case that broke the adopting repository. The contract test named
  `release.yml`'s suite step on its first run, before that step declared
  anything.

### Results summary
- Successes: R1, R2 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <skipUnlessCI keys on POSE_RELEASE_HISTORY_AVAILABLE; verified failing with it set and skipping without it, including under a consumer's CI environment>
- R2 [satisfied] <TestJobsRunningTheGoSuiteCheckOutFullHistory now requires the declaration alongside fetch-depth: 0, and reported release.yml's suite step until it was added>

### Known gaps
- The variable is declared per step. A job that runs the suite in a second step
  without it would be half-covered, and the contract test reads the job rather
  than the step.

---

## 7. Final Report

### Summary
The guard reads something this repository says about its own checkout, and the
workflow contract keeps every job that needs it saying so.

### Follow-ups

- [open] Read the declaration per step rather than per job, so a job that runs the suite twice and declares it once is not reported as covered — owner:unowned crit:low review:2027-03-10
