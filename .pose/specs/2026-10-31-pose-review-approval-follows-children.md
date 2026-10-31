---
slug: pose-review-approval-follows-children
status: in-progress
created_at: 2026-10-31
supersedes:
depends_on: pose-review-subject-later-removal
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:review-approval-follows-children
---

# Spec: A parent's approval stands only while its children's does

## 1. Intent

### Goal

The integration approval of a milestone or roadmap stands only while the review of each child stands, and retained verification picks the bundle matching picks, whatever the seal times say.

### Business value

Independent review of `pose-review-subject-later-removal` found two faults in what eighteen milestone seals are about to rely on. A child rejected after its parent was approved leaves the parent `closed`, fresh and approved (SR-R1-05, high). A bundle that replaced another governs while inputs match, but after drift retention scans by seal time and can return the replaced one (SR-R1-07, medium).

### Constraints

- A bundle's identity does not change: a child's later attestation is not part of the parent's sealed inputs.
- Verification stays read-only, and one operation verifies a child at most once.

### Non-goals

- The freshness of removal witnesses, which is `pose-removal-witness-freshness`.
- External milestone members, which federated acceptance judges, not local children.
- Retiring recorded approvals: a parent recovers when its child's review does, with no reseal.

## 2. Requirements

### Functional

- R1: Preparing a milestone or roadmap shall report a blocker for each child whose review does not stand: not approved, not fresh, with lost attestation history, or, when the parent's lifecycle is done, not closed.
- R2: Verification of a parent, current or retained, shall not report it approved while a child of its bundle does not stand, and shall name the child. Closeout and `pose review check` shall agree.
- R3: The rule shall be transitive: a roadmap does not stand over a milestone whose spec does not.
- R4: Bundle selection shall honour replacement lineage everywhere: a replacement comes before what it replaces and takes its place in time, for current matching, retention and the latest negative review, whatever the seal times, equal ones included.
- R5: A replaced bundle's approval shall not be retained; its negative review still counts.

### Non-functional

- A child is verified at most once per operation.

### Security

- A parent's approval cannot outlive a rejection of the work it integrates.

### Compatibility

- A closed parent whose child's review no longer stands stops verifying approved, which is the point. The changelog says so.

## 3. Technical Plan

### Affected areas

`internal/pose/review_bundle.go`, `internal/pose/review_closeout.go`, `internal/pose/review_reuse_provenance.go`, the manuals.

### Artifacts

- created: .pose/specs/2026-10-31-pose-review-approval-follows-children.md
- created: .pose/starts/pose-review-approval-follows-children.json
- created: .pose/changelogs/unreleased/pose-review-approval-follows-children.md
- modified: .pose/specs/2026-10-30-pose-review-subject-later-removal.md
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_reuse_provenance.go
- created: pose-mcp/internal/pose/review_approval_follows_children_test.go

### Delivery targets

- capability:review-approval-follows-children module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

A parent's verification, review check and closeout report a child that no longer stands.

## 4. Tasks

- [x] Lineage order and retention
- [x] Children stand in preparation, verification and closeout
- [ ] Measure real scopes and cost
- [ ] Independent review and closeout

## 5. Decisions

- D1: The rule reads the child's own verification, so every reason a child stops standing — rejection, lost history, reopening, revoked federation — counts without a list of its own.
- D2: The check runs at verification, not in the bundle: a sealed parent records which children it integrated, and whether they stand is a fact of now.
- D3: One lineage order serves every selection, so matching and retention cannot disagree. A replacement takes the newest position of its chain, since it was sealed after what it replaced whatever its seal time says.

## 6. Validation

### Strategy

Real sealed fixtures for each way a child stops standing, at spec, milestone and roadmap level; a before/after comparison of every real milestone and roadmap; cost on `roadmap:delegation`; mutants.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups

None yet.
