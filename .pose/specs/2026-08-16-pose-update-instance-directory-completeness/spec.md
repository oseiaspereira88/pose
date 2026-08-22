---
slug: pose-update-instance-directory-completeness
status: done
created_at: 2026-08-16
completed_at: 2026-08-16
supersedes:
depends_on:
priority: 0
components: cli, update
delivers: capability:pose-mcp
---

# Spec: pose-update-instance-directory-completeness

---

## 1. Intent

### Goal
Make a plain `pose update` (no `--force`) create every directory the
instance contract (`instanceDirs`, `init.go`) requires — not the
hand-picked 4-directory subset it created before this fix.

### Business value
Discovered live updating real external repositories to v1.4.1:
`storageclose` (this product's own oldest tracked adopter instance) ran
`pose update`, reported `Result: SUCCESS — POSE updated to engine
v1.4.1-dev`, and its very next `pose check --strict` failed with `Broken
reference: ".pose/assessments" (source: AGENTS.md)`. `cmdUpdate`'s
non-`--force` path seeded only `.pose/roadmaps`, `.pose/changelogs/
unreleased`, `.pose/reports/history` and `.pose/feedback` by hand — a
subset of the 14 directories `cmdInit` (and therefore `cmdInstall`, which
calls it) actually guarantees. `.pose/assessments` was one of the ten
missing. This is the same failure shape `pose-upgrade-path-audit-fixes`
(R2/R3) already fixed for `.pose/policy`, `.pose/review-profiles` and the
computed indexes — this spec closes the same gap for the plain directory
contract those fixes didn't touch.

### Constraints
- No public contract change.
- `ensureManagedDirSafe`'s symlink-rejection safety (which the prior
  4-directory loop already used) must be preserved — no regression to a
  plain `os.MkdirAll`.
- Idempotent: an instance that already has all 14 directories is
  unaffected (every directory creation is already skip-if-present).

### Non-goals
- Two further, real gaps found live while updating the same batch of
  external repositories, deliberately NOT fixed by this spec (see Known
  gaps): (a) module-metadata discovery adding a `"."` root entry alongside
  a pre-existing, differently-keyed root alias (`codass`, `micr-omega`
  both hit this); (b) discovery not respecting `.gitignore` (`harne8`'s
  entirely-gitignored `graphforge/` tree got pulled into module-metadata
  discovery). Both are real, but distinct root causes from this spec's
  finding and each deserves its own investigation rather than a rushed
  fix bundled into an unrelated directory-seeding spec.

---

## 2. Requirements

### Functional
- R1: When `pose update` runs without `--force`, it shall create every
  directory in `instanceDirs` (init.go) that is absent — the same
  directory contract `pose install`/`cmdInit` already guarantees for a
  fresh instance.

### Non-functional
- Every directory creation stays behind `ensureManagedDirSafe`'s existing
  symlink-rejection check.

### Security
None — no new write target, only a wider set of already-safe directory
creations.

### Compatibility
An instance already carrying all 14 directories (the common case for any
instance installed after `pose-scaffold-index-template-neutralization`
shipped `cmdInit`'s current directory list) sees zero behavior change.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/maintenance.go` (`cmdUpdate`'s non-`--force`
  directory-seeding loop now iterates `instanceDirs` instead of a
  hand-picked subset)
- `pose-mcp/internal/cli/stack_seed_test.go` (regression coverage)

### Artifacts
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/stack_seed_test.go

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
None.

### Technical risks
None beyond what the existing `ensureManagedDirSafe` symlink guard already
accepts — a strict superset of directories created via the exact same,
already-reviewed function.

---

## 4. Tasks

### Planning
- [x] Reproduced against a real update run (storageclose) before touching
      code.
- [x] Traced the root cause to `cmdUpdate`'s own hand-picked 4-directory
      list versus `cmdInit`'s authoritative 14-directory `instanceDirs`.

