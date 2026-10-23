---
slug: pose-update-dry-run-reports-the-whole-update
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers: capability:update-dry-run-reports-the-whole-update
---

# Spec: The update dry-run reports the whole update, and a stale instance is visible

## 1. Intent

### Goal

Make `pose update --dry-run` show everything `pose update` would do, and make an instance last updated by an older engine visible in `doctor` and `check`.

### Business value

On 2026-10-09 four instances (Harne8, pose-dist, audio-relay, storageclose) were declared adopted on 7.1.0 because `pose update --no-self --dry-run` printed only `instance already at schema v1` and `DRY-RUN — no changes applied`. The real update then merged machinery into AGENTS.md, POSE.md, skills and the spec template, stamped `engine_version`, seeded `.pose/project.json` and opened a configuration review with up to eight decision requests. Every instance had stayed stamped by 6.1.0 since its earlier update; only `pose version` revealed it, and `doctor` and `check --strict` passed silently. Three adoption specs recorded a false conclusion as a result.

### Constraints

The dry-run writes nothing, as today.

## 2. Requirements

### Functional

- R1: `pose update --dry-run` shall list every file it would merge or create, the `engine_version` change, and each configuration-review request it would open, with the same wording the real run prints.
- R2: When nothing would change, the dry-run shall say so explicitly, distinct from "no schema migration".
- R3: `pose doctor` and `pose check` shall warn when the instance's `engine_version` is older than the running engine, naming `pose update`.

## 3. Technical Plan

### Artifacts

Historical D1 declarations (quoted for audit; the cumulative declarations below govern the whole spec):

> - created: .pose/specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.amendments.jsonl

> - created: .pose/specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.md
> - created: .pose/starts/pose-update-dry-run-reports-the-whole-update.json
> - created: pose-mcp/internal/cli/update_dryrun.go
> - created: pose-mcp/internal/cli/update_dryrun_test.go
> - modified: pose-mcp/internal/cli/maintenance.go
> - modified: pose-mcp/internal/cli/check.go
> - modified: pose-mcp/internal/cli/doctor.go
> - modified: pose-mcp/internal/cli/compat_test.go
> - modified: pose-mcp/internal/cli/upgrade_test.go
> - modified: pose-mcp/internal/cli/testdata/direct-print-sites.json

> - created: .pose/adr/2026-10-22-update-preview-records-the-shared-execution-in-an-operation-scoped-view.md
> - created: .pose/reports/2026-10-22-b29-update-preview-design-review.md

Cumulative D1/D2 declarations, including replacement edits declared before implementation:

