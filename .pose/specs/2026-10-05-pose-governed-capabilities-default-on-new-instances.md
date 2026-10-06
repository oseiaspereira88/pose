---
slug: pose-governed-capabilities-default-on-new-instances
status: in-progress
created_at: 2026-10-05
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:governed-capability-toggle
---

# Spec: A new instance adopts the governed capabilities; an existing one toggles them

## 1. Intent

### Goal

Ship agency readiness, contract nodes, atomic start and causality closeout (with the structural overlay) adopted in every new instance, dated the day it was installed, and give an existing instance one command to turn each on or off with its adoption date instead of editing the review policy by hand.

### Business value

The maintainer decided, while resolving the adoption requests of pose-abm-capability-adoption, that these capabilities are the product's default and not a per-repository experiment: a new project should start governed by them, while a project already under way decides when, so nothing it has in flight is re-judged. Today the distributed review policy adopts none of them, so the decisions recorded for pose-dist reach no other project, and adopting one by hand means knowing which version key, date key and overlay entry go together.

### Constraints

Install adopts only when the review policy did not exist before that install; `pose update` never adopts anything, including when it seeds a missing policy. Every date is the instance's own. The distributed policy file stays neutral: the adoption is an install-time decision that is logged, not a file that carries another repository's choices. Turning a capability off is as explicit as turning it on.

### Non-goals

Naming who answers action requests in a new instance; the shipped role map stays empty and is the project's to fill.

## 2. Requirements

### Functional

- R1: When `pose install` creates the review policy, it shall adopt agency readiness, contract nodes, atomic start and causality closeout with the structural-materiality overlay, dating atomic start, causality closeout and the overlay with the install day, and shall log each adoption.
- R2: `pose install` over an existing review policy, and `pose update` in every case, shall adopt nothing.
- R3: `pose adopt <capability>` shall preview the review-policy keys it would set, and with `--apply` write them, dated today or `--date YYYY-MM-DD`; `--off` shall preview and remove the same keys; the result shall be refused when the policy reader would reject it; an unknown capability shall be refused with the known ones listed.
- R4: Effective governance shall name the `pose adopt` command for a supported capability that is not adopted.
- R5: `pose doctor` shall warn when agency readiness is adopted and no principal holds any role in the actions policy.
- R6: The manual and the command help shall describe the default adoption and the toggle.

### Non-functional

- Install and adopt write the review policy atomically.

### Security

- No principal is ever written by install or adopt.

### Compatibility

- Existing instances are unchanged until they run `pose adopt`.

## 3. Technical Plan

### Affected areas

Capability registry in the engine, install, a new `adopt` command, doctor, effective governance, help, manuals.

### Artifacts

- created: .pose/specs/2026-10-05-pose-governed-capabilities-default-on-new-instances.md
- created: .pose/starts/pose-governed-capabilities-default-on-new-instances.json
- created: pose-mcp/internal/pose/governed_capabilities.go
- created: pose-mcp/internal/pose/governed_capabilities_test.go
- modified: pose-mcp/internal/pose/effective_governance.go
- created: pose-mcp/internal/cli/adopt.go
- created: pose-mcp/internal/cli/adopt_test.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/usage.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/progressive_review_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- renamed: .pose/changelogs/unreleased/pose-governed-capabilities-default-on-new-instances.md -> .pose/changelogs/v7.0.0/pose-governed-capabilities-default-on-new-instances.md

### Delivery targets

- capability:governed-capability-toggle module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A new instance adopts agency readiness with an empty role map: a request addressed to a role cannot be answered until the project names a principal. `pose doctor` says so, and a request addressed to a principal is unaffected.

## 6. Validation

### Strategy

Install into empty and pre-configured targets, update over a target without a review policy, and drive `pose adopt` on and off through the reader, asserting the written keys and the refusals.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose ./internal/cli -run 'GovernedCapabilit|Adopt'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestGovernedCapabilitiesAreAdoptedByAFreshInstall check:governed-capabilities-integration
- R2 [satisfied] test:TestGovernedCapabilitiesAreNeverAdoptedOverAnExistingPolicy check:governed-capabilities-integration
- R3 [satisfied] test:TestAdoptTogglesACapabilityThroughTheReader test:TestGovernedCapabilitiesAdoptAndRetireThroughTheReader check:governed-capabilities-integration
- R4 [satisfied] test:TestGovernedCapabilitiesNameTheToggleWhenNotAdopted check:governed-capabilities-integration
- R5 [satisfied] test:TestDoctorWarnsWhenAgencyReadinessHasNoPrincipal check:governed-capabilities-integration
- R6 [satisfied] test:TestAdoptIsDocumented check:governed-capabilities-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

The empty role map in a new instance, described above.

### Follow-ups

None.
