---
slug: pose-shipped-review-profiles-are-schema-v2
status: in-progress
created_at: 2026-09-10
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
delivers:
task_type: bugfix
---

# Spec: The shipped review profiles are at the schema the migration targets

## 1. Intent

### Goal
Ship `milestone-integration.json` and `roadmap-outcome.json` at schema v2, and
stop the migration from reporting a migration it did not perform.

### Business value
Two of the five shipped review profiles carried `schema_version: 1`. The v1-to-v2
migration in `seedAbsentInstanceConfig` copies the distribution's own profile
over the instance's and logs `review-profile (migrated): <name> (v1 -> v2)`. With
the shipped file itself at v1, it copied a v1 file over a v1 file and reported
success — on every `pose update`, forever.

Two costs. The instance never reaches v2 for those profiles, so their criteria
stay outside the closed rule and evidence catalogs that schema v2 enforces: a
v1 profile is deliberately exempt from `validateReviewContractRefs`, which is the
check that stops a profile demanding a class no check may emit. And an operator
reading the log is told a migration succeeded that never happened, which is worse
than silence — it is the failure mode this repository has spent the cycle closing
in other places.

It surfaced while adopting the release in a consumer repository, not from a
report: the log said `migrated` twice on every run and the two files never
changed.

### Constraints
- Both profiles must satisfy schema v2 to be shipped as v2: their criteria's
  rules must resolve and their evidence classes must be emittable. Neither
  declares any rule, and the only class either declares is `integration`.

### Non-goals
- Adding selectors, tools or independence to either profile. Schema v2 permits
  them; it does not require them, and inventing configuration for two profiles
  that never had it is a different change.

---

## 2. Requirements

### Functional
- R1: Every review profile the distribution ships shall be at
  `ReviewPolicySchemaVersion`, enforced.
- R2: The migration shall log a migration only when the file that replaces the
  instance's is actually at the target schema.

### Non-functional
- The two profiles keep their formatting: only the schema line changes.

---

## 3. Technical Plan

### Affected areas
- `.pose/review-profiles/milestone-integration.json`,
  `.pose/review-profiles/roadmap-outcome.json` — the shipped profiles
- `pose-mcp/internal/cli/stack_seed.go` — the migration's copy branch

### Artifacts
- created: .pose/specs/2026-09-10-pose-shipped-review-profiles-are-schema-v2.md
- created: .pose/changelogs/unreleased/pose-shipped-review-profiles-are-schema-v2.md
- created: pose-mcp/internal/cli/shipped_profiles_schema_test.go
- modified: .pose/review-profiles/milestone-integration.json
- modified: .pose/review-profiles/roadmap-outcome.json
- modified: pose-mcp/internal/scaffold/dist/.pose/review-profiles/milestone-integration.json
- modified: pose-mcp/internal/scaffold/dist/.pose/review-profiles/roadmap-outcome.json
- modified: pose-mcp/internal/cli/stack_seed.go

### Technical risks
- Bumping a profile to v2 subjects its criteria to the closed catalogs. Both were
  checked first: neither declares a rule, and `integration` is in
  `ValidEvidenceClasses`. A profile that failed that check would have to be
  reconciled rather than bumped.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Both shipped profiles at v2, enforced by a test (R1)
- [x] Increment 2: The migration verifies before it claims (R2)

### Validation
- [x] The guard shown failing when a shipped profile is put back to v1

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the log line could be removed, or the copy could be verified.
- Decision: verify, and enforce the shipped schema with a test.
- Rationale: removing the message would hide the same failure rather than fix
  it. The test is what makes the copy branch's assumption true; the verification
  is what makes the message honest if a future profile slips.

---

## 6. Validation

### Strategy
Assert the shipped set, and require the assertion to fail when a profile is put
back to v1.

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
- Notes: putting `milestone-integration.json` in the embedded scaffold back to
  `schema_version: 1` fails `TestEveryShippedReviewProfileIsAtTheCurrentSchema`
  with the file named. Before the fix, `pose update` in an adopting repository
  logged both profiles as migrated on every run and left both at v1; after it,
  the profiles load as v2.

### Results summary
- Successes: R1, R2 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestEveryShippedReviewProfileIsAtTheCurrentSchema reads every profile from the embedded scaffold and requires ReviewPolicySchemaVersion, failing when one is reverted>
- R2 [satisfied] <shippedProfileIsCurrentSchema gates the copy-and-log branch; TestTheMigrationOnlyClaimsACopyThatMigrates covers current, stale and unreadable content>

### Known gaps
- Instances that already ran an older `pose update` still hold both profiles at
  v1. The next update replaces them, now that the shipped files are v2.

---

## 7. Final Report

### Summary
The two profiles ship at the schema the migration targets, and the migration no
longer reports a copy that migrates nothing.

### Follow-ups

- [open] Report an instance profile still at schema v1 as a `pose doctor` finding, so an instance that has not run an update since the fix is visible rather than waiting to be noticed — owner:unowned crit:low review:2027-01-10