- created: .pose/specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.amendments.jsonl
- created: .pose/specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.md
- created: .pose/starts/pose-update-dry-run-reports-the-whole-update.json
- created: pose-mcp/internal/cli/update_dryrun.go
- created: pose-mcp/internal/cli/update_dryrun_test.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/compat_test.go
- modified: pose-mcp/internal/cli/upgrade_test.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json
- created: .pose/adr/2026-10-22-update-preview-records-the-shared-execution-in-an-operation-scoped-view.md
- created: .pose/reports/2026-10-22-b29-update-preview-design-review.md
- created: pose-mcp/internal/cli/update_preview_test.go
- modified: pose-mcp/internal/cli/adopt.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/assess.go
- modified: pose-mcp/internal/cli/command_reference.go
- modified: pose-mcp/internal/cli/configuration_review.go
- modified: pose-mcp/internal/cli/contribute.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/init.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/machinery.go
- modified: pose-mcp/internal/cli/managed_docs.go
- modified: pose-mcp/internal/cli/onboarding.go
- modified: pose-mcp/internal/cli/onboarding_context.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/usage.go
- modified: pose-mcp/internal/cli/validate.go
- created: pose-mcp/internal/pose/operation_files.go
- created: pose-mcp/internal/pose/operation_files_test.go
- created: pose-mcp/internal/pose/operation_files_access.go
- created: pose-mcp/internal/pose/operation_files_access_other.go
- created: pose-mcp/internal/pose/operation_git.go
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/action_scoping.go
- modified: pose-mcp/internal/pose/agent_project_context.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/assessment_scan.go
- modified: pose-mcp/internal/pose/capabilities.go
- modified: pose-mcp/internal/pose/capability_catalog.go
- modified: pose-mcp/internal/pose/causality_closeout.go
- modified: pose-mcp/internal/pose/changelog_policy.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/pose/contract_nodes.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/pose/design_basis.go
- modified: pose-mcp/internal/pose/design_delta.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/pose/docs.go
- modified: pose-mcp/internal/pose/docs_review.go
- modified: pose-mcp/internal/pose/dor_policy.go
- modified: pose-mcp/internal/pose/effective_governance.go
- modified: pose-mcp/internal/pose/falsifier_observation.go
- modified: pose-mcp/internal/pose/federated_acceptance.go
- modified: pose-mcp/internal/pose/followup_candidates.go
- modified: pose-mcp/internal/pose/followups_parse.go
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- modified: pose-mcp/internal/pose/git_spawn_cache.go
- modified: pose-mcp/internal/pose/governance_outcomes.go
- modified: pose-mcp/internal/pose/governance_replay.go
- modified: pose-mcp/internal/pose/index_generation.go
- modified: pose-mcp/internal/pose/knowledge.go
- modified: pose-mcp/internal/pose/legacy_inventory.go
- modified: pose-mcp/internal/pose/legacy_ledger.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/obligation_sources.go
- modified: pose-mcp/internal/pose/pointers.go
- modified: pose-mcp/internal/pose/premise_validity.go
- modified: pose-mcp/internal/pose/project_file.go
- modified: pose-mcp/internal/pose/readiness.go
- modified: pose-mcp/internal/pose/release_archive.go
- modified: pose-mcp/internal/pose/release_claimed_fragments.go
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/pose/remediation_lineage.go
- modified: pose-mcp/internal/pose/review_attribution.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_path_attribution.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_reuse_provenance.go
- modified: pose-mcp/internal/pose/review_structure.go
- modified: pose-mcp/internal/pose/roadmaps.go
- modified: pose-mcp/internal/pose/roots.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/spec_transfer.go
- modified: pose-mcp/internal/pose/spec_transfer_locks.go
- modified: pose-mcp/internal/pose/start.go
- created: .pose/reports/2026-10-22-b29-update-preview-implementation.md
- modified: pose-mcp/internal/pose/spec_facts.go
- created: pose-mcp/internal/cli/update_tree_test.go
- created: pose-mcp/internal/cli/update_preview_linux_test.go
- created: pose-mcp/internal/pose/update_preview_facts_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/cli/review_closeout_test.go

- modified: pose-mcp/internal/cli/rule_extension_resolver.go
- modified: pose-mcp/internal/cli/suggest.go
- modified: pose-mcp/internal/pose/path_state.go
- modified: pose-mcp/internal/pose/release_candidate.go
- modified: pose-mcp/internal/pose/release_manifest_integrity.go
- modified: pose-mcp/internal/pose/release_version_source.go
- created: pose-mcp/internal/cli/update_preview_composition_test.go
- created: pose-mcp/internal/pose/update_preview_release_test.go

- modified: pose-mcp/internal/cli/release_review_regression_test.go
- created: .pose/changelogs/unreleased/pose-update-dry-run-reports-the-whole-update.md
- created: .pose/reports/2026-10-23-b29-pending-review-r1.md
- created: .pose/reports/2026-10-23-b29-pending-closeout.md

- created: pose-mcp/internal/pose/operation_files_access_linux_test.go

### Delivery targets

- capability:update-dry-run-reports-the-whole-update module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Context: a dry-run that describes the update is a second implementation of it, and the first one drifted.
- Decision: the dry-run runs the same `cmdUpdate`, without the binary self-update, on a disposable copy of `.pose`, `.agents`, `.claude`, `.codex`, `.github` and the top-level files, then reports the files it would create, modify or remove, the `engine_version` stamp, and the update's own output prefixed `[DRY-RUN]`.
- Rationale: fidelity by construction; the test compares the dry-run's list with the real update's changes on the same fixture.
- Consequences: an update step that reads source directories outside the copy (module discovery for absent seeds) sees none in the dry-run.

### Decision D2
- Date: 2026-10-22
- Basis: R1, R2, R3
- Context: The latest maintainer direction rejects shipping the copy implementation. D1 remains a historical strategy, not the design for this replacement.
- Decision: Use the shared apply/preview execution context and in-memory operation-scoped view described in [the prospective ADR](../adr/2026-10-22-update-preview-records-the-shared-execution-in-an-operation-scoped-view.md). Preserve the current running-engine/no-self preview boundary and normal apply semantics; include force install, discovery, requests, index computation and final governance reads.
- Rationale: One execution algorithm with projected reads eliminates reconstruction and second-planner drift. Partial preview cannot satisfy R1/R2.

