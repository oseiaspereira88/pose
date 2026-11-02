---
slug: pose-community-claims-hold
status: done
created_at: 2026-11-02
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:community-claims-hold
completed_at: 2026-11-02
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

- R1: `pose public-claims` shall treat a release tag link, the label of a release link, a release badge and a version in a sentence that says latest or current as version claims, with or without the `v` and keeping a prerelease suffix, except between the release-history markers of a surface declared `history_block`; a bare version in other prose is history, not a claim.
- R2: The README, in both locales, shall hold no per-release text and no hardcoded count of what ships.
- R3: The quickstart shall claim no time budget while no measurement of the current path exists.
- R4: The demo shall block a delivery, have it reviewed, close it through `pose close`, and fail unless the spec reads done and `pose review verify` approves it.

### Non-functional

- No new read.

### Security

- A false version on a public surface is a supply-chain claim: readers install what the page names.

### Compatibility

- A surface that names an older release as current, in a badge or in a release link now fails the gate, unless it sits in the release-history block of a surface declared `history_block`.

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
- modified: tests/hygiene/temp-cleanup.pins
- modified: docs-site/docs/evolution.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/public-claims.json
- modified: .pose/templates/public-claims.json
- modified: .pose/public/claims.json

### Delivery targets

- capability:community-claims-hold module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`public-claims` recognizes one more shape of version claim.

## 4. Tasks

- [x] Recognize a release tag link
- [x] Make the README evergreen
- [x] Remove the budget the quickstart cannot support
- [x] Close a delivery in the demo
- [x] Independent review and closeout

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

- R1 [satisfied] capability:community-claims-hold evidence:integration check:community-claims-hold-integration test:TestPublicClaimsFailsOnAStaleReleaseTagLink test:TestPublicClaimsFailsOnEveryShapeOfAStaleRelease test:TestOnlyTheReleaseHistoryBlockHoldsPastReleases test:TestAHistoryBlockCountsOnlyOnADeclaredSurface report:.pose/results/delivery-validation.json commit:f5dadca00044c68ddd0275b5e89bfeb1cbd22f5a
- R2 [satisfied] check:community-claims-hold-integration test:TestReadmeHoldsNoPerReleaseText report:.pose/results/delivery-validation.json commit:f5dadca00044c68ddd0275b5e89bfeb1cbd22f5a
- R3 [satisfied] check:community-claims-hold-integration test:TestQuickstartClaimsNoCurrentBudget report:.pose/results/delivery-validation.json commit:f5dadca00044c68ddd0275b5e89bfeb1cbd22f5a
- R4 [satisfied] check:demo-closes-a-delivery report:.pose/results/delivery-validation.json commit:f5dadca00044c68ddd0275b5e89bfeb1cbd22f5a

## 7. Final Report

### Delivered scope

`pose public-claims` now reads a release as a reader meets it: tag links, the label of a release link, badges and sentences that say latest or current, with or without the `v` and keeping a prerelease suffix. Past releases stand only in a marked block of a declared surface. The READMEs hold no per-release text and no count of what ships, the quickstart claims no budget until a current measure exists, and the demo blocks a delivery, has it reviewed and closes it through the gate in under three seconds of machine time, checked by the matrix. Independently reviewed by Codex in four rounds: the first refused a wording change in place of a real close.

### Residual risks

The reviewer of the demo is the script, and it says so. A bare version in prose is treated as history, so a stale version with no word saying it is current passes. The gate covers declared surfaces only.

### Follow-ups

- [open] F1: Measure the quickstart on a clean container and restore a dated budget; Docker was off during this spec. (owner:@oseiaspereira88 crit:medium review:2026-11-27)
- [open] F2: Record the demo under 60 seconds and embed it, and publish a scoped contributor task and the Discussions space, which `community-surfaces` still owes. (owner:@oseiaspereira88 crit:medium review:2026-11-27)
- [open] F3: Decide whether the landing page of Harne8 and the old docs host are surfaces of this gate, which `single-public-truth` still owes. (owner:@oseiaspereira88 crit:medium review:2026-11-27)
