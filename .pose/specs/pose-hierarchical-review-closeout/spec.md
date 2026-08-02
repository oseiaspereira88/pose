---
slug: pose-hierarchical-review-closeout
status: draft
created_at: 2026-08-02
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp, cli, lifecycle, reviews, roadmaps, mcp, scaffold
---

# Spec: Hierarchical review closeout

## 1. Intent

### Goal
Require a complete, current and scope-appropriate review before a spec,
milestone or roadmap can be considered closed, and make review findings drive a
bounded implementation/revalidation loop.

### Business value
POSE already describes a strong review procedure, but review is not a durable
closeout condition. A spec can become `done` after deterministic validation and
follow-up disposition without any review attestation. A milestone is considered
satisfied as soon as all member specs are `done`, and a roadmap is satisfied by
its manually assigned `status: done`. Users therefore have to repeat prompts
such as “review every spec, milestone and roadmap; fix every gap; do not stop
until the full scope is reviewed”. Persisting that intent removes prompt
dependence, prevents unchecked roll-up and gives autonomous orchestration an
objective terminal state.

### Constraints
- Preserve the distinction between deterministic validation and technical
  judgment; a review consumes check evidence but is not itself a build check.
- Bind every final review decision to a canonical digest of the exact closeout
  inputs it covered.
- Keep review profiles versioned, explicit and proportional to spec, milestone
  and roadmap scope.
- Keep review artifacts local, diff-reviewable and valid offline.
- Reuse current rules, requirement traces, reports, amendments, follow-ups and
  roadmap membership instead of inventing parallel sources of truth.
- Minimize reviewer identity to a configured alias or execution identity; never
  turn review records into individual productivity metrics.
- Preserve authorization boundaries: continuous execution may resume in-scope
  implementation, but cannot authorize destructive, privileged or external
  actions that otherwise require approval.
- Stage enforcement for legacy `done` specs and roadmaps.

### Non-goals
- Claim that a checklist guarantees reviewer insight or semantic quality.
- Replace deterministic tests, validation profiles, artifact provenance,
  surface assurance or human approval required by security policy.
- Require a different human reviewer for every repository and risk class.
- Reopen historical closed scopes silently after release; post-closeout gaps
  become explicitly linked remediation work.
- Let autonomous remediation expand beyond the selected terminal scope or alter
  public contracts without the normal spec and ADR gates.

## 2. Requirements

### Functional
- R1: POSE shall address review scopes with typed refs `spec:slug`,
  `milestone:roadmap/id` and `roadmap:slug`, and reject missing, malformed or
  unauthorized refs.
- R2: Versioned review profiles shall declare stable criterion IDs, applicable
  rules, mandatory evidence classes and allowed finding dispositions for each
  scope level.
- R3: A review attempt shall be a first-class immutable artifact containing
  review ID, scope ref, scope digest, profile/version, reviewer identity,
  evidence refs, criterion dispositions, findings, final decision, timestamp
  and optional superseded review ID.
- R4: Every profile criterion shall be disposed as `passed`, `finding` or
  `not-applicable`; `not-applicable` shall require rationale and a final
  decision shall fail when any criterion is absent or malformed.
- R5: Findings shall have stable IDs, severity, evidence, action and disposition;
  `open` and `changes-requested` findings shall block closeout, while accepted
  risk shall require an owner, rationale, review date and policy permission for
  its severity.
- R6: Strict closeout shall accept only a current `approved` review by default;
  `approved-with-reservations` may close only when policy explicitly permits
  every remaining non-blocking accepted risk.
- R7: A canonical scope digest shall exclude transition-only fields such as
  `status`, `completed_at` and review references, but include every input whose
  change invalidates the review.
- R8: The spec digest shall cover requirements, technical plan, amendments,
  requirement trace, declared/observed artifacts, delivery targets and the exact
  validation-result identities used for closeout.
- R9: The milestone digest shall cover ordered membership, member spec closeout
  digests, milestone exit criteria and their evidence; changing or adding a
  member spec shall make the macro review stale.
- R10: The roadmap digest shall cover ordered milestone closeout digests, roadmap
  outcome, cut criteria and their evidence; any stale child review shall make
  the roadmap review stale.
- R11: `pose review-check` shall validate review structure, profile coverage,
  finding dispositions, reviewer policy, evidence resolution and digest
  freshness, and shall explain every blocking or stale condition.