### Implementation
- [x] Replaced the hand-picked list with `instanceDirs`, reusing the
      existing `ensureManagedDirSafe` safety wrapper unchanged.

### Validation
- [x] Run the mandatory checks (below).

---

## 5. Decisions

### Decision 1
- Date: 2026-08-16
- Context: Two ways to close the gap: (a) add `.pose/assessments`
  specifically to the hand-picked list (the one directory that actually
  broke `storageclose`); (b) reuse the full, already-canonical
  `instanceDirs` list `cmdInit` maintains.
- Decision: (b).
- Rationale: (a) fixes today's one observed symptom and leaves the same
  class of gap for the next directory `AGENTS.md`/machinery comes to
  reference that the hand-picked list doesn't happen to include — the
  exact "enumerate by hand" gap shape already on record elsewhere in this
  repository's own follow-up history. (b) removes the second, independent
  list entirely instead of growing it.
- Consequences: `cmdUpdate`'s non-`--force` path now creates the same 14
  directories `cmdInstall` does, by construction, not by two lists staying
  manually in sync.

---

## 6. Validation

### Strategy
Reproduced against the real binary on a real affected instance
(storageclose) before fixing; regression-tested against a synthetic
fixture with `.pose/assessments` and `.pose/adr` removed to simulate the
same gap deterministically.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./... -count=1`
- Scope: whole module
- Expected: PASS, including new
  `TestUpdateSeedsEveryInstanceDirectoryNotOnlyAHandpickedFour`

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
- Date: 2026-08-16
- Environment: linux/amd64, go1.26, local dev checkout of pose-dist.
- Manual: re-ran `pose update` against the real storageclose instance
  after the fix — `.pose/assessments` created, `pose check --strict`
  clean.

### Results summary
- Successes: all deterministic checks green; manual repro confirms fixed.
- Failures: none.
- Warnings: none.

### Requirement trace
- R1 [satisfied] check:TestUpdateSeedsEveryInstanceDirectoryNotOnlyAHandpickedFour

### Known gaps
- Module-metadata root-duplication (a `"."` entry added alongside a
  pre-existing, differently-keyed root alias) — observed live in `codass`
  and `micr-omega`, corrected by hand in each instance, not fixed in code
  by this spec.
- Discovery does not respect `.gitignore` — observed live in `harne8`
  (`graphforge/`, entirely gitignored, got module-metadata entries),
  corrected by hand in that instance, not fixed in code by this spec.

---

## 7. Final Report

### Delivered scope
`pose update` (no `--force`) now creates every directory `instanceDirs`
requires, closing the gap that let `storageclose` (and potentially any
older or partially-seeded instance) report update success while its own
next strict gate failed on a directory nothing had created.

### Files and modules changed
See Artifacts above.

### Validation executed
- Command: `go -C pose-mcp build ./... && go -C pose-mcp vet ./... && go -C pose-mcp test ./... -count=1 && gofmt -l pose-mcp`
- Result: all green
- Command: `pose validate --tolerant --module pose-mcp --report`
- Result: `Result: SUCCESS`
- Manual: re-run against the real, previously-affected storageclose instance
- Result: fixed, confirmed

### Residual risks
See Known gaps above — two independent, real gaps deliberately left for
their own future investigation.

### Follow-ups

- [open] Module-metadata discovery should recognize when a pre-existing entry (any key that does not resolve to a real subdirectory) already represents the project root before adding a new `"."` entry for the same physical directory — needs a real design decision (a non-resolving key could be a deliberate root alias or genuinely stale data; these are not distinguishable without more signal than discovery currently has). (owner:@pose-maintainers crit:medium review:2026-11-16)
- [open] Module/stack discovery (all three walkers: discoverValidationModules, scanModules, internal/pose's capability discovery) should respect `.gitignore` — an entirely-gitignored directory tree (observed: harne8's `graphforge/`) should never be discovered as a governed module. (owner:@pose-maintainers crit:medium review:2026-11-16)
