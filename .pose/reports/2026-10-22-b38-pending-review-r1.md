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

# Final sealed adjudication — 2026-10-22

Actual separate session 01a1637e-df2e-7220-b8c0-8433c2e47625; log /home/go/.local/state/secondary-agent/20261022T090341-codex-2013571. Original R1 changes-requested decision remains unchanged above.

**APPROVED** for the exact sealed B38 subject: [rvb-ef842bee6b11a74f](/tmp/pose-pending-b38/.pose/review-bundles/rvb-ef842bee6b11a74f.json), at source HEAD `e58ca80ef26e3ed0220e9b588cece5576d8c511f`. No unresolved medium-or-higher finding identified.

I independently recomputed these identities:

| Identity | Verified value |
|---|---|
| Bundle | `sha256:ef842bee6b11a74f6f1c8111f35ee5de8d2c6d78b5c9ead2c9429b4f6353ce73` |
| Effective sealed plan contract | `sha256:38a5bda841b0e33394c29a6b0910dcee72839b2d4260796394b8f800b86f03d8` |
| Patch | `sha256:f2f75cbb30b8807c47995aa014905d07a61398fc4820e9ce2380dd8de01bfbcb` |
| Tree | `sha256:46d8a7eeb7674b22065926886dbec7b0cf0af8c748188c8b8bf5cb7a88c7f738` |
| Implementation | `sha256:6575326b412c17f9d7610e9d8c22d0eea91d3130c8a366703c89856d7a8c8215` |

All 32 subject files match their sealed current-byte digests; all 52 referenced attribution blobs resolve and reproduce their Git object identities. The range contains unrelated work: 36 attributed commits versus 98 range commits. My scope judgment follows the attributed changes and sealed files, not the entire range.

**M1 — medium, fixed.** The actual CLI before-fix output demonstrates the causal failure: corrupting only the newer tag changed refusal into approval against older verified `v1.0.0`. The initial serialization failure is a failed setup attempt, not that reproduction.

The production correction addresses the cause:

- Successful Git commands with stderr now return uncertainty instead of apparently complete output.
- `ListReleaseTags` supplies status, plan/prepare inputs, structural checking, adoption witnessing, and backfill.
- Explicit tag resolution checks diagnostic stderr before interpreting exit code 1 as absence.
- Status propagates inventory failure, so `open-next` cannot promote the older release.

The final regression has valid adopted-policy fields and exercises the older verified control, newer incomplete tag, corrupted newer ref, status, structural checker, and backfill. The domain regression distinguishes genuine absence from corruption. Historical R11-L14’s “state only goes down” rationale is superseded for this consumer variant; R13’s historical approval remains historical.

| criterionID | passed/notapplicable/finding | reason |
|---|---|---|
| assumption-integrity | passed | Separates proven absence, unreadability, unsupported evidence, and accepted residual risks. Identity and scanner limitations remain explicit. |
| backend-concurrency | passed | No new concurrent state or synchronization mechanism; collected race checks pass. This does not establish atomic reads across concurrent external Git mutations. |
| backend-contracts | passed | Existing CLI/MCP entrypoints remain; snapshot diagnostics are additive, while stricter rejection is explicitly classified. |
| backend-input-errors | passed | Strict manifest decoding, confined paths, regular tagged-file checks, and error propagation prevent malformed or uncertain inputs becoming proof. |
| backend-integration-impact | passed | Domain verification reaches CLI, MCP status, obligations, public claims, and archival consumers; relevant collected integration checks pass. |
| backend-observability | passed | Field-local issues, snapshot gaps/limits, and contextual Git errors expose failures. Remaining candidate/human-output omissions are owned lows. |
| compatibility | passed | Historical cuts use their own tagged policies and content. Stricter refusal is classified minor; later legitimate evidence and cycles remain supported. |
| correctness | passed | Source tracing and negative/control regressions address projection tampering, wrong tagged trees, lost inputs, and M1’s cross-release selection failure. |
| design-causality | passed | Recomposition and tagged-tree reads replace self-consistent claims with authoritative inputs; shared inventory failure prevents unsafe release selection. Structural mapping follows below. |
| documentation | passed | English, Portuguese, and scaffold manuals describe runtime behavior. Stale authoring prose remains an explicitly owned editorial follow-up. |
| operability | passed | Failures identify the affected source or inventory without repairing tags or manufacturing evidence. Benign Git warnings can conservatively refuse operation. |
| requirements | passed | R1–R8 have relevant source and current collected verification, assessed individually below. |
| scope | passed | Preserves B38 manifest/tag integrity and prior completed scope; does not implement B39 replay, B40 selection, or B41 recovery. |
| security | passed | No shell interpolation or new dependencies; paths and tagged file modes are checked, uncertainty fails closed, and no new secret material was observed. Scanner limits remain disclosed. |
| solution-proportionality | passed | Shared existing Git/domain readers are the smallest sufficient correction. No new storage, provider queries, mutation mechanism, or speculative abstraction was added. |