## 6. Validation

### Strategy

A fixture instance stamped 6.1.0: the dry-run output lists the merges, stamp and requests the real update then performs (compared line by line), and `doctor` warns before the update and not after. The test must fail on the current engine first.

### Replacement test plan — recorded before production changes on 2026-10-22

Risk: high (`pose-mcp` metadata). Original R1/R2 satisfied trace and old Delivered scope below describe the rejected implementation; they do not qualify the replacement. Preserve historical start/amendment records; record D2 as introduced during execution.

| Required scenario | Deterministic command | Expected evidence |
|---|---|---|
| Operation view read-after-write, alias/link replacement, exclusive/atomic/direct-write distinction and failure injection | `go test ./internal/... -run 'UpdateOperation|UpdatePreview' -count=1` | No physical preview mutation, coherent staged reads, typed failures and exact net effects. Register exact final tests/check before collecting. |
| Running-engine update parity, force/nonforce, EN/pt-BR, current/stale/schema migration, requests and policy/profile seeding | `go test ./internal/cli -run 'UpdateDryRun|UpdatePreview|Upgrade|Machinery|ConfigurationReview' -count=1` | Preview/apply share semantic events and per-request content, actual resulting state matches intended effects; no false no-op. |
| Original discovery and adversarial filesystem/permission layouts | `go test ./internal/cli -run 'UpdateDryRun|UpdatePreview|Discovery' -count=1` | Existing symlink corpus, execute-only directory with named child, external/broken links, ignored/submodule/nested modules and unknown relevant reads exercised in unprivileged fixtures. |
| End-to-end no-write audit and force projected gate | `go test ./internal/cli -run 'UpdatePreview.*(Subprocess|NoWrites|Force|Parity)' -count=1` | Actual command preview produces no filesystem effects, including external targets and transient staging/locks; projected final gate sees planned governance state. |
| Existing engine contracts and production composition | `pose validate --strict --report --json-out .pose/results/delivery-validation.json` | All required matrix checks pass at the final source; current scoped provenance and registered integration reachability. |
| Go API, races and platform builds | `go test ./...`; `go vet ./...`; `go test -race ./internal/pose ./internal/cli` | Existing production behavior preserved; no shared global preview state; existing cross-platform build checks retained. |

At the time this prospective plan was recorded, no replacement test above had executed. Current implementer results and exact selectors are recorded in [the implementation report](../reports/2026-10-22-b29-update-preview-implementation.md); the registered required check is `update-preview-integration`. Native global evidence, immutable attribution and independent final review remain with the parent.

### Execution log

2026-10-10: on storageclose (stamped 7.1.0), the dry-run reported the machinery merges, `would modify: .pose/state/machinery-manifest.json` and `would stamp engine_version: 7.1.0 -> 7.3.0-dev`, and left the working tree unchanged. The real update's "instance already at schema v1. Nothing to do." line, printed after merging machinery, now says the manuals and machinery are current for the engine.

2026-10-10, independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): medium severity. `pose update --dry-run --force` reported "the update would fail" because the copy held an empty `.git` and `--force` runs install, which refuses a directory git does not recognise; the real `--force` update succeeded. The copy is now `git init`-ed; `TestUpdateDryRunWithForceRunsOnTheCopy` covers it (reproduced on audio-relay before the fix).

2026-10-10, independent review (agent:independent-gpt-6.1-sol-review): medium severity. The copy skipped symlinks, so with `.pose/policy` symlinked the dry-run predicted 23 files and 12 requests the real update does not create. The copy now follows symlinks as the update reads through them (cycles broken, broken links skipped); `TestUpdateDryRunFollowsASymlinkedPolicy` fails on the previous copy.

2026-10-10, second independent review (agent:independent-gpt-6.1-sol-review): medium severity. Following links turned them into plain directories, hiding the update's refusals: with `.pose/templates` symlinked the dry-run returned 0 and the real update 1. The copy now keeps links as links, pointing at copies of their targets (inside the shadow, or next to it for targets outside the instance), so the update meets the same links and nothing it writes reaches the instance; `hashTree` reads through links to compare content. `TestUpdateDryRunMeetsTheLinksTheUpdateMeets` fails on the previous copy.

