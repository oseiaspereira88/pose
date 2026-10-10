---
slug: pose-update-seeds-an-answerable-maintainer
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers: governance:update-seeds-an-answerable-maintainer
---

# Spec: An update opens only requests someone can answer

## 1. Intent

### Goal

When `pose update` creates `.pose/policy/actions.json` and opens configuration-review requests to the `maintainer` role, the project shall have a way to name its maintainer as part of that step, and answering an unanswerable request shall say why.

### Business value

On 2026-10-09 the update created `actions.json` with `"maintainer": []` in audio-relay and storageclose and opened six requests to that empty role. `pose action resolve` then failed with `action-actor-not-authorized: actor must be an agent: or human: principal`, which reads as a malformed actor rather than "no one holds the maintainer role". The policy had to be edited by hand.

### Constraints

The engine must not invent a principal: a name from git is a suggestion, confirmed by preview and apply, as `pose identity add` already does.

## 2. Requirements

### Functional

- R1: When the update opens requests to a role that no principal holds, it shall say so and print the command that names one (for example `pose identity add human:<name> --role maintainer`, with the git identity as the suggestion).
- R2: `pose setup` shall offer naming the maintainer before the capability decisions when the role is empty.
- R3: `pose action resolve` for a role with no principal shall fail with an error naming the empty role and the command above.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-update-seeds-an-answerable-maintainer.md
- created: .pose/starts/pose-update-seeds-an-answerable-maintainer.json
- created: .pose/changelogs/unreleased/pose-update-seeds-an-answerable-maintainer.md
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/configuration_review_test.go
- modified: pose-mcp/internal/cli/setup.go
- modified: pose-mcp/internal/cli/setup_test.go

### Delivery targets

- governance:update-seeds-an-answerable-maintainer module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Context: measured before implementing. The 2026-10-09 error `actor must be an agent: or human: principal` came from the implementer's script, which passed `null` read from the empty role list as the actor; with a valid actor the engine already answered "<actor> does not hold role maintainer". `pose setup` already offers naming the maintainer (`identity.maintainer`) when nobody holds a role.
- Decision: keep R2 as delivered by `setup`; make the update warn when it opens requests to a role nobody holds, and make the resolve error name the empty role and `pose setup` instead of the actor.
- Rationale: the remaining gap is that nothing said nobody could answer.

## 6. Validation

### Strategy

Fixture without `actions.json`: update, then resolve; the messages name the empty role; after `identity add --role maintainer --apply` the resolve succeeds.

### Execution log

2026-10-10, independent review (agent:independent-gpt-6.1-sol-review): medium severity. `setup` marked `identity.maintainer` done for a principal with a registered key holding only `reviewer`, while nobody held `maintainer`. The step now asks for a maintainer whenever that role is empty under agency readiness or open role requests; `TestSetupWantsAMaintainerEvenWhenYouHoldAnotherRole` fails on the previous step.

### Requirement trace

- R1 [satisfied] governance:update-seeds-an-answerable-maintainer evidence:integration test:TestUpdateNamesAnUnansweredMaintainerRole
- R2 [satisfied] governance:update-seeds-an-answerable-maintainer evidence:integration test:TestSetupOnAFreshInstallNamesTheNextStep test:TestSetupWantsAMaintainerEvenWhenYouHoldAnotherRole
- R3 [satisfied] governance:update-seeds-an-answerable-maintainer evidence:integration test:TestUpdateNamesAnUnansweredMaintainerRole

## 7. Final Report

### Delivered scope

An update that opens requests to a role nobody holds warns and names `pose setup`; answering one says nobody holds the role instead of blaming the actor.

### Residual risks

None yet.

### Follow-ups