| Requirement | Judgment | Reason |
|---|---|---|
| R1 | passed | Exact names, required fields, duplicate/null handling, supported schema, types, versions, and paths are validated; `kind:null` remains the bounded low below. |
| R2 | passed | Manifest projections/input digest are recomputed; tagged fragment bytes, identities, categories, flags, references, and rendered notes are compared. |
| R3 | passed | Notes, policies, version evidence, archives, and missing/unreadable distinctions are checked. M1 closes incomplete tag enumeration. |
| R4 | passed | Only `refs/tags` resolves; annotated tags peel to commits; snapshot inputs come from tagged-tree blobs with regular-file checks. |
| R5 | passed | Content binds worktree copies to the tagged cut without requiring tagged commit equality with current HEAD. |
| R6 | passed | Historical policy and narrowly disclosed recomputation limits preserve legitimate cuts after later changes. |
| R7 | passed | Shared domain checks feed consumers; `Sound()` guards positive claims. Verification does not repair release artifacts or tags. |
| R8 | passed | Current focused and full-module evidence covers tampering, absent snapshots, lightweight/annotated tags, later cycles, and the new corruption controls. |

The plan marks **SD-859c40ea** material. I also dispositioned the other observed structural deltas:

| materialfactID | R or D basis | causal reason |
|---|---|---|
| SD-859c40ea | R8, D1 | Validation-matrix change makes authoritative-input and tag-integrity qualification a required integration check. |
| SD-784aeee4 | R3, R7, R8 | Added integration registration includes the broken-newer-tag and corrupt-versus-absent regressions, preserving the failure boundary. |
| SD-4145a07d | R3, R5 | Historical fragment rename/content reconciliation restores the worktree’s correspondence with the existing tagged cut; it does not rewrite that tag. |

A CLI-only workaround would leave other inventory consumers inconsistent. Matching specific English warning text would be narrower but brittle. The chosen conservative stderr handling is proportionate; revisit it if supported Git configurations emit routine harmless warnings, large inventories produce unacceptable latency, or concurrent external ref changes require transactional snapshot semantics.

The cheapest superficial gate satisfaction would be recomputing manifest digests around false content, supplying a correct worktree beside the wrong tag, or dropping a newer tag from inventory. Blob recomposition, tagged-tree tests, and M1’s two-release regression address those shortcuts. **The adversarial README still lacks a dedicated B38 row**; executable regressions provide coverage, but corpus registration is not claimed.

All 20 existing follow-up entries retain their recorded owner `@core` and review date `2026-12-01`. My dispositions are:

