---
slug: pose-attention-projects-every-source
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-fresh-install-doctor-is-clean
priority: 1
components: pose-mcp
task_type: feature
delivers: capability:attention-projects-every-source
---

# Spec: Attention projects every source

## 1. Intent

### Goal

Project the four obligation sources Attention listed as "not projected yet" — docs review pendencies, capability stale triggers, release queues and findings outside review attestations — so a project that uses them reads complete coverage and sees what they owe.

### Business value

Origin: the open follow-up of `pose-fresh-install-doctor-is-clean` (crit medium), prioritized by the maintainer on 2026-10-09 for the release. Every project with a docs manifest, a release history or a capability assessment read incomplete Attention coverage, so an empty group never meant nothing was owed. The maintainer chose to extend the obligation contract rather than anchor on specs or defer.

### Constraints

Additive to obligation schema v1; existing ids unchanged. Every effect is advisory. No producer invents state: free-form investigations are not a source.

### Non-goals

Blocking any phase on these sources; changing how each source is resolved.

## 2. Requirements

### Functional

- R1: The obligation contract shall accept a project-level origin `xref:<project>/project:<project>` with node kinds `doc` and `capability`.
- R2: An open docs review mark shall be an obligation, addressed to the doc's owner when the manifest names one.
- R3: A non-retired capability mechanism with stale triggers shall be an obligation.
- R4: A release newer than the newest verified one that is not verified or yanked shall be an obligation, unless it was left prepared or failed while a later release was tagged.
- R5: An accepted-risk or follow-up finding past its review date in the latest legacy review record of a scope shall be an obligation.
- R6: The four producers shall run on every read, so a phase is never judged by an unread producer, and a fresh install shall read complete coverage.

### Compatibility

- Source refs may now carry kind `project`; schema and manuals document it.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-attention-projects-every-source.md
- created: .pose/adr/2026-10-09-obligations-may-originate-on-the-project.md
- created: pose-mcp/internal/pose/obligation_sources.go
- created: pose-mcp/internal/pose/obligation_sources_test.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/obligation.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/cli/fresh_install_health_test.go
- modified: pose-mcp/schemas/v1/obligation.schema.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-10-05-pose-fresh-install-doctor-is-clean.md

### Delivery targets

- capability:attention-projects-every-source module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 4. Tasks

### Implementation
- [x] Project-level origin and node kinds
- [x] The four producers, run on every read

### Validation
- [x] Producer tests, fresh install coverage, phase and corpus tests
- [x] Attention measured on pose-dist and Harne8

## 5. Decisions

### Decision D1
- Date: 2026-10-09
- Context: the four sources belong to no spec, and the obligation contract only accepted spec, roadmap or milestone origins.
- Options considered: (a) a project-level origin; (b) anchor on a spec where one exists; (c) defer.
- Decision: (a), chosen by the maintainer.
- Rationale: only (a) makes coverage complete without inventing an owner.
- Consequences: see `.pose/adr/2026-10-09-obligations-may-originate-on-the-project.md`.

### Decision D2
- Date: 2026-10-09
- Context: pose-dist holds 21 unverified release records, almost all superseded, and a prepared v6.3.0 abandoned for v7.0.0.
- Decision: only releases newer than the newest verified one are owed, and a prepared or failed one is abandoned once a later release is tagged.
- Rationale: projecting history would bury the one release actually in flight.

## 6. Validation

### Strategy

`TestAttentionSourceDocsReviewMarksAreOwed`, `TestAttentionSourceCapabilityTriggersAreOwed`, `TestAttentionSourceReleaseQueueSkipsSupersededReleases` and `TestAttentionSourceOverdueFindingsAreOwed` exercise each producer with what is owed and what is not, and validate every obligation. `TestFreshInstallHealthAttentionCoverageIsComplete` reads a fresh install as complete with nothing unused, then an open docs mark as one gate. `TestAPhaseWithAnUnreadProducerIsUnknownNotClear` and the adversarial corpus keep a phase unknown when a release record cannot be read. Measured: pose-dist Attention reads complete, with v7.0.0 tagged and its publication unrecorded; Harne8 shows its four open docs review marks.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose ./internal/cli -run 'AttentionSource|FreshInstallHealthAttention'`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] capability:attention-projects-every-source check:attention-sources-integration evidence:integration test:TestAttentionSourceDocsReviewMarksAreOwed
- R2 [satisfied] capability:attention-projects-every-source check:attention-sources-integration evidence:integration test:TestAttentionSourceDocsReviewMarksAreOwed
- R3 [satisfied] capability:attention-projects-every-source check:attention-sources-integration evidence:integration test:TestAttentionSourceCapabilityTriggersAreOwed
- R4 [satisfied] capability:attention-projects-every-source check:attention-sources-integration evidence:integration test:TestAttentionSourceReleaseQueueSkipsSupersededReleases
- R5 [satisfied] capability:attention-projects-every-source check:attention-sources-integration evidence:integration test:TestAttentionSourceOverdueFindingsAreOwed
- R6 [satisfied] capability:attention-projects-every-source check:attention-sources-integration evidence:integration test:TestFreshInstallHealthAttentionCoverageIsComplete test:TestAPhaseWithAnUnreadProducerIsUnknownNotClear

## 7. Final Report

### Delivered scope

The obligation contract accepts a project-level origin, and Attention projects docs review marks, capability stale triggers, releases in flight and overdue legacy findings on every read. pose-dist now reads complete coverage and shows that v7.0.0 was tagged with no recorded publication; Harne8 shows its four open docs review marks.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups

- [open] Record the publication and verification evidence of v7.0.0, which Attention now shows as tagged with no recorded publication; the next release supersedes it if it is never recorded (owner:@pose-maintainers crit:medium review:2026-10-23)
