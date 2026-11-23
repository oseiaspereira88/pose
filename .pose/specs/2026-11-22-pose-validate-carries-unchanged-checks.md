---
slug: pose-validate-carries-unchanged-checks
status: in-progress
created_at: 2026-11-22
completed_at:
supersedes:
depends_on: pose-changed-scope-validation, pose-validation-evidence-material-binding
remediates:
priority: 1
components: pose-mcp
task_type: feature
surface: standard
delivers: capability:validate-carries-unchanged-checks
---

# Spec: pose-validate-carries-unchanged-checks

## 1. Intent

### Goal
Let `pose validate` refresh the canonical structured result by running only the checks a change can affect and carrying forward, unchanged and labelled, the results of the checks it cannot affect.

### Business value
In the Harne8 batch of 2026-11-22, every material change to one module required the whole 72-check matrix again. Each run took about 35 minutes with three jobs. Intermittent failures in unrelated suites made roughly one run in three useless. A hooks change confined to `scripts/` needed four full runs in one review round. The canonical record is all-or-nothing today. `--changed-from` already selects the affected modules, but it writes a partial record that cannot replace the canonical one in `results_path`, so closeout still needs the full run.

### Constraints
A carried result must say where it ran. It never upgrades its outcome, so a carried failure stays a failure. Any doubt about eligibility widens to a full run, and the record states why. The selection reuses the changed-scope rules and widens them only by explicit declaration. Results and indexes written by earlier engines keep their meaning.

### Non-goals
Caching individual command outputs, hashing arbitrary inputs of a check, carrying across a matrix or engine change, and changing how review bundles seal evidence beyond reading the per-check head.

## 2. Requirements

### Functional
- R1: `pose validate --carry --json-out <results_path>` shall read the canonical record at the configured `results_path` as its base. It shall be refused with exit 2 when no `results_path` is configured, when the output path is not that path, or when combined with `--module`, `--stack`, `--workspace`, `--changed-from` or `--changed-to`.
- R2: The base shall be used only when every condition below holds. Otherwise the run shall execute every check and record `carry.used: false` with the first failing condition as `carry.reason`.
  - Its schema, mode, `matrix_sha256` and `engine_version` equal the current run's. Neither engine build is modified from its recorded revision.
  - Its `git_head` is an ancestor of, or equal to, `HEAD`.
  - It was not cancelled, at run level or in any check. A signal marks the whole run `cancelled`, even when every check was carried, and leaves no material binding available.
  - It recorded `material_clean`: one full predicate held when the head was pinned, before any plan input was read, and again when the run ended. The predicate covers status, hidden index entries and submodule checkouts. Its own working copy matched its head when it started and when it ended, HEAD did not move during the run, no untracked unignored material appeared, no index entry hid edits, neither HEAD's reflog nor any checked-out submodule's reflog grew (readable reflogs are required), mode changes counted whatever `core.fileMode` says, every submodule held its recorded commit. A run that did not validate its recorded head records every material binding as unavailable. A spec whose material the engine cannot compute is unavailable in any run, so an unavailable binding alone does not refuse a base.
  - The filesystem is case-sensitive, and no directory from the root to a module or a declared dependency folds case. An unreadable directory counts as folding, and so does a directory with no ASCII-letter entry to probe.
  - No declared dependency lies outside the repository, and every one is already a clean path that resolves physically to its own spelling, through no symlink, tracked or ignored. Module and dependency paths are ASCII.
  - No index entry, in the repository or a checked-out submodule, is marked assume-unchanged or skip-worktree. No module is reached through a symlink or shares its physical directory with another module. No tracked symlink has its path or target in a module other than the root or in a declared dependency, and no tracked target leaves the repository. This includes the tree of every submodule at its recorded commit. A submodule that cannot be scanned, because it is not checked out or has nested submodules, refuses.
  - No check id occurs twice, in the matrix or in the base.
  - The head and its reflog are pinned before any plan input is read; selection and the end-of-run comparison use that commit.
  - Every submodule is checked out at its recorded commit and has no nested submodules.
  - The working module metadata equals HEAD's, and the selection uses one snapshot read from the pinned commit. Absence is read from the pinned tree, and a listed blob that cannot be read refuses.
  - The working copy is clean in material paths.
