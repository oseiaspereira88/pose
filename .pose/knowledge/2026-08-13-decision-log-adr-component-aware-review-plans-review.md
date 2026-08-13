---
type: decision-log
slug: adr-component-aware-review-plans-review
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-08-13
last_reviewed_at: 2026-08-13
expires_at: 2026-11-11
source_refs:
  spec: "pose-component-aware-review-plans"
  workflow: "feature"
  commands: ["pose assess discover --component pose-mcp/internal/pose", "pose lint-spec pose-component-aware-review-plans --ready-check"]
  external_sources: []  # [{url: "", accessed_at: "YYYY-MM-DD"}]
---

# decision-log: adr-component-aware-review-plans-review

## Context

POSE 1.0 binds immutable review attempts to a scope-kind profile but does not
compose criteria or tool guidance from the mapped components. The implementation
introduces a public schema-v2 policy/profile contract, CLI/MCP projections and a
new plan digest, so the design needs an explicit future-review trigger.

## Current state

ADR `2026-08-12-component-aware-effective-review-plans` is accepted. It chooses
typed one-level overlays, deterministic union into one effective plan, a closed
native-tool catalog and additive schema-v1 compatibility. Implementation is
tracked by `pose-component-aware-review-plans`.

## Next checks

- Verify frontend, backend and multi-component golden plans.
- Verify command/path/symlink/selector negative cases.
- Verify CLI/MCP schema parity and scaffold distribution.
- Review production false-staleness before 2026-11-11.

## Risks

Component maps can be stale and overlay composition can make unrelated metadata
invalidate approvals. Keep mapping provenance visible, include only consumed
inputs in the digest and reject conflicts instead of selecting heuristically.

## Next owner

`@pose-maintainers`.

## References

- `spec:pose-component-aware-review-plans`
- `adr:2026-08-12-component-aware-effective-review-plans`
- https://github.com/oseiaspereira88/pose/issues/13 (accessed 2026-08-13)
