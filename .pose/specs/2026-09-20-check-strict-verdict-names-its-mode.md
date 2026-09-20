---
slug: check-strict-verdict-names-its-mode
status: done
created_at: 2026-09-20
completed_at: 2026-09-20
supersedes:
depends_on:
priority: 2
components: pose-mcp
task_type: bugfix
delivers: surface:check-verdict-mode
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
- created: .pose/changelogs/unreleased/check-strict-verdict-names-its-mode.md
- created: pose-mcp/internal/cli/check_verdict_mode_test.go
- modified: pose-mcp/internal/cli/check.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- surface:check-verdict-mode module:pose-mcp/internal/cli profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

A message change in one command. Reverting is reverting the string and its test.

## 4. Tasks

- [x] Reproduce the mislabelled verdict in a test before changing the message.
- [x] Interpolate the mode in the warning-only verdict, in both locales.
- [x] Confirm escalation and exit codes are untouched.

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

2026-09-20, implemented. The warning-only verdict interpolates `mode`, the way the
zero-warning verdict already did, in both locales. The escalation path above it is
untouched.

Both fixture generators were measured rather than assumed, and the first guesses at
each were wrong:

- A done spec with no changelog fragment produces the warning only when its
  `completed_at` is on or after the changelog adoption date that `pose install`
  stamps. A fixture dated in the past was exempt and produced no warning at all, so
  the first version of this test skipped itself and proved nothing.
- The escalation case needs a `failOrWarn` finding, not merely a malformed file. An
  invalid spec status and an unterminated frontmatter are both plain warnings; a
  malformed `validation-matrix.json` is a warning under `--tolerant` and an error
  under `--strict`, which is the contract this fix must not touch. An earlier
  measurement of its exit code was also wrong, because `$?` was reading a pipe.

That last point corrects the original finding's own first reading: `--strict` is
honored. Only the verdict text was wrong.

### Closeout

2026-09-20 UTC. Bundle `rvb-298789dca8a96ade`, eleven evidence items, under the five
sealed contracts. Attestation `rva-53ba1cff8683bbb5`, `agent:claude-opus-5`,
approved; `review-check` fresh and approved; `closeout-check` terminal.
`surface-check --strict` exits 0 with one inferred-coverage warning kept, for the
module-granularity reason recorded on the specs closed before it.

### Requirement trace

- R1 [satisfied] surface:check-verdict-mode evidence:integration
  check:check-verdict-mode-integration test:TestCheckVerdictModeNamesTheRunItWas —
  strict, tolerant and default runs each name their own mode, and none reports itself
  as another
- R2 [satisfied] surface:check-verdict-mode evidence:integration
  check:check-verdict-mode-integration test:TestCheckVerdictModeKeepsEscalation — a
  warning-only run exits 0 in every mode; a failOrWarn finding is an error under
  `--strict` with exit 1 and a warning under `--tolerant` with exit 0
- R3 [satisfied] surface:check-verdict-mode evidence:integration
  check:check-verdict-mode-integration test:TestCheckVerdictModeNamesTheRunItWas —
  the verdict text is pinned per mode, so the label cannot drift back

## 7. Final Report

### Scope delivered

A warning-only verdict names the run it was. The escalation contract is unchanged and
pinned, so the label fix cannot become a gate change.

### Residual risks

The test asserts the parenthesised mode rather than the whole sentence, so a rewording
of the verdict would not fail it. That is deliberate: pinning the full string would
make every translation edit a test failure, and the mode name is the part that was
wrong.

### Follow-ups

None.
