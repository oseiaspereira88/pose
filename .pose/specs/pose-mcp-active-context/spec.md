---
slug: pose-mcp-active-context
status: in-progress
created_at: 2026-08-06
completed_at:
supersedes:
depends_on: pose-mcp-project-scope-contract, pose-mcp-catalog-conformance
priority: 1
components: pose-mcp
delivers: contract:mcp-active-context
---

# Spec: MCP active connection context

## 1. Intent

### Goal
Let agents distinguish static MCP configuration from the connected server's
live project registry before reading governed repository state.

### Business value
Prevent plausible, schema-valid answers from a stale connection scoped to the
wrong repository and restore a reliable community-feedback submission path.

### Constraints
- Preserve single-project compatibility and existing structured error fields.
- Return logical project identifiers only after authorization.
- Never expose filesystem roots, environment values or secrets.
- Keep connection restart explicit; do not mutate client configuration over MCP.

### Non-goals
- Add project registration/removal commands.
- Make strict project selection the default in this compatibility release.
- Close GitHub issues or publish a release.

## 2. Requirements

### Functional
- R1: `pose_mcp_context` shall identify the live server instance, version,
  transport, start time, registry refresh time and effective selection mode.
- R2: The context tool shall return only logical project IDs authorized for the
  caller and shall probe an optional requested `project_id` without requiring a
  project store to resolve first.
- R3: `project_unknown` shall preserve its existing fields and add authorized
  alternatives plus structured reconnect/explicit-selection remediation.
- R4: `pose doctor --json` shall state that `mcp.config` validates static
  configuration and did not inspect the active connection.
- R5: `report-limitation --submit` and the feature issue template shall use
  labels present in the upstream repository's standard label contract.

### Non-functional
- Keep context output deterministic apart from process identity and timestamps.
- Keep the MCP catalog golden and public docs exactly aligned.

### Security
- Apply the existing policy and Execution Identity to every discovered project.
- Audit allow and deny decisions made during discovery.
- Return no resolved root or secret-shaped configuration content.

### Compatibility
- Add one read-only tool and additive response fields only.
- Preserve legacy implicit default behavior unless strict selection is enabled.

## 3. Technical Plan

### Affected areas
- Root registry snapshots, MCP dispatch/catalog, CLI doctor and feedback intake.
- English and Portuguese operating manuals plus MCP reference documentation.

### Artifacts
- created: .pose/adr/2026-08-06-mcp-active-context-authorized-discovery.md
- created: .pose/changelogs/unreleased/pose-mcp-active-context.md
- created: .pose/reports/2026-08-06-standard-test-plan-baseline-pose-mcp-active-context.md
- created: .pose/reports/history/standard-test-plan-baseline-pose-mcp-active-context.jsonl
- created: .pose/specs/pose-mcp-active-context/spec.md
- created: pose-mcp/internal/cli/report_limitation_test.go
- created: pose-mcp/internal/mcpserver/mcp_context_test.go
- created: pose-mcp/internal/scaffold/dist/.pose/adr/2026-08-06-mcp-active-context-authorized-discovery.md
- created: pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-mcp-active-context.md
- created: pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260804T210330Z-31f2f3fc.md
- created: pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260804T210330Z-ac29cb2f.md
- created: pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260804T210418Z-0a42db04.md
- created: pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260804T210418Z-e430531a.md
- created: pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260804T210456Z-8dbdfeb2.md
- created: pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260804T210456Z-c87c8aec.md
- created: pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260804T210519Z-8ada974e.md
- created: pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260804T210519Z-9673ff6e.md
- created: pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260804T210526Z-a71ddb95.md
- created: pose-mcp/internal/scaffold/dist/.pose/specs/pose-mcp-active-context/spec.md
- modified: .github/ISSUE_TEMPLATE/feature_suggestion.yml
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/results/delivery-validation.json
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: POSE.md
- modified: docs-site/docs/capability-assessment.md
- modified: docs-site/docs/index.md
- modified: docs-site/docs/mcp.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_remediation_test.go
- modified: pose-mcp/internal/cli/report_limitation.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/project_scope_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/pose/roots.go
- modified: pose-mcp/internal/pose/roots_test.go
- modified: pose-mcp/internal/scaffold/dist/.pose/assessments/README.md
- modified: pose-mcp/internal/scaffold/dist/.pose/assessments/consolidated.md
- modified: pose-mcp/internal/scaffold/dist/.pose/assessments/integrations.md
- modified: pose-mcp/internal/scaffold/dist/.pose/assessments/pose-mcp.md
- modified: pose-mcp/internal/scaffold/dist/.pose/assessments/technical-debt.md
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/releases.json
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
- modified: pose-mcp/internal/scaffold/dist/.pose/results/delivery-validation.json
- modified: pose-mcp/internal/scaffold/dist/.pose/specs/pose-doc-command-reference/spec.md
- modified: pose-mcp/internal/scaffold/dist/.pose/specs/pose-locales-alignment/spec.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets
- contract:mcp-active-context module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- Add `pose_mcp_context` to the release-gated tool catalog.
- Add optional remediation fields to `project_unknown` results.
- Add static diagnostic scope fields to the doctor JSON finding.

