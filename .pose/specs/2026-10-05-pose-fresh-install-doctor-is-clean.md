---
slug: pose-fresh-install-doctor-is-clean
status: done
created_at: 2026-10-05
completed_at: 2026-10-06
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:fresh-install-health
---

# Spec: A fresh install reports no warning, and next steps are not warnings

## 1. Intent

### Goal

Make the first `pose doctor` and the first `pose state --attention` of a freshly installed project describe a healthy instance: warnings only for real problems, recommended next steps reported as next steps, and sources the project does not use never presented as gaps.

### Business value

The 7.0.0 walk of a fresh install opened `doctor` with four warnings — an uninstalled pre-commit hook, nobody in the action roles (caused by the new default adoption), a milestone profile demanding evidence in a project with no roadmap, and fourteen shipped stack checks without an evidence class for stacks the project does not have — and Attention with "coverage=INCOMPLETE" for four sources the project does not use. A first impression of red flags trains people to ignore the tool. Part of roadmap pose-v7-onboarding-and-consolidation (milestone entry-and-update).

### Constraints

Nothing true is hidden: a real gap stays a warning, an engine limitation stays visible where it applies, and the JSON keeps every finding. The obligation coverage states fixed by the obligations ADR are unchanged.

### Non-goals

Projecting the four pending Attention sources; that is engine work tracked separately.

## 2. Requirements

### Functional

- R1: `pose doctor` shall support a `next` level for a recommended step that is not a problem: it is listed, counted apart from warnings and errors, and never changes the exit code; the JSON report shall carry the count.
- R2: An uninstalled pre-commit hook, and agency readiness with no principal in any action role, shall be reported as `next` with the command that completes them.
- R3: The class-producers check shall consider milestone and roadmap profiles only when the project has a roadmap.
- R4: The shipped stack catalog shall declare the evidence class of every test and build check, and the evidence-class coverage check shall consider only required checks of stacks the project uses.
- R5: Attention shall mark a pending source the project does not use (no docs manifest, no configured release, no capability assessment, no legacy review or investigation) as not used, and shall not call coverage incomplete because of it; a pending source the project does use keeps coverage incomplete.
- R6: A fresh install shall report no warning and no error in `pose doctor`, and, once committed, complete coverage in Attention; before the first commit Attention keeps saying that no revision binds the answer.

### Non-functional

- No new reads beyond file existence checks.

### Security

- None.

### Compatibility

- `doctor_schema_version` rises to 2 for the new level; consumers of version 1 see one more level value.

## 3. Technical Plan

### Affected areas

Doctor levels and three checks, the shipped stack catalog, obligation coverage, Attention rendering.

### Artifacts

- created: .pose/specs/2026-10-05-pose-fresh-install-doctor-is-clean.md
- created: .pose/starts/pose-fresh-install-doctor-is-clean.json
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/check.go
- created: pose-mcp/internal/cli/fresh_install_health_test.go
- modified: pose-mcp/internal/cli/adopt_test.go
- modified: pose-mcp/internal/cli/adversarial_corpus_test.go
- modified: pose-mcp/internal/cli/doctor_remediation_test.go
- modified: pose-mcp/internal/pose/readiness_phases_test.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/attention.go
- modified: pose-mcp/internal/pose/readiness_phases.go
- modified: pose-mcp/internal/cli/state_attention.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/changelogs/unreleased/pose-fresh-install-doctor-is-clean.md

### Delivery targets

- capability:fresh-install-health module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A consumer that treated any unknown doctor level as an error would see `next` as one; the schema version says the vocabulary changed.

## 6. Validation

### Strategy

A fresh install read through `doctor --json` and `state --attention`; the same instance after a docs manifest is added; the shipped stack catalog read for classes.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli ./internal/pose -run 'FreshInstallHealth|ShippedStack|APhaseWithAnUnreadProducer'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestFreshInstallHealthDoctorReportsNoWarning test:TestDoctorFindingsHaveEvidenceAndRemediationClass check:fresh-install-health-integration
- R2 [satisfied] test:TestFreshInstallHealthDoctorReportsNoWarning test:TestDoctorWarnsWhenAgencyReadinessHasNoPrincipal check:fresh-install-health-integration
- R3 [satisfied] test:TestFreshInstallHealthRoadmapProfilesNeedARoadmap check:fresh-install-health-integration
- R4 [satisfied] test:TestShippedStackChecksDeclareTheirEvidenceClass test:TestFreshInstallHealthDoctorReportsNoWarning check:fresh-install-health-integration
- R5 [satisfied] test:TestFreshInstallHealthAttentionCoverageIsComplete test:TestAPhaseWithAnUnreadProducerIsUnknownNotClear check:fresh-install-health-integration
- R6 [satisfied] test:TestFreshInstallHealthDoctorReportsNoWarning test:TestFreshInstallHealthAttentionCoverageIsComplete check:fresh-install-health-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

Doctor `next` level (schema 2, `next_steps`), hook and action roles as next steps, roadmap-scoped class producers, stack-scoped evidence-class coverage with classes declared on every shipped test and build check, and `not_used` sources in Attention and phase readiness. A committed fresh install reads clean in both.

### Residual risks

None beyond the technical risk.

### Follow-ups

- [open] Project the four pending Attention sources (release queues, docs review pendencies, capability stale triggers, open findings outside attestations) so a project that uses them gets complete coverage. (owner:@pose-maintainers crit:medium review:2026-11-15)
