---
slug: pose-contract-adoption-stamp
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-attestation-evidence-must-be-in-the-bundle
priority: 0
components: pose-mcp
delivers:
---

# Spec: An instance records when it received a governance contract

## 1. Intent

### Goal
Make the exemption that keeps completed closeouts approved actually reach the
instances that need it, and say so when it has not.

### Business value
`pose-attestation-evidence-must-be-in-the-bundle` added
`evidence_vocabulary_reconciled_at` so that a closeout recorded before a rule
existed is not judged by it. Without that date, running the new engine against
POSE's own repository reported **106** failing closeouts; with it, none.

The date reaches nobody. `.pose/policy/` is not a machinery root — machinery is
`.pose/workflows`, `.pose/rules`, `.pose/templates`, `.agents/skills` — so an
update delivers the stricter engine and never the statement of when the instance
received it. The date is in POSE's own policy only because it was added by hand
in the same change.

The adopting repository is already in that state. Its policy carries three
adoption markers and not this one:

```
"adopted_at": "2026-08-23"
"component_aware_adopted_at": "2026-08-23"
"review_bundles_adopted_at": "2026-08-11"
```

So on the day it takes the release, every historical closeout fails, and an
agent working in that repository sees dozens of errors about work nobody
touched. That is not a governance signal; it is noise that trains the reader to
skim the gate.

### Constraints
- Additive only. `pose update` never overwrites instance-owned configuration,
  and this must not be the exception: a date the instance chose, including one
  it deliberately cleared, survives every update.
- The stamp must be visible. Writing to a project's policy silently would be
  worse than not writing at all.

### Non-goals
- Generalising the four adoption markers into one mechanism. That is the right
  shape and is the next spec; this one has to ship with the release, and the
  general form is a larger change to how a completed closeout is judged.

---

## 2. Requirements

### Functional
- R1: `pose update` shall record `evidence_vocabulary_reconciled_at` in the
  instance's review policy when the key is absent, using the date of the update.
- R2: An existing value, including an explicitly empty one, shall be left
  untouched.
- R3: A policy the engine cannot parse shall not be rewritten.
- R4: The stamp shall be reported in the update's output.
- R5: `pose doctor` shall report an instance that has recorded reviews and no
  reconciliation date, naming the key and how to set it, counting reviews in
  both the legacy markdown and the sealed-bundle storage.
- R6: A `pose update` that replaces the binary shall hand off to it, so a
  migration shipped in the new release applies on the update that delivers it.
- R7: A review recorded earlier on the adoption day shall be exempt.

