---
slug: pose-first-governed-loop-quickstart
status: in-progress
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
- modified: .github/workflows/ci.yml

### Technical risks
- A documentation test that only asserts exit codes will pass while the prose
  drifts from the observed output. It must assert on the output the page
  claims, not merely that the commands ran.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Rewrite as one linear loop ending in a gate result (R1, R3)
- [x] Increment 2: Add the blocked-then-resolved beat (R2)
- [x] Increment 3: Executable documentation test (R5)

### Validation
- [x] Executable documentation test passes
- [ ] Measure on a clean environment and publish the measurement (R4) — blocked
      on pose-release-recovery-verification

### Known gaps
- No time budget is published. The previous "first validation in under 10
  minutes" claim predates the current lifecycle and was never re-measured, so
  repeating it would be publishing an unmeasured promise — the same class of
  error as publishing an unreleased version. The page states the loop and its
  steps; the number waits for the measurement.

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
- Date: 2026-09-07
- Environment: local Linux, Go 1.26.5, throwaway git repository
- Notes: every command and its output was captured from a real run in a
  disposable instance before being written into the page — none of the shown
  output is illustrative. The loop turned out to have a better blocking beat
  than the one planned: rather than an undispositioned follow-up, the closeout
  gate refuses a spec marked `done` whose `R1` has no requirement-trace entry.
  That is a sharper demonstration, because it says the thing directly — you
  claimed it is done, and the promise is not connected to any evidence.
- The clean-environment measurement (R4) has not been made: it depends on the
  published installer, which is still returning 404 until
  `pose-release-recovery-verification` closes.

### Results summary
- Successes: R1, R2, R3, R5.
- Failures: none.
- Warnings: R4 blocked upstream, not skipped — see Known gaps.

### Requirement trace
- R1 [satisfied] <docs-site/docs/quickstart.md "The first governed loop" — seven ordered steps, no branch to choose>
- R2 [satisfied] <steps 2 and 6 block, steps 3 and 7 resolve; check:quickstart-loop asserts both refusals>
- R3 [satisfied] <every shown output captured from a real run; check:quickstart-loop asserts the quoted strings>
- R4 [deferred-integration: spec:pose-release-recovery-verification] <cannot be measured while the published installer returns 404>
- R5 [satisfied] <tests/quickstart/first-governed-loop.sh; check:ci-quickstart-loop>

---

## 7. Final Report

### Follow-ups

- [open]
