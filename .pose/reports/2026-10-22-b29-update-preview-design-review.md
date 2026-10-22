# B29 separate design review — 2026-10-22

Actual read-only separate Codex session 01a1623f-b773-7462-acd5-b09a82575cb1, task-designated gpt-6.1-sol; same-vendor fallback while Claude weekly limit was unavailable. No code/test execution or acceptance is inferred from this design review.

**Recommendation: replace the shadow copy with an explicit update execution context backed by an in-memory copy-on-write view of the original filesystem. Both apply and preview must run the same update steps.** Approve that structural choice in an ADR before implementation. A recorder added only to manual/machinery delivery would be incomplete.

This was a read-only source review. I ran no tests, assessment commands, Git mutations, lifecycle commands, or attestation commands. No network access was used.

The canonical [spec’s latest execution-log entry](/tmp/pose-pending-b29/.pose/specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.md) supersedes D1’s copy-based rationale: after the mode-0100 directory mismatch, the maintainer explicitly deferred delivery and directed a recording-based redo. The earlier R1/R2 “satisfied” traces and delivered-scope text describe the rejected implementation; they are not evidence that the replacement is complete.

**Material findings**

- **High — the existing dry-run does not meet the intended filesystem contract.** [dryRunUpdate](/tmp/pose-pending-b29/pose-mcp/internal/cli/update_dryrun.go:36) creates a temporary tree and initializes Git. Its eager copying cannot preserve the distinction between directory enumeration and access to a known child. It also substitutes the discovery environment. Another copier correction does not resolve this architectural problem.
- **High — the update’s effect boundary extends beyond delivery.** [cmdUpdate](/tmp/pose-pending-b29/pose-mcp/internal/cli/maintenance.go:27) reaches configuration seeding, module discovery, policy/profile migration, adoption stamps, indexing, and action-request creation. Under `--force`, [cmdInstall](/tmp/pose-pending-b29/pose-mcp/internal/cli/install.go:23) additionally reaches symlink replacement, legal files, MCP configuration, onboarding/default adoption, another index run, and the final gate.
- **Medium — R1’s request reporting remains incomplete.** The current dry-run counts action journals; the real update reports an aggregate configuration-review message. Neither identifies *each* request semantically. A shared per-capability request event is needed.
- **Medium — “nothing would change” needs a completeness condition.** Existing helpers swallow several discovery, read, and write errors. An empty recorded difference cannot establish R2 when a relevant phase was skipped or could not be evaluated.

**Proposed execution architecture**

Introduce an explicit, operation-scoped context, passed through update/install helpers and the model operations they invoke. It should contain:

- The original absolute root, embedded distribution, resolved options/locales, and one captured time.
- A filesystem/read view and an effect executor.
- Ordered semantic events, filesystem operations, and diagnostics.
- A completeness status: complete, blocked by a known condition, or incomplete because something could not be determined.

Use two executors:

1. **Apply:** performs operations immediately through existing production mechanisms and records their outcomes.
2. **Preview:** reads the original filesystem lazily, stages changes in memory, and records intended effects. It never invokes a real mutator.

Do not first turn the entire update into an atomic transaction. The present installer deliberately reports partial delivery after failures; preserve that execution order and behavior. “Shared operations” need not mean “plan everything, then commit everything.”

The preview view must support reads after staged writes. Examples:

- `recordDeliveredManual` updates `Manuals`; the later machinery-manifest write must preserve those staged records.
- Seeded review policy is subsequently migrated, stamped, and read to decide capabilities.
- The configuration-review spec must be visible when `PrepareActionRequest` validates its origin and requirement target.
- Index computation must see newly staged metadata/policies.
- A staged removal followed by symlink creation must change subsequent `Lstat`, `Stat`, and directory-listing results.

A writer interface alone cannot provide this behavior.

Use distinct operations for direct overwrite, exclusive create, atomic publication, directory creation, removal, and symlink creation. Preserve publication expectations and artifact class. **Do not collapse `os.WriteFile` and `PublishFile` into one generic “write”:** a direct overwrite follows a final symlink, while an atomic rename replaces that directory entry. Their permission requirements also differ.

The overlay should preserve lexical paths and resolve links only according to the operation being modeled. Reads through aliases must observe staged target content; replacing a link must affect the link entry. External link targets may be represented in memory, with diagnostics naming the logical managed path and external-target relationship. No preview operation may write the external target.

Keep a semantic operation ledger separate from the final net difference:

- The ledger explains merges, backups, migrations, request creation, and failures.
- The final difference supports R2 and file totals.
- Directory, mode, and link changes count as effects even when regular-file bytes remain identical.
- Repeated writes to the manifest are not separate changed files.
- Publisher staging files and locks are implementation activity, not ordinary delivered artifacts, but must remain behind the executor.

**Discovery must stay on the original source directories**

[seedModuleMetadataFromDiscovery](/tmp/pose-pending-b29/pose-mcp/internal/cli/stack_seed.go:143) should keep calling the existing `discoverValidationModules` against the original root. Preserve its exclusions, Git-ignore/submodule behavior, Android detection, root-alias heuristic, and existing-entry preservation.

