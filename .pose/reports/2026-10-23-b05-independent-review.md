# B05 — actual separate Codex review

## Design triage — advice only

Run: 20261023T212729-codex-4156211; session: 01a16b4e-3780-7711-a443-21f5bf7a026f. Same-vendor execution; no human or cross-vendor assurance. Verbatim response follows.

Read-only triage at baseline `40b1a50276040cb7735e4c05062e2e67e4164d27`. No writes, tests, native POSE commands, or agents. This is design advice, not final approval.

1. **Use one exact scope-kind validator.** Share the closed `spec` / `milestone` / `roadmap` taxonomy between `ParseScopeRef`, policy-map validation, and the assurance getter. Validate map keys without trimming, case folding, or typo correction. Preserve `ParseScopeRef`’s existing outer reference whitespace handling; changing that is unnecessary for B05. See [review_closeout.go:47](/tmp/pose-b05-work/pose-mcp/internal/pose/review_closeout.go:47).

2. **Validate keys across every supported policy schema, including disabled policies.** Place assurance/independence key validation outside the schema-2+ block. Today assurance validation is schema-gated, while independence validates only values. Keep existing value/default and authority-prerequisite semantics unless separately adjudicated; moving the entire assurance block would broaden the change. Produce deterministic errors naming the field, quoted key, cause, and supported kinds. Preserve permissive unknown top-level fields. See [review_closeout.go:501](/tmp/pose-b05-work/pose-mcp/internal/pose/review_closeout.go:501), [assurance validation:557](/tmp/pose-b05-work/pose-mcp/internal/pose/review_closeout.go:557), and [independence validation:586](/tmp/pose-b05-work/pose-mcp/internal/pose/review_closeout.go:586).

3. **Guard direct getter calls.** `ReviewIdentityAssurance` currently returns `verified` for any matching arbitrary key. Return no assurance for unsupported kinds; retain `declared` for supported kinds with absent configuration. Its existing string signature can remain. See [review_closeout.go:246](/tmp/pose-b05-work/pose-mcp/internal/pose/review_closeout.go:246).

4. **Keep projection entries global for this change.** The evidence supports an intentionally instance-level configuration projection: entries are computed globally, and the optional scope adds historical bundle context afterward. MCP explicitly describes that parameter as selecting the newest sealed bundle for comparison. Thus `verified-identity.Effective=true` means configured verified protection operates for the listed kinds, not necessarily the requested scope. Preserve that boundary and clarify it; per-scope capability applicability would be an additional semantic decision. Validate any supplied scope reference, however, so malformed selectors do not become apparently legitimate scope context. See [effective_governance.go:172](/tmp/pose-b05-work/pose-mcp/internal/pose/effective_governance.go:172), [scope context:228](/tmp/pose-b05-work/pose-mcp/internal/pose/effective_governance.go:228), and [MCP description:2636](/tmp/pose-b05-work/pose-mcp/internal/mcpserver/server.go:2636).

5. **Profiles are structurally scope-keyed, but exclude them from the minimal B05 implementation.** `policy.Profiles[scope.Kind]` makes equivalent validation technically coherent. However, the existing prospective ADR explicitly excludes additional profile/configuration validation. Including `profiles` needs an explicit amendment and compatibility coverage, rather than slipping into assurance/independence hardening. Do not expand into profile-body or selector validation. See [review_plan.go:236](/tmp/pose-b05-work/pose-mcp/internal/pose/review_plan.go:236) and [ADR:23](/tmp/pose-b05-work/.pose/adr/2026-10-23-shared-supported-scope-taxonomy-for-review-policy.md:23).

The material risks are:

- **High — historical baseline protection can degrade.** `parseReviewPolicyAt` uses today’s parser. An old policy containing a newly rejected key becomes unreadable; baseline resolution is discarded, and the plan records an *unprotected review contract* warning. Explicitly cover this case and diagnose it; do not silently introduce a permissive historical parser or claim baseline protection survived. See [review_policy_baseline.go:148](/tmp/pose-b05-work/pose-mcp/internal/pose/review_policy_baseline.go:148), [review_plan.go:222](/tmp/pose-b05-work/pose-mcp/internal/pose/review_plan.go:222), and [warning:354](/tmp/pose-b05-work/pose-mcp/internal/pose/review_plan.go:354).

