---
slug: pose-structural-facts-stay-in-their-commits
status: done
created_at: 2026-10-08
completed_at: 2026-10-08
supersedes:
depends_on: pose-validation-check-additions-are-not-material
remediates:
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:structural-attributed-content
---

# Spec: Structural facts stay in their commits

## 1. Intent

### Goal

Compare each structural subject path over what the scope's own commits changed, instead of the subject's `base..head`, and mark the facts that come from a commit other specs also claim, so the reviewer is asked to confirm them.

### Business value

Origin: the R8 decision (adjust) of the Harne8 ABM field pilot (`xref:proj.harne8/spec:pose-abm-field-pilot`). An independent reviewer found 7 false positives in 80 sampled structural alerts, all from scope mixing in the attributed subject, and the maintainer chose to resolve the follow-ups before structural judgments leave opt-in. Recomputing the 125 bundles of the pilot showed that the range also made the obligation depend on what other specs committed in between: 14 specs were charged for check-only matrix additions the engine exempts, and one spec's own edit was hidden.

### Constraints

Content identity only: no commit or provider ref enters the structural input digest. No sealed bundle is reread differently. The engine still does not judge whether a fact belongs to a scope semantically.

### Non-goals

Revisiting the check-addition exemption; detecting unrelated content inside the scope's own commits; a gate on multi-trailer commits.

## 2. Requirements

### Functional

- R1: When a subject path was changed by the scope's attributed commits and by other commits in between, the structural delta shall compare only the scope's own changes to it.
- R2: When a fact comes from a commit whose `POSE-Spec:` trailers also name another spec, the fact and the plan's material fact shall name that spec in `shared_with`, and the plan shall warn the reviewer to map it or answer its mapping `not-applicable` with the reason.
- R3: The attribution shall name blobs, never commits, and leave the implementation digest unchanged; a subject without it shall be compared over its range as before.
- R4: The sealed bundle shall carry the attribution, and the governance replay shall count `shared_commit_facts`.

### Non-functional

- One bounded Git call per sealed subject; no Git call when nothing is attributed.

### Security

- Blob reads keep the existing byte bound and do not follow symlinks.

### Compatibility

- Bundles sealed before this have no `attribution` and keep their range reading. New bundles get facts whose digests describe the scope's own change, so their display ids differ from the range reading.

## 3. Technical Plan

### Affected areas

Review bundle subject, design delta, review plan structure, governance replay, schemas and the manual.

### Artifacts

- created: .pose/specs/2026-10-08-pose-structural-facts-stay-in-their-commits.md
- created: .pose/adr/2026-10-08-structural-facts-compare-attributed-content.md
- created: .pose/changelogs/unreleased/pose-structural-facts-stay-in-their-commits.md
- created: pose-mcp/internal/pose/review_path_attribution.go
- created: pose-mcp/internal/pose/review_path_attribution_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/design_delta.go
- modified: pose-mcp/internal/pose/review_structure.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/governance_replay.go
- modified: pose-mcp/schemas/v1/review-bundle.schema.json
- modified: pose-mcp/schemas/v1/review-plan.schema.json
- modified: POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:structural-attributed-content module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A path changed in several separate runs yields one comparison per run; on the pilot universe 24 paths had more than one run and no fact was duplicated.

## 4. Tasks

### Implementation
- [x] Record per-path blob segments and shared trailers when the subject is sealed
- [x] Compare segments in the design delta and carry `shared_with` to the plan, its warning and the replay
- [x] Schemas, manual and ADR

### Validation
- [x] Tests that fail on the range reading and pass on the attributed one
- [x] Measure on the pilot universe

## 5. Decisions

### Decision D1
- Date: 2026-10-08
- Context: the pilot's false positives come from scope mixing; the range also makes the obligation depend on interleaving.
- Options considered: (a) mark shared commits only; (b) compare attributed content and mark shared commits; (c) also stop exempting check additions.
- Decision: (b), chosen by the maintainer after the measurement in the ADR.
- Rationale: the obligation must follow the scope's own work; the mark asks for judgment where commits cannot be split; the exemption was a measured decision and the adjudication criterion did not separate knowing a fact from owing it an answer.
- Consequences: see `.pose/adr/2026-10-08-structural-facts-compare-attributed-content.md`.

## 6. Validation

### Strategy

Unit tests over a Git fixture where another spec's commit lands between the scope's commits on the same `go.mod`, and a batch commit carries both trailers. The contrast assertion compares the same subject without attribution and requires the other spec's dependency to appear, so the test fails on the range reading. A read-only measurement over the 125 newest bundles of the pilot (Harne8 and pose-dist) recomputed each with attribution: 254 charged facts become 241, 12 are marked shared (3 of them are 3 of the 7 adjudicated false positives), 14 check-only matrix additions stop being charged, and 1 hidden own edit becomes material.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose -run 'TestABMStructuralAttribution|TestGovernanceReplayCountsSharedCommitFacts'`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] capability:structural-attributed-content check:abm-progressive-review-integration evidence:integration test:TestABMStructuralAttributionKeepsOtherSpecsOut
- R2 [satisfied] capability:structural-attributed-content check:abm-progressive-review-integration evidence:integration test:TestABMStructuralAttributionMarksSharedCommits
- R3 [satisfied] capability:structural-attributed-content check:abm-progressive-review-integration evidence:integration test:TestABMStructuralAttributionIsContentIdentity
- R4 [satisfied] capability:structural-attributed-content check:abm-retrospective-replay-integration evidence:integration test:TestGovernanceReplayCountsSharedCommitFacts

### Known gaps

Mixing inside the scope's own commits and declared ranges that contain a release commit stay with the reviewer.

## 7. Final Report

### Delivered scope

A sealed subject records, per path, the blobs at the ends of each uninterrupted run of the scope's own commits, and which other specs share a commit of that run. The structural delta compares those runs instead of `base..head`; facts from a shared run carry `shared_with` into the plan, whose warning asks the reviewer to map the fact or answer it `not-applicable` with the reason, and the replay counts them. On the pilot universe, 254 charged facts become 241, 12 are marked shared, 14 check-only matrix additions stop being charged under the existing exemption, and one hidden own edit becomes material.

### Residual risks

- Four of the seven adjudicated false positives remain: unrelated content inside the scope's own commit, or a declared range that is a release commit. The reviewer answers them.
- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups

- [open] Evaluate an advisory, never a gate, for a commit carrying several `POSE-Spec:` trailers or a declared range that contains a release commit, once bundles sealed with attribution show how often shared facts occur (owner:@pose-maintainers crit:low review:2026-11-08)