### Non-functional
- The existing update and doctor suites pass.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/stack_seed.go` — the stamp, beside the existing
  additive policy migration
- `pose-mcp/internal/cli/doctor.go` — the diagnostic
- `pose-mcp/internal/cli/maintenance.go` — the handoff to the replaced binary
- `pose-mcp/internal/pose/review_closeout.go` — the adoption cutoff

### Artifacts
- created: .pose/specs/2026-09-08-pose-contract-adoption-stamp.md
- renamed: .pose/changelogs/unreleased/pose-contract-adoption-stamp.md -> .pose/changelogs/v2.0.0/pose-contract-adoption-stamp.md
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/stack_seed_test.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_invisible_failures_test.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/pose/review_closeout.go
- created: pose-mcp/internal/pose/review_closeout_contract_test.go

### Technical risks
- The stamp waives a rule for everything reviewed before the update. Someone who
  updates a year late waives a year of reviews — which is correct, that work was
  reviewed under the previous contract, but it means the date must be the update
  date and not something later.
- Rewriting the policy reformats it, because the engine parses and re-marshals
  rather than editing text. The existing v1→v2 migration already does this, so
  it is the established behaviour rather than a new surprise.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Stamp the date on update when absent, and report it (R1, R2, R3, R4)
- [x] Increment 2: Report an instance without the date that has history (R5)
- [x] Increment 3: Hand off to the replaced binary so the migration runs (R6)
- [x] Increment 4: Make the adoption day inclusive (R7)

### Validation
- [x] Each behaviour asserted, and the wiring confirmed end to end

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: which date to record.
- Options considered: (a) the instance's `adopted_at`; (b) the engine version's
  release date; (c) the date of the update.
- Decision: (c).
- Rationale: (a) is what the v1→v2 migration falls back to, and it is wrong
  here: reviews recorded between adopting POSE and receiving this contract were
  judged by the previous contract, and dating the exemption at `adopted_at`
  would fail all of them — the exact failure this spec exists to prevent. (b)
  is closer but still wrong for an instance that updates late, since the
  contract reached it when it updated, not when the version was published.
  (c) is the only one that states a fact about this instance.

### Decision 2
- Date: 2026-09-08
- Context: whether writing into an instance-owned policy file is acceptable at
  all, given `pose update` is additive-only by contract.
- Decision: yes, bounded to an absent key, and reported.
- Rationale: adding a key that is not there is additive in the same sense
  `seedModuleMetadataFromDiscovery` is — it merges discovered modules into an
  existing file and never touches an existing entry. The alternative is to leave
  every adopting instance broken until its operator finds an undocumented field,
  which is the situation the previous spec created. An explicitly empty value is
  treated as a decision, not an absence, so a project that wants its whole
  history judged by the current contract says so by clearing the field and the
  update respects it.

### Decision 3
- Date: 2026-09-08
- Context: review pointed out that `pose update` replaces the executable and
  then keeps running the old process, so the stamp shipped in a release does
  not happen on the update that delivers it — the instance runs the stricter
  engine against all its history until someone updates a second time.
- Options considered: (a) accept it and document that `pose update` must be run
  twice; (b) re-exec the replaced binary to finish the update.
- Decision: (b), forwarding the original arguments with `--no-self`.
- Rationale: (a) is a footgun aimed at exactly the moment this spec exists to
  protect, and no adopter would know to do it. (b) also fixes the general case:
  every future migration now lands on the update that ships it, not the one
  after. `--no-self` on the handoff makes termination explicit rather than
  relying on the new process's own version check agreeing it is current.

### Decision 4
- Date: 2026-09-08
- Context: the stamp is a date, and the exemption compared the review timestamp
  against its midnight. A review at 09:00 followed by an update at 15:00 the
  same day was not exempt.
- Decision: the cutoff is the end of the stamped day.
- Rationale: the date names the day the contract reached the instance, and work
  reviewed earlier that day was reviewed before it arrived. The alternative
  fails exactly the work most likely to be affected — the reviews closest to the
  update — while the cost is waiving a few hours of same-day reviews recorded
  after it, which is the conservative direction. Applied to the
  component-aware exemption too, since it reads the same kind of date.

---

## 6. Validation

### Strategy
Unit-test the four branches of the stamp and the three of the diagnostic, then
confirm the wiring by running the real command against a real instance — a unit
test of the function would pass even if nothing called it.

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
- Notes: all eight packages pass. Disabling the doctor branch fails its test on
  "expected a review.contract-adoption finding". The stamp's own test calls the
  function directly and would pass even unwired, so the wiring was verified
  separately: `pose install` into a fresh repository, the key removed by hand,
  then `pose update --no-self` printed
  `[INFO] policy (contract adoption): evidence_vocabulary_reconciled_at=2026-09-08`
  and the key was present afterwards.
- Review then found two things the suite had been green through. The diagnostic
  globbed `.pose/review-attempts/`, which nothing in the repository writes:
  `Store.ListReviewAttempts` reads `.pose/reviews/*.md`. So the pre-bundle
  instance the check exists to diagnose always counted zero and got an `ok` —
  and its test passed because the fixture seeded an attestation, exercising the
  one path that did work. Restoring the wrong directory now fails the test on
  `level="ok", want warn`. Reverting the day-inclusive cutoff fails the same-day
  case on `exempt=false, want true`.

### Results summary
- Successes: R1, R2, R3, R4, R5, R6, R7 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <stampContractAdoption is called from seedAbsentInstanceConfig with time.Now().UTC(); the absent branch is asserted, and the end-to-end run above confirms the call site>
- R2 [satisfied] <two subtests: an existing date is unchanged, and an explicitly empty value is not re-stamped>
- R3 [satisfied] <a truncated policy is left byte-identical, asserted>
- R4 [satisfied] <the log line names the key and the date, and was observed in the real run>
- R5 [satisfied] <doctor's review.contract-adoption warns with recorded reviews and no date, reports ok once it is set, and stays ok on an instance with no history; the test drives it through a legacy `.pose/reviews/*.md` attempt and again through a sealed attestation, so neither storage shape is the only one that counts>
- R6 [satisfied] <performSelfUpdate reports whether it replaced the executable, and cmdUpdate hands off to it with --no-self, returning its exit code>
- R7 [satisfied] <reviewPredatesAdoption compares against the end of the stamped day; four cases assert the day before, earlier and later the same day, and the next day, and three more assert an absent, unparseable or wrongly-formatted date exempts nothing>

### Known gaps
- The handoff runs the new binary as a child process rather than replacing this
  one, so the update's output is relayed rather than emitted directly. That is
  the portable choice; `syscall.Exec` is not available on Windows.
- The stamp covers one contract. The next contract change needs another field,
  another stamp and another diagnostic — which is the argument for the general
  mechanism rather than a defect in this one.
- Nothing verifies that a project's chosen date is plausible: a date set far in
  the future waives everything. It is visible in the policy file and in
  `doctor`'s ok message, which is the same exposure the three existing adoption
  markers carry.

---

## 7. Final Report

### Follow-ups

- [done] Collapse the four dated adoption markers into one contract-adoption mechanism. Delivered by `pose-contract-adoption-registry`: `ReviewContracts()` is the one mechanism, and adding a contract there is what makes `pose update` stamp it and `pose doctor` report it. The legacy top-level fields stay on disk on purpose — `stampContractAdoption` prefers them so an engine predating the registry can still read the policy — which is a compatibility encoding, not a second mechanism. — owner:unowned crit:high review:2026-11-08
