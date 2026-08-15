---
slug: pose-scaffold-index-template-neutralization
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority: 1
components: pose-mcp
depends_on:
delivers:
---

# Spec: pose-scaffold-index-template-neutralization

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
Stop `pose init`/`pose install` from seeding a fresh consumer instance's
`.pose/indexes/module-metadata.json` and `.pose/indexes/validation-matrix.json`
with POSE's own development-repository module entries.

### Business value
`github.com/oseiaspereira88/pose#22`: empirically confirmed against a fresh
`pose install` into an empty throwaway repository — `module-metadata.json`
ships with hardcoded `pose-mcp` (`entrypoints: pose-mcp/cmd/pose/main.go`,
`deliveryRole: composition-root`), `mcp-enforce`, and `owner: @pose-maintainers`;
`validation-matrix.json` ships with `moduleOverrides.pose-mcp` (Go tests
against `./internal/pose`, `./internal/cli`, `./internal/mcpserver`) and
`moduleOverrides.docs-site` (Python stack). Both files are byte-identical to
this repository's own dogfooding content. Root cause: the scaffold generator
(`pose-mcp/internal/scaffold/gen/main.go`) copies `.pose/indexes/` verbatim
from pose-dist into the embedded binary dist, and only neutralizes two
specific files (`distpolicy.SelfReferentialPolicyFiles` =
`delivery.json`, `artifacts.json`, both under `.pose/policy/`) — the exact
mechanism that fixed `.pose/policy/*` for issue #17
(`pose-scaffold-self-referential-policy-fix`). `.pose/indexes/*` was never
added to any equivalent neutralization list, so the same leak class recurred
under a different pair of files.

While `pose validate`/`pose index` do not misfire on a fresh instance
(overrides only activate when a same-named directory actually exists, and
none does in a brownfield repo — confirmed empirically, `pose validate
--tolerant --report` runs clean with zero modules matched), the leaked
entries are still committed, versioned, inert-but-wrong ground truth that an
AI agent reading `module-metadata.json` would reasonably treat as accurate
information about the project it is operating in.

### Constraints
- Fix must generalize the existing neutralization mechanism
  (`distpolicy.SelfReferentialPolicyFiles`) rather than hand-patching two
  more files with ad hoc logic — this is the second time this exact leak
  class has appeared (issue #17, then issue #22); the fix must close the
  class, not just the two currently-known instances.
- Must not remove `module-metadata.json`/`validation-matrix.json` entries
  that are legitimately part of *this* repository's (pose-dist's) own
  dogfooded configuration — neutralization only applies to what gets copied
  into the embedded scaffold dist for delivery to consumer instances.

### Non-goals
- Auto-discovering the target repository's real modules at install time —
  that is `pose-stack-detection-consolidation`'s scope
  (`milestone:adaptive-instance-provisioning/stack-detection`), not this
  spec's. This spec only stops shipping the *wrong* static seed; it does not
  replace it with a *correct* dynamic one.
- Any change to `.pose/policy/delivery.json`/`artifacts.json` neutralization
  — already fixed and out of scope for re-litigation.

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
- R1: When the scaffold generator (`go generate ./internal/scaffold`)
  builds the embedded dist, `.pose/indexes/module-metadata.json`'s module
  entries and `.pose/indexes/validation-matrix.json`'s `moduleOverrides`
  shall exclude every entry that only exists because of pose-dist's own
  dogfooding configuration (`pose-mcp`, `mcp-enforce`, `docs-site`,
  `@pose-maintainers`).
- R2: A fresh `pose install`/`pose init` into an empty repository shall
  produce `.pose/indexes/module-metadata.json` with an empty (or neutrally
  seeded, per whatever `pose-stack-detection-consolidation` later defines)
  module set, and `.pose/indexes/validation-matrix.json` with no
  `moduleOverrides` entries.
- R3: pose-dist's own `.pose/indexes/module-metadata.json` and
  `.pose/indexes/validation-matrix.json` (the canonical source, not the
  embedded dist copy) shall be unaffected — neutralization happens only in
  the scaffold generation step, not in the source of truth this repository
  dogfoods against.

### Non-functional
- A regression test must fail the build if any of
  `pose-mcp`/`mcp-enforce`/`docs-site`/`@pose-maintainers` appears anywhere
  under `pose-mcp/internal/scaffold/dist/.pose/indexes/` after
  `go generate ./internal/scaffold` — mirroring how
  `pose-scaffold-self-referential-policy-fix` guarded `.pose/policy/`.

### Compatibility
- Additive/corrective only. Existing consumer instances that already have a
  polluted `module-metadata.json`/`validation-matrix.json` from a prior
  install are not retroactively cleaned by this spec — `pose update` only
  seeds these files "when absent" (`install.go`), so a dirty file already on
  disk is left alone. Cleanup of already-affected instances is a follow-up,
  not blocking this spec's closure.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go`
  (`SelfReferentialPolicyFiles` or a new equivalent list/predicate covering
  `.pose/indexes/`)
- `pose-mcp/internal/scaffold/gen/main.go` (applies the neutralization
  during embedded-dist generation)
- `pose-mcp/internal/scaffold/scaffold_test.go` (regression test)

### Technical risks
- Low: purely subtractive/neutralizing change to a generation-time step;
  no runtime behavior of `pose index`/`pose validate` changes for existing
  logic, since those already ignore overrides for non-existent directories.

---

## 4. Tasks

### Planning
- [x] Confirm the gap empirically against a fresh `pose install` (issue #22
      investigation, this repo, 2026-08-15)
- [x] Confirm root cause: `distpolicy.SelfReferentialPolicyFiles` never
      covered `.pose/indexes/`

### Implementation
- [ ] Extend `distpolicy` with an equivalent neutralization list/predicate
      for `.pose/indexes/module-metadata.json` (strip `pose-mcp`,
      `mcp-enforce` module entries and the `@pose-maintainers` owner
      default) and `.pose/indexes/validation-matrix.json` (strip
      `moduleOverrides.pose-mcp`, `moduleOverrides.docs-site`)
- [ ] Wire the neutralization into `gen/main.go`'s dist-generation walk,
      same call site as the existing `.pose/policy/*` neutralization
- [ ] Add a regression test asserting the embedded dist contains none of
      the four leaked identifiers anywhere under `.pose/indexes/`

### Validation
- [ ] `go -C pose-mcp test ./internal/scaffold/...`
- [ ] Fresh `pose install` into an empty throwaway repo; assert
      `module-metadata.json`/`validation-matrix.json` contents match R2
- [ ] `pose check --strict` on pose-dist itself (confirm the canonical
      source files are untouched, per R3)

---

## 6. Validation

### Strategy
Deterministic: a Go regression test asserting the embedded scaffold dist is
clean, plus one empirical end-to-end fresh-install check against a
throwaway repository confirming R2's exact expected content.

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

- [open] cleanup path for already-affected consumer instances (codass,
  audio-relay, harne8 and any other repo installed before this spec closed)
  — `pose update` only seeds these files when absent, so existing pollution
  is not self-healing; needs its own scoped decision, not assumed here.
