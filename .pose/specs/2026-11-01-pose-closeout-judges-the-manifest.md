---
slug: pose-closeout-judges-the-manifest
status: done
created_at: 2026-11-01
supersedes:
depends_on: pose-roadmap-close-keeps-its-approvals
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:closeout-judges-the-manifest
completed_at: 2026-11-01
---

# Spec: Closeout judges a retained approval as review verify does

## 1. Intent

### Goal

`review verify` and closeout give one answer on whether a closed scope keeps its approval over a changed federated manifest.

### Business value

`review verify` refuses to retain an approval when the manifest moved in a way that bears on authority. `reviewCheck`, behind `pose close milestone:`, `review-check` and the closeout state, then restored the same approval without comparing anything. So at `808ff57a` closeout read approved where verify read superseded, and revoked trust or an edited roadmap was visible on one reader only. A gate that two readers answer differently protects neither.

### Constraints

- What `review verify` retains stays retained: this spec changes the reader that was wrong, not the rule.
- A verification that failed cannot show the manifest stands, so it does not retain under a resolver.
- A scope with no resolver is read as before: no manifest can be judged there.

### Non-goals

- Waiving bundles sealed before manifests existed. Three review rounds found ways through such a waiver, and nothing depends on those approvals; they are resealed instead.
- A matching sealed bundle awaiting its attestation, where closeout may still retain an earlier approval that verify does not. Its manifest is judged by R1, and the retention of approvals given under older rules is deliberate.
- Legacy review attempts of scopes closed before bundles were adopted: they have no bundle, so no manifest, and closeout reads them under the policy's exemption while verify does not. 55 scopes of this repository read that way, before and after this spec.
- Roadmap `depends_on` and the reach of milestones, which go to `pose-milestone-closes-when-reached`.

## 2. Requirements

### Functional

- R1: Whether a closed scope's retained bundle stands against its current manifest shall be decided by one function, called by `review verify` and by closeout.
- R2: When that function refuses, closeout shall not read the scope approved or fresh, shall report the state `review verify` reports, and shall carry a blocker naming the bundle and the manifest change.
- R3: When a verification fails under a resolver and the scope has a retained bundle, closeout shall return that failure, and no legacy attempt shall stand in for the bundle.
- R4: A closed scope `review verify` retains shall stay approved in closeout, including over the guarded close of its roadmap.

### Non-functional

- No second preparation: closeout judges against the manifest the verification already resolved.

### Security

- The change only removes a path to approval.

### Compatibility

- Bundles and attestations are read as they are; nothing is resealed.

## 3. Technical Plan

### Affected areas

`internal/pose/review_bundle.go`, `internal/pose/review_closeout.go`.

### Artifacts

- created: .pose/specs/2026-11-01-pose-closeout-judges-the-manifest.md
- created: .pose/starts/pose-closeout-judges-the-manifest.json
- created: .pose/changelogs/unreleased/pose-closeout-judges-the-manifest.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- created: pose-mcp/internal/pose/closeout_judges_manifest_test.go

### Delivery targets

- capability:closeout-judges-the-manifest module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`review-check`, `pose close milestone:` and the closeout state refuse what `review verify` refuses over a changed manifest.

## 4. Tasks

- [x] Share the judgment between verify and closeout
- [x] Measure every closed scope of this repository, old reader against new
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: drop the closeout fallback and read the verification alone
- Selected option: keep the fallback and pass it through the function verify uses
- Rationale: the fallback also carries approvals given under rules that did not exist then, which verify does not retain; removing it would stale those for a reason unrelated to the manifest
- Consequences: two retention paths remain, and they differ only by that deliberate retention
- Falsifier: a changed manifest over which the two readers disagree

### Decision D2
- Basis: R3
- Minimal option: keep retaining when the verification failed
- Selected option: return the failure under a resolver, ahead of the exemption for legacy attempts
- Rationale: an approval cannot be shown to stand against a manifest that was not read
- Consequences: a closed scope whose preparation fails reports that failure instead of an approval, and the exemption keeps serving only scopes with no retained bundle
- Falsifier: a closed scope in this repository that a verification failure turns unapproved with nothing wrong with its review

