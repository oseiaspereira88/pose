---
slug: pose-validation-scanner-consolidation
status: in-progress
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:
priority: 2
components: pose-mcp
delivers: capability:validation-scanner-consolidation
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
- R1: `scanModules` (`pose index`) and `discoverValidationModules` (`pose
  validate`/`install`/`init`) shall classify a manifest file's stack
  through one shared function, not two independent switch statements.
- R2: the shared classification shall recognize `wrangler.toml`,
  `wrangler.json` and `wrangler.jsonc` as the `cloudflare-workers` stack.
- R3: `validation-matrix.json`'s `stacks` catalog (both the real file and
  the neutral scaffold template `NeutralIndexTemplates` ships) shall carry
  a `cloudflare-workers` entry with at least one runnable check, so a
  detected Cloudflare Workers module resolves to an actionable check set
  instead of none.

### Non-functional
- No regression for any of the six stacks `discoverValidationModules`
  already supported (node, go, rust, java, python, dotnet) — verified by
  running the existing brownfield-discovery test fixtures unchanged
  through the refactored code path.

### Compatibility
- `scanModules`'s public output (`indexedModule.Language`) keeps its
  existing "javascript" label for Node modules — the shared function
  returns "node" (matching `discoverValidationModules`'s existing stack
  name), translated at the `scanModules` call site so `repo-map.json`/
  `services.json`/`packages.json` consumers see no field-value change for
  the four languages `pose index` already reported.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/index.go` (`scanModules`)
- `pose-mcp/internal/cli/validate.go` (`discoverValidationModules`)
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go`
  (`NeutralIndexTemplates`'s hand-maintained `validation-matrix.json`
  literal — a fourth place this catalog is duplicated, discovered while
  implementing R3)
- `.pose/indexes/validation-matrix.json`

### Artifacts
- created: pose-mcp/internal/cli/stack_manifest.go
- created: pose-mcp/internal/cli/stack_manifest_test.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets
- capability:validation-scanner-consolidation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- `validation-matrix.json`'s `stacks` object gains one new key,
  `cloudflare-workers` — additive, no existing key changes shape.

### Technical risks
- Low: `stackForManifestFile` is a pure function extracted verbatim from
  `discoverValidationModules`'s existing switch (no logic change to the
  six already-supported stacks); `scanModules`'s call site translates
  "node" back to its pre-existing "javascript" label, so its output shape
  is unchanged for the four stacks it already reported.
- Low: the `wrangler-version` check uses `severity: optional`, so an
  environment without the `wrangler` binary installed reports a skip/
  warning, never a `--strict` failure (`internal/cli/validate_results.go`'s
  existing optional-severity handling, unchanged by this spec).

---

## 4. Tasks

### Planning
- [x] Read both scanners in full; confirmed `discoverValidationModules`'s
      six-stack switch is the strictly richer one (`pose-stack-detection-
      consolidation` already established this) and `scanModules`'s only
      real additional concern is Docker/Helm/README collection, unrelated
      to stack classification
