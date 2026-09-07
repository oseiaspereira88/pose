---
slug: pose-first-governed-loop-quickstart
status: draft
created_at: 2026-09-06
completed_at:
supersedes:
depends_on: pose-release-recovery-verification
priority: 2
components: docs, cli
delivers:
---

# Spec: First governed loop quickstart

## 1. Intent

### Goal
Make the first fifteen minutes with POSE produce one specific outcome — a
deterministic gate that blocks for a legible reason and then passes — and
measure how long that actually takes.

### Business value
The activation metric for this launch is not `pose installed`. It is **first
governed delivery**: a person creates or imports a spec, POSE resolves what
applies, a gate runs, and the result is deterministic and explicable.

Two things currently prevent that. The install command is dead, which
`pose-release-recovery-verification` fixes. And the published time budget —
"first validation in under 10 minutes", ratified in July — predates the
current lifecycle and has never been re-measured. Publishing a budget that
has not been measured is the same class of error as publishing a version that
has not been released.

### Constraints
- The published budget must be a measurement, on a clean machine, by someone
  following only the written text. If the measurement says fourteen minutes,
  the page says fourteen minutes.
- The quickstart teaches one loop, not the command surface. A reader who
  finishes should be able to explain why the gate blocked — not to have seen
  twenty commands.

### Non-goals
- Covering rules authoring, MCP, CI or analytics. Those are `Use` and
  `Reference` documentation, reached after the loop lands.

---

## 2. Requirements

### Functional
- R1: The quickstart shall take the reader from install to a first
  deterministic gate result in one linear sequence, with no branch the reader
  must choose between.
- R2: The sequence shall include a step where the gate **blocks** for a
  legible reason, and a step where the reader resolves it — the point of POSE
  is not visible in an all-green run.
- R3: Every command shall be copy-pasteable and produce output the page shows,
  so a reader can tell whether their run diverged.
- R4: The elapsed time shall be measured on a clean environment by following
  only the published text, and the published budget shall be that measurement.
- R5: An executable test shall run the documented sequence and fail when a
  documented command or its expected outcome no longer holds.

### Non-functional
- The whole path runs offline after the binary is installed.

---

## 3. Technical Plan

### Affected areas
- `docs-site/docs/quickstart.md`
- `tests/quickstart/`

### Artifacts
- modified: docs-site/docs/quickstart.md
- created: tests/quickstart/first-governed-loop.sh

### Technical risks
- A documentation test that only asserts exit codes will pass while the prose
  drifts from the observed output. It must assert on the output the page
  claims, not merely that the commands ran.

---

## 4. Tasks

### Implementation
- [ ] Increment 1: Rewrite as one linear loop ending in a gate result (R1, R3)
- [ ] Increment 2: Add the blocked-then-resolved beat (R2)
- [ ] Increment 3: Executable documentation test (R5)

### Validation
- [ ] Measure on a clean environment and publish the measurement (R4)

---

## 6. Validation

### Strategy
The quickstart is validated the way a stranger validates it: on a machine with
nothing installed, following only what is written.

### Deterministic checks

#### Test
- Command: `bash tests/quickstart/first-governed-loop.sh`
- Scope: every documented command and its documented outcome
- Expected: exit 0; the blocked step blocks, the resolved step passes

### Execution log
- Date:
- Environment: clean container, no Go toolchain, no checkout
- Notes: record the measured elapsed time here; it becomes the published budget

### Requirement trace
<!-- Filled at closeout. -->

---

## 7. Final Report

### Follow-ups

- [open]
