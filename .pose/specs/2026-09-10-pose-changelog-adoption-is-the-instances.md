---
slug: pose-changelog-adoption-is-the-instances
status: in-progress
created_at: 2026-09-10
completed_at:
supersedes:
depends_on: pose-changelog-and-dor-policy-types
priority: 0
components: pose-mcp
delivers:
task_type: bugfix
---

# Spec: The changelog adoption date belongs to the instance

## 1. Intent

### Goal
Stop shipping this repository's own changelog adoption date to every instance,
and stamp each instance with the day it received the policy.

### Business value
`.pose/policy/changelog.json` was synced byte-for-byte into the scaffold, so
every `pose install` inherited `adopted_at: 2026-08-03` — the day *this*
repository adopted the contract.

For a project starting today that reads as "gate everything", harmless by
accident. For a project migrating into POSE with a history of specs it is not:
every spec completed before 2026-08-03 is silently exempt from needing a
changelog fragment, by a date belonging to someone else. The gate looks adopted
and covers less than the project thinks.

It is the same family as the self-referential delivery roots of issue #17 — a
file whose live content describes this repository being shipped as if it were a
default — in a file that fix did not cover.

### Constraints
- An empty date must not be a malformed policy. `pose check` refused one with
  `adopted_at is required`, so shipping empty alone would break every fresh
  install.

### Non-goals
- Changing what the date means. A done spec completed on or after it still needs
  a fragment.

---

## 2. Requirements

### Functional
- R1: The shipped policy shall carry an empty `adopted_at`, as a neutral
  template rather than a copy of this repository's.
- R2: `pose install` and `pose update` shall stamp the day the instance received
  the policy, when it carries no date.
- R3: An empty date shall mean the contract is not adopted, not that the policy
  is invalid.

### Non-functional
- An instance that already carries a date keeps it; stamping is additive only.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go` — the neutral template
- `pose-mcp/internal/cli/stack_seed.go` — the stamp
- `pose-mcp/internal/cli/check.go` — an empty date is a decision

### Artifacts
- created: .pose/specs/2026-09-10-pose-changelog-adoption-is-the-instances.md
- created: .pose/changelogs/unreleased/pose-changelog-adoption-is-the-instances.md
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/dist/.pose/policy/changelog.json
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/check.go

### Technical risks
- Emptying the date in the scaffold copy alone would have been undone by the
  next `go generate`: the file is synced byte-for-byte, which is the defect.
  It is a neutral template now, listed with the three that already were.

---

## 4. Tasks

### Implementation
- [x] Increment 1: A neutral shipped template (R1)
- [x] Increment 2: The instance is stamped on seed (R2)
- [x] Increment 3: An empty date is not an invalid policy (R3)

### Validation
- [x] A fresh install shown carrying its own date

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the follow-up asked to ship an empty date. Shipping empty alone makes
  `pose check` fail with `adopted_at is required` on every fresh install.
- Decision: ship empty, stamp on seed, and treat empty as unadopted.
- Rationale: an empty date with nothing stamping it would turn the gate off for
  every new project — safer-looking than inheriting a foreign date and worse in
  practice. Stamping is what the review contracts already do, and it is the only
  answer that is true: the engine cannot know when a project adopted the
  contract, but it knows when it first put the policy there.

---

## 6. Validation

### Strategy
Install into an empty repository and require the instance to carry its own date.

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
- Notes: `pose install` into an empty repository logs `policy (changelog
  adoption): adopted_at=2026-09-10` and the instance's policy carries that date,
  not this repository's. The shipped template carries an empty one. This
  repository keeps its own 2026-08-03, which is true of it.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <changelog.json joins SelfReferentialPolicyFiles and NeutralPolicyTemplates ships it with an empty adopted_at; the scaffold copy is regenerated from the template>
- R2 [satisfied] <stampChangelogAdoption writes the day through the raw document, only when the date is absent or empty, and logs what it did; observed on a fresh install>
- R3 [satisfied] <checkChangelogs returns without a finding on an empty date and keeps failing on a malformed file>

### Known gaps
- An instance installed before this change still carries 2026-08-03, and nothing
  reports that the date it is gating by is not its own.

---

## 7. Final Report

### Summary
Each instance gates its changelog by the day it adopted the contract, not by the
day this repository did.

### Follow-ups

- [open] Report a changelog adoption date earlier than the instance's own first spec, so an instance that inherited this repository's date before the fix can see that it is gating by a date that is not its own — owner:unowned crit:low review:2027-03-10
