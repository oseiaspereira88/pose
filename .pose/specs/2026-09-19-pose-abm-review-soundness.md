---
slug: pose-abm-review-soundness
status: in-progress
created_at: 2026-09-19
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: feature
delivers: governance:explicit-judgment
---

# Spec: Explicit judgment and one set of attestation invariants

## 1. Intent

### Goal
Stop the engine from approving, on a reviewer's behalf, criteria that no registered check
reports on, and give every entry point to an attestation the same integrity rules.

### Business value
An approval should mean what it says. Today it can be produced in full without anyone
concluding anything, which makes the whole review record weaker evidence than it appears.

### Constraints
- Every mechanism asserts only what it observes. Collecting evidence is not judging it.
- Historical artifacts stay readable and keep their verdict. No rationale is fabricated.
- The engine never scores quality, calls an LLM, or decides whether a design is good.
- Skills, profiles, schema and scaffold change in the same increment as the runtime, so
  no documented instruction tells an agent to do what the engine now refuses.

### Non-goals
Reviewer authority and identity assurance (`different-actor`, `mandatory-human`) are a
separate contract and are deliberately untouched here — the fourth characterization test
still reproduces, and that is the correct scope boundary, not an oversight.

## 2. Requirements

### Functional
- R1: A criterion no registered producer can answer is prepared as a pendency, never as a
  disposition. `auto-attest --apply` refuses while any pendency remains, records nothing,
  and names each unanswered criterion and why.
- R2: A criterion a registered producer answers is still prepared automatically from the
  sealed evidence, so automation keeps the collection and association it always did.
- R3: A criterion disposed as `finding` must name a finding the same attestation records.
  The Store enforces it, so the CLI, a Store client, a signed import and criterion reuse
  all get the same refusal.
- R4: Missing evidence is never inapplicability. A criterion asking for a class the bundle
  seals cannot be `not-applicable`, and on a scope carrying a delivery target an
  attestation whose every required criterion is dispensed with does not verify.
- R5: A judged criterion recorded as `passed` carries a conclusion; an empty one is
  refused, under the contract the bundle sealed.
- R6: A bundle sealed before the contract existed is never judged by it. The 470
  attestations in this repository and the 25 in the consumer instance keep their verdict
  and stay readable.
- R7: A profile may raise a collected criterion to a judged one and may never declare a
  criterion mechanical with no evidence class, which nothing could answer.
- R8: A component-scoped tool whose component the validation matrix declares runs no check
  is planned as such and may be dispensed with, naming the reason. Everywhere else a
  required tool must still pass, and the engine claims the gap only where the repository
  declared it.

### Non-functional
Deterministic and offline. No new persisted artifact, index, schema version or command.
One accessor resolves a criterion's kind, so preparation, the CLI and the verifier cannot
disagree about what a reviewer still owes.

### Security
Repository content remains untrusted input. A profile is the thing being constrained, so
it cannot widen its own obligation: `kind` composes monotonically across overlays, the way
independence already does, and the weaker reading never wins. Nothing here weakens an
existing gate, and a refused preparation writes nothing at all.