Likewise, `scanModules` must inspect the original source tree. Read update-generated inputs such as module metadata and policies through the staged view. Do not introduce another scanner or fake Git repository.

There is an important split here: source discovery can remain a real read of original source paths, while governance-state consumers need the projected view. Git commands that inspect committed objects or ignore rules can use the original repository. A Git command whose answer depends on files virtually modified by the update cannot silently be treated as inspecting the projected state.

**Exact change map**

Paths below are relative to `/tmp/pose-pending-b29/pose-mcp/internal/`.

| Area | Functions requiring changes or an explicit injected boundary |
|---|---|
| `cli/maintenance.go` | `cmdUpdate`: parse once, select executor, run shared phases. `ensureManagedDirSafe`: retain original-path `Lstat` guards while virtualizing creation. Replace dry-only migration descriptions with events from actual migration decisions. Keep self-update outside preview. |
| `cli/update_dryrun.go` | Replace `dryRunUpdate`, copier, temporary Git setup, and production tree hashing with projected execution/reporting. Keep or relocate engine-version reading/comparison independently. |
| `cli/managed_docs.go` | `refreshManagedDocs`, `recordDeliveredManual`, `deliveredManual`, and identity/locale reads: use the operation view. Keep merge and heading-translation algorithms shared. |
| `cli/machinery.go` | `readMachineryManifest`, `storeMachineryManifest`, `loadMachineryManifest`, `loadMachineryDigests`, `saveMachineryManifest`, `deliverMachinery`, `deliverMachineryFile`, `machineryLocale`: staged reads/writes and shared events. |
| `cli/install.go` | Extract an internal install runner accepting the context; keep CLI parsing outside it. Adapt `recreateClaudeSkillSymlinks`, `configureMCP`, `writeMCPJSON`, `copyTreeInto`, `copyFileWithBackup`, `copyFile`, and target reads. Preserve pre/post-gate and partial-delivery behavior. |
| `cli/stack_seed.go` | `seedAbsentInstanceConfig`, `seedModuleMetadataFromDiscovery`, `migrateInstanceReviewPolicy`, `stampContractAdoption`, `stampChangelogAdoption`: route effects and relevant reads through the context; expose suppressed failures. |
| `cli/init.go`, `cli/adopt.go`, `cli/onboarding.go` | Internal `cmdInit` equivalent; `adoptGovernedCapabilitiesAtInstall`; `scaffoldOnboardingSpec`/`findOnboardingSpec`: virtual directories, policies, exclusive creation, and listing. |
| `cli/configuration_review.go` | `scaffoldConfigurationReview`, `configurationReviewRequests`, `reviewAnswerApplied`: shared staged-state reads, exclusive spec creation, and per-request semantic events. Replace physical-only globbing where necessary. |
| `pose/project_file.go`, `pose/capability_catalog.go` | `ReadProjectFile`/`WriteProjectFile`, `LoadPolicyDocs`, `PolicyDocs.Rendered`/`Write`, adoption-decision reads/writes, `SetReviewedVersion`, capability-state readers: injectable view/effects. |
| `pose/action_request.go`, `pose/action_scoping.go` | `PrepareActionRequest`, `OpenActionRequest`, relevant origin/material resolution, `LoadActionRequest`, `ListActionRequests`: validate staged specs and represent opening journals without disk access. |
| `cli/index.go`, `pose/index_generation.go` | Separate index computation from publication; inject staged governance reads. Cover `PublishIndexGeneration`, input bindings, generation descriptor/content, compatibility files, and retention decisions. |
| `cli/check.go` and model readers reached by it | Final-gate evaluation must consume projected governance state. Identify Git-dependent checks explicitly; do not run the existing physical-root gate and call its answer a projected verdict. |

The last two rows are substantial. Adding an optional filesystem field to `Store` does not automatically redirect package-level readers or newly constructed stores/resolvers. Propagation must be explicit and tested.

**Existing abstractions that can be reused**

There is useful machinery, but no general filesystem overlay or update transaction in the inspected code.

- [PublishFile](/tmp/pose-pending-b29/pose-mcp/internal/pose/writer_publication.go:111) remains the apply implementation for atomic publication. Reuse its expectation/conflict/durability types. Its hooks fire **after staging**, so they cannot implement a write-free preview.
- [PolicyDocs.Rendered](/tmp/pose-pending-b29/pose-mcp/internal/pose/capability_catalog.go:72) already separates rendering/validation from policy publication.
- [PrepareActionRequest](/tmp/pose-pending-b29/pose-mcp/internal/pose/action_request.go:121) provides shared normalization, validation, digest, and identity logic. Its reads still need the staged view.
- `NewIndexGenerationDescriptor` can compute descriptors from projected outputs and bindings. Keep production generation locking/publication intact.
- `scaffold.Dist()` and existing `fs.FS` parameters describe the embedded **source** distribution; they are not a destination abstraction.
- `Store` currently carries roots and caches, not filesystem virtualization.
- `cmdMigrate`’s dry-run is a different advisory surface, not an update simulator to reuse.

**Contract limits the ADR should state**

