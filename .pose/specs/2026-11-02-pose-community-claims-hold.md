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

The public gate catches a false release claim, and the README, the quickstart and the demo say only what is true.

### Business value

The integration review of `community-launch` found the public gate passing a README that named v0.0.1 as the latest release, a README carrying per-release text and a hardcoded tool count that its own spec forbids, a quickstart timing measured on v7.0.0 and presented as current, and a demo that prints "legitimately closed" with no close having run.

### Constraints

- The demo keeps fitting under a minute, so it shows the gate and does not perform a review.

### Non-goals

- Publishing the recording, a contributor task or a Discussions space.
- Which external surfaces the gate owns: a decision of the maintainer.
- A new clean measurement of the quickstart, which needs a container runtime.

## 2. Requirements

### Functional

- R1: `pose public-claims` shall treat a release tag link as a version claim.
- R2: The README, in both locales, shall hold no per-release text and no hardcoded tool count.
- R3: The quickstart shall state the version and date of its timing.
- R4: The demo shall not say a delivery closed unless a close ran.

### Non-functional

- No new read.

### Security

- A false version on a public surface is a supply-chain claim: readers install what the page names.

### Compatibility

- A surface that links an older release tag now fails the gate, unless the surface is declared as history.

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

### Delivery targets

- capability:community-claims-hold module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`public-claims` recognizes one more shape of version claim.

## 4. Tasks

- [ ] Recognize a release tag link
- [ ] Make the README evergreen
- [ ] Date the timing of the quickstart
- [ ] Say what the demo shows
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R4
- Minimal option: change the closing line of the demo
- Selected option: that, and not a real close
- Rationale: a real close needs a review and evidence, which no demo of a minute can hold; what it shows is that a gate refuses and then clears
- Consequences: the demo proves the gate, not a whole delivery
- Falsifier: a reader who takes the demo for a closed delivery

## 6. Validation

### Strategy

A negative case in the public gate, seen failing first, and the gate run over this repository.

### Requirement trace

- R1 [planned] capability:community-claims-hold evidence:integration check:community-claims-hold-integration test:TestPublicClaimsFailsOnAStaleReleaseTagLink
- R2 [planned] check:community-claims-hold-integration test:TestReadmeHoldsNoPerReleaseText
- R3 [planned] check:community-claims-hold-integration test:TestQuickstartDatesItsTiming
- R4 [planned] check:community-claims-hold-integration test:TestDemoDoesNotClaimAClose

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
