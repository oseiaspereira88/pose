---
slug: pose-rule-extension-locale-parity
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:
priority: 3
components: pose-mcp
delivers:
---

# Spec: pose-rule-extension-locale-parity

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
Give `pose-rule-backend-go` and `pose-rule-frontend-react` (the two domain
rule extensions `pose-domain-rule-extension-migration` shipped in v1.3.0) a
pt-BR content variant, so a pt-BR instance installing either receives
locale-consistent content instead of English-only rule text alongside
pt-BR core machinery.

### Business value
Named as a follow-up in `pose-domain-rule-extension-migration`'s Final
Report: "extensions carry no locale variant — a pt-BR instance installing
`pose-rule-backend-go`/`pose-rule-frontend-react` gets English-only
content, a real (if pre-existing, matching kubernetes) gap." This repository's
own governance documents (`README.pt-BR.md`) and `pose-locale-coverage-contract`
(a dedicated gate enforcing English/pt-BR parity for core machinery) treat
pt-BR as a first-class, contractually-covered locale — extensions are the
one distribution path that gate does not reach today, so the parity
guarantee the project already makes has a silent exception.

### Constraints
- Must follow whatever locale-selection mechanism core machinery already
  uses (`--locale`, brownfield auto-detection via `machineryLocale()`) —
  not a parallel, extension-specific mechanism.
- `pose-rule-kubernetes` has the identical gap and predates this work;
  confirm during Technical Plan whether this spec's mechanism should also
  close it or whether that is explicitly out of scope (kubernetes is a
  third extension, not one of the two this roadmap's predecessor shipped).

### Non-goals
- Localizing extension content into any locale beyond pt-BR — no other
  locale exists in this project yet (`pose-locale-coverage-contract`'s own
  Known gaps note this: "Only pt-BR exists, so the contract has never been
  exercised by a second locale").
- Building a general N-locale extension-authoring framework ahead of a
  second real locale actually existing.

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
