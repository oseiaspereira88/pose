---
slug: pose-contract-adoption-registry
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-contract-adoption-stamp
priority: 1
components: pose-mcp
delivers:
---

# Spec: One mechanism for when an instance received a contract

## 1. Intent

### Goal
Collapse the accreting per-contract adoption dates into a registry, so a future
governance contract is stamped, reported and honoured by being declared once.

### Business value
Four dated markers now govern whether a completed closeout is judged by a rule
that did not exist when it was recorded:

| Field | Added because |
| --- | --- |
| `adopted_at` | POSE itself began governing this repository |
| `component_aware_adopted_at` | review plans became component-scoped |
| `review_bundles_adopted_at` | approval moved onto immutable sealed bundles |
| `evidence_vocabulary_reconciled_at` | a passed criterion must cite sealed evidence of a demanded class |

Each arrived the same way: a rule tightened, every closeout under the previous
rule started failing, and the answer was a new field plus a bespoke exemption
function that parses the date and checks the scope is done. The last one cost a
whole spec of its own (`pose-contract-adoption-stamp`) just to reach adopting
instances, because `.pose/policy/` is not machinery.

The cost is not the duplication. It is that the work is invisible until it
fails: a contract change ships, every adopting instance breaks on update, and
the operator sees a wall of failing closeouts about work nobody touched, with no
indication that a date is what is missing. That has now happened once with 106
closeouts in this repository, and would have happened again in the adopting one.

### Constraints
- Existing policies keep working untouched. A project that recorded
  `component_aware_adopted_at` must not have to migrate, and must not have its
  file rewritten into the new shape behind its back.
- The predicates stay per-contract. What "completed before" means genuinely
  differs — component-aware also requires the attempt to carry no plan digest,
  review-bundles compares the scope's creation date rather than the review's —
  and flattening them would be a worse abstraction than four fields.

### Non-goals
- Sealing the contract version into the bundle and validating against it. That
  is the deeper correct answer for the bundle path, and much larger; the
  registry delivers most of the value at a fraction of the cost.
- Migrating `adopted_at`, which is not a per-contract marker: it dates POSE's
  governance of the repository as a whole and is read by delivery and readiness,
  not by an exemption.

---

## 2. Requirements

### Functional
- R1: A registry shall declare the governance contracts whose rules judge work
  retroactively, each with an id, its legacy policy field, and a summary.
- R2: A policy shall record adoption dates in `contract_adoptions`, and a date
  in a legacy field shall continue to be read as that contract's adoption.
- R3: `pose update` shall stamp every registered contract the instance has never
  recorded, and shall not rewrite a legacy field into the map.
- R4: `pose doctor` shall report every registered contract with no recorded
  date, naming the contract and what it requires.
- R5: Adding a contract to the registry shall be sufficient for R3 and R4 —
  neither command shall need to learn about it.
- R6: Every registered contract's date shall be read through the registry
  wherever it is validated or consumed, not only where the map was introduced.
- R7: A legacy field explicitly set to empty shall be treated as a decision, and
  never stamped over.

