---
slug: pose-readme-evaluation-path
status: draft
created_at: 2026-09-06
completed_at:
supersedes:
depends_on: pose-canonical-positioning
priority: 2
components: docs
delivers:
---

# Spec: README as an evaluation path

## 1. Intent

### Goal
Make the README answer one question — "is this worth five minutes of my
time?" — and move everything that answers a different question into the docs.

### Business value
The README currently tries to be both the pitch and the manual. It carries a
"What's new in v1.4.3" section while the product is at 1.7.10, and hardcoded
counts ("10 roadmaps and 115 specs today") that were true against the v1.4.3
cycle and require manual upkeep to stay true.

A README that doubles as a changelog goes stale on every release by
construction. A README that doubles as a manual buries the decision the
reader is actually making. Both failures are visible on the first screen an
evaluator sees.

### Constraints
- English stays primary; `README.pt-BR.md` tracks it.
- Honesty about fit is preserved. A reader for whom POSE is too much structure
  should be able to reach that conclusion from the README.

### Non-goals
- Removing depth. Depth moves to the documentation, where it is reachable.

---

## 2. Requirements

### Functional
- R1: The README shall open with the canonical description, the lifecycle
  diagram and a concrete example, before any capability enumeration.
- R2: The "What's new in v1.<n>" section shall be removed and replaced by a
  pointer to Releases — the README shall carry no per-release content.
- R3: No quantitative claim shall be hardcoded. A count is either generated or
  restated qualitatively.
- R4: The README shall state the trade-off it makes, in terms of when POSE is
  the wrong choice, without framing other frameworks as a preliminary stage of
  POSE.
- R5: The closing statement shall present POSE as a standalone Apache-2.0
  project that also powers Harne8's governance — in that order.

### Compatibility
- `README.pt-BR.md` mirrors the same structure.

---

## 3. Technical Plan

### Affected areas
- `README.md`, `README.pt-BR.md`

### Artifacts
- modified: README.md
- modified: README.pt-BR.md

### Technical risks
- Cutting too far leaves an evaluator unable to judge depth. The 47 MCP tools,
  DORA, SARIF, SBOM and SLSA remain — as proof, in a section reached after the
  reader knows what POSE is, rather than as the opening claim.

---

## 4. Tasks

### Implementation
- [ ] Increment 1: Restructure the opening around the lifecycle (R1)
- [ ] Increment 2: Remove per-release content and hardcoded counts (R2, R3)
- [ ] Increment 3: Rewrite the fit and closing sections (R4, R5)
- [ ] Increment 4: Mirror into `README.pt-BR.md`

---

## 5. Decisions

### Decision 1
- Date: 2026-09-06
- Context: "47 MCP tools + 3 reporters" is currently a headline claim.
- Decision: keep the number, demote it to proof.
- Rationale: a first-time reader cannot tell whether 47 is impressive or
  intimidating, and the likelier reading is "I have to learn 47 things". The
  claim that does work is what the number enables: the agent does not have to
  be told the engineering process, it can query it.
- Consequences: the headline must carry the value on its own, without leaning
  on a countable.

---

## 6. Validation

### Deterministic checks

#### Lint
- Command: `pose docs-check`
- Expected: exit 0

#### Security / Contract
- Command: `pose public-claims --strict`
- Expected: exit 0 — the README asserts no stale version and no non-canonical
  docs host

### Requirement trace
<!-- Filled at closeout. -->

---

## 7. Final Report

### Follow-ups

- [open]
