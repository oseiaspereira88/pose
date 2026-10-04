---
slug: pose-review-attribution-roles
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-review-assurance-disclosure
priority: 0
components: pose-mcp
delivers: surface:review-attribution-roles
task_type: feature
---

# Spec: Version review attribution: prepared, concluded, confirmed and applied

## 1. Intent

### Goal

Add a prospective, versioned attribution block to attestations that records who
prepared, concluded, confirmed and applied a review, how confirmation happened and to
what content it is bound.

### Business value

The 6.3.0 report shows identity alone cannot tell apart: a person wrote the conclusion;
the agent wrote it and a person adopted it; a person authorized a run generically; the
agent wrote and applied everything under an authorized identity. These four cases have
different epistemic value and today read the same.

### Constraints

Roles are optional and may coincide; an absent role stays absent (never copied from
`reviewer`). Old attestations, signatures and digests stay verifiable and are rendered
as `legacy-undifferentiated`. A later clarification is a supplement record, never an
edit of the original.

Program source: backlog items POSE-04 (P0, wave 0) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F01, F02; sources
E01, E07, E16). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Editing the existing ledger; making four roles mandatory on every review; proving that a
declared confirmation happened (that is assurance, handled by verified claims and
trusted channels).

### Anti-mechanization guardrail

Mais campos declarativos não são vendidos como prova de intervenção humana.

## 2. Requirements

### Functional

- R1: An attestation written under the new contract shall carry an `attribution` block with optional `prepared_by`, `concluded_by`, `confirmed_by`, `applied_by` and a `confirmation_mode` of `adopted-conclusions`, `authorized-operation` or `none`.
- R2: A confirmation shall bind to the digest of the exact conclusions and criteria it confirms; changing them invalidates the confirmation for the new content.
- R3: A review prepared and applied by an agent shall keep that attribution even when a person confirms the draft, and the render shall say "prepared by agent, confirmed by person (declared|verified)".
- R4: Authorizing an operation and adopting conclusions shall produce distinguishable records and renders; a generic cycle authorization shall never be promoted to review confirmation.
- R5: `pose review attest` and the MCP equivalent shall accept the roles explicitly and shall refuse to fill a role from `--reviewer` implicitly.
- R6: Attestations without the block shall render `attribution: legacy-undifferentiated`; a supplement record may clarify history by reference without altering the original digest.
- R7: The 6.3.0 cycle attestations shall be a regression fixture: their render must not read as human review.

### Non-functional

- Contract stamped in the bundle registry so verification knows which attribution contract governs a record.

### Security

- `confirmed_by: human:*` written in declared mode renders as a declaration; only the verified path (signed claim bound to the confirmation digest) renders as verified confirmation.

### Compatibility

- Readers accept both shapes; new fields belong to a registered contract version; old binaries refuse new records with a diagnosable error rather than dropping fields.

## 3. Technical Plan

### Affected areas

Attestation schema and writer, review attest CLI/MCP, verify/render, contract registry,
regression fixtures.

### Artifacts

- created: .pose/specs/2026-10-04-pose-review-attribution-roles.md
- created: pose-mcp/internal/pose/review_attribution.go
- created: pose-mcp/internal/pose/review_attribution_test.go
- modified: pose-mcp/internal/pose/review_assurance.go
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/cli/review_attribution.go
- created: pose-mcp/internal/cli/review_attribution_cli_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-review-attribution-roles.md
- created: .pose/review-attribution-supplements/ras-0ca079e2c924c0e8.json
- created: .pose/review-attribution-supplements/ras-21f767fd60315767.json
- created: .pose/review-attribution-supplements/ras-221f26e00b557474.json
- created: .pose/review-attribution-supplements/ras-3eaf568374d8e50a.json
- created: .pose/review-attribution-supplements/ras-4e2ee3a523c62e9a.json
- created: .pose/review-attribution-supplements/ras-6cf659ef89c042dc.json
- created: .pose/review-attribution-supplements/ras-7bc10f2f50911ffc.json
- created: .pose/review-attribution-supplements/ras-805a590f36277320.json
- created: .pose/review-attribution-supplements/ras-88552b7409b34fec.json
- created: .pose/review-attribution-supplements/ras-89fe1c9bd5a6ccd3.json
- created: .pose/review-attribution-supplements/ras-af30e43144313b67.json
- created: .pose/review-attribution-supplements/ras-b1670b461a950223.json
- created: .pose/review-attribution-supplements/ras-be779d9d77d3112e.json
- created: .pose/review-attribution-supplements/ras-d95b251af78afcb9.json
- created: .pose/review-attribution-supplements/ras-f516d63eafabda4f.json
- created: .pose/review-attribution-supplements/ras-fa5875f2b91dc117.json

Reconciled against the tree at activation.

### Delivery targets

- surface:review-attribution-roles module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Field inflation becoming ritual; mitigated by optional roles and by the adversarial corpus case.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-review-attribution-roles`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

### Assumption A1
- Claim: attestation records can gain an optional block without breaking signature verification of old records
- Status: unverified
- Evidence: test:TestReviewAttestationLegacyDigestStable
- Scope: attestation schema v1 readers in pose-mcp
- Affects: R6

### Decision D1
- Basis: R1, R3, R4, A1
- Minimal option: document the ambiguity and rename `reviewer` in renders only
- Selected option: optional versioned attribution block bound to a confirmation digest
- Rationale: renaming cannot distinguish the four observed cases; a bound confirmation can
- Consequences: one more registered contract; renders change for new records only
- Falsifier: real records show the roles are always identical, making the block pure ceremony

### Decision D2
- Basis: R1, R6
- Minimal option: version the attribution by its own block schema_version
- Selected option: version the attribution by its own block schema_version
- Rationale: a registry contract is sealed into every new bundle and held as a requirement; attribution is optional by design, so registering it would either make it mandatory or register a contract that requires nothing
- Consequences: the non-functional note about stamping the contract is replaced by the block version; readers accept both shapes and an absent block reads as legacy-undifferentiated
- Falsifier: a consumer needs to know, from the bundle alone, whether attribution was expected for it


## 6. Validation

### Strategy

Fixtures for the four cases of the analysis (section 8.4), the 6.3.0 regression set,
digest stability of legacy records, and a negative test that `--reviewer human:x` does
not populate `confirmed_by`.

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
- Command: `pose lint-spec pose-review-attribution-roles --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
