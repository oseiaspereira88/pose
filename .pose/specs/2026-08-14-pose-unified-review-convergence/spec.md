---
slug: pose-unified-review-convergence
status: done
created_at: 2026-08-14
completed_at: 2026-08-15
changelog: none
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
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: .agents/skills/pose-review/SKILL.md
- modified: .pose/policy/review.json
- modified: docs-site/docs/ci.md
- modified: pose-mcp/internal/version/version.go

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

### Decision 2
- Date: 2026-08-15
- Context: The implementation of R1-R4 shipped bundled inside commit
  `422907c` (`chore(release): prepare v1.2.0`) instead of a dedicated
  per-spec commit, so this spec was never taken through closeout — it sat
  at `status: in-progress` through the v1.2.0 and v1.2.1 release/verify
  cycles even though the feature was already live and documented in
  `.pose/changelogs/v1.2.0.md`.
- Options considered: (a) close citing the existing commit and correct the
  Artifacts declaration to match its real diff; (b) treat the spec as a
  duplicate of `pose-review-bundle-convergence` and mark it superseded
  without reconciling provenance; (c) leave it open pending a fresh,
  isolated implementation commit that no longer matches what actually
  shipped.
- Decision: (a) — close citing commit `422907c` as the delivering change
  set, `pose-mcp/internal/version/version.go` and `docs-site/docs/ci.md`
  declared as modified (the release version bump the shared commit also
  carried), and `changelog: none` since the user-facing entry already
  exists in `.pose/changelogs/v1.2.0.md`.
- Rationale: The requirements were genuinely implemented and released;
  closing with accurate provenance is more honest than either discarding
  the record (b) or fabricating a commit that never happened (c).
- Consequences: Sealing the review bundle for this change set required
  teaching `reviewBundlePathClass` to recognize `README.md` and
  `compatibility.json` (release-version-bump paths bundled into the same
  commit) instead of failing closed as unclassified — done as a separate,
  independently closed spec: `pose-review-bundle-root-file-classification`.

---

## 6. Validation

### Strategy
Full test suite in `pose-mcp`, CLI E2E tests for `pose review auto-attest`, and `pose validate --strict`.

### Requirement trace
- R1 [satisfied] `TestReviewAutoAttestCLI`, `TestAutoAttestReviewBundle`; shipped commit:422907c; released `.pose/changelogs/v1.2.0.md`.
- R2 [satisfied] `TestReviewRecordDelegatesToBundleAttestationWhenAdopted`; `.pose/policy/review.json` `review_bundles: true` since 2026-08-14; commit:422907c.
- R3 [satisfied] `pose review verify` sole-criteria behavior exercised by this very closeout (`review_verify.state=ready-to-close`/`closed`); commit:422907c.
- R4 [satisfied] `.agents/skills/pose-review/SKILL.md` steps 11-12 (bundle --seal, review auto-attest); commit:422907c.

### Known gaps
- The implementation shipped bundled inside a release-prep commit
  (`422907c`) instead of a dedicated per-spec commit — see Decision 2. This
  closeout is retroactive; it does not re-implement or re-test anything
  beyond what already released in v1.2.0/v1.2.1.

---

## 7. Final Report

### Delivered scope
`pose review auto-attest`, the ReviewBundle/ReviewAttestation track as the
sole canonical review path (legacy `pose review record` delegates to it when
adopted), and updated CLI/skill/policy documentation — all delivered and
released in v1.2.0, carried forward unchanged through v1.2.1. This closeout
retroactively records provenance and review evidence for work that had
already shipped; see Decision 2.

### Files and modules changed
- `pose-mcp/internal/cli/review_closeout.go`, `review_closeout_test.go`
- `pose-mcp/internal/pose/review_bundle.go`, `review_bundle_test.go`,
  `review_closeout.go`
- `.agents/skills/pose-review/SKILL.md` (and pt-BR locale mirror)
- `.pose/policy/review.json`
- `pose-mcp/internal/version/version.go`, `docs-site/docs/ci.md` (the
  shared commit's release version bump, not spec-specific — see Decision 2)
- all commit:422907c

### Validation executed
- `pose artifact-check --spec pose-unified-review-convergence --from 11e7067..422907c --strict`: zero errors.
- `pose validate --strict --json .pose/results/delivery-validation.json`: SUCCESS.
- `pose review bundle spec:pose-unified-review-convergence --seal` + `pose review auto-attest --apply`: sealed and approved.
- `pose review verify spec:pose-unified-review-convergence`: `ready-to-close`, `fresh=true`, `approved=true`.

### Residual risks
- None beyond what v1.2.0/v1.2.1 already carry; no new code executes as a
  result of this closeout.

### Follow-ups
- [open] Monitor adoption of unified review auto-attest across subagents and CI pipelines (owner:@pose-maintainers crit:low review:2026-10-15)

