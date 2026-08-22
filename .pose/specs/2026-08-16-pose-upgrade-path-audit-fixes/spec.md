---
slug: pose-upgrade-path-audit-fixes
status: done
created_at: 2026-08-16
completed_at: 2026-08-16
supersedes:
depends_on:
priority: 0
components: cli, install, update, doctor, validate, index
delivers: capability:pose-mcp
---

# Spec: pose-upgrade-path-audit-fixes

---

## 1. Intent

### Goal
Fix the 11 findings from an audit of `pose update`/`pose install` run across
seven real repositories (audio-relay, audio-relay-android, harne8, pose-dist
itself, storageclose, codass, micr-omega; 35 trials: `--locale en`/`pt-BR`,
each with and without `--force`, plus a bare `pose install .`) on the freshly
cut v1.4.0 release — 2 critical, 3 high, 3 medium, 3 low severity.

### Business value
Every finding was reproduced against real, currently-installed POSE
instances, not synthetic fixtures. Two are silent-failure shapes an operator
would not detect until much later: a plain `pose update` could report
`Result: SUCCESS` while leaving an old instance with broken references
(`.pose/assessments`, `spec-graph.json`, `extensions.lock.json` referenced
by its own just-refreshed manuals but never seeded), undetected by `pose
doctor`; and `MergeManagedDoc` could silently discard an operator's own
hand-written customization outside the sections POSE marks
`instance-owned`, with no warning telling them to look at the
`.pose-backup` copy that *was* written. One finding (H3) reproduces on this
repository's own dogfooding instance today: `pose update --force`/`pose
install` on `pose-dist` itself discovers
`examples/brownfield-kits/direct-adoption/fixture/service`'s illustrative
`go.mod` as a real module, which invalidates the review attestation of
every already-closed spec whose scope touches `module-metadata.json` (3 to
22 specs across the audit's trials) — silently narrowing the product's own
release-integrity guarantee every time a maintainer reinstalls.

### Constraints
- No public contract change beyond what each fix requires: flags, exit
  codes and file layout are unchanged; only behavior that was already
  documented (or clearly unintended) is corrected.
- `MergeManagedDoc`'s existing contract (instance-owned sections always
  survive verbatim; unknown local sections are appended, never dropped)
  must not regress for the common same-locale case — every existing test
  in `managed_docs_test.go` must keep passing unmodified in substance.
- No change to the provenance-digest/review-attestation mechanism itself
  (see Decision 2 — deliberately out of scope).

### Non-goals
- Full transactional rollback of `pose install`/`pose update` on a
  post-write gate failure. This is an already-accepted, already-documented
  product decision (spec `pose-install-gate-failure-recovery-notice`,
  github.com/oseiaspereira88/pose#18) — recovery is `.pose-backup` copies
  plus `git status`/`git diff`, deliberately not a snapshot/restore engine.
  This spec narrows *why* the final gate fails (a pre-flight check now says
  plainly when it is pre-existing debt, not something this run caused) —
  see Decision 1 — without reopening that boundary.
- Redesigning the provenance digest to exempt purely-additive
  `module-metadata.json` changes from invalidating unrelated specs' review
  attestations. This is core delivery-integrity/security machinery
  (`ProvenanceDigest` over the whole graph, by design — see
  `pose-gate-closeout-procedure` session knowledge); H3's fix removes the
  *trigger* (a fixture wrongly discovered as a real module) rather than
  changing what a real module-metadata change is allowed to do. See
  Decision 2.
- A general repo-level exclusion-config mechanism for module discovery.
  `testdata`/`fixture`/`fixtures` are hardcoded, universally-recognized
  synthetic-content conventions (`testdata` is the Go toolchain's own
  name); a configurable exclusion list is a larger, separately-scoped
  feature if a real case ever needs more than these three names.
- A full Android/Kotlin validation profile (checks, `validation-matrix.json`
  entries). Only discovery/classification (`domain: "android"` instead of
  the generic `"java"`) is in scope, consistent with how Cloudflare Workers
  detection landed in `pose-validation-scanner-consolidation` before its
  own validation profile.

---

## 2. Requirements

### Functional

**Critical**
- R1: When `pose update`/`pose install` (with or without `--force`) merges
  AGENTS.md/POSE.md and content outside an `instance-owned` section is
  overwritten, the command shall write a `.pose-backup` copy and print an
  explicit warning naming the file — not only the existing generic
  "merged … preserved" line, which reads as if nothing was lost.
- R2: When `pose update` runs without `--force` on an instance whose
  `.pose/policy`, `.pose/review-profiles`, or `.pose/indexes/{spec-graph,
  extensions.lock}.json` are absent, the command shall seed them
  (additive-only, same as a fresh install) before reporting success.
- R3: When `pose doctor` runs on an instance whose `.pose/schema-version`
  exists but any of `.pose/policy/{delivery,artifacts}.json` or
  `.pose/indexes/{spec-graph,extensions.lock}.json` is missing, it shall
  report a `warn`-level `instance.config-completeness` finding naming the
  missing subsystems.

**High**
- R4: When `pose update`/`pose install`/`pose extension install` resolve a
  document or machinery locale, an explicit `--locale en` request shall be
  honored even when the existing instance is currently in a different
  locale — a caller's own unrequested default of `"en"` shall continue to
  fall through to auto-detection.
- R5: When a `pose update`/`pose install` locale switch merges AGENTS.md/
  POSE.md across locales, a local section whose heading translates to a
  canonical section in the target locale shall be recognized as that same
  section — refreshed or preserved per its `instance-owned` status — not
  appended as an unrelated duplicate.
- R6: When `pose install`'s post-install `check --strict` gate fails, and
  the same gate already failed against the target before this run wrote
  anything, the failure output shall say so explicitly instead of reading
  as something this run caused.
- R7: Repository/module discovery (`pose index`, `pose validate`, `pose
  install`/`init`'s module-metadata seeding, and `internal/pose`'s
  capability discovery) shall exclude `testdata`, `fixture` and `fixtures`
  directories.

**Medium**
- R8: When discovery classifies a Gradle/Maven module (`build.gradle`,
  `build.gradle.kts`, `pom.xml`) that also carries an `AndroidManifest.xml`
  at its conventional location, it shall report domain `"android"` instead
  of the generic `"java"`.
- R9: When `pose doctor` runs, it shall report a `warn`-level
  `module-metadata.orphan-entries` finding naming any `module-metadata.json`
  entry whose path does not resolve to a real directory in the project
  (excluding the root key).

**Low**
- R10: `pose install`'s locale-not-found message shall not fire for the
  empty string produced by auto-detection — only for a real, unresolvable
  explicit request.
- R11: `pose update --force`/`pose install` rerun over an already-installed
  target shall recover the project's declared name/id from its existing
  AGENTS.md before falling back to the current directory's basename.

### Non-functional
- Every fix is additive/corrective only: no existing `.pose-backup`,
  policy, review-profile, spec, roadmap, report or knowledge file is ever
  overwritten or deleted by any of these changes.
- `pose install`'s new pre-flight check (R6) adds one extra `check
  --strict` pass (discarded output) only when `.pose` already exists at the
  target — a fresh install pays nothing extra.

### Security
- No security-sensitive surface touched. R9's orphan-entry check reads
  paths already present in a file the operator controls; no new file
  write, no path escapes `confinedRelativePath`.

### Compatibility
- `resolveDocLocale`/`machineryLocale` gained an `explicit bool` parameter
  (internal, unexported — no public/CLI contract change). Every call site
  was audited and updated (`extension.go`, `install.go`, `maintenance.go`,
  `managed_docs.go`) so a caller passing its own unrequested default
  continues to auto-detect exactly as before; only a caller carrying a
  genuine explicit ask changes behavior (R4).
- `cmdCheck` is unchanged for direct CLI use (`pose check`, git hooks) —
  still resolves locale from the shell. `cmdCheckWithLocale` is a new,
  additive entry point `pose install`'s own post-install gate uses.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/managed_docs.go` — R1, R4, R5: locale-aware
  section merge (`buildHeadingTranslation`,
  `MergeManagedDocAcrossLocale`/`MergeAcrossLocaleDropsLocalContent`),
  `resolveDocLocale`'s explicit-"en" fix, `refreshManagedDocs`'s
  drop-warning.
