---
title: "Make POSE review component-aware and tool-guided"
kind: suggestion
engine_version: 1.0.0
reported_at: 2026-08-12T21:25:22-03:00
---

# POSE Engine Report: Make POSE review component-aware and tool-guided

## Description
POSE review is currently selected only by terminal scope kind: spec, milestone, or roadmap. The effective profile does not vary by mapped component, domain, changed artifact, delivery target, or risk. A frontend change and a backend change therefore receive the same generic criteria, and the review flow does not tell reviewers which native POSE tools can produce relevant evidence.

Proposed direction:

1. Make review a first-class planning mechanism with a deterministic effective review plan for each scope.
2. Compose the base scope profile with selectors derived from spec components, repository component map, governed artifacts, delivery targets, applicable rules, validation matrix, criticality, and cross-component contracts.
3. Support domain and component overlays such as frontend, backend, security, infrastructure, and repository-defined component profiles, with explicit precedence, conflict detection, criterion provenance, and stable deduplication.
4. Recommend applicable POSE tools and expected evidence without executing them implicitly. Examples include assess discover, assess integrate, assess tech-debt, artifact-check, surface-check, roadmap-check, validate, history-check, and domain-specific checks.
5. Expose the effective plan and rationale through CLI and MCP before review record, and bind the resolved plan digest to immutable review attempts so policy or component changes invalidate stale approval.
6. For multi-component scopes, require coverage for every affected component plus explicit integration criteria at component boundaries. Unmapped or ambiguous components must remain visible and fail closed when policy marks coverage required.
7. Preserve offline deterministic operation, project-root confinement, reviewer independence, immutable attempts, and staged compatibility for existing generic profiles.

Acceptance should include distinct frontend/backend plans, repository-defined component overlays, multi-component composition, rule and tool recommendations with provenance, deterministic output, CLI/MCP parity, stale-review invalidation, negative tests for selector ambiguity and path escape, scaffold documentation, locale parity, and migration behavior for schema-v1 policies.

This report requests specification and design only in the current change; implementation belongs to a later dedicated feature scope.

---
### System Context (Auto-generated)
- **POSE Engine Version:** 1.0.0
- **OS/Arch:** linux/amd64
- **Go Version:** go1.26.5
- **Reported At:** 2026-08-12T21:25:22-03:00
- **Kind:** suggestion
