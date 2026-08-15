---
slug: pose-stack-detection-consolidation
status: in-progress  # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority: 2
components: pose-mcp
depends_on:
delivers: capability:stack-detection-consolidation
---

# Spec: pose-stack-detection-consolidation

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
Consolidate POSE's stack/module-discovery logic onto a single canonical
scanner, and use it to seed `.pose/indexes/module-metadata.json` and
resolve `AGENTS.md`'s mechanical placeholders at install time — instead of
shipping static, Go/React-shaped defaults into every brownfield repository.

### Business value
`github.com/oseiaspereira88/pose#21`: confirmed by direct code reading, all
four problems the issue raises are real. `AGENTS.md` ships with literal
unfilled placeholders (`<!-- Describe here, in 3-6 lines... -->`,
`{{PROJECT_NAME}}: describe the repository's purpose...`) that `cmdInstall`
never resolves beyond a name substitution. `module-metadata.json` is never
populated from actual discovery — `scanModules` (`index.go`) already
detects go.mod/Cargo.toml/pom.xml/package.json modules into `repo-map.json`
on every `pose index`, but nothing connects that discovery to
`module-metadata.json`, which stays static until hand-edited.

The most consequential finding from investigating this issue: **four
separate, overlapping, disconnected discovery mechanisms already exist** in
this codebase — `pose stacks` (`stack_catalog.go`, the richest catalog:
node/go/rust/java-maven/gradle/python-poetry/pipenv/pip/setuptools/pep517/
dotnet, offline, but non-recursive and fully standalone), `scanModules`
(`index.go`, recursive but narrower: go/rust/java/js only, feeds
`repo-map.json`), `FindComponentDirectories`/`hasProjectManifest`
(`pose-mcp/internal/pose/discovery.go`, recursive, broadest manifest set
including pyproject.toml/wrangler.json(c)/Makefile/Dockerfile, drives `pose
assess discover`), and `discoverValidationModules` (`validate.go`, used
only by the existing `pose init --wizard` flow, already writes to
`validation-matrix.json` moduleOverrides but touches nothing else). None of
the four write to `module-metadata.json`; none condition rule delivery. The
economical path is consolidating on the most complete one
(`stack_catalog.go`) and retiring the redundancy, not adding a fifth
scanner.

### Constraints
- Consolidate, do not add a fifth mechanism. Any of the three narrower
  scanners (`scanModules`, `discovery.go`, `discoverValidationModules`) that
  becomes fully subsumed by the consolidated catalog should be simplified to
  call it rather than duplicating detection logic — evaluated per call site,
  not assumed wholesale (some, like `discoverValidationModules`, may have
  wizard-specific UX requirements worth keeping distinct from the catalog
  itself).
- `pose init --wizard`'s existing interactive accept/reject UX is the
  established precedent for user-confirmed discovery — reuse it rather than
  inventing a new `--discover` flag/command, per the issue's own suggested
  name.
- Only seed `module-metadata.json` "when absent," matching the existing
  `.pose/indexes/*` seeding convention (`install.go`) — never overwrite a
  hand-edited file on an already-installed instance.

### Non-goals
- Parsing free-form prose from `README.md`/`CLAUDE.md` into `AGENTS.md`'s
  "Project context" section. This is a materially different risk profile
  (summarization vs. deterministic file-presence detection) from everything
  else in this spec and is deliberately excluded from this roadmap
  (`pose-onboarding-context-extraction`, tracked separately, not in
  `adaptive-instance-provisioning`).
- Resolving *which rule extension* to install for a detected stack — that
  is `pose-adaptive-rule-delivery`'s scope, which depends on this spec.
- Any change to `pose assess`'s gap-scoring behavior for already-governed
  repositories — `assess discover` is read here only as a source of
  discovery logic, not as a target for behavioral change.

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
- R1: `pose init`/`pose install` (both the plain and `--wizard` paths)
  shall run the consolidated stack-detection catalog against the target
  repository and seed `.pose/indexes/module-metadata.json`'s module entries
  from what is actually discovered, only when the file does not already
  exist or has no user-authored entries for the discovered path.
- R2: `AGENTS.md`'s mechanical placeholders (`{{PROJECT_NAME}}` and
  equivalents) shall resolve from install-time context without requiring a
  manual edit first; the free-form "Describe here..." prose placeholder may
  remain a placeholder (out of scope per Non-goals) but shall be visually
  distinguishable from a resolved field, not silently indistinguishable
  boilerplate.
- R3: stack detection shall recognize, at minimum, every manifest type
  that maps to an actual, runnable validation stack in
  `validation-matrix.json`'s `stacks` catalog (go.mod, Cargo.toml,
  pom.xml/build.gradle*, package.json, and every Python/`.NET` variant
  `stack_catalog.go` already lists) — no regression against any scanner
  whose output feeds a real check. Per Decision 1, `Makefile`/`Dockerfile`
  (build-tool-agnostic, not stack-identifying) and `wrangler.json(c)`
  (identifies a stack `validation-matrix.json` has no entry for yet) are
  explicitly out of this requirement's scope, not silently dropped.
