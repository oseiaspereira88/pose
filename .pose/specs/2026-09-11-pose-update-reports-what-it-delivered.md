---
slug: pose-update-reports-what-it-delivered
status: in-progress
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-fragment-error-clarity
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: An update that delivered its files says so

## 1. Intent

### Goal
When `pose install`, and `pose update --force` through it, has delivered the
machinery and then finds the instance's own state invalid, it shall report that
failure through the final gate — which says whether it predates the run and that
nothing is rolled back — instead of stopping at the index step.

### Business value
Reported against `pose upgrade`: every machinery tree had been delivered, and the
run still ended in `falha na atualização de scaffolds` because one changelog
fragment was corrupt. Reproduced on 5.0.1: `update --force` puts the machinery
back, then `pose index` fails on the fragment, `install` returns on the spot,
and `update` prints "scaffold refresh failed" with exit 1.

Two things were wrong. The failure named the refresh, which had succeeded. And
the explanations `install` already has — "this target already failed the same
gate before this run touched anything", and "files were already written; this
command does not roll back" — live on the final gate, which the early return
never reached. An operator was told their upgrade failed, with no way to know it
had not.

### Constraints
- The exit code stays non-zero: the instance fails its strict gate, and scripts
  rely on that.

### Non-goals
- Making install transactional; the recovery notice covers that and rollback
  stays out of scope, as `pose-install-gate-failure-recovery-notice` decided.
- Validating instance state before delivery: it would block an upgrade on debt
  unrelated to it.

---

## 2. Requirements

### Functional
- R1: An index failure after delivery shall not end the run before the final
  gate.
- R2: When the gate then fails, the run shall report whether the failure
  predates it and that the delivered files are not rolled back.
- R3: When the gate passes but the index failed, the run shall still exit
  non-zero and say that delivery happened and the index did not.
- R4: `pose update` shall not describe a refresh that wrote its files as failed.

### Non-functional
- Exit codes are unchanged for every case that already failed.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/install.go` — the index step before the final gate
- `pose-mcp/internal/cli/maintenance.go` — `update --force`'s failure message

### Artifacts
- created: .pose/specs/2026-09-11-pose-update-reports-what-it-delivered.md
- renamed: .pose/changelogs/unreleased/pose-update-reports-what-it-delivered.md -> .pose/changelogs/v5.0.2/pose-update-reports-what-it-delivered.md
- created: pose-mcp/internal/cli/update_reports_delivered_state_test.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: .pose/specs/2026-08-10-pose-fragment-error-clarity.md

### Technical risks
- A run whose index fails for a reason this run caused now reaches the gate
  instead of stopping. The gate's message distinguishes pre-existing failures
  from new ones, so a new failure is not reported as debt.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Continue to the final gate after an index failure (R1, R2)
- [x] Increment 2: Exit non-zero when only the index failed (R3)
- [x] Increment 3: Reword update's message (R4)

### Validation
- [x] The test fails against 5.0.1 with the reported symptoms

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: the follow-up asked whether the index step should warn rather than
  fail, or run before delivery. Three options were put to the owner: validate
  before delivering (A), let the final gate report an index failure (B), or B
  with exit 0 when the failure predates the run (C).
- Decision: B, chosen by the owner.
- Rationale: A blocks upgrades on unrelated debt, which is what the report
  objected to. C would turn a failing strict gate into success for scripts. B
  keeps the verdict and makes the message true — the gate already knows how to
  say what predates the run and what cannot be undone.

---

## 6. Validation

### Strategy
Reproduce the reported case on a real instance, and require delivery, a
non-zero exit and an honest message.

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
- Date: 2026-09-11
- Environment: local, Go 1.26
- Notes: reproduced with the 5.0.1 engine on an installed instance holding a
  corrupt fragment: the four machinery trees delivered, then `pose index:
  release lifecycle: malformed release fragment …/broken.md` and `pose update:
  scaffold refresh failed`, exit 1. Against that code the new test fails on the
  missing pre-existing and no-rollback messages and on the "scaffold refresh
  failed" line; with the fix it passes, as do the install, update and upgrade
  tests.

### Results summary
- Successes: R1, R2, R4 verified; R3 implemented.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestUpdateForceWithACorruptFragmentReportsWhatWasDelivered removes a workflow, corrupts a fragment, runs update --force, and requires the workflow back>
- R2 [satisfied] <the same test requires "already failed the same gate before this run" and "does not roll back" in stderr, and a non-zero exit>
- R3 [satisfied] <install returns 1 with a delivery-happened message when the gate passes after an index failure; not exercised by a test — no fixture makes the index fail while the strict gate passes>
- R4 [satisfied] <the same test requires stderr not to say "scaffold refresh failed">

### Known gaps
- R3's branch has no test: every index failure constructed here also fails the
  strict gate.

---

## 7. Final Report

### Summary
An upgrade that delivered everything no longer reports itself as failed; the
failure it does report is the instance's, and says whether it was already there.

### Follow-ups
