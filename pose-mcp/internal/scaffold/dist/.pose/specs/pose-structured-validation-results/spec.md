---
slug: pose-structured-validation-results
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-standalone-dogfood
priority: 15
---

# Spec: Structured validation result contract

## 1. Intent

### Goal
emit stable JSON plus interoperable JUnit and SARIF projections from one result model.
### Business value
Unlocks CI annotations, MCP, traceability, analytics and Harness.
### Constraints
- Preserve human-readable logs and deterministic outcomes.
### Non-goals
- Translate arbitrary tool output perfectly or replace native reporters.

## 2. Requirements

### Functional
- R1: Every check result shall include stable ID, command metadata, timing, severity, outcome and skip reason.
- R2: The CLI shall emit versioned JSON and optional JUnit/SARIF projections.
- R3: Partial, tolerated and infrastructure failures shall remain distinguishable.

### Non-functional
- Keep output ordering stable and bound captured output.

### Security
- Redact configured secrets and omit inherited environment values.

### Compatibility
- Text output remains usable while machine formats are additive.

## 3. Technical Plan

### Affected areas
- Validation domain, CLI, reports/history, MCP and CI action.

### API/contract changes
- Define a versioned result schema and output conventions.

### Data/storage changes
- Persist schema version and evidence references.

### Technical risks
- Lossy projections may collapse POSE outcomes unless extensions are documented.

### Primary references
- [JSON Schema](https://json-schema.org/specification)
- [SARIF 2.1.0](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [JSON Schema](https://json-schema.org/specification): validate emitted text + exit code only; no machine-readable per-check identity, timing, skip reason or outcome distinction; context resumed from knowledge:contract-baseline-handoff.

### Implementation
- [x] Model canonical check, run and aggregate schemas: `validationRunResult`/`checkResult` (schema_version 1) with stable IDs (`<module>/<stack>/<name>`), command metadata, configured env (secrets redacted), severity, outcome vocabulary `pass|fail|error|skipped`, deterministic skip reasons, exit code, duration and bounded output tail; run-level `partial` outcome keeps tolerated failures distinguishable (ADR `2026-07-19-versioned-validation-result-contract`). ([reference](https://json-schema.org/specification))
- [x] Implement deterministic JSON and documented JUnit/SARIF mappings: `--json/--junit/--sarif <path>` (confined to project root); JUnit module→testsuite with error/failure/skipped mapping and severity in classname; SARIF 2.1.0 with one rule per check, levels by severity and `pose/*` properties preserving the full POSE outcome — text output unchanged and authoritative. ([reference](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html))
- [x] Add golden cases for pass, fail, partial, skip, timeout and redaction: `validate_results_test.go` pins JSON contract (pass/fail/error/skip + counts + partial), required-failure exit semantics, secret redaction in metadata/output/projections, bounded capture, well-formed JUnit and SARIF envelopes and output-path confinement; per-check timeout execution belongs to `pose-validation-runtime-guardrails` (next milestone) — the contract already carries duration and the `error` outcome it will use. ([reference](https://json-schema.org/specification))

### Validation
- [x] Run `go test ./pose-mcp/internal/cli/... -run 'Validate|Report|SARIF|JUnit'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://json-schema.org/specification))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-versioned-validation-result-contract.md` (Accepted): one canonical versioned JSON model with documented projections over adopting JUnit/SARIF as canonical (both lossy for POSE semantics); infra `error` never masquerades as check failure; every skip carries its deterministic reason; inherited environment never enters results.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Validate|Report|SARIF|JUnit'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-structured-validation-results --ready-check`.

### Requirement trace
- R1 [satisfied] stable ID, command metadata, timing, severity, outcome and skip reason per check; check:test (TestValidateStructuredJSONContract)
- R2 [satisfied] versioned JSON plus JUnit/SARIF projections behind confined output flags; check:test (TestValidateJUnitProjection, TestValidateSARIFProjection, TestValidateOutputPathConfined) report:2026-07-19-standard-validate-native.md
- R3 [satisfied] partial, tolerated and infrastructure failures stay distinguishable (run partial, check error vs fail); check:test (TestValidateStructuredJSONContract, TestValidateRequiredFailureOutcome)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'Validate|Report|SARIF|JUnit' -count=1` — SUCCESS (nine tests including redaction and confinement).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite).
- `pose validate --strict --module pose-mcp --json .pose/reports/validation-latest.json --report` — SUCCESS (first real structured result emitted and retained).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-structured-validation-results --strict` — SUCCESS.

## 7. Final Report

### Delivered scope

Canonical versioned result model with stable identities and distinguishable
outcomes; additive `--json/--junit/--sarif` emission with confined paths;
documented projections preserving POSE semantics via extensions; bounded,
redacted output capture; deterministic skip reasons; behavior tests for every
outcome class; operating-manual documentation and ADR. Text output remains
unchanged for humans.

### Residual risks

- JUnit cannot express tolerated failures natively — documented lossy edge;
  the canonical JSON is the fidelity source.

### Follow-ups

- [covered: pose-validation-runtime-guardrails] Per-check timeout and resource ceilings emitting the `error` outcome with timing.
- [covered: pose-changed-scope-validation] Selection reasons for scope-filtered skips on top of this contract.
- [open] Adopt --sarif in the CI security surface once code-scanning upload is wired for validation results. (owner:@pose-maintainers crit:low review:2026-10-16)
