# Review: pose-release-lifecycle-closure

## Decision

Approved after remediation for `spec:pose-release-lifecycle-closure` by
`agent:independent-release-review-1`.

## Evidence reviewed

- Focused lifecycle, changelog, Git, provider-evidence, backfill and MCP tests.
- Full Go suite, shell syntax, workflow-security contract, strict POSE/spec/
  skills gates, catalog/scaffold parity and module validation.
- Real repository backfill: `v0.9.0` archive detected at medium confidence;
  later tags lack local manifests/evidence and remain low-confidence gaps.
- Integration/technical-debt assessments and `govulncheck`; no new debt marker
  or called vulnerability attributable to this diff.

## Findings

- Resolved, high: fragment and evidence reads could follow symlinks outside the
  project. Symlink fragments are rejected and evidence paths resolve under root.
- Resolved, high: provider evidence accepted unknown fields, credential-bearing
  URLs and unsafe asset names. Decoding is closed and identity fields are
  minimized, HTTPS-only and digest/path validated.
- Resolved, high: recorded tagged/published/verified evidence did not compare its
  commit to the actual immutable tag. All strong transitions now require an
  exact tag commit match.
- Resolved, medium: status existed only dynamically although the spec required
  an indexed projection. `pose index` now writes deterministic `releases.json`.
- Resolved, medium: historical backfill apply had no useful legal operation. It
  now versions only the reviewed inventory and never fabricates release facts.
- Open findings: none.

## Automation safety

The local script refuses dirty candidates and existing tags, stages nothing,
pushes only the new tag and exits with a recovery instruction on provider
failure. Tagged CI validates the prepared candidate, reads immutable notes and
retains minimized publication evidence without overwriting an existing asset.
