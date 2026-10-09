---
slug: pose-adopt-request-keeps-the-reason
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

# Spec: A declined or deferred configuration answer keeps its reason

## 1. Intent

### Goal

Record why a capability was declined or deferred when the decision arrives as an answer to a configuration-review request.

### Business value

`pose adopt <cap> --decline|--defer --reason <text>` records a reason in `adoption-decisions.json`, but `pose adopt --request <act-id> --apply` refuses `--reason`, and `pose action resolve` carries only the option id. On 2026-10-09, Harne8 deferred `signed-attestations` and `verified-identity` until a native issuer exists, and audio-relay and storageclose declined federation because they are single repositories; those reasons reached only spec prose, not the decision record the engine reads back.

## 2. Requirements

### Functional

- R1: `pose action resolve` shall accept an optional free-text rationale stored in the answer event and covered by the answer's digest.
- R2: `pose adopt --request <act-id> --apply` shall write that rationale as the decision's reason, or accept `--reason` when the answer carries none; a decline or defer without any reason shall be refused, as the direct path does.
- R3: `pose adopt --list` shall show the reason of a declined or deferred capability whichever path recorded it.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-adopt-request-keeps-the-reason.md

Implementation artifacts are declared when the spec starts.

## 5. Decisions

No decision recorded yet; the spec is a draft.

## 6. Validation

### Strategy

Decline through a request with a rationale; `adoption-decisions.json` and `adopt --list` show it; a decline through a request with no reason is refused.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-09 from the POSE 7.1.0 adoption in Harne8, pose-dist, audio-relay and storageclose.

### Residual risks

None yet.

### Follow-ups
