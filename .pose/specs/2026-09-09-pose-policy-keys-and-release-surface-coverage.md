---
slug: pose-policy-keys-and-release-surface-coverage
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-doctor-reports-unread-policy-keys, pose-doctor-fixtures-exercise-production-path
priority: 0
components: pose-mcp
delivers:
---

# Spec: Every modelled policy reports its unread keys, and the release surface runs under test

## 1. Intent

### Goal
Extend the unread-key finding to every policy the engine models, and run the
doctor's coverage audit over the rest of the CLI, covering what it finds on the
surface that most needs it.

### Business value
Two follow-ups, one shape. Both parents closed a gap in one place and left the
same gap everywhere else.

`pose-doctor-reports-unread-policy-keys` recovered, as a finding, what dropping
`DisallowUnknownFields` gave up: a key the engine does not read is ignored, so a
misspelling takes the default and the setting silently does nothing. It did that
for the review policy, because that is where the decoder had to be relaxed. The
exposure is identical in every other policy — and identical is the point, since
an operator who learns the finding exists for one file has no way to know it
does not exist for the next.

`pose-doctor-fixtures-exercise-production-path` measured which code the suite
executes and found three unreached behavioural branches in a single command. Run
over the rest of the CLI the same measurement found something larger: the entire
release command surface executed zero statements. Nothing had ever run `release
plan`, `check`, `notes`, `record`, `status` or `open-next` outside a real
release — including the rule that evidence must name the commit the tag points
at, which is what keeps a release record honest.

That is the surface this month's two release failures came from.

### Constraints
- The finding must be clean on a fresh install, or it teaches operators to
  ignore it.
- Known keys must be derived from the structs. A restated list drifts the first
  time a field is added, and then reports a real key as unknown — a failure
  indistinguishable from the misspelling the check exists to catch.

### Non-goals
- Giving `changelog.json` and `dor.json` named policy types. Both are read
  through anonymous structs local to their commands, and both would report true
  findings the engine itself causes; that is its own change.
- Covering every uncovered command surface. This covers the release surface and
  records what the audit found elsewhere.

---

## 2. Requirements

### Functional
- R1: Every policy file the engine models shall report keys it does not read, as
  a doctor finding, with the keys derived from the structs that read it.
- R2: A file read by more than one struct shall be held to the union, so a key
  one reader models is never reported as unread.
- R3: A key the convention marks as documentation rather than configuration
  shall not be reported.
- R4: A policy the engine reads from a fallback location shall be examined
  where the engine reads it, not only at the current path.
- R5: A shipped policy shall be either held to its keys or recorded, in writing,
  as having no type to derive them from.
- R6: The release command surface shall execute under test — every command and
  helper, not only the ones a release happens to reach.

