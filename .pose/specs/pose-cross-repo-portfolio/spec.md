---
slug: pose-cross-repo-portfolio
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-mcp-project-scope-contract, pose-requirement-evidence-traceability
priority: 33
---

# Spec: Cross-repository portfolio projections

## 1. Intent

### Goal
project dependencies, readiness, ownership and critical paths across repositories without moving authority.
### Business value
Extends strong local governance to organization planning visibility.
### Constraints
- Repositories remain authoritative; central state is a reconciled projection.
### Non-goals
- Turn roadmaps into capacity scheduling or replace project tools.

## 2. Requirements

### Functional
- R1: Cross-repo references shall use stable organization/project/artifact identities and policy.
- R2: Projection shall explain blocked paths, stale sources and conflicts.
- R3: Views shall expose ownership and criticality without fabricating capacity.

### Non-functional
- Support incremental updates, eventual consistency and source revisions.

### Security
- Enforce tenant/project authorization and filter restricted metadata.

### Compatibility
- Local typed references remain valid; cross-repo syntax is additive.

## 3. Technical Plan

### Affected areas
- Reference schema, MCP projects, indexes, ingestion and Portal.

### API/contract changes
- Add global artifact references and reconciliation states.

### Data/storage changes
- Store revisioned projections with timestamps and tombstones.

### Technical risks
- Stale projections can misprioritize work unless freshness is prominent.

### Primary references
- [Backstage software catalog](https://backstage.io/docs/features/software-catalog/)
- [CloudEvents specification](https://cloudevents.io/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Backstage software catalog](https://backstage.io/docs/features/software-catalog/).

### Implementation
- [ ] Create an ADR for global identity, authority and consistency. ([reference](https://backstage.io/docs/features/software-catalog/))
- [ ] Implement versioned events and cross-repo resolution. ([reference](https://cloudevents.io/))
- [ ] Test stale, unauthorized, renamed, deleted and cyclic cases. ([reference](https://backstage.io/docs/features/software-catalog/))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Roadmap|Project|CrossRepo|Readiness'` and retain evidence. ([reference](https://backstage.io/docs/features/software-catalog/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://cloudevents.io/))

## 5. Decisions

- Create an ADR before changing this contract; compare [Backstage software catalog](https://backstage.io/docs/features/software-catalog/).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Roadmap|Project|CrossRepo|Readiness'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-cross-repo-portfolio --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Stale projections can misprioritize work unless freshness is prominent.
- Follow-ups: none until implementation starts.

