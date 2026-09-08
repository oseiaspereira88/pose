---
slug: pose-public-claims-onboarding
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-public-claims-contract
priority: 1
components: pose-mcp
delivers:
---

# Spec: A way in to the public claims contract

## 1. Intent

### Goal
Make `pose public-claims` usable by an instance that has never declared a
contract: tell the operator what is missing and how to start, and ship a
template so the contract stops being knowledge that exists only in this
repository.

### Business value
`public-claims` is listed in `pose help` among the deterministic gates, beside
`check`, `validate` and `lint-spec`. On any instance that has not hand-authored
an undocumented file, running it produces:

```
pose public-claims: open .pose/public/claims.json: no such file or directory
```

Nothing in that names the real situation. It does not say the contract is
opt-in, that no scaffold creates it, what it would do, or how to begin. The
operator is left with a filesystem error from a command the tool advertises as
first-class.

The gap is not the message alone. `.pose/public/` is absent from
`scaffold/dist/.pose/`, so no instance has ever received the shape of the file;
the only example is this repository's own `claims.json`, written by hand. An
adopting project that wants the guarantee has to reverse-engineer the schema
from a source tree.

That matters more for this gate than for most, because of what it guards. It is
the mechanism that stops a public surface contradicting a released fact — the
failure that has already recurred twice in an adopting repository, where the
same stale version reached nineteen places across eighteen files before anyone
noticed. A gate against that failure which no one can turn on protects nothing.

### Constraints
- An absent contract still fails. Passing would let a pipeline report claims as
  verified when nothing was checked, which is worse than the error being
  unhelpful.
- The template must not encourage discovery. Surfaces are declared on purpose:
  scanning for version-shaped strings would flag changelogs and release notes,
  which are supposed to name old versions.

### Non-goals
- A `pose new-public-claims` scaffold command. The template plus a copy line is
  enough to answer "how do I start", and a command would need its own contract,
  help entry and tests for a file most instances write once.
- Enabling the gate by default. It is opt-in by design; a project decides which
  surfaces make claims.

---

## 2. Requirements

### Functional
- R1: With no contract present, the command shall report that the instance
  declares none, that it is opt-in, and how to start one.
- R2: The command shall keep exiting non-zero in that case, so a pipeline never
  reads an unrun gate as a passing one.
- R3: A template shall be delivered to every instance as machinery, and the
  message shall point at it by path.

### Non-functional
- The existing public-claims suite passes unchanged.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/publicclaims.go` — the absent-contract path
- `.pose/templates/public-claims.json` — the template, delivered by machinery

### Artifacts
- created: .pose/specs/2026-09-08-pose-public-claims-onboarding.md
- created: .pose/changelogs/unreleased/pose-public-claims-onboarding.md
- created: .pose/templates/public-claims.json
- created: pose-mcp/internal/scaffold/dist/.pose/templates/public-claims.json
- modified: pose-mcp/internal/cli/publicclaims.go
- modified: pose-mcp/internal/cli/publicclaims_test.go

### Technical risks
- The template's `surfaces` entries are examples, and a project that copies them
  without editing declares surfaces it may not have. The message says to copy
  and the template says to edit; a wrong path is reported by the gate itself
  rather than passing silently.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Explain an absent contract instead of reporting a stat error (R1, R2)
- [x] Increment 2: Ship a template and point the message at it (R3)

### Validation
- [x] The message is asserted, and the assertion fails without the change

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: whether an absent contract should still fail. An existing test
  asserted exit 2, so the failure was a deliberate choice by whoever wrote it.
- Options considered: (a) exit 0, treating "not configured" as nothing to
  verify; (b) a distinct exit code for "cannot run" versus usage error; (c) keep
  the code, fix the message.
- Decision: (c).
- Rationale: (a) lets a pipeline that wired the gate report claims as verified
  when nothing was checked, which is the failure the gate exists to prevent.
  (b) is defensible but overturns an existing decision for a small gain and
  risks breaking anything keying on the code. What actually confused an operator
  was the message, and that is what this changes.
- Consequences: an instance without the contract still fails the gate, now with
  a message that says why and what to do.

### Decision 2
- Date: 2026-09-08
- Context: where the template lives.
- Decision: `.pose/templates/`, which is a machinery root.
- Rationale: machinery is delivered to every instance on update, so the template
  arrives without anyone knowing to look for it — which is the whole problem
  being solved. The generator that mirrors pose-dist into the embedded scaffold
  carries it to new instances too, and its test fails if the two diverge.

---

## 6. Validation

### Strategy
Assert the message an operator actually sees, and confirm the assertion fails
against the previous behaviour.

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
- Date: 2026-09-08
- Environment: local, Go 1.26
- Notes: all eight packages pass, and the pre-existing
  `TestPublicClaimsFailsWithoutContractOrVersionSource` passes unchanged, which
  is the evidence that the exit contract was preserved. Disabling the new branch
  reproduces the original output verbatim —
  `open .../claims.json: no such file or directory` — and the new test names
  each thing the message must carry. `go generate ./internal/scaffold` synced
  102 files so the embedded scaffold carries the template.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <publicclaims.go — the os.ErrNotExist branch states that the instance declares none, that it is opt-in, why surfaces are declared, and the copy line; TestPublicClaimsExplainsAnAbsentContract asserts each>
- R2 [satisfied] <the branch returns 2, and the pre-existing failure test passes unchanged>
- R3 [satisfied] <.pose/templates/public-claims.json is under a machinery root and mirrored into the embedded scaffold by go generate; the message names it by path and the test asserts the path appears>

### Known gaps
- The template documents `current-only` and `none` by example in its notes. If
  the contract gains a third mode, the template is a second place to update and
  nothing enforces that it is.

---

## 7. Final Report

### Follow-ups

- [open] Have the template's version_claims modes derive from the contract schema so a new mode cannot be documented in only one place — owner:unowned crit:low review:2026-12-08