- `pose-mcp/internal/cli/machinery.go` — R4: `machineryLocale` explicit
  parameter.
- `pose-mcp/internal/cli/install.go` — R1, R4, R5, R6, R10, R11: mirrors
  the locale-aware merge for its own AGENTS.md/POSE.md path, pre-flight
  check, identity recovery, locale-message fix, `commandLocale` now
  reflects an explicit `--locale`.
- `pose-mcp/internal/cli/maintenance.go` — R2, R4: `seedAbsentInstanceConfig`
  wired into the non-`--force` path; `machineryLocale` call updated.
- `pose-mcp/internal/cli/extension.go` — R4: `machineryLocale` call
  updated.
- `pose-mcp/internal/cli/stack_seed.go` — R2: `seedAbsentInstanceConfig`
  (index/module-metadata/policy/review-profiles seeding factored out of
  `cmdInstall` so `cmdUpdate` can call the same additive-only logic).
- `pose-mcp/internal/cli/check.go` — R6: `cmdCheckWithLocale`; also fixes
  an adjacent, previously-unknown bug — the final summary line
  (`Resultado: SUCESSO`/`FALHA`) was unconditionally Portuguese regardless
  of locale, for every direct `pose check` invocation, not only the
  internal one this spec set out to fix.
- `pose-mcp/internal/cli/doctor.go` — R3, R9: two new findings.
- `pose-mcp/internal/cli/validate.go` — R7, R8: `testdata`/`fixture(s)`
  excluded from `discoverValidationModules`; `isAndroidModule`.