### Non-functional
- The findings are clean on a fresh install.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_closeout.go` — the derivation helper
- `pose-mcp/internal/cli/policy_keys.go` — which policies are held to their keys
- `pose-mcp/internal/cli/doctor.go` — one loop in place of one block
- `pose-mcp/internal/cli/release_surface_coverage_test.go` — R6

### Artifacts
- created: .pose/specs/2026-09-09-pose-policy-keys-and-release-surface-coverage.md
- created: .pose/changelogs/unreleased/pose-policy-keys-and-release-surface-coverage.md
- created: pose-mcp/internal/cli/policy_keys.go
- created: pose-mcp/internal/cli/policy_keys_test.go
- created: pose-mcp/internal/cli/doctor_policy_keys_test.go
- created: pose-mcp/internal/cli/release_surface_coverage_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: .pose/specs/2026-09-09-pose-doctor-reports-unread-policy-keys.md
- modified: .pose/specs/2026-09-09-pose-doctor-fixtures-exercise-production-path.md
- modified: .pose/specs/2026-09-07-pose-review-plan-producible-evidence-classes.md

### Technical risks
- Seven findings where there was one is seven chances to be noisy. Each was run
  against a fresh install and reports ok; the annotation exemption is what keeps
  it that way.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Derive known keys from any set of policy structs (R1, R2)
- [x] Increment 2: One table, seven policies, one loop in the doctor (R1, R3, R4)
- [x] Increment 3: Shipped policies are wired or exempted in writing (R5)
- [x] Increment 4: The release surface under test (R6)

### Validation
- [x] Each new finding shown firing, and silent on a fresh install
- [x] The release evidence rule shown failing when removed

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: three more policies were named by the follow-up; seven have a type.
- Decision: wire every policy with a named struct, and exempt the two without
  one in writing, with a test that fails if a shipped policy is neither.
- Rationale: wiring exactly the three named would be right the day it was
  written and silently wrong afterwards — the shape that has now been wrong
  three times in this repository, most recently within one CI run.

### Decision 2
- Date: 2026-09-09
- Context: `changelog.json` ships `categories`, which its only reader does not
  model; `dor.json` ships none of the keys `readiness.go` looks for.
- Decision: leave both exempt rather than wire them.
- Rationale: wiring them would make `pose doctor` warn, on every install, about
  keys the engine itself ships. The finding would be true and the remedy would
  be the engine's, not the operator's. Recorded as a follow-up.

### Decision 3
- Date: 2026-09-09
- Context: the audit found 36 functions at zero coverage across the CLI.
- Decision: cover the release surface and record the rest.
- Rationale: it is the largest single uncovered surface, it is the one whose
  first execution has always been a real release, and it is the one that failed
  in production twice this month. Covering all 36 in one change would be a
  change nobody can review.

---

## 6. Validation

### Strategy
Every new finding is shown firing on a misspelling and silent on a fresh
install; the release rule most worth having is shown failing when removed.

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
- Date: 2026-09-09
- Environment: local, Go 1.26
- Notes: on a fresh `pose install`, all seven `*.policy-keys` findings report
  ok. Injecting one misspelled key into each of the six new files produces
  exactly one warn each, naming the key — `_comment`, present in three of them,
  is not reported. Removing the commit comparison from `cmdReleaseRecord` fails
  `TestReleaseRecordRefusesEvidenceThatDoesNotMatchTheTag`. Package coverage for
  `internal/cli` went from 69.9% to 71.2%, and `release_lifecycle.go` from 0 of
  20 functions executed to 20 of 20.

### Results summary
- Successes: R1 through R6 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <policyKeyChecks holds review, delivery, artifacts, capabilities, docs, release and state to keys taken from their own structs by PolicyKnownKeys; TestDoctorReportsUnreadKeysInEveryModelledPolicy fires one warn per file>
- R2 [satisfied] <capabilities.json passes both capabilityPolicy and capabilityTriggerPolicy; TestDoctorAcceptsCapabilityKeysReadByEitherConsumer fails if either reader's keys are missing>
- R3 [satisfied] <PolicyKeyIsAnnotation exempts the `_`-prefixed keys the shipped policies carry; TestDoctorDoesNotReportTheAnnotationKey>
- R4 [satisfied] <the release check also reads .pose/release-policy.json, where LoadReleasePolicy falls back; TestDoctorReadsTheReleasePolicyWhereTheEngineDoes>
- R5 [satisfied] <TestEveryShippedPolicyIsHeldToItsKeysOrExempted reads the shipped policy set from the embedded scaffold and fails on any file that is neither wired nor exempted, and on an exemption for a file no longer shipped>
- R6 [satisfied] <release_lifecycle.go went from 0 of 20 functions executed to 20 of 20; the tests go through cmdRelease, the production dispatch, against a prepared and tagged fixture>

### Known gaps
- The audit found 36 functions at zero coverage in `internal/cli`. This covers
  the release surface; `cmdReviewCheck`, `cmdHistoryCheck`, `cmdDocsSync`,
  `cmdDocsReview`, `cmdContributeSubmit`, `cmdRoadmapCheck` and the spec-readiness
  helpers in `check.go` remain unexecuted.
- `changelog.json` and `dor.json` are not held to their keys.

---

## 7. Final Report

### Summary
Seven policies now report the keys they carry and the engine does not read, and
the release command surface runs under test for the first time.

### Follow-ups

- [done] Give changelog.json and dor.json named policy types, and reconcile what they ship with what is read. Done in `pose-changelog-and-dor-policy-types`: `categories` now governs the valid set in all three places it was written out, dor.json ships an explicit empty `adopted_at` saying the gate is opt-in, and both are held to their keys — the exemption list is empty — owner:unowned crit:medium review:2026-12-09
- [open] Cover the remaining command surfaces the audit found at zero: review-check, history-check, docs-sync, docs-review, contribute-submit, roadmap-check and the spec-readiness helpers in check.go — owner:unowned crit:medium review:2026-12-09
