---
slug: pose-open-backlog-reconciliation
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-review-assurance-disclosure
priority: 0
components: pose-mcp
task_type: refactor
---

# Spec: Reconcile open specs and follow-ups against requirements and current evidence

## 1. Intent

### Goal

Produce a requirement-by-requirement inventory of the twelve non-terminal specs and a
triage of the open follow-ups, then apply only the dispositions that evidence and the
competent authority support.

### Business value

The index lists 12 non-terminal specs and the historical snapshot 123 open follow-ups,
with an explicit warning that the counts are not the implementation backlog. New
features created from raw counts would repeat delivered work.

### Constraints

Code existence never authorizes an auto-close without adequacy to the requirement.
Harne8-owned integration items point to the Harne8 authority (`harne8-pose-open-
integrations-reconciliation`) without asserting completion. External operations (Issues,
Discussions, host redirects, native channels) are not performed.

Program source: backlog items POSE-06 (P0, wave 0) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F10; sources
E01, E18, E22, E23, E25, E26, E31, E32). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Publishing Issues or Discussions; reversing the deferrals of native package channels or
the docs host; implementing new features for open statuses.

### Anti-mechanization guardrail

A existência de código não autoriza auto-close sem adequação ao requisito.

## 2. Requirements

### Functional

- R1: Every remaining requirement of the twelve non-terminal specs shall be classified as implementation, evidence, review, acceptance, adoption or external operation, with the source consulted.
- R2: The quickstart spec shall distinguish the measured automated run (6.964 s, excluding reading and human development) from any human acceptance still required, and the R4 criterion shall be adjudicated or amended explicitly.
- R3: Each open follow-up shall be classified as material implementation, evidence review/acceptance, adoption/pilot, external or human operation, accepted/deferred residual, duplicate or satisfaction candidate, or missing intent/owner.
- R4: A disposition shall be applied only with evidence and an identified authority; candidates without that stay open with the reason recorded.
- R5: The reconciliation report shall name its snapshot (commit, index digest) and shall not compare counts across snapshots as productivity.

### Non-functional

- Report retained under `.pose/reports/` and linked from each touched spec.

### Security

- No external publication.

### Compatibility

- Dispositions use the existing follow-up formats.

## 3. Technical Plan

### Affected areas

The twelve non-terminal specs, follow-up inventory, reports.

### Artifacts

- created: .pose/specs/2026-10-04-pose-open-backlog-reconciliation.md
- created: .pose/reports/2026-10-pose-open-backlog-reconciliation.md
- created: .pose/results/pose-open-backlog-reconciliation.json
- modified: .pose/specs/2026-09-08-pose-contract-adoption-registry.md
- modified: .pose/specs/2026-09-09-pose-bundles-seal-the-contracts-that-govern-them.md
- modified: .pose/specs/2026-09-10-pose-bundle-findings-take-the-contract-the-legacy-path-had.md
- modified: .pose/specs/2026-09-10-pose-changelog-adoption-is-the-instances.md
- modified: .pose/specs/2026-09-19-pose-abm-review-authority.md
- modified: .pose/specs/2026-09-26-pose-federated-milestone-scoped-acceptance.md
- modified: .pose/specs/2026-09-26-pose-roadmap-gate-scopes-milestones-and-external-members.md
- modified: .pose/specs/2026-08-08-pose-dependency-pin-refresh.md
- modified: .pose/specs/2026-09-06-pose-release-security-gate-integrity.md
- modified: .pose/specs/2026-09-19-pose-abm-review-soundness.md
- modified: .pose/specs/2026-09-29-test-git-repos-run-no-background-maintenance.md
- modified: .pose/specs/2026-10-02-pose-dependabot-runtime-repair.md
- modified: .pose/specs/2026-10-04-pose-review-assurance-disclosure.md
- modified: .github/dependabot.yml
- modified: .pose/specs/2026-08-16-pose-update-instance-directory-completeness.md
- created: .pose/changelogs/unreleased/pose-trace-test-refs-resolve.md
- created: .pose/changelogs/unreleased/pose-range-names-its-other-work.md
- created: .pose/changelogs/unreleased/pose-attention-projects-every-source.md

Reconciled against the tree at activation.

### Technical risks

- Over-closing to shrink numbers; guarded by evidence-plus-authority rule and review.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-open-backlog-reconciliation`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Read each spec and its evidence; record per-requirement classification; review the
dispositions before applying; `pose followups --open` before/after with snapshot ids.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./...`
- Scope: engine packages touched by this spec
- Expected: pass, including the new negative tests

#### Lint
- Command: `cd pose-mcp && go vet ./...`
- Scope: pose-mcp
- Expected: no findings

#### Security / Contract
- Command: `pose lint-spec pose-open-backlog-reconciliation --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
