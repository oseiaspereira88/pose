# B29 replacement implementation — in-progress, 2026-10-22

Current continuation evidence is recorded in “Source-review fixes — 2026-10-23” below. Earlier continuation/composition results and the native 146/146 run on `7d9c07c` are historical evidence after these edits.

Canonical spec: `pose-update-dry-run-reports-the-whole-update`. Original start, historical D1 and failed copy-based rounds are retained. Prospective accepted operation-scoped-view ADR and high-risk test plan preceded production edits; D2 was recorded natively as introduced during execution. The amendment author `@pose-maintainers` denotes governance ownership; it was recorded by automated lead Codex, not human confirmation or policy adoption.

Initial delegated implementation: Claude Opus5.5, actual launcher session bcaaad50-95ff-4ee5-a658-3e6cbf4d2caf, logs `/home/go/.local/state/secondary-agent/20261022T091311-claude-2052150`. It stopped at session limit (reset14:10America/Recife), before a final implementation report. No passing test, completed scope or review is inferred from the partial run. Its uncommitted operation filesystem/view and explicit Store/CLI propagation are preserved as WIP, including prospective artifact declarations; 98 claims cover all changed production paths. No remote publication or shared-checkout changes.

Historical checkpoint plan: Primary will commit an explicitly unqualified checkpoint, integrate already merged B30 from main (47d1cf0), then hand the same pending implementation to separate Codex to continue. Re-run any interrupted command and validate behavior, not just compilation. This is not an attestation or closeout. Final source, tests, composition evidence and separate review remain pending.


## Codex continuation — 2026-10-22

Implementer: task-designated Codex gpt-6.1-sol, continuing local merge `e91a39f` and unqualified checkpoint `62bcd5a`. Read the actual delegated session events at the path above; its terminal event is a session-limit rejection. Re-ran validation from compilation onward. No earlier passing result was inferred. This report is implementer evidence, not the independent final review or a spec closeout. The spec stays `in-progress`.

### Execution and contracts

Retain the explicit `Files` / `PreviewFiles` implementation and the shared `instanceUpdate.apply` path. Plain update and force install make the same decisions through the invocation's filesystem. Preview reads original paths lazily, stages effects only in memory and reports semantic update lines with a preview prefix. The ordered operation ledger remains distinct from net content/directory/link/mode changes. The running-engine boundary is explicit; preview performs no binary self-update. Native wrappers retain their ordinary behavior for other commands.

Keep direct overwrite, exclusive creation and atomic publication distinct. Native `PublishFile` still owns staging, expectations, durability and partial delivery; preview never enters that publisher or acquires its locks. Repair onboarding's missed physical exclusive create. Reject publication over an existing directory, matching the native refusal, without hiding its children. Preserve retired machinery retention, no implicit extension installation, existing force behavior and the missing-policy default adoption/onboarding branch.

Carry projected governance through seeding, migration/stamps, capability requests, indexes, resolver-created Stores and final gates. B30's shared `ParseSpecTrailerLog` / `ParseDeliveryValidationRun` remain in `artifact_integrity.go` / `surface_check.go`. B30's new facts reader now uses the Store view for input bindings, delivery policy, validation results and explicit history attribution; committed Git history stays a real read. No B30 binding, generation-pinning or failure checks were removed. Use the captured operation time for retention, docs deadlines and assessment age, as well as requests and stamps.

`updateReadView` carries the explicit view and exposes relevant unexpected read loss, including failures old best-effort helpers swallowed. Configuration-review, capability, discovery, malformed metadata/profile/changelog and skipped-manual evaluation gaps prevent a complete/no-op preview. Failed optional effects remain observable without converting native update to a transaction. Migration success lines follow a successful write. A known tolerated refusal with no net change gets a refusal verdict rather than “would change nothing” or an invented fatal native outcome.

### Reproductions and corrections

1. The inherited checkpoint did not compile: older tests still referenced removed `hashTree` / `diffTrees`. Move those helpers to test-only `update_tree_test.go`; production preview contains no copier or eager tree snapshot.
2. The inherited execute-only-directory and real-source-discovery regression fixtures pass as the actual unprivileged uid 1000. The original rejected copier is not restored to reproduce its historical defect. The new operation-view tests access named children without enumerating the directory.
3. Force preview with missing review policy and onboarding spec physically created the onboarding spec. The fixed-clock parity test failed on this write before changing `scaffoldOnboardingSpecIn` to `Files.CreateExclusive`.
4. A current instance with a malformed/unreadable review policy or inaccessible source directory returned a successful “would change nothing”. Added negative tests failed before the diagnostic changes. Further negative tests reproduced the same false no-op for malformed review profiles and changelog policy; these now report `INCOMPLETE`. Corrupt Git discovery also prevents a complete verdict.
5. Preview publication accepted replacing a directory where native publication refuses. `TestUpdateOperationPublicationRefusesADirectory` failed before the correction.
6. The full suite exposed historical duplicate artifact declarations, one removed migration-only phrase and a direct-print ratchet violation. Keep D1 declarations quoted for audit, consolidate exact cumulative claims once, assert the shared migration wording and render preview lines through `cliout`.

