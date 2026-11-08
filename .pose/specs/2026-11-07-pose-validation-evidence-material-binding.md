---
slug: pose-validation-evidence-material-binding
status: in-progress        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-07
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components:          # optional, inline comma-separated list: modules/components touched (e.g. mcp-server, cli) — used by pose_list_specs' `components` filter
task_type:           # optional: feature | bugfix | refactor — a key of taskTypes in .pose/policy/dor.json; selects the sections the Definition of Ready requires (default: defaultTaskType)
surface: standard
delivers: capability:validation-evidence-material-binding
---

# Spec: pose-validation-evidence-material-binding

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
Bind validation evidence to the material it validated, so a commit that changes only derived administrative records no longer forces a full rerun of the matrix.

### Business value
B34 closed with a strict matrix of about 354 s run twice, at a functional checkpoint and again at an editorial one, because the evidence is bound to the change set's commits and an index, result or spec-status commit with the spec trailer moves them. Follow-up opened at B34 closeout (criticality medium, review 2026-12-06).

### Constraints
Evidence for an open spec must still be rejected when any material the checks exercised changed, when the checkout is incomplete, or when the engine or validation matrix changed. Existing results without the new field keep today's rule. The administrative path classes are a closed list owned by the engine, not by an instance.

### Non-goals
Caching individual check outputs, skipping checks by heuristic, weakening review-bundle seals, or changing the rule for specs already `done`. The typed `federated-roadmap` error follow-up is a separate spec.

---

## 2. Requirements

> Definition of Ready (entry gate): before `status: in-progress`, functional
> requirements must have **acceptance criteria with stable IDs** (`- R<N>: ...`).
> Published IDs are never renumbered; a removed criterion is marked as
> withdrawn. Verify with `pose lint-spec <slug> --ready-check`.
>
> Optional EARS form: `- R1: When <trigger>, the <system> shall <behavior>.`
> Verify an opted-in spec with `pose lint-spec <slug> --ears`.

### Functional
- R1: A validation run shall record a `material_digest` per scope, computed over the head state (path, mode and blob) of every path the change set names, minus administrative records that are still regular files there, plus the validation matrix and, in the scoped digest, the claims and the engine version.
- R2: Evidence shall remain current for an open spec while its recomputed `material_digest` equals the recorded one, even when the change set head moved through commits whose changed paths all fall in administrative classes.
- R3: Evidence shall be rejected as stale when any material path, the matrix or the engine changed, when the head tree cannot be read, or when the checkout is incomplete. A present material binding governs and never falls through to the commit-bound rules. The refusal names the scope; listing the first differing path needs a stored manifest and is a follow-up.
- R4: A symlink, submodule or directory in an administrative class, a path that left a class, the destination of a rename, and any file in a class that is not a data format, shall be material.
- R5: Results without `material_provenance` shall be judged exactly as before, and the commit-bound digests shall not change representation.
- R6: Review bundle sealing, `review verify`, `closeout-check` and `close` shall consume the same domain verdict, with no per-channel copy of the rule.
- R8: A validation run shall record a material binding only when the working copy equals the committed material in every path outside the administrative classes, untracked files included; otherwise it records the binding as unavailable for every scope.
- R9: Review bundle sealing shall judge material-bound evidence against the current tree, recomputing each change set's digest from its stored inputs (names and rename destinations) instead of trusting the digest the persisted index recorded.
- R10: The governance read view shall retain at most 40,000 observations (it was 25,000), so that the closeout plan of a corpus of this size, which observes 26,051 entries, is judged `current` instead of `unknown`; the metadata cap rises with it, from 8 MiB to 16 MiB, because 40,000 retained observations no longer fit in 8 MiB; the input-byte cap is unchanged.
- R7: A delegated review shall run its diagnostics in the sealed checkout without republishing derived records into it: the review process carries `POSE_SEALED_CHECKOUT` set to the checkout's path, and `refreshDeliveryIndex` leaves that checkout alone and no other root.

### Non-functional
- A closeout whose administrative commits follow a passing run shall not rerun checks; the decision is computed from git objects, in time proportional to the changed paths.

### Security
- The administrative class list is closed and path-exact; a path outside it never counts as administrative, and symlinks or renames into a class are material.

### Compatibility
- Additive result field, classified minor. Needs an ADR for the evidence-binding contract before implementation.

---

## 3. Technical Plan

### Affected areas
- pose-mcp/internal/pose: delivery_surface.go (`deliveryEvidenceCurrent`), delivery_validation_run.go, delivery_integrity_compact.go, review_bundle.go; cli validate_results.go.

### Artifacts
<!-- Declare exact project-relative source-tree paths: created, modified,
     renamed (old -> new), removed, or one `none: <reason>` entry. -->
