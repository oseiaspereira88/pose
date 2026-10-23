# B29 replacement implementation — in-progress, 2026-10-22

Current composition evidence is recorded in the B28/B30/B38 section below; earlier continuation results are historical baseline evidence.

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