- [x] Investigated `internal/pose/discovery.go` (the actual `discovery.go`
      the follow-up's file name pointed at) and confirmed it is a
      DIFFERENT concern (`hasProjectManifest`: boolean component presence
      for `pose component-discover`/tech-debt assessment, own manifest
      list including Makefile/Dockerfile/wrangler.json(c)) — see Decision 1
      for why it is out of scope here

### Implementation
- [x] Extracted `stackForManifestFile` (`stack_manifest.go`) from
      `discoverValidationModules`'s switch, unchanged in behavior (R1)
- [x] Added `wrangler.toml`/`wrangler.json`/`wrangler.jsonc` →
      `cloudflare-workers` to the shared function (R2)
- [x] Wired `scanModules` to call the shared function, translating
      `"node"` back to `"javascript"` for output compatibility (R1,
      Compatibility)
- [x] Added a `cloudflare-workers` entry (one optional `wrangler-version`
      check) to `.pose/indexes/validation-matrix.json` (R3)
- [x] Found and fixed the SAME catalog hand-duplicated a fourth time, in
      `NeutralIndexTemplates`'s literal — added the identical entry there
      too (R3); `go generate ./internal/scaffold`'s existing drift test
      would not have caught a divergence here since it doesn't diff
      `stacks` content field-by-field, only structural shape

### Validation
- [x] New unit/integration tests: `stackForManifestFile` full marker
      table, `scanModules` real-fixture detection across all shared
      stacks, `discoverValidationModules` Cloudflare Workers detection,
      `pose install` seeding a `cloudflare-workers` module end to end
- [x] `go test ./...`, `go vet ./...`, `gofmt -l .`: all clean

---

## 5. Decisions

> Optional section. Use it when the implementation involves trade-offs or
> alternatives.

### Decision 1
- Date: 2026-08-15
- Context: the predecessor spec's follow-up named "`scanModules`/
  `discovery.go`" as the consolidation target, phrased as if it were one
  unit. Investigation found `discovery.go` is actually
  `internal/pose/discovery.go` — a different package, different purpose
  (`hasProjectManifest`: boolean "does this directory look like a
  component" presence check for `pose component-discover`/tech-debt
  assessment, feeding `FindComponentDirectories`), and a different marker
  list (includes `Makefile`, `Dockerfile`, `wrangler.json(c)` as generic
  presence signals — not stack classification).
- Options considered: (a) fold `hasProjectManifest` into the shared
  `stackForManifestFile` classification too, so all three scanners share
  one table; (b) scope this spec to the two functions that do genuinely
  equivalent work — `scanModules` and `discoverValidationModules`, both
  classifying a manifest into exactly one stack string — and leave
  `hasProjectManifest` untouched.
- Decision: (b).
- Rationale: `hasProjectManifest` answers a different question (presence,
  not classification) and already recognizes markers the classification
  table deliberately does not treat as stack-defining on their own
  (`Makefile`, `Dockerfile`) — merging it would either lose that broader
  presence signal or force stack classification to grow spurious cases.
  The predecessor follow-up's actual complaint — "a stack-detection fix
  applied to one will silently not apply to the other" — is fully
  addressed by unifying the two functions that do the same job; folding in
  a third, differently-scoped function would be scope creep past what was
  reported.
- Consequences: `internal/pose/discovery.go`'s `hasProjectManifest` keeps
  its own independent marker list. If it drifts from the classification
  table in the future (e.g. a new stack marker added to one but not the
  other), that is a distinct, smaller risk than the one this spec closes
  — tracked as a Follow-up, not fixed here.

---

## 6. Validation

### Strategy
Table-driven unit test for the shared function across every recognized and
unrecognized marker; real-fixture integration tests for each of the two
call sites (`scanModules` via a temp dir with all seven stacks present,
`discoverValidationModules` for the three Cloudflare Workers marker
variants); one end-to-end `pose install` test confirming a
`cloudflare-workers`-only module gets seeded into `module-metadata.json`
with a matching entry in the seeded `validation-matrix.json`.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./...`
- Scope: whole module
- Expected: PASS

#### Lint
- Command: `go -C pose-mcp vet ./...` / `gofmt -l .`
- Scope: whole module
- Expected: clean

#### Build
- Command: `go -C pose-mcp build -trimpath -o ./pose ./cmd/pose`
- Scope: `cmd/pose`
- Expected: builds

### Execution log
- Date: 2026-08-15
- Environment: local (Linux, Go toolchain per `go.mod`)
- Notes: full suite clean on first run after wiring both call sites and
  the neutral-template literal fix.

### Results summary
- Successes: `go test ./...`, `go vet ./...`, `gofmt -l .` all clean;
  four new tests (one table-driven, three integration) all pass.
- Failures: none.
- Warnings: none.

### Requirement trace
- R1 [satisfied] test:TestScanModulesDetectsSharedStacksAndTranslatesNodeLabel test:TestDiscoverValidationModulesDetectsCloudflareWorkers
- R2 [satisfied] test:TestStackForManifestFileRecognizesEveryMarker
- R3 [satisfied] test:TestInstallSeedsModuleMetadataForCloudflareWorkersModule

### Known gaps
- None identified for the scope this spec covers (Decision 1). The
  Cloudflare Workers check itself (`wrangler --version`) is a minimal
  presence sanity check, not a real build/test/deploy validation — a
  deeper Cloudflare Workers check set (bundling, type-checking) is future
  scope once real usage surfaces what matters.

---

## 7. Final Report

### Delivered scope
`scanModules` (`pose index`) and `discoverValidationModules` (`pose
validate`/`install`/`init`) now classify manifest files through one shared
function, `stackForManifestFile`, instead of two independently maintained
switch statements. Cloudflare Workers (`wrangler.toml`/`.json`/`.jsonc`) is
now a recognized stack with a matching `validation-matrix.json` entry, in
both the real file and the neutral scaffold template — a fourth, previously
undiscovered hand-duplication of the same catalog, found and fixed as part
of this work. `internal/pose/discovery.go`'s `hasProjectManifest` was
deliberately left out of scope (Decision 1) — a different-purpose function
the follow-up's file name pointed at but that does not do equivalent work.

### Files and modules changed
- `pose-mcp/internal/cli/stack_manifest.go` (new): shared classification.
- `pose-mcp/internal/cli/stack_manifest_test.go` (new): tests.
- `pose-mcp/internal/cli/index.go`: `scanModules` uses the shared function.
- `pose-mcp/internal/cli/validate.go`: `discoverValidationModules` uses it.
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go`: added the
  `cloudflare-workers` entry to the neutral template literal.
- `.pose/indexes/validation-matrix.json`: added the `cloudflare-workers`
  stack entry.

### Validation executed
- Command: `go -C pose-mcp test ./...`
- Result: SUCCESS
- Command: `go -C pose-mcp vet ./...` / `gofmt -l .`
- Result: clean

### Residual risks
- None beyond the accepted Known gap (minimal Cloudflare check depth).

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

- [open] `internal/pose/discovery.go`'s `hasProjectManifest` carries its
  own independent marker list (Decision 1) — if it drifts noticeably from
  `stackForManifestFile`'s coverage in the future, worth revisiting whether
  they should share a base list while keeping their distinct
  presence-vs-classification semantics.
- [open] a deeper Cloudflare Workers check set (bundling/type-checking,
  not just `wrangler --version`) once real usage shows what a meaningful
  gate looks like for that stack.
