# B38 separate source review R1

Session 01a16230-2485-7891-b1aa-0513fb61303f. Actual separate Codex fallback after Claude weekly limit, same vendor, task-designated gpt-6.1-sol; no human/cryptographic/cross-vendor independence.

**Code verdict: changes requested — one medium finding.** Final collection and seal adjudication remain pending separately.

Reviewed PR198’s full `origin/main...HEAD` scope at `e27f1f75593cbca988f5f783ff0c563438e0a6cb`, against `origin/main` at `2d42b84e3a065bb30cf9e7882c19a4f2535453f7`, including the spec, historical review reports, corrective changes, and latest approved review/attestation.

**M1 — A broken newer tag ref can make `open-next` approve against an older verified release**

Sources:

- [gitOutput](/tmp/pose-pending-b38/pose-mcp/internal/pose/release_manifest_integrity.go:309) discards captured stderr when Git exits successfully.
- [GetReleaseStatus](/tmp/pose-pending-b38/pose-mcp/internal/pose/release_lifecycle.go:471) trusts that tag listing as a complete inventory.
- [appendReleasePolicyChecks](/tmp/pose-pending-b38/pose-mcp/internal/cli/release_lifecycle.go:838) likewise checks only the listing’s exit status.
- [cmdReleaseOpenNext](/tmp/pose-pending-b38/pose-mcp/internal/cli/release_lifecycle.go:700) approves using the first remaining release.

Historical R11-L14 records that a corrupt loose tag ref can produce Git’s `ignoring broken ref` warning while disappearing from the successful listing. Its low-severity rationale says the state “only goes down.” That rationale misses selection across releases: removing a newer release from the inventory can promote an older verified release into the position used for approval.

Reproducer, **in a synthetic fixture only**:

1. Keep an intact, sound `v1.0.0` with legitimate tagged/published/verified events and an adopted release policy.
2. Create a loose `v1.1.0` tag pointing at a commit without that version’s manifest. Keep no `.pose/releases/v1.1.0` directory.
3. Control: status includes `v1.1.0` with a gap; `release open-next --version v1.2.0` refuses.
4. Replace only the loose `refs/tags/v1.1.0` contents with invalid ref text.
5. Git omits that ref with a warning. Both inventory consumers discard the warning. Status then selects sound, verified `v1.0.0`, and `open-next` reaches its success branch.

This is a **source-derived consumer variant**, not a corruption probe executed in this session. The Git omission primitive is documented by the historical executed review; I independently traced its current production consequences.

**Why medium:** distinguishable loss of the release inventory changes a refusal into positive next-cycle approval. This affects R3/R7/R8 and the release-integrity rule. It does not require implementing provider-event replay from B39.

**Minimum necessary fix:** preserve broken-ref/incomplete-enumeration failures in a shared tag-inventory reader and propagate them through status and the structural checker. For explicitly requested tags, distinguish corrupt refs from proven absence. Add the two-release regression above, retaining controls for genuinely absent tags and projects that never adopted release governance.

**Historical findings and production integration**

The current implementation addresses the earlier blocking classes through strict manifest decoding, fragment-blob recomposition, historical policy binding, regular-file checks in tagged trees, candidate verification, worktree/tag binding, and `Sound()` checks in publication and next-cycle consumers.

I inspected the latest approved r13 report and unsigned attestation rather than adopting their conclusion. R12-F01’s unreadable/deleted-policy fix remains present and its registered regressions passed. R13-L20’s incomplete adoption witness remains a separate low residual: source inspection supports its narrower advisory effect. M1 instead extends R11-L14 into a positive approval path.

The canonical adversarial README has no dedicated case proving this broken-ref/two-release boundary.

**Checks executed in this session**

- `go test ./... -count=1` in `pose-mcp`: passed.
- Targeted domain/CLI release, historical-review, archive, snapshot and tag tests: passed.
- Targeted MCP release tests: passed.
- `go vet ./...`: passed.
- `git diff --check origin/main...HEAD`: passed.
- Added Go-line scan for `TODO`, `FIXME`, `panic(` and `stub`: no matches.
- `recurrence-check --tolerant --window-days 14`: zero flagged keys; one resolved cluster.
- Final working tree: clean.

These passing checks do not exercise M1’s two-release corruption scenario. No new mutation or corruption probes were executed.

**Rules applied during review**

Applied `AGENTS.md`, `pose-review`, `.pose/workflows/review.md`, and the installed rules for backend Go, security, release integrity, documentation style, delivery evidence, knowledge governance, and recurrence. Consulted the immutable-release-ledger, declared-version-source, archival-witness, and temporal-evidence decisions. Frontend and infrastructure rules were inapplicable.

**Collection and tool dispositions**

Canonical context resolved successfully. Bundle explanation failed because `releases.json` does not match generation `gen-8c5963c8463bb53e`; consequently its displayed prepared bundle, `rvb-0b109462147a8b37`, has no observed patch/tree or collected evidence and cannot support final approval.

Fresh strict validation collection, artifact/surface reconciliation, scanners, assessments, sealing, attestation, and completion checks remain with the parent. I did not run `assess tech-debt` or `assess integrate`: their CLI implementations unconditionally persist reports, conflicting with this read-only assignment. Historical assessment results were treated as historical evidence.

**Session identity and limits**

OpenAI Codex, task-designated `gpt-6.1-sol`, execution `/root`; same vendor as the lead. No claim of human, cryptographic, cross-vendor, or proven cognitive independence. No repository source, Git state, policy, attestation, or lifecycle changes were made; no GitHub messages or contribution drafts were created.