### Data/storage changes
- None. Runtime metadata remains in memory for one server process.

### Technical risks
- Project discovery could leak tenant membership if IDs are not policy-filtered.
- Context handling could accidentally inherit store resolution and fail at the
  exact point it is intended to diagnose.
- Retrying issue submission could create duplicates; use only labels guaranteed
  by the repository contract instead of retrying a possibly successful request.

## 4. Tasks

### Planning
- [x] Read project state, feature workflow, cumulative rules and related ADRs.
- [x] Run `pose assess discover --component pose-mcp`.
- [x] Record the public-contract decision and risk-based test plan.

### Implementation
- [x] Add a path-free root-registry context snapshot.
- [x] Add authorized live-context dispatch and unknown-project remediation.
- [x] Update the catalog golden and public documentation.
- [x] Mark doctor MCP evidence as static-only.
- [x] Align feedback labels with the upstream repository.

### Validation
- [x] Run focused unit, authorization, contract and CLI tests.
- [x] Run full Go tests, race coverage and vet for `pose-mcp`.
- [x] Run MCP integration and technical-debt assessments.
- [x] Run strict POSE validation and closeout gates.

## 5. Decisions

- ADR `.pose/adr/2026-08-06-mcp-active-context-authorized-discovery.md`
  (Accepted): each process reports only facts it witnesses; the MCP server
  exposes policy-filtered live context while local doctor remains explicitly
  static. Rejected an apparent end-to-end local doctor and unfiltered registry
  enumeration.

## 6. Validation

### Strategy
Exercise root snapshot units, HTTP MCP contract behavior, authorization denial,
unknown and ambiguous selection, doctor machine output, feedback label mapping,
catalog/docs conformance and the full Go module. No external dependency is
required; OPA-unavailable behavior remains covered by the existing policy suite.

### Deterministic checks

| Class | Scenario | Command | Expected evidence |
|---|---|---|---|
| Required unit | Snapshot ordering, selection modes, no paths | `go -C pose-mcp test ./internal/pose -run 'Roots.*Context|Roots.*Projects' -count=1` | Stable IDs/mode and no root in serialized context |
| Required integration | Live context and requested-project probe | `go -C pose-mcp test ./internal/mcpserver -run 'MCPContext|UnknownProject|ProjectIDSchema|Catalog' -count=1` | HTTP tool result, golden and docs PASS |
| Required negative/security | Anonymous/unauthorized context denied; IDs filtered | `go -C pose-mcp test ./internal/mcpserver -run 'MCPContext.*Policy|MCPContext.*Identity' -count=1` | Policy denial and no unauthorized ID/path |
| Required CLI | Static doctor scope and feedback labels | `go -C pose-mcp test ./internal/cli -run 'Doctor.*MCP|ReportLimitation' -count=1` | `connection_checked=false`; standard labels |
| Required module | Full suite | `go -C pose-mcp test ./... -count=1` | All packages PASS |
| Required static | Go vet | `go -C pose-mcp vet ./...` | No findings |
| Required contract | POSE validation | `pose validate --strict --module pose-mcp --report --report-task test-plan-baseline-pose-mcp-active-context` | Strict report PASS |
| Required structure | Repository contract | `pose check --strict` | PASS |

Invalid JSON and missing optional fields retain generic catalog/dispatch coverage.
Timeout is not applicable because the context tool performs no network or
filesystem I/O beyond the existing policy gate. Policy unavailability remains
fail-closed through `PolicyGate.Evaluate` and is covered by its existing tests.

### Execution log
- Date: 2026-08-06
- Environment: linux/amd64, Go 1.26.5.
- Notes: Focused and full suites passed. The first structured delivery result
  correctly remained bound to the pre-change Git HEAD; immutable attribution
  and closeout are pending the implementation commit.
