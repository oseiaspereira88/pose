---
slug: pose-component-aware-review-plans
status: done
created_at: 2026-08-13
completed_at: 2026-08-13
supersedes:
depends_on: pose-hierarchical-review-closeout
priority: 10
components: pose-mcp
delivers: governance:component-aware-review-plans
---

# Spec: Component-aware, tool-guided review plans

## 1. Intent

### Goal
Make review a first-class POSE mechanism that deterministically composes the
right criteria, reviewer policy, native tool recommendations and evidence
expectations for the actual components and domains affected by a scope.

### Business value
The current review policy selects one static profile for each terminal scope
kind: spec, milestone or roadmap. It does not distinguish a React frontend from
a Go backend, does not compose coverage for a multi-component change and does
not tell the reviewer which POSE tools can produce relevant evidence. This
creates ceremonial reviews precisely where POSE already has stronger context in
the repository map, component metadata, rules, assessment engines, validation
matrix, artifact ledger and delivery graph.

A resolved review plan turns that context into an inspectable contract before
review starts. Reviewers retain judgment, but receive scope-specific criteria,
tool guidance and evidence expectations with provenance. Closeout can then
prove that every affected component and boundary was reviewed under the plan
that was current for the reviewed digest.

This scope was requested in
[issue #13](https://github.com/oseiaspereira88/pose/issues/13).

### Constraints
- Preserve offline and deterministic operation.
- Keep review execution separate from implementation execution.
- Recommend tools without executing them or expanding caller authority.
- Reuse observed POSE context; do not infer components from names alone when a
  governed mapping exists.
- Preserve immutable review attempts and digest-bound freshness.
- Keep all selected paths and evidence confined to the authorized project root.
- Treat policy, profile and effective-plan schemas as public versioned
  contracts.
- Require an ADR before implementation because this changes the structural
  review/profile contract.

### Non-goals
- Replace reviewer judgment with deterministic checks.
- Execute shell commands embedded in repository-owned profiles.
- Make every recommended tool mandatory.
- Invent component ownership, criticality, language or domain when mapping is
  missing.
- Add framework-specific static analyzers to the POSE core.
- Implement the feature in the change that creates this draft spec.
- Change lifecycle semantics unrelated to review planning and closeout.

## 2. Requirements

### Functional
- R1: When a caller requests review for a typed scope, POSE shall resolve one
  immutable effective review plan before an attempt is recorded.
- R2: The effective plan shall include the scope ref, scope digest, plan digest,
  base profile, selected overlays, affected components, component provenance,
  applicable rules, criteria, tool recommendations, evidence expectations,
  reviewer-independence requirement, warnings and blockers.
- R3: POSE shall derive affected components from explicit spec components,
  Git-observed artifacts attributed to the spec, declared delivery targets and
  the repository component map; each mapping shall expose its source rather
  than collapsing to an unexplained component list.
- R4: POSE shall resolve component language, domain, owner, criticality and
  validation profile from governed module metadata when available, and shall
  represent unavailable metadata explicitly.
- R5: Profile selection shall start with the terminal-scope base profile and
  add matching language, domain, component, delivery-kind and risk overlays;
  an overlay shall not silently remove a base criterion or weaken reviewer
  independence.
- R6: Selection and composition shall use a documented deterministic order:
  base scope, language overlays sorted by ID, domain overlays sorted by ID,
  component overlays sorted by component path, delivery/risk overlays sorted
  by ID, then the synthetic cross-component boundary criterion.
- R7: Criteria with the same stable ID shall deduplicate only when their
  semantic contract is identical; incompatible descriptions, evidence classes,
  rules or requiredness shall block plan resolution with both provenances.
- R8: A scope affecting multiple components shall include the union of every
  component plan and at least one integration criterion for each observed
  component boundary or inter-component contract.
- R9: Policy shall configure whether unmapped, ambiguous or metadata-incomplete
  components produce a warning or a blocker; strict component coverage shall
  fail closed without fabricating a fallback match.
- R10: Tool recommendations shall use stable catalog IDs and contain a safe argv
  template, applicability rationale, expected evidence classes, required or
  recommended status, source criterion and preconditions.
- R11: The built-in tool catalog shall cover, when applicable, `pose suggest
  review --path`, `pose assess discover`, `pose assess integrate`, `pose assess
  tech-debt`, `pose validate`, `pose artifact-check`, `pose surface-check`,
  `pose roadmap-check`, `pose history-check`, `pose knowledge-check`, `pose
  skills-check`, `pose review-check` and closeout checks.
- R12: POSE shall never execute a recommendation while resolving or displaying
  a plan; execution remains an explicit caller action under existing
  authorization.
- R13: The CLI shall expose `pose review-plan <scope> [--json] [--explain]` and
  print both the effective plan and why each profile, criterion, rule and tool
  was selected.
- R14: MCP shall expose the same project-scoped read-only plan through
  `pose_review_plan`, with schema and values shared with the CLI projection.
- R15: `pose review record` shall scaffold criteria from the effective plan,
  record its digest and reject an explicitly supplied stale plan digest.
- R16: `pose review-check` shall require exact effective-plan coverage and shall
  make an approved attempt stale when policy, applicable profiles, component
  mapping, governed artifacts, delivery targets, relevant rules or required
  evidence expectations change.
- R17: The attempt shall record evidence actually used and dispositions for
  required criteria and every effective-plan tool. Required evidence-collection
  tools shall block approval unless they passed with matching evidence;
  recommended tools that were not used shall remain visible without becoming
  blockers. Completion tools shall be recorded as deferred until the immutable
  attempt exists, then enforced by `review-check` and `closeout-check`.
- R18: Built-in overlays shall demonstrate materially distinct frontend and
  backend plans. Frontend coverage shall include user-visible behavior,
  accessibility, state/network failure and surface reachability; backend
  coverage shall include contracts, input/error behavior, concurrency where
  applicable, observability and integration impact.
- R19: Repositories shall be able to add versioned component overlays without
  modifying POSE core, while only referencing known rules, evidence classes and
  native tool catalog IDs.
- R20: Workflows and the `pose-review` skill shall require plan inspection at
  review start and guide the reviewer through required and recommended tools in
  plan order.

### Non-functional
- R21: Identical repository state, policy and scope shall produce byte-stable
  plan JSON, ordering and SHA-256 digest across repeated runs.
- R22: Plan resolution shall remain bounded to the addressed scope, attributed
  artifacts, registered component metadata and referenced policy/profile files.
- R23: Human and JSON output shall expose the smallest actionable reason for
  unmapped components, selector conflicts, unknown tools and missing evidence.
- R24: Resolution shall remain read-only and idempotent; no assessment,
  validation, review or index artifact may be mutated implicitly.

### Security
- R25: Profile tool entries shall reference an allowlisted native catalog ID and
  typed arguments; arbitrary executables, shell fragments and control operators
  shall be rejected.
- R26: Component, profile, rule and evidence paths shall be canonicalized and
  confined beneath the authorized project root, including symlink checks.
- R27: A repository profile shall not weaken mandatory human or different-actor
  independence selected by a more restrictive policy or criticality overlay.
- R28: CLI and MCP projections shall preserve existing redaction and shall not
  expose personal identity, prompt content, secrets or unbounded command output.

### Compatibility
- R29: Schema-v1 review policies, profiles and attempts shall remain readable
  and shall resolve to their current generic behavior until a repository opts
  into component-aware planning.
- R30: Migration to the new policy schema shall be explicit, dry-run capable and
  adoption-date aware; it shall not retroactively invalidate exempt legacy done
  scopes.
- R31: Existing approved attempts shall remain auditable; an attempt shall be
  required to supersede only when its governed open scope resolves under the
  new plan after adoption.
- R32: CLI additions shall be backward compatible, and the MCP catalog change
  shall update the golden contract, documentation and capability metadata in
  the same implementation.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: selector model, component resolution, effective-plan
  composition, digest binding and review-check integration.
- `pose-mcp/internal/cli`: plan/explain command and plan-aware review recording.
- `pose-mcp/internal/mcpserver`: project-scoped read-only plan projection and
  catalog contract.
- `.pose/policy` and `.pose/review-profiles`: schema-v2 opt-in, selectors,
  overlays and native tool catalog references.
- `.pose/workflows`, `.agents/skills`, `POSE.md` and docs-site: operational
  review sequence and tool guidance.
- Locales and embedded scaffold: parity for every distributed contract.

### Consulted context
- `knowledge:contract-baseline-handoff` for CLI/MCP catalog, project-scope and
  validation contract expectations.
- `adr:2026-08-02-immutable-hierarchical-review-and-closeout-evidence` for
  immutable attempts, scope digests and hierarchical closeout.
- `spec:pose-hierarchical-review-closeout` for the current profile, policy and
  review-attempt model that this spec extends.
- `issue:13` for the reported component-awareness and tool-guidance gap.
- `adr:2026-08-12-component-aware-effective-review-plans` ratifies typed
  overlays, deterministic composition, the closed tool catalog and schema-v1
  migration.
- `knowledge:pr15-component-aware-review-provenance` records the remediation
  evidence and the independent-review handoff for PR #15.

### Artifacts
- modified: .pose/specs/pose-component-aware-review-plans/spec.md
- created: .pose/adr/2026-08-12-component-aware-effective-review-plans.md
- created: .pose/knowledge/2026-08-13-decision-log-adr-component-aware-review-plans-review.md
- renamed: .pose/changelogs/unreleased/pose-component-aware-review-plans.md -> .pose/changelogs/v1.1.0/pose-component-aware-review-plans.md
- created: pose-mcp/internal/pose/review_plan.go
- created: pose-mcp/internal/pose/review_plan_test.go
- created: pose-mcp/schemas/v1/review-plan.schema.json
- created: .pose/review-profiles/frontend-review.json
- created: .pose/review-profiles/backend-review.json
- modified: .pose/policy/review.json
- modified: .pose/review-profiles/spec-closeout.json
- removed: .pose/changelogs/unreleased/review-legacy-done-scope-exemption.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/project-state.md
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/releases.json
- modified: .pose/indexes/repo-map.json
- modified: .pose/indexes/spec-graph.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/reports/2026-08-13-standard-validate-native.md
- modified: .pose/reports/history/standard-validate-native.jsonl
- modified: .pose/workflows/review.md
- modified: .agents/skills/pose-review/SKILL.md
- modified: POSE.md
- modified: docs-site/docs/mcp.md
- modified: locales/pt-BR/POSE.md
- modified: locales/pt-BR/.pose/workflows/review.md
- modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/closeout_tool_test.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/scaffold/dist/.pose/policy/review.json
- modified: pose-mcp/internal/scaffold/dist/.pose/review-profiles/spec-closeout.json
- created: pose-mcp/internal/scaffold/dist/.pose/review-profiles/frontend-review.json
- created: pose-mcp/internal/scaffold/dist/.pose/review-profiles/backend-review.json
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/review.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/review.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/releases.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/repo-map.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
- modified: pose-mcp/schemas/README.md

### Delivery targets
- governance:component-aware-review-plans module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### Proposed contract model
Introduce a read-only `ReviewPlan` projection with these stable concepts:

- `scope`, `scope_digest`, `plan_digest` and `base_profile`;
- `components[]` with path, kind, language, domain, criticality, metadata status
  and mapping provenance;
- `selected_profiles[]` with selector match, source path and composition order;
- `criteria[]` with stable ID, requiredness, rules, evidence classes and every
  contributing profile;
- `tools[]` with catalog ID, typed argv template, applicability, requiredness,
  expected evidence and source criterion;
- `independence`, `warnings`, `blockers` and ordered explain events.

Extend schema-v2 review profiles with optional selectors and tool references:

```json
{
  "schema_version": 2,
  "id": "frontend-review",
  "version": 1,
  "scope": "spec",
  "selectors": {
    "languages": ["javascript", "typescript"],
    "domains": ["frontend"],
    "component_ids": []
  },
  "criteria": [],
  "tools": [
    {
      "id": "surface-check",
      "requiredness": "recommended",
      "preconditions": ["delivery-target-declared"],
      "evidence_classes": ["reachability"]
    }
  ]
}
```

The concrete selector grammar and profile inheritance rules must be ratified in
the required ADR before implementation. The runtime shall parse selectors as
typed data, never as an expression language or executable string.

### Resolution algorithm
1. Parse and authorize the typed scope.
2. Load the base scope policy/profile and adoption behavior.
3. Resolve explicit components and observed artifact/delivery ownership.
4. Join each path to the longest matching registered component root.
5. Load governed metadata and applicable domain rules.
6. Match typed overlay selectors and reject ambiguous or invalid matches.
7. Compose profiles in the R6 order, preserving criterion provenance.
8. Add integration coverage for multi-component boundaries and contracts.
9. Resolve native tool catalog entries and typed argument templates.
10. Canonicalize the projection and compute `plan_digest`.
11. Include the plan digest in the review attempt and effective scope digest.

### Native tool recommendation matrix

| Trigger | Tool catalog ID | Expected contribution |
|---|---|---|
| Every mapped component | `suggest-review` | Workflow, skill, cumulative rules and validation trail for the component path. |
| Review start or unknown module shape | `assess-discover` | Component structure, language, LOC and visible debt baseline. |
| More than one component or public contract | `assess-integrate` | Providers, consumers, protocol boundaries and uncovered integration gaps. |
| Code review | `assess-tech-debt` | New or unresolved TODO, FIXME, panic and stub findings. |
| Attributed implementation diff | `artifact-check` | Declared versus Git-observed artifact provenance. |
| Any implementation scope | `validate` | Registered deterministic checks for the affected module/stack. |
| Surface/capability delivery | `surface-check` | Composition, reachability and fresh evidence. |
| Roadmap outcome review | `roadmap-check` | Milestone and outcome criteria roll-up. |
| Governance/history artifacts changed | `history-check` | Append-only history integrity. |
| Knowledge artifacts changed or consulted | `knowledge-check` | Frontmatter, sensitivity and lifecycle governance. |
| Skills/workflows changed | `skills-check` | Agent Skills and distributed contract conformance. |
| Review completion | `review-check`, `closeout-check` | Exact plan coverage, freshness and remaining blockers. |

Repository profiles may select only cataloged tool IDs. The runtime owns argv
construction, path confinement and parameter typing. A recommendation can be
`required` only through trusted policy/profile data and shall still require an
explicit caller execution.

### Data/storage changes
- Add schema-v2 policy/profile fields for opt-in overlay selection and strict
  unmapped-component behavior.
- Add `plan_digest` and optional effective-plan reference to new immutable
  review attempts while retaining the base `profile` field.
- Do not persist generated plans by default; canonical JSON output is
  reproducible from governed inputs. Evidence records may reference an exported
  plan report when a workflow chooses to version it.
- Keep existing review files immutable and readable.

### API/contract changes
- Add CLI command `pose review-plan <scope> [--json] [--explain]`.
- Add read-only MCP tool `pose_review_plan` with required `scope` and existing
  project selection/authorization semantics.
- Add a versioned review-plan JSON schema shared by CLI and MCP.
- Extend `pose review record` with optional `--plan-digest`; dry-run output shall
  always show the resolved plan digest.
- Extend review attempts additively with `plan_digest`.
- Bump review policy/profile schema only through an explicit upgrade path.

### Technical risks
- Component maps can be stale or coarser than the implementation path; expose
  provenance and ambiguity instead of selecting silently.
- Too many overlays can create noisy or contradictory plans; use stable IDs,
  deterministic composition and fail-visible conflicts.
- Tool guidance can become a disguised command-execution surface; keep a closed
  native catalog and typed argv construction.
- Recomputing plans can make reviews stale after unrelated metadata edits;
  include only selectors and metadata actually consumed by the effective plan.
- Framework labels such as frontend/backend can overfit; ship useful defaults
  while allowing repository-owned component overlays.
- Multi-component specs can under-declare components; reconcile declared and
  observed evidence and surface the mismatch.
- Schema-v1 repositories need a low-friction migration; keep generic behavior
  until explicit opt-in and supply a dry-run upgrade explanation.

### Rollout
1. Ratify schema, selector precedence, trust boundary and migration in an ADR.
2. Add read-only plan resolution behind schema-v2 opt-in.
3. Ship human/JSON CLI explain output with frontend, backend and multi-component
   fixtures.
4. Add MCP projection and freeze CLI/MCP parity in the golden catalog.
5. Bind new review attempts to plan digests without changing schema-v1
   closeout behavior.
6. Update workflows, skill, manual, locales and embedded scaffold.
7. Enable strict component coverage only through explicit repository policy.

## 4. Tasks

### Planning
- [x] Record the required ADR for the effective-plan schema, selector
  precedence, trusted tool catalog and schema-v1 migration.
- [x] Freeze frontend, backend, multi-component, unmapped and conflicting
  selector fixtures before runtime implementation.
- [x] Define the versioned JSON schema and CLI/MCP golden projection.
- [x] Confirm every native tool ID and typed argument contract against the
  current CLI catalog.

### Implementation
- [x] Implement component resolution with mapping provenance and ambiguity
  handling.
- [x] Implement typed selectors, deterministic overlay composition and conflict
  detection.
- [x] Implement the closed native tool recommendation catalog.
- [x] Implement canonical effective plans and stable plan digests.
- [x] Add `pose review-plan` human, JSON and explain output.
- [x] Add `pose_review_plan` with project-scoped authorization.
- [x] Bind new attempts and `review-check` to the effective plan digest.
- [x] Add schema-v1 compatibility and schema-v2 adoption-aware migration.
- [x] Update review workflow, skill, policy/profile references and manuals.
- [x] Regenerate and verify locales and embedded scaffold.

### Validation
- [x] Prove distinct frontend and backend effective plans from the same base
  scope profile.
- [x] Prove multi-component union and required integration-boundary coverage.
- [x] Prove deterministic composition and digest stability.
- [x] Prove stale-review invalidation for consumed policy/component changes and
  stability for unrelated metadata changes.
- [x] Prove rejection of arbitrary commands, selector ambiguity, path escape,
  symlink escape, unknown tools and independence weakening.
- [x] Prove schema-v1 compatibility and adoption-date behavior.
- [x] Run focused, full-suite, contract, scaffold and strict POSE gates.

## 5. Decisions

### Decision 1: Resolve a plan, not a single component profile
- Date: 2026-08-13
- Status: accepted by
  `adr:2026-08-12-component-aware-effective-review-plans`.
- Context: A spec can affect several components and still needs scope-level
  correctness, security and compatibility criteria.
- Options considered: select one most-specific profile; review each component
  independently; compose one effective plan with provenance.
- Decision: Compose a scope base profile with every applicable overlay into one
  immutable effective plan.
- Rationale: Composition preserves cross-cutting requirements and makes
  multi-component completeness measurable.
- Consequences: Profile conflicts become explicit contract errors and plan
  canonicalization becomes release-gated.

### Decision 2: Recommend only cataloged native tools
- Date: 2026-08-13
- Status: accepted by
  `adr:2026-08-12-component-aware-effective-review-plans`.
- Context: Repository-defined commands would make review planning an execution
  and injection surface.
- Options considered: free-form command strings; documentation-only prose;
  stable tool IDs resolved by POSE.
- Decision: Store stable native tool IDs and typed applicability data in
  profiles; let POSE render safe argv templates without executing them.
- Rationale: Reviewers get actionable guidance while policy remains auditable
  and non-executable.
- Consequences: New recommendation types require catalog evolution and contract
  tests.

### Decision 3: Bind approval to the effective plan digest
- Date: 2026-08-13
- Status: accepted by
  `adr:2026-08-12-component-aware-effective-review-plans`.
- Context: An approval under a generic plan must not remain current after a
  newly mapped critical component adds required coverage.
- Options considered: bind only to scope content; store a narrative plan
  snapshot; bind to a canonical effective-plan digest.
- Decision: Include the effective-plan digest in new immutable attempts and in
  freshness evaluation.
- Rationale: Changes to actually consumed review policy become falsifiable
  staleness without invalidating reviews for unrelated metadata.
- Consequences: Canonical input selection needs positive and negative golden
  tests.

## 6. Validation

### Strategy
Use a neutral fixture repository containing a React frontend, Go backend,
shared REST contract, one multi-component spec, governed delivery targets and
an intentionally unmapped path. Resolve plans before recording reviews, compare
human/JSON/MCP projections, record approvals, mutate one consumed input at a
time and prove exact freshness behavior. Run every recommendation test in
read-only mode and assert that plan resolution creates no files or processes.

### Deterministic checks
- Required — unit: `go -C pose-mcp test ./internal/pose -run
  'ReviewPlan|ReviewProfile|ComponentSelector|PlanDigest' -count=1`.
- Required — CLI: `go -C pose-mcp test ./internal/cli -run
  'ReviewPlan|ReviewRecord|ReviewCheck' -count=1`.
- Required — MCP: `go -C pose-mcp test ./internal/mcpserver -run
  'ReviewPlan|Closeout|Catalog' -count=1`.
- Required — negative/security: `go -C pose-mcp test ./internal/pose ./internal/cli
  ./internal/mcpserver -run
  'ReviewPlan.*(Traversal|Symlink|Command|Ambiguous|Unknown|Independence)'
  -count=1`.
- Required — full suite: `go -C pose-mcp test ./... -count=1` and `go -C pose-mcp vet
  ./...`.
- Required — integration assessment: `pose assess integrate` after CLI/MCP and file
  contract changes.
- Required — scaffold parity: `go -C pose-mcp test ./internal/scaffold -run
  TestEmbeddedDistMatchesPoseDist -count=1`.
- Required — structure: `pose check --strict`, `pose history-check --strict`, `pose
  skills-check --strict` and `pose lint-spec pose-component-aware-review-plans
  --strict`.
- Required — delivery gate: `pose validate --strict --module pose-mcp --report` followed by
  `pose surface-check --spec pose-component-aware-review-plans --strict`.

### Execution log
- 2026-08-13, post-merge closeout: PR #16 review identified successive
  provenance gaps after the preparation, approval, state-refresh and mandatory
  assessment and post-closeout discovery commits. Append-only change set
  `cs-94bce6d13479` supersedes the intermediate records and covers the exact
  24-path retained branch range
  `504fdf3c8de09c7136775d33829446d608561483..d1d0268b44ef6ea69520068af026d5329fb80277`.
  The branch also retains the provider merge object `b9807d2` as ancestry, so
  both the independently reviewed merge content and final discovery baseline
  remain reproducible after the transient provider ref moves.

### Risk-based cases

| Scenario | Command | Expected evidence |
|---|---|---|
| Frontend-only spec | `go -C pose-mcp test ./internal/pose -run TestReviewPlanSelectsDistinctFrontendBackendAndBoundaryCoverage -count=1` | Frontend criteria and surface/reachability guidance appear; backend-only criteria do not. |
| Backend-only spec | `go -C pose-mcp test ./internal/pose -run TestReviewPlanSelectsDistinctFrontendBackendAndBoundaryCoverage -count=1` | Backend contract/error/concurrency guidance appears; frontend accessibility criteria do not. |
| Frontend plus backend | `go -C pose-mcp test ./internal/pose -run TestReviewPlanSelectsDistinctFrontendBackendAndBoundaryCoverage -count=1` | Both component plans and an integration-boundary criterion appear exactly once. |
| Repository component override | `go -C pose-mcp test ./internal/pose -run TestReviewPlanOrdersOverlaysByCategoryContract -count=1` | Component overlay composes after domain overlay; language/domain refs and component paths follow R6 ordering. |
| Duplicate identical criterion | `go -C pose-mcp test ./internal/pose -run TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped -count=1` | Criterion deduplicates with all contributing sources retained. |
| Conflicting criterion | `go -C pose-mcp test ./internal/pose -run TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped -count=1` | Resolution blocks with both profile refs and conflicting fields. |
| Unmapped or metadata-incomplete component | `go -C pose-mcp test ./internal/pose -run 'TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped|TestReviewPlanFailsClosedOnIncompleteMetadata' -count=1` | Warning or blocker follows policy, missing metadata is explicit and no defaulted selector value is consumed. |
| Unknown tool or free-form command | `go -C pose-mcp test ./internal/pose -run TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped -count=1` | Profile validation rejects the entry before plan creation. |
| Tool lifecycle and required coverage | `go -C pose-mcp test ./internal/pose ./internal/cli -run 'TestReviewPlanToolsFollowLifecycleOrder|TestReviewCheckRequiresCurrentEffectivePlanDigestAndCoverage|TestReviewRecordRequiresRequiredToolDispositions' -count=1` | Evidence tools precede completion gates; missing, duplicate or unevidenced required tools block approval; CLI scaffolds explicit recommended/deferred dispositions. |
| Relevant policy/component change | `go -C pose-mcp test ./internal/pose -run TestReviewCheckRequiresCurrentEffectivePlanDigestAndCoverage -count=1` | Existing attempt becomes stale because plan digest changes. |
| Unrelated owner metadata change | `go -C pose-mcp test ./internal/pose -run TestReviewPlanDigestIsDeterministicAndIgnoresUnconsumedOwner -count=1` | Plan digest and approval remain current when owner was not consumed by selection. |
| Schema-v1 repository | `go -C pose-mcp test ./internal/pose -run TestReviewPlanSchemaV1RemainsGenericAndResolutionIsReadOnly -count=1` | Custom legacy rule/evidence namespaces, generic profile behavior and existing attempt readability remain unchanged. |
| Read-only guarantee | `go -C pose-mcp test ./internal/pose -run TestReviewPlanSchemaV1RemainsGenericAndResolutionIsReadOnly -count=1` | Filesystem snapshot proves plan resolution performs no mutation or tool execution. |

### Requirement trace
- R1 [satisfied] test:TestReviewPlanCLIProjectsJSONAndPinsReviewRecord — a typed scope resolves an immutable plan before review recording and pins its digest.
- R2 [satisfied] test:TestReviewPlanSelectsDistinctFrontendBackendAndBoundaryCoverage — the projection exposes components, profiles, criteria, tools, evidence expectations, independence, warnings and blockers with provenance.
- R3 [satisfied] test:TestReviewPlanSelectsDistinctFrontendBackendAndBoundaryCoverage — explicit components, attributed artifacts, delivery targets and the repository map contribute visible component sources.
- R4 [satisfied] test:TestReviewPlanFailsClosedOnIncompleteMetadata — governed metadata is projected when present and missing/defaulted fields remain explicitly unavailable.
- R5 [satisfied] test:TestReviewPlanPreservesStricterIndependenceAndRejectsAmbiguousSelectors — typed overlays compose without removing base coverage or weakening independence.
- R6 [satisfied] test:TestReviewPlanOrdersOverlaysByCategoryContract — overlays follow the documented category and within-category order.
- R7 [satisfied] test:TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped — identical IDs conflict visibly when their semantic contracts differ.
- R8 [satisfied] test:TestReviewPlanSelectsDistinctFrontendBackendAndBoundaryCoverage — multi-component plans contain the union plus cross-component integration coverage.
- R9 [satisfied] test:TestReviewPlanFailsClosedOnIncompleteMetadata test:TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped — warning/blocker policy applies to unmapped, ambiguous and metadata-incomplete components without fallback matches.
- R10 [satisfied] test:TestReviewPlanRecommendationsAreClosedAndNonExecutable — recommendations contain catalog IDs, safe argv, rationale, evidence, requiredness and preconditions.
- R11 [satisfied] test:TestReviewPlanToolsFollowLifecycleOrder contract:pose-help-tool-catalog — applicable native assessment, validation, integrity, review and closeout tools are represented in lifecycle order.
- R12 [satisfied] test:TestReviewPlanSchemaV1RemainsGenericAndResolutionIsReadOnly — plan resolution performs no mutation or tool execution.
- R13 [satisfied] test:TestReviewPlanCLIProjectsJSONAndPinsReviewRecord — CLI human/JSON/explain projection and plan-aware recording are covered.
- R14 [satisfied] governance:component-aware-review-plans evidence:integration check:delivery-integration test:TestReviewPlanToolUsesTheSameProjectScopedProjection — MCP exposes the same confined project-scoped projection.
- R15 [satisfied] test:TestReviewPlanCLIProjectsJSONAndPinsReviewRecord test:TestReviewRecordRequiresRequiredToolDispositions — recording scaffolds effective coverage and rejects stale plan digests.
- R16 [satisfied] test:TestReviewCheckRequiresCurrentEffectivePlanDigestAndCoverage — review-check enforces exact current plan/tool coverage and invalidates consumed-input drift.
- R17 [satisfied] test:TestReviewRecordRequiresRequiredToolDispositions test:TestReviewCheckRequiresCurrentEffectivePlanDigestAndCoverage — attempts persist criterion/tool dispositions, require matching evidence and defer completion gates explicitly.
- R18 [satisfied] test:TestReviewPlanSelectsDistinctFrontendBackendAndBoundaryCoverage — built-in frontend and backend overlays produce materially distinct coverage.
- R19 [satisfied] test:TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped — repository overlays use typed selectors and only known rule/evidence/tool contracts.
- R20 [satisfied] test:TestReviewPlanToolsFollowLifecycleOrder contract:pose-review-workflow — workflow/skill plan inspection and tool execution order match the runtime lifecycle.
- R21 [satisfied] test:TestReviewPlanDigestIsDeterministicAndIgnoresUnconsumedOwner — identical consumed inputs produce stable ordering and SHA-256 digests.
- R22 [satisfied] test:TestReviewPlanSelectsDistinctFrontendBackendAndBoundaryCoverage test:TestReviewPlanRejectsComponentAndArtifactSymlinkEscapes — resolution stays bounded to the addressed scope and registered roots.
- R23 [satisfied] test:TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped test:TestReviewPlanFailsClosedOnIncompleteMetadata — output identifies the actionable conflict, unknown tool and missing metadata reason.
- R24 [satisfied] test:TestReviewPlanSchemaV1RemainsGenericAndResolutionIsReadOnly — repeated resolution is read-only and idempotent.
- R25 [satisfied] test:TestReviewPlanRecommendationsAreClosedAndNonExecutable test:TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped — arbitrary executables, shell fields and control operators are rejected.
- R26 [satisfied] test:TestReviewPlanRejectsComponentAndArtifactSymlinkEscapes test:TestReviewPlanToolRejectsMissingAndTraversalScope — component, artifact and MCP paths are canonicalized and confined.
- R27 [satisfied] test:TestReviewPlanPreservesStricterIndependenceAndRejectsAmbiguousSelectors — overlays cannot weaken mandatory-human or separate-execution policy.
- R28 [satisfied] test:TestReviewPlanToolUsesTheSameProjectScopedProjection test:TestReviewPlanToolRejectsMissingAndTraversalScope — CLI/MCP projections remain project-scoped, bounded and free of execution output.
- R29 [satisfied] test:TestReviewPlanSchemaV1RemainsGenericAndResolutionIsReadOnly test:TestReviewCheckGrandfathersOnlyCompletedPreMigrationAttempts — schema-v1 custom namespaces and pre-adoption readability remain intact.
- R30 [satisfied] test:TestReviewCheckGrandfathersOnlyCompletedPreMigrationAttempts contract:review-policy-schema-v2 — opt-in adoption is explicit and does not retroactively invalidate exempt completed scopes.
- R31 [satisfied] test:TestReviewCheckGrandfathersOnlyCompletedPreMigrationAttempts — historical approved attempts remain auditable and superseding is required only for governed open scopes.
- R32 [satisfied] governance:component-aware-review-plans evidence:integration check:delivery-integration test:TestReviewPlanCLIProjectsJSONAndPinsReviewRecord test:TestReviewPlanToolUsesTheSameProjectScopedProjection — additive CLI/MCP contracts, catalog golden, docs and capability metadata move together.

### Known gaps
- The accepted ADR deliberately permits only one-level overlays selected by
  policy; deeper inheritance remains a review trigger, not an implicit feature.
- Component boundaries are only as accurate as attributed artifacts and the
  repository map; the design makes this uncertainty visible but cannot infer
  runtime topology without evidence.
- External framework-specific analyzers remain extension concerns; the core
  plan can recommend only registered POSE tools and validation-matrix checks.

## 7. Final Report

### Delivered scope
Implemented and merge-verified schema-v2 component-aware review plans across
the Go domain, CLI, MCP, immutable attempt freshness, built-in profiles and the
distributed POSE review contract. The merged state received a superseding
independent review through PR #16, and guarded lifecycle closeout completed on
2026-08-13.

### Files and modules changed
- `pose-mcp/internal/pose/review_plan.go`: deterministic plan resolver, typed
  overlay composition, component provenance and closed native-tool catalog.
- `pose-mcp/internal/pose/review_closeout.go`: schema-v2 policy/profile parsing,
  plan-bound attempts and adoption-aware compatibility for completed reviews.
- CLI/MCP contracts: `pose review-plan`, `pose_review_plan` and plan-aware
  `pose review record` / `pose review-check`.
- Review profiles, workflows, skills, manuals, locales and embedded scaffold:
  component-specific guidance distributed with the engine.

### Validation executed
- `pose lint-spec pose-component-aware-review-plans --ready-check`: SUCCESS at
  the planning gate.
- `pose assess discover --component pose-mcp/internal/pose`: 6,677 production
  LOC, 3,767 test LOC and zero TODO/FIXME/panic/stub findings at baseline.
- Focused domain/CLI/MCP tests: SUCCESS, including negative security cases.
- `go test ./... -count=1`: SUCCESS.
- `go vet ./...`: SUCCESS.
- `pose validate --strict --module pose-mcp --report`: SUCCESS.
- `pose assess integrate`: 52 observed contracts; `pose_review_plan` is visible
  as a provider with no in-repository consumer, consistent with other public MCP
  tools.
- `pose history-check --strict`, `pose skills-check --strict` and strict spec
  lint: SUCCESS.
- `pose check --strict`: SUCCESS after regenerating the governed index and
  structured delivery evidence for the implementation commit.
- Artifact reconciliation at the implementation commit: all declared change
  paths matched the observed range; repository-wide orphan warnings remain.
- Closeout reconciliation: `cs-94bce6d13479` matches all 24 paths through the
  retained, independently reviewed commit `d1d0268b44ef6ea69520068af026d5329fb80277`;
  the append-only correction preserves every prior intermediate change set and
  the reviewed PR merge object in branch ancestry.
- PR #15 merge commit `504fdf3c8de09c7136775d33829446d608561483`
  passed 11 GitHub checks with zero failures; the Codex connector reviewed the
  final implementation commit `566e9a39a1baa291015ec20ce67b874857df4c6f`
  without additional findings.

### Residual risks
- Existing stale delivery evidence remains outside this spec; this change does
  not rewrite historical result provenance.
- The component map has no roots for repository governance/docs files, so they
  remain visible as unmapped warnings under the configured warning policy.
- Governance/docs paths without component-map roots remain visible as accepted
  low-severity review warnings; they do not weaken strict artifact, validation
  or surface gates.

### Follow-ups
- [done] Implementation authorized on 2026-08-13 after ADR and test-plan
  approval; delivery is tracked by this spec.
- [covered: pose-governance-gate-activation] The recurring `validate-native`
  signal was classified and its first 45-day intervention registered; the
  post-window verdict remains owned by the governance gate. See
  knowledge:escalation-validate-native.
