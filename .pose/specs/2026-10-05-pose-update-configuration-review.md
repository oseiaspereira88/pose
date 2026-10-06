---
slug: pose-update-configuration-review
status: done
created_at: 2026-10-05
completed_at: 2026-10-06
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:configuration-review
---

# Spec: An update that brings capabilities asks the maintainer, in a spec

## 1. Intent

### Goal

When `pose update` finds capabilities new since the last configuration review, it scaffolds a configuration-review spec for this engine version and opens one action request per capability — adopt, decline or defer, with the recommendation and the day-to-day effect — addressed to the maintainer role. Nothing is adopted until the maintainer answers; an answered request is applied with `pose adopt --request <act-id> --apply` or by `pose setup`, and the decision records the request it came from.

### Business value

The maintainer asked that versions introducing features make `pose update` suggest decisions on them, always confirmed with the user. Action requests are POSE's confirmation channel: they appear in Attention for the maintainer, carry a proof when the answer is signed, and leave a record of who decided what. Part of roadmap pose-v7-onboarding-and-consolidation (milestone entry-and-update).

### Constraints

Idempotent: one review spec per engine version, never duplicated, never re-asking a capability that has an open request or a recorded decision. An update with nothing new writes nothing. The spec and requests are questions; only an answer changes policy.

### Non-goals

Deciding for the maintainer, or adopting the catalog's defaults in an existing instance.

## 2. Requirements

### Functional

- R1: `pose update` shall, when capabilities are new, create `.pose/specs/<date>-pose-configuration-review-<version>.md` listing each with its effect, introduction version and recommendation, and open one decision request per capability (options adopt, decline, defer; recommended adopt when the catalog turns it on in new instances), restricting the review spec's closeout, addressed to the `maintainer` role.
- R2: A second update for the same version, or a capability with an open request or a recorded decision, shall create nothing new.
- R3: `pose adopt --request <act-id> [--apply]` shall apply an answered review request: adopt turns the capability on, decline and defer record the decision with the answer's reason and the request id; an open, cancelled or foreign request shall be refused.
- R4: `pose setup` shall list review requests waiting for an answer and answered ones not yet applied, and at a terminal apply an answered one after confirmation.

### Non-functional

- None.

### Security

- An answer applies policy only through the request's authority rules, so the review inherits role checks and signature verification.

### Compatibility

- Existing instances only see a review when an update brings something undecided.

## 3. Technical Plan

### Affected areas

Update, a review spec template, action requests opened by the engine, `pose adopt --request`, setup.

### Artifacts

- created: .pose/specs/2026-10-05-pose-update-configuration-review.md
- created: .pose/starts/pose-update-configuration-review.json
- created: pose-mcp/internal/cli/configuration_review.go
- created: pose-mcp/internal/cli/configuration_review_test.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/adopt.go
- modified: pose-mcp/internal/cli/setup.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- renamed: .pose/changelogs/unreleased/pose-update-configuration-review.md -> .pose/changelogs/v7.0.0/pose-update-configuration-review.md

### Delivery targets

- capability:configuration-review module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- An update writing a spec could break the instance's strict check; the test runs the check after the update.

## 6. Validation

### Strategy

An older instance updated: the review spec and one request per new capability appear, a second update adds nothing, answers are applied through `pose adopt --request`, and the strict check passes throughout.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run 'ConfigurationReview'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestConfigurationReviewAsksOncePerCapabilityAndChangesNothing check:configuration-review-integration
- R2 [satisfied] test:TestConfigurationReviewAsksOncePerCapabilityAndChangesNothing check:configuration-review-integration
- R3 [satisfied] test:TestConfigurationReviewAppliesOnlyAnsweredRequests check:configuration-review-integration
- R4 [satisfied] test:TestConfigurationReviewAppliesOnlyAnsweredRequests test:TestSetupAnswersAndAppliesAReviewRequestAtATerminal check:configuration-review-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

The update writes the review spec (localized) and one decision request per undecided capability, idempotently; `pose adopt --request` applies answered requests with their reason and id; `pose setup` lists open and answered requests and, at a terminal, answers as the maintainer (signing under verified assurance) and applies.

### Residual risks

None beyond the technical risk.

### Follow-ups
