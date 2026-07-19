---
slug: pose-mcp-project-scope-contract
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-mcp-catalog-conformance
priority: 20
---

# Spec: MCP project-scope contract

## 1. Intent

### Goal
make project selection explicit and consistent across every multi-root MCP tool.
### Business value
Prevents ambiguous reads and policy decisions as POSE moves beyond one repository.
### Constraints
- A default project is convenience only; ambiguity must fail closed.
### Non-goals
- Introduce tenant storage into the local CLI.

## 2. Requirements

### Functional
- R1: Every project-capable tool shall expose the same `project_id` schema and resolution rules.
- R2: Unknown, unauthorized or ambiguous projects shall return distinct structured errors.
- R3: Tool results and audit events shall identify the resolved project without leaking host paths.

### Non-functional
- Keep single-project stdio ergonomics unchanged.

### Security
- Authorize the resolved project and tool together before repository access.

### Compatibility
- Allow an announced deprecation window for legacy implicit selection.

## 3. Technical Plan

### Affected areas
- MCP schemas, root resolver, bootstrap, policy input, audit and docs.

### API/contract changes
- Standardize `project_id`, error codes and resolution precedence.

### Data/storage changes
- No repository migration; update catalog and policy fixtures.

### Technical risks
- Path-derived roots can bypass logical identity if authorization occurs too late.

### Primary references
- [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- [MCP authorization specification](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools): only 11/20 pose_* tools advertised project_id; resolution errors were untyped strings; no compatibility path existed for tightening multi-project ambiguity.

### Implementation
- [x] Specify resolution precedence and structured project errors: precedence unchanged (argument → header → default); `pose.ProjectUnknownError`/`pose.ProjectAmbiguousError` replace untyped errors; `structuredContent.error_code` added to the tool-error path (ADR `2026-07-19-mcp-project-scope-resolution-and-structured-selection-errors`). ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))
- [x] Apply one project schema and authorization hook to every relevant tool: all 20 pose_* tools now declare the identical project_id property (golden regenerated); authorization already evaluates the requested project_id before store access (unchanged, confirmed correct). ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization))
- [x] Test unknown, duplicate, unauthorized, default and discovered roots: `roots_test.go` covers unknown (typed, path-free), ambiguous-no-default, compat-mode implicit default under multi-project, strict-mode rejection, single-project immunity to strict mode, explicit-override precedence and rescan-discovered projects; `project_scope_test.go` covers schema consistency and both structured HTTP error paths; unauthorized reuses the existing policy-deny path (already tested). ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))

### Validation
- [x] Run `go test ./pose-mcp/... -run 'Project|Root|Authorization'` and retain evidence (see §6 and `.pose/reports/`). ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-mcp-project-scope-resolution-and-structured-selection-errors.md` (Accepted): typed errors + universal schema + opt-in strict mode, over leaving errors opaque (already failed) and over immediately failing closed (breaks the compatibility requirement); `StrictSelection` is the announced deprecation window, off by default, provably inert for single-project deployments.

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Project|Root|Authorization'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-mcp-project-scope-contract --ready-check`.

### Requirement trace
- R1 [satisfied] all 20 pose_* tools share the identical project_id schema; check:test (TestProjectIDSchemaConsistencyAcrossCatalog)
- R2 [satisfied] unknown vs ambiguous project selection return distinct structured error_codes; check:test (TestUnknownProjectIDReturnsStructuredError, TestAmbiguousProjectSelectionReturnsStructuredError, TestRoots_UnknownProjectIsTypedAndPathFree, TestRoots_AmbiguousNoDefault, TestRoots_StrictModeRejectsImplicitDefaultUnderMultiProject) report:2026-07-19-standard-validate-native.md
- R3 [satisfied] structured errors and audit metadata carry only the logical project_id, never the filesystem root; check:test (TestRoots_UnknownProjectIsTypedAndPathFree)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/pose -run 'Roots' -count=1` — SUCCESS (nine tests).
- `go -C pose-mcp test ./internal/mcpserver -run 'ProjectID|UnknownProject|AmbiguousProject' -count=1` — SUCCESS.
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite, golden catalog regenerated for the 9 schema additions).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-mcp-project-scope-contract --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Uniform `project_id` schema across all 20 `pose_*` tools; typed
`ProjectUnknownError`/`ProjectAmbiguousError` replacing untyped resolution
errors; structured `error_code` (`project_unknown`/`project_ambiguous`) on
the tool-error path, never leaking the resolved filesystem root;
`POSE_MCP_STRICT_PROJECT_SELECTION` opt-in fail-closed mode for multi-project
deployments, provably inert for single-project ones; operating-manual and
`mcp.md` documentation; ADR.

### Residual risks

- The strict flag is opt-in — the misrouting risk it prevents stays live
  under default configuration until an operator adopts it, per the
  documented deprecation window.

### Follow-ups

- [open] Promote POSE_MCP_STRICT_PROJECT_SELECTION to the default once multi-project deployments have had a full release cycle to adopt it explicitly. (owner:@pose-maintainers crit:medium review:2026-10-23)
