---
slug: pose-extension-reference-publication
status: draft
created_at: 2026-08-07
completed_at:
supersedes:
depends_on: pose-machinery-distribution-contract, pose-extension-catalog-lifecycle
priority: 3
components: extensions, release
delivers:
---

# Spec: Prove the extension chain with a real published extension

## 1. Intent

### Goal
Publish one genuine signed extension end to end through the release-signing
pipeline, so the extension contract is demonstrated on a real artifact rather
than only in unit tests.

### Business value
`pose extension install/verify` is implemented and unit-tested, and the signing
pipeline is exercised on the engine's own release artifacts. What has never
happened is the two meeting: no extension has ever been signed, published,
fetched and verified by a consumer. Every claim about third-party extension
distribution is therefore inference from two separately-working halves.

This was R4 of `pose-machinery-distribution-contract`. It came out because that
spec could finish without it and because satisfying it requires deciding *what*
to publish — a product question, not a delivery-contract question.

### Constraints
- The published extension must be something the project would ship anyway.
  Inventing a throwaway artifact to satisfy a checkbox proves nothing about the
  chain, because nobody would ever install it.
- Verification must run as a consumer: no producer credentials, no local build.

### Non-goals
- Building an extension registry or a catalog service.
- Changing the extension format or its signing scheme.

---

## 2. Requirements

### Functional
- R1: One real extension shall be signed and published through the existing
  release-signing pipeline.
- R2: A clean-host run shall install that extension from its published location
  and verify its signature before execution, with the evidence recorded.
- R3: An extension whose signature does not verify shall be rejected by the same
  path, proving the check is load-bearing rather than decorative.

### Non-functional
- The verification run must be repeatable by anyone, not only by a maintainer
  with repository credentials.

### Security
- An unsigned or tampered extension stays installable only under the existing
  explicit opt-in, and that opt-in must be visible in the run's output.

### Compatibility
- No change to the extension format; this spec exercises what exists.

---

## 3. Technical Plan

### Affected areas
- Release-signing pipeline — extend to an extension artifact.
- `pose extension install|verify` — exercised, not modified.

### Artifacts
<!-- Declared at closeout. -->
- created: .pose/specs/pose-extension-reference-publication/spec.md

### API/contract changes
- None expected.

### Technical risks
- The choice of artifact is the risk: if nothing in the project is genuinely
  worth shipping as an extension, this spec should be closed as `wont-do`
  rather than satisfied with a synthetic one.

---

## 4. Tasks

### Planning
- [ ] Decide which artifact is worth publishing, or conclude that none is

### Implementation
- [ ] R1: sign and publish it through the release pipeline
- [ ] R2: clean-host install + verify, evidence recorded
- [ ] R3: negative case — a tampered extension is rejected

### Validation
- [ ] Consumer-side run reproducible without producer credentials

---

## 6. Validation

### Strategy
Mirror what `verify-release.yml` does for the engine: fetch the published
extension on a clean host, verify signature before executing anything, then
repeat with a deliberately corrupted copy and require rejection.

### Deterministic checks

#### Security / Contract
- Command: `pose extension verify <published-extension-dir>`
- Scope: signature verification on a clean host
- Expected: passes for the published artifact, fails for a tampered copy

### Requirement trace
<!-- Filled at closeout. -->

### Known gaps
- Blocked on a product decision, not on engineering: there may simply be nothing
  worth publishing yet.

---

## 7. Final Report

<!-- Filled at closeout. -->

### Follow-ups

- [open] If no artifact is worth publishing after a full cycle, close this as wont-do and say plainly in the docs that third-party extension distribution is unproven end to end. (owner:@pose-maintainers crit:medium review:2026-12-18)
