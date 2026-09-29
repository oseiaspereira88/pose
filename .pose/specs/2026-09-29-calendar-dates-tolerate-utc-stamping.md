---
slug: calendar-dates-tolerate-utc-stamping
status: done
created_at: 2026-09-29
completed_at: 2026-09-29
supersedes:
depends_on:
priority: 1
components: pose-mcp, docs
task_type: bugfix
delivers: surface:lifecycle-date-order
---

# Spec: A local completion date is not earlier than a UTC creation date

## 1. Intent

### Goal

A spec created and closed on the same local evening, west of UTC, passes the
lifecycle gate. Today `lint-spec` refuses it with `completed_at is earlier than
created_at`.

### Business value

`pose new-spec` stamps `created_at` with the UTC date. The quickstart and
`POSE.md` tell the reader to fill `completed_at` by hand, and a person writes
their local date. At 23:33 in UTC-3 the two are 2026-09-29 and 2026-09-28, so the
documented first governed loop fails for anyone in the Americas every evening.
It surfaced when the quickstart and launch demo scripts, both CI gates, failed
locally in that window and passed in CI, which runs in UTC.

### Constraints

Dates stay bare calendar dates, and POSE keeps stamping them in UTC. A real
regression is still an error: the rule must not become a formality.

### Non-goals

Stamping dates in local time. That would change every stamp the engine writes and
make the same repository record different dates depending on where the command
ran.

## 2. Requirements

- R1: Between two bare dates, a `completed_at` one calendar day before
  `created_at` passes the lifecycle gate. Same-day completions still pass.
- R2: Two or more days earlier is still refused, and so is any regression when
  either value is a full timestamp.
- R3: Knowledge artifacts apply the same tolerance to `last_reviewed_at` against
  `created_at`.
- R4: The frontmatter reference states that POSE stamps dates in UTC and what the
  gate tolerates.

## 3. Technical Plan

`lintOneSpec` records whether each value parsed as a bare date. One helper,
`completedBeforeCreated`, accepts one day of skew when both are bare dates and
compares exactly otherwise. `pose maintenance`'s knowledge check uses it too.
Between a UTC date and any local date the difference is at most one day, so one
day is the whole tolerance.

### Artifacts

- created: .pose/specs/2026-09-29-calendar-dates-tolerate-utc-stamping.md
- created: .pose/changelogs/unreleased/calendar-dates-tolerate-utc-stamping.md
- created: pose-mcp/internal/cli/calendar_date_skew_test.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: docs-site/docs/frontmatter.md
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- surface:lifecycle-date-order module:pose-mcp/internal/cli profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

The gate accepts strictly more than before, and only by one day between bare
dates. Reverting restores the refusal.

## 4. Tasks

- [x] Reproduce the refusal in the local evening window and confirm CI passes in UTC.
- [x] Accept one day of skew between bare dates in the spec gate and the knowledge check.
- [x] Pin the tolerance and its limits in a test that fails without the fix.
- [x] Document the rule in the frontmatter reference.

## 5. Decisions

### Decision D1

- Status: active
- Tolerate the skew in the comparison instead of stamping local dates. Stamping
  locally fixes one person's evening but makes a repository's dates depend on
  where each command ran, and CI would then disagree with laptops. The
  comparison is the one place that must reason about two dates of unknown zone.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Tolerance and its limits | `go test ./internal/cli -run CompletionOneDayBeforeUTCCreation -count=1` (from pose-mcp) | one day between bare dates passes; two days, and instants, are refused |
| Quickstart in the failing window | `bash tests/quickstart/first-governed-loop.sh` at 23:50 UTC-3 | the documented loop completes |
| Demo in the failing window | `bash examples/demo/record.sh --verify` at 23:50 UTC-3 | blocked, then legitimately closed |

### Execution log

2026-09-29 02:33 UTC (23:33 UTC-3): the quickstart and demo scripts failed
locally with `completed_at is earlier than created_at` and passed under `TZ=UTC`.
`new-spec` and `close` stamp UTC dates; the scripts, like a reader following
the quickstart, write the local date.

2026-09-29, implemented. With the tolerance removed, the new test fails on "local
date one day behind the UTC stamp". With it, the test passes, and at 23:50
UTC-3 the quickstart and demo both complete locally.

### Closeout

2026-09-29 UTC. Full matrix 39/39 into the results path; bundle
`rvb-f904299eb1b7ef43`, 36 evidence items; attestation `rva-0f74dd5a4f85fe3a`,
`agent:claude-opus-5-5`, approved with five explicit judgments. A first
attestation cited the previous bundle's design evidence id, was refused by
`review-check`, and was replaced before closing.

### Requirement trace

- R1 [satisfied] surface:lifecycle-date-order evidence:integration check:calendar-date-skew-integration test:TestCompletionOneDayBeforeUTCCreationIsZoneSkew — same-day and one-day-behind bare dates pass the gate
- R2 [satisfied] surface:lifecycle-date-order evidence:integration check:calendar-date-skew-integration test:TestCompletionOneDayBeforeUTCCreationIsZoneSkew — two days earlier, an instant pair and an instant/date pair
  are still refused
- R3 [satisfied] surface:lifecycle-date-order evidence:integration check:calendar-date-skew-integration test:TestCompletionOneDayBeforeUTCCreationIsZoneSkew — `pose maintenance` warns on `last_reviewed_at` through the
  same `completedBeforeCreated` helper
- R4 [satisfied] surface:lifecycle-date-order evidence:integration check:calendar-date-skew-integration test:TestCompletionOneDayBeforeUTCCreationIsZoneSkew — docs-site/docs/frontmatter.md states the UTC stamping and
  the one-day tolerance

## 7. Final Report

### Scope delivered

A spec created and closed on the same local evening west of UTC passes the
lifecycle gate, and the quickstart and demo CI gates no longer depend on the hour
and zone of the machine running them.

### Residual risks

A completion recorded one day early by mistake now passes silently. Accepted: the
gate cannot tell that mistake from the skew, and the skew is the common case.

### Follow-ups

None.
