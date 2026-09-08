---
slug: pose-doctor-selected-profiles-only
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-diagnose-invisible-governance-failures
priority: 1
components: pose-mcp
delivers:
---

# Spec: Doctor reports only the review profiles policy selects

## 1. Intent

### Goal
Narrow `review.evidence-vocabulary` to the profiles `.pose/policy/review.json`
selects, so it reports on the review plans an instance builds rather than on
every file in `.pose/review-profiles/`.

### Business value
The check shipped in v1.8.0 and its first run against an adopting repository
flagged four profiles — all four the shipped ones that repository had already
replaced:

```
[!] review.evidence-vocabulary 4 review profile(s) demand evidence classes no
    registered check may emit: backend-review.json, frontend-review.json,
    milestone-integration.json, spec-closeout.json
```

Policy there selects its own variants, none of which was flagged. The check was
reporting a true property of files on disk and a false one about the plans the
instance actually builds.

That is not an edge case. It is the shape every project lands in the moment it
owns its profiles and leaves the originals in place — which is the documented
way to reconcile evidence classes without `pose update` overwriting the work.

The cost is specific to what this check is for. It exists to make an invisible
failure visible; a warning an operator must investigate to dismiss spends the
attention the check was built to earn, and teaches them to skim it.

### Constraints
- `validate.evidence-class-coverage` stays as it is. A check declaring no
  evidence class is discarded wherever it is used, so reading what is declared
  is correct there.
- No new configuration. The policy already records what it selects.

### Non-goals
- Reporting unselected profiles at a lower severity. `doctor` has ok, warn and
  error; an inert file deserves none of them.

---

## 2. Requirements

### Functional
- R1: The check shall inspect only the profiles named by `profiles` and
  `overlay_profiles` in the review policy.
- R2: A profile present on disk but not selected shall produce no finding, even
  when it demands classes no check may emit.
- R3: The check shall stay silent when the review policy cannot be read, rather
  than falling back to the whole directory.

### Non-functional
- The existing doctor suite passes, with its fixtures corrected where they were
  passing incidentally.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor.go` — profile selection

### Artifacts
- created: .pose/specs/2026-09-08-pose-doctor-selected-profiles-only.md
- renamed: .pose/changelogs/unreleased/pose-doctor-selected-profiles-only.md -> .pose/changelogs/v1.8.1/pose-doctor-selected-profiles-only.md
- modified: .pose/specs/2026-09-08-pose-diagnose-invisible-governance-failures.md
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_invisible_failures_test.go

### Technical risks
- A profile selected by a policy the check cannot parse goes unreported. That is
  the same silence as an instance with no review policy, and preferable to
  reverting to a directory scan whose output the operator has already learned to
  dismiss.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Read the selected refs from the review policy (R1, R3)
- [x] Increment 2: Inspect only those profiles (R2)

### Validation
- [x] Unselected offending profile reports ok; selected one still warns

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: the two existing tests passed before this change, and would have
  passed after it for the wrong reason — they wrote a profile the fixture's
  policy never selected, and the check found it by scanning the directory.
- Decision: seed the policy in both, so each asserts through the path the
  production code takes.
- Rationale: a test that passes because the code reads a directory, when the
  contract is about what policy selects, is not evidence about the contract. It
  is the third fixture in this session's work to confirm the case that works
  rather than the case that runs; correcting them is worth more than the
  refinement itself.

---

## 6. Validation

### Strategy
Assert the new behaviour directly, and confirm the corrected fixtures still
cover the original one.

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
- Date: 2026-09-08
- Environment: local, Go 1.26
- Notes: all eight packages pass. Restoring the unfiltered iteration turns
  `TestDoctorIgnoresAProfileThePolicyDoesNotSelect` red. The two pre-existing
  tests failed on the first build of this change — correctly, since their
  fixtures never selected the profile they wrote — and pass once the policy is
  seeded.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <doctor.go reads GetReviewPolicy and iterates the ids in profiles and overlay_profiles, stripping the @version suffix>
- R2 [satisfied] <TestDoctorIgnoresAProfileThePolicyDoesNotSelect writes an offending unselected profile and asserts an ok finding that does not name it>
- R3 [satisfied] <the block is guarded on GetReviewPolicy returning no error, so an unreadable policy yields no finding rather than a directory scan>

### Known gaps
- Selection is read from the policy, not from a resolved plan, so a profile
  named by policy but never matched by any scope is still inspected. That is
  intentional: it is configured to be used, and an unsatisfiable class in it is
  a latent failure rather than an inert file.

---

## 7. Final Report

### Follow-ups

- [open] Audit the doctor fixtures for others that pass by scanning rather than through the path production takes — owner:unowned crit:medium review:2026-11-08
