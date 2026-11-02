---
slug: pose-community-claims-hold
status: in-progress
created_at: 2026-11-02
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:community-claims-hold
completed_at:
---

# Spec: What the project says in public holds

## 1. Intent

### Goal

The public gate catches a false release claim in the forms a reader meets it, the README and the quickstart say only what holds, and the demo closes a delivery through the gate.

### Business value

The integration review of `community-launch` found the public gate passing a README that named v0.0.1 as the latest release, a README carrying per-release text and a hardcoded tool count that its own spec forbids, a quickstart timing measured on v7.0.0 and presented as current, and a demo that prints "legitimately closed" with no close having run.

### Constraints

- The demo keeps fitting under a minute, with its reviewer stated as the script.

### Non-goals

- Publishing the recording, a contributor task or a Discussions space.
- Which external surfaces the gate owns: a decision of the maintainer.
- A new clean measurement of the quickstart, which needs a container runtime: until then the page claims no budget.

## 2. Requirements

### Functional

- R1: `pose public-claims` shall treat a release tag link, a release badge and a version in prose as version claims, except between the release-history markers of a surface declared `history_block`.
- R2: The README, in both locales, shall hold no per-release text and no hardcoded count of what ships.
- R3: The quickstart shall claim no time budget while no measurement of the current path exists.
- R4: The demo shall block a delivery, have it reviewed, close it through `pose close`, and fail unless the spec reads done and `pose review verify` approves it.

### Non-functional

- No new read.

### Security

- A false version on a public surface is a supply-chain claim: readers install what the page names.

### Compatibility

- A surface that names an older release in prose, in a badge or in a tag link now fails the gate, unless it sits in the release-history block of a surface declared `history_block`.

## 3. Technical Plan

### Affected areas

`internal/cli/publicclaims.go`, `README.md`, `README.pt-BR.md`, `docs-site/docs/quickstart.md`, `examples/demo/record.sh`.

### Artifacts

- created: .pose/specs/2026-11-02-pose-community-claims-hold.md
- created: .pose/starts/pose-community-claims-hold.json
- created: .pose/changelogs/unreleased/pose-community-claims-hold.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/cli/publicclaims.go
- modified: pose-mcp/internal/cli/publicclaims_test.go
- modified: README.md
- modified: README.pt-BR.md
- modified: docs-site/docs/quickstart.md
- modified: examples/demo/record.sh
- modified: docs-site/docs/evolution.md
- modified: .pose/public/claims.json

### Delivery targets

- capability:community-claims-hold module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`public-claims` recognizes one more shape of version claim.

## 4. Tasks

- [ ] Recognize a release tag link
- [ ] Make the README evergreen
- [ ] Remove the budget the quickstart cannot support
- [ ] Close a delivery in the demo
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R4
- Minimal option: change the closing line of the demo and keep its lint
- Selected option: a real close, reusing the path of `tests/quickstart/first-governed-loop.sh`, which takes under three seconds of machine time
- Rationale: the release cut criteria of `community-launch` ask for a delivery that is blocked and then legitimately closed, and a wording cannot redefine that; the first attempt at this spec tried and was refused in review
- Consequences: the script stands in for the reviewer and says so in the conclusion it records; a check of the matrix runs it
- Falsifier: a reader who takes the scripted review for an independent one

## 6. Validation

### Strategy

A negative case in the public gate, seen failing first, and the gate run over this repository.

### Requirement trace

- R1 [planned] capability:community-claims-hold evidence:integration check:community-claims-hold-integration test:TestPublicClaimsFailsOnAStaleReleaseTagLink test:TestPublicClaimsFailsOnEveryShapeOfAStaleRelease test:TestOnlyTheReleaseHistoryBlockHoldsPastReleases test:TestAHistoryBlockCountsOnlyOnADeclaredSurface
- R2 [planned] check:community-claims-hold-integration test:TestReadmeHoldsNoPerReleaseText
- R3 [planned] check:community-claims-hold-integration test:TestQuickstartClaimsNoCurrentBudget
- R4 [planned] check:demo-closes-a-delivery

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
