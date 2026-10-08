---
slug: pose-attention-lists-only-open-obligations
status: in-progress
created_at: 2026-10-08
completed_at:
supersedes:
depends_on: pose-state-attention
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers:
---

# Spec: Attention lists only what is still open

## 1. Intent

### Goal

Keep satisfied, waived and cancelled obligations out of Attention's `for_actor` and `gates`, the lists that say what waits on a person or on a gate, as they are already out of `blocking`.

### Business value

The Harne8 side of the agency-readiness pilot (`xref:proj.harne8/spec:harne8-adopt-agency-readiness`) answered a decision through the trusted channel: the request was satisfied and closeout was no longer restricted, yet Attention still listed it under `for_actor`, and Harne8's "Atenção" tab showed it as depending on the person. An answered request presented as pending is the friction Attention exists to remove.

### Constraints

The full obligation report is unchanged: a closed obligation is still listed there with its satisfaction. Only the grouping changes, by the same predicate `Restricts` already uses.

### Non-goals

Changing residual debt, coverage or the obligation report itself.

## 2. Requirements

### Functional

- R1: When an obligation is satisfied, waived or cancelled, Attention shall leave it out of `for_actor` and `gates`, with or without an actor query.
- R2: When an obligation is pending or invalidated, Attention shall keep listing it under `for_actor` or `gates` as before, and `blocking` shall be unchanged.

### Non-functional

- None.

### Security

- None.

### Compatibility

- A reader that relied on closed obligations appearing in `for_actor` or `gates` now finds them only in the report.

## 3. Technical Plan

### Affected areas

Attention grouping.

### Artifacts

- created: .pose/specs/2026-10-08-pose-attention-lists-only-open-obligations.md
- modified: pose-mcp/internal/pose/attention.go
- created: pose-mcp/internal/pose/attention_open_only_test.go
- created: .pose/changelogs/unreleased/pose-attention-lists-only-open-obligations.md

### Technical risks

- None beyond the compatibility note.

## 5. Decisions

### Decision D1
- Date: 2026-10-08
- Context: `BuildAttention` grouped every non-residual obligation by recipient, regardless of satisfaction, while `blocking` used `Restricts`.
- Options considered: (a) filter by the same open predicate as `Restricts`; (b) add a separate list of closed obligations.
- Decision: (a).
- Rationale: the lists answer "what waits", and the report already holds the closed ones.
- Consequences: an invalidated answer reappears in `for_actor`, as it is open again.

## 6. Validation

### Strategy

A test over a report with every satisfaction state, with and without an actor query; it failed before the change, listing the answered, waived and cancelled requests and the met gate.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose -run TestAttentionListsOnlyOpenObligations`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestAttentionListsOnlyOpenObligations check:test evidence:unit
- R2 [satisfied] test:TestAttentionListsOnlyOpenObligations check:test evidence:unit

### Known gaps

## 7. Final Report

### Delivered scope

`BuildAttention` lists under `for_actor` and `gates` only pending or invalidated obligations; closed ones stay in the report. Found by the Harne8 agency-readiness rehearsal.

### Residual risks

Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
