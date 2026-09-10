---
slug: pose-compat-gate-pose-md-preservation
status: in-progress
completed_at:
created_at: 2026-09-10
supersedes:
depends_on: pose-compat-gate-manual-refresh-assertion
priority: 0
components: pose-mcp
task_type: refactor
delivers:
---

# Spec: The release gate holds POSE.md to the preservation property

## 1. Intent

### Goal
The upgrade lab in the release compatibility gate shall customize POSE.md and
require the customization to survive every supported upgrade, as it already
does for AGENTS.md.

### Business value
The follow-up assumed POSE.md takes the same path as AGENTS.md and only lacked
the assertion. Measured, it does not. A note appended to the end of AGENTS.md
lands in an instance-owned section and stays in the manual. The same note
appended to POSE.md lands in an engine-owned section: the upgrade drops it from
the manual, warns, and keeps it in `POSE.md.pose-backup`.

So the gate had only ever proven the keep-in-place half of the contract. The
half where the upgrade removes what an instance wrote — and the backup is the
only thing standing between that and a silent loss — was never exercised by a
real prior release.

### Constraints
- The gate downloads each supported prior release, so it runs in the release
  workflow; a local run needs the network.

### Non-goals
- Changing which sections of either manual are instance-owned.

---

## 2. Requirements

### Functional
- R1: The upgrade lab shall append the customization marker to POSE.md as well
  as AGENTS.md.
- R2: Each supported upgrade pair shall fail unless the marker survives in
  POSE.md or in `POSE.md.pose-backup`.

### Non-functional
- The same assertion covers both manuals, so they cannot drift apart.

---

## 3. Technical Plan

### Affected areas
- `tests/release/compat.sh` — the upgrade lab's fixture and assertions

### Artifacts
- created: .pose/specs/2026-09-10-pose-compat-gate-pose-md-preservation.md
- renamed: .pose/changelogs/unreleased/pose-compat-gate-pose-md-preservation.md -> .pose/changelogs/v5.0.1/pose-compat-gate-pose-md-preservation.md
- modified: tests/release/compat.sh
- modified: .pose/specs/2026-08-07-pose-compat-gate-manual-refresh-assertion.md

### Technical risks
- None beyond the gate's own: a prior release that could not keep the note
  would now fail the pair, which is the point.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Customize POSE.md and assert it in one loop with AGENTS.md (R1, R2)

### Validation
- [x] Measure which branch of the assertion each manual satisfies
- [x] Show the assertion fails without the backup

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the assertion accepts the marker in the manual or in its backup.
  POSE.md satisfies it only through the backup.
- Decision: keep the either-or assertion rather than require the marker in the
  manual.
- Rationale: dropping content from an engine-owned section is the documented
  contract (pose-managed-doc-content-preservation); losing it without a backup
  is what is not allowed. Requiring it in place would fail every pair for
  behaviour that is correct.

---

## 6. Validation

### Strategy
Run the full gate locally against every supported prior release, and check the
new assertion both ways.

### Deterministic checks

#### Gate
- Command: `COMPAT_REPORT=<tmp> bash tests/release/compat.sh`
- Scope: this repository
- Expected: `Result: COMPATIBLE`

#### Syntax
- Command: `bash -n tests/release/compat.sh`
- Scope: `tests/release/compat.sh`
- Expected: exit 0

### Execution log
- Date: 2026-09-10
- Environment: local, Go 1.26, Linux, network
- Notes: all four supported pairs — 1.1.0, 1.0.0, 0.19.0 and 0.18.2 to 5.0.0 —
  pass with the POSE.md assertion, and the gate reports COMPATIBLE. A manual
  reproduction of one pair showed where each marker ends up: in AGENTS.md
  itself, and only in `POSE.md.pose-backup` for POSE.md, with `[WARN] backed up
  customized: POSE.md`. Moving the backup aside makes the assertion fail.
  shellcheck is not installed locally; CI runs it over `tests/release/`.

### Results summary
- Successes: R1, R2 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <check_upgrade_pair appends the marker to AGENTS.md and POSE.md in the fixture>
- R2 [satisfied] <one loop requires the marker in each manual or its .pose-backup; the local gate run passed all four pairs, and the assertion fails with the backup moved aside>

### Known gaps
- None.

---

## 7. Final Report

### Summary
The release gate now exercises the half of the preservation contract where the
upgrade removes what an instance wrote and a backup has to keep it.

### Follow-ups
