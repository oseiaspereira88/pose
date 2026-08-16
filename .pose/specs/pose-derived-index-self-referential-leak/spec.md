---
slug: pose-derived-index-self-referential-leak
status: done
created_at: 2026-08-16
completed_at: 2026-08-16
supersedes:
depends_on:
priority: 0
components: cli, install, update, scaffold, distpolicy
delivers: capability:pose-mcp
---

# Spec: pose-derived-index-self-referential-leak

---

## 1. Intent

### Goal
Stop the embedded scaffold from shipping pose-dist's own computed
`.pose/indexes/{repo-map,services,packages,spec-graph,roadmaps,
delivery-integrity,releases}.json` and `.pose/indexes/extensions.lock.json`
to installed instances — the same leak class issue #17 closed for
`.pose/policy/{delivery,artifacts}.json` and issue #22 closed for
`module-metadata.json`/`validation-matrix.json`, but in the eight index
files neither of those fixes covered.

### Business value
Discovered while closing out `pose-upgrade-path-audit-fixes`: regenerating
the embedded scaffold (`go generate ./internal/scaffold`, a routine
closeout step) diffed `delivery-integrity.json`, `repo-map.json`,
`services.json` and `spec-graph.json` — the embedded copies were stale
snapshots of pose-dist's own real graph (111 specs, real module owners,
this repository's own delivery claims), not neutral defaults. That alone
was latent, not yet a live defect: `pose install` always calls `cmdIndex`
again at the very end of its own run, silently overwriting whatever had
just been seeded from the stale snapshot before an operator could ever see
it.

`pose-upgrade-path-audit-fixes` (R2) changed that. It made a plain `pose
update` (no `--force`) seed `.pose/indexes/*.json` too, to fix a different,
real bug — an old instance whose refreshed manuals already referenced
subsystems nothing had ever seeded. `pose update`'s non-`--force` path
never called `cmdIndex` afterward, so this newly exposed the latent leak on
a path where it was previously unreachable: reproduced empirically —
`pose update` on a fixture project with zero specs of its own left
`spec-graph.json` claiming 111 specs belonging to pose-dist.

### Constraints
- No public contract change: `.pose/indexes/*.json` schemas are unchanged,
  only their *shipped default content* changes, exactly like issue #17 and
  #22.
- Must not regress the scaffold drift guard
  (`TestEmbeddedDistMatchesPoseDist`).
- The neutral placeholder for each excluded file must be schema-valid, so a
  fresh install/update never breaks a reader that touches it before
  `cmdIndex` runs — same requirement `NeutralPolicyTemplates`/
  `NeutralIndexTemplates` already satisfy for the files they cover.
- `pose install`/`pose update` must keep never overwriting an index file
  that already exists (unrelated, pre-existing invariant, must stay
  intact).

### Non-goals
- Retroactively repairing an instance already seeded with pose-dist's real
  graph before this fix (same boundary issue #17's fix drew for
  `delivery.json`/`artifacts.json` — detection via `pose doctor`, not
  silent auto-repair). No `pose doctor` check is added for this in this
  spec: an already-contaminated `spec-graph.json`/`repo-map.json`/etc. is
  self-correcting the moment the operator runs `pose index` (or `pose
  install`, or `pose update --force`) — unlike the policy-file case, there
  is no `enabled: false` gate silently suppressing the problem, so the
  urgency and the fix are different in shape.
- Unifying `distpolicy.SelfReferentialIndexFiles`'s two different neutral-
  content strategies (fully-empty vs. partially-generic-content-preserved)
  into one mechanism. `validation-matrix.json` still needs its `stacks`/
  `deliveryProfiles` subtrees preserved; the eight files this spec adds are
  fully computed, so a fully-empty shell is correct for all of them. Both
  strategies already coexist in `NeutralIndexTemplates()`; this spec adds
  more of the "fully empty" kind, not a new kind.

---

## 2. Requirements

### Functional
- R1: The embedded scaffold shall never ship `.pose/indexes/{repo-map,
  services,packages,spec-graph,roadmaps,delivery-integrity,releases,
  extensions.lock}.json` containing pose-dist's own computed graph, module
  list, delivery claims or release history.
- R2: When `pose install`/`pose update` (with or without `--force`) seeds
  any of the seven `cmdIndex`-computed index files because it was absent,
  the command shall immediately recompute all of them for the target via
  `cmdIndex` — the neutral placeholder must never be a target's final,
  observed state when a fresh computation is possible.
- R3: `extensions.lock.json`'s neutral placeholder (an empty extension
  registry) is not recomputed by anything — it remains the target's actual
  state until the operator runs `pose extension install`, matching
  `module-metadata.json`'s existing `modules: {}` precedent.

### Non-functional
- Every neutral index placeholder must be schema-valid and match the exact
  shape `cmdIndex` itself produces for a target with zero specs, modules,
  roadmaps and releases — verified empirically against a real fresh
  install, not hand-guessed, including the `delivery-integrity.json`
  digests (confirmed deterministic: sha256 over an empty claims/
  change-sets set, identical across two independent fresh installs).

