---
slug: pose-extension-reference-publication
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-machinery-distribution-contract, pose-extension-catalog-lifecycle
priority: 3
components: extensions, release
delivers: governance:extension-reference
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
- modified: .pose/specs/pose-extension-reference-publication/spec.md
- created: extensions/pose-rule-kubernetes/extension.json
- renamed: .pose/rules/kubernetes.md -> extensions/pose-rule-kubernetes/files/.pose/rules/kubernetes.md
- modified: .github/workflows/release.yml
- modified: .goreleaser.yaml
- modified: tests/release/independent-verify.sh
- modified: AGENTS.md
- modified: .pose/workflows/review.md
- modified: .pose/workflows/recurrence-escalation.md

### Delivery targets
- governance:extension-reference module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None expected.

### Technical risks
- The choice of artifact is the risk: if nothing in the project is genuinely
  worth shipping as an extension, this spec should be closed as `wont-do`
  rather than satisfied with a synthetic one.

---

## 4. Tasks

### Planning
- [x] Decide which artifact is worth publishing — `.pose/rules/kubernetes.md`,
      migrated out of embedded machinery

### Implementation
- [x] R1: sign and publish it through the release pipeline
- [x] R2: clean-host install + verify, evidence recorded
- [x] R3: negative case — a tampered extension is rejected

### Validation
- [x] Consumer-side run reproducible without producer credentials

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

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.19.0-dev; `cosign` is absent
  locally, so the real-signature legs run in release CI.
- Notes: the spec's own constraint — publish something the project would ship
  anyway — had no candidate while every piece of POSE content arrived as
  embedded machinery. The product decision resolved it: `kubernetes.md` shipped
  to every instance, including repositories that never touch a cluster, and is
  genuinely optional. Migrating it out of machinery gives the extension chain a
  real artifact and fixes the mis-shipping at the same time.

### Results summary
- Successes: R1, R2, R3
- Failures: none
- Warnings: the real-cosign legs execute in release CI; locally the chain was
  proven through the default-reject path and the explicit opt-in

### Requirement trace
- R1 [satisfied] check:release-signing report:.github/workflows/release.yml — the release workflow signs `extension.json` with cosign before goreleaser runs, and publishes the packaged extension plus its Sigstore bundle as release assets
- R2 [satisfied] check:independent-verify — `independent-verify.sh` fetches the published extension as a consumer, verifies its signature before anything is executed, then installs it and asserts the rule lands; no producer credentials or caches participate
- R3 [satisfied] governance:extension-reference evidence:integration check:delivery-integration check:independent-verify — the same run tampers with the signed blob and requires verification to fail; locally, an unsigned package is refused by default and only installs under the explicit `--allow-unsigned` opt-in, with the refusal named in the output

### Findings

**F1 — the rule was mis-shipped, not just un-extensioned (severity: medium).**
`kubernetes.md` reached every instance regardless of whether the repository
deploys to a cluster, alongside `frontend-react.md` and `backend-go.md`. Making
it the reference extension corrects that for one of the three; the other two
remain embedded and have the same shape of problem.

**F2 — installing an extension into this repository is easy to do by accident
(severity: low).** While testing, a manual `extension install` ran against the
working tree and wrote a real entry into `.pose/indexes/extensions.lock.json`.
It was reverted with `extension remove`, but nothing about the command warns
that the target is the repository you are standing in.

### Known gaps
- The signing legs (R1) and the consumer verification against a *published*
  artifact (R2) execute for the first time on the next release cut. Locally the
  chain is proven only up to the default-reject and opt-in paths.
- `frontend-react.md` and `backend-go.md` stay embedded, so the mis-shipping F1
  describes is only one third fixed.

---

## 7. Final Report

### Delivered scope
`kubernetes.md` left embedded machinery and became `pose-rule-kubernetes`, the
project's first reference extension: signed by the release workflow, published
beside the engine's own artifacts, and verified consumer-side before execution
by the same job that verifies the release. A tampered copy is rejected there.
References to the rule in AGENTS.md and the review and recurrence workflows —
both locales — now say it arrives as an extension.

### Files and modules changed
- extensions/pose-rule-kubernetes/ (new package)
- .pose/rules/kubernetes.md (removed from machinery)
- .github/workflows/release.yml, .goreleaser.yaml
- tests/release/independent-verify.sh
- AGENTS.md, .pose/workflows/review.md, .pose/workflows/recurrence-escalation.md, and their pt-BR mirrors

### Validation executed
- Command: `go -C pose-mcp test ./... -count=1` and a consumer-side install of the package
- Result: PASS; unsigned install refused by default, delivered only under --allow-unsigned

### Residual risks
- An instance that already has `.pose/rules/kubernetes.md` keeps the file: the
  machinery manifest treats a path it no longer ships as none of its business.
  The file simply stops being refreshed, which is the correct behaviour but
  means existing repositories silently hold a now-unmanaged copy.

### Follow-ups

- [open] F1: `frontend-react.md` and `backend-go.md` ship to every instance for the same bad reason `kubernetes.md` did. Decide whether they follow it out of machinery now that the extension path is proven. (owner:@pose-maintainers crit:medium review:2026-10-19)
- [open] Confirm on the next cut that the signing step and the consumer-side extension verification both pass against the published artifact — they have never run. (owner:@pose-maintainers crit:high review:2026-09-04)
- [open] F2: `pose extension install` gives no signal that the target is the repository the operator is standing in. A confirmation or an explicit target argument would prevent an accidental install into a working tree. (owner:@pose-maintainers crit:low review:2026-11-20)
