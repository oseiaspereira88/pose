---
slug: pose-validation-scanner-consolidation
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:
priority: 2
components: pose-mcp
delivers:
---

# Spec: pose-validation-scanner-consolidation

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
Consolidate `pose-mcp`'s duplicate module-scanning logic
(`scanModules`/`discovery.go` vs. `discoverValidationModules`) onto the
single scanner `pose-stack-detection-consolidation` (v1.3.0) already chose
as the reuse target, and add Cloudflare Workers (`wrangler.json(c)`) to
`validation-matrix.json`'s `stacks` catalog so its existing detection
signal becomes actionable rather than dead weight.

### Business value
Named as a deferred follow-up in `pose-stack-detection-consolidation`'s
Final Report: `discoverValidationModules` was deliberately chosen over
building a fifth scanner or refactoring the existing ones, but the
pre-existing `scanModules`/`discovery.go` path was left untouched and now
overlaps in responsibility with the one `pose install`/`pose init` use for
module-metadata seeding. Two scanners doing related work is a real
maintenance and correctness liability — a stack-detection fix applied to
one will silently not apply to the other, the exact shape of gap this
roadmap's `#21`-driven predecessor was formed to close. Independent of the
consolidation, `wrangler.json(c)` is already a name the codebase
recognizes somewhere in this area but currently has no corresponding
`stacks` catalog entry, so Cloudflare Workers detection currently
identifies nothing actionable.

### Constraints
- No behavior regression for any of the five already-supported stacks
  (Node, Go, Rust, Java, Python-all-managers, .NET) — this is a refactor
  of scanning machinery, not a change to what gets detected for those.
- Consolidation direction (fold `scanModules`/`discovery.go` into
  `discoverValidationModules`, or the reverse) is an open technical
  question for this spec's Technical Plan, not pre-decided here.

### Non-goals
- Adding new stack support beyond Cloudflare Workers — other stacks named
  in backlog follow-ups (poetry/pipenv/dotnet-solution fixtures, etc.) are
  separate, already-tracked items.
- Changing `validationProfile` in `module-metadata.json`'s schema — flagged
  elsewhere (`pose-monorepo-validation-advisory` follow-up) as unread dead
  weight, but that is a distinct decision from this spec's scanner
  consolidation.

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
