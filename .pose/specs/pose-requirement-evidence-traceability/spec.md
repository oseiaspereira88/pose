---
slug: pose-requirement-evidence-traceability
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-standalone-dogfood
priority: 10
---

# Spec: Requirement-to-evidence traceability

## 1. Intent

### Goal
link stable requirement IDs to checks, results, commits and approval evidence.
### Business value
Makes the closeout gate explain why each promised behavior was accepted.
### Constraints
- Keep links explicit and reviewable; never infer compliance from file proximity.
### Non-goals
- Replace test frameworks or require one issue tracker.

## 2. Requirements

### Functional
- R1: Each active requirement shall map to declared verification cases with stable IDs.
- R2: Closeout shall identify satisfied, withdrawn or explicitly waived requirements.
- R3: Reports and MCP shall expose bidirectional requirement-to-result traversal.

### Non-functional
- Keep the trace schema diff-friendly and valid offline.

### Security
- Minimize actor identity and avoid confidential test output.

### Compatibility
- Existing specs remain readable through an additive migration.

## 3. Technical Plan

### Affected areas
- Spec contract, linting, reports/history, indexes and MCP.

### API/contract changes
- Add stable verification-link fields and closeout rules.

### Data/storage changes
- Add append-only trace records or versioned spec fields.

### Technical risks
- Mechanical link coverage can be mistaken for evidence quality.

### Primary references
- [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/)
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/): R-IDs existed since the DoR gate but nothing linked them to checks, reports or commits; 10 pre-contract done specs identified as the migration population.

### Implementation
- [x] Design stable requirement, verification-case and evidence identifiers: in-spec `### Requirement trace` grammar — `R<N> [satisfied|waived: reason|withdrawn]` with structured refs `check:`, `test:`, `report:`, `commit:` (ADR `2026-07-19-requirement-trace-contract`); templates (en + pt-BR) scaffold the section. ([reference](https://opentelemetry.io/docs/concepts/signals/))
- [x] Extend lint, report and index paths with bidirectional validation: `internal/pose/trace.go` parser (requirements ↔ entries ↔ `by_evidence` reverse index); `lint-spec` fails on malformed/orphaned entries always and on missing coverage at done-with-section; `spec.trace.*` metrics; new MCP tool `pose_requirement_trace` (golden catalog regenerated, docs updated). ([reference](https://slsa.dev/spec/v1.2/))
- [x] Add fixtures for satisfied, waived, stale and orphaned evidence: `internal/pose/trace_test.go` (bidirectional, malformed, sectionless) and `internal/cli/trace_lint_test.go` (complete closeout, missing requirement, orphan, legacy warning). ([reference](https://opentelemetry.io/docs/concepts/signals/))

### Validation
- [x] Run `go test ./pose-mcp/... -run 'Requirement|Evidence|Trace'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://opentelemetry.io/docs/concepts/signals/))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://slsa.dev/spec/v1.2/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-requirement-trace-contract.md` (Accepted): in-spec trace subsection over a separate trace file and over proximity inference; staged migration (legacy done specs warn, new specs fully enforced via template); additive MCP catalog change reviewed through the golden diff.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Requirement|Evidence|Trace'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-requirement-evidence-traceability --ready-check`.

### Requirement trace
- R1 [satisfied] trace grammar parsed with stable IDs and structured refs; check:test (TestParseRequirementTraceBidirectional, TestTraceCloseoutComplete)
- R2 [satisfied] closeout enforces satisfied/waived/withdrawn coverage with reasons; check:test (TestTraceCloseoutMissingRequirement, TestParseRequirementTraceMalformedEntries)
- R3 [satisfied] bidirectional traversal via lint metrics and MCP; check:test (TestRequirementTraceTool) report:2026-07-19-standard-validate-native.md

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/pose ./internal/cli ./internal/mcpserver -run 'Trace|RequirementTrace|Catalog' -count=1` — SUCCESS.
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite, golden catalog regenerated for the additive tool).
- `pose check --strict` — SUCCESS; `pose lint-spec --all --strict` — SUCCESS with exactly the 10 expected legacy warnings.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

In-spec requirement trace contract (grammar, parser, lint gates, metrics),
bidirectional MCP projection (`pose_requirement_trace`), template scaffolding
in both locales, fixtures for every disposition and failure mode, staged
legacy migration and ADR. This spec dogfoods its own contract in §6.

### Residual risks

- Mechanical link coverage can be mistaken for evidence quality — review
  owns quality; the gate owns existence and consistency.

### Follow-ups

- [open] Flip the legacy done-without-trace warning to an error once the 10 pre-contract specs gain trace sections or are archived. (owner:@pose-maintainers crit:medium review:2026-10-23)