2026-10-10, third independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): two medium findings and one low. (1, 2) A directory reached through a link and by its own path (`.claude/skills -> ../.agents/skills`) was copied twice, and the second pass failed on the links and read-only files the first wrote, where the update succeeds; the copy is now idempotent. `TestUpdateDryRunCopiesALinkedDirectoryOnce` fails on the previous copy. (3) Action request ids hash the second they are opened, so the copy's file names were not the update's, and `TestUpdateDryRunListsWhatTheUpdateChanges` failed about 1 in 300 runs; the dry-run now counts the requests it would open instead of naming them, and the test compares counts (40 consecutive runs pass).

2026-10-10, fourth independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): medium severity. The "already copied" shortcut took a directory as copied because its path existed, which a link to one file inside it had created first; with `.pose/policy -> ../config/policy` and an alias to one policy file, the dry-run predicted 31 changes and 12 requests against the update's 9 and 8. The shortcut is gone: a linked directory is always copied, and the copy skips what is already there. `TestUpdateDryRunCopiesADirectoryAfterALinkIntoIt` fails on the previous copy.

2026-10-10, fifth independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): medium severity. A link to a directory that contains it (`.pose/self -> .`, `.agents/up -> ..`) made the copy reach the same link again and fail with "file exists", where the update succeeds; the self case regressed in 5e2579af. Such a link is now linked to the copy of its target without copying the target into itself, and a link already recreated is left alone. `TestUpdateDryRunLinksADirectoryThatContainsTheLink` fails on the previous copy.

2026-10-10, sixth independent review (agent:independent-gpt-6.1-sol-review): high severity. A broken link was recreated with its original target, so with `.pose/rules/security.md` pointing at a missing absolute path the dry-run created that external file and still said nothing was applied. A broken link is now recreated still broken, at the same missing path inside the shadow when the target was inside the instance, or at a missing path next to the shadow otherwise; no link in the copy points outside it. `TestUpdateDryRunNeverWritesThroughABrokenLink` fails on the previous copy.

2026-10-10, seventh independent review (agent:independent-gpt-6.1-sol-review): medium severity. The stand-in for an external broken link lost whether its parent directory exists, so with the target's directory present the dry-run said the update would fail while the update created the file. The stand-in's parent now exists exactly when the original's does. `TestUpdateDryRunAgreesOnABrokenLinkIntoAnExistingDirectory` fails on the previous copy.

2026-10-10, eighth independent review (agent:independent-gpt-6.1-sol-review): medium severity. A broken link at the top of the instance (`AGENTS.md` pointing into a missing directory) was left out of the copy, so with `--force` the dry-run predicted success where the update fails. Top-level links to files and broken top-level links are now copied as links. `TestUpdateDryRunKeepsABrokenTopLevelLink` fails on the previous copy.

2026-10-10, ninth independent review (agent:independent-gpt-6.1-sol-review): medium severity, outside symlink layouts. Directories were recreated 0755, so with `.pose` read-only the update failed while the dry-run said it would change nothing. The copy now applies each directory's original permissions after copying, deepest first, and makes them writable again to remove the copy. `TestUpdateDryRunKeepsDirectoryPermissions` fails on the previous copy.

2026-10-10, tenth independent review (agent:independent-gpt-6.1-sol-review): medium severity. The copy read every file at the root, so an unrelated unreadable file (mode 000) made the dry-run fail where the update succeeds. An unreadable file is now reproduced as an empty file with the same mode, and an unreadable directory as an empty one with its mode, so the update meets the same refusals; comparisons skip what cannot be read. `TestUpdateDryRunToleratesUnreadableFiles` fails on the previous copy.

2026-10-10, after the tenth independent review (a directory with mode 0100 can be read by name but not listed, so the copy came out empty): the maintainer decided that POSE 7.4.0 ships without this spec. Its changelog fragment is withdrawn from `.pose/changelogs/unreleased/`, so 7.4.0's notes do not announce the dry-run; the spec stays open and is redone for 7.5.0 with the approach of the open follow-up (the update recording what it would do instead of running on a copy). The copy-based dry-run already on main stays in the code until then.

### Requirement trace

