---
slug: pose-abm-doctor-state-health
status: in-progress
created_at: 2026-09-19
completed_at:
components: pose-mcp
task_type: bugfix
delivers: surface:doctor-state-health
---

# Spec: Include project-state health in doctor diagnostics

## 1. Intent

Close the diagnostic gap identified by Harne8 instance-health R3: doctor
reported no state error while state rejected broken document pointers.
Use the same parser and pointer resolver without mutating derived artifacts.

## 2. Requirements

- R1: Report broken derived pointers and unreadable or malformed state as
  detectable errors, using the existing state parser and resolver.
- R2: Treat absent optional state as uninitialized, not broken. Distinguish
  pointer validity from staleness, tampering and pending refresh.
- R3: Preserve read-only diagnosis and explicit remediation; doctor must not
  silently refresh state or rewrite historical evidence.
- R4: Expose findings through existing CLI text/JSON and test the real command.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-doctor-state-health.md
- modified: pose-mcp/internal/cli/doctor.go
- created: pose-mcp/internal/cli/doctor_state_health_test.go
- modified: .pose/results/delivery-validation.json

### Delivery targets
- surface:doctor-state-health module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Implementation approach

Extend doctor with additive state diagnostics. Reuse Store.ProjectState and
ValidatePointers; stat distinguishes absence from access errors. No schema
version bump, new dependencies, new fixer or adoption change. State errors
remain detectable, not fixable. Cap displayed broken references at ten while
retaining the total count. Existing diagnostic redaction remains authoritative.
Knowledge consulted: knowledge:module-metadata-discovery-invalidates-review-provenance;
avoid unrelated discovery/index changes before sealing. Revert the isolated
commit to remove the additive diagnostics.

## 4. Tasks

- [x] Add negative and positive command-level regression cases.
- [x] Implement shared-parser diagnosis and bounded output.
- [x] Validate module and local consumer, review and close through POSE.

## 5. Decisions

Keep health dimensions separate: valid pointers do not imply fresh or untampered
state. Missing optional state remains valid but explicitly uninitialized.
No autonomous remediation is justified by detecting a broken reference.

## 6. Validation

Required before closeout:

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Absent/valid/broken/malformed/tampered state; no writes | `go test ./internal/cli -run TestDoctorState -count=1` | Correct findings and unchanged bytes |
| Module integration and CLI reachability | `pose validate --strict --module pose-mcp --json .pose/results/delivery-validation.json` | Six existing checks pass |
| Consumer diagnostic | `pose doctor --json` and `pose state` with rebuilt binary in Harne8 | Consistent pointer health; no rollout claim |

### Execution log

- 2026-09-19: `go test ./internal/cli -run TestDoctorState -count=1` passed for
  absent, valid, broken, malformed, tampered and stale state. The command-level
  tests also verified that doctor leaves the source bytes unchanged and that
  `doctor` and `state` agree on pointer validity.
- 2026-09-19: the module implementation was exercised with `go test ./internal/cli
  -count=1`, `go vet ./...` and `go build ./...`; all passed. The rebuilt binary
  was run against the Harne8 checkout with JSON doctor output and `pose state`;
  this is consumer evidence only and does not claim rollout or adoption.
- 2026-09-19: strict POSE structure and the spec ready check passed before
  closeout. Review bundle sealing remains after the attributed implementation
  commit, so no review result is claimed by this log.

### Requirement trace

- R1 [satisfied] `TestDoctorStateHealth`, `TestDoctorStateAbsentAndMalformed`
- R2 [satisfied] `TestDoctorStateHealth`, `TestDoctorStateIntegrityAndFreshness`
- R3 [satisfied] `TestDoctorStateHealth`, with unchanged-state-bytes assertions
- R4 [satisfied] `TestDoctorStateHealth`, `TestDoctorStateAbsentAndMalformed`

## 7. Final Report

### Delivered scope

Doctor now exposes read-only project-state artifact, pointer, integrity and
freshness findings through the existing text and JSON surfaces. Missing optional
state is reported as uninitialized; malformed, unreadable or broken state is
diagnosable with bounded detail and explicit remediation. No fixer, refresh or
historical rewrite was added.

### Validation executed

- `go test ./internal/cli -run TestDoctorState -count=1`: SUCCESS.
- `go test ./internal/cli -count=1`: SUCCESS.
- `go vet ./...`: SUCCESS.
- `go build ./...`: SUCCESS.
- Rebuilt consumer binary: doctor/state comparison completed against the Harne8
  checkout without a rollout claim.

### Residual risks

- [open] The Harne8 instance-health adoption must still compose this diagnostic
  with its state/index and metadata gates (owner:@harne8-platform crit:medium
  review:2026-10-03)
