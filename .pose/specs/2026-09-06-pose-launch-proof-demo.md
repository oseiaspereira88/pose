---
slug: pose-launch-proof-demo
status: in-progress
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
- created: examples/demo/record.sh
- modified: .github/workflows/ci.yml

### Technical risks
- A scenario contrived to fail is unconvincing. The blocked state should be
  one POSE produces routinely — an undispositioned follow-up is both the most
  common and the most explicable.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Scripted scenario reaching a genuine blocked state (R1, R3)
- [x] Increment 2: Resolution path to a closed delivery (R2)
- [ ] Increment 3: Record and trim under 60s (R4)
- [ ] Increment 4: Embed on landing and README (R5)

---

## 6. Validation

### Deterministic checks

#### Test
- Command: `bash examples/demo/record.sh --verify`
- Scope: the scenario reaches the blocked state and then the closed state
- Expected: exit 0; the recorded output matches a live run

### Execution log
- Date: 2026-09-07
- Environment: local Linux, Go 1.26.5
- Notes: the scenario builds a real Go module with a genuinely passing test,
  so the green checks are green rather than staged. `go test`, `go vet`,
  `go build` and `pose validate --strict` all pass; `pose lint-spec --strict`
  then refuses to close the delivery because R1 has no requirement-trace
  entry. The script fails loudly if the closeout gate ever *passes* at that
  point — an all-green scenario would make a recording that argues against the
  product.

### Results summary
- Successes: R1, R2, R3.
- Failures: none.
- Warnings: R4 and R5 need a recording and a page edit — see Known gaps.

### Requirement trace
- R1 [satisfied] <examples/demo/record.sh; check:demo-scenario-verify asserts the block and its reason>
- R2 [satisfied] <same script, resolution step; asserts spec.trace.missing=0 afterwards>
- R3 [satisfied] <the scenario is a script in the repository, re-runnable and re-recordable; check:ci-demo-scenario>
- R4 [deferred-integration: spec:pose-launch-proof-demo] <the recording itself is a capture step, not code>
- R5 [deferred-integration: spec:harne8-pose-launch-surfaces] <landing embedding belongs to the site repository>

### Known gaps
- No recording exists yet. Producing one means pointing asciinema, vhs or a
  screen capture at the non-`--verify` run, which is paced for exactly that.
  Keeping the scenario as a script rather than a checked-in video means it can
  be re-recorded when output changes instead of silently becoming a stale
  artifact — but it also means the artifact the launch actually needs is not
  done until someone records it.
- Embedding on the landing page is governed by the site repository's spec.

---

## 7. Final Report

### Follow-ups

- [open]
