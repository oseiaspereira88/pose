---
slug: pose-public-claims-publication-provenance
status: draft
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: 
priority: 0
components: pose-mcp
task_type: bugfix
---

# Spec: Separate candidate, prepared and published versions in public claims

## 1. Intent

### Goal

Stop presenting a locally read engine version as a published release, while keeping the
offline candidate-consistency gate.

### Business value

`pose public-claims` reads `engine_version` from `compatibility.json` and labels it
`released_version`. At `392aaa5a` the 6.3.0 manifest is prepared and untagged while the
latest published release is 6.2.0. The gate must work before a release exists, so its
name, not its existence, is the defect.

### Constraints

No network call in the default check. Publication is proven only by retained release
evidence (manifest with publication record, tag evidence) produced by the existing
release lifecycle.

Program source: backlog items POSE-02 (P0, wave 0) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F04; sources
E01, E02, E03, E27, E28). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Querying GitHub on every check; publishing 6.3.0 as part of this spec.

### Anti-mechanization guardrail

Preservar o gate de consistência sem transformar metadado local em fato externo.

## 2. Requirements

### Functional

- R1: The public-claims result shall report `candidate_version`, `prepared_version` and `published_version` as separate fields, each with its provenance source.
- R2: A prepared manifest without publication evidence shall report the version as prepared and `published_version` as unproven, never as published.
- R3: Candidate consistency (claims, compatibility metadata and engine version agree) shall remain verifiable offline before publication.
- R4: `published_version` shall require retained publication evidence from the release lifecycle; its absence renders as `unknown`/`unproven` with the reason.
- R5: The legacy `released_version` field shall keep its current value for one minor with a documented deprecation, and the terminal label shall change immediately to say what was verified.

### Non-functional

- Deterministic output for the same tree.

### Security

- Local metadata is untrusted input for publication claims; it can only support candidate consistency.

### Compatibility

- JSON consumers of `released_version` get a deprecation note and a new field; the meaning of an existing field is not changed silently.

## 3. Technical Plan

### Affected areas

Public claims gate, its JSON contract and template, release lifecycle evidence lookup.

### Artifacts

- created: .pose/specs/2026-10-04-pose-public-claims-publication-provenance.md
- modified: pose-mcp/internal/cli/publicclaims.go
- modified: pose-mcp/internal/cli/publicclaims_test.go
- modified: .pose/templates/public-claims.json
- modified: .pose/public/claims.json
- modified: pose-mcp/internal/cli/release_manifests.go
- created: .pose/changelogs/unreleased/pose-public-claims-publication-provenance.md

The paths above are the planned surface at 2026-10-04; reconcile them at activation
against the tree, as the ABM specs did, before the first implementation commit.

### Technical risks

- Consumers that parse `released_version` as published; covered by keeping the field and documenting the transition.

## 4. Tasks

### Planning
- [ ] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-public-claims-publication-provenance`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Fixture with a prepared, untagged manifest: today the output says released; after the
fix it says prepared and unproven. A second fixture with retained publication evidence
reports published.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./...`
- Scope: engine packages touched by this spec
- Expected: pass, including the new negative tests

#### Lint
- Command: `cd pose-mcp && go vet ./...`
- Scope: pose-mcp
- Expected: no findings

#### Security / Contract
- Command: `pose lint-spec pose-public-claims-publication-provenance --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
