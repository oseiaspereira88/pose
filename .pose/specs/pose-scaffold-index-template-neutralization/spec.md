---
slug: pose-scaffold-index-template-neutralization
status: in-progress  # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority: 1
components: pose-mcp
depends_on:
delivers: capability:scaffold-index-template-neutralization
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
- R4: a regression test shall fail the build if the embedded scaffold
  dist's `.pose/indexes/module-metadata.json`/`validation-matrix.json`
  ever drifts from their neutral templates after
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

### Artifacts
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go
- modified: pose-mcp/internal/scaffold/gen/main.go
- modified: pose-mcp/internal/scaffold/scaffold_test.go
- created: .pose/changelogs/unreleased/pose-scaffold-index-template-neutralization.md

### Delivery targets
- capability:scaffold-index-template-neutralization module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

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
- [x] Added `SelfReferentialIndexFiles` and `NeutralIndexTemplates()` to
      `distpolicy.go`, extending `IsIncluded` to exclude
      `.pose/indexes/module-metadata.json`/`validation-matrix.json` from
      the wholesale sync — mirroring `SelfReferentialPolicyFiles`. Unlike
      the policy templates, the index templates preserve
      `validation-matrix.json`'s generic `stacks`/`deliveryProfiles`
      content and only blank `modules`, `defaults.owner`/`domain`, and
      `moduleOverrides`.
- [x] Wired `NeutralIndexTemplates()` into `gen/main.go`'s existing
      neutral-template write loop (merged with `NeutralPolicyTemplates()`
      into one map, same call site)
- [x] Extended `scaffold_test.go`'s drift guard to check both template
      sources; added `TestSelfReferentialIndexFilesExcluded` and
      `TestNeutralIndexTemplatesAreSchemaValidAndClean` to
      `distpolicy_test.go`, asserting `modules`/`moduleOverrides` are
      empty and `owner`/`domain` are blank while `stacks`/
      `deliveryProfiles` stay populated

### Validation
- [x] `go -C pose-mcp test ./internal/scaffold/...`: SUCCESS
- [x] `go -C pose-mcp test ./...`, `go vet ./...`, `gofmt -l .`: all clean
- [x] Fresh `pose install` into an empty throwaway git repo (`/tmp`,
      cleaned up after): `module-metadata.json` ships `modules: {}`,
      `owner`/`domain` blank; `validation-matrix.json` ships
      `moduleOverrides: {}` while `stacks`/`deliveryProfiles` remain fully
      populated (6 stacks, 5 profiles) — R2 confirmed
- [x] `git diff --stat` on pose-dist's own canonical
      `.pose/indexes/module-metadata.json`/`validation-matrix.json`:
      empty — R3 confirmed, only the embedded dist copy changed

---

## 6. Validation

### Strategy
Deterministic: a Go regression test asserting the embedded scaffold dist is
clean, plus one empirical end-to-end fresh-install check against a
throwaway repository confirming R2's exact expected content.

### Requirement trace
- R1 [satisfied] check:TestSelfReferentialIndexFilesExcluded
  check:TestNeutralIndexTemplatesAreSchemaValidAndClean — leaked entries
  excluded and structurally verified absent.
- R2 [satisfied] empirical fresh-install check against a throwaway
  repository — exact expected content confirmed.
- R3 [satisfied] `git diff --stat` against pose-dist's own canonical
  `.pose/indexes/*` — no diff.
- R4 [satisfied] `TestEmbeddedDistMatchesPoseDist` (`scaffold_test.go`) —
  fails the build if the embedded dist ever drifts from either neutral
  template again.

### Known gaps
- None identified.

---

## 7. Final Report

### Delivered scope
Extended the neutralization mechanism issue #17 established for
`.pose/policy/*` to cover `.pose/indexes/module-metadata.json` and
`.pose/indexes/validation-matrix.json`. A fresh `pose install`/`pose init`
no longer seeds a consumer instance with pose-mcp's own dogfooded module
entries, owner, domain, or module-specific check overrides — while the
generic, reusable `stacks`/`deliveryProfiles` catalog in
`validation-matrix.json` ships unchanged. Cleanup of already-affected
consumer instances is explicitly out of scope (see Follow-ups).

### Files and modules changed
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go`:
  `SelfReferentialIndexFiles`, `NeutralIndexTemplates()`, `IsIncluded`
  extended.
- `pose-mcp/internal/scaffold/gen/main.go`: merged neutral-template write
  loop.
- `pose-mcp/internal/scaffold/scaffold_test.go`: drift guard checks both
  template sources.
- `pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go`: two new
  regression tests.
- `pose-mcp/internal/scaffold/dist/**`: regenerated.

### Validation executed
- `go -C pose-mcp test ./...`: SUCCESS.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l .`: clean.
- Manual end-to-end fresh install against a throwaway `/tmp` repository:
  confirmed clean `module-metadata.json`/`validation-matrix.json`.

### Residual risks
- None identified for new installs. Already-affected instances remain
  polluted until they are individually remediated (tracked as an open
  follow-up, not blocking this spec).

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
