**Decision: changes requested.** The three R1 findings are corrected on inspection. One adjacent functional gap remains; final validation and metadata closeout are separately pending.

Reviewer: Codex, OpenAI, agent `/root`, 2026-10-20. This is a **separate same-vendor execution**. Cross-vendor review was unavailable because of the weekly limit. No cross-vendor approval or human confirmation is asserted.

Subject: baseline `1e0a313ba1d87cebd4a355eeb3b83c0bd080e5e8` through HEAD `a4c58e91cba769de833c7f06f4b6ed33e89ce37a`, plus observed working edits. The merge index has no unresolved entries. Metadata changed concurrently during inspection, so this report does not identify an immutable sealed working-tree subject.

**Remaining finding**

**Medium — persisted-run relevance omits the full materialization guard.**

Locations: [review_dispatch.go:762](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/review_dispatch.go:762), [review_dispatch.go:690](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/review_dispatch.go:690), [review_dispatch.go:890](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/review_dispatch.go:890).

`ReviewRunStillGoverns` now requires readable Git identity and HEAD, but then delegates to `dispatchNoLongerGoverns` without calling `verifyBundleMaterialization`. Re-preparation compares semantic digests; its final materialization check examines subject entries only and explicitly skips submodules.

Two variants remain:

- After a valid completed dispatch, replace a consumed review profile or rule—outside the attributed subject—with a symlink to identical bytes. Profile and consumed-input loading follow the link, preserving the prepared digest. The persisted consumer can still report that the run governs, although dispatch’s regular-input guard would reject that materialization. See [review_closeout.go:610](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/review_closeout.go:610) and [review_bundle.go:1565](/tmp/pose-b07-closeout-work/pose-mcp/internal/pose/review_bundle.go:1565).
- For a persisted run whose subject includes a submodule, hide a tracked child edit with `assume-unchanged`. Re-preparation hashes the index’s gitlink pointer; the fallback materialization check skips the submodule, leaving its hidden child content unchecked.

**Action:** apply the shared bundle-materialization guard when deciding persisted current relevance, while preserving historical record access. Add persisted-consumer regressions for a same-byte consumed-input symlink and a hidden relevant-submodule edit, each with an intact positive control.

This is a static implementation finding, not an executed reproduction. It establishes a false current-relevance classification; it does not establish an attestation or closeout bypass.

**Prior findings and other inspected controls**

| Area | Conclusion |
|---|---|
| Persisted Git loss | **Passed on inspection:** unconditional identity and HEAD reads precede relevance evaluation; actual-dispatch metadata-loss regression added. |
| Closeout deletion/type exemptions | **Passed on inspection:** exemptions require regular materialization; nine exact-name deletion/symlink/directory cases added. |
| Run ordering and identity | **Passed on inspection:** parsed instants, deterministic ID tie-breaker, derived-field reset, and non-vacuous history assertions. |
| Persistence/transcripts | **Passed on inspection:** record identity, persisted-copy equality, transcript digest, missing records, and supersession losses are checked. |
| Publication revalidation | **Passed on inspection:** origin is checked before execution, after execution, and after completed-record publication; late conflict writes superseding stale history. |
| Output authority | **Passed on inspection:** adapters receive no derived-directory write exemption; ignored disk changes are compared. |
| Bounds | **Passed on inspection** for the declared disk observations: 100,000 entries and 256 MiB. This is not an overall adapter resource bound. |
| MCP compatibility | **Passed on inspection:** `include_plan` is additive and opt-in; omission preserves the existing response. |

All eight B07 probe files were inspected, including persistence, transcript, publication-window, semantic relevance, path, identity, and hidden-materialization cases.

**Requirement conclusions**

| Requirements | Conclusion |
|---|---|
| R1 — unavailable Git fails closed | **Passed on inspection** for the reviewed execution and persisted-consumer entrypoints. |
| R2–R3 — relevant materialization, types and links | **Failed** in persisted relevance by the remaining finding. Direct dispatch guards are stronger. |
| R4 — explicit output authority | **Passed on inspection.** |
| R5 — historical versus current relevance | **Failed** for the remaining materialization variants; stale/failed history separation is otherwise present. |
| R6 — semantic freshness and worktree identity | **Passed on inspection** at execution boundaries; persisted materialization needs correction. |
| R7–R8 — diagnostics and regression validation | **Needs evidence:** broad coverage exists, but the remaining variants need regressions and fresh execution. |

**Review dimensions**

| Dimension | Conclusion |
|---|---|
| Requirements | **Failed** pending the finding above. |
| Correctness | **Failed** for persisted current-relevance classification. |
| Security | **Failed** for that integrity boundary; no containment claim. |
| Compatibility | **Passed on inspection; needs fresh runtime evidence.** Stricter refusal is documented as minor. |
| Scope | **Passed on inspection:** B07 and separate waiver retirement are identifiable; the baseline also contains inherited B06 work. |
| Documentation | **Needs consolidation:** retain `in-progress` and distinguish historical validation from the final subject. |
| Operability | **Needs evidence:** diagnostics, bounded observations and retained history are present; fresh suite/race/native results remain pending. |

The separate waiver retirement `d957b4f` **passes static scope/security review**: it removes the unused expired exception and updates commentary without changing workflow pins or enforcement.

**Rules and evidence limits**

Applied `pose-review`, the review workflow, and backend-Go, security, documentation-style, delivery-evidence and knowledge-governance rules; consulted the B07 spec, relevant ADRs, handoff and prior report.

Only file and Git reads were performed. `git diff --check` passed. No tests, assessments, scanners, native validation, sealing, attestation, edits, commits, pushes or messages were performed. Current bundle/plan digests were not independently resolved.

The supplied full-suite, **11/11 mutation**, and **141/141 native-validation** results predate the latest fixes and are **not fresh evidence for this subject**. Final validation and metadata closeout remain evidence work, distinct from the functional finding.

Before/after observations do not establish sandbox containment, detect every transient restored edit, or provide atomicity against arbitrary external writers. This report records independent reviewer judgment, not POSE approval or completed closeout.