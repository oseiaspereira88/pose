**Decision: changes requested.** Three functional findings remain; final validation and metadata closeout are separately pending.

Reviewer: Codex, OpenAI, agent `/root`, 2026-10-20. This is a fresh **same-vendor review**. Cross-vendor review was unavailable because of the weekly limit.

Subject: baseline `1e0a313ba1d87cebd4a355eeb3b83c0bd080e5e8` through HEAD `210fc468bb18d0deb966d70ba65694f491d060d8`, plus observed working edits, including the latest PR195 reconciliation and all eight B07 probe files. The index still contains unresolved merge entries. Findings below concern current file contents.

Only file and Git reads were performed. No edits, commits, pushes, external messages, test execution, sealing, or attestation occurred.

**Findings**

1. **High — persisted-run freshness can accept loss of Git metadata.**  
   [review_dispatch.go:695](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/review_dispatch.go:695), [review_dispatch.go:868](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/review_dispatch.go:868)

   `ReviewRunStillGoverns` delegates to `dispatchNoLongerGoverns`. That function checks HEAD only when `hasGitMetadata` returns true; `sourceMaterializationDiffers` also returns success when metadata is absent. Meanwhile, bundle preparation retains the exported-tree status fallback.

   Consequently, removing Git metadata after a completed dispatch can bypass revision and executable-mode verification while persisted bundle, policy, transcript, and semantic bytes remain readable. This consumer can report that the run still governs without establishing the Git preconditions required of a dispatch.

   **Action:** require readable Git identity and revision unconditionally when evaluating a persisted dispatch’s current relevance. Preserve exported-tree behavior for historical projections separately. Add an actual-dispatch regression that removes `.git`, changes executable mode, and verifies rejection by `ReviewRunStillGoverns`.

2. **Medium — closeout output exemptions suppress deletion and type replacement.**  
   [closeout_plan.go:244](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/closeout_plan.go:244), [closeout_plan.go:273](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/closeout_plan.go:273)

   `ungeneratedChanges` detects materialization differences, then removes them solely through `closeoutProducedPath`. Exact checkpoint, configured results, and canonical index names return true without requiring an existing regular file or checking ancestor links.

   For example, deleting a tracked `.pose/indexes/repo-map.json`, or replacing the exact scope checkpoint with a symlink, is classified as generated and omitted from the dirty result. The shared `WorktreeDivergenceFor` exemption path already requires `dispatchRegularInputPath`; closeout does not.

   **Action:** validate materialization before granting these exemptions. Keep deletions, special files, and symlink replacements as divergence or unknown. Test the exact exempt names, including the configured results path. This finding establishes a false context-clean result; it does **not** establish that every subsequent closeout gate can be bypassed.

3. **Medium — run history remains lexically ordered, and its ordering probe can pass vacuously.**  
   [review_dispatch.go:179](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/review_dispatch.go:179), [b07_r4b_probe_test.go:182](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/b07_r4b_probe_test.go:182)

   `ReadReviewRuns` compares `StartedAt` strings. Thus `00:00:00.1Z` sorts before the earlier `00:00:00Z`; offsets also invalidate textual ordering.

   The new probe copies a persisted run and clears only `RunID`, retaining `TranscriptPath`. `recordReviewRun` hashes that retained path before replacing it, while validation clears the path before recomputing identity. Both added records can therefore be rejected. The probe ignores read errors/losses and accepts `i == j == -1`, so it need not exercise ordering.

   **Action:** order parsed instants with a deterministic tie-breaker. Construct valid fresh records in the probe and assert that both records were read with no losses before comparing positions. The production comparator predates this reconciliation; the added regression does not establish its correction.

**Requirement conclusions**

These are static-review conclusions, not executed-check results.

| Requirement | Conclusion |
|---|---|
| R1 — unknown Git never certifies eligibility | **Failed:** persisted-run fallback in finding 1. |
| R2 — hidden bytes, absence, links and types | **Failed:** findings 1–2 leave materialization gaps. Dispatch’s direct guards otherwise inspect bytes and modes independently of index flags. |
| R3 — exact paths and boundaries | **Passed on inspection** for NUL status/rename handling and dispatch ancestor checks; closeout exemption handling needs finding 2 corrected. |
| R4 — explicit output authority | **Failed in closeout** by finding 2. **Passed on inspection** for dispatch’s stricter no-adapter-writes behavior. |
| R5 — stale history versus current fulfillment | **Failed overall** by finding 1. Record/transcript integrity, supersession-loss rejection, semantic re-preparation, and late-publication revalidation are present. |
| R6 — semantic relevance and worktree identity | **Passed on inspection** at dispatch execution boundaries; persisted-run Git-loss handling remains deficient. |
| R7 — diagnostics, coverage and bounded retry | **Needs evidence**, with finding 3 affecting reliable history ordering. Typed Git failures and persistent-incoherence rejection are present. |
| R8 — regression coverage and positive controls | **Needs evidence:** broad probe coverage exists, but the ordering probe is ineffective and findings 1–2 need targeted cases. |

**Review dimensions**

| Dimension | Conclusion |
|---|---|
| Requirements | **Failed** pending the findings above. |
| Correctness | **Failed** for current-relevance and closeout classification. |
| Security | **Failed** for those integrity boundaries. No sandbox containment is claimed. |
| Compatibility | **Passed on inspection:** MCP `include_plan` is additive and opt-in; stricter refusal is explicitly classified minor. Runtime compatibility needs fresh validation. |
| Scope | **Passed on inspection:** B07 changes and separate waiver retirement have identifiable purposes; the baseline also includes inherited B06 changes. |
| Documentation | **Needs evidence/consolidation:** spec remains `in-progress`, appropriately withholding closeout. Historical evidence must remain distinguished from the reconciled subject. |
| Operability | **Needs evidence:** useful failure causes, retained historical transcripts, process cleanup, and bounded disk observations exist; finding 3 weakens history presentation. |

The cheapest formal bypass examined was obtaining a clean/current classification through output naming or exported-tree fallback. Dispatch blocks directory-name write authority, but findings 1–2 show that these defenses are not yet consistent across consumers.

**Separate waiver retirement**

`d957b4f` **passes static scope/security review**: it removes the expired exception and updates its explanatory comment. It changes neither workflow pins nor waiver/tag enforcement. The reported workflow-contract pass was not independently rerun here.

**Rules and evidence limits**

Applied `pose-review`, `.pose/workflows/review.md`, and the backend-Go, security, documentation-style, delivery-evidence, and knowledge-governance rules. The relevant ADRs and B07 handoff were consulted.

The read-files-and-Git-only restriction prevented executing tests, integration/debt assessments, recurrence checks, strict validation, or native review commands. No current bundle/plan digest was independently resolved, and no approval or human confirmation is asserted.

The supplied full-suite pass and **11/11 mutation result predate the latest reconciliation** and are not treated as fresh evidence. Historical PR195 validation likewise does not validate the present working tree. After correcting the findings and resolving the merge index, obtain fresh probe, full-suite, race, security, native strict-validation, artifact, and surface evidence bound to the final subject.

Before/after observations do not provide sandbox containment, prevent transient restored edits, or establish atomicity against arbitrary external filesystem/Git writers. This report is independent reviewer judgment, **not a POSE approval or completed closeout**.