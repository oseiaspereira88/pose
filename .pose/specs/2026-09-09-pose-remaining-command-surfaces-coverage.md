---
slug: pose-remaining-command-surfaces-coverage
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-policy-keys-and-release-surface-coverage
priority: 0
components: pose-mcp
delivers:
---

# Spec: The command surfaces the audit found at zero

## 1. Intent

### Goal
Bring the six commands and the spec-readiness helpers the coverage audit named
off zero, through the dispatch a user reaches them by.

### Business value
The audit run in `pose-policy-keys-and-release-surface-coverage` found 36
functions in `internal/cli` that the suite never executed. The release surface
was the largest and was covered there; this is the rest of what it named:
`review-check`, `history-check`, `docs-sync`, `docs-review`, `contribute
submit`, `roadmap-check`, and the helpers the Definition of Ready is built from.

A command nobody executes is a command whose refusals are assumptions. Every one
of these has a usage path that returns an exit code a script acts on, and none
had been observed returning it.

### Constraints
- `contribute submit` reaches `gh issue create`. Everything before that is
  covered; the submission itself is an outward action and is not exercised.
- `docs-sync push` talks to a Conductor and is likewise left alone.

### Non-goals
- The rest of the 36. `assess`, the install and self-update helpers,
  `runServeMCP` and the smaller helpers are not what the follow-up named.

---

## 2. Requirements

### Functional
- R1: Each of the six commands shall execute under test through `Main`, so one
  that stops being reachable fails here rather than in a terminal.
- R2: Each shall be observed refusing an invocation it cannot serve, with the
  exit code a caller branches on.
- R3: `specReady` and the helpers it is built from shall be exercised on the
  cases that decide a verdict: an empty section, prose with no numbered
  criterion, a commented-out section, and a configured task type.
- R4: Where a command needs a precondition, the fixture shall provide it rather
  than the assertion being weakened to whatever the command happened to answer.

### Non-functional
- No test performs an outward action.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/remaining_surfaces_coverage_test.go` — all of it

### Artifacts
- created: .pose/specs/2026-09-09-pose-remaining-command-surfaces-coverage.md
- renamed: .pose/changelogs/unreleased/pose-remaining-command-surfaces-coverage.md -> .pose/changelogs/v4.0.0/pose-remaining-command-surfaces-coverage.md
- created: pose-mcp/internal/cli/remaining_surfaces_coverage_test.go
- modified: .pose/specs/2026-09-09-pose-policy-keys-and-release-surface-coverage.md

### Technical risks
- Three of the first fixtures asserted against what the command answered rather
  than what it does: `history-check` needs a history directory, `docs-sync
  export` needs a manifest, and `roadmap-check` needs delivery profiles and a
  spec before it reaches a criterion at all. Each was found by the assertion
  failing, which is the fixture doing its job — a weaker assertion would have
  passed against a command that never ran its gate.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The six commands through Main, refusals included (R1, R2)
- [x] Increment 2: The readiness helpers on the cases that decide (R3)
- [x] Increment 3: Fixtures give each command its precondition (R4)

### Validation
- [x] Every named function off zero, measured

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: `contribute submit` runs `gh issue create`, and `docs-sync push`
  writes to a Conductor.
- Decision: cover everything up to the outward call and stop.
- Rationale: a test that opens an issue on every run is not a test. The
  refusals before it — no argument, a contribution that does not exist — are
  where the exit codes are decided, and the assertion also requires the refusal
  to happen before the command announces a submission.

### Decision 2
- Date: 2026-09-09
- Context: `roadmap-check` on the first fixture reported no criteria and
  passed, in both modes.
- Decision: give the fixture the delivery profiles and spec the graph builder
  needs, and record the early return as a follow-up rather than change it here.
- Rationale: `buildCurrentDeliveryGraph` returns before it loads any roadmap
  when the profile index is absent, so the command reports an empty criteria
  list and exit 0. That is a gate reading as met because the path that would
  fail was never reached — worth its own change, not a silent fix inside a
  coverage spec.

---

## 6. Validation

### Strategy
Execute each command through `Main` and require the exit code and the message,
then measure that every named function left zero.

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
- Notes: `internal/cli` coverage goes from 71.2% to 72.9%, and none of the
  twelve named functions reports 0.0% any more. Three fixtures had to be
  corrected against what the commands actually require, each found by an
  assertion failing rather than by reading the code.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <every case runs through runCLI, which calls Main in the fixture directory; cmdReviewCheck, cmdHistoryCheck, cmdDocsSync, cmdDocsReview, cmdContributeSubmit and cmdRoadmapCheck all report coverage>
- R2 [satisfied] <TestTheseCommandsRefuseAnInvocationTheyCannotServe requires exit 2 and the message for six usage paths; the review-check, roadmap-check and contribute-submit tests each add a refusal of their own>
- R3 [satisfied] <TestSpecReadyReadsTheSectionsTheDoRRequires covers a complete spec, an empty section, prose with no numbered criterion and a configured task type; TestSpecSectionsIgnoreWhatIsCommentedOut, TestSectionFilledSkipsScaffoldProse, TestFrontmatterBodyHandlesBothShapes and TestParseRoadmapReadsMilestonesAndFallsBackToTheFilename cover the helpers>
- R4 [satisfied] <the history fixture creates the history directory, the docs-sync fixture writes a manifest, and the roadmap fixture writes delivery profiles and a spec; each also keeps the missing-precondition case as its own assertion>

### Known gaps
- `roadmap-check` reports no criteria and exits 0 when the delivery profile
  index is absent. The test pins the behaviour with the profiles present; the
  early return is recorded below.
- The remaining zero-coverage functions the audit found are untouched.

---

## 7. Final Report

### Summary
The six commands and the readiness helpers execute under test, and their
refusals are observed rather than assumed.

### Follow-ups

- [done] `roadmap-check` returns before loading any roadmap when `.pose/indexes/validation-matrix.json` is absent, so it reports zero cut criteria and exits 0 — a gate reading as met because the path that would fail is never reached; roadmap criteria include check: and manual-review: refs that need no delivery profile. Fixed in `pose-roadmap-check-reaches-its-gate`: neither an absent profile index nor an absent specs directory skips the evaluation, so every criterion gets the verdict it deserves and an unresolved delivery ref blocks as it should. (owner:unowned crit:high review:2026-11-09)
- [open] Cover what the audit found and this spec did not: the assess trio, the install and self-update helpers, runServeMCP, parseSinceDate, inferSuggestDomain, siblingSpecStatus and hasOpenChild (owner:unowned crit:low review:2027-01-09)