- created: pose-mcp/internal/pose/material_binding.go
- created: pose-mcp/internal/pose/material_binding_test.go
- created: pose-mcp/internal/cli/material_engine.go
- created: pose-mcp/internal/mcpserver/material_engine.go
- created: .pose/adr/2026-11-07-validation-evidence-binds-to-validated-material.md
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/delivery_integrity_compact.go
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/pose/delivery_validation_run.go
- modified: pose-mcp/internal/pose/spec_facts.go
- created: pose-mcp/internal/cli/sealed_checkout_refresh_test.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/delegation_refusal.go
- created: pose-mcp/internal/cli/validate_material_worktree_test.go
- modified: pose-mcp/internal/pose/delivery_integrity_cache.go
- modified: pose-mcp/internal/pose/delivery_integrity_cache_test.go
- modified: pose-mcp/internal/pose/governance_read_view.go
- modified: pose-mcp/internal/pose/governance_read_view_probe_test.go
- modified: pose-mcp/internal/pose/governance_read_view_integration_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/cli/validate_results.go
- modified: pose-mcp/internal/cli/change_set_root_test.go
- modified: pose-mcp/internal/cli/governance_operation_performance_test.go

### Delivery targets

- capability:validation-evidence-material-binding module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/delivery_surface.go

### API/contract changes
- 

### Data/storage changes
- 

### Technical risks
- 

---

## 4. Tasks

### Planning
- [ ] Confirm intent
- [ ] Identify affected modules

### Implementation
- [ ] Implement incrementally

### Validation
- [ ] Run the mandatory checks

---

## 5. Decisions

> ADR: `.pose/adr/2026-11-07-validation-evidence-binds-to-validated-material.md`.
>
> Optional section. Use it when the implementation involves trade-offs or
> alternatives. For a material basis, keep structured `Assumption A<N>` and
> `Decision D<N>` nodes in this section and run
> `pose lint-spec <slug> --design-check` (advisory).
> Only for a material premise, `Valid scope:` names the context it holds in and
> `Stale trigger: doc:<path>@<pin>` pins the content it was judged against; a
> change to that content projects a `premise-stale` judgment obligation. There is
> no calendar expiry, and trivial premises need neither field.
> Likewise, a material decision may add `Expected effect:` and `Falsifier check:
> check:<module>/<check>`; a failed indexed result of that check projects a
> reconsideration candidate, never a verdict. Small fixes need neither.

```markdown
### Assumption A1
- Claim: a material premise
- Status: unverified
- Evidence: contract:example
- Scope: version or bounded context
- Affects: R1

### Decision D1
- Basis: R1, A1
- Minimal option: keep the current flow
- Selected option: keep the current flow
- Rationale: concrete reason
- Consequences: durable effect
- Falsifier: condition that would invalidate the choice
```

### Decision <N>
- Date:
- Context:
- Options considered:
- Decision:
- Rationale:
- Consequences:

---

## 6. Validation