- `pose-mcp/internal/cli/index.go` — R7, R8: same exclusion and Android
  classification applied to `scanModules` (kept in sync per
  `pose-validation-scanner-consolidation`'s own consolidation rationale).
- `pose-mcp/internal/pose/discovery.go` — R7: same exclusion in the third,
  independent walker (`internal/pose`'s capability discovery).
- `pose-mcp/internal/cli/cli_test.go`, `managed_docs_test.go`,
  `validate_root_and_nodemodules_test.go` — regression coverage plus two
  pre-existing assertions that had (unknowingly) baked in the
  hardcoded-Portuguese summary bug this spec fixes.
- `pose-mcp/internal/cli/doctor_instance_config_test.go`,
  `install_locale_identity_test.go` — new regression coverage.

### Artifacts
- modified: pose-mcp/internal/cli/managed_docs.go
- modified: pose-mcp/internal/cli/managed_docs_test.go
- modified: pose-mcp/internal/cli/machinery.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/extension.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/cli/validate_root_and_nodemodules_test.go
- created: pose-mcp/internal/cli/doctor_instance_config_test.go
- created: pose-mcp/internal/cli/install_locale_identity_test.go
- created: .pose/knowledge/2026-08-16-decision-log-module-metadata-discovery-invalidates-review-provenance.md

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
None on the public CLI surface (flags, exit codes, output structure keys
for `--json` commands are unchanged). Two new `pose doctor --json` finding
codes are additive: `instance.config-completeness`,
`module-metadata.orphan-entries`.

### Technical risks
- The `en` short-circuit in `resolveDocLocale` depends on every call site
  correctly passing `explicit`. Missed one and R4 silently regresses to
  the old auto-detect-only behavior for that path, not a crash — audited
  all four call sites by grep (`resolveDocLocale(`, `machineryLocale(`)
  before closing this out.
- `buildHeadingTranslation` assumes canonical EN/pt-BR AGENTS.md/POSE.md
  keep the same section count and order (verified: 10/10 and 12/12 today).
  A future edit that adds a section to only one locale's canonical file
  returns `nil` (safe: falls back to literal same-language matching,
  reproducing the pre-fix duplication behavior for that one file until the
  locales are back in lockstep) rather than mismatching sections.
- `testdata`/`fixture`/`fixtures` exclusion is a breaking discovery change
  for any real POSE instance that has (unusually) named a genuine
  deliverable module exactly `fixture`/`fixtures`/`testdata` — accepted:
  this convention is strong enough (and `testdata` specifically already
  invisible to `go build`/`go test`) that a false negative here is far
  less likely than the false positive it fixes.

---

## 4. Tasks

### Planning
- [x] Reproduced all 11 findings against the real binary before writing
      any fix (see the published audit report for the original repro
      matrix across 7 repositories).
- [x] Read the exact code path for each finding before deciding a fix —
      two findings turned out narrower/different than first reported
      once read against source (see Decisions 3 and 4).

### Implementation
- [x] R1: drop-warning parity between `pose install`'s merge path (already
      had it) and `pose update`'s `refreshManagedDocs` (didn't).
