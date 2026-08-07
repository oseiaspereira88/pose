---
slug: pose-delivery-surface-assurance
status: done
created_at: 2026-08-02
completed_at: 2026-08-03
supersedes:
depends_on: pose-artifact-provenance-ledger
priority: 2
components: pose-mcp, cli, validation, roadmaps, mcp, scaffold
delivers: surface:delivery-integrity-cli, capability:delivery-integrity-graph, contract:delivery-integrity-mcp
---

# Spec: Delivery surface and composition assurance

## 1. Intent

### Goal
Prove that declared delivery targets are composed into real product entrypoints,
are backed by the artifacts that introduced them, and satisfy executable
roadmap cut criteria.

### Business value
Issue [#7](https://github.com/oseiaspereira88/pose/issues/7) demonstrates a
structural false positive: 60 specs and every build-oriented gate passed while
29 services were never constructed, two modules were absent from the shipped
binary and five tested UI components were unreachable. Artifact provenance
alone would also pass that repository. This spec completes the combined model
by making human-facing surfaces and internal composition explicit and by
preventing spec-local success from rolling up to a false roadmap completion.

### Constraints
- Build on the provenance ledger from
  `pose-artifact-provenance-ledger`; do not create an independent surface index.
- Keep the engine stack-agnostic: POSE defines evidence classes and executes
  declared checks, while each repository implements framework-specific checks.
- Treat reachability as composition from a declared production entrypoint, not
  as source-code import analysis performed by POSE.
- Reuse the structured validation executor and result contract; do not add raw
  shell commands to specs or roadmaps.
- Keep qualitative review possible through a referenced, versioned report, but
  never let manual evidence satisfy required reachability.
- Stage legacy migration and preserve existing validation behavior until policy
  or a repository schema bump activates the new gate.

### Non-goals
- Implement React, Vue, Wails, CLI, API or dependency-injection analysis in the
  POSE engine.
- Guarantee that a repository's reachability test is semantically strong; POSE
  guarantees declared class, execution, result and traceability.
- Replace unit, type, lint, build or security checks.
- Use follow-up counts as detection; follow-ups remain an amplifier for already
  perceived debt.
- Make accessibility, contrast or visual-regression classes universally
  required for non-UI targets.

## 2. Requirements

### Functional
- R1: A spec shall declare zero or more typed delivery refs in flat frontmatter
  as `delivers: surface:id, contract:id, capability:id, infrastructure:id,
  governance:id`, supporting multiple refs without assigning permanent code
  ownership to the spec.
- R2: Each delivery ref shall have one machine-parseable declaration in
  `### Delivery targets` containing a confined module, a validation profile and
  its relationship to a production entrypoint.
- R3: Module metadata and the artifact ledger shall detect changed modules
  configured as product-surface or composition roots when the spec omits a
  delivery declaration; tolerant mode shall warn and activated strict mode
  shall fail.
- R4: Validation checks shall carry a closed `evidenceClass` independent of
  their executable name, initially `build`, `unit`, `integration`,
  `reachability`, `a11y`, `design-system`, `contrast` and
  `visual-regression`.
- R5: Delivery profiles shall declare required and optional evidence classes;
  every `surface` profile shall require `reachability` plus `integration` or
  `e2e`-level evidence, and every composed `capability` profile shall require
  integration from a production composition root.
- R6: Requirement-trace evidence shall accept `evidence:unit`,
  `evidence:integration`, `evidence:e2e` or `evidence:manual` plus typed
  `surface:`, `capability:` and `artifact:` refs, preserving current refs and
  dispositions.
- R7: A `done` spec that declares a surface shall fail strict closeout unless
  every surface is referenced by a satisfied requirement with integration or
  e2e evidence and a passed required check result; `deferred-integration` shall
  be accepted only when it references an existing non-terminal spec and shall
  keep the current spec from asserting the surface as delivered.
- R8: `pose surface-check` shall validate declaration completeness, profile
  coverage, current check results and graph connectivity from observed artifacts
  through capability or contract edges to a production entrypoint.
- R9: The shared `delivery-integrity.json` shall add delivery, entrypoint,
  validation-result and roadmap nodes plus typed edges, and shall emit stable
  cross-findings including `unconnected-artifact`, `unconsumed-capability`,
  `surface-without-provenance`, `surface-without-reachability` and
  `undeclared-delivery`.
- R10: Roadmaps shall declare stable cut criteria that reference registered
  delivery refs, named validation checks or versioned manual-review reports;
  raw command text shall be invalid.
- R11: `pose roadmap-check --strict` shall reject `status: done` unless all
  member specs are terminal-successful, every cut criterion has current passing
  evidence and no required delivery-integrity finding remains in the roadmap
  scope.
- R12: CLI JSON and project-scoped read-only MCP views shall expose the same
  surfaces, composition paths, cut-criterion status and findings with explicit
  reasons for missing or stale evidence.
- R13: The default distribution shall include a UI-surface workflow, a
  cumulative surface rule and a closeout skill that teach declaration,
  reachability, accessibility and composed-delivery validation before closeout.

### Non-functional
- Produce deterministic graph ordering, path explanations and finding IDs for
  identical specs, ledger, policy and result evidence.
- Keep graph construction bounded by governed specs, artifacts, modules,
  validation results and roadmap membership.
- Show the full path that passed or broke: spec to artifact to capability or
  surface to entrypoint to check result to roadmap criterion.
- Reuse current result timeout, output-limit, isolation and skip-reason
  semantics.

### Security
- Execute only structured validation-matrix programs and arguments; reject raw
  roadmap commands and unknown evidence classes.
- Confine module, artifact, report and entrypoint refs to the authorized project
  root.
- Treat manual reports as evidence references, not executable input or trusted
  content.
- Reuse project-scope MCP authorization and redact absolute paths and sensitive
  validation output.

### Compatibility
- Keep existing specs, traces, validation matrices and roadmaps readable.
- Emit warnings for legacy specs that change configured surface roots without
  `delivers`; do not infer a false delivery ref.
- Add `evidenceClass` and delivery profiles without changing execution of checks
  that omit them.
- Activate strict closeout and roadmap completion only by policy before the
  planned repository schema bump.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose`: delivery-ref grammar, target/profile model, trace
  evidence levels and combined graph projection.
- `pose-mcp/internal/cli`: validation-matrix schema, `surface-check`,
  `roadmap-check`, lint-spec, index and check integration.
- `pose-mcp/internal/mcpserver`: read projections and catalog golden contract.
- `.pose/templates`, roadmaps, module metadata, validation matrix, English and
  pt-BR workflows/rules/skills, POSE manual and embedded scaffold.

### API/contract changes
- Add flat spec metadata and a body declaration:

  ```yaml
  delivers: surface:drill-runner, capability:session-runner
  ```

  ```text
  ### Delivery targets
  - surface:drill-runner module:frontend profile:web-ui entrypoint:app-shell
  - capability:session-runner module:internal/runner profile:composed-capability entrypoint:cmd/desktop
  ```

- Extend validation checks with `evidenceClass`; add top-level
  `deliveryProfiles` whose required classes are validated by `pose check`.
- Extend requirement-trace refs and levels without changing existing
  disposition grammar.
- Add roadmap cut-criterion grammar based only on registered refs:

  ```text
  ## Cut criteria
  - C1: surface:drill-runner check:frontend-reachability evidence:e2e
  - C2: capability:session-runner check:desktop-composition evidence:integration
  - C3: manual-review:.pose/reports/ui-audit.md
  ```

- Add focused human and JSON projections for `surface-check` and
  `roadmap-check`; both consume the same generated graph as artifact-check.

### Data/storage changes
- Extend `delivery-integrity.json` with `delivery`, `entrypoint`,
  `validation-result` and `roadmap-criterion` nodes.
- Add typed edges `delivers`, `implemented-by`, `composes`, `reaches`,
  `validated-by` and `gates`; keep artifact claim/observation edges unchanged.
- Store only named check identity, evidence class, outcome, report ref and
  freshness inputs in the graph; validation output remains in its existing
  bounded result artifact.
- Add policy for delivery-root detection, profile requirements, evidence
  freshness and staged enforcement.

### Delivery targets
- surface:delivery-integrity-cli module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
- capability:delivery-integrity-graph module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- contract:delivery-integrity-mcp module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Artifacts
- modified: .agents/skills/README.md
- modified: .pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/module-metadata.json
- modified: .pose/indexes/spec-graph.json
- modified: .pose/indexes/task-map.json
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/rules/delivery-evidence.md
- modified: .pose/specs/pose-delivery-surface-assurance/spec.md
- modified: .pose/templates/spec.md
- modified: .pose/workflows/feature.md
- modified: POSE.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/mcp.md
- modified: locales/pt-BR/.pose/rules/delivery-evidence.md
- modified: locales/pt-BR/.pose/templates/spec.md
- modified: locales/pt-BR/.pose/workflows/feature.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/skills_check_test.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_results.go
- modified: pose-mcp/internal/cli/validate_results_test.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/delivery_integrity_test.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/trace.go
- modified: pose-mcp/internal/scaffold/scaffold.go
- created: .agents/skills/pose-surface-closeout/SKILL.md
- created: .pose/policy/delivery.json
- created: .pose/rules/delivery-surface.md
- created: .pose/workflows/ui-surface.md
- created: locales/pt-BR/.agents/skills/pose-surface-closeout/SKILL.md
- created: locales/pt-BR/.pose/rules/delivery-surface.md
- created: locales/pt-BR/.pose/workflows/ui-surface.md
- created: pose-mcp/internal/cli/surface_check.go
- created: pose-mcp/internal/cli/surface_check_test.go
- created: pose-mcp/internal/mcpserver/surface_assurance_tool_test.go
- created: pose-mcp/internal/pose/delivery_surface.go
- created: pose-mcp/internal/pose/delivery_surface_test.go

### Technical risks
- A weak repository-authored reachability test can still pass; reference kits
  and negative fixtures must define minimum production-entrypoint semantics.
- Frontmatter and body declarations can drift; lint must require exact ref-set
  equality.
- Evidence can become stale after an artifact or entrypoint changes; the graph
  must bind results to the artifact/change-set digest and fail freshness.
- Mandatory UI checks can burden non-visual products; profiles, not global
  defaults, control a11y and visual classes.
- Roadmap completion can become expensive; the check should reuse indexed facts
  and execute named checks only when explicitly requested.

### Rollout
1. Parse delivery refs, profiles and evidence levels; generate graph findings in
   observability mode and publish a migration report.
2. Enable strict surface closeout per configured delivery root and require fresh
   reachability for new surface specs.
3. Enable roadmap cut criteria after all included specs use the new contract.
4. Make delivery declaration mandatory for newly scaffolded specs under the new
   schema while legacy specs remain visible as warnings.

## 4. Tasks

### Planning
- [x] Reuse the shared delivery-integrity ADR created by
  `pose-artifact-provenance-ledger` and add the exact evidence-class, graph-edge
  and roadmap-cut semantics before moving to `in-progress`.
- [x] Convert issue #7's dead UI, unconstructed services, absent modules and
  prose-only roadmap criterion into checked-in failing fixtures.
- [x] Define a control fixture where artifact-check passes but surface-check
  fails, plus the reciprocal provenance failure.

### Implementation
- [x] Extend spec parsing and linting for typed `delivers` refs and exact target
  declarations.
- [x] Extend validation-matrix checks and structured results with
  `evidenceClass` and delivery profiles.
- [x] Extend requirement trace with evidence levels and delivery/artifact refs.
- [x] Implement delivery-root detection from module metadata plus observed
  artifact changes.
- [x] Extend the combined graph and implement cross-findings with full paths.
- [x] Implement `pose surface-check` and freshness binding to change-set digests.
- [x] Add roadmap cut-criterion parsing and `pose roadmap-check --strict` without
  raw command execution.
- [x] Add CLI/MCP schema parity, golden fixtures and project-scoped read views.
- [x] Add UI-surface workflow, rule, closeout skill, reference checks, locale
  parity, POSE manual, changelog and embedded scaffold updates.

### Validation
- [x] Run focused delivery-ref, trace, matrix, graph, surface and roadmap tests.
- [x] Run negative security tests for raw commands, traversal, unknown classes,
  stale evidence and unauthorized project refs.
- [x] Run an end-to-end fixture where all build checks and artifact checks pass
  while unreachable UI and uncomposed services fail the new gate.
- [x] Run the full Go suite, catalog golden test and embedded-distribution parity.
- [x] Run strict POSE structure, both spec lints and module validation gates.

## 5. Decisions

### Decision 1: Model delivery targets, not frontend frameworks
- Date: 2026-08-02
- Context: The observed failure is most visible in UI but also affects backend
  services never constructed by the production binary.
- Options considered: add React-specific analysis; model human surfaces only;
  model typed delivery targets plus production entrypoints.
- Decision: Model stack-neutral `surface`, `contract`, `capability`,
  `infrastructure` and `governance` refs, with reachability required by profile.
- Rationale: The missing guarantee is composition, while framework analysis
  belongs in repository-authored checks.
- Consequences: POSE gains semantic check classes without taking a dependency on
  any UI or dependency-injection framework.

### Decision 2: Reference registered checks from roadmap criteria
- Date: 2026-08-02
- Context: Executable roadmap criteria are needed, but embedding command text
  duplicates the validation matrix and expands the command-execution surface.
- Options considered: free-form prose; raw commands; named structured checks and
  versioned manual reports.
- Decision: Permit only registered delivery/check refs and confined manual
  report refs.
- Rationale: One executor and one result contract preserve security,
  reproducibility and skip semantics.
- Consequences: Repositories must name their cut checks before a roadmap can
  consume them.

## 6. Validation

### Strategy
Use a polyglot fixture that reproduces the exact sibling-issue interaction:
canonical artifact claims and green build/unit checks coexist with unreachable
UI, unused constructors and a roadmap whose narrative outcome is false. The
artifact gate must pass; surface and roadmap gates must fail with the composed
path. A corrected fixture must then pass without POSE understanding its source
framework.

### Deterministic checks
- Test: `go -C pose-mcp test ./internal/pose ./internal/cli ./internal/mcpserver -run 'Surface|Delivery|Reachability|RoadmapCheck|EvidenceClass' -count=1`.
- Full suite: `go -C pose-mcp test ./... -count=1`.
- Catalog: `go -C pose-mcp test ./internal/mcpserver -run 'Catalog|DeliveryIntegrity' -count=1`.
- Scaffold parity: `go -C pose-mcp test ./internal/scaffold -run TestEmbeddedDistMatchesPoseDist -count=1`.
- Structure: `pose check --strict`.
- Spec readiness: `pose lint-spec pose-delivery-surface-assurance --ready-check`.
- Spec lint: `pose lint-spec pose-delivery-surface-assurance --strict`.
- Delivery gate: `pose validate --strict --module pose-mcp --report`.

### Execution log
- 2026-08-02, planning: issue evidence and current spec, trace, validation,
  report, index and roadmap contracts inspected; implementation checks pending.
- 2026-08-02, planning validation: readiness and strict lint passed; the full Go
  suite and embedded-scaffold parity passed after regenerating the mirror.
- 2026-08-03, implementation start: dependency milestone
  `provenance-foundation` is terminal; the shared ADR was selected and the
  risk-based plan below activated before code changes.
- 2026-08-03, implementation: added typed delivery declarations, profiles,
  evidence classes, combined graph paths/findings, strict surface/roadmap gates,
  CLI/MCP parity and the UI-surface governance kit.
- 2026-08-03, review remediation: required Git-observed artifacts for delivery
  edges, made validation freshness independent of generated evidence commits,
  created missing result directories, preserved true deferred-integration
  semantics and reconciled evolving declarations across immutable change sets.
- 2026-08-03, final validation: focused and full Go suites, four-class module
  validation, strict surface assurance, scaffold parity, POSE structural gates,
  assessments and vulnerability analysis passed.

### Risk-based test plan

| Scenario | Command | Expected evidence |
|---|---|---|
| Unit — typed refs, targets and profiles | `go -C pose-mcp test ./internal/pose -run 'Delivery|Surface' -count=1` | Frontmatter/body equality, closed refs and profile invariants pass; drift and unknown classes fail. |
| Integration — false-green composition | `go -C pose-mcp test ./internal/cli -run 'SurfaceCheck|RoadmapCheck' -count=1` | Green artifact/unit evidence cannot satisfy unreachable surfaces or uncomposed capabilities. |
| Contract — structured validation | `go -C pose-mcp test ./internal/cli -run 'EvidenceClass|ValidationResult' -count=1` | Evidence classes persist with provenance digest and stale results fail. |
| Contract — CLI/MCP graph parity | `go -C pose-mcp test ./internal/mcpserver -run 'SurfaceAssurance|DeliveryIntegrity' -count=1` | Same graph, paths, criteria and findings are project-scoped. |
| Security — confined criteria | `go -C pose-mcp test ./internal/pose ./internal/cli -run 'RoadmapCriteria|SurfaceSecurity' -count=1` | Raw commands, traversal and unknown refs/classes are rejected without execution. |
| Required module gate | `pose validate --strict --module pose-mcp --json .pose/results/delivery-validation.json --report` | Unit/build/integration/reachability results are current and reusable by surface closeout. |

### Results summary
- Successes: all thirteen requirements are implemented. The real negative
  fixtures keep artifact/unit evidence green while rejecting unreachable or
  uncomposed delivery; the corrected graph exposes current, digest-bound paths.
- Failures: review found four semantic gaps; each was remediated and covered by
  regression before approval.
- Warnings: legacy orphan and pre-adoption roadmap findings remain observable
  warnings. They do not weaken strict gates for adopted specs or new criteria.

### Requirement trace
- R1 [satisfied] test:TestDeliveryTargetsRequireExactTypedFrontmatterAndBodyRefs.
- R2 [satisfied] test:TestDeliveryTargetsRequireExactTypedFrontmatterAndBodyRefs.
- R3 [satisfied] test:TestSurfaceCheckFailsUnreachableDelivery; report:.pose/reports/2026-08-03-pose-delivery-surface-assurance-review.md.
- R4 [satisfied] test:TestValidateWritesEvidenceClassAndProvenanceDigest.
- R5 [satisfied] test:TestDeliverySurfaceFailsGreenArtifactWithUnreachableSurface; evidence:integration evidence:e2e
- R6 [satisfied] test:TestTraceAcceptsDeliveryAndEvidenceRefs; evidence:unit evidence:integration
- R7 [satisfied] test:TestDeferredIntegrationDoesNotAssertDeliveryOrSatisfyRoadmapCriterion; surface:delivery-integrity-cli evidence:integration
- R8 [satisfied] check:delivery-reachability; surface:delivery-integrity-cli evidence:e2e
- R9 [satisfied] test:TestDeliverySurfaceFailsGreenArtifactWithUnreachableSurface; capability:delivery-integrity-graph evidence:integration
- R10 [satisfied] test:TestRoadmapCriteriaRejectRawCommandsAndRequireRegisteredRefs.
- R11 [satisfied] test:TestRoadmapCheckRejectsIncompleteMemberAndMissingEvidence.
- R12 [satisfied] test:TestSurfaceAssuranceToolReadsProjectScopedGraph; contract:delivery-integrity-mcp evidence:integration
- R13 [satisfied] check:skills-check-strict; report:.pose/reports/2026-08-03-pose-delivery-surface-assurance-review.md.

### Known gaps
- POSE proves registered check execution, freshness and graph composition; the
  semantic strength of a repository-authored reachability check remains a
  review responsibility documented by the new workflow and skill.

## 7. Final Report

### Delivered scope
Implemented stack-neutral delivery targets, profile-driven evidence,
Git-observed composition paths, strict surface and roadmap assurance, and
matching CLI/MCP projections plus a default UI-surface governance kit.

### Files and modules changed
- `pose-mcp/internal/pose`, `internal/cli` and `internal/mcpserver`: model,
  deterministic gates, validation results and project-scoped projections.
- `.pose`, manuals, docs, locales and embedded scaffold: policy, profiles,
  templates, workflow, rule and closeout skill.

### Validation executed
- Commands: focused negative/positive delivery suites; `go test ./...
  -count=1`; catalog and scaffold parity; `pose check --strict`; ready/strict
  lint; `pose validate --strict --module pose-mcp`; `pose surface-check
  --strict`; assessments; `govulncheck ./...`.
- Result: all blocking gates passed with current provenance-bound unit,
  integration and reachability evidence.

### Residual risks
- The engine can guarantee evidence shape and freshness, not the semantic depth
  of repository-authored reachability checks; reference fixtures and human
  review remain necessary.

### Follow-ups
No deferred implementation follow-up. Release consumption and publication are
owned by `pose-release-lifecycle-closure`, the next ordered roadmap spec.