- R12: `pose closeout-check` shall compose structural checks, deterministic
  validation, artifact/surface evidence when applicable, current review and
  finding disposition for spec, milestone and roadmap refs.
- R13: A milestone shall remain open after all member specs become `done` until
  its macro review and exit criteria pass; a roadmap shall reject `status: done`
  until all milestone closeouts and its roadmap review are current and approved.
- R14: A macro-review gap shall create or link bounded remediation work inside
  the same roadmap scope; membership and digest changes shall automatically
  invalidate the prior macro review and require re-review.
- R15: A policy-controlled `continuous-closeout` mode shall persist the terminal
  scope and instruct orchestration to repeat implement, validate, review and
  remediate until `pose closeout-check` reports terminal success or a genuine
  approval/external blocker.
- R16: Continuous mode shall expose the next governed action and reason without
  treating elapsed time, token limits or one completed child scope as terminal.
- R17: Reviewer-independence policy shall support same-actor separate review
  execution, different actor and mandatory human approval profiles, selected by
  scope risk and module criticality.
- R18: CLI JSON and a project-scoped read-only MCP view shall expose review state,
  current/stale decisions, findings, closeout blockers, next action and terminal
  state using the same schema.
- R19: Manual lifecycle edits shall remain possible for Git workflows, but
  `pose check --strict` shall reject any `done` scope that lacks a current review
  required by the adopted policy; an optional `pose close` command shall perform
  the same gate before applying an atomic lifecycle transition.

### Non-functional
- Produce byte-stable scope digests, finding IDs and ordered blocker output for
  identical repository state and policy.
- Compute closeout state without network access and without executing review
  evidence as code.
- Keep digest construction bounded by the addressed scope and referenced
  evidence.
- Explain the complete roll-up path from a blocked roadmap to the smallest spec,
  criterion, finding or stale evidence that requires action.
- Make repeated `review-check` and `closeout-check` calls idempotent.

### Security
- Confine every scope, report and evidence path to the authorized project root.
- Validate structured refs and never evaluate commands embedded in review text.
- Reuse project-scope MCP authorization and current redaction rules.
- Store reviewer aliases or execution IDs only; prohibit email, personal name,
  prompt contents and unbounded model transcripts in structured review records.
- Require stronger reviewer independence for security-critical modules through
  policy rather than an agent-controlled override.
- Prevent `continuous-closeout` from granting new credentials, approvals,
  external writes or destructive authority.

### Compatibility
- Keep current review Markdown reports readable as narrative evidence but do not
  infer approval from unchecked “Human Review Needed” boxes.
- Warn for legacy `done` scopes without review attestations during
  observability rollout.
- Activate strict review closeout by adoption date and scope policy before a
  repository schema bump makes it the scaffold default.
- Preserve current spec and roadmap lifecycle values and current validation
  behavior when review policy is absent.
- Treat existing milestones as derived open/closed projections; do not require a
  hand-maintained milestone status field.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: review artifact parser, profile model, canonical
  scope digest, freshness evaluation and hierarchical closeout projection.
- `pose-mcp/internal/cli`: `review-check`, `closeout-check`, optional `close`,
  lifecycle lint integration, roadmap readiness and explain output.
- `pose-mcp/internal/mcpserver`: project-scoped closeout-state projection and
  catalog golden contract.
- `.pose/review-profiles`, `.pose/policy`, templates, workflows, rules, skills,
  POSE manual, locales and embedded scaffold.

### API/contract changes
- Add immutable review attempts under `.pose/reviews/` with flat frontmatter and
  machine-parseable criteria and findings:

  ```text
  ---
  review_id: rvw-2026-08-02-001
  scope: spec:example
  scope_digest: sha256:...
  profile: spec-closeout@1
  reviewer: agent:review-run-42
  decision: approved
  reviewed_at: 2026-08-02T15:00:00Z
  supersedes:
  ---

  ## Criteria
  - CR1 [passed] report:validation.md

  ## Findings
  - F1 [resolved] severity:high evidence:test:regression action:fixed
  ```

- Add versioned profiles for `spec-closeout`, `milestone-integration` and
  `roadmap-outcome`.
- Add `pose review-check scope-ref`, `pose closeout-check scope-ref` and
  `pose close scope-ref` with human and JSON outputs.
- Add a read-only MCP tool such as `pose_closeout_state` only after CLI/MCP
  schemas share one golden fixture.
- Add execution policy fields for terminal scope, remediation behavior,
  permitted in-scope spec creation and explicit stop conditions.

