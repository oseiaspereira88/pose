---
slug: pose-seal-names-carried-forward-evidence
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-closeout-regenerates-before-sealing
priority: 1
components: pose-mcp
delivers:
---

# Spec: Sealing says when evidence did not observe the change

## 1. Intent

### Goal
Name, at seal time, the evidence that ran against a commit other than the head
the bundle approves.

### Business value
`deliveryEvidenceCurrent` decides what counts as current, and it accepts a
result whose provenance digest matches the graph or whose scope is already
closed. That is right: re-running every check to re-seal a finished spec would
be theatre, and the closed-scope bridge exists because an immutable closeout
cannot retroactively acquire a newer field.

What it does not do is say so. A reviewer reading a sealed bundle sees a list of
passed checks and cannot tell a result produced against this subject from one
carried forward from an earlier commit. That distinction is the whole question
an attestation answers — did this evidence observe this change — and it was
recoverable only by comparing two commit hashes nobody compares.

`pose-closeout-regenerates-before-sealing` documented the sequence that avoids
carrying evidence forward and recorded, in its own known gaps, that
documentation is the weakest form of the fix: skipping the step does not fail,
so nothing tells the operator it happened.

### Constraints
- A warning, never a blocker. The engine already decided this evidence is
  current; this says how it is current, and failing on it would refuse the
  closed-scope case the bridge exists for.

### Non-goals
- Deciding whether carried-forward evidence is acceptable. That is the
  reviewer's judgement, and it is judgement they could not previously make.

---

## 2. Requirements

### Functional
- R1: Sealing shall warn when sealed evidence records a git head other than the
  subject's, naming each such result and the subject head.
- R2: It shall not warn when every result observed the subject head.
- R3: It shall not block.

### Non-functional
- No change to what counts as current.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go` — the seal-time warning

### Artifacts
- created: .pose/specs/2026-09-09-pose-seal-names-carried-forward-evidence.md
- created: .pose/changelogs/unreleased/pose-seal-names-carried-forward-evidence.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go

### Technical risks
- A result with no recorded git head is not compared, so evidence from a
  producer that does not stamp one is silently treated as observing the subject.
  Warning on an absent head would fire on every legacy result and teach the
  reader to skim.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Warn on evidence whose head differs from the subject's (R1, R3)

### Validation
- [x] Silent when every result observed the subject, and the fixture fails if that is untrue

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: warning versus blocking.
- Decision: warning.
- Rationale: blocking would refuse exactly the case `deliveryEvidenceCurrent`'s
  closed-scope bridge exists to allow — an immutable closeout whose evidence
  cannot be regenerated against a head that moved past it. The reviewer needs to
  know, not to be stopped; what was missing was the telling.

---

## 6. Validation

### Strategy
Seal with one result moved off the subject head and assert it is named, then
seal untouched and assert silence — with the second test failing its own fixture
if the results do not in fact observe the head, so silence cannot come from
there being nothing to say.

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
- Notes: all eight packages pass. Removing the call leaves the warning test with
  no warnings to find. The silent-case test asserts up front that the fixture's
  evidence really does record the subject head, so it cannot pass by the
  comparison never running.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <staleEvidenceWarnings compares each result's GitHead against Subject.Head and names the differing ones with the short subject head; TestSealWarnsWhenEvidenceRanAgainstAnotherCommit moves one result's head and asserts it is named>
- R2 [satisfied] <TestSealIsSilentWhenEvidenceObservedTheSubject fails its fixture if any result's head differs, then asserts no such warning>
- R3 [satisfied] <the same warning test asserts the evidence does not appear among the blockers>

### Known gaps
- Evidence with no recorded git head is not compared, so a producer that does
  not stamp one is treated as having observed the subject.
- The warning fires at seal time only. An attestation recorded later against a
  bundle sealed earlier repeats nothing.

---

## 7. Final Report

### Follow-ups

- [open] Carry the carried-forward warning into review verify, so it is visible when reading an attestation rather than only when sealing — owner:unowned crit:low review:2027-01-09
