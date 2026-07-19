# ADR: Harne8 control-plane composition — ratified boundaries, evidence reconciliation closes the loop

## Status
Accepted (2026-07-19) — spec `pose-harne8-control-plane-integration`

## Context

This is the final spec of the 7-roadmap portfolio, and the only one whose
subject — Conductor, Harness, GraphForge and Portal — are Harne8 products
that do not exist in this repository. Every prior roadmap already built
most of the actual composition surface as a byproduct of its own spec:
`pose-safe-validate-orchestration` gave POSE a digest-pinned,
approval-gated, idempotent request/submit state machine
(`pose_validate_request/approve/submit/status/cancel`) with a pluggable
`HarnessExecutor`; earlier work gave POSE `Reporter`/`conductor_run_*`
for external run reporting, Execution Identity (ADR-007) for
workload-identity-shaped RunID/Scopes/ExpiresAt, and `PolicyGate`/OPA for
policy bundles and audit. This spec's own Task list asks first to
"ratify component responsibilities and contracts through ADRs" — the
Decision here is largely about naming what already composes, closing the
one genuinely missing link, and proving the Compatibility requirement
(the open core completes local workflows when Harne8 is absent) with a
real, executable test rather than an assertion.

Alternatives considered:

1. **Build a mock/fake Conductor, Harness, GraphForge and Portal in this
   repository** to have something concrete to integration-test against.
   Rejected: fakes for products that don't exist yet would need to guess
   at their real wire contracts, producing tests that pass against an
   invented API and prove nothing about the real one — false confidence
   is worse than an honestly-scoped local contract.
2. **Leave evidence reconciliation implicit** (the Harness reports a
   result some other way, out of this spec's scope). Rejected: R2 is
   explicit — "Harness results shall be identity-bound and reconcile into
   evidence without silent mutation" — and nothing in the existing
   codebase closed that loop; `HarnessExecutor.Submit` hands control to
   the Harness, but nothing brought a result back.
3. **Ratify the five responsibilities explicitly (table + ADR), close the
   one real gap (evidence reconciliation) as a new, identity-bound,
   append-only local contract, and prove offline degradation with an
   executable test** that unsets every Harne8-related env var and runs a
   full local workflow.

## Decision

Option 3.

- **Component responsibilities are a five-row table**
  (`docs-site/docs/architecture.md`, "Mechanism 15"): POSE governs,
  Conductor orchestrates, Harness executes, GraphForge contextualizes,
  Portal presents — the spec's own Constraint, made concrete with which
  existing surface implements each role.
- **`pose reconcile-evidence`** (`internal/cli/harness_evidence.go`) is
  the new piece: `record` requires `run_id` (identity-bound, R2),
  `request_id` (ties back to the orchestrator's request), `execution_id`
  (from `HarnessExecutor.Submit`) and `plan_digest` (ties back to the
  exact approved plan) — refusing to accept a "result" that doesn't
  reference which request and which identity produced it. A second
  record for a `request_id` that already has evidence is rejected
  outright unless `--allow-supersede` is passed, and even then a new
  record is appended (never edits or deletes the prior one), explicitly
  linking back via `supersedes_recorded_at` — "without silent mutation"
  is a rejection at write time, not a policy a reader has to trust.
- **`pose portfolio-projection` (already built by `pose-cross-repo-portfolio`)
  and `pose semantic-suggest` (already built by
  `pose-semantic-governance-assist`) are ratified as the R3 contract**:
  both are already policy-filtered (project authorization, sensitivity)
  and tenant-scoped — Portal/GraphForge consuming them is a documentation
  decision, not new code.
- **`pose_validate_submit`'s existing idempotent resubmit is ratified as
  the R1 durable-state contract**: the in-process orchestrator registry
  is explicitly a local reference implementation (its own doc comment
  already says so); a real multi-replica deployment centralizes run
  state in Conductor via the same `Reporter`/`conductor_run_*` interface
  that already exists. Nothing needed to change here — this spec confirms
  the design already satisfies R1 rather than adding new state machinery.
- **Offline degradation is proven, not asserted**:
  `TestOpenCoreCompletesLocalWorkflowsWithoutHarne8` unsets every
  Harne8-related env var (`CONDUCTOR_URL`, `POSE_MCP_IDENTITY_SECRET`,
  `HARNE8_PROJECTS_DIR`, `OTEL_EXPORTER_OTLP_ENDPOINT`, etc.) and runs a
  full local governed workflow (install → new-spec → check → validate →
  followups → doctor → portfolio-projection) end to end.
- **SLOs, backpressure, retry and disaster recovery for the hosted
  components are explicitly out of this repository's scope** — Harne8's
  own operational responsibility, per the Non-goal ("never move
  repository authority into a required hosted service"). Documenting
  this boundary explicitly (rather than silently omitting it) is itself
  the deliverable for that non-functional requirement, given none of
  those hosted components exist in this repository to have an SLO
  against.

## Consequences

- Positive: every claim in this ADR about "X already composes with Y" is
  backed by an existing, already-tested code path — this spec adds
  exactly one new local contract (evidence reconciliation) instead of a
  large speculative integration surface for products this repository
  cannot verify.
- Positive: the five-responsibility table gives a stable place to check
  "whose job is this" the next time a Harne8-adjacent feature is
  proposed, reducing the boundary-erosion risk the spec's own Technical
  risk names.
- Negative: evidence reconciliation ships as a CLI command
  (`pose reconcile-evidence`), not an MCP tool — a real Harness would
  need to shell out or otherwise invoke the CLI rather than call an MCP
  tool directly. Chosen for consistency with every other locally-testable
  contract this roadmap built (DORA events, semantic feedback, portfolio
  projection) and to avoid growing the MCP catalog's golden-fixture
  surface for a contract with no real caller yet; tracked as a follow-up
  to add an MCP-exposed variant once a real Harness integration exists to
  validate the wire shape against.
- Neutral: this spec closes the entire 7-roadmap portfolio. No further
  roadmap exists in `product-roadmaps.md` beyond `insights-enterprise-scale`
  at the time of writing.