### Compatibility
The contract is stamped into the bundle at seal time through the existing
`governing_contracts` registry, not through an editable adoption date. Adding `kind` to a
planned criterion changes the plan digest, so an open review is re-prepared under the
corrected contract — which is the intent, and is bounded to open scopes because a
completed one is exempted by the mechanism that already exists for the three contracts
before this one.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose` (criterion contract, preparation, verification),
`pose-mcp/internal/cli` (attest and auto-attest), the distributed skills, the manual, the
CLI reference, the review-plan schema and the embedded scaffold.

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-review-soundness.md
- created: pose-mcp/internal/pose/abm_review_soundness_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: pose-mcp/schemas/v1/review-plan.schema.json
- modified: .agents/skills/pose-review/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: .agents/skills/pose-feature/SKILL.md
- modified: POSE.md
- modified: docs-site/docs/cli.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/releases.json
- modified: .pose/indexes/spec-graph.json
- modified: .pose/results/delivery-validation.json
- created: .pose/reports/2026-09-19-standard-validate-native.md
- modified: .pose/reports/history/standard-validate-native.jsonl
- created: .pose/review-bundles/rvb-8322a4c339433094.json
- created: .pose/review-bundles/rvb-998ab137871a47b5.json

Files below the scaffold entry are produced by the governed flow itself — indexes, the
validation result set, the validate report and the sealed bundles this review went
through. They are declared so `artifact-check` reconciles claims with observed paths
instead of reporting the engine's own output as undeclared.

### Delivery targets
- governance:explicit-judgment module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
`ReviewCriterionProfile` and `ReviewPlanCriterion` gain `kind`, a closed set of
`mechanical` and `judgment`. Absent, it is derived rather than defaulted: a criterion that
names evidence classes has a producer, one that names none does not. `AutoAttestReviewBundle`
keeps its signature; `PrepareReviewAttestation` is added and returns the prepared
attestation plus the pendencies. A new governing contract, `explicit-judgment`, is
registered and stamped at seal.

The distributed profiles are deliberately left unedited: the derivation already classifies
their criteria correctly, and annotating them would change every plan digest for no
semantic gain.

### Data/storage changes
None. Preparation writes nothing, and a refused `--apply` records no attestation.

### Technical risks
Every open review is re-prepared, because the plan digest changes. Accepted: re-adopting
the corrected contract in both repositories is a precondition of the increment, not a side
effect of it. A scope whose plan is entirely judgment now needs a reviewer before it can
close, which is the point and is also the largest behavior change an existing automation
will notice.

## 4. Tasks

### Planning
- [x] Reproduce the four characterization behaviors against the release under change.
- [x] Measure which proposed rule would invalidate historical records, and gate only that one.

### Implementation
- [x] Add the criterion kind contract, its derivation and its monotone composition.
- [x] Split preparation from approval and report pendencies through the CLI.
- [x] Centralize finding-reference and applicability invariants in the Store.
- [x] Reconcile skills, manual, CLI reference, schema and embedded scaffold.

### Validation
- [x] Invert the characterization suite into acceptance tests.
- [x] Run the module matrix.

## 5. Decisions

### Decision 1: derive the kind instead of defaulting it
- Context: the distributed profiles predate the field. `spec-closeout` has five criteria
  with no evidence class out of seven, `milestone-integration` four of five, and
  `roadmap-outcome` six of six.
- Options considered: default absent criteria to mechanical, preserving current behavior;
  require every profile to declare `kind` before the contract applies; derive it.
- Decision: derive it from the presence of evidence classes.
- Rationale: defaulting to mechanical would keep the hole open for exactly the profiles
  that exercise it most. Requiring a declaration would make the contract inert until every
  profile is edited. The derivation states something the engine can defend — no class means
  no registered producer — and a profile can still override upward.
- Consequences: a roadmap closeout now requires six explicit judgments. That is the
  intended reading of a roadmap outcome, and it is the scope where the gap cost most.

### Decision 2: stamp the contract into the bundle rather than date its adoption
- Context: R6 requires historical records to keep their verdict.
- Options considered: an `explicit_judgment_adopted_at` policy date; the existing
  `governing_contracts` stamped at seal.
- Decision: register the contract so it is stamped at seal, and gate only the judgment
  conclusion rule on it.
- Rationale: a date can be edited after the fact; a stamp cannot. The repository's own
  mechanism already says so, one contract earlier.
- Consequences: the finding-reference and applicability rules are not gated, because
  measurement showed they invalidate nothing: across 495 attestations in both repositories
  there are zero findings, zero criteria disposed as `finding`, zero all-inapplicable
  attestations and zero not-applicable dispositions lacking a rationale.

## 6. Validation

### Strategy
The four behaviors recorded against 5.0.8 are the acceptance corpus: each was reproduced
as a passing characterization test before the change and must no longer reproduce after
it, except the independence one, which belongs to a different contract. Each new rule also
has a positive case, so the suite proves discrimination rather than blanket refusal.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./... -count=1`
- Scope: pose-mcp
- Expected: SUCCESS, including the seven new acceptance tests

#### Lint
- Command: `cd pose-mcp && go vet ./...`
- Scope: pose-mcp
- Expected: no findings

#### Typecheck
- Command: `cd pose-mcp && go build ./...`
- Scope: pose-mcp
- Expected: SUCCESS

