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
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
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
- R7: A delegated review shall run its diagnostics in the sealed checkout without republishing derived records into it: the review process carries `POSE_SEALED_CHECKOUT`, and `refreshDeliveryIndex` leaves the checkout alone when it is set.

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
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_results.go
- modified: pose-mcp/internal/cli/change_set_root_test.go
- modified: pose-mcp/internal/cli/governance_operation_performance_test.go

### Delivery targets
<!-- When `delivers` is populated, declare the exact same refs here. Profiles
     and evidenceClass requirements come from validation-matrix.json. Leave the
     section empty when the spec delivers nothing typed. One line per target,
     for example: surface:example module:path/to/module profile:web-ui
     entrypoint:path/to/production-entrypoint (as a "- " list item). -->

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
<!-- How the feature will be validated end to end. -->

### Deterministic checks

#### Test
- Command:
- Scope:
- Expected:

#### Lint
- Command:
- Scope:
- Expected:

#### Typecheck
- Command:
- Scope:
- Expected:

#### Build
- Command:
- Scope:
- Expected:

#### Security / Contract
- Command:
- Scope:
- Expected:

### Execution log
- Date:
- Environment:
- Notes:

### Results summary
- Successes:
- Failures:
- Warnings:

### Requirement trace
<!-- At closeout, one bullet per declared R-ID (spec pose-requirement-evidence-traceability):
- R<N> [satisfied] <verification case; structured refs: check:<name> test:<id> report:<file> commit:<sha>>
- R<N> [satisfied] surface:<id> evidence:integration check:<reachability-check>
- R<N> [deferred-integration: spec:<non-terminal-slug>] surface:<id>
- R<N> [waived: <reason>]
- R<N> [withdrawn: <reason>]
Missing or orphaned IDs fail `pose lint-spec --strict` on done specs. -->

### Known gaps
<!-- Temporary limitations, blocked checks, deferred validations. -->

---

## 7. Final Report

### Delivered scope
<!-- What was implemented and what was intentionally left out. -->

### Files and modules changed
- 

### Validation executed
- Command:
- Result:

### Residual risks
- 

### Follow-ups

<!--
One format, one bullet per follow-up:

  - [<disposition>] <what remains and why> (owner:@alias crit:low|medium|high review:YYYY-MM-DD)

The ownership group is required on [open] items and must be the LAST thing on
the bullet, in parentheses. The bullet may wrap onto indented lines. Ownership
written any other way — after a dash, mid-sentence, without parentheses — is
ignored: the item reads as unowned, with no criticality and no review date, and
never becomes overdue. `pose lint-spec` warns when it sees that.

Valid dispositions:
  [open]                  live backlog without a dedicated spec (needs the ownership group)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded

When the spec is marked `status: done`, every follow-up MUST have a
disposition — `pose followups --open` aggregates the open ones.
-->

- [open] 
