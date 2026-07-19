---
slug: product-integrity
status: active
created_at: 2026-07-18
depends_on:
---

# Roadmap: Product integrity

**Portfolio order:** 1 of 7
**Outcome:** make every public POSE contract accurate, internally consistent and exercised by POSE itself.

This roadmap is first because version, MCP, installation and dogfooding drift undermine every later distribution or ecosystem investment. It closes the P0 truth gap identified in the capability assessment.

## Milestone: contract-baseline
- after:
- target_start: 2026-07-20
- target_due: 2026-08-01
- specs: pose-version-contract, pose-standalone-dogfood

**Exit gate:** one authoritative version source exists and the standalone repository produces its own governed planning and evidence artifacts.

## Milestone: public-accuracy
- after: contract-baseline
- target_start: 2026-08-03
- target_due: 2026-08-14
- specs: pose-mcp-catalog-conformance, pose-public-install-contract

**Exit gate:** released docs, MCP discovery and install instructions match the tested binary without placeholders or undocumented capabilities.

## Milestone: release-compatibility
- after: public-accuracy
- target_start: 2026-08-17
- target_due: 2026-08-28
- specs: pose-release-compatibility-matrix

**Exit gate:** each release proves its binary, schema, scaffold, MCP metadata and public documentation are mutually compatible.

## Risk controls

- Block release when generated and source contracts diverge.
- Treat undocumented public behavior as a defect, not as roadmap progress.
- Preserve backwards compatibility until an ADR and migration path exist.