Historical D1 trace, retained for audit. These old satisfaction claims do not qualify D2; use the replacement implementation report and the parent's eventual current evidence for that judgment.

> - R1 [satisfied] capability:update-dry-run-reports-the-whole-update evidence:integration test:TestUpdateDryRunListsWhatTheUpdateChanges test:TestUpdateDryRunWithForceRunsOnTheCopy test:TestUpdateDryRunFollowsASymlinkedPolicy test:TestUpdateDryRunMeetsTheLinksTheUpdateMeets test:TestUpdateDryRunCopiesALinkedDirectoryOnce test:TestUpdateDryRunCopiesADirectoryAfterALinkIntoIt test:TestUpdateDryRunLinksADirectoryThatContainsTheLink test:TestUpdateDryRunNeverWritesThroughABrokenLink test:TestUpdateDryRunAgreesOnABrokenLinkIntoAnExistingDirectory test:TestUpdateDryRunKeepsABrokenTopLevelLink test:TestUpdateDryRunKeepsDirectoryPermissions test:TestUpdateDryRunToleratesUnreadableFiles
> - R2 [satisfied] capability:update-dry-run-reports-the-whole-update evidence:integration test:TestUpdateDryRunListsWhatTheUpdateChanges
> - R3 [satisfied] capability:update-dry-run-reports-the-whole-update evidence:integration test:TestUpdateDryRunListsWhatTheUpdateChanges

### Current D2 requirement trace

The registered `update-preview-integration` check and full native collection passed at source `3f03b36`; the later sticky-directory correction requires a fresh collection. Final independent approval and governed closure remain pending.

- R1 [satisfied] capability:update-dry-run-reports-the-whole-update evidence:integration test:TestUpdatePreviewParityAcrossUpdateBranches test:TestUpdatePreviewNamesEachDecisionRequestAsTheUpdateDoes test:TestUpdatePreviewForceStagesInstructionsBeforeFinalGate test:TestUpdatePreviewHasNoPhysicalMutationEvents
- R2 [satisfied] capability:update-dry-run-reports-the-whole-update evidence:integration test:TestUpdatePreviewDoesNotCallLostEvaluationNoChange test:TestUpdatePreviewDistinguishesRefusedOperationsFromNoOp test:TestUpdatePreviewSourceReviewFilesystemReachability test:TestUpdatePreviewUnreadableEmptySkillDirectoryIsIncomplete
- R3 [satisfied] capability:update-dry-run-reports-the-whole-update evidence:integration test:TestStaleInstanceEngineVersionContracts

## 7. Final Report

Historical D1 report below, retained unchanged for audit. It describes the rejected copier, not a current delivery claim. D2 remains `in-progress`; see the replacement implementation report for current source and validation evidence.

### Replacement qualification — 2026-10-23

D2 uses one shared update execution with an explicit operation-scoped in-memory view. Reads see staged files, original source discovery stays authoritative, and complete previews report file effects, engine stamps and individual request semantics. Unknown access, umask, inode alias, special-file or removal effects remain INCOMPLETE rather than successful no-ops. Native apply is unchanged. Fresh canonical strict validation passes146/146 at3f03b36, with zero skips/errors; see [the closeout report](../reports/2026-10-23-b29-pending-closeout.md) and [implementation evidence](../reports/2026-10-22-b29-update-preview-implementation.md). Source review R1/R2 rejections and fixes remain historical; final approval and closure are pending.

### Historical D1 delivered scope

`pose update --dry-run` runs the update on a disposable copy and lists what it would change and stamp; `doctor` and `check` warn while the instance was last updated by an older engine.

### Residual risks

- Module discovery for absent seeds reads source directories the dry-run copy does not include.
- Symlink layouts. Eight review rounds each found a rarer layout where the copy and the instance differed; all eight are fixed and tested. On 2026-10-10 the maintainer decided that the ninth round is the last for this spec: a symlink layout not yet covered is a residual risk tracked by the follow-up below, and only a defect outside symlink layouts blocks closing. The copy never points a link outside itself, so an uncovered layout can make the dry-run's prediction wrong, not make it write into the instance or elsewhere.

### Follow-ups

- [done] Replace the copier with operation-scoped projected execution recording the shared update. D2 implements this follow-up directly; operation, parity, original-discovery, permission and mutation regressions qualify the replacement. Historical copier defects remain documented above; unsupported knowledge is explicit INCOMPLETE.
