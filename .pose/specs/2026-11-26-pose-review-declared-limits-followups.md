---
slug: pose-review-declared-limits-followups
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-26
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
---

# Spec: pose-review-declared-limits-followups

## 1. Intent

### Goal
Close the follow-ups left by `pose-review-declared-threat-model`: the remaining Markdown the accepted-limits reader misreads, a limit named by a non-approving draft, the manual `--finding` path, the published schemas, and the text output of the checks.

### Business value
The spec closed on 2026-11-26 after eleven review rounds; its Final Report lists two follow-ups and its approving attestation five more (round 11, N1–N5).

### Constraints
Each item keeps the behaviour its origin spec delivered; a fix that changes an answer states it as a decision.

### Non-goals
Reopening the decisions of the origin spec.

## 2. Requirements

### Functional
- R1: An indented continuation line (one to three spaces) that starts with `# ` shall not end the section and cut the limit: it is either read as the limit's text or reported, and the limit is not sealed in part (round 11, N1).
- R2: An unterminated `<!--` anywhere after a closed comment, and a `<!--` written inside inline code in a limit, shall be reported, and no limit shall be sealed cut by them; a comment inside inline code shall keep its text (rounds 9 N4, 10 N4, 11 N2).
- R3: `pose review attest --finding` shall accept a limit for a `within-limit` finding, or the CLI reference shall stop listing it there (round 11, N3).
- R4: A non-approving draft whose follow-up names a limit the bundle did not seal shall be refused or recorded without `within-limit`, so no attestation names a limit that does not exist (round 11, N4).
- R5: The published review plan and bundle schemas shall declare `accepted_limits` (and `limitations`), with the schema drift test covering them (round 11, N5).
- R6: `pose review-check` and `pose closeout-check` shall show within-limit findings in their text output, as their JSON does (rounds 9 N5, 10 N5).

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp`: `accepted_limits.go`, `review_conclusions.go`, `review_bundle.go`, the attest CLI, the published schemas, `review-check`/`closeout-check` output, `docs-site/docs/cli.md`.

### Artifacts
Declared when the spec starts.

### Delivery targets
Declared when the spec starts.

### API/contract changes
To be detailed when the spec starts.

### Data/storage changes
To be detailed when the spec starts.

### Technical risks
To be detailed when the spec starts.

## 4. Tasks

### Planning
- [ ] Confirm each requirement against the current engine before starting.

### Implementation
- [ ] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log

### Results summary

### Requirement trace

### Known gaps

## 7. Final Report

### Delivered scope

### Files and modules changed

### Validation executed

### Residual risks

### Follow-ups
Origin: `pose-review-declared-threat-model` (done 2026-11-26).