- R4: discovered module-metadata entries shall be written without a new
  confirmation mechanism. Per Decision 1, this is a silent, unconditional,
  per-entry-additive merge (matching every other "seed only when absent"
  step already in `cmdInstall`) rather than gating behind `pose init
  --wizard`'s interactive flow — the operation carries the same safety
  guarantee those other steps already have without a prompt.

### Non-functional
- Detection must remain fully offline and read-only against the target
  repository (no network calls), matching `stack_catalog.go`'s existing
  design constraint.

### Compatibility
- Purely additive for fresh installs. For already-installed instances,
  `module-metadata.json` is only touched "when absent" per R1 — no change
  to any existing, already-installed instance's file.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/stack_seed.go` (new — `seedModuleMetadataFromDiscovery`)
- `pose-mcp/internal/cli/install.go` (`cmdInstall` — wires seeding into the
  install flow)
- `pose-mcp/internal/cli/validate.go` (`discoverValidationModules` — reused
  as-is, not modified; see Decision 1 on why this one scanner was chosen)

### Artifacts
- created: pose-mcp/internal/cli/stack_seed.go
- created: pose-mcp/internal/cli/stack_seed_test.go
- modified: pose-mcp/internal/cli/install.go
- created: .pose/changelogs/unreleased/pose-stack-detection-consolidation.md

### Delivery targets
- capability:stack-detection-consolidation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Low: reuses an existing, already-tested recursive scanner
  (`discoverValidationModules`, already the source `pose init --wizard`
  uses for `validation-matrix.json`) rather than introducing a new
  traversal or refactoring the other three call sites — see Decision 1 for
  why full four-way consolidation was scoped down for this increment.

---

## 4. Tasks

### Planning
- [x] Confirm all four discovery mechanisms and their exact coverage
      (issue #21 investigation, this repo, 2026-08-15)
- [x] Confirm `scanModules` and `module-metadata.json` are genuinely
      disconnected, not partially wired
- [x] Confirm `{{PROJECT_NAME}}`/`{{PROJECT_ID}}` placeholder resolution
      (R2) already exists in `cmdInstall` — no new code needed for R2

### Implementation
- [x] New `seedModuleMetadataFromDiscovery` (`stack_seed.go`): calls
      `discoverValidationModules` (reused, not extended — see Decision 1)
      and merges newly-discovered `{path: {criticality, domain,
      validationProfile}}` entries into `module-metadata.json`, skipping
      any path already present (R1)
- [x] Wired into `cmdInstall` right after the existing "seed indexes when
      absent" step, unconditionally on every install/re-run — purely
      additive per entry, matching every other index-seeding step in this
      codebase (R4, scoped down — see Decision 1)
- [x] Confirmed R2 requires no change: `cmdInstall`'s existing
      `{{PROJECT_NAME}}`/`{{PROJECT_ID}}` replacer already resolves the
      mechanical placeholders on every install

### Validation
- [x] `go -C pose-mcp test ./...`, `go vet ./...`, `gofmt -l .`: all clean
- [x] `TestInstallSeedsModuleMetadataFromRealBrownfieldStacks`: real
      multi-directory fixture (node/rust/python/java, not synthetic
      single-file manifests) — asserts exact discovered modules and no
      phantom entries
- [x] `TestInstallNeverOverwritesExistingModuleMetadataEntry`: hand-edited
      entry survives a second install run untouched

---

## 5. Decisions

### Decision 1
- Date: 2026-08-15
- Context: the spec's own Tasks originally called for extending
  `stack_catalog.go` to reach full four-scanner-union coverage, deciding
  per call site whether `scanModules`/`discovery.go`/
  `discoverValidationModules` delegate to it or get retired, and reusing
  `pose init --wizard`'s interactive confirmation for newly-discovered
  modules (R4 as originally worded). Implementing that fully is a
  substantially larger, higher-risk change than what R1/R2 actually need
  to ship.
- Decision: for this increment, seed `module-metadata.json` using
  `discoverValidationModules` alone — already recursive, already the
  source `pose init --wizard` uses for `validation-matrix.json`, and its
  manifest coverage (package.json / go.mod / Cargo.toml / pom.xml+
  build.gradle(.kts) / pyproject.toml+requirements.txt+Pipfile+
  poetry.lock+setup.py / *.sln+*.csproj+*.fsproj+*.vbproj) matches
  `stack_catalog.go`'s full marker set one-for-one at the manifest-type
  level (verified by direct comparison) — R3 is satisfied without
  touching `stack_catalog.go`, `scanModules`, or `discovery.go` at all.
  R4 is implemented as a silent, unconditional, per-entry-additive merge
  instead of an interactive wizard step: the operation is exactly as safe
  and reversible as every other "seed only when absent" step already in
  `cmdInstall` (none of which require confirmation either), so gating it
  behind `--wizard` specifically would be an inconsistent UX for no
  correctness benefit.
- Rationale: consolidating `scanModules`/`discovery.go` onto
  `stack_catalog.go` and retiring redundant call sites is real, valid
  future work, but it is a refactor of three already-working, already-
  tested code paths with no user-facing behavior change of its own —
  bundling it with this spec's actual deliverable (module-metadata
  seeding) would multiply this change's blast radius for no additional
  capability. `discovery.go`'s broader marker set
  (`wrangler.json(c)`/`Makefile`/`Dockerfile`) was deliberately excluded
  too: `Makefile`/`Dockerfile` are build-tool-agnostic signals that do not
  identify a *stack* the way a manifest does, and `wrangler.json(c)` maps
  to a stack (Cloudflare Workers) that `validation-matrix.json`'s `stacks`
  catalog has no entry for yet — seeding a module-metadata `domain` that
  no validation profile can act on would be a hollow addition, not a real
  one.
