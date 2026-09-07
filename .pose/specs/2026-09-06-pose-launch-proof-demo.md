---
slug: pose-launch-proof-demo
status: draft
created_at: 2026-09-06
completed_at:
supersedes:
depends_on: pose-first-governed-loop-quickstart
priority: 3
components: docs
delivers:
---

# Spec: Launch proof demo

## 1. Intent

### Goal
Produce a recording, under a minute, in which POSE blocks a delivery that
every conventional check has already passed — and then legitimately closes it.

### Business value
The thesis of the launch is one sentence: *"done" is not evidence*. It is very
hard to argue and very easy to show. A recording in which tests pass, lint
passes, build passes, and the delivery is still blocked because a follow-up
has no disposition communicates the entire product in about fifteen seconds.

An all-green recording does the opposite. It shows a tool that agrees with the
agent, which is what every reader already assumes exists.

### Constraints
- The run must be real. A staged terminal recording of output POSE did not
  produce would violate the project's own evidence policy at the exact moment
  it is asking to be trusted about evidence.
- The demo must be reproducible from the repository, so it can be re-recorded
  when the output changes rather than becoming a stale artifact.

### Non-goals
- Narration, music, or a feature tour.

---

## 2. Requirements

### Functional
- R1: The demo shall show a delivery in which the conventional checks pass and
  POSE still blocks, with the blocking reason legible on screen.
- R2: The demo shall then show the block resolved by a real disposition, and
  the delivery closing.
- R3: The recording shall be produced by executing a scripted scenario in the
  repository, so it is reproducible and re-recordable.
- R4: The total runtime shall be under 60 seconds and the content shall be
  legible without audio.
- R5: The demo shall be embedded on the landing page and in the README.

### Non-functional
- Text must remain legible at the width the landing page renders it.

---

## 3. Technical Plan

### Affected areas
- `examples/` — the demo scenario
- landing page and README embedding

### Artifacts
- created: examples/demo/blocked-then-closed/
- created: examples/demo/record.sh
- modified: README.md

### Technical risks
- A scenario contrived to fail is unconvincing. The blocked state should be
  one POSE produces routinely — an undispositioned follow-up is both the most
  common and the most explicable.

---

## 4. Tasks

### Implementation
- [ ] Increment 1: Scripted scenario reaching a genuine blocked state (R1, R3)
- [ ] Increment 2: Resolution path to a closed delivery (R2)
- [ ] Increment 3: Record and trim under 60s (R4)
- [ ] Increment 4: Embed on landing and README (R5)

---

## 6. Validation

### Deterministic checks

#### Test
- Command: `bash examples/demo/record.sh --verify`
- Scope: the scenario reaches the blocked state and then the closed state
- Expected: exit 0; the recorded output matches a live run

### Requirement trace
<!-- Filled at closeout. -->

---

## 7. Final Report

### Follow-ups

- [open]