7. Strengthening the audit from the internal entry to a subprocess first exposed test-harness startup writes; the audit now builds and runs the actual CLI before/after the watch boundary. With usage enabled, it then reproduced a real external usage-directory creation after preview. The update parser now suppresses post-command usage recording on its own invocation-local output context when it selects dry-run. No environment/global preview flag is set. `TestUpdatePreviewLeavesUsageRecordingForApply` proves a later apply still records usage and a later preview preserves that storage.

8. Race validation exposed another historical test helper comparing timestamp-derived request IDs across two separate previews. Normalize only IDs in that helper; fixed-clock branch parity continues comparing exact request bytes and semantic output.

### Behavioral evidence

- `TestUpdatePreviewParityAcrossUpdateBranches`: fixed-clock semantic output, return codes, exact net effects, file bytes, modes and links across stale/current, plain/force, explicit/automatic locales, English switch, schema zero/future refusal, policy/profile v1 migration, deliberately empty adoption dates, curated root alias metadata, missing computed indexes, missing-policy default adoption/onboarding, existing open and answered-unapplied requests and final-gate refusal.
- `TestUpdatePreviewDiscoversSubmodulesAndroidAndCuratedMetadata`: original source traversal, initialized submodule-local ignores, nested modules, Android classification, curated metadata preservation and root alias without a duplicate `.`. The existing symlink corpus remains active.
- `TestUpdatePreviewNamesEachDecisionRequestAsTheUpdateDoes`: capability, origin requirement, question, role and recommendation match apply. Fixed-clock parity also compares persisted request bytes, rather than request counts alone.
- `TestUpdatePreviewPreservesPartialDeliveryAtFailureBoundaries`: manual backup/publication, machinery backup, exclusive request creation, generation file, descriptor commit and generation retention failures share native/preview output and partial effects. Final-gate refusal is separately covered in the branch matrix.
- `TestUpdatePreviewDoesNotCallLostEvaluationNoChange` and `TestUpdatePreviewDistinguishesRefusedOperationsFromNoOp`: unknown evaluation, known refusal and a completed no-op have different verdicts; native best-effort return behavior remains intact.
- `TestUpdateOperationFactsReadProjectedPolicyResultsAndHistory`: staged B30 policy/result/history reads and generation bindings do not fall through to physical state.
- `TestStaleInstanceEngineVersionContracts` plus the doctor/check stale-to-current integration test retain R3 behavior. Missing/invalid-JSON stamps are omitted, suffixes/prefixes retain existing interpretation, current/future versions do not warn; the legacy tolerant numeric-part interpretation is explicitly pinned.
- Linux `TestUpdatePreviewHasNoPhysicalMutationEvents`: build before watching, then execute the actual CLI binary as a child with usage recording enabled while inotify watches every fixture directory, `.git`, external targets and `TMPDIR`. Reject mutation events including transient staging/locks and verify internal/external dangling targets remain absent. This is a subprocess filesystem-event audit, **not** a syscall audit. Snapshot and operation tests also cover modes, links and contents.

Register `update-preview-integration` as a required integration check in the instance validation matrix, using real nonempty selectors for `TestUpdatePreview`, `TestUpdateDryRun`, `TestUpdateOperation` and `TestStaleInstanceEngine` across CLI/model packages. No global matrix execution or generated-index rewrite was performed.

### Validation environment and limits

Run from `pose-mcp` with `GOTOOLCHAIN=go1.26.10 TMPDIR=/var/tmp/pose-b07-validation-0bf729f GOFLAGS=-buildvcs=false`. The VCS flag applies only to validation build stamping. Do not touch unrelated `/tmp/.git` or weaken fixture guards. No new dependencies were installed.

Final local check results are recorded below. This does not substitute for the parent's native delivery gates. Logs reside under the approved validation root. `govulncheck`, `golangci-lint` and `gitleaks` are unavailable in this environment; do not infer those scans from Go tests. `go mod verify` passed (`all modules verified`). No dependency manifests changed.

The parent still owns Git attribution/commit trailers, current-main reconciliation, native global matrix/index evidence, sealing, independent final review and lifecycle closure. These were explicitly excluded from this continuation. No Git staging, commits, ref changes, fetch, merge, reset, push, external messages, contribution drafts, release or real-project adoption was performed here.


### Final local check results — 2026-10-22

All commands below exited 0 on the final production implementation. Commands are relative to `pose-mcp` and use the environment above.

