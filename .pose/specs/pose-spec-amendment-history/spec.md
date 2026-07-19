---
slug: pose-spec-amendment-history
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-requirement-evidence-traceability
priority: 11
---

# Spec: Spec amendment history

## 1. Intent

### Goal
record material intent and acceptance-criteria changes as append-only reviewed amendments.
### Business value
Prevents a spec from being rewritten after evidence without auditable rationale.
### Constraints
- Keep editorial corrections lightweight while preserving material changes.
### Non-goals
- Record every spelling correction as an amendment.

## 2. Requirements

### Functional
- R1: Material requirement additions, withdrawals or semantic changes shall create an amendment event.
- R2: Each event shall identify affected IDs, rationale, author/reviewer and timestamp.
- R3: Closeout shall reject unacknowledged amendments made after referenced evidence.

### Non-functional
- Keep events merge-friendly and deterministic.

### Security
- Use repository identities or pseudonymous IDs; minimize personal data.

### Compatibility
- Published IDs are never renumbered and withdrawn criteria remain addressable.

## 3. Technical Plan

### Affected areas
- Spec lifecycle, linter, indexes, MCP and changelog guidance.

### API/contract changes
- Define material amendment and approval semantics.

### Data/storage changes
- Store append-only amendment entries with a schema version.

### Technical risks
- Over-sensitive detection can burden harmless editorial work.

### Primary references
- [OpenSpec](https://github.com/Fission-AI/OpenSpec)
- [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [OpenSpec](https://github.com/Fission-AI/OpenSpec): no amendment mechanism existed; git history shows diffs but not materiality, rationale or review; context resumed from knowledge:contract-baseline-handoff.

### Implementation
- [x] Define material-change taxonomy and amendment schema: `.pose/specs/<slug>/amendments.jsonl` (schema 1) with `baseline|added|withdrawn|semantic|editorial`, affected R-IDs, rationale, pseudonymous author/reviewer aliases, RFC3339 timestamp and post-change hash per ID (ADR `2026-07-19-append-only-spec-amendment-history`). ([reference](https://github.com/Fission-AI/OpenSpec))
- [x] Detect unrecorded semantic changes to published requirements: normalized-text sha256 fingerprints per R-ID; `lint-spec` on done specs with a history rejects changed/added/removed requirements lacking an acknowledging event; withdrawn IDs stay addressable with empty hashes (never renumbered). ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))
- [x] Render history and test post-evidence mutation cases: `pose amend --list` renders events plus pending acknowledgments; `amend_test.go` covers baseline→pass, post-evidence mutation→fail, acknowledgment→pass, silent removal→fail, withdrawn acknowledgment→pass and input validation; MCP tool `pose_spec_amendments` projects events + unacknowledged findings. ([reference](https://github.com/Fission-AI/OpenSpec))

### Validation
- [x] Run `go test ./pose-mcp/... -run 'Amendment|Requirement'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://github.com/Fission-AI/OpenSpec))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-append-only-spec-amendment-history.md` (Accepted): sibling append-only JSONL over git archaeology and inline sections; hash-based deterministic detection with one-line `editorial` acknowledgment as the lightweight path; opt-in activation per spec via first baseline; additive MCP catalog change.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Amendment|Requirement'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-spec-amendment-history --ready-check`.

### Requirement trace
- R1 [satisfied] material additions/withdrawals/semantic changes create events via pose amend; check:test (TestAmendBaselineAndCloseoutGate, TestAmendRemovalNeedsWithdrawnEvent)
- R2 [satisfied] events carry IDs, rationale, author/reviewer aliases and RFC3339 timestamp, enforced by LoadAmendments; check:test (TestAmendValidation, TestAmendList)
- R3 [satisfied] closeout rejects unacknowledged post-evidence changes; check:test (TestAmendBaselineAndCloseoutGate) report:2026-07-19-standard-validate-native.md

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'Amend' -count=1` — SUCCESS (all five behavior tests).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite, golden catalog regenerated for `pose_spec_amendments`).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-spec-amendment-history --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Append-only amendment contract (`amendments.jsonl`, schema 1), `pose amend`
command (baseline, material changes, editorial acknowledgment, history
rendering), deterministic hash-based closeout gate, MCP projection
(`pose_spec_amendments`), lint metrics, operating-manual documentation and
ADR. Adoption is opt-in per spec via the first baseline.

### Residual risks

- Hash detection flags every rewording; the one-line editorial
  acknowledgment is the accepted mitigation for harmless edits.
- Rationale truthfulness is owned by review, not the gate.

### Follow-ups

- [open] Record baselines for the active-roadmap specs as they enter execution, making the gate effective beyond fixtures. (owner:@pose-maintainers crit:medium review:2026-10-23)
