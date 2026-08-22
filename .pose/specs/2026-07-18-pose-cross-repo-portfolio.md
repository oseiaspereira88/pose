---
slug: pose-cross-repo-portfolio
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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

- ADR `.pose/adr/2026-07-19-cross-repo-portfolio-reuses-mcp-project-authorization.md` (Accepted): reuse `pose.ScanProjectsDir`/`pose.ParseRootsJSON` — the MCP server's own project authorization — verbatim rather than a second discovery mechanism; `xref:<project_id>/<slug>` additive to the existing `depends_on` grammar; four distinct, explicit resolution states (resolved/blocking/reason) rather than a collapsed boolean; ownership/criticality sourced from each project's own `module-metadata.json`, no capacity/velocity/ETA field anywhere; revisioned projection with tombstones for disappeared artifacts. Rejected: a second feature-specific project registry (duplicated authorization boundary, drift risk); an unrestricted directory scan (violates the Security requirement outright).

## 6. Validation

**Strategy:** validate xref resolution against authorized/unauthorized/unknown targets, blocked-vs-not-done semantics, stale-source detection, ownership/criticality exposure without fabricated capacity, zero filesystem-path leakage in output, tombstone reconciliation across runs, and backward-compatible `depends_on` parsing.

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/cli/... -run 'PortfolioProjection|XrefDependsOn' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-cross-repo-portfolio --ready-check`.

### Requirement trace
- R1 [satisfied] `xref:<project_id>/<spec-slug>` is a stable, typed reference validated by `depXrefRE`; only projects in the authorized allowlist are ever resolved; check:test (TestPortfolioProjectionResolvesAuthorizedXref, TestPortfolioProjectionRejectsUnauthorizedProject, TestXrefDependsOnPassesReadyCheck)
- R2 [satisfied] blocked (target not done), stale (source mtime beyond threshold) and unknown/unauthorized cross-references are each an explicit, distinct reason, never silently merged; check:test (TestPortfolioProjectionExplainsBlockedXref, TestPortfolioProjectionMarksStaleSource, TestPortfolioProjectionExplainsUnknownSpec)
- R3 [satisfied] ownership/criticality exposed per project from `module-metadata.json`; no capacity-shaped field exists in the output; check:test (TestPortfolioProjectionOwnershipCriticalityNoFabricatedCapacity)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli/... -run 'PortfolioProjection|XrefDependsOn' -v -count=1` — SUCCESS (10 tests).
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold`.
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-cross-repo-portfolio --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Constraint (repositories remain authoritative; central state is a reconciled projection): the projection is written only to the invoking repository's own `.pose/reports/`; nothing is ever written back into another project's directory — `discoverAuthorizedProjects`/`buildPortfolioProjection` only read other projects.
- Security (tenant/project authorization; filter restricted metadata): `TestPortfolioProjectionRejectsUnauthorizedProject` proves an on-disk-but-unregistered project is never resolved; `TestPortfolioProjectionNeverLeaksFilesystemPaths` proves no absolute path of any project ever reaches the output.
- Compatibility (local typed references remain valid; cross-repo syntax additive): `TestXrefDependsOnPassesReadyCheck` proves a spec with an `xref:` reference still passes the pre-existing DoR readiness gate unmodified.
- Data/storage (revisioned projections with timestamps and tombstones): `TestPortfolioProjectionTombstonesRemovedSpecs` proves a disappeared spec is carried forward as an explicit, timestamped tombstone rather than silently vanishing between runs.

## 7. Final Report

- Delivered scope: `pose portfolio-projection` — cross-repository dependency/readiness/ownership/criticality reconciliation reusing the MCP server's exact project-authorization boundary, a new additive `xref:` reference grammar, explicit blocked/stale/unauthorized/unknown resolution states, and a revisioned, tombstoned projection persisted to `.pose/reports/portfolio-projection.json`.
- Residual risk: stale projections can still misprioritize work if a reader ignores the `stale`/`stale-source` signal — mitigated by making staleness a first-class, always-present field on every projected spec and every xref resolution rather than an opt-in flag a caller could forget to check.
- Follow-ups: none — the requirement families are satisfied with executed evidence and no sandbox-unavailable gap (the whole feature is local filesystem reconciliation; no network or external infrastructure is needed to test it end to end).
