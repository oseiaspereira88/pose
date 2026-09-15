---
slug: pose-discovery-gitignore-inside-submodules
status: done
completed_at: 2026-09-15
created_at: 2026-09-15
supersedes:
depends_on: pose-discovery-gitignore-and-root-alias-fix
priority: 0
components: pose-mcp
task_type: bugfix
delivers: capability:discovery-honours-submodule-gitignore
---

# Spec: Discovery honours a submodule's own .gitignore

## 1. Intent

### Goal
A path that a submodule's own `.gitignore` excludes shall be treated as ignored
by every discovery walker, exactly as a path the enclosing repository ignores.

### Business value
On harne8, `pose index` produced a `repo-map.json` with one line more than the
same commit indexed in a clean clone: `pose-dist/docs-site/.pytest_cache/README.md`,
a local pytest cache inside the POSE submodule that git ignores. The index
depended on who ran it.

`GitIgnoredPaths` lists ignored paths with one `git ls-files --others --ignored`
at the repository root. Git does not descend into submodules for that command:
measured on harne8, the root reports nothing under `.pytest_cache`, and the same
command run inside `pose-dist` reports it. The walkers do descend into the
submodule, so whatever it ignores reached them as tracked content.

The same set decides which modules `pose validate` and `pose index` discover. A
`go.mod` or `package.json` left in an ignored directory of a submodule became a
governed module. `pose-discovery-gitignore-and-root-alias-fix` made discovery
honour `.gitignore` and does not mention submodules.

### Constraints
- Content a submodule tracks is still discovered.
- An uninitialised submodule, or a repository without git, degrades to the
  current behaviour and never fails discovery.

### Non-goals
- Deciding whether submodule content should be discovered at all; that is
  unchanged.
- Removing entries an already-contaminated instance recorded before the fix.

---

## 2. Requirements

### Functional
- R1: `GitIgnoredPaths` shall report, prefixed with the submodule path, every
  path an initialised submodule ignores, recursively for nested submodules.
- R2: `pose index` module and file discovery shall not report a path ignored
  inside a submodule, and shall still report the submodule's tracked content.
- R3: `pose validate` module discovery shall not discover a module inside a
  directory a submodule ignores.
- R4: An uninitialised submodule shall be skipped, and `GitIgnoredPaths` shall
  return.
- R5: `GitIgnoredPaths` shall report an ignored name byte for byte, including a
  non-ASCII name git would C-quote and a name with a leading space, at the root
  and inside a submodule.

### Non-functional
- One `git ls-files` for the ignored paths and one for the gitlinks per
  repository, not one process per path.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/discovery.go` — `GitIgnoredPaths`, shared by the
  component walker, `pose index` and `pose validate`

### Artifacts
- created: .pose/specs/2026-09-15-pose-discovery-gitignore-inside-submodules.md
- created: .pose/changelogs/unreleased/pose-discovery-gitignore-inside-submodules.md
- modified: pose-mcp/internal/pose/discovery.go
- created: pose-mcp/internal/pose/discovery_submodule_test.go
- created: pose-mcp/internal/cli/discovery_submodule_test.go

### Delivery targets
- capability:discovery-honours-submodule-gitignore module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Git run inside an uninitialised submodule, an empty directory, resolves to
  the enclosing repository, which lists the same gitlink again. Descending into
  it without a check never returns; the walk only enters a submodule that has
  its own `.git`.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Ask each initialised submodule for its ignored paths (R1, R2, R3)
- [x] Increment 2: Skip an uninitialised submodule (R4)
- [x] Increment 3: Read ignored paths NUL-separated (R5)

### Validation
- [x] Each new test fails against the code without the corresponding change

---

## 5. Decisions

### Decision 1
- Date: 2026-09-15
- Context: `git ls-files` has `--recurse-submodules`, which would keep one call.
- Decision: one call per initialised submodule, prefixed with its path.
- Rationale: git 2.55 rejects `--recurse-submodules` combined with `--others`
  (`fatal: ls-files --recurse-submodules unsupported mode`), and untracked
  ignored paths are what this query exists for.

### Decision 2
- Date: 2026-09-15
- Context: submodules could be read from `.gitmodules`.
- Decision: read gitlinks (mode 160000) from the index with `git ls-files --stage`.
- Rationale: the index is what git checks out; `.gitmodules` can name paths
  that no longer exist or omit ones that do.

### Decision 3
- Date: 2026-09-15
- Context: review of PR #115 pointed out that line output C-quotes a
  non-ASCII name (`"caf\303\251/"`), and the per-line `TrimSpace` strips a
  leading space that belongs to the name. Both predate this spec at the root,
  and the submodule walk reuses the same parsing.
- Decision: pass `-z` and split on NUL, without trimming.
- Rationale: the reported path has to equal the directory a walker visits;
  `core.quotePath` is on by default, so the quoted form is what most machines
  produce.

---

## 6. Validation

### Strategy
Build a repository with a submodule whose own `.gitignore` excludes `cache/`,
leave a manifest and a README in `mod/cache/` next to a tracked module in
`mod/tool/`, and read what each walker reports.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-15
- Environment: local, Go 1.26
- Notes: observed on harne8 at 493d377: `repo-map.json` gained
  `pose-dist/docs-site/.pytest_cache/README.md` locally and not in a clean
  clone. Against the code without the fix,
  TestGitIgnoredPathsReportsPathsIgnoredInsideASubmodule fails with an empty
  set, TestScanModules_RespectsGitignoreInsideSubmodule with `mod/cache` as a
  module, and TestDiscoverValidationModules_RespectsGitignoreInsideSubmodule
  with `[mod/cache mod/tool]`. With the `.git` check removed,
  TestGitIgnoredPathsSkipsAnUninitialisedSubmodule fails after 10 s saying
  `GitIgnoredPaths` did not return; without its own deadline the package hit
  the `go test` timeout. On harne8 at 493d377, with the cache still in the
  checkout, `pose index` built from origin/main writes the `.pytest_cache` line
  into `repo-map.json`; built from this branch it leaves `repo-map.json`
  identical to the committed one. With line output,
  TestGitIgnoredPathsKeepsNamesGitWouldQuote fails missing `café/`, ` spaced/`
  and `mod/café/` (the set holds `"caf\303\251/"` and `spaced/`). In a
  repository ignoring `/café/` that holds `café/go.mod`, `pose index` built
  without `-z` writes it into `repo-map.json` and `packages.json`; built with it,
  neither file mentions it and the tracked `app` module is still indexed.

### Results summary
- Successes: R1–R5 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestGitIgnoredPathsReportsPathsIgnoredInsideASubmodule requires mod/cache/ and not mod/tool/; fails with an empty set without the fix>
- R2 [satisfied] <TestScanModules_RespectsGitignoreInsideSubmodule requires no mod/cache module, manifest or README and still requires mod/tool>
- R3 [satisfied] <TestDiscoverValidationModules_RespectsGitignoreInsideSubmodule requires exactly [mod/tool]>
- R4 [satisfied] <TestGitIgnoredPathsSkipsAnUninitialisedSubmodule requires a return within 10 s, the parent's scratch/, and nothing under mod/; fails on the deadline with the check removed>
- R5 [satisfied] <TestGitIgnoredPathsKeepsNamesGitWouldQuote requires café/, " spaced/", mod/café/ and mod/cache/ with core.quotePath forced on; fails on the first three with line output>

### Known gaps
- Nested submodules follow the same recursion, but no test builds one.

---

## 7. Final Report

### Summary
What a submodule ignores is ignored by discovery, so `pose index` no longer
depends on caches left inside a submodule checkout.

### Follow-ups
