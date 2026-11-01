---
slug: pose-closeout-judges-the-manifest
status: in-progress
created_at: 2026-11-01
supersedes:
depends_on: pose-roadmap-close-keeps-its-approvals
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:closeout-judges-the-manifest
completed_at:
---

# Spec: Closeout judges a retained approval as review verify does

## 1. Intent

### Goal

`review verify` and closeout give one answer on whether a closed scope keeps its approval over a changed federated manifest.

### Business value

`review verify` refuses to retain an approval when the manifest moved in a way that bears on authority. `reviewCheck`, behind `pose close milestone:`, `review-check` and the closeout state, then restored the same approval without comparing anything. So at `808ff57a` closeout read approved where verify read superseded, and revoked trust or an edited roadmap was visible on one reader only. A gate that two readers answer differently protects neither.

### Constraints

- What `review verify` retains stays retained; the only addition to it is R5.
- A verification that failed cannot show the manifest stands, so it does not retain under a resolver.
- A scope with no resolver is read as before: no manifest can be judged there.

### Non-goals

- A matching sealed bundle awaiting its attestation, where closeout may still retain an earlier approval that verify does not. Its manifest is judged by R1, and the retention of approvals given under older rules is deliberate.
- Legacy review attempts of scopes closed before bundles were adopted: they have no bundle, so no manifest, and closeout reads them under the policy's exemption while verify does not. 55 scopes of this repository read that way, before and after this spec.
- Roadmap `depends_on` and the reach of milestones, which go to `pose-milestone-closes-when-reached`.

## 2. Requirements

### Functional

- R1: Whether a closed scope's retained bundle stands against its current manifest shall be decided by one function, called by `review verify` and by closeout.
- R2: When that function refuses, closeout shall not read the scope approved or fresh, shall report the state `review verify` reports, and shall carry a blocker naming the bundle and the manifest change.
- R3: When a verification fails under a resolver and the scope has a retained bundle, closeout shall return that failure, and no legacy attempt shall stand in for the bundle.
- R4: A closed scope `review verify` retains shall stay approved in closeout, including over the guarded close of its roadmap.
- R5: A bundle with no manifest, sealed before the engine began to seal manifests on 2026-09-24, shall stand against a current manifest that is ready, has no blockers and names no other project, while the scope's own artifact is no newer than the seal, and against no other. A later bundle with no manifest shall not.

### Non-functional

- No second preparation: closeout judges against the manifest the verification already resolved.

### Security

- R1 to R3 only remove a path to approval. R5 adds one, for seals that bound nothing, and only where the scope answers to its own project alone.

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

- [ ] Share the judgment between verify and closeout
- [ ] Measure every closed scope of this repository, old reader against new
- [ ] Independent review and closeout

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
- Basis: R5
- Minimal option: let the old seals stale, as verify already read them
- Selected option: a seal with no manifest, dated before manifests entered the engine, stands where the current manifest binds only its own project and the scope's own artifact is no newer than the seal
- Rationale: applying R1 turned nine milestones of two August roadmaps unapproved for a field that did not exist when they were sealed, while their current manifests name only their own done specs; the difference bears on no authority. The absent field alone is not proof of age, since a store with no resolver seals none today (review round 1), so the date bounds it, fail-closed for an instance that adopted later. Member specs are not dated: the migration of the specs to the flat layout on 2026-08-22 is newer than these seals, and they answer for themselves as children
- Consequences: both readers keep those milestones approved, and their two roadmaps, which verify refused through them, now read closed; a manifest naming another project still stales them
- Falsifier: a seal with no manifest standing over a manifest that names another project, carries a blocker or is not ready, over a later edit of its own artifact, or sealed after 2026-09-24

## 6. Validation

### Strategy

Real seal and attestation in the federated fixture, each case asserted through both readers, with the closeout tests shown failing without the change. The old and new CLI compared over all 602 scopes of this repository: no approval lost, two roadmaps gained through R5. `check --strict` passes.

### Requirement trace

- R1 [planned] capability:closeout-judges-the-manifest evidence:integration check:closeout-judges-manifest-integration test:TestCloseoutRefusesARoadmapEditVerifyRefused test:TestCloseoutRefusesRevokedTrust test:TestRetainedApprovalIsNotJudgedWithoutAResolver — readers agree
- R2 [planned] check:closeout-judges-manifest-integration test:TestCloseoutRefusesARoadmapEditVerifyRefused test:TestCloseoutRefusesARoadmapEditWhenClosed — refusal and state
- R3 [planned] check:closeout-judges-manifest-integration test:TestCloseoutRefusesAFailedVerification test:TestCloseoutRefusesALegacyAttemptOverAFailedVerification — no judgment, no approval
- R4 [planned] check:closeout-judges-manifest-integration test:TestCloseoutKeepsWhatVerifyRetains — retention kept
- R5 [planned] check:closeout-judges-manifest-integration test:TestOldSealStandsWhileNothingMoved test:TestOldSealStalesOverALaterEdit test:TestSealWithoutAManifestAfterTheyExistedStales test:TestOldSealRefusesWhatItCouldNotHaveBound — old seals

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