- R3: With a usable base, the run shall select modules with the changed-scope rules over `base.git_head..HEAD`, excluding the `results_path` file itself. It shall also exclude derived administrative records of the engine's closed list (`posemodel.IsAdministrativePath`, the classes material bindings ignore) that are regular files, or absent, at `HEAD`. Otherwise the index and state commits that follow every record would select every module. It shall execute every check of a selected module. Every check of an unselected module whose base result exists and was not skipped shall be carried. A module with any check lacking such a base result shall be selected with the reason `carry: no prior result for <check id>`.
- R4: A carried check shall keep the base result's outcome, exit code, duration, limit state, output and evidence class. It shall add `carried_from`, the head the check actually ran at, preserved through chained carries, and `carried_generated_at`. Counts shall add `carried`. Carried failures and errors shall count toward the run outcome exactly like executed ones.
- R5: The record shall add `engine_version` on every run: the engine version plus the VCS revision the binary was built from, when the build records it. With `--carry` it shall also add a `carry` object: `used`, `reason`, `base_head`, `changed` (the number of files) and `selected` (module to reason).
- R6: The changed set shall show every gitlink change whatever the Git configuration, and shall read Git names NUL-delimited and without rename pairing, so a quoted or multi-line name keeps its owner and a rename names both its source and its destination. A root module `.` owns the files no other module owns, whatever it declares. Each changed file independently selects its most specific owner, every module enclosing that owner (the root module encloses them all), every module beneath it, and every module whose declared dependency contains it, lies beneath it or is the root `.`. Every eligible dependency counts, whatever its order. The dependency case applies to discovered and undiscovered dependencies alike. The exception is the file's own owner, which reaches its dependents through the transitive widening. A file no module owns also selects the root module and every module that is not self-contained. Module metadata that does not decode is an error, never a partial reading. A module declared `"selfContained": true` in `.pose/indexes/module-metadata.json` shall not be selected by a change outside every module. It is still selected by changes inside it, by its declared `dependsOn` and by policy. This shall hold for `--changed-from` and `--carry` alike, with the reason recorded for the modules it widens.
- R7: Reading the record into the delivery index shall use a check's `carried_from` and `carried_generated_at` as that result's head and time. Every consumer of a result's head therefore judges the commit a carried check really ran at.

### Non-functional
- R8: A run where every module is carried shall execute no check and finish in the time of the selection and the record write.

### Security
- R9: Revisions read from the base record shall pass the same confinement as `--changed-from` revisions before any Git call.