### Data/storage changes
- Store each finalized review attempt as an immutable Markdown artifact; create
  a new attempt with `supersedes` after remediation instead of editing approval
  history.
- Generate review projections from artifacts and profiles; allow the later
  delivery-integrity graph to ingest reviews and `reviewed-by`/`gates` edges
  without making the review mechanism depend on artifact provenance.
- Derive milestone closure from member spec closeouts, criteria and macro review;
  do not add manually mutable milestone status.
- Extend roadmap projections with derived milestone and roadmap closeout state.

### Canonical digest inputs
- Spec: canonical requirements/plan/trace, amendments, closeout evidence IDs and
  governed delivery declarations; exclude lifecycle transition metadata and the
  review artifact itself.
- Milestone: canonical milestone definition, ordered member spec closure digests
  and exit-criterion evidence IDs.
- Roadmap: canonical outcome/cut criteria, ordered milestone closure digests and
  roadmap-level evidence IDs.
- Every level: profile version and applicable policy digest.

### Technical risks
- A self-review can become ceremonial; independence profiles and separate review
  execution identities must be visible and enforceable.
- Digest normalization can omit meaningful inputs or create false staleness;
  golden fixtures must pin included and excluded fields.
- Macro review can discover work after child specs close; remediation membership
  must preserve history instead of silently rewriting a completed spec.
- Continuous mode can drift in scope; every remediation spec must cite a review
  finding and remain inside the selected roadmap and public-contract boundary.
- Review artifacts can accumulate; indexes and housekeeping need deterministic
  supersession without deleting history.

### Rollout
1. Scaffold profiles and review artifacts; project review state and stale
   decisions without changing closeout exit codes.
2. Require spec reviews for newly started specs under opt-in policy.
3. Enable milestone and roadmap macro reviews after member specs expose closure
   digests.
4. Enable `continuous-closeout` only for explicitly selected terminal scopes and
   permitted remediation actions.
5. Make hierarchical review the default for new repository schemas while legacy
   completed scopes remain visible as warnings.

## 4. Tasks

### Planning
- [ ] Record an ADR for immutable review attempts, canonical scope digests,
  derived milestone closure and autonomous remediation boundaries before this
  spec becomes `in-progress`.
- [ ] Convert current false-closeout behavior into fixtures: done spec without
  review, all-done milestone without macro review, manually done roadmap without
  outcome review and stale approval after remediation.
- [ ] Freeze review artifact, profile, policy and closeout-state schemas with
  golden files.
- [ ] Define the bootstrap review used to close this first mechanism before its
  own strict policy becomes active.

### Implementation
- [ ] Implement review/profile parsers, typed scope validation and immutable
  supersession chains.
- [ ] Implement canonical scope digests with explicit include/exclude fixtures.
- [ ] Implement criterion, finding, accepted-risk and independence-policy gates.
- [ ] Implement `pose review-check` with human and JSON explanations.
- [ ] Implement hierarchical `pose closeout-check` for specs, milestones and
  roadmaps.
- [ ] Change milestone readiness from “all specs done” to derived reviewed
  closure and guard roadmap `status: done`.
- [ ] Implement optional atomic `pose close` transitions without weakening Git
  review workflows.
- [ ] Implement `continuous-closeout` policy and next-action projection with
  bounded remediation semantics.
- [ ] Add project-scoped MCP closeout state and golden catalog parity.
- [ ] Update review/feature/closeout workflows and skills, templates, rules,
  POSE manual, locales, changelog and embedded scaffold.

### Validation
- [ ] Run focused parser, digest, finding, lifecycle, readiness and MCP tests.
- [ ] Run negative tests for path escape, evidence execution, forged reviewer
  overrides, stale digests, scope drift and unauthorized project refs.
- [ ] Run an end-to-end loop that requests changes, adds bounded remediation,
  revalidates, supersedes the stale review and closes each hierarchy level.
- [ ] Run the full Go suite and embedded-distribution parity test.
- [ ] Run strict POSE structure, spec lint and module validation gates.

## 5. Decisions

### Decision 1: Review approval is digest-bound evidence
- Date: 2026-08-02
- Context: A review checkbox or prose report can remain apparently valid after
  the implementation or scope changes.
- Options considered: checkbox in each artifact; mutable latest-review fields;
  immutable attempts bound to canonical scope digests.
- Decision: Store immutable review attempts and require a current digest match
  at closeout.
