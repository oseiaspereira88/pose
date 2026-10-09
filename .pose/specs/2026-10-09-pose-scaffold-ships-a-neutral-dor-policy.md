---
slug: pose-scaffold-ships-a-neutral-dor-policy
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
changelog: none
delivers: governance:scaffold-ships-a-neutral-dor-policy
---

# Spec: The scaffold ships a neutral Definition of Ready policy

## 1. Intent

### Goal

Keep pose-dist's own Definition of Ready adoption out of the scaffold every new instance is installed from.

### Business value

On 2026-10-09 the maintainer adopted the Definition of Ready in pose-dist (`7eeca305`, configuration review of 7.1.0). `.pose/policy/dor.json` was still synced verbatim into the embedded scaffold, so the drift guard turned CI and Security red on main (red signals #136 and #137), and a regenerated scaffold would have shipped `adopted_at: 2026-10-09` to every new instance, gating its specs from another repository's date. `changelog.json`, `review.json` and `actions.json` had already been fixed the same way.

### Constraints

New instances keep the same `dor.json` they received before: `adopted_at` empty and the same task types.

## 2. Requirements

### Functional

- R1: `.pose/policy/dor.json` shall be a self-referential policy file, excluded from the verbatim sync and shipped from a neutral template with `adopted_at` empty.
- R2: The embedded scaffold and its drift guard shall agree, so `go test ./internal/scaffold/...` passes with pose-dist's own DoR adopted.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-scaffold-ships-a-neutral-dor-policy.md
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go

### Delivery targets

- governance:scaffold-ships-a-neutral-dor-policy module:pose-mcp profile:release-governance entrypoint:pose-mcp/internal/scaffold/distpolicy/distpolicy.go

## 5. Decisions

No material decision: the file joins the existing self-referential list, as `changelog.json`, `review.json` and `actions.json` did.

## 6. Validation

### Strategy

`TestEmbeddedDistMatchesPoseDist` failed on main from `7eeca305`; after the change it passes, and `TestNeutralDoRPolicyShipsTheGateOff` asserts the template ships `adopted_at` empty.

### Requirement trace

- R1 [satisfied] governance:scaffold-ships-a-neutral-dor-policy evidence:unit test:TestNeutralDoRPolicyShipsTheGateOff test:TestSelfReferentialPolicyFilesExcluded
- R2 [satisfied] governance:scaffold-ships-a-neutral-dor-policy evidence:unit test:TestEmbeddedDistMatchesPoseDist

## 7. Final Report

### Delivered scope

`dor.json` is shipped from a neutral template; pose-dist keeps its own adoption.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
