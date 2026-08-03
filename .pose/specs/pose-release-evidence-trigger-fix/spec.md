---
slug: pose-release-evidence-trigger-fix
status: in-progress
created_at: 2026-08-03
completed_at:
supersedes:
depends_on: pose-release-lifecycle-closure
priority: 0
components: release-workflows
delivers: governance:release-integrity
---

# Spec: Release evidence trigger fix

## 1. Intent

### Goal
Make retained publication evidence describe only public assets and guarantee an
independent verifier runs after a tag-triggered release.

### Business value
The first governed cut, v0.16.0, proved that a release created with the default
Actions token does not emit a downstream `release: published` workflow event.
It also showed that evidence generated from the build directory includes
internal files not present in the public release. Both produce false confidence.

### Constraints
- Preserve v0.16.0 and its honest published-but-unverified state.
- Use least-privilege read permissions in the independent job.
- Query provider-visible assets rather than producer workspace contents.

### Non-goals
- Rewrite v0.16.0 or overwrite its retained evidence asset.

## 2. Requirements

### Functional
- R1: Publication evidence shall list only assets returned by the provider release API.
- R2: Successful tag-triggered Release workflows shall trigger Verify release through workflow_run even when publication uses GITHUB_TOKEN.
- R3: The verifier shall check out the exact tag and run only for successful v-prefixed Release runs.

### Non-functional
- Keep the fix limited to the two workflow contracts.

### Security
- Grant the verifier only contents and attestations read permissions.

### Compatibility
- Preserve manual and release-published verification triggers.

## 3. Technical Plan

### Affected areas
- Release publication and independent verification workflows.

### Artifacts
- modified: .github/workflows/release.yml
- modified: .github/workflows/verify-release.yml
- modified: .pose/specs/pose-release-evidence-trigger-fix/spec.md

### Delivery targets
- governance:release-integrity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- Add a workflow_run trigger and provider-API asset enumeration.

### Data/storage changes
- None.

### Technical risks
- Duplicate triggers are possible only for releases published with a token that
  also emits release events; jobs are read-only and idempotent.

## 4. Tasks

### Planning
- [x] Reproduce both v0.16.0 gaps from public API/workflow evidence.

### Implementation
- [x] Query public release assets for evidence.
- [x] Add successful tag workflow_run verification trigger.

### Validation
- [ ] Pass workflow security and observe v0.16.1 independent verification.

## 5. Decisions

### Decision 1
- Date: 2026-08-03
- Context: GITHUB_TOKEN-originated release events do not recursively trigger workflows.
- Options considered: PAT; repository dispatch; workflow_run.
- Decision: Use workflow_run with a separate least-privilege verifier job.
- Rationale: No long-lived credential is introduced.
- Consequences: The verifier derives and checks out the completed run's tag.

## 6. Validation

### Strategy
Cut v0.16.1 and require both Release and Verify release runs to succeed; compare
retained evidence asset names with the provider's public asset set.

### Deterministic checks
- Test: `go -C pose-mcp test ./internal/version -run WorkflowSecurity -count=1`.
- Structure: `pose check --strict`.

### Execution log
- 2026-08-03: v0.16.0 Release succeeded; no Verify release run existed and retained evidence contained producer-only files.

### Results summary
- Successes: root causes reproduced and minimal workflow fix applied.
- Failures: final public verification pending v0.16.1.
- Warnings: v0.16.0 remains intentionally unverified.

### Requirement trace
- R1 [satisfied] report:.pose/reports/2026-08-03-release-evidence-trigger-fix-review.md.
- R2 [satisfied] check:workflow-security.
- R3 [satisfied] check:workflow-security.

### Known gaps
- None after the v0.16.1 verification run succeeds.

## 7. Final Report

### Delivered scope
Provider-visible evidence enumeration and reliable independent verification trigger.

### Files and modules changed
- `.github/workflows/release.yml`, `.github/workflows/verify-release.yml`.

### Validation executed
- Workflow security, strict POSE and live v0.16.1 runs.

### Residual risks
- Duplicate read-only verification runs may occur with non-default provider tokens.

### Follow-ups
No follow-up.