The preview can promise the complete intended operation set for the **running engine**, given observed inputs. It cannot promise that a later apply succeeds.

- Dry-run performs no self-update/network lookup. State that it previews the running engine’s instance update, equivalent in scope to `--no-self`; a later default update may hand off to a newer binary.
- Permission bits alone do not prove write success. ACLs, read-only mounts, security policy, quota, disk exhaustion, concurrent changes, link support, and durability failures cannot all be established without attempting effects.
- Preserve known rejections: `.pose` symlinks and managed-directory symlinks must be rejected as the real path rejects them. Do not globally reject links the updater currently accepts.
- Distinguish a known blocker from an unknown outcome. Never print “the update would fail” solely because preview cannot prove success.
- If a projected gate cannot be evaluated faithfully, report that uncertainty and mark downstream request creation conditional. **That is an intermediate implementation, not complete R1 parity.**
- Capture time once per invocation. Request IDs/date-based paths are preview values, not reservations for a later run.

For R1, emit the same semantic event in apply and preview, with the dry-run prefix added by the renderer. Each request event should identify capability, origin requirement, recipient role, question, and recommendation. Opaque IDs alone—and aggregate counts—are insufficient.

For R2, print “nothing would change” only after all relevant phases were evaluated and no content, directory, link, mode, or request effect remains. Unknown conditions require an explicit incomplete-preview result.

For R3, retain the existing warnings in [check](/tmp/pose-pending-b29/pose-mcp/internal/cli/check.go:151) and [doctor](/tmp/pose-pending-b29/pose-mcp/internal/cli/doctor.go:326). Consider reusing `CompareReleaseVersions` rather than maintaining a separate permissive numeric parser, after pinning missing, malformed, development, and future-version behavior.

**Realistic implementation stages and breadth**

1. **ADR/spec correction:** replace D1; define self-update, unknown-condition, request-wording, and no-op contracts.
2. **Execution boundary:** explicit context, apply adapter, in-memory view, operation/event types, captured time. Prove no real mutation is reachable in preview.
3. **Plain update:** manuals, machinery, metadata, config seeds, migrations/stamps, review spec and requests.
4. **Index path:** shared computation plus projected generation publication/retention. This is required even for plain update when computed indexes are seeded.
5. **Force path:** installer branches, default adoption/onboarding, MCP, links, init, projected final gate.
6. **Parity/reporting:** semantic event rendering, net effects, R3 regression coverage, remove the rejected copier.

Budget **roughly 20–25 core production files plus additional model/check/resolver readers**, plausibly **30–50 production files overall**, with approximately **8–12 test files**. This is an estimate, not a completed call-graph count. The minimum safe boundary is determined by reachable effects and reads, not the original spec’s artifact list.

The trade-off is broader dependency injection now in exchange for eliminating repeated filesystem reconstruction defects. A general POSIX emulator would be excessive; a narrowly supported operation view is reasonable, provided unsupported behavior becomes visible incompleteness instead of fabricated success.

**Testing and observability**

Reuse the current upgrade, machinery, locale/identity, stack-seeding, configuration-review, index-generation, and delivered-state fixtures. The existing force dry-run test proves nonmutation/success on its fixture; it does not compare force effects comprehensively.

Required new coverage:

- Compare preview operations/events with instrumented apply outcomes and actual final state, using a fixed clock. Compare request semantics, not just counts.
- Matrix: force/nonforce; stale/current/schema-zero/future schema; automatic and explicit English/Portuguese locale switches; renamed checkout identity.
- Real nested source modules, ignored directories, initialized submodules, Android detection, root aliases, absent computed indexes, preserved curated metadata.
- v1 policy/profile migrations; explicit empty adoption dates; custom fields; missing review policy under force; onboarding/default adoption; duplicate/open/answered-unapplied requests.
- Existing symlink corpus, plus dangling/internal/external targets, shared targets, ancestor links, atomic-replace versus direct-overwrite behavior, and generation-directory refusals.
- Unprivileged permission tests, especially mode `0100` directories with readable named children; read-only parent versus read-only file; unrelated unreadable files; inaccessible relevant inputs.
- Inject failure at each meaningful operation boundary, including backups, exclusive request creation, index publication/retention, and final-gate failure.
- A subprocess syscall audit for preview: reject filesystem mutations—including temp files, locks, Git index changes, and external link targets. Final content hashing alone misses transient or metadata-only writes.

Diagnostics should carry phase, operation, path, cause, and certainty. Relevant read/discovery loss must prevent a complete/no-change verdict. Preserve best-effort apply behavior where necessary, but make the lost outcome observable in both modes.

Run focused parity tests first, then module tests/build/vet and applicable validation-matrix checks. These were **not executed in this read-only design review**.

Rules consulted: `pose-review`, review workflow, backend Go, security, and delivery evidence. Sealing, attestation, assessments with possible writes, and lifecycle changes were omitted under the explicit task restriction.

**Review decision: changes required before B29 can claim R1/R2.** The ADR should authorize the shared execution context and projected read view across the full update path, including indexes and force-install gates. R3 can remain independently covered; it does not establish whole-update preview correctness.