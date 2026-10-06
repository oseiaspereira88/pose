---
slug: pose-authority-claim-project-is-not-the-audience
status: in-progress
created_at: 2026-10-05
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: spec:pose-abm-review-authority@defect-fix
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:authority-claim-project-binding
---

# Spec: A signed authority claim binds its project and its audience separately

## 1. Intent

### Goal

Give the two bindings of a signed authority claim their own meaning: `project` names the project the decision governs and is compared with a new `authority_project` in the review policy; `audience` names the verifier installation meant to accept it and stays compared with `authority_audience`. Both review claims and action-request claims follow the same rule.

### Business value

pose-abm-review-authority left open that `Project` and `Audience` were both compared with `authority_audience`, so one field was redundant and a claim could not say which project it governs apart from which verifier it was issued to. With one Harne8 installation serving several projects, the audience is shared, and only a project binding stops a claim issued for one project from being replayed in another. No instance has adopted `verified`, and the Harne8 issuer is not built yet, so 7.0.0 is the moment to settle the contract it will implement. The maintainer chose this in Decision 5 (A-extended, alternative b).

### Constraints

The project binding comes from the review policy, which the protected baseline covers, never from the environment or the directory name, which a verifier run can change. Verified mode stays opt-in.

### Non-goals

Building the Harne8 issuer; changing the claim's signature or envelope format.

## 2. Requirements

### Functional

- R1: The review policy shall accept `authority_project`; verified identity assurance shall require it to be a valid project id, as it requires `authority_audience`.
- R2: A review authority claim shall be refused when its `project` differs from `authority_project` or its `audience` differs from `authority_audience`, each with its own reason; a claim whose project and audience are both right shall pass.
- R3: An action-request claim shall follow the same rule, and shall be refused when the review policy declares no `authority_project`.
- R4: The manual shall describe both bindings.

### Non-functional

- None beyond the shared constraints.

### Security

- A claim issued for another project served by the same verifier installation is refused.

### Compatibility

- Breaking only for a policy that already adopted `verified` without `authority_project`; no instance has.

## 3. Technical Plan

### Affected areas

Review policy reader, review authority verification, action-request claim verification, manuals.

### Artifacts

- created: .pose/specs/2026-10-05-pose-authority-claim-project-is-not-the-audience.md
- created: .pose/starts/pose-authority-claim-project-is-not-the-audience.json
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/review_authority_test.go
- modified: pose-mcp/internal/pose/action_resolution_test.go
- created: pose-mcp/internal/pose/authority_project_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- renamed: .pose/changelogs/unreleased/pose-authority-claim-project-is-not-the-audience.md -> .pose/changelogs/v7.0.0/pose-authority-claim-project-is-not-the-audience.md

### Delivery targets

- capability:authority-claim-project-binding module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- None beyond the compatibility note.

## 6. Validation

### Strategy

Signed fixture claims for review and action requests with the right binding, a foreign project under the same audience, and a foreign audience; a verified policy without `authority_project`.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose -run 'AuthorityProject|ABMReviewAuthority|ActionResolution'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestAuthorityProjectIsRequiredByVerifiedAssurance check:authority-project-integration
- R2 [satisfied] test:TestAuthorityProjectBindsAReviewClaimApartFromItsAudience test:TestABMReviewAuthorityValid check:authority-project-integration
- R3 [satisfied] test:TestAuthorityProjectBindsAnActionClaimApartFromItsAudience test:TestAVerifiedClaimBoundToTheRequestIsAccepted check:authority-project-integration
- R4 [satisfied] test:TestAuthorityProjectIsDocumented check:authority-project-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None.

### Follow-ups

None.