### Compatibility
- R10: A base record without `engine_version` shall never be used. Records without the new fields shall be read exactly as before.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/cli` (argument parsing, the run loop and record fields of `validate`, changed-scope selection), `pose-mcp/internal/pose` (reading the structured result into delivery results), the manual and the CLI help.

### Artifacts
- created: .pose/specs/2026-11-22-pose-validate-carries-unchanged-checks.md
- created: .pose/adr/2026-11-22-validate-carries-unchanged-checks.md
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_scope.go
- modified: pose-mcp/internal/cli/validate_results.go
- created: pose-mcp/internal/cli/validate_carry.go
- created: pose-mcp/internal/cli/validate_carry_test.go
- created: pose-mcp/internal/cli/validate_engine_identity_test.go
- modified: pose-mcp/internal/pose/delivery_validation_run.go
- created: pose-mcp/internal/pose/delivery_validation_run_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:validate-carries-unchanged-checks module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/validate.go

### API/contract changes
Additive fields in the structured validation result: `engine_version`, `carry`, `counts.carried`, and per check `carried_from` and `carried_generated_at`. There is a new `validate --carry` flag and a new optional `selfContained` field in module metadata.

### Data/storage changes
None beyond the additive record fields.

### Technical risks
An instance that marks a module `selfContained` while its checks read root files would carry stale results. The declaration is explicit, documented and opt-in, and the default stays the full fallback.

## 4. Tasks

### Planning
- [x] ADR for carrying results and the self-contained declaration.

### Implementation
- [x] Record fields and base eligibility.
- [x] Selection excluding the results file, the self-contained declaration and the no-prior-result widening.
- [x] Carried entries in the run loop and counts.
- [x] Per-check head in the delivery reading.
- [x] Manual and help.

### Validation
- [x] Unit and fixture tests for every requirement, each shown failing before its code.
- [x] `pose validate --module pose-mcp` and `check --strict`.

## 5. Decisions

### Decision D1
- Basis: R3, R4, R7
A carried result is the base's result, not a new observation. It keeps the head where it ran, and consumers judge that head. The run's material binding is still computed at `HEAD`. This holds only because the selection guarantees that a carried module's inputs did not change between the two heads, under the changed-scope rules and the instance's declarations. See ADR `2026-11-22-validate-carries-unchanged-checks`.

### Decision D2
- Basis: R2, R10
Eligibility compares the whole matrix digest and the engine version instead of per-check definitions. A matrix or engine change reruns everything. That is the existing rule for material bindings, and it keeps the comparison obviously sound.

### Decision D3
- Basis: R6
Out-of-module changes keep selecting every module unless a module declares itself self-contained. Instances opt in per module, so a repository that never declares anything gets today's behaviour.

## 6. Validation

### Strategy
Go tests in `internal/cli` drive real Git fixtures through `validate --carry`. They cover:
- the base conditions of R2, one fixture each;
- a change in one module that carries the other;
- chained carries;
- a carried failure;
- the self-contained declaration of R6, under both flags;
- the refusals of R1;
- the per-check head reading of R7.

Each test is run against the code before its requirement and must fail there.

### Deterministic checks

#### Test
- Command: `go test ./internal/cli/ ./internal/pose/` (in `pose-mcp`), then `pose validate --module pose-mcp`.

#### Lint
- Command: `go vet ./...` (in `pose-mcp`).

#### Typecheck
- Command: compiled by the tests.

#### Build
- Command: `go build ./...` (in `pose-mcp`).

#### Security / Contract
- Command: `pose check --strict`.

### Execution log
- 2026-11-22: the new fixtures in `validate_carry_test.go` and `delivery_validation_run_test.go` drive real Git repositories through `validate --carry`. Each requirement's code was mutated out and its test failed: the per-check head reading, the self-contained declaration, the canonical-result exclusion and the derived-record exclusion.
- 2026-11-22: `go test ./...` in `pose-mcp` first failed on `TestDirectPrintSitesOnlyShrink` and `TestEmbeddedDistMatchesPoseDist`; the carry lines now go through `cliout` and the embedded manual was regenerated. A full `pose validate` then failed the post-install gate because the manual cited derived-record directories absent from a fresh install; the text was rewritten without those paths.
- 2026-11-22: `pose validate --json-out .pose/results/delivery-validation.json` with a binary built from `606a0fa8` passed 236 of 236 checks in 786 s.
- 2026-11-22: dogfood: `validate --carry --explain` at `c4182727` used that record as base. The record and index commits counted as zero changed files. Five of six modules were carried, and `pose-mcp` ran because its criticality is high. The run passed with `counts.carried` 3, and the record is kept at `.pose/bin/batch-20261118/engine-a-dogfood-carried.json` in the Harne8 batch.

- 2026-11-22: review round 1 (rrn-0dd98c4aab8b5918, Codex) found F1 (high). A base that validated a dirty working copy could be carried. The record now has `material_clean`, and a base without it, or with an unavailable material binding, is refused. `TestCarryRefusesABaseThatValidatedADirtyWorkingCopy` reproduces the sequence. It failed with the check mutated out and passes with it.
- 2026-11-22: review round 2 (rrn-1c033e4c349d7794) confirmed R1-F1 fixed and found four new issues. N1: a rename's source path was missing from the changed set. N2: a self-contained root module was exempt from its own files. N3: quoted, newline-delimited Git names lost their owner. N4: modified engine builds at one revision shared an identity. All four are fixed, each with a regression test that failed when its fix was mutated out.
- 2026-11-22: review round 3 (rrn-3461f6ff36909419) confirmed N1–N4 fixed and found two more issues. N5: a deleted dependency left its self-contained dependents carried. N6: module metadata that did not decode was ignored. Both are fixed, each with a regression test that failed when its fix was mutated out.
- 2026-11-22: lead's own sweep before round 4: a gitlink above a declared dependency changed only the gitlink path, so the self-contained dependent stayed carried. A changed path now selects the modules and dependencies under it, tested through the shared selection; the test failed with the rule mutated out.
- 2026-11-23: review round 4 (rrn-4b28f2ff95acf9c4) found N7–N10.
  - N7: dependency and ancestor widening applied only to unowned files.
  - N8: a root dependency `.` never matched.
  - N9: modules aliased through symlinks.
  - N10: `material_clean` ignored HEAD moving during the run.
  The selection was rewritten so that each changed file applies every rule independently. Aliased modules refuse the carry, and a moved HEAD clears `material_clean`. Each regression test failed with its fix mutated out.
- 2026-11-23: review round 5 (rrn-b07ee93779a65e40) found N11–N14.
  - N11: the first matching dependency masked later ones.
  - N12: a moved HEAD kept the material bindings.
  - N13: case aliases on case-insensitive filesystems.
  - N14: the head was captured after planning.
  All four are fixed, each with a regression test that failed when its fix was mutated out.
- 2026-11-23: review round 6 (rrn-88e0428bda9bc9c7) found N15 and N16. N15: a discovered ancestor dependency missed changes in nested modules. N16: declared dependencies reached through symlinks. Both are fixed, each with a regression test that failed when its fix was mutated out.
- 2026-11-23: review round 7 (rrn-85704a129332a685) found N17–N19.
  - N17: an absent dependency leaf beneath a symlink.
  - N18: dependency case aliases.
  - N19: duplicate check ids sharing one base result.
  The per-path alias checks gave way to global rules: a case-insensitive filesystem refuses, and so does any tracked symlink whose path or target is in a non-root module or a declared dependency, or whose target leaves the repository. Symlinks between unowned paths are allowed, which covers both repositories' `.claude/skills` links. Duplicate check ids refuse. A mutation of the link-path rule first survived, which exposed a missing test for a module's outward link; that test was added. The matrix duplicate check is redundant with the base check under an equal matrix digest, so the two were mutated together.
- 2026-11-23: review round 8 (rrn-0375009e985e6a3d) found N20–N23.
  - N20: symlinks inside submodules.
  - N21: untracked material created during the run.
  - N22: case-folded directories within a case-sensitive filesystem.
  - N23: `diff.ignoreSubmodules` hiding gitlink changes.
  All four are fixed, each with a regression test that failed when its fix was mutated out.
- 2026-11-23: review round 9 (rrn-dfc7e3348240ce19) found N24–N26.
  - N24: gitlinks inside a module were not scanned.
  - N25: an enclosing module was not selected when a nested module changed.
  - N26: case folding in ancestor directories of a dependency.
  All three are fixed, and the lead extended N25 so the root module encloses every module. Each regression test failed when its fix was mutated out.
- 2026-11-23: review round 10 (rrn-29af03286365811e) found N27–N29. N27: the case probe failed open on an unusable first entry. N28: dependencies outside the repository were accepted. N29: the ADR and this summary misdescribed the self-contained fallback. All three are fixed; N27 and N28 have regression tests that failed when their fixes were mutated out.
- 2026-11-23: review round 11 (rrn-c0b4006e2a8ff303) found N30–N32.
  - N30: unclean dependency paths traversing symlinks.
  - N31: assume-unchanged and skip-worktree hiding edits from status.
  - N32: Unicode normalization aliases.
  All three are fixed conservatively, each with a regression test that failed when its fix was mutated out.
- 2026-11-23: review round 12 (rrn-abb61f059dc6d700) found N33–N35.
  - N33: dependencies reached through ignored symlinks.
  - N34: HEAD moving away and back between the endpoints.
  - N35: non-ASCII case swaps in the case probe.
  All three are fixed conservatively, each with a regression test that failed when its fix was mutated out. The first N35 mutation survived, which exposed a test entry without an ASCII letter; the test was corrected.
- 2026-11-23: review round 13 (rrn-f6546af4c9e5625e) found N36–N38 and the follow-up N39.
  - N36: `core.fileMode=false` hid mode changes.
  - N37: the head was pinned after the plan inputs were read.
  - N38: uninitialized submodules looked clean.
  N36 and N38 have regression tests that failed when their fixes were mutated out; the first N38 fixture did not isolate the fix and was corrected. N37 is an ordering fix: the pin now precedes the matrix and module reads, and a race before planning cannot be reproduced deterministically. N39, carried checks in `--report`, is follow-up F2.
- 2026-11-23: review round 14 (rrn-db44fd7f02f7b981) found N40 and N41, and N42 restated F2. N40: submodule and hidden-index conditions were checked only at the end. N41: start cleanliness was sampled after the plan inputs were read. One predicate, `validationMaterialClean`, now runs at the pin before the plan is read and again at the end. `TestValidateConditionClearedDuringTheRunIsNotClean` failed with the start predicate mutated back to status only.
- 2026-11-23: review round 15 (rrn-16091dd78a12cc0b; the first attempt rrn-7d33ba9fc274421f stopped on the sandbox's read-only scratch) found N43 and N45, and N44 restated F2. N43: submodule movement was not witnessed. N45: an all-carried run ignored a signal. The movement witness now covers every submodule's reflog, and a signal sets the run-level `cancelled`, which bases refuse. Both regression tests failed when their fixes were mutated out.
- 2026-11-23: review round 16 (rrn-328651cf23a32126) found N46–N48, and F2 was restated.
  - N46: working module metadata, an administrative record, steered the selection.
  - N47: a cancellation left bindings available.
  - N48: nested submodules went unwitnessed.
  All three are fixed, each with a regression test that failed when its fix was mutated out. The first N48 fixture did not isolate the fix and was corrected.
- 2026-11-23: review round 17 approved in rrn-6741de0e146e3777, but the conclusion was refused for a round-prefixed finding id. The repeat rrn-243366b14aa5fea4 was concluded as changes-requested in rva-b084f3e3849f1ed5 with N50: the metadata was verified and then reread from disk. The carry now selects with one snapshot read from the pinned commit. `TestCarrySelectsWithThePinnedMetadataSnapshot` failed with a disk reread mutated back in. The other repeated finding is F2.
- 2026-11-23: review round 18 (rrn-5ed84b0f66cce29b, attestation rva-f39a887e59b3aaaa) found N52: an unreadable metadata blob read as absent. N53 restated F2 and N54 restated F1. Absence now comes from the pinned tree, and a listed blob that cannot be read is an error. `TestCarryRefusesAnUnreadableMetadataBlob` failed when the old fallback was mutated back in.
- 2026-11-23: review round 19 (rrn-fc6566de3c7eedf0) approved, but the conclusion was refused at preflight because decision D1 named no requirement. Each decision now states its basis: D1 rests on R3, R4 and R7, D2 on R2 and R10, D3 on R6.
- 2026-11-23: the round-20 run showed that the rule refusing any base with an unavailable material binding refused every real base: a clean record of this repository holds 77 bindings the engine cannot compute. `material_clean` is the witness, and that rule was removed. The round-20 delegation also failed before running because the reviewer note exceeded 16 KiB, so the note was condensed.
- 2026-11-23: review round 21 (rrn-20db50998dd91866) found no code defect. Its run failed because `pose assess` wrote inside the sealed checkout. Its draft found N57: the manual still stated the removed rule. All four manual copies were corrected. The dogfood after round 21 carried five of six modules from a clean base with zero changed files; `pose-mcp` ran because of its high-criticality policy.
- 2026-11-23: review round 22 approved (rrn-41f73b6e7465d0bb, attestation rva-c09024015650a57b). `pose close` then refused with `undeclared-delivery`, because the spec declared no delivery target. The spec now declares `capability:validate-carries-unchanged-checks`, whose entrypoint is the `validate` command.

### Results summary
All new and existing tests pass, and the strict matrix is green. In this repository, `pose-mcp` holds 233 of the 236 checks and is high criticality, so carrying saves little here. In an instance with many modules, a change confined to one of them reruns that module, the modules enclosing it and its declared dependents. A changed file that no module owns also reruns every module that is not declared self-contained.

### Requirement trace
- R1 [satisfied] test:TestCarryRefusesIncompatibleFlags
- R2 [satisfied] test:TestCarryFallsBackToAFullRun test:TestCarryRefusesAFilteredBase test:TestCarryRefusesABaseThatValidatedADirtyWorkingCopy test:TestEngineIdentityRefusesModifiedBuilds test:TestCarryRefusesModulesReachedThroughASymlink test:TestValidateHeadMovingDuringTheRunIsNotClean test:TestValidateHeadMovingDuringTheRunBindsNoMaterial test:TestCarryRefusesModulesSharingADirectory test:TestCarryPlansAgainstThePinnedHead test:TestCarryRefusesDependenciesReachedThroughASymlink test:TestCarryRefusesASymlinkInsideAModule test:TestCarryRefusesASymlinkFromAModuleToAnUnownedPath test:TestCarryAllowsSymlinksBetweenUnownedPaths test:TestCarryDetectsACaseInsensitiveFilesystem test:TestCarryRefusesDuplicateCheckIDs test:TestValidateUntrackedMaterialCreatedDuringTheRunIsNotClean test:TestCarryRefusesAnAliasingSymlinkInsideASubmodule test:TestCarryDetectsACaseFoldedDirectory test:TestCarryScansAGitlinkInsideAModule test:TestCarryRelevantDirsIncludeEveryAncestor test:TestCarryCaseProbeSkipsUnusableEntries test:TestCarryRefusesDependenciesOutsideTheRepository test:TestCarryRefusesUncleanDependencies test:TestCarryRefusesNonASCIIPaths test:TestCarryRefusesHiddenIndexEntries test:TestCarryRefusesDependenciesThroughAnIgnoredSymlink test:TestValidateHeadMovingAndReturningIsNotClean test:TestCarryCaseProbeUsesOnlyASCIILetters test:TestValidateModeChangeHiddenByConfigurationIsNotClean test:TestValidateUninitializedSubmoduleIsNotClean test:TestValidateConditionClearedDuringTheRunIsNotClean test:TestValidateSubmoduleMovingAndReturningIsNotClean test:TestCarryRefusesABaseCancelledAtRunLevel test:TestCarryRefusesWorkingMetadataThatDiffersFromHead test:TestCancellationMakesBindingsUnavailable test:TestValidateNestedSubmodulesAreNotClean test:TestCarrySelectsWithThePinnedMetadataSnapshot test:TestCarryRefusesAnUnreadableMetadataBlob
- R3 [satisfied] test:TestCarryRunsOnlyTheChangedModule test:TestCarryRecordFileIsNotAChange test:TestCarryIgnoresDerivedAdministrativeRecords test:TestCarrySelectsAModuleWithoutAPriorResult
- R4 [satisfied] test:TestCarryKeepsTheOriginalHeadAcrossChainedCarries test:TestCarryKeepsACarriedFailureAFailure
- R5 [satisfied] test:TestCarryRunsOnlyTheChangedModule
- R6 [satisfied] test:TestCarryRootChangeRunsEverythingUnlessSelfContained test:TestChangedScopeHonoursSelfContainedModules test:TestCarrySelectsBothSidesOfARename test:TestCarryKeepsTheOwnerOfAQuotedName test:TestCarryRootModuleOwnsRootFiles test:TestCarryDeletedDependencySelectsItsDependents test:TestCarryRejectsUndecodableMetadata test:TestChangedScopeGitlinkAboveADependencySelectsItsDependents test:TestCarryDeletedNestedDependencySelectsItsDependents test:TestCarryRootDependencyIsChangedByRootFiles test:TestChangedScopeEvaluatesEveryDeclaredDependency test:TestChangedScopeDiscoveredAncestorDependencySelectsItsDependents test:TestChangedScopeIgnoresSubmoduleHidingConfiguration test:TestChangedScopeSelectsTheEnclosingModule test:TestChangedScopeRootModuleEnclosesEveryModule
- R7 [satisfied] test:TestDeliveryValidationRunReadsTheCarriedHead
- R8 [satisfied] test:TestCarryRecordFileIsNotAChange
- R9 [satisfied] test:TestCarryFallsBackToAFullRun
- R10 [satisfied] test:TestCarryFallsBackToAFullRun

R9 is enforced by `safeGitRevRE` in `planCarry` before any Git call; the not-an-ancestor case drives a forged head through it. R10 is the `base without engine version` case.

### Known gaps
Review bundles and closeout consume the per-check head through the delivery index. No separate end-to-end test seals a bundle over a carried record.

## 7. Final Report

### Delivered scope
`pose validate --carry` refreshes the canonical structured result. It runs the modules the changed-scope rules select since the canonical base and carries the other checks with their outcome and the head where they ran. The changed set ignores the result file and the derived administrative records that material bindings already ignore. Any doubt about the base runs everything, and the record says why. Modules can declare `selfContained` to stop being widened by changes outside every module. The delivery index reads a carried result as an observation of its own head. The record now names the engine version together with its build revision.

### Files and modules changed
`pose-mcp/internal/cli` (validate, its scope selection, its result fields and the help catalogue), `pose-mcp/internal/pose` (the delivery reading of the result), the manual in English and pt-BR with their embedded copies, this spec and its ADR.

### Validation executed
See the execution log: the package tests with mutation checks, the full `go test ./...` and the strict matrix with 236 of 236 checks.

### Residual risks
An instance that declares a module self-contained while its checks read files outside it would carry a stale result. The declaration is explicit and documented, and the default stays the full fallback.

### Follow-ups
- [open] F1: cover with an end-to-end test that a review bundle sealed over a carried record judges the carried head (seal, verify and closeout-check on a fixture). (owner:@pose-maintainers crit:low review:2026-12-15)
- [open] F2: `--carry --report` lists carried checks as executed commands without their `carried_from` head and time; exclude them from the executed summary and label reused results (review round 13, N39). (owner:@pose-maintainers crit:low review:2026-12-15)
