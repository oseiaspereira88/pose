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

Pending actual final execution and exact bundle binding. Design advice is not approval.
