---
slug: pose-abm-progressive-review
status: in-progress
created_at: 2026-09-19
completed_at:
depends_on: pose-abm-review-soundness, pose-abm-review-authority, pose-abm-design-basis, pose-abm-structural-delta
priority: 1
components: pose-mcp
task_type: feature
delivers: governance:abm-progressive-review
---

# Spec: Progressive review from explicit obligations

## 1. Intent

Implement the POSE portion of the Harne8 ABM foundation roadmap in this executor
repository. Keep the stable coordinator slug. Owner: @pose-maintainers.
Ship opt-in review profiles first, then connect observations and explanations
without introducing a second policy engine. The coordinator remains a program
record; this document records implementation and its remaining acceptance gates.

### Constraints

Preserve offline operation, explicit adoption and historical evidence. Judgment
requires an explicit reviewer conclusion. Do not activate profiles by installing
or updating the distribution. Do not claim the whole roadmap from one increment.

## 2. Requirements

- R1: Distribute engineering-judgment and high-criticality-review as opt-in
  profiles; match high and critical explicitly.
- R2: Derive baseline/elevated/critical explanations from the effective plan,
  including trigger, source, policy and obligation, without another policy engine.
- R3: Require assumption-integrity, design-causality and solution-proportionality
  when applicable; include negative space and speculative extensibility in the last.
- R4: Preserve the independence floor against overlays and author-controlled
  metadata; review governance changes under a protected policy baseline.
- R5: Distinguish declared preflight forecasts from final observations and show
  additional obligations when observed scope expands.
- R6: Add no ABM action/document/approval to trivial changes without a material
  trigger; do not infer triviality from a Markdown extension.
- R7: Require specific judgment rationale and advisory mapping for material
  structural deltas; distinguish missing evidence, N/A and accepted risk.
- R8: Share the plan across CLI/MCP consumers, deduplicate obligations and expose
  uncertainty explicitly. Portal composition belongs to Harne8 review-experience.

## 3. Technical Plan

### Increment 1

Reuse schema-v2 selectors and monotone composition. Both profiles select declared
surface, capability, contract, infrastructure or governance targets. The criticality
profile additionally matches high/critical and raises independence to different-actor.
These conjunctive selectors keep target-free editorial scopes light, including in
a high component. A governance target remains material even with a Markdown entrypoint.
Both profiles contribute the same three explicit judgment criteria, deduplicated by
the existing composer, and no new tools. Missing metadata keeps existing diagnostics.

This first increment deliberately does not infer that an undeclared target or a
missing structural observation proves triviality. Observed structural selectors,
band explanations, protected baseline comparison and obligation deltas remain
acceptance work in this same spec. No new public schema is needed for the profiles.

### Increment 2

Explain the resolved plan instead of resolving it a second time. `band` and
`bands` name, per reason, the selector facts that matched, whether the fact was
declared, observed, read from policy or left undecided, where it is readable, the
ref that made it consequential and the obligations it produced. Criticality
`high`/`critical` or a raised independence yields `critical`; another matched
overlay yields `elevated`. An adopted overlay that cannot decide a component —
metadata missing, or a selected field declared empty — yields an `unknown` entry
with no obligations, which neither raises nor lowers the band.

`projection` re-resolves the plan over declared scope alone through the same
selection, composition and tool builders, so a delta is always attributable to a
fact rather than to a second set of rules. It reports the forecast and, when
delivery provenance attributed more scope than the spec declared, the components,
profiles, criteria, tools, floor and band that only the observation produced.
Each component carries an `origin` for the same reason.

Neither derivation enters `digestReviewPlan`. Given identical digested inputs the
summary is a pure function of them, so it adds no obligation and supersedes no
sealed review; a summary that moved the digest would be indistinguishable, to a
verifier, from a real change in obligations. Observed structural selectors, the
protected policy baseline and the R7 advisory mapping remain pending.

### Increment 3

A diff that changes the review contract cannot be the authority that approves it.
When the scope touches `.pose/policy/` or `.pose/review-profiles/` — the two
directories that are the contract, since both carry their own schema version, and
rule bodies stay out because it is the criterion contract that is compared — the
plan resolves the contract a second time at the change set's
resolved base — same parsers, same selectors, same composer, different revision —
and restores upward only: the stricter independence, a dropped criterion (with
`protected-baseline:` provenance) and a criterion softened from required to
optional or from judged to collected. A baseline weaker than the diff changes
nothing, which is what keeps this a direction rather than a second engine.

