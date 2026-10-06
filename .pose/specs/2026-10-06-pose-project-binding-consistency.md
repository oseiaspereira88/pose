---
slug: pose-project-binding-consistency
status: in-progress
created_at: 2026-10-06
completed_at:
depends_on:
remediates: spec:pose-project-identity-file@defect-fix, spec:pose-setup-command@defect-fix
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:project-binding-consistency
---

# Spec: Refuse disagreeing project bindings before setup or governance writes

## 1. Intent

### Goal

Check every explicit binding for the selected root and make setup use that same resolved identity.

### Business value

The release review found that setting POSE_DEFAULT_PROJECT_ID skips conflicting entries in POSE_PROJECT_ROOTS. Setup separately reads only the project file, so it can describe a conflicting configuration as ready. Both violate the identity spec's refusal contract.

### Constraints

Preserve agreeing declarations and environment-only identities. Reuse the existing resolver; no identity migration. Context consulted: knowledge:multirepo-review-continuation.

### Non-goals

Changing the project id grammar, adoption defaults or identity assurance.

## 2. Requirements

### Functional

- R1: Resolution shall refuse any binding for this root that disagrees with the selected environment or file identity, including when POSE_DEFAULT_PROJECT_ID is set; the diagnostic shall name both identities.
- R2: Setup shall resolve identity through the same resolver, refuse conflicts before interactive mutations and report environment-only declared identities correctly.
- R3: Agreeing declarations shall continue to resolve and setup shall remain read-only without input.

### Security

Never attribute a governance write to a root with conflicting explicit declarations.

### Compatibility

Previously conflicting configurations fail with the existing conflicting-project-binding code.

## 3. Technical Plan

### Affected areas

EnvironmentArtifactResolver and buildSetupPlan, with negative and positive regression tests.

### Artifacts

- created: .pose/specs/2026-10-06-pose-project-binding-consistency.md
- created: .pose/starts/pose-project-binding-consistency.json
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/project_file_test.go
- modified: pose-mcp/internal/cli/setup.go
- modified: pose-mcp/internal/cli/setup_test.go
- created: .pose/changelogs/unreleased/pose-project-binding-consistency.md

### Delivery targets

- capability:project-binding-consistency module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

Existing conflicting configurations need their bindings corrected. No schema changes.

## 6. Validation

### Strategy

Reproduce a conflicting roots binding while the default agrees with project.json; reproduce setup describing a conflict as ready. Confirm agreeing and environment-only identities. Run the registered project identity and setup integration checks, then the complete strict matrix.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose ./internal/cli -run 'ProjectFile|Setup' -count=1`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestProjectFileChecksRootsEvenWithAnExplicitDefault check:project-identity-integration
- R2 [satisfied] test:TestSetupRefusesConflictingProjectBindingsBeforePrompting test:TestSetupReportsAnEnvironmentOnlyDeclaredIdentity check:setup-command-integration
- R3 [satisfied] test:TestProjectFileChecksRootsEvenWithAnExplicitDefault test:TestSetupOnAFreshInstallNamesTheNextStep check:setup-command-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

All explicit root bindings are checked even with a selected default. Setup resolves the same identity, reports environment-only declarations and refuses conflicts before prompting. On 2026-10-06 all three new regressions failed before the fix; `go test ./internal/pose ./internal/cli -run 'ProjectFile|Setup' -count=1` passed afterwards.

### Residual risks

None beyond the stated conflict diagnostic.

### Follow-ups