- **High — MCP currently suppresses projection errors.** CLI governance propagates them, but `pose_project_state` omits governance when its reader fails and can return success. B05 needs explicit invalid-policy diagnosis here to meet domain/CLI/MCP parity. See [state_governance.go:32](/tmp/pose-b05-work/pose-mcp/internal/cli/state_governance.go:32) and [server.go:1396](/tmp/pose-b05-work/pose-mcp/internal/mcpserver/server.go:1396).

- **Historical readability differs from current verification.** `LoadReviewBundle` independently checks sealed records and digests; preserve that reader, payloads, stamps, and missing-assurance default of `declared`. Current verification still reads current policy and may refuse an invalid configuration. Describe this as an unavailable current evaluation, without rewriting or declaring the historical record invalid. See [review_bundle.go:1916](/tmp/pose-b05-work/pose-mcp/internal/pose/review_bundle.go:1916), [sealed defaults:258](/tmp/pose-b05-work/pose-mcp/internal/pose/review_bundle.go:258), and [verification:2802](/tmp/pose-b05-work/pose-mcp/internal/pose/review_bundle.go:2802).

The [existing prospective ADR:17](/tmp/pose-b05-work/.pose/adr/2026-10-23-shared-supported-scope-taxonomy-for-review-policy.md:17) already classifies stricter refusal as **breaking/major**, including schema 1. Retain that classification, with no automatic policy migration, adoption, or historical restamping.

## Sealed delivery review

Actual run 20261024T015741-codex-237680; session 01a16c45-9770-7f10-a10d-ef6ec9581985. Same-vendor separate execution, no human or authenticated/cross-vendor assurance. Verbatim final response follows.

Decision: **APPROVED**, conditioned on final-head PR CI, including required Security checks, being green before merge. No B05 blocker or residual finding identified. This does not claim closeout, publication, or merge checks are complete.

This is the separate Codex/OpenAI read-only review execution. The requested target was `gpt-6.1-sol`; this runtime provides no authenticated model/session receipt to independently verify that identifier. No human confirmation, authenticated reviewer independence, cognitive independence, or cross-vendor assurance is claimed. I made no writes, ran no tests or native POSE commands, spawned no agents, and recorded no attestation.

The actual [sealed bundle](/tmp/pose-b05-work/.pose/review-bundles/rvb-03f3e1b04bb0969a.json) binds:

| Identity | Exact value |
|---|---|
| Bundle | `rvb-03f3e1b04bb0969a` |
| Bundle digest | `sha256:03f3e1b04bb0969a3d83dce99f1e31f0efacb4d9e9e7656a93bf2a394a314142` |
| Plan digest | `sha256:cda61cde7d4843389099a224279feef552fb80600a33ab3328e2e7176f58883e` |
| Implementation digest | `sha256:fbd58f6a99a72ab6a548e5d670a00781fabf3a20d738d8d84a35d09551ccbfa1` |
| Patch digest | `sha256:005837993dcaf5196c2e382a90cd71bd2a181313debbd095771640afb8eb778c` |
| Tree digest | `sha256:ae485497d183b9a95ec114bb3d8140aa97a5c03a9ac04dde53ca09164e427fe4` |
| Source | `5b4161b83bb0bc4718fa55753f79556bb88846f9` |
| Git source tree | `476952cf1783a81f9ae110db892782992a4839b2` |
| Baseline | `40b1a50276040cb7735e4c05062e2e67e4164d27` |

Product code remains unchanged from requested source `bf8225f796a780d202f544723c4ac9486f4e3082`; intervening commits contain attribution, trace, and execution documentation.

I independently recomputed the bundle, plan, implementation, patch, and tree digests. All 17 subject entries match immutable Git content and current file bytes. Consumed file hashes match. The attributed range contains exactly five commits, all carrying the exact B05 trailer. Four semantic-section hashes match; R1–R6 and D1–D2 remain unchanged since managed start. The sealed basis matches the start receipt. No gitlink changed.

The current [canonical validation result](/tmp/pose-b05-work/.pose/results/delivery-validation.json) reports **148/148 passed**, without failures or skips, at the sealed source. All 144 sealed validation entries match its checks, classes, source, and provenance digest `sha256:6176381d536be76b4a7fc5eada732c74d374821d45ee615733f50fcb8b0a4141`. The additional structural evidence binds to the same implementation.

All **15 actual plan criteria** follow.

