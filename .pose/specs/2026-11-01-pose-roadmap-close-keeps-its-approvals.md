---
slug: pose-roadmap-close-keeps-its-approvals
status: in-progress
created_at: 2026-11-01
supersedes:
depends_on: pose-review-approval-follows-children
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:roadmap-close-keeps-its-approvals
---

# Spec: Closing a roadmap keeps the approvals that closed it

## 1. Intent

### Goal

The guarded transition of a roadmap to done leaves standing the approvals it rested on, and those of milestones that were waiting for it.

### Business value

A sealed federated manifest binds the revision and digest of the roadmap file, and a closed scope keeps its approval only while the manifest is byte-identical. Writing `status: done` moves both. So closing `delegated-review` turned its seven approved milestones and the roadmap itself to superseded, and the eleven of `delegation` with them; `delegation` could then not be sealed. Every roadmap would hit this on closing, and a closed roadmap would read as unapproved for good.

### Constraints

- A change that bears on authority still stales a closed scope: trust policy, another project's source, a new blocker, a member, a moved or missing artifact.
- Any edit of a roadmap beyond its lifecycle fields still stales it.
- Sealed bundles keep their format and their digests.

### Non-goals

- Reviewing whether `pose close milestone` should judge the manifest as `review verify` does.
- Narrowing `depends_on` between roadmaps, which goes with `pose-milestone-closes-when-reached`.

## 2. Requirements

### Functional

- R1: A closed scope whose current federated manifest differs from the sealed one only by the revision, digest or status of a same-project roadmap or milestone, where that roadmap's file differs between the two revisions only in its `status` and `completed_at`, shall keep its approval.
- R2: A sealed `not-done` blocker of a dependency that is now done shall not stop the approval from standing.
- R3: Any other difference, including a status that left done, a new blocker, a change of trust, another project's source, a spec's source or a roadmap edit outside those two fields, shall stale the closed scope as before.

### Non-functional

- Git is read only when the manifests differ.

### Security

- The comparison reads committed bytes of both revisions, and a revision it cannot read does not stand.

### Compatibility

- Bundles sealed earlier are read as they are; nothing is resealed for this.

## 3. Technical Plan

### Affected areas

`internal/pose/review_bundle.go`.

### Artifacts

- created: .pose/specs/2026-11-01-pose-roadmap-close-keeps-its-approvals.md
- created: .pose/starts/pose-roadmap-close-keeps-its-approvals.json
- created: .pose/changelogs/unreleased/pose-roadmap-close-keeps-its-approvals.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/roadmap_close_keeps_approvals_test.go

### Delivery targets

- capability:roadmap-close-keeps-its-approvals module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

A closed scope stands over a lifecycle change of a same-project roadmap.

## 4. Tasks

- [ ] Judge a manifest difference by what changed
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: drop revisions and digests from the comparison
- Selected option: keep the byte comparison and excuse one difference, proved by reading both revisions of the roadmap
- Rationale: the rule that a changed manifest stales a closed scope protects authority, and a lifecycle field carries none; anything not proved to be that stays stale
- Consequences: sealed manifests keep meaning what they meant, at the cost of two Git reads when they differ
- Falsifier: a closed scope that stands over a manifest differing in anything but those fields

### Decision D2
- Basis: R2
- Minimal option: reseal every milestone that was waiting on the dependency
- Selected option: a `not-done` blocker cleared by the dependency reaching done is not a difference
- Rationale: the gate that held the milestone back was met, and nothing the review judged changed
- Consequences: milestones that wait on another roadmap close without a second review of the same bytes
- Falsifier: an approval standing over a blocker that is still true

## 6. Validation

### Strategy

Real seal, attestation and close of a roadmap in the federated fixture, verifying the milestone and its parent after the transition, and each refused difference over a real two-commit history. The gate fails on the byte comparison.

### Requirement trace

- R1 [planned] capability:roadmap-close-keeps-its-approvals evidence:integration check:roadmap-close-keeps-approvals-integration test:TestClosingARoadmapKeepsItsMilestoneApproved test:TestManifestStandsOverALifecycleChange — The transition leaves the approval standing.
- R2 [planned] check:roadmap-close-keeps-approvals-integration test:TestManifestStandsOverACompletedDependency — A cleared not-done does not stale it.
- R3 [planned] check:roadmap-close-keeps-approvals-integration test:TestManifestDifferencesThatStale test:TestEditingAClosedRoadmapStalesItsMilestone — Anything else stales it.

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