Two cases previously made a contract change unreviewable, because the plan refused
to exist and the change landed with no plan at all: `enabled: false`, and removing
or breaking the profile the policy points at. Both now resolve under the protected
contract and record the weakening. `ScopeDigest` had the same coupling — it read
the profile from the tree — so an absent profile now contributes empty bytes, a
distinct digest input, while a malformed one is still a hard error.

Restorations reach the digest through criteria, independence and the explain trail.
The structured `policy_baseline` report, like the band summary, stays a projection.
A contract change whose base cannot be resolved is reported unprotected with a
warning and an `unknown` band; POSE does not imply protection it did not have.

### Artifacts

- created: .pose/specs/2026-09-19-pose-abm-progressive-review.md
- created: .pose/review-profiles/engineering-judgment.json
- created: .pose/review-profiles/high-criticality-review.json
- created: pose-mcp/internal/scaffold/dist/.pose/review-profiles/engineering-judgment.json
- created: pose-mcp/internal/scaffold/dist/.pose/review-profiles/high-criticality-review.json
- created: pose-mcp/internal/pose/progressive_review_test.go
- created: pose-mcp/internal/cli/progressive_review_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/releases.json
- modified: .pose/indexes/spec-graph.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/results/abm-progressive-review-validation.json
- created: .pose/changelogs/unreleased/pose-abm-progressive-review.md
- created: pose-mcp/internal/pose/review_bands.go
- modified: pose-mcp/internal/pose/review_plan.go
- created: pose-mcp/internal/pose/review_policy_baseline.go
- created: pose-mcp/internal/pose/review_policy_baseline_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/state/components/pose-mcp.json

### Delivery targets

- governance:abm-progressive-review module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

The existing backend-go delivery profile requires integration evidence. Register a
dedicated integration producer for installation and real CLI plan resolution.

### Rollout and reversal

Install the files without changing overlay_profiles. Preview explicit adoption in
a temporary instance using review-plan. Removing the adopted overlay refs reverts
this increment's extra criteria; retain immutable prior reviews.

## 4. Tasks

- [x] Confirm existing selectors/composition and register this implementation plan.
- [x] Distribute opt-in profiles and cover installation, selection and no-downgrade.
- [x] Derive band explanations and declared/observed obligation deltas from the
  common plan; expose undecided selectors without escalating them.
- [ ] Add observed structural selectors and the R7 advisory mapping.
- [x] Enforce the protected policy baseline, including a disabled policy and a
  removed profile in the reviewed diff.
- [ ] Validate the full requirement corpus, review and close through POSE.

## 5. Decisions

Reuse the existing profile contract for increment 1. Declared delivery kinds are
the initial material trigger; component criticality alone is insufficient to
classify an editorial change. This is a bounded first producer, not the final
structural trigger contract. The high profile raises independence, while verified
identity assurance continues to depend on the authority policy already implemented.

Consulted knowledge:module-metadata-discovery-invalidates-review-provenance;
avoid unrelated metadata changes while collecting review evidence.

## 6. Validation

Required plan recorded before implementation:

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Opt-in, duplicate criteria, trivial scopes, authority Markdown | `go test ./internal/pose -run ABMProgressiveReview -count=1` | Judgment only after matching adoption; trivial plan unchanged |
| High, critical, medium, unknown, existing independence floor | `go test ./internal/pose -run 'ABMCriticality\|ABMPolicyDowngrade' -count=1` | Exact selectors, explicit unknown and no weaker floor |
| Install and actual CLI JSON | `go test ./internal/cli -run ABMProgressiveReview -count=1` | Installed profiles resolve in real command; policy remains opt-in |
| Bands, undecided selectors, plan identity, observed expansion | `go test ./internal/pose -run 'ABMProgressiveReviewBands\|ABMProgressiveReviewUnknown\|ABMProgressiveReviewObserved\|ABMProgressiveReviewCriticalityEscalates' -count=1` | Explained trigger/basis/source/policy/obligation; unknown priced at zero; digest unchanged; expansion attributed |
| Shared plan across consumers | `go test ./internal/cli -run ABMProgressiveReview -count=1` | CLI JSON equals the store plan; band and forecast visible without `--explain` |
| Protected baseline: restoration, disabled policy, removed profile, unresolvable base, ordinary scope | `go test ./internal/pose -run ABMProtectedBaseline -count=1` | Floor and criteria restored upward with provenance; contract change reviewable in every case; unprotected states stated, not assumed |
| Distribution parity | `go test ./internal/scaffold -run TestEmbeddedDistMatchesPoseDist -count=1` | Embedded and canonical assets agree |
| Full module | `pose validate --strict --module pose-mcp` (from repository root) | Build, tests, vet and registered integration checks pass |

