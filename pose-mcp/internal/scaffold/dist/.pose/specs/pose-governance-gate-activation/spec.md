---
slug: pose-governance-gate-activation
status: draft
created_at: 2026-08-07
completed_at:
supersedes:
depends_on:
priority: 3
components: governance, assessment
delivers:
---

# Spec: Turn the tolerated governance warnings into gates that bite

## 1. Intent

### Goal
Complete the adoption of governance mechanisms POSE already ships but still runs
in warning mode, and fix the one evidence-identity defect that makes a governed
claim silently retarget.

### Business value
Each of these gates was built, tested and then deliberately left tolerant until
the repository could satisfy it. That was the right call at build time and is
now the reason none of them can fail: a legacy spec without a requirement trace,
an overdue follow-up, an unrecorded amendment baseline — all pass today. A gate
that cannot fail is documentation, not governance, and this repository is the
reference instance others copy.

The assessment item is different in kind and is the reason this spec is not
purely procedural: debt marker ids are positional (`DEBT-001`), so a spec citing
one has evidence that silently points at a different marker as soon as an
earlier one is added. `pose-assessment-engine-precision` fixed exactly this
defect for integration gaps and left debt ids untouched.

### Constraints
- Flipping a gate to strict must come after the repository satisfies it, never
  before. Each requirement pairs the flip with the cleanup that earns it.
- Existing debt citations must keep resolving across the id change, or the
  change trades one silent retarget for a loud break.

### Non-goals
- Adding new governance mechanisms. Everything here already exists.
- Revisiting whether any of these gates should exist.

---

## 2. Requirements

### Functional
- R1: Debt marker ids shall derive from file and marker identity rather than
  scan position, so a citation cannot retarget when an earlier marker appears.
- R2: The ten pre-contract specs shall gain requirement-trace sections or be
  archived, and the legacy done-without-trace warning shall then become an
  error.
- R3: Amendment baselines shall be recorded for active-roadmap specs as they
  enter execution, making the amendment gate effective beyond fixtures.
- R4: `--fail-overdue` shall be adopted in the quarterly governance audit.
- R5: The feature and bugfix workflows and skills shall cite knowledge refs, so
  citation becomes routine rather than exceptional.
- R6: The first quarterly audit run and the first real recurrence intervention
  shall be reviewed and their verdicts recorded.
- R7: `--sarif` shall be adopted in the CI security surface once code-scanning
  upload accepts validation results.

### Non-functional
- No gate flips without the repository first passing it locally in strict mode.

### Security
- R7 publishes validation findings to code scanning; confirm no finding body
  carries a path or value outside the repository before enabling upload.

### Compatibility
- R1 changes the shape of debt ids once. Citations must be migrated in the same
  change, exactly as `gap_id` was.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/techdebt.go` — id derivation (R1).
- The ten pre-contract specs — trace sections or archival (R2).
- `.pose/workflows/` and `.agents/skills/` — knowledge citation (R5).
- `.github/workflows/security.yml` — SARIF upload (R7).

### Artifacts
<!-- Declared at closeout. -->
- created: .pose/specs/pose-governance-gate-activation/spec.md

### API/contract changes
- Debt marker ids change shape once (R1), the same one-time break
  `pose-assessment-engine-precision` accepted for `gap_id`.

### Technical risks
- R2 is the largest: ten legacy specs is real archaeology, and archiving one is
  a judgement call about whether its requirements still describe the system.
- Flipping a gate in the same change that cleans up for it makes the diff hard
  to review. Prefer cleanup first, flip second, in separate commits.

---

## 4. Tasks

### Planning
- [ ] Enumerate the ten pre-contract specs and decide trace-or-archive for each
- [ ] Confirm which debt citations exist today, so R1's migration is bounded

### Implementation
- [ ] R1: derive debt ids from file and marker identity, migrate citations
- [ ] R2: cleanup then flip the legacy trace warning to an error
- [ ] R3: record amendment baselines for active-roadmap specs
- [ ] R4: adopt --fail-overdue in the quarterly audit
- [ ] R5: cite knowledge refs from the workflows and skills
- [ ] R6: review the first audit run and the first recurrence intervention
- [ ] R7: adopt --sarif in the security surface

### Validation
- [ ] Each flipped gate demonstrated failing on a deliberately broken fixture
- [ ] Run the mandatory checks

---

## 6. Validation

### Strategy
A gate flip is only proven by watching it fail: for each, construct the
violation it is supposed to catch and confirm a non-zero exit, then confirm the
repository itself passes. R1 is proven by adding a marker earlier in the scan
and asserting existing citations still resolve.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/pose ./internal/cli -run "Debt|Trace|Amendment|Followup" -count=1`
- Scope: debt id identity and the flipped gates
- Expected: ok

#### Security / Contract
- Command: `pose check --strict` and `pose lint-spec --all --strict`
- Scope: the repository under the newly strict gates
- Expected: SUCCESS

### Requirement trace
<!-- Filled at closeout. -->

### Known gaps
- R6 is calendar-bound: the first quarterly audit is dated 2026-10-01 and cannot
  be reviewed before it runs.

---

## 7. Final Report

<!-- Filled at closeout. -->

### Follow-ups

- [open] If R2's archaeology shows most of the ten pre-contract specs describe behaviour that no longer exists, prefer archiving them over retrofitting traces — a trace invented years later is evidence of nothing. (owner:@pose-maintainers crit:medium review:2026-10-23)
