---
slug: pose-adopt-request-keeps-the-reason
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
delivers: governance:adopt-request-keeps-the-reason
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
- created: .pose/starts/pose-adopt-request-keeps-the-reason.json
- created: .pose/changelogs/unreleased/pose-adopt-request-keeps-the-reason.md
- modified: pose-mcp/internal/cli/configuration_review.go
- modified: pose-mcp/internal/cli/configuration_review_test.go
- modified: pose-mcp/internal/cli/adopt.go
- modified: pose-mcp/internal/cli/help_catalog.go

### Delivery targets

- governance:adopt-request-keeps-the-reason module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Context: `pose action resolve` already accepts `--reason`, stored in the answer event and read back by `adopt --request`; `adopt --list` already shows a decision's reason. The gap was the fallback: an answer without a reason was recorded as "answered decline in act-…".
- Decision: refuse to record a decline or deferral without a reason, and let `adopt --request` take `--reason` when the answer has none.
- Rationale: R1 and R3 already held; the placeholder is what let the reasons of audio-relay and storageclose go unrecorded on 2026-10-09, which the independent closeout review found on 2026-10-10.

## 6. Validation

### Strategy

Decline through a request with a rationale; `adoption-decisions.json` and `adopt --list` show it; a decline through a request with no reason is refused.

### Requirement trace

- R1 [satisfied] governance:adopt-request-keeps-the-reason evidence:integration test:TestConfigurationReviewAppliesOnlyAnsweredRequests
- R2 [satisfied] governance:adopt-request-keeps-the-reason evidence:integration test:TestDeclinedRequestNeedsAReason
- R3 [satisfied] governance:adopt-request-keeps-the-reason evidence:integration test:TestDeclinedRequestNeedsAReason

## 7. Final Report

### Delivered scope

A declined or deferred configuration answer is recorded only with a reason: the answer's own, or `adopt --request … --reason`.

### Residual risks

None yet.

### Follow-ups
