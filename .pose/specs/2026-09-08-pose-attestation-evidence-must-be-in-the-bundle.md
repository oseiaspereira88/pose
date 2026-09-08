---
slug: pose-attestation-evidence-must-be-in-the-bundle
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-review-plan-producible-evidence-classes
priority: 0
components: pose-mcp
delivers:
---

# Spec: A passed criterion must be supported by the bundle it approves

## 1. Intent

### Goal
Tie a criterion's `passed` disposition to evidence the sealed bundle actually
contains, of a class the criterion asks for — and stop `auto-attest` inventing a
reference when there is none.

### Business value
An attestation is a claim that criteria were judged against a subject. Nothing
connected the claim to the subject. `validateBundleAttestation` checked that
every required criterion appeared, that dispositions were spelled correctly,
that `not-applicable` carried a rationale — and never once looked at
`criterion.Evidence`. A `passed` could cite evidence absent from the bundle,
evidence of a class the criterion never asked for, or nothing at all, and the
attestation verified.

Three closeouts in an adopting repository were approved that way and reported
clean by the gate:

| Criterion | Class the plan required | What the attestation cited |
| --- | --- | --- |
| `backend-integration-impact` | `integration` | a `build` result from `go vet` |
| `frontend-accessibility` | `a11y` | an `e2e` result |
| `frontend-surface-reachability` | `reachability` | the same `e2e` result |

One of the three bundles sealed **zero** evidence while its attestation cited
two references. The other two carried only evidence from an unrelated component.

`auto-attest` is the other half. When the bundle sealed nothing of a required
class it wrote `<class>:auto-attest` — a reference to nothing — or reached for
the first unrelated ref in the bundle, or fell back to `docs:auto-attest`. Since
no consumer checked the reference, the command reported a passed criterion
supported by a string. That is what collapses the distinction between a judged
review and a stamped one.

Underneath both sits a third problem that makes the first two unavoidable: the
shipped profiles demand `test` and `validation`, and neither is in
`ValidEvidenceClasses`. No registered check may emit them, so those criteria
could **only ever** be satisfied by something invented. `pose-review-plan-
producible-evidence-classes` fixed exactly this for tools in 1.7.12 and left the
criterion side open, noting it "does not block a manual attestation". It does
block this one: without it, every plan built from a shipped profile becomes
unsatisfiable the moment evidence is actually required.

### Constraints
- The three dispositions keep their meanings. Only `passed` is constrained;
  `not-applicable` already requires a rationale, which is the reviewer's
  judgement standing in for evidence, and `finding` records a problem rather
  than a clearance.
- A documentation-only scope with no delivery target must stay attestable. The
  engine already draws that line for the bundle as a whole
  (`reviewScopeRequiresValidationEvidence`); this reuses it rather than
  inventing a second rule.

### Non-goals
- Removing `pose review auto-attest`. It stops fabricating, which is what made
  it dangerous; whether the command should exist is a product decision, and it
  is now honest either way.
- Reconciling any specific project's profiles. The engine stops planning
  unsatisfiable criteria; which classes a project's checks emit is its own work.

---

## 2. Requirements

### Functional
- R1: A criterion recorded `passed` whose evidence reference is not present in
  the sealed bundle shall block verification, naming the criterion and the
  reference.
- R2: A criterion recorded `passed` whose evidence is of a class the criterion
  does not ask for shall block verification, naming the required classes and the
  class cited.
- R3: A criterion recorded `passed` with no evidence at all shall block
  verification.
- R4: `auto-attest` shall not synthesise an evidence reference. Where the scope
  is expected to carry validation evidence it shall refuse, naming the criterion,
  the missing class and the way out; where it is not, it shall record the
  criterion `not-applicable` with a rationale naming what is absent.
- R5: A criterion demanding an evidence class no registered check may emit shall
  block the plan, naming the criterion, the class and the profile. The class
  shall not be erased.
- R6: The review profiles shipped with POSE shall demand only classes a
  registered check may emit.
- R7: A completed scope whose approval was recorded before a dated
  `evidence_vocabulary_reconciled_at` shall retain it, and the waiver shall
  cover only the evidence-support checks this spec introduces.

