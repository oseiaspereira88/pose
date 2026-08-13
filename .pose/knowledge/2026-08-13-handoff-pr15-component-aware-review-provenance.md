---
type: handoff
slug: pr15-component-aware-review-provenance
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-08-13
last_reviewed_at: 2026-08-13
expires_at: 2026-09-12
source_refs:
  spec: "pose-component-aware-review-plans"
  workflow: "review"
  commands: ["pose review-check spec:pose-component-aware-review-plans", "pose artifact-check --spec pose-component-aware-review-plans --strict", "pose surface-check --spec pose-component-aware-review-plans --strict"]
  external_sources: [{url: "https://github.com/oseiaspereira88/pose/pull/15", accessed_at: "2026-08-13"}]
---

# handoff: pr15-component-aware-review-provenance

## Context

Formal review of spec `pose-component-aware-review-plans` (PR #15) ran in a
separate execution under `same-actor-separate-execution` and recorded attempt
`rvw-20260813T032130Z-c42b8c1e` with decision `changes-requested`. The
implementation quality, tests and security posture passed; the blocking gap is
delivery provenance, not code.

## Current state

- Remediation executed on 2026-08-13: every implementation commit carries the
  `POSE-Spec: pose-component-aware-review-plans` trailer, change set
  `cs-f16aa75f2706` (`range:dbee77a2..73ebafb`, 52 paths / 6 commits) is
  recorded, the integrity index was regenerated and the delivery validation
  evidence was rebound to provenance digest `sha256:06a1889c…`.
- `artifact-check --strict` (explicit range) and `surface-check --strict` now
  pass with zero error findings; `pose validate --strict --module pose-mcp`
  is green including scaffold parity.
- Re-review ran in a separate execution and recorded
  `rvw-20260813T063956Z-b11e7bcc` (supersedes `rvw-20260813T032130Z-c42b8c1e`)
  with decision `changes-requested`: the provenance findings are resolved, but
  four probe-verified spec-compliance gaps (from the codex-connector review of
  PR #15) now block approval.
- `required-tool-enforcement` (high, open): `ReviewCheck` never evaluates
  `plan.Tools`; required tools can be skipped without blocking approval
  (R16/R17, workflow step 3).
- `schema-v1-evidence-namespace` (high, open): closed rule/evidence catalogs
  apply to schema-v1 profiles too, breaking previously readable custom
  evidence classes and rule refs (R29).
- `metadata-incomplete-fail-closed` (high, open): `metadataStatus.isComplete`
  and `missingFields` are discarded; metadata-incomplete components neither
  warn nor block and defaulted values reach selector matching (R4/R9).
- `tool-lifecycle-order` (medium, open): alphabetical tool sort puts
  completion gates before evidence-collection tools although the workflow
  instructs plan-order execution (R20).
- `overlay-order-text` (low, open): within-category overlay ordering sorts by
  first matched component then ref; spec R6 text says language/domain overlays
  sort by ID. Align code or spec text.
- `unmapped-governance-warnings` (low, accepted-risk): governance/docs paths
  have no component-map roots and remain warnings per policy; documented
  residual risk of the spec.

## Next checks

- Remediate the four code findings with regression tests (required-tool
  enforcement, schema-v1 catalog gating, metadata-incomplete fail-closed,
  lifecycle tool ordering); keep prior review attempts immutable.
- Then record a superseding approval attempt in a separate execution, pass
  `pose review-check spec:pose-component-aware-review-plans`, and run
  `pose closeout-check` plus the spec closeout flow.

## Risks

- Do not merge PR #15 while the delivery gate fails; the spec itself declares
  `surface-check --strict` as a required delivery gate.
- When fixing provenance, keep prior review attempts immutable and supersede
  explicitly; do not rewrite recorded attempts.
- Recurrence-check flags `validate-native` failures (8 runs, pre-dating this
  spec) — unrelated to PR #15 but worth watching at the portfolio level.

## Next owner

`@pose-maintainers` (PR author should push the provenance remediation commit).

## References

- `spec:pose-component-aware-review-plans`
- `adr:2026-08-12-component-aware-effective-review-plans`
- Review attempt `.pose/reviews/rvw-20260813T032130Z-c42b8c1e.md`
- https://github.com/oseiaspereira88/pose/pull/15 (accessed 2026-08-13)
