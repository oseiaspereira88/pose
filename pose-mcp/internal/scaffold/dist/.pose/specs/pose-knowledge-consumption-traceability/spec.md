---
slug: pose-knowledge-consumption-traceability
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-requirement-evidence-traceability
priority: 14
---

# Spec: Knowledge consumption and semantic assist

## 1. Intent

### Goal
show when governed knowledge influences work and offer optional human-reviewed semantic retrieval.
### Business value
Makes expiring memory useful without turning POSE into an opaque vector store.
### Constraints
- Keep deterministic metadata authoritative and semantic ranking advisory.
### Non-goals
- Automatically apply restricted knowledge or make model output a gate.

## 2. Requirements

### Functional
- R1: Specs and reports shall cite consumed knowledge by stable reference.
- R2: Usage signals shall inform review without extending TTL automatically.
- R3: Semantic suggestions shall expose rationale, filter sensitivity and require confirmation.

### Non-functional
- Operate fully without a semantic backend.

### Security
- Enforce sensitivity before retrieval and require approved providers.

### Compatibility
- Existing knowledge remains valid with additive usage metadata.

## 3. Technical Plan

### Affected areas
- Knowledge schema/checks, spec links, MCP and optional adapter.

### API/contract changes
- Define deterministic consumption references and advisory responses.

### Data/storage changes
- Add minimized usage events with artifact ID, purpose and timestamp.

### Technical risks
- Popularity can preserve stale knowledge unless owners still review it.

### Primary references
- [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework)
- [MCP security best practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework): knowledge had TTL/owners/sensitivity but zero consumption visibility; the live instance's only artifact (knowledge:contract-baseline-handoff) had no citation signal despite being load-bearing across milestones.

### Implementation
- [x] Define consumption events and sensitivity-safe retrieval policy: `knowledge:<slug>` citation tokens in spec bodies as the stable reference contract; usage derived deterministically from the repository (no manual event stream); `restricted` artifacts filtered before any retrieval; semantic adapters require an approving ADR, none approved by default (ADR `2026-07-19-knowledge-consumption-refs-and-advisory-retrieval`). ([reference](https://www.nist.gov/itl/ai-risk-management-framework))
- [x] Add reference validation and usage projections: `pose knowledge-check` fails dangling refs (`knowledge.ref_failures`); `pose knowledge-usage` projects per-artifact citations (citing specs, owner, expiry) and states the TTL invariant — usage never extends `expires_at`. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))
- [x] Prototype explainable semantic ranking with confirmation tests: `pose knowledge-suggest` ranks non-restricted artifacts with the deterministic lexical engine, exposes shared-term rationale and score, shows the restricted-filtered count, and requires human confirmation before citing. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))

### Validation
- [x] Run `go test ./pose-mcp/... -run 'Knowledge|Sensitivity|Usage'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://www.nist.gov/itl/ai-risk-management-framework))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-knowledge-consumption-refs-and-advisory-retrieval.md` (Accepted): derived citations over a manual usage-event stream; deterministic lexical baseline over embedding backends as the shipped default; sensitivity filtering precedes retrieval; TTL immutability under usage signals; adapters gated behind per-provider ADRs.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Knowledge|Sensitivity|Usage'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-knowledge-consumption-traceability --ready-check`.

### Requirement trace
- R1 [satisfied] stable knowledge:<slug> refs validated against governed artifacts; check:test (TestKnowledgeRefValidation) report:2026-07-19-standard-validate-native.md
- R2 [satisfied] usage projection informs review without touching expires_at; check:test (TestKnowledgeUsageProjection)
- R3 [satisfied] advisory suggestions expose rationale, filter restricted before retrieval and require confirmation; check:test (TestKnowledgeSuggestFiltersRestrictedAndExplains)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'Knowledge' -count=1` — SUCCESS.
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite).
- `pose knowledge-usage` on the live instance — projects the real citation of knowledge:contract-baseline-handoff by this milestone's specs.
- `pose check --strict` and `pose knowledge-check` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Stable `knowledge:<slug>` citation contract with dangling-ref gate; derived
usage projection (`knowledge-usage`) honoring TTL immutability; deterministic
explainable advisory retrieval (`knowledge-suggest`) with pre-retrieval
sensitivity filtering and mandatory confirmation; operating-manual
documentation; ADR. Fully offline — no semantic backend shipped or required.

### Residual risks

- Popularity can preserve stale knowledge only if owners rubber-stamp
  reviews — the projection informs but never extends TTL by design.
- Uncited-but-used knowledge stays invisible; citing is workflow discipline.

### Follow-ups

- [open] Cite knowledge refs from the feature/bugfix workflows and skills so citation becomes routine, then review usage in the first quarterly audit. (owner:@pose-maintainers crit:low review:2026-10-08)
