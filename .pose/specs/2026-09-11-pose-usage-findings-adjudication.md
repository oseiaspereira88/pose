---
slug: pose-usage-findings-adjudication
status: draft
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-usage-metrics
priority: 0
components: pose-mcp
task_type: feature
delivers:
---

# Spec: A person can say whether a usage finding was real

## 1. Intent

### Goal
Let a person record, per finding `pose usage` observed, whether it was `valid`,
`wont-fix` or a `false-positive`, and report those verdicts beside — never in
place of — the automatic observation counts.

### Business value
`pose usage` answers which tools ran and what they found: runs with findings,
unique, new, resolved and reopened findings per tool. It cannot answer whether
those findings were worth finding. A gate that fires often looks valuable; one
that fires often and is ignored every time is noise, and today the two read the
same. The adjudicated false-positive rate per tool is the number that separates
them, and it is what a decision to tighten, loosen or retire a gate needs.

The follow-up asking for this was written when `pose-usage-metrics` closed
(2026-08-10) and was reviewed as overdue on 2026-09-11. It was deferred then,
not because it is unimportant, but because it needs one design decision first —
recorded below — and a patch release is the wrong place to make it.

### Constraints
- Automatic observations stay as they are; a verdict is a separate fact about
  them and never rewrites one (ADR `local-usage-events-and-outcome-aware-aggregation`).
- No individual productivity signal: a verdict may name who gave it, but no
  report aggregates by person.
- Usage events stay outside the worktree and keep hashing finding identities.

### Non-goals
- Adjudicating findings from gates that emit one conservative observation
  without a stable identity; there is nothing to point at.
- Feeding verdicts back into a gate's behaviour. Suppressing a finding is a
  policy change, made where the policy lives.

---

## 2. Requirements

### Functional
- R1: A command shall record a verdict — `valid`, `wont-fix` or `false-positive`
  — for a finding named by its tool and stable finding identity, with a reason.
- R2: `pose usage` and `pose_usage` shall report, per tool, adjudicated counts
  and the false-positive rate among adjudicated findings, separately from the
  automatic counts.
- R3: A verdict shall be append-only: a later verdict supersedes an earlier one
  for the same finding, and both remain in the record.
- R4: A verdict for a finding no local event has observed shall be kept and
  reported as unmatched, not dropped.

### Non-functional
- The journal is readable and reviewable without POSE.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/usage` — verdict storage and the join at aggregation
- `pose-mcp/internal/cli` — the recording command
- `pose-mcp/internal/mcpserver` — `pose_usage` report shape

### Artifacts
- created: .pose/specs/2026-09-11-pose-usage-findings-adjudication.md
- modified: .pose/specs/2026-08-10-pose-usage-metrics.md

### Technical risks
- Finding identities are persisted only as HMACs under a per-machine salt. A
  verdict keyed by that HMAC does not match on another machine; a verdict keyed
  by the raw identity puts that identity somewhere the usage ADR chose not to.
  Decision 1 is about exactly this.

---

## 4. Tasks

### Implementation
- [ ] Increment 1: Decide storage and identity; write the ADR (Decision 1)
- [ ] Increment 2: Record verdicts (R1, R3)
- [ ] Increment 3: Report them beside the automatic counts (R2, R4)

### Validation
- [ ] A verdict recorded on one machine matches the same finding observed on another

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: where verdicts live and how they name a finding. The usage ADR keeps
  events outside the worktree, per machine, with finding IDs hashed under a
  local salt, because IDs can carry module or file paths.
- Options:
  - A. A local verdict journal beside the events, keyed by the HMAC. Private and
    consistent with the ADR, but per machine: a team never shares a verdict, CI
    never sees one, and a new clone starts from nothing.
  - B. A tracked, append-only journal in the repository, keyed by tool and raw
    finding identity, with verdict, reason, `by:` alias and date. Shared and
    reviewable in pull requests like a follow-up disposition; each machine joins
    it to its own events by hashing the raw identity with its own salt at read
    time. The identities it records are check and finding IDs the repository
    already contains.
  - C. Verdicts as governed artefacts that already exist — a decision-log or an
    accepted risk per finding. No new store, but heavy for a per-finding
    judgement and awkward to aggregate.
- Recommendation: B. A verdict is a governance decision, which belongs where the
  team can see and review it; the privacy concern behind hashing was persisting
  IDs outside review, which B does not do.
- Status: pending — the owner decides, then the ADR is written, before Increment 2.

---

## 6. Validation

### Strategy
Record verdicts against findings observed by real runs, on two machines or two
salts, and require the report to match and separate them.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

### Execution log
- Date: 2026-09-11
- Environment: not started
- Notes: draft; nothing implemented.

### Results summary
- Successes: none yet.
- Failures: none.

### Requirement trace
- R1 [waived: draft, not implemented] <pending Decision 1>
- R2 [waived: draft, not implemented] <pending Decision 1>
- R3 [waived: draft, not implemented] <pending Decision 1>
- R4 [waived: draft, not implemented] <pending Decision 1>

### Known gaps
- Everything; this spec records the intent and the decision it needs.

---

## 7. Final Report

### Summary
Deferred from 5.0.2 with its open decision stated.

### Follow-ups