### Decision D3
- Status: withdrawn
- Basis: R1
- Minimal option: let the old seals stale, as verify already read them
- Selected option: a seal with no manifest, dated before manifests entered the engine, stands where the current manifest binds only its own project and each artifact it names is, byte for byte, in the tree at the seal
- Rationale: applying R1 turned nine milestones of two August roadmaps unapproved for a field that did not exist when they were sealed, while their current manifests name only their own done specs; the difference bears on no authority. The absent field alone is not proof of age, since a store with no resolver seals none today (review round 1), so the date bounds it, fail-closed for an instance that adopted later. A commit date does not serve: the migration of the specs to the flat layout on 2026-08-22 is newer than these seals, yet moved no bytes. Child verification does not serve either, since a done spec keeps its approval over an edit (review round 2). Bytes tell the two apart
- Consequences: both readers keep those milestones approved, and their two roadmaps, which verify refused through them, now read closed; a manifest naming another project still stales them
- Falsifier: a seal with no manifest standing over a manifest that names another project, carries a blocker or is not ready, over a later edit of an artifact it names, or sealed after 2026-09-24

## 6. Validation

### Strategy

Real seal and attestation in the federated fixture, each case asserted through both readers, with the closeout tests shown failing without the change. The old and new CLI compared over all 602 scopes of this repository: nine milestones of two August roadmaps lose the approval closeout restored for them, which verify already refused, and nothing else changes. `check --strict` passes.

### Requirement trace

- R1 [satisfied] capability:closeout-judges-the-manifest evidence:integration check:closeout-judges-manifest-integration test:TestCloseoutRefusesARoadmapEditVerifyRefused test:TestCloseoutRefusesRevokedTrust test:TestRetainedApprovalIsNotJudgedWithoutAResolver test:TestSealWithoutAManifestStales report:.pose/results/delivery-validation.json commit:cc6203a8d3d761961a76336ba62a823b11b9d033 — readers agree
- R2 [satisfied] check:closeout-judges-manifest-integration test:TestCloseoutRefusesARoadmapEditVerifyRefused test:TestCloseoutRefusesARoadmapEditWhenClosed report:.pose/results/delivery-validation.json commit:cc6203a8d3d761961a76336ba62a823b11b9d033 — refusal and state
- R3 [satisfied] check:closeout-judges-manifest-integration test:TestCloseoutRefusesAFailedVerification test:TestCloseoutRefusesALegacyAttemptOverAFailedVerification report:.pose/results/delivery-validation.json commit:cc6203a8d3d761961a76336ba62a823b11b9d033 — no judgment, no approval
- R4 [satisfied] check:closeout-judges-manifest-integration test:TestCloseoutKeepsWhatVerifyRetains report:.pose/results/delivery-validation.json commit:cc6203a8d3d761961a76336ba62a823b11b9d033 — retention kept

## 7. Final Report

### Delivered scope

`review verify` and closeout now share one judgment of a retained approval against the current manifest, and a failed verification ends closeout ahead of the exemption for legacy attempts. Independently reviewed by Codex in four rounds: the first found a legacy attempt restoring approval over a failed verification, and rounds one to three each broke a waiver for seals older than manifests, which was withdrawn.

### Residual risks

Nine milestones of `adaptive-instance-provisioning` and `agentic-onboarding` were sealed before manifests existed and now read superseded in closeout as they always did in verify; `check --strict` still passes. Closeout may still retain an approval verify does not where a matching bundle awaits attestation, its manifest judged. The 55 scopes closed before bundles read approved in closeout only, under the policy's exemption.

### Follow-ups

- [open] F1: Reseal the nine August milestones sealed before manifests existed, then recheck `adaptive-instance-provisioning` and `agentic-onboarding` (review F7). (owner:@oseiaspereira88 crit:low review:2026-11-27)
