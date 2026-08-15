---
slug: pose-adaptive-rule-delivery
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority: 3
components: pose-mcp
depends_on: pose-domain-rule-extension-migration, pose-stack-detection-consolidation
delivers:
---

# Spec: pose-adaptive-rule-delivery

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
Connect detected stack (`pose-stack-detection-consolidation`) to a concrete
rule-extension install decision (`pose-domain-rule-extension-migration`),
so a fresh install receives the domain rule matching its actual stack
instead of either a wrong default or nothing at all.

### Business value
`github.com/oseiaspereira88/pose#21` (adaptive rule delivery) and
`github.com/oseiaspereira88/pose#24` (extension-based delivery) both name
this outcome, but neither is buildable in isolation: adaptively installing
"the right rule for the stack" is not a well-defined operation until the
candidate rules are individually installable, versioned artifacts (which
`pose-domain-rule-extension-migration` produces) resolved against a real
detection signal (which `pose-stack-detection-consolidation` produces).
This spec is deliberately the join of the two, not a restatement of either.

Scope is intentionally bounded to whatever extensions exist by the time
this spec starts — initially `pose-rule-backend-go`,
`pose-rule-frontend-react`, and the pre-existing `pose-rule-kubernetes`.
It does not require or assume a general catalog: a small, in-repo,
curated stack-to-extension mapping is sufficient and matches what actually
exists today.

### Constraints
- Cannot start implementation before both `pose-domain-rule-extension-migration`
  and `pose-stack-detection-consolidation` reach `status: done` — the
  roadmap's `adaptive-delivery` milestone declares this dependency
  explicitly (`after: rule-extensionization, stack-detection`).
- Must not silently install an extension without either an explicit
  baseline default the user can decline, or an interactive confirmation —
  reuses `pose init --wizard`'s established accept/reject UX rather than a
  new confirmation surface, per `pose-stack-detection-consolidation`'s R4.
- Must not fail or block install when a detected stack has no matching
  extension yet (e.g. Rust, Python) — installing zero domain rules and
  saying so is the correct behavior until those extensions are authored,
  not an error.

### Non-goals
- Authoring new stack extensions beyond what
  `pose-domain-rule-extension-migration` already produces — adding Rust/
  Python/mobile/etc. rule extensions is future content work, not part of
  wiring the resolution mechanism.
- Building catalog/registry/discovery infrastructure — the mapping this
  spec needs is small and curated, not discovered.

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
- R1: at install time, for each stack `pose-stack-detection-consolidation`
  detects, the install flow shall resolve a curated stack-to-extension
  mapping and, when a match exists, offer to install the matching rule
  extension (auto-install a baseline or prompt, per Constraints) via the
  existing `pose extension install` mechanism.
- R2: when a detected stack has no matching extension, install shall
  proceed without error, and the outcome (no domain rule installed for
  that stack) shall be visible to the user/agent, not silent.
- R3: no repository shall receive a domain rule for a stack it does not
  use, and no repository with a detected, matched stack shall be left
  without that rule after a default (non-`--wizard`) install.

### Compatibility
- Additive for fresh installs. Does not retroactively install extensions
  into already-existing instances — an instance that predates this spec
  keeps whatever `.pose/rules/` content it already has until it next runs
  through an install/update path this spec touches.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/install.go` (`cmdInstall` — wire resolution step
  after stack detection)
- `pose-mcp/internal/cli/extension.go` (`pose extension install` — reused,
  not modified, as the actual install mechanism)
- new: a curated stack-to-extension mapping (exact location TBD — likely a
  small static table alongside `stack_catalog.go`, not a new config file
  format)

### Technical risks
- Low, conditional on its two prerequisite specs already being done: this
  spec is primarily a join/wiring exercise between two already-validated
  mechanisms, not new primitive capability.

---

## 4. Tasks

### Planning
- [x] Confirm this spec is a join of two prerequisites, not independently
      buildable (issue #21/#24 investigation, this repo, 2026-08-15)
- [ ] Blocked: wait for `pose-domain-rule-extension-migration` and
      `pose-stack-detection-consolidation` to reach `status: done`

### Implementation
- [ ] TBD — blocked on prerequisites

### Validation
- [ ] TBD — blocked on prerequisites

---

## 6. Validation

### Strategy
To be defined once prerequisites are done: end-to-end install tests across
each currently-mapped stack (Go, React, Kubernetes) confirming the correct
extension installs, plus a negative test confirming graceful no-op for an
unmapped stack (R2).

### Requirement trace
<!-- At closeout, one bullet per declared R-ID (spec pose-requirement-evidence-traceability):
- R<N> [satisfied] <verification case; structured refs: check:<name> test:<id> report:<file> commit:<sha>>
- R<N> [satisfied] surface:<id> evidence:integration check:<reachability-check>
- R<N> [deferred-integration: spec:<non-terminal-slug>] surface:<id>
- R<N> [waived: <reason>]
- R<N> [withdrawn: <reason>]
Missing or orphaned IDs fail `pose lint-spec --strict` on done specs. -->

### Known gaps
- Blocked on `pose-domain-rule-extension-migration` and
  `pose-stack-detection-consolidation`; this spec cannot start
  implementation until both are `status: done`.

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

- [open] non-Go/React/Kubernetes stack extensions (Rust, Python, mobile,
  Cloudflare Workers, etc.) — this spec wires resolution for whatever
  extensions exist; authoring new ones is separate content work with no
  spec of its own yet. 
