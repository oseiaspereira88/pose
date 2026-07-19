# ADR: Versioned validation result contract

## Status
Accepted (2026-07-19) — spec `pose-structured-validation-results`

## Context

`pose validate` emitted human text and an exit code; the only machine trace
was the report outcome. CI annotations, MCP consumers, requirement traces,
analytics and the future Harness all need structured results — and the next
milestone (changed-scope selection, runtime guardrails) is only trustworthy
if skipped and partial outcomes are explicit, not implied. Comparable
ecosystems standardize on JUnit XML for CI test surfaces and
[SARIF 2.1.0](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html)
for code-scanning annotation.

Alternatives considered:

1. **Emit only JUnit/SARIF** — both are lossy for POSE semantics (JUnit has
   no "tolerated" level; SARIF has no first-class skip reason); adopting a
   foreign model as canonical would collapse outcomes the gates depend on.
2. **One canonical versioned JSON model with documented projections** —
   POSE owns its semantics; interoperability formats are derived views.

## Decision

Option 2:

- **Canonical model** (`schema_version: 1`): per check — stable ID
  (`<module>/<stack>/<name>`), module, stack, program/args, configured env
  (secret values redacted), severity, outcome, skip reason, exit code,
  duration and bounded output tail; per run — mode, filters, aggregate
  outcome and counts. Ordering is deterministic (modules sorted, checks in
  declared order).
- **Outcome vocabulary (R3):** `pass | fail | error | skipped` per check —
  `error` means the tool could not run (infrastructure) and never
  masquerades as a check failure; `skipped` always carries its
  deterministic selection reason. Run outcome adds `partial` for
  tolerated-only failures, so text `SUCCESS` with warnings stays
  distinguishable from a clean pass.
- **Emission (R2):** `--json/--junit/--sarif <path>` (paths confined to the
  project root); text output is unchanged and remains authoritative for
  humans — machine formats are additive.
- **Projections:** JUnit maps module→testsuite, check→testcase,
  error→`<error>`, fail→`<failure>`, skip→`<skipped>` with reason; POSE
  severity survives in the classname suffix. SARIF 2.1.0 emits one rule per
  check and results only for fail/error (level by severity), with
  `pose/outcome`, `pose/severity`, `pose/mode` in properties — the
  documented extension that keeps projections from being silently lossy.
- **Security:** captured output is a bounded tail (4 KiB) with configured
  secret-keyed env values (`token|secret|password|key|credential|private`)
  redacted from metadata and output; the inherited process environment
  never enters the result.

## Consequences

- Positive: CI annotation, MCP, traceability (`report:`/`check:` refs),
  analytics and Harness consume one stable contract; golden behavior is
  pinned by tests for pass, fail, error, skip, redaction and confinement.
- Positive: the changed-scope milestone can now record skipped checks with
  reasons instead of silently narrowing coverage.
- Trade-off: dual emission (text + files) costs a bounded in-memory capture
  per check; 4 KiB tails keep that negligible.
- Residual: JUnit's tolerated-failure gap remains — documented, and the
  canonical JSON is always available where fidelity matters.
