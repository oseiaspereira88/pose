---
slug: pose-delegated-verdict-destinations
status: in-progress
created_at: 2026-10-27
completed_at:
supersedes:
depends_on: pose-delegated-review-conclusions
priority: 3
components: pose-mcp
task_type: feature
delivers: capability:delegated-verdict-destinations
---

# Spec: An adjudication or smoke run records its verdict where its origin expects it

## 1. Intent

### Goal

Give the verdict of a delegated adjudication and of a delegated smoke run a governed destination, and let the engine record it there under the rules a review conclusion follows.

### Business value

R2 of [`pose-delegated-review-conclusions`](2026-10-27-pose-delegated-review-conclusions.md) asks for these verdicts to be recorded "where the origin expects them" — the adjudicated spec's decision record, the smoke target's evidence — and neither destination exists. The maintainer chose on 2026-10-27 to deliver review conclusions first and to define these destinations in a spec of their own.

### Constraints

The same checks as a review conclusion apply: an intact, signed run of a current bundle, a complete typed verdict, idempotence, refusals recorded with their reason. A verdict never becomes an approval of another scope.

### Non-goals

Review conclusions (delivered by `pose-delegated-review-conclusions`); turning the capability on.

## 2. Requirements

### Functional

- R1: An `adjudication` run that passes every check shall record its verdict as a decision record of the adjudicated spec, readable by the spec's lint and closeout, and the adjudicated questions shall carry the verdict's answer for each.
- R2: A `smoke` run that passes every check shall record, for each delivery target it exercised, an observation that the surface checks read as evidence of that target, with the commands it ran and what it observed.
- R3: A verdict that fails a check, is incomplete or does not parse, shall record no governed effect and keep its reason, as a review conclusion does.

## 3. Technical Plan

### Artifacts

- modified: .pose/specs/2026-10-27-pose-delegated-verdict-destinations.md
- created: .pose/starts/pose-delegated-verdict-destinations.json
- created: .pose/changelogs/unreleased/pose-delegated-verdict-destinations.md
- created: pose-mcp/internal/pose/delegated_verdicts.go
- created: pose-mcp/internal/pose/delegated_verdicts_test.go
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/pose/review_rounds.go
- modified: pose-mcp/internal/pose/review_brief.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/surface_check_test.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/review_run_provenance_test.go
- created: pose-mcp/internal/cli/delegated_verdicts_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegated-verdict-destinations module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/review_conclude.go

## 4. Tasks

- [x] Adjudication verdict as a decision record
- [x] Smoke observation as surface evidence
- [x] Refusals and idempotence
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: the engine writes the verdict into the adjudicated spec's Decisions or Validation section
- Selected option: an immutable adjudication record per run in the `adjudications` directory of `.pose`, naming the run and the proof the issuer signed, holding the verdict per question — each decision `D<n>` and requirement `R<n>` the spec records, which the brief of template `adjudication-brief/2` lists and the pose-draft block answers in `questions`; `lint-spec` reads the records of the spec, verifies each against its run, and fails on a question the current bundle's adjudication says does not hold. The maintainer chose this on 2026-10-27
- Rationale: the engine never edits a governed spec, so its change set stays the implementer's; a record verified against its run's proof and draft cannot say what the run did not
- Consequences: a spec's adjudications are read beside it, not in it; a question the current bundle's verified adjudication says does not hold fails `lint-spec` and refuses `close`; an adjudication of an earlier bundle is kept but no longer gates either. Every record is admitted on read under what its conclusion required: its proof signed by the issuer it names, the independence its plan requires, and a verdict answering exactly what was asked (review round 1)
- Falsifier: an adjudication record that verifies while stating a verdict the run's draft did not give

### Decision D2
- Basis: R2
- Minimal option: a record of its own that the surface check gains a reader for
- Selected option: validation results in the format `pose validate` writes, of the new evidence class `smoke`, one required check per delivery target the smoke run exercised, passing when it was observed, in the `results/smoke` directory of `.pose` beside the results path, which `surface-check` reads after verifying each document against its run; a delivery profile may then demand `smoke`. The maintainer chose this on 2026-10-27
- Rationale: the surface check already decides delivery from validation results by class, module and provenance; a smoke observation is one more result, not a new kind of evidence
- Consequences: `smoke` joins the closed vocabulary of evidence classes; no profile demands it until an instance adds it. A smoke result names the target and the spec it observed and is evidence of that target only; its head, time and provenance come from the run, its bundle and the current index, never from the document's bytes, so it is current only while the run's bundle governs its scope (review round 1)
- Falsifier: a smoke result the surface check counts that no verified smoke run observed

### Decision D3
- Basis: R3
- Minimal option: separate commands to record each verdict
- Selected option: `pose review conclude` and the automatic conclusion handle adjudication and smoke runs under the same checks, lock, refusals and idempotence as a review conclusion, plus one more check: the verdict answers exactly what the brief asked — every question, every delivery target, nothing else
- Rationale: one path keeps one bar; a verdict missing a question or naming an unasked target is not what the run was asked
- Consequences: runs briefed with `adjudication-brief/1` and `smoke-brief/1` carry no typed verdict and are refused with that reason, while still verifying with their template
- Falsifier: a recorded verdict that leaves a question or a target unanswered


## 6. Validation

### Strategy

Fixtures for each kind: a passing run records its verdict at its destination; each failed check, an incomplete verdict and a replay record nothing.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: split on 2026-10-27 from R2 of `pose-delegated-review-conclusions`.

### Residual risks

None yet.

### Follow-ups
