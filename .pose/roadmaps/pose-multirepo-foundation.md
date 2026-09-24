---
slug: pose-multirepo-foundation
status: active
created_at: 2026-09-21
depends_on:
---

# Roadmap: Consistent multi-repository governance in POSE

Owner: @pose-maintainers. One authority per executable artifact and evidence-backed
composition. This roadmap owns only the four implementation specs below.
Consumer rollout belongs to project `proj.harne8`, roadmap
`harne8-multirepo-consistency`; no consumer spec is copied here.

## Architectural basis
- [Qualified authority](../adr/2026-09-21-qualified-artifact-authority-and-explicit-spec-transfer.md).
- [Federated acceptance](../adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md).
- Reuse `pose-cross-repo-portfolio` and existing project authorization. Do not
  reopen historical closeouts or introduce a second registry.
- Dates/release version remain uncommitted; use the existing release workflow.
  This plan does not introduce a separate intermediate ABM release.

## Milestone: resolution
- after: 
- target_start:
- target_due:
- specs: pose-qualified-artifact-resolution

### Gate de saída / Exit gate
All supported layouts and interfaces agree; checkout relocation preserves identity; failed external lookups never resolve a local same-slug spec.

## Milestone: composition
- after: resolution
- target_start:
- target_due:
- specs: pose-federated-roadmap-acceptance

### Gate de saída / Exit gate
Two independent Git projects compose spec/milestone/roadmap gates. Status-only approval fails; changed pinned evidence stales the coordinator review.

## Milestone: transfer
- after: resolution
- target_start:
- target_due:
- specs: pose-spec-authority-transfer

### Gate de saída / Exit gate
Preview, concurrent retry and interruption preserve history with no two executable authorities. Existing executors reconcile without replacing evidence.

## Milestone: agent-flow
- after: composition, transfer
- target_start:
- target_due:
- specs: pose-agent-project-context

### Gate de saída / Exit gate
Parent and executor entrypoints resolve the same qualified task, create no shadow spec and close only the authorized scope. Installed CLI/MCP and scaffold agree.

## Cut criteria

- C1: governance:federated-roadmap-acceptance check:federated-roadmap-acceptance-integration
- C2: check:federated-roadmap-negative-gates

## Activation and closeout
Composition implementation is active. Before starting each remaining spec, accept its ADRs,
reconcile Artifacts, declare its proposed target, register dedicated evidence
producers and run readiness. Existing profiles are not evidence for this corpus.
Materialize native typed Cut criteria before activating composition and exercise
one negative case per criterion. No consumer roadmap is a prerequisite here.

Close only after four source spec reviews/closeouts and generic fixture
composition pass, then run `pose roadmap-check pose-multirepo-foundation --strict`,
`pose review verify roadmap:pose-multirepo-foundation` and
`pose close roadmap:pose-multirepo-foundation`. A status count never closes it.
