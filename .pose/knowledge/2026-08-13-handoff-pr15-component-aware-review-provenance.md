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

- Review attempt is recorded, digest-bound (`review.fresh=true`) and blocks
  closeout with three open findings.
- `provenance-gap` (high, open): the implementation commits carry no
  `POSE-Spec:` trailers and no attributed Git change-set range, so
  `delivery-integrity.json` holds 52 claims but 0 change sets for the spec.
  `artifact-check --strict` and `surface-check --strict` fail with 52
  `action-mismatch` findings already visible in the committed index.
- `reconciliation-claim-drift` (medium, open): the PR description claims
  "43 declared / 43 observed" while the committed index shows 52 mismatches;
  reconciliation must be re-run after the evidence commits.
- `overlay-order-text` (low, open): within-category overlay ordering sorts by
  first matched component then ref; spec R6 text says language/domain overlays
  sort by ID. Align code or spec text.
- `unmapped-governance-warnings` (low, accepted-risk): governance/docs paths
  have no component-map roots and remain warnings per policy; documented
  residual risk of the spec.

## Next checks

- Record attributed change-set evidence for the implementation range, e.g.
  `pose report --change-from <origin/main-base> --change-to <impl-commit>`
  with spec attribution, or add `POSE-Spec:` trailers to the PR commits.
- Regenerate `.pose/indexes/delivery-integrity.json` and re-run:
  `pose artifact-check --spec pose-component-aware-review-plans --strict` and
  `pose surface-check --spec pose-component-aware-review-plans --strict`.
- Re-run `pose validate --strict --module pose-mcp --report`, then record a
  superseding review attempt with decision `approved` (or
  `approved-with-reservations`) and pass `pose review-check`.

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
