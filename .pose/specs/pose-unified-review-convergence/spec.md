---
slug: pose-unified-review-convergence
status: in-progress
created_at: 2026-08-14
completed_at:
supersedes:
depends_on:
priority: 1
components: cli, pose-mcp, scaffold
delivers: capability:unified-review-convergence, contract:review-auto-attest
---

# Spec: pose-unified-review-convergence

> Single POSE spec template. Fill the relevant sections; remove the ones that
> don't apply. Keep the order: Intent → Requirements → Technical Plan →
> Tasks → Decisions → Validation → Final Report.
>
> **Lifecycle:** update `status` as you go (`draft` → `in-progress` → `done`).
> On completion, run the closeout flow (skill `pose-spec-closeout`): set
> `status: done`, fill `completed_at` and disposition every follow-up.

---

## 1. Intent

### Goal
Unify the POSE review architecture into a single canonical track based on immutable `ReviewBundles` and `ReviewAttestations`, deprecating legacy Markdown review records, and introduce `pose review auto-attest` for seamless evidence binding in subagents and CI.

### Business value
Eliminates confusion between dual review paths, removes fragile manual tool disposition flags during attestation, and enables completely automated, deterministic review verification for subagents and CI pipelines.

### Constraints
- Must remain deterministic, offline-capable, and backward-compatible for existing historical reviews.
- Strict schema validation for bundles and attestations must remain enforced.

### Non-goals
- Online/remote third-party orchestration dependencies (remains strictly offline/local-first).

---

## 2. Requirements

### Functional
- R1: When a review bundle is sealed, `pose review auto-attest <bundle-id>` shall automatically inspect the effective plan, extract matching evidence from `.pose/results/delivery-validation.json` and verification reports, and record a valid `ReviewAttestation`.
- R2: The review policy and CLI shall establish `ReviewBundle` + `ReviewAttestation` as the single canonical review track, deprecating legacy `pose review record`.
- R3: The review verification gate (`pose review verify <scope>`) shall validate bundle attestations as the sole completion criteria for all POSE scopes.
- R4: The review skill (`.agents/skills/pose-review/SKILL.md`) shall guide subagents to execute `pose review bundle <scope> --seal` followed by `pose review auto-attest <bundle-id>`.

### Non-functional
- Fast execution (< 500ms) for auto-attestation and verification.
- Clear error messages when required evidence is missing from validation artifacts.

### Security
- Reviewer identity and evidence hashes must remain tamper-proof and cryptographically checked against Git-observed scope digests.

### Compatibility
- Existing historical review attestations and bundles remain valid.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/`
- `pose-mcp/internal/pose/`
- `.agents/skills/pose-review/`
- `.pose/policy/review.json`
- `POSE.md` and documentation

### Artifacts
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: .agents/skills/pose-review/SKILL.md
- modified: .pose/policy/review.json

### Delivery targets
- capability:unified-review-convergence module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/review_closeout.go
- contract:review-auto-attest module:pose-mcp profile:api-contract entrypoint:pose-mcp/internal/cli/review_closeout.go

---

## 4. Tasks

### Planning
- [x] Create ADR and define unified review architecture
- [x] Author technical spec and requirements

### Implementation
- [x] Implement `pose review auto-attest` in `review_closeout.go`
- [x] Update review policy to make ReviewBundles the sole canonical track
- [x] Update skills and documentation for subagent review workflows

### Validation
- [x] Unit & integration tests in `review_bundle_test.go` and `review_closeout_test.go`
- [x] Full matrix strict validation with `pose validate --strict`

---

## 5. Decisions

### Decision 1
- Date: 2026-08-14
- Context: Deprecating dual review tracks (legacy markdown vs review bundle)
- Options considered: Keep both indefinitely vs unify into ReviewBundles
- Decision: Unify into ReviewBundles as single track
- Rationale: Eliminates non-convergent invalidation loops and confusion
- Consequences: Cleaner CLI and automated subagent workflows

---

## 6. Validation

### Strategy
Full test suite in `pose-mcp`, CLI E2E tests for `pose review auto-attest`, and `pose validate --strict`.

---

## 7. Final Report

### Follow-ups
- [open] Monitor adoption of unified review auto-attest across subagents and CI pipelines (owner:@pose-maintainers crit:low review:2026-10-15)