### Execution log

2026-09-19: discovery completed before implementation (43,346 production LOC,
32,746 test LOC, no TODO/FIXME markers); coordinator ready-check passed.

2026-09-20 UTC: targeted corpus and embedded distribution parity passed. Full
`pose validate --strict --module pose-mcp` passed all seven checks (build, test,
vet, ABM integration, delivery integration, reachability and bundle convergence);
result: `.pose/results/abm-progressive-review-validation.json`. This is pre-commit
increment evidence, not terminal delivery evidence for the full spec.
The installed binary's global `pose check --strict` reports historical bundles
with unknown `implementation_digest`; the source models that field. Check the
candidate binary before interpreting this as an implementation regression.

Increment committed as `eff370a` with this spec's trailer. Candidate
`artifact-check --spec pose-abm-progressive-review --strict` exited 0; historical
orphan warnings remain outside this increment. Its regenerated delivery index
is included in the artifact inventory.
Candidate verification can read the old design-basis bundle, but reports it
`superseded` after shared manual/validation inputs changed. Historical attestations
are preserved; this increment does not manufacture replacement approvals.
The candidate's repository-wide check was interrupted without a final verdict;
do not record it as passed. Its targeted readiness check passed. The candidate
module matrix passed 7/7 after allowing loopback listeners for integration tests.
Use `surface-check --results .pose/results/abm-progressive-review-validation.json`
for this increment: the default results path still contains the previous task's
evidence. Reconcile the newly declared generated index before refreshing provenance.

Increment 2 committed as `22fa8bf` with this spec's trailer. Against that commit
the candidate `artifact-check --spec pose-abm-progressive-review --strict` exited 0
with 24 claims and 24 observed artifacts; the 361 orphan findings are the same
historical governed paths reported before this increment and lie outside it.
Candidate `validate --strict --module pose-mcp` passed 7/7 (build, test, vet, the
ABM integration producer, delivery integration, reachability and bundle
convergence) into `.pose/results/abm-progressive-review-validation.json`, and
candidate `surface-check --spec pose-abm-progressive-review --strict --results`
that path passed with one target, seven results and zero findings.

`pose index` then declared two generated indexes this spec had not claimed —
`releases.json` and `spec-graph.json`, which had never carried this spec's
changelog entry or dependency edges. They are now claimed. As in increment 1,
this is pre-commit increment evidence for increment 2, not terminal delivery
evidence for the spec: three requirements remain open.

Increment 2 defect injection, so the corpus is known to fail before it passes:
suppressing criticality escalation, suppressing undecided-selector entries,
resolving the forecast from observed scope, and admitting the summary into
`digestReviewPlan` each failed exactly the case that asserts it.

After `a657d0d`, artifact-check passed with 15 claims and 15 observed artifacts.
Candidate validation passed 7/7 with current scoped provenance. Candidate
`surface-check --spec pose-abm-progressive-review --strict --results
.pose/results/abm-progressive-review-validation.json` passed: one target, seven
results, zero findings. This proves composition of increment 1, not satisfaction
of the requirements still listed as pending above.

Increment 3 defect injection: not restoring the floor, not re-adding a dropped
criterion, not restoring a softened one, refusing a plan for a disabled policy, and
claiming protection without applying it each failed exactly the case that asserts
it. The full module suite and `go vet` stayed green.

### Requirement trace

No requirement is terminally satisfied yet. Increment 1 exercises R1, R3, the
overlay floor of R4, a declared-trigger corpus for R6 and deduplication in R8.
Increment 2 adds R2 in full — band explanations carrying trigger, basis, source,
policy and obligation, derived without a second engine — R5 in full, and the
uncertainty half of R8. Increment 3 completes R4: the independence floor holds
against overlays and author metadata, and a governance change is now reviewed under
a protected contract baseline. R7 and the observed structural triggers still needed
by R6 require subsequent implementation.

## 7. Final Report

Increments 1 to 3 are implemented and verified through installation, CLI, MCP
catalog parity and the scoped delivery gate. The spec remains in-progress for
observed structural triggers and the R7 advisory mapping. No release, deployment, global adoption or full roadmap closeout is
claimed.
All remaining acceptance work stays in this spec's requirements/tasks.
