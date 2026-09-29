---
slug: pose-doctor-reports-a-profile-left-behind
status: done
created_at: 2026-09-10
completed_at: 2026-09-29
supersedes:
depends_on: pose-shipped-review-profiles-are-schema-v2
priority: 0
components: pose-mcp
delivers: surface:doctor-reports-a-profile-left-behind
task_type: feature
---

# Spec: pose doctor reports a review profile left below the current schema

## 1. Intent

### Goal
Report, as a `pose doctor` finding, any review profile in the instance below the
schema the engine enforces.

### Business value
Two profiles shipped at `schema_version: 1` for long enough that every instance
installed in that window still carries them, and the migration that should have
moved them copied a v1 file over a v1 file and reported success. The shipped
files are v2 now, so the next `pose update` migrates them — but until it runs,
the instance is carrying profiles exempt from the closed rule and evidence
catalogs schema v2 enforces, and nothing says so.

That exemption is the substance, not the version number. A v1 profile skips
`validateReviewContractRefs`, which is the check that stops a profile demanding
an evidence class no registered check may emit. An instance can therefore plan a
gate only a fabricated disposition can pass, and the defect that made it possible
has already been fixed — the instance just has not heard.

### Constraints
- An instance with no review profiles must produce no finding. Saying something
  on every fresh repository is how a check is learned to be ignored.

### Non-goals
- Migrating on `doctor`. The diagnostic reports; `pose update` is what changes
  the instance.

---

## 2. Requirements

### Functional
- R1: A profile below `ReviewPolicySchemaVersion` shall be reported, named, with
  the remedy.
- R2: A profile already current shall not be named.
- R3: An instance with no profiles shall produce no finding.

### Non-functional
- The finding reads the instance's profiles, not the shipped set.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor.go` — the new `review.profile-schema` check

### Artifacts
- created: .pose/specs/2026-09-10-pose-doctor-reports-a-profile-left-behind.md
- renamed: .pose/changelogs/unreleased/pose-doctor-reports-a-profile-left-behind.md -> .pose/changelogs/v5.0.0/pose-doctor-reports-a-profile-left-behind.md
- created: pose-mcp/internal/cli/doctor_profile_schema_test.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: .github/workflows/ci.yml
- modified: .github/workflows/release.yml
- modified: .github/workflows/security.yml
- modified: pose-mcp/internal/cli/release_compatibility_test.go
- modified: pose-mcp/internal/version/workflow_history_depth_test.go

Backfilled on 2026-09-29: `6209806` (Two follow-ups the v4.0.1 fixes opened (#84)) carries this spec's trailer and also changed the 5 paths declared last. They are claimed so the Git change set reconciles; the commit bundled other work, so a claim here records provenance, not that this spec designed the change.

### Delivery targets

- surface:doctor-reports-a-profile-left-behind module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

Declared on 2026-09-29 at closeout: the delivery-surface policy (adopted 2026-08-04) requires a typed target for each delivery root the change set touched, and the spec predates that requirement being applied to it.

### Technical risks
- The check reads every profile in the directory, including ones a project
  authored itself. That is the intent: a hand-written v1 profile has the same
  exemption as a shipped one.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The check reports, accepts and stays quiet (R1, R2, R3)

### Validation
- [x] A current profile shown not being named alongside a stale one

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the finding could name the version alone or say what being behind
  costs.
- Decision: say what it costs.
- Rationale: `schema_version: 1` means nothing to an operator on its own. That a
  v1 profile is exempt from the check which stops it demanding an unsatisfiable
  class is the reason to act, and the remedy is one command.

---

## 6. Validation

### Strategy
Report on an instance holding one stale profile and one current one, and require
the current one not to be named.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-10
- Environment: local, Go 1.26
- Notes: an instance with `legacy.json` at v1 and `current.json` at v2 warns
  naming only the first. This repository reports ok, its own profiles having been
  migrated by the v4.0.1 update.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestDoctorReportsAReviewProfileBelowTheCurrentSchema requires the warn, the profile's name and a hint naming `pose update`> surface:doctor-reports-a-profile-left-behind evidence:integration
- R2 [satisfied] <the same test requires the current profile not to appear in the message — naming it would tell an operator to migrate what is already migrated>
- R3 [satisfied] <TestDoctorSaysNothingWhenThereAreNoProfiles requires no finding at all on an instance with none> surface:doctor-reports-a-profile-left-behind evidence:integration

### Known gaps
- The check reports the schema and not what a stale profile actually declares.
  A v1 profile demanding an unsatisfiable class is the concrete harm, and this
  reports the condition rather than the instance.

---

## 7. Final Report

### Summary
An instance carrying a profile the engine has moved past says so, instead of
waiting for someone to notice.

### Follow-ups

- [open] Report what a stale profile actually declares that schema v2 would refuse, so the finding names the harm rather than the condition (owner:unowned crit:low review:2027-03-10)