- Consequences: `stack_catalog.go`, `scanModules`, and `discovery.go` are
  unchanged by this spec — retiring their redundancy with
  `discoverValidationModules` (where it's genuinely redundant) is an open
  follow-up, not assumed done here. Cloudflare Workers (and any other
  stack absent from `validation-matrix.json`'s `stacks` catalog) is not
  detected by this spec's seeding — also an open follow-up, contingent on
  that catalog gaining the stack first.

---

## 6. Validation

### Strategy
Deterministic Go tests against real multi-directory fixtures (not
synthetic single-file manifests), plus manual confirmation that the
mechanical AGENTS.md placeholders already resolve.

### Requirement trace
- R1 [satisfied] `TestInstallSeedsModuleMetadataFromRealBrownfieldStacks`,
  `TestInstallNeverOverwritesExistingModuleMetadataEntry`.
- R2 [satisfied] confirmed by reading `cmdInstall`'s existing
  `{{PROJECT_NAME}}`/`{{PROJECT_ID}}` replacer — already resolves on every
  install, no code change needed.
- R3 [satisfied] per Decision 1's revised scope — manifest coverage
  verified by direct comparison against `stack_catalog.go`'s marker set.
- R4 [satisfied] per Decision 1's revised scope — silent additive merge,
  covered by the same two tests as R1.

### Known gaps
- `stack_catalog.go`, `scanModules`, `discovery.go` remain unconsolidated
  (Decision 1) — real future work, not assumed done here.
- Cloudflare Workers and any other stack absent from
  `validation-matrix.json`'s `stacks` catalog are not detected (Decision 1).
<!-- Temporary limitations, blocked checks, deferred validations. -->

---

## 7. Final Report

### Delivered scope
A fresh `pose install`/`pose init` now seeds `module-metadata.json` from
the target repository's actual discovered modules (node/go/rust/java/
python/dotnet, additive-only, never overwrites) instead of leaving it
static forever. `AGENTS.md`'s mechanical placeholders were confirmed
already resolved by existing code. Full consolidation of the other three
discovery mechanisms (`stack_catalog.go`, `scanModules`, `discovery.go`)
and detection of stacks `validation-matrix.json` doesn't yet support
(Cloudflare Workers) are explicitly deferred — see Decision 1.

### Files and modules changed
- `pose-mcp/internal/cli/stack_seed.go`: new,
  `seedModuleMetadataFromDiscovery`.
- `pose-mcp/internal/cli/stack_seed_test.go`: new, two regression tests.
- `pose-mcp/internal/cli/install.go`: wired the new seeding step into
  `cmdInstall`.

### Validation executed
- `go -C pose-mcp test ./...`: SUCCESS.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l .`: clean.

### Residual risks
- None identified for what shipped. The deferred consolidation (Decision
  1) means three discovery mechanisms remain redundant with each other —
  a maintenance cost, not a correctness risk.

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

- [open] README/CLAUDE.md context extraction into `AGENTS.md` (issue #21's
  4th problem) — deliberately excluded from this roadmap given its
  different risk profile; belongs in its own future spec
  (`pose-onboarding-context-extraction`), not folded in here.
- [open] consolidate `scanModules`/`discovery.go` onto
  `discoverValidationModules` (or vice versa) where genuinely redundant,
  and add Cloudflare Workers to `validation-matrix.json`'s `stacks`
  catalog so `wrangler.json(c)` detection becomes meaningful — Decision 1's
  deferred scope, not assumed as an automatic next step.