- [x] R2: `seedAbsentInstanceConfig` factored out, called unconditionally.
- [x] R3, R9: two new `pose doctor` findings.
- [x] R4: `resolveDocLocale`/`machineryLocale` explicit parameter, all
      call sites updated.
- [x] R5: `buildHeadingTranslation` + locale-aware merge, wired into both
      `refreshManagedDocs` and `cmdInstall`.
- [x] R6: pre-flight `check --strict` + honest failure message.
- [x] R7: `testdata`/`fixture`/`fixtures` excluded in all three walkers.
- [x] R8: `isAndroidModule`, wired into `validate.go` and `index.go`.
- [x] R10: locale `""` no longer triggers the not-available message.
- [x] R11: identity recovery before defaulting to the directory basename.

### Validation
- [x] Run the mandatory checks (below) — all green.
- [x] Re-ran the exact repro for every finding against the rebuilt binary
      (not only the new Go unit tests) — see Execution log.

---

## 5. Decisions

### Decision 1
- Date: 2026-08-16
- Context: H2/R6 — `pose install`'s post-install gate failure leaves
  already-written files with no rollback, by an existing, deliberate
  design decision (issue #18). The audit found this triggers on causes
  unrelated to the install/update itself (pre-existing governance debt in
  the target) in every one of its trial reproductions.
- Options considered: (a) leave it exactly as documented, treat as
  "already resolved, not a new finding"; (b) build transactional
  rollback; (c) add a pre-flight check that detects whether the target
  already fails the same gate before any write, and say so plainly in the
  final failure message.
- Decision: (c).
- Rationale: (a) ignores that the audit's own evidence shows the
  documented decision's blast radius is wider than its own follow-up
  anticipated ("decide whether the index step should warn rather than
  fail" — this generalizes that exact question). (b) rebuilds a large,
  already-rejected mechanism and risks new bugs in exchange for solving a
  problem (opaque causation) that doesn't require it. (c) directly answers
  "is this failure honest about what did not happen" — the follow-up's own
  suggested resolution — with a small, safe, additive check.
- Consequences: `pose install` now runs `check --strict` up to twice on an
  already-installed target (once silently before any write, once for real
  after) — negligible cost, no change to a fresh install. Full rollback
  remains explicitly out of scope (see Non-goals) — this does not reopen
  that decision, it only makes the existing failure mode legible.

### Decision 2
- Date: 2026-08-16
- Context: H3 — `pose-dist`'s own `examples/brownfield-kits/.../fixture/
  service/go.mod` is discovered as a real module, and *any* change to
  `module-metadata.json` (even purely additive) recomputes the global
  provenance digest and supersedes unrelated specs' review attestations.
- Options considered: (a) fix only the trigger (exclude fixture/testdata
  directories from discovery); (b) additionally scope the provenance
  digest so a module-metadata change only invalidates reviews whose scope
  actually includes the changed module.
- Decision: (a) only.
- Rationale: (b) is a change to core delivery-integrity/security machinery
  — the digest is deliberately computed over the whole graph (see session
  knowledge `pose-gate-closeout-procedure`: "o digest é global do grafo,
  não por spec"), and narrowing it is a real design question (what counts
  as "unrelated," whether a narrower digest weakens the guarantee it
  exists to provide) that deserves its own spec, review and explicit
  trade-off discussion — not a side effect of fixing a false-positive in
  module discovery. (a) alone removes the actual defect (a fixture
  wrongly treated as a real module) without touching that boundary.
- Consequences: a *genuine* module-metadata change (a real module added,
  removed or reclassified) still invalidates every closed spec's review
  attestation that touches it, exactly as designed today — reseal/reattest
  is still required after such a change, same as documented in
  `pose-gate-closeout-procedure`. Narrowing that is left for a future spec
  if it proves worth pursuing.

### Decision 3
- Date: 2026-08-16
- Context: The original audit (C1) reported `pose update` losing a
  hand-edited AGENTS.md customization "with no backup, unlike machinery
  files." Reading `refreshManagedDocs` before fixing anything showed it
  already wrote a `.pose-backup` unconditionally on any change — verified
  empirically (backup file present, containing the dropped line) before
  writing R1.
- Options considered: (a) trust the original report and add a backup
  mechanism that already existed; (b) re-verify against source and the
  real binary, then scope the fix to what was actually missing.
- Decision: (b).
- Rationale: (a) would have added a redundant, dead second backup path and
  missed the real gap — the operator is never *told* content was dropped
  (the log says "merged … preserved," which is misleading precisely
  because it's true for instance-owned sections and silently false for
  everything else). Fixing the wrong mechanism would not have closed the
  actual trust gap.
- Consequences: R1 is a warning-and-consistency fix (matching `pose
  install`'s own already-correct behavior for the identical scenario), not
  a new backup mechanism. The severity of this finding is downgraded from
  "data loss" to "misleading silence" in this spec's own framing, though
  the audit report's original severity ranking (written before this
  investigation) is left as published.

### Decision 4
- Date: 2026-08-16
- Context: H1 (`--locale` ignored/inconsistent without `--force`) turned
  out to be two distinct root causes, not one: (a) `resolveDocLocale`
  special-cased "en" as never-explicit, so a genuine `--locale en` request
  fell through to auto-detection; (b) even once (a) is fixed, a locale
  switch's section merge matched local headings literally, so a
  translated section read as new content and got appended rather than
  recognized as the same section.
- Options considered for (a): threading an explicit boolean vs. having
  every caller stop using `"en"` as its own unrequested default (e.g.
  switch defaults to `""`).
- Decision: explicit boolean parameter, not changing what `""`/`"en"`
  callers already pass.
- Rationale: several call sites (`cmdInstall`'s own locale variable
  defaults to `"en"`) have this baked in for reasons unrelated to this fix
  (rendering docs when `docsPrefix` is computed downstream expects `"en"`,
  not `""`, per its own `if locale != "en"` check). Re-deriving every
  caller's default would touch more surface for no behavioral gain over a
  single explicit flag threaded through the two low-level functions.
- Consequences: `resolveDocLocale`/`machineryLocale` carry one extra
  parameter each; every call site was individually audited (see Technical
  risks) rather than relying on a type system guarantee.

---

## 6. Validation

### Strategy
Every fix was (1) reproduced against the real `pose` binary before being
fixed, using throwaway fixtures under a scratch directory — never a live
repository; (2) fixed at the root cause identified by reading the actual
code path, not the audit report's inference alone (see Decisions 3 and 4);
(3) re-verified against the rebuilt binary with the same repro; (4) given a
Go regression test exercising the same scenario through the public
`cmd*`/exported-helper surface, not just the throwaway shell repro.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./... -count=1`
- Scope: every package (`internal/cli`, `internal/pose`, `internal/mcpserver`,
  `internal/scaffold`, `internal/scaffold/distpolicy`, `internal/usage`,
  `internal/version`, `internal/observability`)
- Expected: PASS, zero failures — including 9 new tests
  (`TestResolveDocLocaleHonorsExplicitEnglish`,
  `TestRefreshManagedDocsSwitchesLocaleWithoutDuplicating`,
  `TestRefreshManagedDocsWarnsAndBacksUpDroppedContent`,
  `TestDiscoverValidationModules_IgnoresFixtureDirectories`,
  `TestDiscoverValidationModules_ClassifiesAndroidSeparatelyFromJava`,
  `TestDoctorFlagsIncompleteInstanceConfig`,
  `TestDoctorFlagsOrphanModuleMetadataEntries`,
  `TestInstallDoesNotWarnOnAutoDetectedLocale`,
  `TestForcedUpdatePreservesProjectIdentityAcrossRename`,
  `TestInstallWarnsWhenTargetAlreadyFailsBeforeAnyWrite`) and 2 corrected
  pre-existing assertions (`TestCheckNativeParityAndSchemaFailures`,
  `TestNativeCommandLocaleMessagesAndStableAnchors`) that had unknowingly
  depended on the hardcoded-Portuguese summary-line bug this spec also
  fixes as an adjacent discovery (see Technical Plan).

#### Lint
- Command: `go -C pose-mcp vet ./...` and `gofmt -l .`
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
  repository) — the audit's own test binary, rebuilt after every fix.
- Manual repros re-run against the rebuilt binary for R1, R2 (+doctor
  detection before/after), R4 (locale-ignored and locale-duplicated cases,
  4 combinations), R5, R6, R7 (isolated copy of this repository itself),
  R8, R9 (synthetic ghost + mis-cased entries), R10, R11 (directory
  rename) — every one confirmed fixed by direct observation of command
  output/file state, not inferred from the unit tests alone.

### Results summary
- Successes: `go build ./...`, `go vet ./...`, `gofmt -l .` (empty), `go
  test ./... -count=1` (all packages) — all green. Every one of the 11
  findings' manual repro confirmed fixed against the rebuilt binary.
- Failures: none.
- Warnings: none.

### Requirement trace
- R1 [satisfied] test:TestRefreshManagedDocsWarnsAndBacksUpDroppedContent
- R2 [satisfied] test:TestDoctorFlagsIncompleteInstanceConfig (seeding half)
- R3 [satisfied] test:TestDoctorFlagsIncompleteInstanceConfig (finding half)
- R4 [satisfied] test:TestResolveDocLocaleHonorsExplicitEnglish
- R5 [satisfied] test:TestRefreshManagedDocsSwitchesLocaleWithoutDuplicating
- R6 [satisfied] test:TestInstallWarnsWhenTargetAlreadyFailsBeforeAnyWrite
- R7 [satisfied] test:TestDiscoverValidationModules_IgnoresFixtureDirectories
- R8 [satisfied] test:TestDiscoverValidationModules_ClassifiesAndroidSeparatelyFromJava
- R9 [satisfied] test:TestDoctorFlagsOrphanModuleMetadataEntries
- R10 [satisfied] test:TestInstallDoesNotWarnOnAutoDetectedLocale
- R11 [satisfied] test:TestForcedUpdatePreservesProjectIdentityAcrossRename

### Known gaps
- H2/R6 narrows the failure message; it does not make `pose install`/
  `pose update` transactional. See Non-goals and Decision 1.
- H3/R7 removes the false-positive trigger; a genuine, intentional
  module-metadata change still invalidates unrelated closed specs' review
  attestations by design. See Non-goals and Decision 2.
- The audit's L1 finding (gate diagnostics following system `$LANG`) is
  fixed for `pose install`'s own post-install gate and for `pose check`'s
  previously-unconditional-Portuguese summary line; `pose update` does not
  independently invoke `cmdCheck`, so it was not a second instance of the
  same gap.

---

## 7. Final Report

### Delivered scope
All 11 findings from the pre-v1.4.0-release upgrade-path audit (2
critical, 3 high, 3 medium, 3 low) are fixed and regression-covered, plus
one adjacent bug discovered while fixing R6 (`pose check`'s final summary
line was unconditionally Portuguese regardless of locale, for every direct
invocation — not only the internal call this spec set out to fix). Two
findings (H2, H3) are narrowed rather than architecturally resolved by
deliberate, documented decision — see Decisions 1 and 2 and Non-goals.

### Files and modules changed
See Artifacts above.

### Validation executed
- Command: `go -C pose-mcp build ./... && go -C pose-mcp vet ./... && go -C pose-mcp test ./... -count=1 && gofmt -l pose-mcp`
- Result: all green
- Command: `pose validate --tolerant --module pose-mcp --report`
- Result: `Result: SUCCESS`
- Manual: every finding's original repro re-run against the rebuilt binary
- Result: all 11 confirmed fixed

### Residual risks
See Technical risks (§3) and Known gaps (§6) above — all accepted as
deliberate scope boundaries, not oversights.

### Follow-ups

- [open] Narrowing the provenance digest so a purely-additive `module-metadata.json` change does not invalidate unrelated closed specs' review attestations (Decision 2's accepted boundary) — worth its own spec if the review-reseal cascade this causes proves costly in practice beyond this one fixture-discovery case. (owner:@pose-maintainers crit:medium review:2026-11-20)
- [open] A configurable per-repository exclusion list for module discovery, if a real case ever needs more than the hardcoded `testdata`/`fixture`/`fixtures` names this spec adds (Non-goals). (owner:@pose-maintainers crit:low review:2026-11-20)
- [open] A full Android/Kotlin validation profile (`validation-matrix.json` checks) now that discovery classifies the domain correctly — same deferred-scope pattern `pose-validation-scanner-consolidation` used for Cloudflare Workers. (owner:@pose-maintainers crit:low review:2026-11-20)
- [wont-do: already a deliberate, documented decision, see Non-goals and Decision 1] Full transactional rollback of `pose install`/`pose update` on a post-write gate failure.
