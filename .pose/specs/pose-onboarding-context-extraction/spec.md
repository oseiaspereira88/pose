---
slug: pose-onboarding-context-extraction
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:
priority: 1
components: pose-mcp
delivers:
---

# Spec: pose-onboarding-context-extraction

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
When `pose init`/`pose install` runs against a brownfield repository that
already has a `README.md` (and/or `CLAUDE.md`), populate `AGENTS.md`'s
`## Project context` section from that existing content instead of leaving
the current placeholder comments (`<!-- Describe here... -->`,
`<repo>: describe the repository's purpose...`) for the operator to fill by
hand.

### Business value
`github.com/oseiaspereira88/pose#21`, problems 1 and 4 — the two of the
issue's four problems v1.3.0's `adaptive-instance-provisioning` roadmap did
not address (it closed problems 2 and 3: hardcoded domain rules, empty
module metadata). That roadmap's own text named this exact gap and
deliberately deferred it: "Reading and summarizing a repository's existing
prose documentation (README/CLAUDE.md) into AGENTS.md was evaluated and
deliberately excluded... it carries a materially different risk profile
(summarization, not deterministic file-presence detection)." A brownfield
onboarding today produces an `AGENTS.md` that looks identical whether the
target repository has rich existing documentation or none at all — the tool
never looks.

### Constraints
- Extraction must be conservative: prefer excerpting/structuring existing
  prose (headings, first paragraphs, stated purpose) over generative
  summarization, to keep the result auditable and avoid hallucinated
  project claims ending up in a governance document.
- Must not silently overwrite an `AGENTS.md` a team has already customized
  — same merge-preserving behavior `MergeManagedDoc` already gives
  `POSE.md`/`AGENTS.md` sections elsewhere (`pose-install-locale-autodetect`
  follow-up already flagged this merge path's section-matching is
  heading-text-based, not structural; reuse carefully rather than assuming
  it is a solved problem for this new content).
- A repository with neither `README.md` nor `CLAUDE.md` must fall back to
  today's placeholder behavior unchanged — no regression for the common
  case of a genuinely empty/new repository.

### Non-goals
- Parsing `docs/*` directories or architecture docs beyond the two named
  root files — issue #21 names `docs/*` as a "such as" example, not a
  requirement; a deeper docs crawl is a larger, separately-scoped follow-up
  if this proves valuable.
- The issue's proposed `pose init --discover`/`pose assess onboard` flag as
  a distinct onboarding mode — this spec makes the existing `pose init`/
  `pose install` path smarter, not a new command surface.

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