#### Build
- Command: `pose validate --strict --module pose-mcp`
- Scope: pose-mcp
- Expected: SUCCESS on every registered check

#### Security / Contract
- Command: `cd pose-mcp && go test ./internal/pose ./internal/cli -run 'ABMReviewSoundness|ReviewAutoAttest|ReviewCriterionDispositions|DirectPrint' -count=1`
- Scope: attestation invariants across Store and CLI
- Expected: refusal at every entry point; the print ratchet holds

### Execution log
- Date: 2026-09-19 (R8 added during implementation, after the first hand-answered review
  reached a required tool nothing could feed; the spec had no baseline to amend against,
  since `pose start` does not exist yet)
- Environment: local worktree, Go toolchain, no network
- Notes: the characterization file from the originating review was copied into the tree,
  run against the change and removed. Three of its four tests now fail, which is the
  intended inversion; `TestABMAuditSelfDeclaredIndependence` still passes and is out of
  scope for this spec.

### Results summary
- Successes: full `go test ./...` for pose-mcp; seven new acceptance tests; the three
  reproduced defects no longer reproduce.
- Failures: none outstanding.
- Warnings: adding `kind` changes the plan digest, so open reviews are re-prepared.

### Requirement trace
- R1 [satisfied] test:TestABMReviewSoundnessJudgmentIsNotAutoApproved test:TestReviewAutoAttestCLI
- R2 [satisfied] test:TestReviewAutoAttestCLI test:TestReviewCriterionDispositionsRefuseJudgmentByOmission
- R3 [satisfied] test:TestABMReviewSoundnessOrphanFindingRefusedByStore
- R4 [satisfied] test:TestABMReviewSoundnessBlanketNotApplicableRefused test:TestABMReviewSoundnessNotApplicableContradictedBySealedEvidence
- R5 [satisfied] test:TestABMReviewSoundnessJudgmentNeedsAConclusion
- R6 [satisfied] test:TestABMReviewSoundnessUnstampedBundleKeepsItsVerdict
- R7 [satisfied] test:TestABMReviewSoundnessMechanicalCriterionNeedsAProducer
- R8 [satisfied] test:TestABMReviewSoundnessToolWithoutProducerIsDispensable

### Known gaps
Reviewer authority remains declarative: `different-actor` and `mandatory-human` are still
satisfied by a prefix. That is the next contract, not a gap in this one.

The first review in this repository that had to be answered by hand was also the first to
surface R8. The diff touches `docs-site/docs/cli.md`, so `docs-site` entered the plan as a
component and `validate docs-site` became a required tool — while the matrix declares, in
`moduleOverrides.docs-site`, `replaceDefaultChecks` with an empty check list. The plan was
asking for evidence the repository had explicitly said it does not produce, and the
reviewer's only options were to cite another component's result, which is false, or to
stay blocked. R8 closes it by reading that declaration, which is the same refusal the
profile loader already performs one level up, per component instead of per class.

---

## 7. Final Report

### Delivered scope
The judgment contract, the finding-reference and applicability invariants, the tool-side
applicability of R8, the preparation split, and the reconciliation of skills, manual,
schema and scaffold. Authority, structural
observation and governance outcomes are not in this increment.

### Files and modules changed
- pose-mcp: criterion contract, preparation, verification, CLI, schema, embedded scaffold
- repository governance: three skills, the manual and the CLI reference

### Validation executed
- Command: `cd pose-mcp && go test ./... -count=1`
- Result: SUCCESS

### Residual risks
- Open reviews are re-prepared under the new plan digest; completed scopes are exempt by
  the sealed-contract mechanism.
- A profile author can still ask for judgment everywhere and make review expensive. The
  engine does not police proportionality; that is the progressive-review contract.

### Follow-ups

- [open] Reviewer authority is still satisfied by a declared prefix, so `different-actor`
  and `mandatory-human` assert identity they do not verify (owner:@pose-maintainers
  crit:high review:2026-10-03)
- [resolved] A component the validation matrix declares runs no check produced a required
  tool that could not be honestly dispositioned; the plan now reads that declaration and
  the tool is dispensable with a reason (R8)
