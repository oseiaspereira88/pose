# ADR: Component-aware effective review plans

## Status
Accepted (2026-08-13) — spec `pose-component-aware-review-plans`

## Context

POSE 1.0 selects one review profile from the terminal scope kind (`spec`,
`milestone` or `roadmap`). That makes approval immutable and digest-bound, but
the selected criteria do not reflect the components that changed. A React UI,
a Go service and a multi-component REST change can therefore receive the same
generic plan even though the repository map, module metadata, artifact ledger,
delivery targets, rules and validation matrix already describe their different
risk surfaces.

The extension must remain deterministic and offline. Repository-owned profile
data cannot become a shell-command surface, an overlay cannot weaken a stricter
independence requirement, and migration cannot invalidate schema-v1 approvals
or legacy scopes retroactively.

Options considered:

1. Select the single most-specific profile — rejected because a
   multi-component scope loses cross-cutting or sibling-component criteria.
2. Require one independent review artifact per component — rejected because it
   fragments the scope decision and makes boundary coverage implicit.
3. Let profiles embed arbitrary commands — rejected because repository content
   would become an execution and injection surface.
4. Compose one effective plan from typed overlays and cataloged native tools —
   selected because it preserves scope-level approval while making component
   coverage, provenance and tool guidance explicit.

## Decision

Introduce schema-v2 review policy/profile support as an explicit opt-in. Keep
the existing scope profile as the base, then compose applicable overlays in
this byte-stable order:

1. base scope profile;
2. language overlays ordered by profile ref;
3. domain overlays ordered by profile ref;
4. component overlays ordered by component path then profile ref;
5. delivery-kind and criticality overlays ordered by profile ref;
6. one synthetic integration-boundary criterion for multi-component scopes.

Resolve components from explicit spec component refs, structured artifact
claims, structured delivery targets and the indexed delivery-integrity reverse
map. Join paths to the longest registered root in `repo-map.json`; retain the
mapping source and expose unmapped or ambiguous refs according to policy.
Consume only metadata used by selector matching or output so unrelated index
changes do not invalidate a plan.

Profiles use typed selectors (`languages`, `domains`, `component_ids`,
`delivery_kinds`, `criticalities`) and typed native-tool references. A policy
lists the allowed overlay profile refs. No expression language, recursive
filesystem discovery or arbitrary executable field is accepted.

Compose criteria by stable ID. Identical contracts deduplicate and retain all
profile provenance. Conflicting descriptions, rules, evidence classes or
requiredness produce a plan blocker; an overlay cannot remove a base criterion.
Choose the most restrictive reviewer-independence value across policy and
selected overlays (`mandatory-human` > `different-actor` >
`same-actor-separate-execution`).

Expose a canonical read-only `ReviewPlan` through `pose review-plan` and
`pose_review_plan`. Render recommendations from a closed native catalog with
typed argument arrays. Plan resolution never executes a tool or mutates the
project.

Add `plan_digest` to new immutable attempts and include it in review freshness.
`pose review record` shows the effective digest in dry-run output and accepts
an optional expected digest to prevent time-of-check/time-of-record drift.
Schema-v1 attempts without `plan_digest` remain valid under schema-v1 policy;
after explicit schema-v2 adoption, new/open scopes require current plan-bound
attempts while exempt legacy done scopes retain their adoption behavior.

## Consequences

- Positive: frontend, backend and repository-specific components receive
  distinct, explainable review coverage.
- Positive: multi-component scopes prove both per-component and boundary
  review without splitting the approval artifact.
- Positive: reviewers receive safe, actionable POSE tool guidance with expected
  evidence and criterion provenance.
- Positive: policy, profile or consumed component-context changes make approval
  deterministically stale.
- Trade-off: review profile schemas and canonical plan JSON become public
  compatibility contracts requiring golden tests.
- Trade-off: stale or coarse component maps remain visible as warnings/blockers
  and cannot be repaired by heuristic invention.
- Trade-off: adding a recommended native tool requires catalog and contract
  evolution even when the underlying CLI command already exists.
- Neutral: tool execution remains explicit and separately authorized; a plan is
  guidance and closeout policy, not an executor.

## Review triggers

Revisit this decision if POSE adopts signed third-party review-tool providers,
if one-level overlays cannot express real repository policy without duplication,
or if component mapping produces material false-staleness in production. Do not
add an expression language or arbitrary command execution without a new ADR and
security-negative test plan.
