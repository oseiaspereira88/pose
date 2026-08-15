---
slug: pose-extension-catalog-resolution
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-extension-catalog-lifecycle
priority: 4
components: pose-mcp
delivers:
---

# Spec: pose-extension-catalog-resolution

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
Give `pose extension install` a way to resolve an extension ID
(`pose-rule-backend-go`, `pose-rule-frontend-react`, ...) to a real,
signature-verified package through a defined catalog source, instead of
requiring the operator to already have a local directory containing it.

### Business value
`pose-adaptive-rule-delivery` (v1.3.0) made `pose doctor` recommend the
right extension for a detected stack, but the recommendation stops at
"install `pose-rule-backend-go`" — the operator still has to source the
package by hand, because `cmdExtensionInstall`
(`pose-mcp/internal/cli/extension.go`) only accepts a local directory path.
This is the deferred scope `pose-adaptive-rule-delivery`'s Final Report
named explicitly: "needed before the doctor advisory can become a single
runnable command." It also closes a gap between promise and implementation:
`pose-extension-catalog-lifecycle` (2026-07-19, `status: done`) already
specifies "R3: A signed catalog shall support discovery" — read against
today's `cmdExtensionInstall`, discovery of *new* packages by ID was never
actually built; what exists is listing/verifying an already-installed
catalog. Completing R3 as originally intended is this spec's throughline,
not a new promise.

### Constraints
- Must not weaken the existing trust model: `pose extension install`
  already verifies cosign/Sigstore signatures on the resolved package
  before applying it — resolution adds a lookup step in front of that
  verification, it does not replace or bypass it.
- The extension whitelist restriction (installs confined to
  `.agents/skills/`, `.pose/workflows/`, `.pose/rules/`, `.pose/templates/`)
  is unchanged by this spec.
- Requires deciding a concrete catalog source (a versioned index published
  alongside POSE releases, a GitHub-hosted registry, or something else) —
  this is a real trust-boundary decision and belongs in this spec's
  Decisions section (and possibly its own ADR) before implementation, not
  assumed here.

### Non-goals
- A public, unmoderated extension marketplace —
  `pose-extension-catalog-lifecycle`'s own Non-goals already rule this out
  and this spec inherits that boundary.
- Executing installer scripts as part of resolution — resolution only
  locates and hands off a package; installation semantics (dry-run,
  transactional apply, conflict handling) are unchanged from what
  `cmdExtensionInstall` already does for a local directory today.

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
- R1: 

### Non-functional
- 

### Security
- 

### Compatibility
- 

---

## 3. Technical Plan

### Affected areas
- 

### Artifacts
<!-- Declare exact project-relative source-tree paths: created, modified,
     renamed (old -> new), removed, or one `none: <reason>` entry. -->
- modified: path/to/file

### Delivery targets
<!-- When `delivers` is populated, declare the exact same refs here. Profiles
     and evidenceClass requirements come from validation-matrix.json. -->
<!-- none yet: delivery root and profile to be decided during Technical Plan -->

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

> Optional section. Use it when the implementation involves trade-offs or
> alternatives.

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
Every follow-up starts with a bracketed disposition. When the spec is marked
`status: done`, every follow-up MUST have one (use `[open]` for the untriaged
ones — `pose followups --open` aggregates them).

Valid dispositions:
  [open]                  not yet triaged (live backlog)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded
-->

- [open] 
