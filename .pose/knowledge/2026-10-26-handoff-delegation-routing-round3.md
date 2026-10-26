---
type: handoff
slug: delegation-routing-round3
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-10-26
last_reviewed_at: 2026-10-26
expires_at: 2026-11-25
source_refs:
  spec: pose-delegation-routing
  workflow: .pose/workflows/review.md
  commands:
    - pose review verify spec:pose-delegation-routing
  external_sources: []
---

# Handoff: routing review residual follow-ups

## Context

Independent round 3 of commit 040cae6f41090c07e72dd27626cbaace4ddb5f7b reconciled B09, F06 and F07 as fixed. The reviewer is the maintainer-authorized declared identity agent:independent-gpt-6.1-sol-review (Codex/OpenAI); implementation was Claude/Anthropic. The review records two low-severity residual issues. Reproductions and original assertions are retained locally in .review-work/round3/new-probes.log and its overlays; that directory is deliberately untracked. Reproduction recipes below do not depend on those files.

## Current state

F08: internal/cli/delegate.go computes independent_of_lead using vendor inequality alone. Configure a schema-2 agent with vendor unknown, model unknown, cli sh and read command [sh]. Both `pose delegate agents --lead anthropic/opus --json` and its text form disclose independent_of_lead=true. The correct conservative result is false or explicitly unknown. The auto router, achieved independence and run-backed authority do not consume this boolean. Acceptance is limited to this disclosure defect, not permission to credit unknown identities with independence.

F09: internal/pose/delegation_policy.go uses omitempty for routing.on_unavailable. Parse a valid policy whose only agent matches anthropic/opus and whose routing.on_unavailable is []; auto refuses it. Pass that policy through CanonicalDelegationPolicy and ParseDelegationPolicy again; the list disappears and the default same-model fallback selects the agent. The existing production canonical writer only converts schema 1, which has no such routing list, or prints examples; it does not rewrite existing schema-2 policy. Acceptance is limited to that currently unconsumed serialization path. Do not add a schema-2 policy writer without addressing it; declared agent order must also survive any future serializer used for existing policy.

## Next checks

Add a CLI regression for unknown agent vendors in agents text and JSON, preserving the documented vendor-only meaning of independent_of_lead. Add a policy round-trip regression distinguishing omitted and explicitly empty on_unavailable. Run the existing routing, migration, continuity and provenance suites. Review again by 2026-11-25, or before any production schema-2 policy rewrite consumer is introduced.

## Risks

F08 may mislead a user choosing an agent, but does not establish a run proof or approval. F09 would enable fallbacks a caller explicitly disabled if a future writer used the helper on existing schema-2 policy. Neither requires changing the execution, ledger, containment or provenance contracts. R10 remains a shared detector delivery; actual task invocation belongs to pose-delegation-task and is not represented as executed here.

## Next owner

@oseiaspereira88 owns F08 and F09, both low, review 2026-11-25. Keep them as open follow-ups in the spec and link their dispositions to the native review judgments. No upstream contribution draft or issue was created.

## References

Spec: `.pose/specs/2026-10-26-pose-delegation-routing.md`. ADR: `.pose/adr/2026-10-26-delegation-is-a-first-class-execution-surface.md`. Review report: `.review-work/round3/REPORT.md` (local evidence only).