### Non-functional
- The existing review suite passes, with its fixtures corrected where they were
  passing incidentally.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go` — attestation validation and
  `AutoAttestReviewBundle`
- `pose-mcp/internal/pose/review_plan.go` — criterion evidence classes
- `pose-mcp/internal/pose/review_closeout.go` — the dated migration exemption
- `.pose/policy/review.json` — the reconciliation date
- `.pose/review-profiles/*.json` — the shipped profiles' demands
- `.pose/adr/2026-08-13-sealed-review-bundles-and-attestations.md` — the
  attestation clause this revises

### Artifacts
- created: .pose/specs/2026-09-08-pose-attestation-evidence-must-be-in-the-bundle.md
- renamed: .pose/changelogs/unreleased/pose-attestation-evidence-must-be-in-the-bundle.md -> .pose/changelogs/v2.0.0/pose-attestation-evidence-must-be-in-the-bundle.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: pose-mcp/internal/pose/review_plan_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: .pose/policy/review.json
- modified: pose-mcp/internal/scaffold/dist/.pose/policy/review.json
- modified: .pose/adr/2026-08-13-sealed-review-bundles-and-attestations.md
- modified: .pose/review-profiles/backend-review.json
- modified: .pose/review-profiles/frontend-review.json
- modified: .pose/review-profiles/milestone-integration.json
- modified: .pose/review-profiles/spec-closeout.json
- modified: pose-mcp/internal/scaffold/dist/.pose/review-profiles/backend-review.json
- modified: pose-mcp/internal/scaffold/dist/.pose/review-profiles/frontend-review.json
- modified: pose-mcp/internal/scaffold/dist/.pose/review-profiles/milestone-integration.json
- modified: pose-mcp/internal/scaffold/dist/.pose/review-profiles/spec-closeout.json

### Technical risks
- The exemption is a waiver, and a waiver is a place a future change could hide.
  It is dated, opt-in, bounded to completed scopes, and covers only the checks
  this spec adds — but nothing stops a project setting the date forward. That is
  the same exposure `component_aware_adopted_at` already carries.
- This fails attestations that already exist. That is the intent — they are
  unsupported and were reported as sound — but an instance updating to it will
  see closeouts it considered finished stop verifying. The remedy is a
  superseding attestation, which the append-only model already provides; there
  is no edit path and none is added.
- Dropping unproducible classes changes plan digests, and therefore bundle
  digests. A bundle sealed before the change does not match one sealed after,
  which is the ordinary consequence of a governed input changing and is what
  bundle supersession exists for.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Reject a passed criterion the bundle does not support (R1, R2, R3)
- [x] Increment 2: Make auto-attest refuse or disposition instead of fabricating (R4)
- [x] Increment 3: Block on an unproducible criterion class instead of erasing it (R5)
- [x] Increment 4: Reconcile the shipped profiles to producible classes (R6)
- [x] Increment 5: Keep completed closeouts recorded before the reconciliation (R7)

### Validation
- [x] Each rejection asserted, and each assertion shown to fail without its change

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: what `auto-attest` should do when the bundle seals nothing of a
  required class.
- Options considered: (a) keep fabricating; (b) refuse always; (c) refuse where
  the scope is expected to carry validation evidence, and record
  `not-applicable` with a rationale where it is not.
- Decision: (c).
- Rationale: (b) was implemented first and broke documentation-only specs with
  no delivery target — a legitimate and common case where no evidence is
  collected by design, and refusing would have made the command unusable for
  them. The engine already separates those two situations for the bundle as a
  whole, one line above the code being changed; reusing that line is better than
  inventing a second rule that could drift from it. Either way the absence is
  visible in the attestation instead of dressed as a pass.
- Consequences: on a scope that carries a delivery target, a missing class is
  now a hard stop for the automated path, and a reviewer must run the check or
  record the disposition themselves.

### Decision 2
- Date: 2026-09-08
- Context: whether the criterion-side vocabulary filter belongs here. The spec
  that fixed the tool side deliberately deferred it.
- Decision: include it. The shipped profiles demand `test` and `validation`,
  neither of which any check may emit, so with R1 in place every plan built from
  them becomes unsatisfiable and the only way through is the fabrication R4
  removes. The deferral was correct while nothing checked the reference; it is
  not once something does.

### Decision 4
- Date: 2026-09-08
- Context: this spec first gave criteria the same treatment as tools — drop the
  unproducible class, warn, continue. Review caught what that does, and it is
  the opposite of the spec's own goal: a criterion left with no class accepts
  any sealed evidence, so `backend-observability`, which asked for `test`, would
  have been satisfied by a `build` result. The demand did not weaken; it
  disappeared.
- Options considered: (a) drop and warn; (b) widen `ValidEvidenceClasses` so the
  demands become true; (c) reconcile the shipped profiles to producible classes
  and block on anything left.
- Decision: (c).
- Rationale: (a) is the self-defeating case above — in a change whose whole
  argument is that a `passed` must mean something, it makes a class of criteria
  mean nothing. (b) keeps every instance working without editing a profile, but
  it ratifies an overlapping vocabulary — `test` alongside `unit`,
  `integration`, `e2e` — which is the confusion the earlier reconciliation was
  undoing, and it would let a check declare `manual-review` as its evidence
  class. (c) fixes the data that was wrong and leaves a blocker as the backstop
  so it cannot silently recur.
- Consequences: an instance whose own profiles demand an unproducible class is
  now blocked at plan time rather than quietly permissive. `pose doctor`'s
  `review.evidence-vocabulary` check, shipped in 1.8.0 and scoped to selected
  profiles in 1.8.1, already names exactly which profile to fix.

### Decision 6
- Date: 2026-09-08
- Context: with the engine changes in place, `pose check --strict` on POSE's own
  repository reported **106** failing closeouts. Reconciling the shipped
  profiles brought it to 52. Both numbers are work that was genuinely reviewed,
  failing because a rule that did not exist when the review happened now
  applies, or because `test` became `unit, integration, e2e`.
- Options considered: (a) re-review 52 completed specs; (b) ship anyway and let
  every adopting instance discover it; (c) a dated migration exemption.
- Decision: (c), as `evidence_vocabulary_reconciled_at` in the review policy.
- Rationale: (a) is not review, it is re-signing to satisfy a tool. (b) makes
  POSE's own gate fail on `main`, which is the least credible way to ship a
  change about governance integrity. (c) is the pattern this engine already uses
  for exactly this situation: `component_aware_adopted_at` exists because the
  criteria contract changed once before. It is opt-in, dated, and bounded to a
  scope already done, so it cannot excuse a review recorded afterwards, and it
  waives only the evidence-support checks — a malformed attestation is still
  rejected, which is asserted.
- Consequences: an instance adopting this sets its own date, or accepts that its
  historical closeouts need superseding attestations. The exemption is visible
  in the policy file rather than compiled in.

### Decision 5
- Date: 2026-09-08
- Context: mapping each unproducible demand to what actually supports it.
- Decision: `test` → `unit, integration, e2e`; `contract` → `integration, e2e`;
  `observability` → `unit, integration`; `integration-test` → `integration`;
  `requirement-trace` and `validation`, on `requirements` and `correctness`,
  fold into the test classes those criteria already imply.
- Rationale: `evidence_classes` is a list of acceptable classes, any one of
  which satisfies the criterion, so the mapping is about naming the real
  producers rather than picking one. `test` was plainly an umbrella for the
  three that exist. `observability` is the weakest of these: nothing emits it,
  and it is diagnosability, which in practice is asserted by the tests that
  exercise the failure paths — recorded as a known gap rather than pretended
  away.

### Decision 3
- Date: 2026-09-08
- Context: four test fixtures failed on the first build of R1, all for the same
  reason: `approvedBundleAttestation` cited a fixed `integration:bundle-test`
  that appeared in no bundle, and the CLI test passed `--evidence test:bundle`
  on the command line.
- Decision: correct the fixtures to cite what their bundles seal, and seed the
  `test`-class result the fixture plans ask for.
- Rationale: those tests proved a signature was well-formed, never that it was
  supported — the property this spec is about. It is the fourth fixture family
  in this session to confirm the case that works rather than the case that runs,
  and the pattern is now frequent enough to be worth naming: a fixture that
  hard-codes an identifier the system under test is supposed to resolve will
  pass whatever the system does with it.

---

## 6. Validation

### Strategy
Assert each rejection against a bundle that reproduces it, assert the supported
attestation still verifies, and disable each change in turn to confirm its test
depends on it.

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
- Notes: all eight packages pass. Restoring the drop-and-warn behaviour turns
  `TestCriterionDemandingAnUnproducibleClassBlocksThePlan` red on `blockers = []`.
  `pose check --strict`, run with a binary built from this branch, went 106
  errors → 52 after the profiles were reconciled → SUCCESS after the dated
  exemption. Each number was measured, not predicted: the first attempt at
  isolating the cause compared the installed v1.8.1 binary against this one and
  was not a comparison at all. Disabling the validator branch leaves
  `blockers = ""` where the test wants "absent from the sealed bundle"; restoring
  the fabrication in `auto-attest` fails the refusal test on "auto-attest
  invented a reference for a class the bundle does not seal". Both new
  validation tests also assert that a properly supported attestation is accepted,
  so neither can pass by rejecting everything.

### Results summary
- Successes: R1, R2, R3, R4, R5, R6, R7 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <reviewCriterionEvidenceBlockers builds the sealed ref set from bundle.Payload.Evidence and rejects a citation absent from it; TestAttestationCriterionMustCiteEvidenceTheBundleSeals/absent_from_the_bundle asserts the blocker>
- R2 [satisfied] <the same function compares the cited ref's class against planned.EvidenceClasses; TestAttestationCriterionMustCiteEvidenceOfARequiredClass picks a sealed ref of a class the criterion does not ask for and asserts the blocker names both>
- R3 [satisfied] <the empty-evidence branch; TestAttestationCriterionMustCiteEvidenceTheBundleSeals/nothing_at_all>
- R4 [satisfied] <AutoAttestReviewBundle returns an error naming the criterion, class and remedy when requiresEvidence, and appends a not-applicable criterion with a rationale otherwise; TestAutoAttestRefusesRatherThanInventingAReference strips the integration results and asserts the error names the class and the way out>
- R5 [satisfied] <composeReviewCriteria appends a blocker naming the criterion, the unproducible classes and the profile, and leaves EvidenceClasses intact; TestCriterionDemandingAnUnproducibleClassBlocksThePlan asserts both the blocker and that the class was not erased>
- R7 [satisfied] <evidenceVocabularyLegacyExempt mirrors componentAwareLegacyAttemptExempt: a date in policy, an attempt reviewed before it, and a scope already done; it waives the plan-digest comparison, the legacy evidence-class match and, through validateBundleAttestationWith, the sealed-bundle evidence checks. TestEvidenceSupportIsWaivedOnlyForApprovalsPredatingTheRule asserts the waiver covers the evidence blockers and nothing else>
- R6 [satisfied] <the four shipped profiles demand only classes in ValidEvidenceClasses, asserted by a script over .pose/review-profiles and mirrored into the embedded scaffold by go generate; the suite would block on any that were missed>

### Known gaps
- **The two vocabularies are still two.** `ValidEvidenceClasses` says what a
  check may emit; `reviewEvidenceClassCatalog` says what a profile may demand.
  Six of nineteen classes are in both. Ten a profile may demand can never be
  produced — `contract`, `integration-test`, `lint`, `manual-review`,
  `observability`, `requirement-trace`, `security-scan`, `test`, `typecheck`,
  `validation` — and three a check may emit can never be demanded:
  `contrast`, `design-system`, `visual-regression`. This spec reconciles the
  shipped profiles to the intersection and blocks on the rest; it does not
  unify the lists, which would stop a profile declaring an unsatisfiable class
  at load time instead of at plan time, and is a larger break because it fails
  profile loading rather than planning.
- `backend-observability` now asks for `unit, integration`. Nothing emits an
  observability class, and diagnosability is not the same property as a test
  passing; the mapping is the closest true statement available, not an exact
  one.
- Evidence is matched by reference and class, not by component. In a
  multi-component bundle an `integration` result from one component can support
  a criterion scoped to another, because the sealed set is consulted globally.
  All three closeouts that motivated this spec are caught anyway — two by class,
  one because the references were not in the bundle at all — but the narrower
  case is open.
- Tool dispositions are still not checked against sealed evidence. The tool path
  has its own coverage evaluation and its own `validation:auto-attest`
  fabrication, which this spec does not touch.
- R5 is asserted through the suite rather than by a dedicated test: the shipped
  profiles' unproducible demands are what the other tests would fail on without
  it. A direct test would be better.

---

## 7. Final Report

### Follow-ups

- [open] Check tool dispositions against sealed evidence and remove the validation:auto-attest fabrication, the way criteria now are — owner:unowned crit:high review:2026-11-08
- [open] Unify ValidEvidenceClasses and reviewEvidenceClassCatalog so a profile cannot declare a class no check may emit — owner:unowned crit:high review:2026-11-08
- [open] Match evidence to the criterion's component, so a sibling component's result cannot support it — owner:unowned crit:high review:2026-11-08