- Date: 2026-08-06 (post-implementation review pass)
- Notes: Corrected the changelog fragment frontmatter (`kind` -> `category`),
  which was failing `pose check --strict`. Aligned the engine-limitation issue
  template with the same standard label as the CLI and extended the label test
  to cover both intake templates. Regenerated indexes and assessments without
  `--component`, restoring the `mcp-enforce` component that a scoped
  `pose assess discover` had dropped from the consolidated view. Resynced the
  embedded scaffold via `go generate ./internal/scaffold` after the first
  strict run failed on dist drift; the repeat run passed and the structured
  delivery result is now bound to the implementation commit.
- Date: 2026-08-06 (code review hardening)
- Notes: Bounded policy-backed discovery to 64 probes per call with explicit
  truncation reporting, since `project_unknown` remediation is an implicit
  error path a client can trigger repeatedly. Added selection remediation for
  connections with no default root, assembled the context response once
  instead of mirroring the remediation slice, and replaced the local helpers
  with `slices.Contains`/`slices.Sort`. Documented the no-OPA discovery
  posture in the ADR and public MCP reference.

### Results summary
- Successes: focused units/integration/security, full `go test ./...`, race
  tests, `go vet`, strict POSE validation, govulncheck and gitleaks passed.
- Failures: initial structured validation detected scaffold drift introduced by
  the mandatory assessment; regeneration removed the drift and the repeat passed.
- Warnings: `pose assess integrate` reports the existing repository-wide
  unobserved-consumer class, including the newly added tool; no broken consumer
  was found. Govulncheck reported one imported and one required-module advisory
  that are not reachable from this code.

### Requirement trace
- R1 [satisfied] contract:mcp-active-context evidence:integration check:delivery-integration test:TestMCPContextReportsActivePathFreeConnection test:TestMCPContextStdioTransport
- R2 [satisfied] contract:mcp-active-context evidence:integration check:delivery-integration test:TestMCPContextIdentityFiltersProjectsAndDeniesAnonymous test:TestMCPContextProbesUnknownProjectWithoutStoreResolution
- R3 [satisfied] contract:mcp-active-context evidence:integration check:delivery-integration test:TestUnknownProjectErrorOffersAuthorizedAlternativesWithoutPaths
- R4 [satisfied] contract:mcp-active-context evidence:integration check:delivery-integration test:TestDoctorMCPFindingIsExplicitlyStatic
- R5 [satisfied] contract:mcp-active-context evidence:integration check:delivery-integration test:TestReportLimitationUsesStandardUpstreamLabels

### Known gaps
- A client-owned stale stdio process cannot be recreated inside the unit suite;
  the contract exposes the instance/probe evidence needed to detect it in use.

## 7. Final Report

### Delivered scope
Implemented a policy-filtered active-context tool, path-free registry snapshot,
structured unknown-project remediation, explicit static-only doctor evidence,
standard community feedback labels, golden catalog evolution and operator docs.

### Files and modules changed
- `pose-mcp`, public MCP docs and POSE scaffold manuals.

### Validation executed
- Command: `pose validate --strict --module pose-mcp --report --report-task test-plan-baseline-pose-mcp-active-context`.
- Result: SUCCESS; report `2026-08-06-standard-test-plan-baseline-pose-mcp-active-context.md`.

### Residual risks
- Real stale-client recovery depends on the host MCP client honoring explicit
  restart/reconnect; the server now makes the old process identity observable.

### Follow-ups
- [duplicate: pose-mcp-project-scope-contract] Promote strict multi-project selection only through a separately announced compatibility change; the scope-contract spec already tracks promoting `POSE_MCP_STRICT_PROJECT_SELECTION` to the default after a full adoption cycle.
- [open] `pose assess tech-debt` treats a marker as covered when any active spec merely names its component (`documentCoversDebt` in `pose-mcp/internal/pose/techdebt.go`), which silently flipped DEBT-001 (`scaffold.go:23` panic) to `covered_by_spec` with no real relationship and dropped its recommended follow-up; require a file-level or explicit debt-id reference before claiming coverage, because a component name is not evidence. (owner:@pose-maintainers crit:medium review:2026-09-06)
- [open] `pose assess discover --component <slug>` rewrites the repository-wide `.pose/assessments/README.md` and `consolidated.md` from the scoped result, dropping every component it did not scan and orphaning that component's own report and state file; scoped discovery must merge into the consolidated view instead of replacing it. (owner:@pose-maintainers crit:medium review:2026-09-06)
- [open] Integration gap IDs (`GAP-0NN`) are assigned positionally, so adding one contract renumbers every later gap and invalidates external references; derive stable IDs from the contract identity instead. (owner:@pose-maintainers crit:low review:2026-11-06)