### Security
- No security-sensitive surface touched. `cmdIndex` was already called
  unconditionally by `pose install`; this spec only adds one additional,
  conditional call inside `seedAbsentInstanceConfig` (both call sites),
  gated on whether a computed index file was actually just seeded.

### Compatibility
- An instance that already has all eight files (the overwhelmingly common
  case — any instance installed before this fix already has them, seeded
  or hand-populated) is completely unaffected: the seed loop's existing
  "skip if present" check means neither the new neutral placeholders nor
  the new `cmdIndex` call ever run for it.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go` (exclusion list +
  neutral placeholder source of truth)
- `pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go` (regression
  coverage; one prior assertion reversed — see Decisions)
- `pose-mcp/internal/scaffold/dist/.pose/indexes/*.json` (regenerated
  embedded output)
- `pose-mcp/internal/cli/stack_seed.go` (`seedAbsentInstanceConfig` now
  tracks whether a computed index file was seeded and calls `cmdIndex`)

### Artifacts
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/repo-map.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/packages.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/roadmaps.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/releases.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/extensions.lock.json
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/stack_seed_test.go

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
None. Only the shipped default content of eight files changes; their
schemas, and every command's flags/exit codes, are unchanged.

### Technical risks
- `seedAbsentInstanceConfig` now calls `cmdIndex` (a fuller operation:
  scans the tree, reads specs/roadmaps, builds the delivery graph, reads
  release status) instead of a plain file copy, but only when at least one
  computed index file was actually absent — the common case (already-
  installed instance) pays nothing extra. Accepted: this is the same cost
  `pose install` has always unconditionally paid.
- The neutral placeholders assume `cmdIndex`'s empty-state output shape is
  stable. If a future change to `cmdIndex` alters what it emits for an
  empty project (a new top-level key, a different digest algorithm), the
  placeholders would drift from what `cmdIndex` immediately overwrites
  them with — harmless in practice (the placeholder is corrected within
  the same run whenever R2's condition fires), but worth noting: unlike
  `NeutralPolicyTemplates`, these placeholders don't need to stay accurate
  forever, only until `cmdIndex` next runs.

---

## 4. Tasks

### Planning
- [x] Confirmed the leak with `go generate ./internal/scaffold`'s diff
      while closing out `pose-upgrade-path-audit-fixes`.
- [x] Traced why it was previously latent (`cmdInstall` always calls
      `cmdIndex` at the end) and what changed to expose it (R2 of
      `pose-upgrade-path-audit-fixes` added seeding to `cmdUpdate`'s
      non-`--force` path, which never called `cmdIndex`).
- [x] Read `distpolicy.go` in full before designing the fix — discovered
      `SelfReferentialIndexFiles`/`NeutralIndexTemplates` already exist and
      already closed this exact leak class for two other files (issue
      #22), so this spec extends an established mechanism rather than
      inventing a new one.

### Implementation
- [x] Extended `SelfReferentialIndexFiles` with the eight files.
- [x] Authored their neutral placeholders in `NeutralIndexTemplates()`,
      each verified against a real fresh install's actual `cmdIndex`
      output (not hand-guessed).
- [x] Regenerated the embedded scaffold (`go generate ./internal/scaffold`).
- [x] Wired `cmdIndex` into `seedAbsentInstanceConfig`, gated on whether a
      computed index file was actually seeded.

### Validation
- [x] Run the mandatory checks (below).

---

## 5. Decisions

### Decision 1
- Date: 2026-08-16
- Context: `distpolicy_test.go`'s existing `TestSelfReferentialIndexFilesExcluded`
  asserted `repo-map.json` must stay on the wholesale sync allowlist — a
  prior, deliberate decision by issue #22's fix.
- Options considered: (a) trust the prior assertion and find another
  explanation for the leak; (b) verify empirically whether `repo-map.json`
  is actually safe to sync today, and reverse the prior decision if not.
- Decision: (b) — verified (git diff during `go generate` showed real
  pose-dist module data in the embedded copy), then reversed.
- Rationale: issue #22's fix predates `pose-upgrade-path-audit-fixes`
  exposing this file to the update path. Its authors' reasoning — visible
  in the code comment removed by this spec — most likely held only because
  `cmdInstall` always corrects it moments later; that assumption silently
  stopped holding once a second seed path existed that doesn't correct it.
  Trusting new empirical evidence over an old comment's now-stale
  assumption is the same discipline `pose-upgrade-path-audit-fixes`
  applied to its own Decisions 3 and 4.
- Consequences: `repo-map.json` moves from "always synced" to "neutral
  placeholder, corrected by `cmdIndex`" — behaviorally invisible to every
  caller that already reaches `cmdIndex` (i.e., all of them, after R2),
  and closes the actual leak for the one caller that didn't yet.

### Decision 2
- Date: 2026-08-16
- Context: Two ways to guarantee a target ends up with correct (not just
  neutral-empty) computed index content after seeding: (a) ship a richer
  neutral placeholder that's "good enough" to leave as-is; (b) always
  follow a computed-index seed with a real `cmdIndex` computation.
- Decision: (b).
- Rationale: (a) is not achievable for these files — "correct" content is
  inherently target-specific (this target's own modules, specs, delivery
  claims), so no static placeholder can be "good enough"; only a real scan
  produces it. `cmdIndex` already exists, is idempotent, and `pose
  install` already trusted it unconditionally — reusing it is smaller and
  more consistent than inventing a partial-recompute path.
- Consequences: a target's `.pose/indexes/*.json` are genuinely correct
  immediately after any `pose install`/`pose update` that had to seed them,
  not just "no longer wrong" — strictly better than the neutral-placeholder-
  only approach the policy/module-metadata fixes used, because those files
  have no equivalent one-command recomputation `cmdIndex` already performs
  for the target as a side effect of definitely-needed work.

---

## 6. Validation

### Strategy
Empirical verification against the real binary at every step: read
`cmdIndex`'s actual output for a genuinely fresh, empty project (twice, to
confirm determinism) before authoring a single neutral placeholder;
reproduce the exact contamination scenario from the discovery (`pose
update` on a fixture project) before and after the fix; run the full
existing test suite plus new/updated regression tests.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./... -count=1`
- Scope: whole module, in particular `internal/scaffold/distpolicy`
  (updated `TestSelfReferentialIndexFilesExcluded`,
  `TestNeutralIndexTemplatesAreSchemaValidAndClean`) and
  `internal/scaffold` (`TestEmbeddedDistMatchesPoseDist`, the drift guard)
- Expected: PASS, zero failures

#### Lint
- Command: `go -C pose-mcp vet ./...` and `gofmt -l pose-mcp`
- Scope: whole module
- Expected: clean vet; empty gofmt output

#### Build
- Command: `go -C pose-mcp build -o <tmp>/pose ./cmd/pose`
- Scope: `cmd/pose`
- Expected: builds; binary used for every manual repro below

#### Security / Contract
- Command: `pose validate --tolerant --module pose-mcp --report`
- Scope: pose-mcp
- Expected: `Result: SUCCESS`

### Execution log
- Date: 2026-08-16
- Environment: linux/amd64, go1.26, local dev checkout of pose-dist (this
  repository).
- Manual repro before the fix: fixture project (zero of its own specs),
  policy/review-profiles/some indexes removed to simulate an old instance,
  `pose update` (no `--force`) → `spec-graph.json` ends up with 111 specs
  belonging to pose-dist.
- Manual repro after the fix: identical fixture and steps →
  `spec-graph.json` ends up with 0 specs, matching the fixture's own real
  state. `pose check --strict` clean in both the fixture and a full
  isolated copy of pose-dist itself.

### Results summary
- Successes: `go build ./...`, `go vet ./...`, `gofmt -l .` (empty), `go
  test ./... -count=1` (all packages, including the drift guard) — all
  green. Manual repro confirms the leak is closed.
- Failures: none.
- Warnings: none.

### Requirement trace
- R1 [satisfied] check:TestSelfReferentialIndexFilesExcluded check:TestEmbeddedDistMatchesPoseDist
- R2 [satisfied] manual repro (spec-graph.json 111→0 specs on the fixture)
- R3 [satisfied] check:TestNeutralIndexTemplatesAreSchemaValidAndClean

### Known gaps
- No automated migration for instances already seeded with pose-dist's own
  graph via the exposed `pose update` path before this fix — self-heals on
  the target's next `pose index`/`pose install`/`pose update --force` (see
  Non-goals). The window this could have affected real instances is small:
  `pose-upgrade-path-audit-fixes` (which introduced the exposure) and this
  fix landed in the same closeout session, never independently released.

---

## 7. Final Report

### Delivered scope
Closed the same self-referential-leak class issue #17 (policy) and issue
#22 (module-metadata/validation-matrix) already closed, for the eight
`.pose/indexes/*.json` files neither covered — discovered as a direct,
concrete consequence of `pose-upgrade-path-audit-fixes`'s own R2 exposing
a previously-latent leak on a new code path. Fixed by extending the
existing `distpolicy` exclusion mechanism (not a new one) plus wiring a
real `cmdIndex` recomputation into the seed path so a target ends up with
correct content, not just neutral-empty content.

### Files and modules changed
See Artifacts above.

### Validation executed
- Command: `go -C pose-mcp build ./... && go -C pose-mcp vet ./... && go -C pose-mcp test ./... -count=1 && gofmt -l pose-mcp`
- Result: all green
- Command: `pose validate --tolerant --module pose-mcp --report`
- Result: `Result: SUCCESS`
- Manual: contamination repro before/after on a fixture project and a full
  isolated pose-dist copy
- Result: leak closed, confirmed empirically

### Residual risks
See Technical risks (§3) and Known gaps (§6) above — accepted as
deliberate scope boundaries, not oversights.

### Follow-ups

- [wont-do: no urgency to auto-repair — see Non-goals] Retroactive detection/repair (`pose doctor` check) for an instance already contaminated via the brief exposure window before this fix — self-corrects on the target's next `pose index`/install/update anyway.