### Strategy
Unit tests on the digest, the administrative classification and the verdict; git-backed tests on change-set resolution (administrative commits, symlinks, renames, shallow checkouts, untrailered commits); working-copy guard tests; capacity tests for the read view; the full strict matrix on a clean clone; and an independent delegated review by a different vendor.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1`
- Scope: pose-mcp
- Expected: all packages pass

#### Lint
- Command: `gofmt -l` on the touched files, `go vet ./internal/pose ./internal/cli`
- Scope: pose-mcp
- Expected: no output

#### Typecheck
- Command: `go build ./...`
- Scope: pose-mcp
- Expected: builds

#### Build
- Command: `go build ./cmd/pose`
- Scope: pose-mcp
- Expected: builds

#### Security / Contract
- Command: `pose validate --strict --jobs 4` on a clean clone
- Scope: whole matrix
- Expected: 189/189 pass

### Execution log
- Date: 2026-11-07
- Environment: clean clone, `LC_ALL=C LANGUAGE=C` (a pt_BR locale makes seven integration checks fail with git exit 128 on non-git fixtures)
- Notes: the implementation went through many independent delegated review passes by Codex gpt-6.1-sol (a different vendor); each pass's defects were corrected in the commits above and their run records were discarded with the scratch clones they ran in. The retained record is the approving review attached to the sealed bundle.

### Results summary
- Successes: strict matrix 189/189; unit, git-backed and integration tests of the three packages.
- Failures: none at closeout.
- Warnings: artifact-check reports orphan warnings for pose-mcp test files that no spec claims; they predate this spec.

### Requirement trace
- R1 [satisfied] test:TestMaterialSetDigestIgnoresAdministrativeRecordsOnly test:TestChangeSetMaterialDigestIgnoresAdministrativeCommits check:pose-mcp/test
- R2 [satisfied] test:TestMaterialEvidenceSurvivesHeadMoveButNotMaterialChange test:TestChangeSetMaterialDigestIgnoresAdministrativeCommits
- R3 [satisfied] test:TestMaterialBindingGovernsAndNeverFallsThrough test:TestUnreadableMaterialNeverBindsAndIsRecordedAsUnavailable test:TestChangeSetMaterialDigestIsUnreadableInAShallowCheckout test:TestChangeSetMaterialDigestReadsTheCheckoutTree
- R4 [satisfied] test:TestChangeSetMaterialDigestSeesSymlinksAndRanges test:TestChangeSetMaterialDigestKeepsAMovedFileMaterial test:TestCodeInsideAnAdministrativeClassIsMaterial
- R5 [satisfied] test:TestMaterialDigestDoesNotChangeTheCommitBoundDigests test:TestLegacyDigestsIgnoreTheMaterialFields
- R6 [satisfied] test:TestEvidenceIsJudgedAgainstTheCurrentMaterial test:TestFactResultKeySeparatesMaterialBindings check:delivery-integrity-index-compaction-integration
- R7 [satisfied] test:TestRefreshDeliveryIndexLeavesASealedCheckoutAlone
- R8 [satisfied] test:TestMaterialWorktreeCleanSeesUncommittedMaterial test:TestMaterialTrackedUnchangedIgnoresUntrackedBuildProducts
- R9 [satisfied] test:TestEvidenceIsJudgedAgainstTheCurrentMaterial
- R10 [satisfied] test:TestGovernanceReadViewProbeRecordCapacityAndOverflow test:TestGovernanceReadViewOperationSharesRecordCapacityAndAuditsAcrossRoots

### Known gaps
- The first differing path of a stale binding is not named (R3 narrowed).
- The limits listed in the ADR (ignored files, a data file copied or moved into a class and edited uncommitted, concurrent writers, sparse checkouts, a run whose graph could not be collected) are accepted.

---

## 7. Final Report

### Delivered scope
Validation evidence binds to the material it validated: each trailer change set carries a `material_digest` over the head state (path, mode, blob) of the paths it names, minus administrative records; a validation run records `material_provenance` per scope; evidence stays current while the digest holds, so commits that only touch indexes, results, state, assessments or review records no longer force the matrix to rerun. Unreadable or shallow material, an uncommitted material change, a legacy recorded set and a mismatched binding all make evidence stale instead of falling back to the commit-bound rules. Review bundle sealing recomputes the digest from the current tree, and a delegated review no longer lets `artifact-check` republish the delivery index inside its sealed checkout. The governance read view cap is recalibrated from 25,000 to 40,000 observations (metadata cap 8 to 16 MiB) because the closeout plan of this corpus observes 26,051 entries. Left out: naming the first differing path, an immutable execution snapshot, copy-provenance tracking.

### Files and modules changed
- pose-mcp/internal/pose (material_binding.go, delivery_surface.go, delivery_integrity*.go, delivery_validation_run.go, spec_facts.go, review_bundle.go, review_dispatch.go, delegation_refusal.go, governance_read_view.go)
- pose-mcp/internal/cli (artifact_integrity.go, validate.go, validate_results.go, index.go, material_engine.go)
- pose-mcp/internal/mcpserver (material_engine.go)
- ADR validation-evidence-binds-to-validated-material; POSE.md and its scaffold and pt-BR copies (read view capacity)

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head
- Result: 189/189 pass; independent delegated review by Codex gpt-6.1-sol, its conclusion recorded by the engine with `pose review conclude`.

### Residual risks
- Accepted in the ADR: ignored files, a data file copied or moved into a class and edited uncommitted, a concurrent writer that changes and restores material during a run, sparse or skip-worktree checkouts, a run whose delivery graph could not be collected.
- Spec facts judge a material binding from the indexed tree entries and report freshness as unknown for it.
- The closeout plan of this corpus is within about 35% of the new read view cap; the corpus keeps growing.

### Follow-ups

- [open] Name the first differing material path when a binding is stale; that needs a stored manifest per scope. (owner:@pose-maintainers crit:low review:2027-01-07)
- [open] Validate an immutable execution snapshot, which would close the concurrent-writer, sparse-checkout and ignored-file limits of the material binding. (owner:@pose-maintainers crit:low review:2027-01-07)
- [open] Module discovery fails on non-git fixtures under a non-English locale: `git ls-files` exits 128 and the localized "not a repository" message is not recognized, so seven integration checks fail with `LANG=pt_BR` and pass with `LC_ALL=C`. (owner:@pose-maintainers crit:medium review:2026-12-07)
- [open] Make the governance read view cap scale with the corpus (or archive settled review records) instead of a constant that closeout outgrows every few weeks. (owner:@pose-maintainers crit:medium review:2026-12-07)