| Existing follow-up subject | Disposition and reason |
|---|---|
| `version_evidence.kind:null` | Accept low: equivalent to empty engine kind; source/value binding remains. |
| Auxiliary-file filters | Accept low: inconsistent refusal, without proof of divergent-content acceptance. |
| Candidate/status diagnostics | Accept low: presentation omissions do not remove integrity gaps. |
| Integration test pattern | Accept low: M1 is now registered; older omitted families remain covered by the full module suite. |
| Strict changelog loader in plan/prepare | Accept low: can produce an unusable local candidate; strict consumers refuse it. |
| Stale authoring prose | Accept low: editorial contradiction remains disclosed. |
| Byte-identical worktree links | Accept low: tagged files require regular modes; worktree type policy remains inconsistent. |
| Historical smoke error/zero-fixture handling | Accept low: these smokes cannot establish complete historical coverage; separate production gates remain. |
| Cut versus adoption witness | Accept low: conservative over-enforcement remains possible. |
| Non-directory planning inventory entries | Accept low: planning can omit them; status/check reject them. |
| Strict pending-fragment policy loading | Accept low: bounded existing loader inconsistency remains. |
| Non-SemVer tag parity | Accept low: conservative checker/status disagreement. |
| Tagged adoption-policy validation | Accept low: advisory adoption inference is weaker than full snapshot validation. |
| R11-L14 corrupt refs | Unsafe variant superseded by M1 and fixed; persisted open entry is not evidence of an unresolved medium. Packed-ref behavior lacks a dedicated new regression. |
| Lost `GIT_DIR` | Accept low: environment-specific repository detection limitation remains; no positive approval path established. |
| Tree/blob tags | Accept low: excessive structural refusal. |
| Re-deriving `previous_version` | Accept low: syntax/precedence checked; historical relationship is not independently re-derived. |
| Silent `ForeignFragmentClaims` errors | Accept existing low: incomplete auxiliary claim diagnostics remain; no expanded claim of coverage. |
| Unadopted-project tag-scan cost | Accept low: performance concern remains. |
| R13-L20 adoption-witness parity | Accept low: can omit advisory status/obligations; checker refuses and `open-next` cannot establish verified approval. |

**Evidence and tool dispositions.** The completed [canonical validation](/tmp/pose-pending-b38/.pose/results/delivery-validation.json) records **143/143 passed, zero failed/errors/skips**, at the requested HEAD and scoped provenance `sha256:d85140bb32b5eeee475221c65b880e66cc7430baf8f44c1c531e7f09bb881080`. Its final log agrees. All 139 collected sealed evidence entries match canonical results; the 140th entry is structural evidence. The earlier 141/143 run remains failed and was not waived.

I inspected complete collector outputs: surface has one target and zero findings; artifact reconciliation has 49 claims, 104 observed paths, and 660 repository orphan warnings, all warning severity; strict structural checking succeeds with 30 warnings; history and skills succeed; recurrence reports zero flagged keys. Integration assessment reports 63 heuristic unobserved-consumer gaps, not a clean integration certification. Tech-debt coverage remains **INCOMPLETE** for the two named legacy specs, with zero uncovered markers among scanned sources. Component discovery is the earlier baseline, not a fresh final-HEAD assessment.

My actual executions were read-only source/Git inspection, hash recomputation, context resolution, bundle explanation, review-plan inspection, `git diff --check`, and **auto-attest without `--apply`**. Auto-attest reports seven mechanical criteria and eight pending judgments, answered above. I ran no tests or vulnerability scanners. Gitleaks/govulncheck were unavailable on PATH; prior security reports are historical evidence, not current scanner executions. I did not rerun assessments that persist files.

`review-check`, `closeout-check`, attestation recording, and any closure discovery refresh remain deferred to the parent after actual attribution. This approval does not claim those completion gates passed.

Rules applied: bugfix review under `AGENTS.md`, `pose-review`, review workflow, backend Go, security, release integrity, documentation style, delivery evidence, and knowledge governance. Frontend/infrastructure rules were inapplicable.

Actual role: OpenAI Codex execution `/root`, task-designated `gpt-6.1-sol`, separate same-vendor fallback. Runtime model/session identity and separation are declared, not independently authenticated. No human, cryptographic, cross-vendor, or proven cognitive-independence claim. No files, Git state, policies, attestations, lifecycle records, tags, releases, or external messages were written.