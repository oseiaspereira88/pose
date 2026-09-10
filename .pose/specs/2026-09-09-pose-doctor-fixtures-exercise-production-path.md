---
slug: pose-doctor-fixtures-exercise-production-path
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-doctor-selected-profiles-only
priority: 1
components: pose-mcp
delivers:
---

# Spec: Audit which doctor branches the suite actually reaches

## 1. Intent

### Goal
Find the doctor branches no test executes, and cover the ones that describe
behaviour rather than guard against a malformed file.

### Business value
`pose-doctor-selected-profiles-only` recorded a follow-up: audit the doctor
fixtures for others that pass by scanning rather than through the path
production takes. That was written after two of its own tests turned out to pass
because the check read a directory, while the contract was about what policy
selects.

Reading the tests again would find the same class of gap only where I already
suspected it. Coverage answers it directly: it says which statements the suite
executes, and a branch nothing executes is a branch whose regression is
invisible however carefully it was written.

Fifteen blocks in the diagnostics were never reached. Twelve are defensive —
`git` missing from PATH, a malformed profile, an unparseable schema stamp — and
testing them buys little. Three describe behaviour:

- **`review.evidence-vocabulary` inspects a profile's criteria and its tools,
  and only the criteria half ever ran.** The check has two halves and the suite
  proved one.
- **`policy.artifact-roots` had never returned `ok` in a test.** A check that
  only ever proves its warning could be reporting on nothing and no one would
  know.
- **`mcp.config` has three answers and one was executed.** The untested branch
  is the one that matters most: a configuration that exists and points somewhere
  other than the native binary.

### Constraints
- Cover behaviour, not defensive guards. A test that forces `git` off PATH
  proves the engine degrades, which is worth less than the maintenance it costs.

### Non-goals
- Raising coverage as a number. The uncovered blocks were read individually and
  three were selected on what they mean, not to move a percentage.

---

## 2. Requirements

### Functional
- R1: The tool half of `review.evidence-vocabulary` shall be exercised.
- R2: `policy.artifact-roots` shall be exercised in both directions.
- R3: `mcp.config` shall be exercised for a configuration that is present and
  not the native binary.

### Non-functional
- Each new test asserts the opposite answer too, so none can pass by the check
  having one reachable outcome.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor_fixture_audit_test.go` — the added coverage

### Artifacts
- created: .pose/specs/2026-09-09-pose-doctor-fixtures-exercise-production-path.md
- renamed: .pose/changelogs/unreleased/pose-doctor-fixtures-exercise-production-path.md -> .pose/changelogs/v4.0.0/pose-doctor-fixtures-exercise-production-path.md
- created: pose-mcp/internal/cli/doctor_fixture_audit_test.go

### Technical risks
- Coverage says a statement ran, not that an assertion depended on it. Each test
  here also asserts the contrary case, which is the cheap defence; it is not a
  proof that the assertion is load-bearing.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Measure which diagnostics branches the suite reaches
- [x] Increment 2: Cover the three that describe behaviour (R1, R2, R3)

### Validation
- [x] The three previously unreached blocks are now executed

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: how to audit "fixtures that pass by scanning rather than through the
  production path".
- Options considered: (a) re-read the doctor tests; (b) measure coverage of the
  diagnostics and read the unreached blocks.
- Decision: (b).
- Rationale: (a) is how the original gap survived. Re-reading finds what the
  reader already suspects, and the whole point of the follow-up is that the
  suspicion was wrong twice. Coverage names the blocks without an opinion, and
  the judgement then applies to a short list rather than to thirty tests.

### Decision 2
- Date: 2026-09-09
- Context: twelve of the fifteen unreached blocks are defensive guards.
- Decision: leave them.
- Rationale: forcing `git` off PATH or feeding a truncated schema stamp proves
  the engine degrades rather than panics, which is real but worth less than the
  fixtures it costs. The three selected are the ones where a silent regression
  would change what an operator is told.

---

## 6. Validation

### Strategy
Measure before and after, and confirm the specific blocks that were unreached
are the ones now executed.

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
- Date: 2026-09-09
- Environment: local, Go 1.26
- Notes: `go test -run TestDoctor -coverprofile` reported `runDoctorDiagnostics`
  at 93.6% with fifteen unreached blocks. After the three tests, the same
  measurement reports 95.4% and the blocks at lines 315, 365 and 735 — the
  `mcp.config` non-native branch, the `policy.artifact-roots` ok branch and the
  tool half of `review.evidence-vocabulary` — are executed. The remaining twelve
  are the defensive guards listed in Decision 2.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestDoctorReportsAnUnproducibleClassDemandedByATool writes a profile whose criteria are clean and whose tool demands an unproducible class, and asserts the finding names it; block 735 moves from unreached to executed>
- R2 [satisfied] <TestDoctorAcceptsArtifactRootsThatResolve asserts ok for a root that exists and warn for one that does not; block 365 moves from unreached to executed>
- R3 [satisfied] <TestDoctorReportsAnMCPConfigThatIsNotTheNativeBinary asserts warn for an npx-based configuration and ok for the native one; block 315 moves from unreached to executed>

### Known gaps
- Twelve defensive blocks remain unreached, by decision.
- This audited `pose doctor` only. The same measurement applied to the other
  commands would likely find the same shape, and has not been run.

---

## 7. Final Report

### Follow-ups

- [done] Run the same coverage audit over the other command surfaces, since this one found three unreached behavioural branches in a single command. Run in `pose-policy-keys-and-release-surface-coverage`: it found 36 functions at zero, the largest being the whole release surface, which now runs under test from 0 of 20 functions to 20 of 20; the rest are carried as a follow-up there. — owner:unowned crit:medium review:2026-12-09
