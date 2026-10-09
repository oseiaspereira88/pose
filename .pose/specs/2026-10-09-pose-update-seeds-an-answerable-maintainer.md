---
slug: pose-update-seeds-an-answerable-maintainer
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers:
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

Implementation artifacts are declared when the spec starts.

## 5. Decisions

No decision recorded yet; the spec is a draft.

## 6. Validation

### Strategy

Fixture without `actions.json`: update, then resolve; the messages name the empty role; after `identity add --role maintainer --apply` the resolve succeeds.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-09 from the POSE 7.1.0 adoption in Harne8, pose-dist, audio-relay and storageclose.

### Residual risks

None yet.

### Follow-ups
