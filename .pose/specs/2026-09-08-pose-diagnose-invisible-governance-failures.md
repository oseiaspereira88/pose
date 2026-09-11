---
slug: pose-diagnose-invisible-governance-failures
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
delivers:
---

# Spec: Make invisible governance failures diagnosable

## 1. Intent

### Goal
Surface the two failure modes where POSE silently discards something it was
given, so that the operator learns the cause from POSE instead of from reading
the engine's source.

### Business value
A single adopting repository hit five blockers in one working session. Every
one was eventually fixable in minutes; finding the cause took hours, and in
four of the five it required reading this engine's Go source:

| Blocker | How the cause was found | Fix |
| --- | --- | --- |
| Change set spanning a whole branch | auditing three weeks of commit messages | rebase with correct slugs |
| Change set inflated by a stale ref | root cause was recorded **wrong twice** | one `git update-ref -d` |
| Evidence classes no check may emit | reading `review_plan.go` | edit a profile |
| "no immutable attributed change set" | reading `review_bundle.go` | run `pose index` |
| Submodule fails the subject classifier | reading `review_bundle.go` | ~30 lines |

The asymmetry is the finding: POSE's gates are sound — they refused a spec whose
requirement was claimed without evidence, refused to seal against a dirty tree,
and caught a requirement trace written as prose. What costs the operator is not
satisfying a gate. It is discovering why the gate failed.

Two causes recur underneath, and both are the same shape: POSE reduces
what it knows and reports a downstream symptom instead of the upstream loss.

1. **Two evidence vocabularies never checked against each other.** The
   validation matrix accepts nine classes; the shipped review profiles reference
   nine; three intersect. A profile can demand a class no check is permitted to
   emit, and nothing reports it until a bundle resolves `evidence=0`.
2. **A diagnostic that hides its own inputs.** `artifact-check` reports
   `undeclared <path>` and advises narrowing the attributed change set, without
   naming the change set, its base, or how many commits it spans. Those exist
   only under `--json`. When the base is a commit no branch points at — the
   `refs/original/` backup `git filter-branch` leaves behind — the default
   output gives the operator nothing to pull on.

### Constraints
- Diagnose, do not decide. A doctor check reports; it never edits an instance.
- No check may fail closed on a condition an operator cannot act on.
- A diagnostic must reflect how the engine actually behaves, not how an
  adjacent tool behaves. The first version of this spec proposed a third check
  for `POSE-Spec:` lines Git does not parse as trailers; it was dropped on
  finding that POSE never uses Git's trailer parser and reads the line
  regardless, so the check would have warned about a condition that costs the
  engine nothing.

### Non-goals
- Removing `pose review auto-attest`. That command fabricates evidence for
  criteria whose classes have no producer, and while it exists the distinction
  between a judged review and a stamped one collapses. Removing it deletes a
  public command, which is a product decision, not a diagnostic one. Recorded as
  a follow-up.
- Cross-validating criterion evidence classes at plan time. The tool side was
  fixed in `pose-review-plan-producible-evidence-classes`; the criterion side
  does not block a manual attestation, and folding it in here would widen a
  diagnostic change into a contract change.

---

## 2. Requirements

### Functional
- R1: `pose doctor` shall report any evidence class referenced by an active
  review profile that no registered check is permitted to emit, naming the
  class and the profile.
- R2: `pose doctor` shall report any check in the validation matrix that
  declares no `evidenceClass`, since its results are discarded when evidence is
  collected for a review.
- R3: `pose artifact-check` shall print the resolved change set's base, head and
  commit count in its default output, not only under `--json`.