### Non-functional
- The existing review, update and doctor suites pass.
- A policy key the engine does not model survives the stamp.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_closeout.go` — the registry, the lookup, and
  the shared half of the exemptions
- `pose-mcp/internal/cli/stack_seed.go` — the stamp, over the registry
- `pose-mcp/internal/cli/doctor.go` — the diagnostic, over the registry

### Artifacts
- created: .pose/specs/2026-09-08-pose-contract-adoption-registry.md
- renamed: .pose/changelogs/unreleased/pose-contract-adoption-registry.md -> .pose/changelogs/v2.0.0/pose-contract-adoption-registry.md
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/stack_seed_test.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_invisible_failures_test.go
- modified: pose-mcp/internal/pose/review_closeout_contract_test.go

### Technical risks
- Two places can now record the same contract's date. The map wins, and a
  legacy field is read when the map is silent, so the resolution is total and
  ordered — but an instance carrying both, disagreeing, is a confusing state
  nothing reports. It is visible in one file, which is why this is a risk rather
  than a blocker.
- The stamp writes through the raw JSON document rather than the typed policy,
  because marshalling the struct would drop any key the engine does not model.
  That is asserted, but it means the stamp does not benefit from the type.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Declare the registry and resolve a contract's adoption date (R1, R2)
- [x] Increment 2: Rewrite the exemptions onto the shared predicate (R2)
- [x] Increment 3: Stamp and report over the registry (R3, R4, R5)
- [x] Increment 4: Read every contract's date through the registry (R6)
- [x] Increment 5: Treat a cleared legacy field as a decision (R7)

### Validation
- [x] A contract added to the registry is stamped and reported with no other change

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: whether to unify the exemption predicates as well as the dates.
- Decision: no — only the date lookup and the "scope is done and the review
  predates the date" half are shared.
- Rationale: the three predicates differ in ways that are not incidental.
  Component-aware additionally requires the attempt to carry no plan digest,
  because an attempt that has one was recorded under component-aware planning
  and is not legacy however old it is. Review-bundles compares the scope's
  creation date rather than the review's. Forcing those into one signature would
  produce a function with three unrelated flags, which is harder to read than
  three functions that each say what they mean.

### Decision 2
- Date: 2026-09-08
- Context: what to do with the four existing fields.
- Options considered: (a) migrate them into `contract_adoptions` on update;
  (b) read both, map first, and never rewrite; (c) deprecate and warn.
- Decision: (b).
- Rationale: (a) rewrites a project's policy to say the same thing in a
  different shape, which is churn an operator has to review for no behavioural
  gain, and it would fight the additive-only contract that
  `pose-contract-adoption-stamp` was careful to respect. (c) spends the
  operator's attention on a migration that buys them nothing. Reading both is a
  few lines and makes the change invisible to every existing instance.
- Consequences: the legacy fields are permanent, and a new contract simply never
  gets one.

### Decision 4
- Date: 2026-09-08
- Context: review found the map-first contract was true for one contract and
  false for two. `loadReviewPolicy` still required and parsed
  `component_aware_adopted_at` and `review_bundles_adopted_at` directly, so a
  policy recording those dates only in `contract_adoptions` failed to load, and
  `reviewBundlesLegacyAttemptExempt` still read the legacy field.
- Decision: validate and consume every registered contract's date through
  `ContractAdoptedAt`.
- Rationale: a registry that advertises a shape working in one place out of
  three is worse than no registry — it invites a project to adopt the map and
  then fail to load. The rewrite covered the exemptions I happened to touch, not
  the ones the contract promised.

### Decision 5
- Date: 2026-09-08
- Context: an instance that deliberately empties a legacy field is asking for
  its whole history to be judged by the current contract. The typed policy
  renders that identically to an absent field, so the stamp wrote a real date
  into the map — and the map wins, silently reversing the choice on the update
  after.
- Decision: check the raw document for the legacy key's presence before
  stamping, and expose `LegacyContractField` so the caller can.
- Rationale: the distinction only survives in the raw JSON, which is also why
  the stamp already writes through the raw document rather than the struct. The
  previous spec was careful about a cleared value in the map and I carried that
  care to the map alone; the legacy field needed it too, and more, since it is
  the shape every existing instance has.

### Decision 3
- Date: 2026-09-08
- Context: proving R5. The first version of the doctor test listed the contract
  ids its fixture expected, so it asserted the registry's current contents
  rather than the property that the registry drives the report.
- Decision: build the fixture from `ReviewContracts()`.
- Rationale: adding a probe contract to the registry made that test fail while
  the behaviour was correct — the fixture, not the code, was wrong. It is the
  same shape as four other fixtures corrected in this session: a test that
  hard-codes an identifier the system is supposed to resolve passes whatever the
  system does with it, and fails for the wrong reason when the system changes.

---

## 6. Validation

### Strategy
Assert the resolution order and the stamp's boundaries directly, then prove R5
by adding a contract that has no code anywhere else and checking both commands
handle it.

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
- Notes: all eight packages pass, including every pre-existing review test with
  the three exemptions rewritten onto the shared predicate — which is the
  evidence that the rewrite preserved their behaviour. R5 was proven by adding
  a `future-contract-probe` entry with no code referring to it: `pose update`
  stamped it and `pose doctor` named it, with no change to either command. That
  probe also exposed the hard-coded fixture in Decision 3, which was corrected
  before the probe was removed.
- Review then found the map-first contract held for one contract out of three,
  and that a legacy field deliberately set to empty was stamped over — reversing
  the instance's own choice. Both are asserted, and both assertions fail against
  the previous code: `pose: review-bundles adoption date must be YYYY-MM-DD` on
  a policy that records it only in the map, and `stamped 2026-09-08 over a
  deliberately cleared legacy field`.

### Results summary
- Successes: R1, R2, R3, R4, R5, R6, R7 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <ReviewContract and reviewContracts in review_closeout.go, exposed by ReviewContracts()>
- R2 [satisfied] <ContractAdoptedAt reads contract_adoptions first and falls back to the contract's legacy field; asserted by the doctor test recording dates through the map and by the stamp test's legacy-field subtest>
- R3 [satisfied] <stampContractAdoption iterates ReviewContracts() and skips anything ContractAdoptionRecorded reports, so a legacy field is never rewritten into the map; six subtests cover stamped, legacy, existing, cleared, unmodelled keys and malformed>
- R4 [satisfied] <doctor's review.contract-adoption iterates the registry and names each unrecorded contract with its summary>
- R5 [satisfied] <a probe contract with no other code was stamped and reported; neither command references a contract id>
- R6 [satisfied] <loadReviewPolicy validates the component-aware and review-bundles dates through ContractAdoptedAt, and reviewBundlesLegacyAttemptExempt consumes it the same way; TestPolicyRecordingDatesOnlyInTheMapIsValidAndHonoured loads a policy carrying no legacy field and asserts all three resolve, plus that the legacy field is still read when the map is silent and loses when both are present>
- R7 [satisfied] <stampContractAdoption checks the raw document for LegacyContractField before stamping; TestStampContractAdoptionRespectsAClearedLegacyField asserts the cleared contract is skipped while a contract the policy says nothing about is still stamped>

### Known gaps
- Only the three git fixtures in `review_bundle_test.go` disable auto gc. Every
  other test package that inits a repository has the same exposure, and a
  package-level default would cover them; that is a wider change than this spec
  should carry.
- The error text for an invalid adoption date no longer names the legacy key,
  since the value may have come from either place. It names the contract
  instead, which is the thing to look up.
- `adopted_at` stays outside the registry. It is not a per-contract marker — it
  dates POSE's governance of the repository and is read by delivery and
  readiness rather than by an exemption — but that means one of the four dates
  an operator sees still works differently from the others.
- Nothing reports an instance whose map and legacy field disagree about the same
  contract. The map wins, which is defined, but the disagreement is silent.

---

## 7. Final Report

### Follow-ups

- [open] Seal the governing contract version into the review bundle and validate against it, rather than dating exemptions in policy — owner:unowned crit:medium review:2027-01-08
- [open] Report a policy whose contract_adoptions and legacy field disagree about the same contract — owner:unowned crit:low review:2026-12-08
- [open] Disable git auto gc for every test fixture that inits a repository, not only the three that flaked — owner:unowned crit:low review:2026-12-08