- Rationale: Freshness becomes falsifiable and history remains reviewable.
- Consequences: Any relevant change requires a new review attempt; digest
  normalization becomes a release-gated contract.

### Decision 2: Derive milestone closure
- Date: 2026-08-02
- Context: Milestones currently have no lifecycle state and become satisfied
  arithmetically when all specs are `done`.
- Options considered: add a manually edited milestone status; continue deriving
  from spec status only; derive from child closeouts, criteria and macro review.
- Decision: Derive milestone closure from reviewed children plus an approved
  milestone review.
- Rationale: A manually mutable flag recreates the roadmap weakness at a new
  level.
- Consequences: Adding remediation to a milestone invalidates its macro review
  automatically.

### Decision 3: Persist autonomy as a bounded terminal policy
- Date: 2026-08-02
- Context: Users repeatedly restate that agents should continue through review
  findings until an entire scope is reviewed and complete.
- Options considered: rely on prompt wording; let agents infer persistence;
  version a terminal scope and allowed continuation transitions.
- Decision: Add `continuous-closeout` as explicit policy consumed by
  orchestration.
- Rationale: The stop condition becomes machine-readable without broadening
  authority.
- Consequences: POSE exposes state and next action; Conductor or the active agent
  performs the loop and still stops for genuine approval or external blockers.

## 6. Validation

### Strategy
Use repository fixtures that begin with green deterministic checks but missing,
rejected or stale reviews. Prove that each hierarchy level stays open, that a
finding creates bounded remediation, and that only fresh approval for the
recomputed digest enables closure. Include single-agent separate-pass and
different-actor policies.

### Deterministic checks
- Test: `go -C pose-mcp test ./internal/pose ./internal/cli ./internal/mcpserver -run 'Review|Closeout|ScopeDigest|MilestoneClosure|ContinuousCloseout' -count=1`.
- Full suite: `go -C pose-mcp test ./... -count=1`.
- Scaffold parity: `go -C pose-mcp test ./internal/scaffold -run TestEmbeddedDistMatchesPoseDist -count=1`.
- Structure: `pose check --strict`.
- Spec readiness: `pose lint-spec pose-hierarchical-review-closeout --ready-check`.
- Spec lint: `pose lint-spec pose-hierarchical-review-closeout --strict`.
- Delivery gate: `pose validate --strict --module pose-mcp --report`.

### Execution log
- 2026-08-02, planning: current review workflow, report template, spec closeout,
  milestone readiness and roadmap structural checks inspected; implementation
  checks remain pending.
- 2026-08-02, planning validation: readiness and strict lint passed;
  `pose validate --strict --module pose-mcp`, the full Go checks and embedded
  scaffold parity passed after regeneration.

### Results summary
- Successes: The planning contract separates validation from review, defines
  hierarchical freshness, persists the autonomous terminal condition and passes
  readiness, strict spec lint, module validation and scaffold parity.
- Failures: None claimed as implementation evidence.
- Warnings: Current project state is stale by age and reports a hand-edited
  Architecture section; refresh is outside this planning-only change.

### Requirement trace
Requirement trace will be populated only after implementation evidence exists;
this draft does not claim any requirement as satisfied.

### Known gaps
- Exact profile criterion IDs and the bootstrap-review exception must be frozen
  in the implementation ADR.
- The first delivery-integrity graph schema must decide whether review artifacts
  are embedded or referenced while preserving this spec's independent rollout.

## 7. Final Report

### Delivered scope
Planning artifact only: this draft defines hierarchical review and autonomous
closeout as a future POSE mechanism. No runtime capability or closure guarantee
is claimed.

### Files and modules changed
- `.pose/specs/pose-hierarchical-review-closeout/spec.md`.
- `.pose/roadmaps/delivery-integrity.md` membership and ordering.

### Validation executed
- Commands: repository review/closeout contract inspection; `pose lint-spec
  pose-hierarchical-review-closeout --ready-check`; `pose lint-spec
  pose-hierarchical-review-closeout --strict`; `pose validate --strict --module
  pose-mcp`; embedded scaffold parity test.
- Result: spec, Go test, vet, module validation and parity checks passed; global
  `pose check --strict` remains blocked by the pre-existing broken POSE.md
  reference to absent `.pose/feedback`.

### Residual risks
- A mechanically complete review can still be shallow; profiles, independence
  policy and evidence quality remain essential review concerns.

### Follow-ups
No follow-up is opened by this planning revision; implementation tasks remain
inside this draft spec.
