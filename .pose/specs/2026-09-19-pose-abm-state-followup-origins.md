---
slug: pose-abm-state-followup-origins
status: in-progress
created_at: 2026-09-19
completed_at:
components: pose-mcp
task_type: bugfix
delivers: surface:state-followup-origins
---

# Spec: Preserve follow-up origins in project state

## 1. Intent

Repair the producer behind Harne8 instance-health R1. `pose state` reports
two invalid `spec:docs:...` pointers. Aggregated follow-ups include document
and capability origins; the renderer incorrectly prefixes every origin with
`spec:`. Do not weaken the resolver or rewrite historical evidence.

## 2. Requirements

- R1: Render document origins as existing `doc:` pointers, preserving paths.
- R2: Preserve ordinary spec pointers and capability labels without inventing
  spec or component identities.
- R3: Regenerate state through its producer; missing documents must remain
  diagnostic failures rather than being silently accepted.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-state-followup-origins.md
- modified: pose-mcp/internal/cli/state_providers.go
- modified: pose-mcp/internal/cli/docs_review_test.go

### Implementation approach

Use the existing origin discriminator only at the rendering boundary.
Preserve aggregate follow-up JSON and resolver contracts. No dependencies,
policy changes or metadata discovery writes. Consulted
knowledge:module-metadata-discovery-invalidates-review-provenance to avoid
unrelated index refresh invalidating reviews. Revert the isolated fix to roll back.

### Delivery targets
- surface:state-followup-origins module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

## 4. Tasks

- [x] Reproduce failure in Harne8 and identify the producer.
- [x] Cover document origin rendering and missing-document rejection.
- [x] Run module checks and regenerate consumer state.
- [ ] Review and close through POSE after evidence is available.

## 5. Decisions

Correct origin rendering, not slug validation: accepting `docs:` as a spec
would hide a real type mismatch. Capability origins stay plain labels because
the resolver has no capability-pointer contract.

## 6. Validation

Before implementation: require a regression in the existing docs follow-up
fixture, state tests preserving ordinary spec origins, and resolver rejection
after removing a referenced document. Run `go test ./...`, `go vet ./...`, and
`go build ./...`; regenerate Harne8 state using the built binary and compare
diagnostics. No release or rollout claim follows from this local verification.

### Requirement trace
- R1 [satisfied] <document origin rendering and resolution regression> evidence:unit
- R2 [satisfied] <state tests preserve spec references; direct review confirms capability labels are not promoted to pointers> evidence:unit
- R3 [satisfied] <missing-document rejection regression and Harne8 native state refresh verified> evidence:unit

## 7. Final Report

2026-09-19: `go test ./... -count=1`, vet and build passed. Native
`validate --strict --module pose-mcp` passed 6/6 outside the sandbox, which
otherwise prevents HTTP test listeners. Harne8 `state refresh`, `state` and
`check --strict` passed with the rebuilt binary. Initial negative-test extension
exposed that the fixture declared but did not create its document; creating it
before checking existence and removing it afterward made the regression explicit.

Implementation verified locally; governed review/closeout still pending. No
release installation or full instance-health claim.

### Follow-ups
None within this bounded producer fix. Harne8 adoption and broader diagnostics
remain requirements of its existing instance-health spec, not claims of this delivery.