### Non-functional
- Each diagnostic degrades to silence when the artifact it inspects is absent,
  never to a false negative reported as health.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor.go` — two new diagnostics
- `pose-mcp/internal/cli/artifact_integrity.go` — change-set provenance in the
  default output

### Artifacts
- created: .pose/specs/2026-09-08-pose-diagnose-invisible-governance-failures.md
- renamed: .pose/changelogs/unreleased/pose-diagnose-invisible-governance-failures.md -> .pose/changelogs/v1.8.0/pose-diagnose-invisible-governance-failures.md
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_test.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/artifact_integrity_test.go

### Technical risks
- Both doctor checks warn rather than error. An instance can legitimately carry
  a profile it does not use, or a check whose results feed no review, and
  failing closed on either would make `doctor` unusable in exactly the
  repositories that most need to run it.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Report unproducible evidence classes in active profiles (R1)
- [x] Increment 2: Report matrix checks with no evidence class (R2)
- [x] Increment 3: Print change-set provenance by default (R3)
- [x] Increment 4: Report only the profiles policy selects (R1, after v1.8.0)

### Validation
- [x] Each diagnostic red on a fixture that reproduces the failure, green otherwise

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: where the two diagnostics belong.
- Options considered: (a) a new command; (b) `pose doctor`; (c) inline warnings
  at each gate.
- Decision: (b).
- Rationale: `doctor` already answers "is this instance healthy", already checks
  hooks, policy roots and change-set attribution, and is the one command an
  operator runs when something is wrong and they do not yet know what. A new
  command would need to be discovered first, which is the problem being solved.
  (c) is right for a failure the gate itself can see — the tool-class drop
  warning added in 1.7.12 is exactly that — but these two are properties of
  the instance, not of one gate's run.

### Decision 2
- Date: 2026-09-08
- Context: this spec opened with a third diagnostic, for `POSE-Spec:` lines Git
  does not parse as trailers. Writing it meant reading how the engine reads that
  line, and the engine does not use Git's parser at all:
  `allCommitsWithSpecTrailers` scans the message for the prefix, and nothing in
  the distribution calls `%(trailers` or `interpret-trailers`.
- Decision: drop the check before shipping it.
- Rationale: the condition costs the engine nothing. Warning about it would have
  taught operators that a harmless format breaks attribution — the same false
  belief that produced this check, now shipped as a diagnostic and therefore
  authoritative. A diagnostic that misattributes cause is worse than a missing
  one, because it directs the investigation.
- Consequences: the misreading it was built on had already reached a decision
  log, a merged pull request and a published review reply in the adopting
  repository, and was retracted there. The rule that survives: before diagnosing
  that a tool lost data, confirm which path it reads that data through.

---

## 6. Validation

### Strategy
Every diagnostic must be red on a fixture that reproduces the real failure and
green on one that does not, so a check that observes nothing cannot pass by
accident.

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
- Date: 2026-09-08
- Environment: local, Go 1.26
- Notes: all eight packages pass. Each of the three changes was reverted in turn
  to confirm its test fails without it — filtering disabled makes
  `review.evidence-vocabulary` miss the unproducible class, the class check
  short-circuited makes `validate.evidence-class-coverage` miss the unclassed
  one, and dropping the provenance fields makes the artifact-check assertion
  fail. Both doctor checks are asserted green on a fixture that does not
  reproduce the failure, so neither can pass by observing nothing.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <doctor.go: review.evidence-vocabulary, scoped to the profiles .pose/policy/review.json selects; tests assert warn on a selected profile naming a class outside ValidEvidenceClasses, ok when every class is producible, and ok when the offending profile is on disk but unselected>
- R2 [satisfied] <doctor.go: validate.evidence-class-coverage; test asserts warn on a matrix check with no evidenceClass and ok otherwise>
- R3 [satisfied] <artifact_integrity.go prints artifact.change_set.base/head/commits by default; test asserts the fields appear without --json>

### Known gaps
- Closed after v1.8.0 shipped: `review.evidence-vocabulary` reported every
  profile on disk, so the first real run against an adopting instance flagged
  the four shipped profiles that instance had already replaced — noise the
  operator has to investigate to dismiss, and precisely the shape a project
  lands in when it owns its profiles and leaves the originals in place. The
  check now reads `.pose/policy/review.json` and inspects only the profiles
  policy selects. `validate.evidence-class-coverage` still reads what is
  declared rather than what runs, which is intended: a check with no class is
  discarded wherever it is used.
- The criterion side of the evidence vocabulary is reported but not enforced
  anywhere: a criterion naming an unproducible class still drives
  `pose review auto-attest` to invent a ref. The diagnostic makes it visible;
  the follow-up is what would close it.

---

## 7. Final Report

### Follow-ups

- [done] Decide the fate of pose review auto-attest, which fabricates evidence for criteria whose classes have no producer. Decided: it stays and refuses. `AutoAttestReviewBundle` no longer invents a reference for either criteria or tools — where the scope carries validation evidence it errors and names what to run, and where it does not it records not-applicable with the reason. The fabrication the question was about is gone. (owner:unowned crit:high review:2026-10-08)
