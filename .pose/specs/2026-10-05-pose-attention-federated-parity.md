---
slug: pose-attention-federated-parity
status: in-progress
created_at: 2026-10-05
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: spec:pose-state-attention@defect-fix
priority: 1
components: pose-mcp
surface: minimal
task_type: bugfix
delivers: surface:state-attention
---

# Spec: Attention reads federated obligations on the CLI as MCP does

## 1. Intent

### Goal

`pose state --attention` shall project the same obligations as `pose_obligations` for the same snapshot, including federated acceptance blockers.

### Business value

The Harne8 attention contract test (harne8-attention-surface R5) found 25 `federated-acceptance-blocker` obligations that MCP reported and the CLI did not, for the same snapshot digest: the CLI built a plain store without the federated resolver, so a person reading the CLI saw fewer restrictions than the platform.

### Constraints

Same domain call on both surfaces; no change to what MCP returns.

### Non-goals

Changing which producers exist.

## 2. Requirements

### Functional

- R1: `pose state --attention` shall read obligations through the governed store (federated resolver and project identity) used by `pose close` and MCP, so CLI and MCP return the same obligation IDs for the same snapshot.

### Non-functional

- None beyond the shared constraints.

### Security

- Federated reads stay within the declared project roots; nothing new is read.

### Compatibility

- Output shape unchanged; a federated project may now see restrictions it previously missed.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/cli/state_attention.go`.

### Artifacts

- created: .pose/specs/2026-10-05-pose-attention-federated-parity.md
- created: .pose/changelogs/unreleased/pose-attention-federated-parity.md
- modified: pose-mcp/internal/cli/state_attention.go
- modified: pose-mcp/internal/cli/state_attention_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- surface:state-attention module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- None beyond reading the same federated sources MCP already reads.

## 6. Validation

### Strategy

A federated fixture where a consumer milestone depends on a stale source: the CLI must report the blocker MCP reports.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run StateAttention`
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:state-attention evidence:integration <the CLI reads through the governed store; the Harne8 attention contract test compares CLI and MCP on one snapshot> test:TestStateAttentionReadsThroughTheGovernedStore check:state-attention-integration check:state-attention-reachability

### Known gaps

None.

## 7. Final Report

### Delivered scope

`pose state --attention` reads obligations through `cliGovernedStore`, the store `pose close` and MCP use, so federated acceptance blockers reach the CLI.

### Residual risks

None.

### Follow-ups

