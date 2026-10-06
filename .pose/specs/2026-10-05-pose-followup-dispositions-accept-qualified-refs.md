---
slug: pose-followup-dispositions-accept-qualified-refs
status: done
created_at: 2026-10-05
completed_at: 2026-10-06
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: spec:pose-qualified-artifact-resolution@defect-fix
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:qualified-followup-dispositions
---

# Spec: A follow-up can be covered by a spec in another project

## 1. Intent

### Goal

Accept `xref:<project>/spec:<slug>` as the target of a `covered`, `duplicate` or `spawned` follow-up disposition, verified against the other project when it is bound and accepted when it is not.

### Business value

Decision 5 of the backlog reconciliation confirmed that two pose-abm-review-authority follow-ups are covered by `harne8-action-request-confirmation-channel`, a Harne8 spec. Qualified references resolve everywhere else in POSE — dependencies, targets, roadmap members — but `lint-spec` reads a disposition target only as a local slug and refuses the qualified one as "a missing spec", so a follow-up owned by another project can only stay falsely open.

### Constraints

A target that can be checked is checked: when the other project is bound and the spec does not exist there, the disposition is refused. An unbound project is not evidence that the spec is missing, so it is accepted, as an unchecked qualified dependency would be reported rather than invented. Local targets keep today's rule.

### Non-goals

Reading the target's content for the covered-anchor warning across projects.

## 2. Requirements

### Functional

- R1: A `covered`, `duplicate` or `spawned` follow-up whose target is a well-formed `xref:<project>/spec:<slug>` shall pass `lint-spec` when the project is not bound in this environment.
- R2: When the project is bound, the disposition shall be refused if the spec does not exist there and accepted if it does.
- R3: A malformed qualified target, or one naming a non-spec artifact, shall be refused with the reason.
- R4: The manual shall state that a disposition target may be qualified.

### Non-functional

- No resolution is attempted for a local target.

### Security

- None beyond the shared constraints.

### Compatibility

- Additive: every disposition valid before stays valid.

## 3. Technical Plan

### Affected areas

Follow-up disposition lint, manuals.

### Artifacts

- created: .pose/specs/2026-10-05-pose-followup-dispositions-accept-qualified-refs.md
- created: .pose/starts/pose-followup-dispositions-accept-qualified-refs.json
- modified: pose-mcp/internal/cli/lintspec.go
- created: pose-mcp/internal/cli/lintspec_qualified_followup_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- renamed: .pose/changelogs/unreleased/pose-followup-dispositions-accept-qualified-refs.md -> .pose/changelogs/v7.0.0/pose-followup-dispositions-accept-qualified-refs.md

### Delivery targets

- capability:qualified-followup-dispositions module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A follow-up covered by an unbound project is accepted unverified; it is verified wherever both projects are bound.

## 6. Validation

### Strategy

Two fixture projects, one with the covering spec; lint a done spec whose follow-up names it qualified, with and without the binding, plus a missing spec, a malformed target and a roadmap target.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run QualifiedFollowup`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestQualifiedFollowupIsAcceptedWhenTheProjectIsNotBound check:qualified-followup-integration
- R2 [satisfied] test:TestQualifiedFollowupIsVerifiedWhenTheProjectIsBound check:qualified-followup-integration
- R3 [satisfied] test:TestQualifiedFollowupRefusesMalformedTargets check:qualified-followup-integration
- R4 [satisfied] test:TestQualifiedFollowupIsDocumented check:qualified-followup-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

The unverified acceptance for an unbound project, above.

### Follow-ups

None.