| id | passed/finding/not-applicable | individual reason |
|---|---|---|
| assumption-integrity | passed | Known defaults, global projection semantics, historical-read separation, and assessment limitations are explicit and supported by source and regressions. |
| backend-concurrency | passed | No concurrency machinery changed; sealed unit and integration evidence passes. |
| backend-contracts | passed | Stricter refusal is explicitly breaking/major; CLI/MCP behavior and preserved historical contracts are documented and tested. |
| backend-input-errors | passed | Sorted exact-key validation produces localized field, quoted-key, and unsupported-kind diagnostics; protected-baseline errors retain their typed cause. |
| backend-integration-impact | passed | Sealed `policy-scope-validation-integration` exercises domain, CLI, and MCP boundaries successfully. |
| backend-observability | passed | CLI reports configuration errors; MCP returns a tool error instead of silently omitting governance. |
| compatibility | passed | Known defaults and unknown top-level fields remain accepted. Invalid selector compatibility is intentionally tightened, with no migration or adoption. |
| correctness | passed | Shared taxonomy governs parsing, both policy maps, and direct assurance lookup; focused and full canonical checks pass. |
| design-causality | passed | Shared validation removes selector/consumer disagreement; typed baseline refusal prevents protection downgrade. Structural mapping appears below. |
| documentation | passed | English/Portuguese manuals and embedded distribution describe runtime behavior; corrected parity passes canonical validation. |
| operability | passed | Diagnostics identify the invalid selector and supported kinds without rewriting operator configuration. |
| requirements | passed | R1–R6 have relevant sealed unit/integration evidence covering rejection, defaults, consumers, future kinds, and sealed history. |
| scope | passed | Changes implement B05 assurance/independence validation and its consumption boundaries; unrelated profile validation remains excluded. |
| security | passed | Invalid maps fail as a whole across supported schemas, including disabled policies; no partial protection or baseline-warning downgrade remains. Required final-head Security CI remains a merge condition. |
| solution-proportionality | passed | One small shared validator plus targeted consumer fixes addresses the defect without policy migration, historical parsing exceptions, or speculative taxonomy expansion. |

The bundle contains **one material structural fact**.

| SD-id | Rn or Dn | causal reason |
|---|---|---|
| SD-402ecdb3 | D2 | The prospective taxonomy ADR establishes exact supported selectors, whole-policy refusal, historical-record preservation, and breaking/major classification, preventing consumers from interpreting configuration under conflicting contracts. |

Rules applied during review: mixed security bugfix/documentation change; `.pose/workflows/review.md`, `backend-go.md`, `security.md`, `documentation-style.md`, `delivery-evidence.md`, and cumulative recurrence guidance. Knowledge governance was consulted; no knowledge artifact changed. Frontend and infrastructure rules are inapplicable.

Actual tool dispositions:

| Tool | Requirement | Disposition |
|---|---|---|
| artifact-check | required | Inspected fresh output: 21 claims, one immutable attributed change set, no B05 findings. |
| surface-check | required | Inspected fresh output: no findings; composed capability connects artifacts, production entrypoint, and current integration evidence. |
| validate | required | Inspected current strict run: 148/148 passed. |
| validate — pose-mcp | required | Covered by the same canonical run’s registered module checks. |
| assess-design | recommended | Inspected sealed structural evidence and its sole material fact. |
| assess-discover — pose-mcp | recommended | Inspected start assessment reuse; no delivery-completion refresh claimed. |
| assess-integrate | recommended | Inspected: 64 contracts, 63 heuristic unobserved consumers; not proof of runtime absence. |
| assess-tech-debt | recommended | Inspected: incomplete coverage due two pre-existing invalid specs; observed marker covered, no debt-free claim. |
| suggest-review — pose-mcp | recommended | Native invocation skipped under read-only instruction; resolved plan profiles and applicable rules inspected directly. |
| review-check | required, completion-only | Deferred until the primary records this actual response in a fresh attestation. |
| closeout-check | required, completion-only | Deferred until review recording and verification. |

Strict structural checking passed with 30 warnings; spec lint passed with its optional Tasks warning. Recurrence execution and additional scanners were not run in this read-only session. The invalid-selector shortcut is covered by B05’s negative regressions; no new review gate was introduced requiring a new adversarial-corpus row.

No risk waiver or invented finding is supplied. Preserve this response verbatim; translate only these actual judgments into a fresh attestation for the exact bundle above.
