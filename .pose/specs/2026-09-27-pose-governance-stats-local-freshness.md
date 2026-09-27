---
slug: pose-governance-stats-local-freshness
status: done
created_at: 2026-09-27
supersedes:
depends_on: pose-abm-governance-outcomes
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:governance-stats-local-freshness
completed_at: 2026-09-27
---

# Spec: Governance stats verify freshness locally from every caller

## 1. Intent

### Goal
Make `pose_governance_stats` answer from the MCP server with the same result
and at the same cost as `pose stats governance` from the CLI.

### Business value
Harne8's portfolio shows governance outcomes through pose-mcp. On Harne8 the
CLI answered in about 30 seconds, while the MCP tool ran for more than ten
minutes without answering. The projection verifies each attested bundle's
freshness (161 bundles there). The CLI calls it on a plain local store; the MCP
server calls it on its federated store, where every verification resolves the
qualified dependencies again. The same call could therefore also produce a
different freshness count than the CLI.

### Constraints
The projection stays read-only and local by definition. The report shape does
not change.

### Non-goals
Do not change what counts as fresh, or the review verification itself.

## 2. Requirements

- R1: Governance outcomes verify bundle freshness through a local store on the caller's root, so a federated store never resolves qualified dependencies for the projection.
- R2: The report from a federated store equals the report from a local store on the same root.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/governance_outcomes.go` (`GovernanceOutcomes`).

### Artifacts
- created: .pose/specs/2026-09-27-pose-governance-stats-local-freshness.md
- modified: pose-mcp/internal/pose/governance_outcomes.go
- modified: pose-mcp/internal/pose/governance_outcomes_test.go
- created: .pose/changelogs/unreleased/pose-governance-stats-local-freshness.md

### Delivery targets
- contract:governance-stats-local-freshness module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Verify freshness with `Store{Root: s.Root}` inside `GovernanceOutcomes`.
Rollback is a revert.

## 4. Tasks

- [x] Reproduce the slow MCP answer on Harne8.
- [x] Write the regression and prove it fails without the change.
- [x] Verify freshness through a local store.
- [x] Run the matrix, review and close.

## 5. Decisions

The fix sits in the projection rather than in the MCP handler, so no caller can
reintroduce the federated path by handing it a different store.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| R1/R2 | `go test ./internal/pose -run TestGovernanceOutcomesVerifiesFreshnessLocally` | No qualified resolution; reports equal. |
| Harne8 consumer | `pose_governance_stats` over MCP stdio on Harne8 | Answers in about the CLI's time with the same counts. |

### Execution log
2026-09-27: on Harne8, the MCP call with the pinned `8ad0143` binary did not
answer within ten minutes, and `pose stats governance` answered in 30 s. The
regression resolved qualified dependencies twice before the change and none
after it. A candidate binary answered the MCP call in 30 s with the CLI's
counts (161 freshness checks, 93 stale).

At `fdb742d` the full matrix passed 32/32. Bundle `rvb-348ec0f09e9d11d2` was
approved by attestation recorded by the agent under explicit authorization from
the user to self-attest.

### Requirement trace
- R1 [satisfied] test:TestGovernanceOutcomesVerifiesFreshnessLocally
- R2 [satisfied] test:TestGovernanceOutcomesVerifiesFreshnessLocally

## 7. Final Report

### Delivered scope
Governance outcomes verify freshness locally, so the MCP tool answers like the
CLI.

### Residual risks
The projection still verifies up to 256 bundles per call; its cost grows with
the review history.

### Follow-ups