| Command | Actual result / evidence |
|---|---|
| `go test ./internal/cli ./internal/pose -run '^(TestUpdatePreview|TestUpdateDryRun|TestUpdateOperation|TestStaleInstanceEngine)' -count=1` | Both packages pass; `b29-focused-final.log` (CLI 4.558s, model 0.025s). Includes the compiled-CLI inotify audit with usage enabled. |
| `go test ./... -count=1` | All packages pass; `b29-go-test-final.log` (CLI 43.277s, model 34.968s). Earlier failed runs and corrections are described above; no passing result was inferred from them. |
| `go vet ./...` | Exit 0, no findings. |
| `go test -race ./internal/cli ./internal/pose -run '^(TestUpdatePreview|TestUpdateDryRun|TestUpdateOperation|TestStaleInstanceEngine|TestIndexGeneration|TestFacts)' -count=1` | Both packages pass; `b29-race.log` (CLI 19.662s, model 2.931s). These are affected race tests, not a claim of running every repository test under race detection. |
| `go test ./internal/cli -run '^TestUpdateDryRunFollowsASymlinkedPolicy$' -count=20` | Pass after the historical helper's timestamp-ID normalization (1.236s). |
| `go mod verify` | Exit 0: `all modules verified`. |
| `CGO_ENABLED=0 GOOS=<os> GOARCH=<arch> go build -trimpath -o /var/tmp/pose-b07-validation-0bf729f/b29-pose-<os>-<arch> ./cmd/pose` | Exit 0 for the six concrete pairs from `.goreleaser.yaml`: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`. This is build evidence, not runtime testing on macOS/Windows or a release. |
| `git diff --check` | Exit 0. |

The focused integration selector is registered, nonempty and executed directly. The full suite includes existing compatibility, machinery, configuration-review, scoped request, index-generation, parser/binding and source-discovery regressions. Tests that themselves require unavailable authenticated release/network history retain their existing skip contracts; this run is not release compatibility certification. Tool availability limits for vulnerability/lint/secret scanners remain stated above; the changed diff was inspected for accidental secrets, and new fixtures use synthetic identities and content.

R1, R2 and R3 have semantic implementation evidence in the tests above. No implementation blocker remains identified in the exercised shared update path. The independent reviewer must assess the broader propagated call graph and evidence; this implementer does not qualify its own work as independently reviewed, sealed or closed. Preview still states the ADR's runtime boundary: later apply can fail under concurrency, ACL/security policy, disk/quota, durability or platform/provider failures not proven by a read-only observation. Unknown evaluation yields `INCOMPLETE`; it never becomes a successful no-op.

### Exact continuation paths

These paths differ from local merged checkpoint `e91a39f`; the preserved replacement propagation in that checkpoint is declared cumulatively in the spec. This is a worktree inventory, not immutable artifact reconciliation. Leave every listed change uncommitted for the parent.

- `.pose/indexes/validation-matrix.json`
- `.pose/reports/2026-10-22-b29-update-preview-implementation.md`
- `.pose/specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.md`
- `pose-mcp/internal/cli/assess.go`
- `pose-mcp/internal/cli/check.go`
- `pose-mcp/internal/cli/maintenance.go`
- `pose-mcp/internal/cli/managed_docs.go`
- `pose-mcp/internal/cli/onboarding.go`
- `pose-mcp/internal/cli/release_lifecycle.go`
- `pose-mcp/internal/cli/review_closeout_test.go`
- `pose-mcp/internal/cli/stack_seed.go`
- `pose-mcp/internal/cli/update_dryrun.go`
- `pose-mcp/internal/cli/update_dryrun_test.go`
- `pose-mcp/internal/cli/update_preview_linux_test.go`
- `pose-mcp/internal/cli/update_preview_test.go`

## Parent adjacent-path regression — 2026-10-23

After the separate implementation continuation completed, the primary added `TestUpdateOperationTrailingSeparatorRequiresDirectory` in the already declared operation tests. Before the fix, literal file paths and absolute symlink targets ending in a separator incorrectly allowed preview reads/writes while native operations refused them; `/tmp/pose-b29-trailing-before.log` records all four failures. Preserve the trailing directory requirement by appending a dot traversal component instead of dropping the separator in `splitComponents`. Focused update/operation/preview/stale-instance tests pass after the correction (model 0.044s, CLI 5.938s), log `/tmp/pose-b29-trailing-after.log`. Earlier full-suite/race/build results above remain explicitly results of the preceding source; fresh native final collection follows the consolidated source commit. No concurrency mechanism, dependencies or native apply operation changed.
- `pose-mcp/internal/cli/update_tree_test.go`
- `pose-mcp/internal/cli/usage.go`
- `pose-mcp/internal/pose/discovery.go`
- `pose-mcp/internal/pose/docs.go`
- `pose-mcp/internal/pose/index_generation.go`
- `pose-mcp/internal/pose/knowledge.go`
- `pose-mcp/internal/pose/operation_files.go`
- `pose-mcp/internal/pose/operation_files_test.go`
- `pose-mcp/internal/pose/spec_facts.go`
- `pose-mcp/internal/pose/update_preview_facts_test.go`


Parent provenance note: actual continuation session 01a16457-e576-7b42-8f04-9bcde86f4e63, logs /home/go/.local/state/secondary-agent/20261022T130043-codex-2377776. Primary inspected final response and supplied results; no independent approval is inferred. Next checkpoint records this implementation before composing reviewed B28/B38; their new readers and projected-final-gate paths must be requalified after integration.


## B28/B30/B38 merge composition — continuation resumed 2026-10-23

Implementer evidence for worktree `/tmp/pose-pending-b29`, `HEAD` `56449e47764280a7dc7eab8b986fc6c1d955d8af`, incoming `MERGE_HEAD` `fe765cf8aaaaff3ec93e91818e69e0781f92116a`. Compare stages 2 and 3 of the three production conflicts. Reconcile their contents without staging, committing, aborting or replacing the merge. Git's three `UU` index entries intentionally remain for the parent to stage after review; no conflict markers remain in their worktree contents. Leave the named-check matrix union unchanged. No native global validation, index generation, sealing, attestation, closeout or release action was run for this composition. Earlier local B29 and incoming B28/B38 evidence does not substitute for the results here.

### Source changes and preserved contracts

Declare additional exact artifacts in the canonical B29 spec before production/test edits. Reconcile `cli/release_lifecycle.go`, `pose/release_archive.go` and `pose/release_lifecycle.go` by retaining the incoming strict algorithms and carrying the explicit B29 filesystem through every reachable current-tree read. Preserve strict/duplicate-key manifest decoding, fragment recomposition and attribution, frozen tagged policy/content, missing versus corrupt refs, `ListReleaseTags` successful-stderr failure, `Sound()` positive-consumer guards and historical-cut behavior. Real committed blobs/trees and refs remain Git reads.

Add explicit `In` readers for `ProvenAbsent`, candidate gaps/supersession, strict changelog policy, release/version policies, archive identities, frozen-file gaps and version source confinement/reads. Store status, release checker witnesses, journals and candidate/tagged worktree comparisons use the operation's Files. Native wrappers keep their existing call signatures and strict behavior. Newly staged files participate in both existence proofs and content validation; unreadable paths and broken links do not become absence.

Port B28's `ruleFileState`, workflow/skill availability and suggest-domain confinement readers to optional explicit Files with native defaults. Add `cmdSuggestIn`; ordinary CLI/MCP routing still uses the native wrapper. The merged checker's task-map and required-reference reads already carry `checker.fsys()` and are retained. Original source dependency discovery remains physical and read-only; it was not replaced with projected source copies. Preserve B30's committed trailer history, pinned generation/latest-facts bindings and shared result parsing; its staged policy/result/history regression remains active.

Remove the newly reachable B38 tagged-version decoder's physical temporary-directory materialization. Decode authoritative `gitBlobAt` bytes with the same bounded text/JSON parser used by the native regular-file reader. Keep blob type/path and tagged policy/source validation before decoding. This removes transient preview writes without substituting worktree bytes or suppressing any version-source gap.

Changed production paths in this composition:

- `pose-mcp/internal/cli/release_lifecycle.go`
- `pose-mcp/internal/cli/rule_extension_resolver.go`
- `pose-mcp/internal/cli/suggest.go`
- `pose-mcp/internal/pose/path_state.go`
- `pose-mcp/internal/pose/release_archive.go`
- `pose-mcp/internal/pose/release_candidate.go`
- `pose-mcp/internal/pose/release_lifecycle.go`
- `pose-mcp/internal/pose/release_manifest_integrity.go`
- `pose-mcp/internal/pose/release_version_source.go`

### Composition regressions and observed failures

- Add `cli/update_preview_composition_test.go`: `TestUpdatePreviewForceStagesInstructionsBeforeFinalGate` deletes a shipped rule, workflow and skill, then runs the real shared force operation. The projected final gate succeeds while physical references stay absent, the B28 availability reader sees staged files, and fixed-time apply emits exactly the preview's semantic output. No final gate is bypassed.
- In the same file, `TestUpdatePreviewReleaseCurrentTreeAndHistoricalTagStaySeparate` checks staged version/policy candidate defects, later-policy tolerance for an immutable tagged cut, staged archived-fragment corruption and strict staged-manifest refusal. The correct worktree cannot replace the cut held by the tag. A valid text-source tag remains readable with nonexistent `TMPDIR`.
- Add `pose/update_preview_release_test.go`: `TestUpdateOperationReleaseReadsStagedInventoryAndProvesAbsence` removes physical release inputs, stages their bytes and directories, and checks sound Store status plus archival witnesses. The staged policy exists despite its physical absence; a staged dangling policy link is unreadable, not missing.
- Extend `cli/update_preview_linux_test.go`: the actual compiled CLI's force preview now also reaches an adopted governed release and its committed text-version snapshot. Prepare/tag fixture inputs before installing watches. With usage enabled, inotify watches the instance, `.git`, external link targets and `TMPDIR`, including transient create/remove/lock events. Both plain and force cases pass without mutation. This is a filesystem-event audit, not a syscall audit.
- The first complete `go test ./... -count=1` failed only `TestR9AVersionSourceThatCouldNotBeReReadIsNotApproved`: its old fault injection made `TMPDIR` nonexistent to break the former materialization helper. Update that declared test in `cli/release_review_regression_test.go` to retain the fail-closed requirement at the authoritative input: missing, malformed and oversized committed version sources are rejected even after the current worktree is repaired. The positive control passes with nonexistent `TMPDIR`, reflecting the new in-memory decoder. A first targeted revision exposed test-fixture `t.TempDir` inheriting the deliberately invalid environment; scope that environment to the control subtest. The corrected targeted test passes. No production guard was relaxed.

### Executed composition validation

Run from `pose-mcp` as uid 1000 with `GOTOOLCHAIN=go1.26.10 TMPDIR=/var/tmp/pose-b07-validation-0bf729f GOFLAGS=-buildvcs=false`. Fixed test time stays 2026-10-22 by design. The validation directory contains logs and binaries, not native governance attestations.

- Focused final command: `go test ./internal/cli ./internal/pose -run 'UpdatePreview|UpdateDryRun|UpdateOperation|StaleInstanceEngine|Suggest|DistributedInstructions|Facts|ReleaseCorrupt|BrokenNewerTag|Manifest|Snapshot|CorrectCut|Release.*Historical|TwoRelease|TestR9AVersionSource' -count=1` — pass; CLI 8.036s, model 8.870s. Log: `/var/tmp/pose-b07-validation-0bf729f/b29-composition-focused-r2.log`. Includes original discovery/execute-only-directory, shared operation decisions, request/no-op/refusal controls, B28 availability, B30 facts, B38 corrupt/broken-newer tags and the expanded compiled-CLI mutation audit.
- `go test ./... -count=1` — final rerun passes every package; CLI 48.362s, model 41.837s. Log: `/var/tmp/pose-b07-validation-0bf729f/b29-composition-full-r2.log`. This includes B38 R2–R12, manifest/tag historical controls and all existing two-release consumers. Preserve the first failing log at `b29-composition-full.log` as the reproduction above.
- `go vet ./...` — pass after the final test correction, empty log `/var/tmp/pose-b07-validation-0bf729f/b29-composition-vet-r2.log`.
- Initial focused race command over update/suggest/facts/release/manifest/snapshot selectors — pass; CLI 35.007s, model 28.386s, log `b29-composition-race.log`. `go test -race ./internal/cli ./internal/pose -count=1` also passes in full after the test correction: CLI 82.201s, model 68.802s; log `/var/tmp/pose-b07-validation-0bf729f/b29-composition-race-full.log`.
- Six CGO-free `go build -trimpath -o <validation-path> ./cmd/pose` builds — pass for the `.goreleaser.yml` Cartesian product `GOOS=linux,darwin,windows`, `GOARCH=amd64,arm64`, `CGO_ENABLED=0`. Binaries are `/var/tmp/pose-b07-validation-0bf729f/b29-composition-<os>-<arch>[.exe]`; corresponding `b29-composition-build-<os>-<arch>.log` files are empty. Final subsequent edits affect tests/report/spec only, not these production inputs.
- `git diff --check` — pass. Read-only `git rev-parse HEAD MERGE_HEAD` confirms the original merge heads above; no staging or Git mutation was performed.

Keep the spec `in-progress`. Parent still owns source freeze, any pending-PR reconciliation, native global matrix evidence (expected named union 146), immutable artifact reconciliation, staging/merge commit and fresh separate final review. No independent approval, vendor/human identity proof or spec closure is claimed by this implementer report.


## Source-review fixes — 2026-10-23

Implementer: task-designated Codex gpt-6.1-sol, continuing committed source `7d9c07cc722318fba3789887f81aa0bb0b5e937a` in `/tmp/pose-pending-b29`. Read the canonical spec, accepted D2 ADR, prior implementation evidence and actual source review `.pose/reports/2026-10-23-b29-pending-review-r1.md`. Preserve that review and the historical rejection unchanged. This section records implementation evidence, not an independent approval or attestation. No agents or secondary reviews were launched. The parent owns source freeze and the next independent review, which the user requires to use Claude Opus. Keep the spec `in-progress`.

The task prompt and canonical Validation section supplied the prospective regression plan. Declare the one new artifact, `operation_files_access_linux_test.go`, in the canonical spec before editing tests. Reuse the already declared operation-files production/tests and shared CLI test file. No new architectural contract or POSIX emulator is introduced; unsupported knowledge remains an explicit D2 gap.

### Reproduction before production edits

All logs below are under `/var/tmp/pose-b07-validation-0bf729f/`. Run Go commands from `pose-mcp`, as uid 1000, with `GOTOOLCHAIN=go1.26.10 TMPDIR=/var/tmp/pose-b07-validation-0bf729f GOFLAGS=-buildvcs=false`.

- `go test ./internal/pose -run 'TestUpdateOperation(AbsoluteLinkComponentTraversal|RestrictiveUmaskSubprocess|HardLinkDirectAndAtomic)$' -count=1` failed on unchanged production. `b29-r1-before.log` records the initial result, including a harness error: the first restrictive umask persisted into the next subtest's fixture setup. Restore the previous mask at each subtest's end, exclusively inside the disposable child. Re-run before editing production; `b29-r1-before-fixed-harness.log` records all actual medium failures without the harness error: missing absolute-target component accepted, symlink-before-dot-dot selected the lexical victim, staged child write/stat/read succeeded under umask `0111` where native access failed, both exclusive-file reread APIs succeeded under `0400` where native access failed, and direct hard-link overwrite returned success with no gap.
- `TestUpdateOperationHardLinkDirectAndAtomic` retains successful real apply controls: direct overwrite changes the external alias; atomic publication replaces one entry and leaves the alias unchanged. Preview itself changes neither physical entry. Its atomic branch passed even before the fix.
- Adjacent component-walk scouting found physical directory search before dot-dot was also bypassed. Add an unprivileged `physical-search-before-dotdot` child subtest before correcting that path. `go test ./internal/pose -run '^TestUpdateOperationRestrictiveUmaskSubprocess$' -count=1` failed with `physical search skipped: <nil>` while native access returned permission denied; `b29-r1-before-physical-search.log` preserves this second reproduction. The earlier staged-permission subtests already passed at that point.

### Corrections and exercised behavior

- M1: preserve original absolute symlink target components when resetting the root; also preserve components of the incoming path. Traverse each component before interpreting dot-dot. Search checks cover staged directories and physical dot/dot-dot components; physical checks use effective-access kernel queries and disclose unsupported queries. `TestUpdateOperationAbsoluteLinkComponentTraversal` compares real reads/writes for absent components and symlink components, and confirms the resolved external location. `TestUpdatePreviewSourceReviewFilesystemReachability` reaches these layouts through the shared force update and confirms the lexical victim is untouched.
- M2: use one staged permission gate for `ReadFile`, `Open`, `ReadDir`, directory search and subsequent writes. New staged entries belong to the effective creator; sufficient owner rights permit the modeled access. Missing owner rights do not prove denial under root, capabilities or inherited ACLs: conservatively return `ErrPreviewUnsupported` and persist an evaluation gap. Do not optimistically succeed or deny a privileged operation based only on owner bits. A direct overwrite retains its physical origin, so subsequent staged access uses the kernel's effective-access query on the original entry, preserving real ownership/ACL behavior. Existing native reads and physical write-access checks remain. Restrictive umask tests run in a real unprivileged disposable child; no production or parent-test process changes its umask.
- M3: read link count from safe Unix inode metadata before direct overwrite. Multiply linked files, or unavailable link-count metadata, return unsupported and record a gap without staging a misleading single-path overwrite. Do not scan for aliases. Atomic publication remains independent and can sever the link relationship; a subsequent direct overwrite of its staged replacement does not inherit the old inode restriction. Shared force-update regression requires nonzero status, `INCOMPLETE` and the inode-alias diagnostic even through best-effort callers.
- Low unknown umask: return both value and observation availability. Read Linux `/proc/self/status` only on Linux; remove the guessed `022` fallback. Missing safe knowledge on macOS/other platforms, or unavailable Linux status, makes needed direct/exclusive/directory creation evaluation unsupported with a persistent gap. `TestUpdateOperationUnknownUmaskIsIncomplete` injects unavailable knowledge and checks all three paths, no optimistic staged node and no physical creation. Atomic publication's explicit final chmod mode remains usable without umask knowledge when its parent already exists.

### Executed final-source validation

All commands in this table exited 0 with the environment above. No production or test edits followed these final runs. Report-only edits record the results.

| Command (relative to `pose-mcp`) | Actual result and log under validation root |
|---|---|
| `go test ./internal/cli ./internal/pose -run 'TestUpdatePreviewSourceReviewFilesystemReachability\|TestUpdateOperation' -count=1` | Initial corrected regressions pass: CLI 0.225s, model 0.046s; `b29-r1-regressions.log`. This precedes the physical-search extension and does not replace final evidence. |
| `go test ./internal/cli ./internal/pose -run '^(TestUpdatePreview\|TestUpdateDryRun\|TestUpdateOperation\|TestStaleInstanceEngine)' -count=1` | Final focused pass: CLI 6.283s, model 0.039s; `b29-r1-focused-final.log`. Includes every new regression, unknown-umask injection and shared-command reachability. |
| `go test ./... -count=1` | All packages pass, CLI 58.805s, model 52.614s; `b29-r1-full-final.log`. |
| `go vet ./...` | Pass, empty `b29-r1-vet-final.log`. |
| `go test -race ./internal/cli ./internal/pose -count=1` | Both complete affected-package suites pass, CLI 92.095s, model 78.681s; `b29-r1-race-full-final.log`. No focused selector was used. |
| `go test ./internal/cli -run '^TestUpdatePreviewHasNoPhysicalMutationEvents$' -count=1 -v` | Plain and force compiled-CLI previews pass with usage enabled; 2.483s; `b29-r1-inotify-final.log`. |
| `CGO_ENABLED=0 GOOS=<os> GOARCH=<arch> go build -trimpath -o /var/tmp/pose-b07-validation-0bf729f/b29-r1-final-<os>-<arch>[.exe] ./cmd/pose` | All six pairs pass: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64. Empty per-pair logs: `b29-r1-build-final-linux-amd64.log`, `b29-r1-build-final-linux-arm64.log`, `b29-r1-build-final-darwin-amd64.log`, `b29-r1-build-final-darwin-arm64.log`, `b29-r1-build-final-windows-amd64.log`, `b29-r1-build-final-windows-arm64.log`. |
| `git diff --check` (repository root) | Pass after source/test edits and again after report edits. |

The first full-suite/vet/full-race/audit/build batch (`b29-r1-full.log`, `b29-r1-vet.log`, `b29-r1-race-full.log`, `b29-r1-inotify.log`, `b29-r1-build-<os>-<arch>.log`) also passed. The subsequent physical-search production correction and its new regression justify the final batch above; do not use the earlier batch as final-source proof. `b29-r1-after-initial.log` records the first operation-model pass after the three original fixes. Two mistakenly launched Go commands from the repository root failed to find the module; they executed no tests and supply no validation evidence. The corrected module-directory invocations are the ones recorded here.

### Limits and handoff

This is deliberately conservative permission evaluation, not a full privilege/ACL simulator. A newly staged entry missing needed owner rights yields INCOMPLETE even when elevated privilege might make native apply succeed. Unsupported platforms do not invent link counts or access facts. macOS/Windows builds establish compilation only; they do not establish runtime parity. Creation requiring unavailable umask knowledge is explicitly incomplete. Existing future concurrency, ACL/security-policy, quota, durability and provider boundaries remain; direct hard-link inode alias emulation is unsupported rather than advertised as complete. No full-filesystem alias search, global preview switch, production umask mutation or physical preview scratch file was added. Native apply methods and publisher behavior were not changed.

The inotify audit observes filesystem mutation events, including transient staging/locks, with usage enabled; it is not a syscall audit or proof over every possible layout. Scanners previously documented as unavailable were not fabricated. No dependencies changed. The pre-edit native 146/146 result on `7d9c07c` cannot qualify this edited source. No native global commands, index generation, sealing, closure, contribution creation, staging, commits, pushes, merges, releases or adoption were performed by this continuation. Pre-existing parent assessment/index/state/result/report changes were left alone.

Exact paths edited by this continuation (all uncommitted):

- `.pose/specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.md`
- `.pose/reports/2026-10-22-b29-update-preview-implementation.md`
- `pose-mcp/internal/pose/operation_files.go`
- `pose-mcp/internal/pose/operation_files_access.go`
- `pose-mcp/internal/pose/operation_files_access_other.go`
- `pose-mcp/internal/pose/operation_files_test.go`
- `pose-mcp/internal/pose/operation_files_access_linux_test.go` (new)
- `pose-mcp/internal/cli/update_preview_test.go`


## Parent R2 remediation — 2026-10-23

Actual separate Codex source review R2 (run `/home/go/.local/state/secondary-agent/20261023T042723-codex-3928768`) confirmed all R1 items addressed and requested changes for special-file direct writes and unreadable-directory removal. Its unmodified response follows R1 in the review report. No rejected review is retroactively approved.

New tests reproduced both before production edits (`/tmp/pose-b29-r2-before.log`): direct device writes staged regular bytes; the shared force-update device link returned complete; unprivileged empty mode-000 directories were removable natively with both Remove and RemoveAll, while the preview treated enumeration failure as native refusal without a gap. The shared force-update skill-directory fixture also lacked INCOMPLETE.

Direct physical overwrite now requires a regular file; unsupported special-file effects persist a gap. Failure to observe directory emptiness or recursive contents persists an unsupported-evaluation gap instead of claiming removal must fail. Atomic replacement remains independent. Focused full update/operation/preview/stale-engine selectors pass (model0.058s, CLI6.290s), `/tmp/pose-b29-r2-after.log`. Tests include the real shared force-update caller, a native successful empty-directory removal/relink control, and subprocess permission controls as uid1000. No new production paths or dependencies were added. Native apply stays unchanged.

Previous corrected-source native collection at e7b768e yielded145/146: only published-release journey failed with HTTP403. Direct response headers confirmed public GitHub quota60, remaining0. This is failed evidence, not waived success. Final collection uses existing configured gh authentication in child environment only; token bytes are never printed or written to artifacts. Earlier local govulncheck found no vulnerabilities and operation race passed2.097s. Six final e7b768e builds passed; these are preceding-source evidence after the R2 production edits.


## Parent R3 sticky-directory remediation — 2026-10-23

Separate source R3 confirmed previous fixes and found that parent write/search permission does not establish sticky-directory replacement authority. The source-derived foreign-owned layout was not recreated using privileged ownership changes. Instead, unprivileged owned sticky fixtures enforce conservative unsupported behavior for Remove, RemoveAll, nested recursive entries and atomic Publish; native owned-entry controls remain permitted. Shared force-update coverage also verifies INCOMPLETE while native owned-entry replacement succeeds. Before the fix all five guard tests failed (`/tmp/pose-b29-r3-before.log`).

The operation view now detects physical sticky parents before removal or replacement, including recursive child entries. It persists an unsupported-authority gap without fabricating foreign ownership or privilege facts; creation and direct overwrites remain separate operations. Focused operation/update/preview/stale-engine tests pass (model0.144s, CLI16.305s), `/tmp/pose-b29-r3-after.log`. This is deliberately conservative even for an owned entry; no claim of an independently executed foreign-owned native refusal is made. Native apply remains unchanged. Fresh native validation and separate review follow.

Initial artifact reconciliation found14 missing action observations. Ten changes were performed in a merge commit, whose trailer-based observation did not expose these paths; a native report with an explicit immutable baseline-to-source range will qualify those changes without changing engine attribution or pretending they were normal commits. Four prospectively listed files were not touched at all (`review_assurance.go`, `review_policy_baseline.go`, `governed_effects.go`, `review_trust_time.go`); remove their unused declarations. No source change or artifact success is inferred from declared intent.


## Parent provider and inherited-mode qualification — 2026-10-23

Prospective Validation was extended before production edits for provider and default-ACL uncertainty. Actual tests then reproduced two related gaps (`/tmp/pose-b29-provider-before.log`): `/proc/self/comm` is regular but truncates native writes instead of storing arbitrary bytes; default ACL inheritance removed child directory search permission despite the requested mode/umask. The native virtual-file write is confined to a disposable test child's task name. Linux default ACL controls ran as uid1000 and did not skip.

Physical direct overwrites now require an observed ordinary storage provider; unknown and virtual providers remain INCOMPLETE. The initial bounded Linux provider set is ext, xfs, btrfs, tmpfs, overlay, nfs, f2fs and ramfs; other platforms/providers do not get invented byte semantics. New regular-file/directory creation checks for default-ACL inheritance on a physical parent before predicting mode, while newly staged parents inherit only already-qualified mode knowledge. Unknown observation or a default ACL produces INCOMPLETE. Known explicit-mode atomic publication remains separate. This is bounded observation, not a universal filesystem/privilege emulator.

Special-file reads/opens are also unsupported rather than invoking device/FIFO semantics as ordinary contents; `/tmp/pose-b29-special-read-before.log` records the previous silent success controls on the null device. Final focused operation/update/preview/stale-engine suite passes (model0.052s, CLI6.367s), `/tmp/pose-b29-provider-final-focused.log`. No dependencies, CLI flags, native apply methods or source-discovery semantics changed. Fresh full native qualification, platform builds and separate review follow this source checkpoint; older146/146 runs are preserved as earlier-source evidence.


## Parent R4 reporting remediation — 2026-10-23

Actual separate R4 response confirms earlier fixes but rejects a raw special-file comparison in net-change reporting. Both new no-writer FIFO subprocess regressions reproduced the hang before correction (operation timeout5s, real shared-update schema timeout10s), `/tmp/pose-b29-r4-before.log`. The source's ordinary read guards did not cover `netChange`'s direct original-content comparison.

Classify nonregular original entries from metadata and describe atomic replacement without opening them. Original content comparison also keeps unknown provider semantics incomplete rather than performing a raw read on an unqualified provider. Add corresponding special-replacement verbs in pt-BR rendering; preserve native apply. Operation Publish→Changes uses an IN_OPEN watch and verifies no FIFO opening or physical replacement, while the shared update with FIFO schema terminates with INCOMPLETE and preserves its physical FIFO. Final focused operation/update/preview/stale-engine tests pass (model0.055s, CLI6.428s), `/tmp/pose-b29-r4-after.log`. Parent-owned subprocess temporary roots ensure even timeout reproductions are cleaned by the parent.

On source24e3328, artifact reconciliation passed with0errorfindings after explicit immutable-range observation and surface-check passed with0errorfindings. These are preceding-source gates; fresh native collection, index, artifact/surface and actual sealed review follow the reporting correction. No prior source rejection is relabeled approved.


## Parent R5 partial-removal remediation — 2026-10-23

Actual separate R5 confirms earlier corrections but identifies that recursive preflight can return a known permission refusal while native RemoveAll has already removed accessible siblings. New unprivileged controls reproduce both nested0555-child and nonwritable-parent variants, proving native partial deletion before checking preview's missing gap. A real force-update retention fixture creates five aged, noncurrent generation directories and proves native cleanup deletes an ordinary sibling while retaining the protected child. All three old preview paths lacked INCOMPLETE (`/tmp/pose-b29-r5-before.log`).

Any recursive preflight failure now retains an unsupported partial-effects gap; inability to remove a directory through its parent also remains incomplete because the native call may already have emptied it. Known direct non-directory failures remain distinct. No exact recursive-delete emulator or native apply change was introduced. Focused operation/update/preview/stale-engine tests pass (model0.057s, CLI6.376s), `/tmp/pose-b29-r5-after.log`, including actual retention reachability. Fresh current-source collection and separate adjudication follow.
