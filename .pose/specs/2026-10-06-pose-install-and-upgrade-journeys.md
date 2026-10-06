---
slug: pose-install-and-upgrade-journeys
status: in-progress
created_at: 2026-10-06
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:install-and-upgrade-journeys
---

# Spec: A fresh install and an upgrade from the last release are walked in CI

## 1. Intent

### Goal

Walk, in CI, the two ways a project meets this engine — a fresh install, and an instance installed by the latest published release and then updated — asserting what onboarding promises: a strict check that passes, a doctor with no warning, one next step, a configuration review that asks once per new capability, and a signed answer applied with `pose adopt --request`. Fix what walking them exposed.

### Business value

Every onboarding change so far was tested on fixtures built by the current engine. The release that matters is the one users already have: an upgrade from it must read clean and must not leave a review addressed to a role nobody holds. Part of roadmap pose-v7-onboarding-and-consolidation (milestone guided-onboarding).

### Constraints

The previous release is downloaded from its published assets and checked against its `checksums.txt`; the journeys never use the machine's keys or git configuration.

### Non-goals

Upgrades from releases older than the latest published one.

## 2. Requirements

### Functional

- R1: `tests/journeys/install-and-upgrade.sh` shall walk a fresh install (strict check, clean doctor, the onboarding spec as the next step, nothing to review) and an upgrade from the latest published release (strict check, clean doctor naming the pending decisions, one review request per new capability, a second update asking nothing, a signed answer applied with `pose adopt --request`); CI and `scripts/verify.sh` shall run it.
- R2: `pose setup` shall make registering a maintainer a step to do when nobody holds a role and an open request is addressed to a role, not only when agency readiness is on.
- R3: `pose doctor` shall name an available rule extension by its catalog id (`pose extension install <id>`), the command that resolves it, instead of a path placeholder.

### Non-functional

- None.

### Security

- The downloaded release is verified against its published checksum before it runs.

### Compatibility

- None.

## 3. Technical Plan

### Affected areas

A journey script and its CI step, setup's identity step, doctor's rule-extension hint.

### Artifacts

- created: .pose/specs/2026-10-06-pose-install-and-upgrade-journeys.md
- created: .pose/starts/pose-install-and-upgrade-journeys.json
- created: tests/journeys/install-and-upgrade.sh
- created: pose-mcp/internal/cli/journeys_script_test.go
- modified: pose-mcp/internal/cli/setup.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: .github/workflows/ci.yml
- modified: scripts/verify.sh
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-install-and-upgrade-journeys.md

### Delivery targets

- capability:install-and-upgrade-journeys module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- The journey needs the network to fetch the previous release; the Go test runs only with `POSE_JOURNEYS=1`, which the validation matrix and CI provide.

## 6. Validation

### Strategy

The journey script against the latest published release.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && POSE_JOURNEYS=1 go test ./internal/cli -run Journeys`
- Expected: pass

### Requirement trace

### Known gaps

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None beyond the technical risk.

### Follow-ups
