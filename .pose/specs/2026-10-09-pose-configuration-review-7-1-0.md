---
slug: pose-configuration-review-7-1-0
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
remediates:
priority: 1
components:
task_type: feature
surface: minimal
delivers:
changelog: none
---

# Spec: Review the capabilities POSE 7.1.0 brings

## 1. Intent

### Goal

Decide, for each capability this project has not decided under POSE 7.1.0, whether to adopt, decline or defer it. Nothing changes until the maintainer answers.

### Business value

An update brings capabilities; deciding each one explicitly keeps none in limbo and records why it is on or off.

### Constraints

Each decision is an action request to the `maintainer` role (`pose action list`, `pose setup`). An answer is applied with `pose adopt --request <act-id> --apply` or by `pose setup`; adopting dates the capability with the day it is applied, so earlier work is not re-judged.

### Non-goals

Changing anything the maintainer did not answer.

## 2. Requirements

### Functional

- R1: The maintainer shall adopt, decline or defer `definition-of-ready` — a spec created on or after the date cannot move to in-progress until Intent, Requirements with stable ids and Technical Plan are filled (since 4.0.0; recommended: adopt, as new instances do).
- R2: The maintainer shall adopt, decline or defer `overlay:engineering-judgment` — a scope selected by its declared risk is also reviewed for engineering judgment (since 6.0.0; no recommendation).
- R3: The maintainer shall adopt, decline or defer `overlay:high-criticality-review` — a scope touching a critical component carries the stricter review its criticality asks for (since 6.0.0; no recommendation).

### Non-functional

- None.

### Security

- None.

### Compatibility

- None.

## 3. Technical Plan

### Affected areas

POSE policy only.

### Artifacts

- created: .pose/specs/2026-10-09-pose-configuration-review-7-1-0.md
- created: .pose/actions/act-2299f4cbbb16aa92.jsonl
- created: .pose/actions/act-2af5a58184fbd577.jsonl
- created: .pose/actions/act-8baf0b912e8f8404.jsonl
- created: .pose/policy/adoption-decisions.json
- modified: .pose/policy/dor.json
- modified: .pose/policy/review.json
- modified: .pose/state/machinery-manifest.json

### Technical risks

- None.

## 6. Validation

### Strategy

`pose setup` lists no capability decision pending.

### Deterministic checks

#### Health
- Command: `pose doctor`
- Expected: no `setup.capabilities` step

### Requirement trace

- R1 [satisfied] evidence:manual <definition-of-ready adotada pelo maintainer em act-8baf0b912e8f8404>
- R2 [satisfied] evidence:manual <overlay:engineering-judgment adotada pelo maintainer em act-2299f4cbbb16aa92>
- R3 [satisfied] evidence:manual <overlay:high-criticality-review adotada pelo maintainer em act-2af5a58184fbd577>

### Known gaps

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

### Follow-ups
