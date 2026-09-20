---
slug: check-strict-verdict-names-its-mode
status: draft
created_at: 2026-09-20
completed_at:
supersedes:
depends_on:
priority: 2
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: `check` states the mode it actually ran in

## 1. Intent

### Goal

`pose check --strict` that finds only non-escalating warnings prints
`SUCCESS — (tolerant mode) with N warning(s)` and exits 0. The run was strict; the
verdict names the wrong mode.

### Business value

The verdict line is what a reviewer, a CI log and a spec's execution log quote. A
strict run reporting itself as tolerant invites two opposite errors: recording a
strict pass that was never claimed, or re-running the gate believing `--strict` was
dropped. This was measured during the closeout of `pose-abm-progressive-review`,
where the first reading of the label was in fact wrong — the flag is honored — and
the correction cost a round of source reading that the label should have saved.

### Constraints

Do not change which findings escalate. `--strict` already turns every `failOrWarn`
finding into an error; the remaining warnings are warnings by design in both modes.
This is a reporting defect, not a gate defect, and the fix must not convert it into
one by making design-warnings fatal.

### Non-goals

Reducing the ten historical warnings this repository currently carries. Changing
the exit code of a warning-only run.

## 2. Requirements

- R1: A warning-only verdict names the mode the run used, for `--strict`,
  `--tolerant` and the default, in both locales.
- R2: The escalation behaviour is unchanged: errors fail, `failOrWarn` findings
  escalate under `--strict`, and a warning-only run still exits 0.
- R3: A test pins the verdict text per mode, so the label cannot drift back.

## 3. Technical Plan

`cmdCheckWithLocale` builds the warning-only verdict with a literal
`(tolerant mode)` while the zero-warning verdict interpolates `mode`. Interpolate
`mode` in both, in `POSE.md`'s locale pair as well, and cover the three modes.

### Artifacts

- created: .pose/specs/2026-09-20-check-strict-verdict-names-its-mode.md

Remaining artifacts are declared when the fix is implemented.

### Rollout and reversal

A message change in one command. Reverting is reverting the string and its test.

## 4. Tasks

- [ ] Reproduce the mislabelled verdict in a test before changing the message.
- [ ] Interpolate the mode in the warning-only verdict, in both locales.
- [ ] Confirm escalation and exit codes are untouched.

## 5. Decisions

Recorded as its own spec rather than as a follow-up on
`pose-abm-progressive-review`: that spec is closed and its sealed review is bound to
its body, so amending it to carry an unrelated defect would stale a valid closeout
to record something that was never in its scope.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Verdict text per mode | `go test ./internal/cli -run CheckVerdictMode -count=1` | strict, tolerant and default runs each name their own mode |
| Escalation unchanged | `go test ./internal/cli -run Check -count=1` | errors still fail; warning-only run still exits 0 |

### Execution log

2026-09-20: observed while verifying the closeout of `pose-abm-progressive-review`
with the candidate binary: `check --strict` exited 0 and printed
`Result: SUCCESS — (tolerant mode) with 10 warning(s)`, with zero errors. Source
reading confirmed `--strict` is parsed and passed to the checker, so the escalation
is intact and only the label is wrong. Not fixed in that spec's change set.

## 7. Final Report

Not implemented yet.
