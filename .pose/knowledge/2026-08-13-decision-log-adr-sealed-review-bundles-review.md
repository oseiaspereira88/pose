---
type: decision-log
slug: adr-sealed-review-bundles-review
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-08-13
last_reviewed_at: 2026-08-13
expires_at: 2026-11-11
source_refs:
  spec: "pose-review-bundle-convergence"
  workflow: "feature"
  commands: ["pose lint-spec pose-review-bundle-convergence --ready-check", "pose validate --strict --module pose-mcp --report --report-task review-bundle-convergence"]
  external_sources: []
---

# decision-log: adr-sealed-review-bundles-review

## Context

The current full-body review scope digest lets closeout bookkeeping alter the
subject that was just approved. Component-aware review also exposed transient
provider merge provenance and excessive full-plan rereview. The proposed ADR
separates a canonical sealed bundle from its attestation and makes POSE the
offline verifier while an optional Conductor workflow owns orchestration.

## Current state

ADR `2026-08-13-sealed-review-bundles-and-attestations` is accepted. The core,
CLI, read-only MCP, schemas, opt-in migration, scoped provenance bridge,
English/pt-BR distribution and fresh-repository convergence smoke are
implemented under `pose-review-bundle-convergence`.

## Next checks

- Prove `prepare -> validate -> seal -> review -> attest -> close` converges
  without a second bundle or review.
- Prove lifecycle, state, assessment, report and attestation mutations do not
  change the sealed bundle.
- Prove every governed semantic, source, policy and required-evidence mutation
  supersedes the affected bundle.
- Verify synthetic merge/squash subjects by patch and tree/content digests.
- Verify policy-bounded targeted criterion reuse and complete final projection.
- Verify a new delivery target cannot stale unrelated closed specs and that no
  historical evidence must be regenerated to recover the strict gate.
- Verify repeated mapping warnings and inactive completion tools are grouped in
  human output while canonical JSON retains complete provenance.
- Verify offline flow, installed-binary parity and schema-v1/v2 migration before
  promoting the feature from preview.

## Risks

Semantic projection may omit a meaningful input or retain an operational input,
causing false freshness or false staleness. Keep unknown paths fail-closed,
require include/exclude golden fixtures and review production deltas after the
first opt-in release. Criterion reuse may miss cross-cutting effects; disable it
for high-criticality profiles until sliced-input evidence proves safe.

## Next owner

`@pose-maintainers`.

## References

- `spec:pose-review-bundle-convergence`
- `adr:2026-08-13-sealed-review-bundles-and-attestations`
- `adr:2026-08-12-component-aware-effective-review-plans`
- `knowledge:pr15-component-aware-review-provenance`
