---
slug: pose-review-declared-threat-model
status: in-progress
created_at: 2026-11-23
completed_at:
supersedes:
depends_on: 
remediates:
priority: 4
components: pose-mcp
task_type: feature
surface: standard
delivers: capability:review-declared-threat-model
---

# Spec: pose-review-declared-threat-model

## 1. Intent

### Goal
Let a spec declare the accepted residual limits of a guarantee it makes, and let the review brief treat findings inside those limits as follow-ups to classify, not blockers to chase.

### Business value
Each of the 23 rounds of `pose-validate-carries-unchanged-checks` found a more remote adversarial variant: concurrent HEAD movement, case folding, Unicode normalization, symlinks inside submodules. Every one was real, but no round could converge, because the spec had no place to say which environment the guarantee assumes. A cap on rounds would approve something unreviewed; a declared threat model converges by explicit classification.

### Constraints
A declared limit never hides a defect inside the declared scope. The reviewer may dispute a limit; the dispute is a finding with its own classification. Declared limits are visible in the closeout and the attestation.

### Non-goals
Limiting the number of review rounds.

## 2. Requirements

### Functional
- R1: A spec shall be able to declare `### Accepted limits` entries with stable ids (`L<N>`), each naming the assumption and why it is accepted.
- R2: The review brief shall list the declared limits and instruct the reviewer to classify a finding that falls inside one as a follow-up naming the limit, or to dispute the limit explicitly.
- R3: The conclusion and the attestation shall record which findings were classified under which limit.
- R4: `lint-spec --ready-check` shall require accepted limits to be justified and the ADR, when present, to reference them.

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp` (CLI and engine), the manual and its embedded copy.

### Artifacts
- created: .pose/specs/2026-11-23-pose-review-declared-threat-model.md
- created: pose-mcp/internal/pose/accepted_limits.go
- created: pose-mcp/internal/pose/accepted_limits_test.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_rounds.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/pose/review_brief.go
- modified: pose-mcp/internal/pose/review_brief_test.go
- modified: pose-mcp/internal/pose/review_brief_versions_test.go
- modified: pose-mcp/internal/pose/review_brief_conclusion_rules_test.go
- modified: pose-mcp/internal/pose/testdata/review_brief_template_digests.json
- modified: pose-mcp/internal/cli/lintspec.go
- created: pose-mcp/internal/cli/accepted_limits_lint_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md


### Delivery targets

- capability:review-declared-threat-model module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/lintspec.go


### Accepted limits
- L1: The limits are written by the implementer, so a limit can be broader than it should be. Accepted because the brief presents them as data to verify and a disputed limit is a blocking finding of its own.
- L2: A spec edited after sealing changes its limits only through a new bundle. Accepted because the limits enter the plan digest, so an edit stales the review as any other change does.

### API/contract changes
A spec section `### Accepted limits` with `- L<N>: <assumption>. Accepted because <reason>.` entries. The review plan and the bundle seal them as `accepted_limits` (omitted when none, so every other plan and bundle keeps its digest). A draft finding may carry `limit`; the attestation records it with disposition `within-limit`. Brief template `review-brief/11`. `lint-spec --ready-check` checks the section.


### Data/storage changes
`accepted_limits` in review plans and bundles and `limit` in draft and attestation findings, all omitted when empty.


### Technical risks
A limit could be used to wave a real defect through. It is the spec's own declaration, sealed into the plan digest and visible in the brief and the attestation; the reviewer may dispute it with a finding of its own, which then blocks as any finding does.


## 4. Tasks

### Planning
- [ ] Confirm the requirements against the current engine before starting.

### Implementation
- [ ] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3
A finding inside a declared limit is neither an open finding nor an accepted risk: it gets its own disposition, `within-limit`, which does not block an approval whatever its severity, requires a sealed limit and a reason, and records the limit's id. Accepted risks keep their severity bound; a limit is the spec's statement of scope, decided before the review.

### Decision D2
- Basis: R1, R4
The limits are sealed in the plan and the bundle, and in the plan digest, so the brief and the conclusion read them from what was reviewed, never from the current spec; both fields are omitted when empty so no earlier plan or bundle changes digest.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-25: implemented R1–R4: `ParseAcceptedLimits`, the plan and bundle `accepted_limits`, the draft `limit` and the `within-limit` disposition, `review-brief/11` and the ready check. Each has a test that fails with it mutated out; the first digest test survived because the spec edit changed the scope digest too, and now compares the plan's digest with and without the limits.
- 2026-11-25: review round 1 (rrn-d1680df442b23d12) requested changes. N1: in a scope with several specs (a milestone or a roadmap) each limit id is qualified by its spec as `<spec>#L<N>`, with the spec recorded, so a finding names exactly one sealed limit. N2: the reason must follow `Accepted because`, not only the marker. N3: an ADR names a limit only by the id as a word, so L10 does not name L1. Each fix has a test that fails with it mutated out.
- 2026-11-25: review round 2 (rrn-256db7e6eea5a6e2) found that punctuation after the marker passed as a reason and a one-letter first word failed. The reason must now hold at least two words with letters (`hasAcceptedLimitReason`), with a test that fails with the rule mutated out.
- 2026-11-25: review round 3 (rrn-0af0d413761142d8) found that the marker matched inside a sentence, so 'Not accepted because ...' passed. The marker must now open a sentence (start of the text or after `.`, `;`, `:`, `!` or `?`), as the format `<assumption>. Accepted because <reason>.` states; the test fails with the boundary mutated out.
- 2026-11-25: review round 4 (rrn-721e2a4c9bd37e2e) found that an entry with no assumption, or with an empty reason followed by other prose, passed. The entry is now read structurally: an assumption of at least two words, the marker opening a sentence, and a reason of at least two words within that same sentence. Each rule has a test case that fails with it mutated out.
- 2026-11-25: review round 5 (rrn-9f6867b3c3e42377): N1, a whitespace-only reason of a `within-limit` finding is refused; N2, the ready check fails closed on an ADR directory or file it cannot read, keeping the absent-directory case. Both have tests that fail with them mutated out.


### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.


### Requirement trace
- R1 [satisfied] test:TestAcceptedLimitsAreParsed, test:TestTheReviewPlanSealsTheAcceptedLimits, test:TestLimitsAcrossSpecsAreQualified, test:TestAnAcceptedLimitNeedsWordsAsItsReason
- R2 [satisfied] test:TestTheCurrentBriefListsTheAcceptedLimits
- R3 [satisfied] test:TestAFindingWithinASealedLimitDoesNotBlockApproval
- R4 [satisfied] test:TestReadyCheckJudgesAcceptedLimits


### Known gaps
None.


## 7. Final Report

### Delivered scope
Specs declare accepted limits; the review plan and bundle seal them; `review-brief/11` lists them and tells the reviewer to classify a finding inside one with `limit`; the attestation records it `within-limit`, which does not block an approval; `lint-spec --ready-check` requires each limit to say why it is accepted and an ADR citing the spec to name it.


### Files and modules changed
`pose-mcp` (plan, bundle, conclusion, brief, lint), the manual in both locales and its embedded copies.


### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.


### Residual risks
L1 and L2 above.


### Follow-ups
None.


