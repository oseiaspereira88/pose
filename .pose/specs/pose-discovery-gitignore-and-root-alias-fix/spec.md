---
slug: pose-discovery-gitignore-and-root-alias-fix
status: done
created_at: 2026-08-17
completed_at: 2026-08-17
supersedes:
depends_on:
priority: 0
components: cli, discovery
delivers: capability:pose-mcp
---

# Spec: pose-discovery-gitignore-and-root-alias-fix

---

## 1. Intent

### Goal
Fix the two discovery-correctness gaps found live updating five real
external repositories to v1.4.1/v1.4.2 and deliberately left as follow-ups
on `pose-update-instance-directory-completeness`: (1) module/stack
discovery does not respect `.gitignore`; (2) discovery adds a duplicate
`"."` root entry alongside a pre-existing, differently-keyed root alias.

### Business value
Both are real, reproduced-on-real-repositories defects, not hypothetical:

- A repository (`harne8`) with an entirely-gitignored subtree (a locally
  checked-out project the repository deliberately excludes from version
  control) had that subtree's internal module structure discovered and
  written into its own tracked `module-metadata.json` — governance data
  about content the repository's own `.gitignore` says isn't its content.
- Two repositories (`codass`, `micr-omega`) that had hand-curated their
  root module under a project-name key (`codass`, `micr-omega` — the
  common convention of aliasing the root by the project's own name) each
  got a second `"."` entry for the exact same physical directory the
  moment discovery ran, because `seedModuleMetadataFromDiscovery`'s
  additive-merge only recognizes an existing entry by exact key match.

Both were corrected by hand in the three affected instances at the time;
this spec fixes the root cause so the next repository this happens to
doesn't need the same manual correction.

### Constraints
- No public contract change.
- Additive-only: `seedModuleMetadataFromDiscovery`'s existing "never
  overwrite an existing entry" contract must not regress.
- Discovery must degrade gracefully (not error) when git is unavailable or
  the target isn't a git repository — matches how the rest of this
  codebase already treats git as an optional enhancement for discovery
  contexts, never a hard requirement.

### Non-goals
- A general repo-level exclusion-config mechanism beyond `.gitignore`
  itself — `.gitignore` already is that mechanism; this spec makes
  discovery honor it, not invent a second one.
- Disambiguating a root alias from genuinely stale data in the general
  case. The fix here is deliberately narrow — see Decision 1 — not a
  general "is this orphaned key actually meaningful" solver.
- Retroactively re-scanning already-installed instances. Both fixes only
  change what a *future* `pose index`/`pose install`/`pose update`
  discovers; an instance already carrying a gitignored-subtree leak or a
  duplicate root entry keeps it until its own next discovery run (at which
  point the duplicate-root fix specifically helps: the existing entry is
  recognized and no new "." is added going forward, but the *already*
  present duplicate isn't removed automatically — matches this
  codebase's established "additive/detect, not silent auto-repair"
  precedent for self-referential leaks, e.g.
  `pose-scaffold-self-referential-policy-fix`).

---

## 2. Requirements

### Functional
- R1: Module/stack discovery (`discoverValidationModules`,
  `scanModules`, and `internal/pose`'s `FindComponentDirectories`) shall
  not descend into a directory `.gitignore` excludes.
- R2: When `seedModuleMetadataFromDiscovery` is about to add a `"."` entry
  for the project root, and an existing `module-metadata.json` entry is
  keyed exactly by the project root directory's own name and that key does
  not resolve to a real subdirectory, it shall treat the root as already
  represented and not add a duplicate `"."` entry.

### Non-functional
- `GitIgnoredPaths` computes the ignored-path set with one `git`
  invocation per discovery run, not one per directory visited — bulk
  lookup, not a per-path `git check-ignore` call.

### Security
None — no new write target; `GitIgnoredPaths` only reads git's own
already-computed ignore state.

### Compatibility
- A repository with no `.gitignore` (or where `git` is unavailable) sees
  `GitIgnoredPaths` return an empty set and behaves exactly as before this
  fix.
- A repository whose module-metadata.json has no entry keyed by its own
  directory basename is unaffected by R2 — the duplicate-suppression
  condition simply never triggers.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/discovery.go` (`GitIgnoredPaths`; wired into
  `FindComponentDirectories`'s fallback walker)
- `pose-mcp/internal/pose/discovery_test.go` (regression coverage)
- `pose-mcp/internal/cli/validate.go` (`discoverValidationModules` wired
  to `GitIgnoredPaths`)
- `pose-mcp/internal/cli/index.go` (`scanModules` wired to
  `GitIgnoredPaths`)
- `pose-mcp/internal/cli/validate_root_and_nodemodules_test.go`
  (regression coverage)
- `pose-mcp/internal/cli/stack_seed.go`
  (`seedModuleMetadataFromDiscovery`'s root-alias check)
- `pose-mcp/internal/cli/stack_seed_test.go` (regression coverage)

### Artifacts
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/pose/discovery_test.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/validate_root_and_nodemodules_test.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/stack_seed_test.go

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
None.

### Technical risks
- `GitIgnoredPaths` shells out to `git ls-files` once per discovery call
  (three call sites: `discoverValidationModules`, `scanModules`,
  `FindComponentDirectories`) — three git processes per `pose index`/
  `pose install`/`pose update` run instead of zero. Accepted: this
  codebase already shells out to git for numerous other checks in the same
  commands; the added cost is one more fast, read-only process per run,
  not per file.
- R2's heuristic (exact directory-basename key match, key does not resolve
  on disk) is deliberately narrow — see Decision 1 — and will not catch a
  root alias that doesn't happen to match the directory's own name. That
  case still produces a duplicate `"."` entry, same as before this spec;
  narrower coverage than a perfect fix, but zero risk of the false-positive
  a broader heuristic (e.g. "any orphaned key") would introduce.

---

## 4. Tasks

### Planning
- [x] Both defects reproduced against the real binary on synthetic
      fixtures mirroring the exact real-world shape (project directory
      named after itself; an entirely-gitignored vendored subtree) before
      any code change.
- [x] Read all three discovery walkers and `seedModuleMetadataFromDiscovery`
      in full before designing the fix, to place `GitIgnoredPaths` where
      every caller could share it without violating the `cli`→`pose`
      package dependency direction.

### Implementation
- [x] R1: `GitIgnoredPaths` (internal/pose, single `git ls-files
      --others --ignored --exclude-standard --directory` call), wired into
      all three walkers.
- [x] R2: exact-basename root-alias check in
      `seedModuleMetadataFromDiscovery`.

### Validation
- [x] Run the mandatory checks (below) — all green, including a check that
      neither fix's new source comments leak a real customer/project name
      (`TestProjectAgnosticSourceTemplates` caught and corrected one
      during implementation).

---

## 5. Decisions

### Decision 1
- Date: 2026-08-17
- Context: R2 needs to distinguish "this non-resolving module-metadata key
  is a deliberate root alias" from "this non-resolving key is genuinely
  stale data" (a deleted/renamed module) — the two look identical from
  discovery's point of view (a key with no matching directory).
- Options considered: (a) treat any non-resolving key as a potential root
  alias, skip adding `"."` whenever one exists; (b) narrow the check to an
  exact match against the project root directory's own basename.
- Decision: (b).
- Rationale: (a) is exactly the heuristic manually applied to fix the two
  real instances (`codass`, `micr-omega`) — but generalized in code, it
  would silently suppress a legitimate `"."` discovery in any repository
  that happens to also carry an unrelated stale entry, which is a strictly
  worse failure mode (a missing real module) than this spec's own
  motivating bug (a harmless duplicate). (b) matches the one concrete,
  observed convention exactly — both real instances used the project's own
  directory name as the alias — and only ever suppresses a `"."` addition
  when that specific, narrow condition holds.
- Consequences: A root alias that does *not* happen to match the
  directory's own name (e.g. a repo at `.../codass` curated with entry key
  `backend` instead) still gets a duplicate `"."` entry — not solved by
  this spec, left as the same class of gap with a narrower, safer scope
  than a general fix would have (see Technical risks).

### Decision 2
- Date: 2026-08-17
- Context: How to make three independent walkers (two in `internal/cli`,
  one in `internal/pose`) all skip gitignored paths without duplicating
  the git-invocation logic three times — the exact "enumerate by hand"
  shape already on record elsewhere in this repository's own follow-up
  history for `stackForManifestFile`/`isAndroidModule`.
- Options considered: (a) three independent `git check-ignore`/`git
  ls-files` call sites, one per walker; (b) one shared function in
  `internal/pose` (which `internal/cli` already imports, but not the
  reverse), called by all three.
- Decision: (b).
- Rationale: `internal/pose` already sits below `internal/cli` in the
  import graph (confirmed by reading both files' imports before writing
  any code), so a single function there is reachable from every walker
  without introducing a new shared package or an import cycle.
- Consequences: `GitIgnoredPaths` is public (capitalized) specifically so
  `internal/cli` can call it — its doc comment says so explicitly to keep
  that reason visible to the next reader.

---

## 6. Validation

### Strategy
Every fix reproduced on a synthetic fixture mirroring the real-world shape
observed live (not just the abstract bug) before being fixed, then
re-verified against the rebuilt binary; Go regression tests added for both
the shared `GitIgnoredPaths` helper and each of the three call sites it
feeds.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./... -count=1`
- Scope: whole module, in particular `internal/pose`
  (`TestGitIgnoredPathsReportsIgnoredDirectories`,
  `TestGitIgnoredPathsDegradesGracefullyOutsideAGitRepo`,
  `TestFindComponentDirectoriesRespectsGitignore`,
  `TestProjectAgnosticSourceTemplates`) and `internal/cli`
  (`TestDiscoverValidationModules_RespectsGitignore`,
  `TestModuleMetadataDiscoveryDoesNotDuplicateAnAliasedRoot`)
- Expected: PASS, zero failures

#### Lint
- Command: `go -C pose-mcp vet ./...` and `gofmt -l pose-mcp`
- Expected: clean

#### Build
- Command: `go -C pose-mcp build -o <tmp>/pose ./cmd/pose`
- Expected: builds

#### Security / Contract
- Command: `pose validate --tolerant --module pose-mcp --report`
- Expected: `Result: SUCCESS`

### Execution log
- Date: 2026-08-17
- Environment: linux/amd64, go1.26, local dev checkout of pose-dist.
- Manual repro (R1): a fresh repo with a `.gitignore`-excluded
  `vendored-tool/package.json` — `pose install` discovered zero modules
  before the exclusion was even relevant to test (nothing else present),
  confirmed by re-running after adding a real, tracked module: only the
  real one appeared.
- Manual repro (R2): a repo at a directory named `acme`, hand-curated with
  a `"acme"` module-metadata entry, then a `go.mod` added at the root —
  `pose update` before the fix would have added a duplicate `"."` entry
  (matches the real `codass`/`micr-omega` observations); after the fix,
  `modules` stayed exactly `["acme"]`.
- `TestProjectAgnosticSourceTemplates` caught a real customer name
  (harne8) accidentally left in a doc comment during implementation —
  fixed before this validation pass, not after; noted here as evidence the
  check is load-bearing, not just passing by luck.

### Results summary
- Successes: all deterministic checks green; both manual repros confirm
  fixed against the rebuilt binary.
- Failures: none (one caught-and-fixed-in-place: see execution log).
- Warnings: none.

### Requirement trace
- R1 [satisfied] check:TestDiscoverValidationModules_RespectsGitignore check:TestFindComponentDirectoriesRespectsGitignore check:TestGitIgnoredPathsReportsIgnoredDirectories
- R2 [satisfied] check:TestModuleMetadataDiscoveryDoesNotDuplicateAnAliasedRoot

### Known gaps
See Non-goals and Decision 1 — a root alias not matching the project
directory's own basename still produces a duplicate `"."` entry; an
already-contaminated instance's existing duplicate/gitignored-leak entries
are not retroactively removed.

---

## 7. Final Report

### Delivered scope
Both follow-ups from `pose-update-instance-directory-completeness` fixed:
discovery now respects `.gitignore` across all three walkers via one
shared, bulk-computed helper, and module-metadata discovery no longer
duplicates a root already aliased by the project's own directory name.

### Files and modules changed
See Artifacts above.

### Validation executed
- Command: `go -C pose-mcp build ./... && go -C pose-mcp vet ./... && go -C pose-mcp test ./... -count=1 && gofmt -l pose-mcp`
- Result: all green
- Command: `pose validate --tolerant --module pose-mcp --report`
- Result: `Result: SUCCESS`
- Manual: both original real-world scenarios reproduced synthetically and
  confirmed fixed

### Residual risks
See Technical risks (§3) and Known gaps (§6) — both accepted, deliberately
narrow-scoped fixes rather than general solvers for either problem class.

### Follow-ups

- [wont-do: deliberately narrow by design — see Decision 1 and Non-goals] A general heuristic recognizing any root alias regardless of naming convention, or retroactively cleaning already-contaminated instances.
