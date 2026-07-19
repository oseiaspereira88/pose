# ADR: MCP-Conductor-Harness trust boundary for safe validation orchestration

## Status
Accepted (2026-07-19) — spec `pose-safe-validate-orchestration`

## Context

There is real, recurring demand for a `pose_validate` MCP tool an agent
could call directly — but an MCP tool that spawns arbitrary local
subprocess execution on `tools/call` is exactly the unsafe pattern the
spec's non-goal forbids ("allow arbitrary commands or silent execution on
tools/call"). The roadmap's own risk control is explicit: "separate
read-only MCP governance from execution orchestration." A real sandboxed
executor (the Harness) is a Harne8-platform component that does not exist
as code in this open-source repository — so the spec cannot be "implement a
sandbox," it has to be "define the contract a sandbox plugs into."

Alternatives considered:

1. **A `pose_validate` tool that runs `pose validate` in-process on
   `tools/call`** — exactly the forbidden pattern: an MCP call becomes
   unreviewed local code execution, with no approval step and no isolation.
2. **Defer the whole spec until a real Harness exists** — leaves the actual
   security-relevant surface (what a request/approval/result contract must
   guarantee) undesigned and untested until it's too late to review cheaply.
3. **Own the plan/approval/result state machine locally; execution is a
   pluggable interface a real Harness implements.**

## Decision

Option 3 — the trust boundary is explicit and three-part:

- **MCP (pose-mcp, this repo) requests work.** `pose_validate_request`
  resolves an immutable plan — the exact validation matrix bytes (SHA-256),
  git HEAD and requested filters, digest-pinned — and stores it
  `pending_approval`. No check ever runs from this call.
- **POSE owns plan/approval/result semantics.** `pose_validate_approve`
  requires the caller to echo the exact plan digest (substitution — the
  plan drifting between resolution and approval — is rejected, not
  silently re-approved) and requires a bound, HMAC-verified, unexpired
  Execution Identity (ADR-007) regardless of the server's default OPA
  policy mode: a dev/allow-all deployment still cannot approve
  orchestrated validation anonymously. A decided request cannot be
  re-decided (no replay).
- **The Harness executes.** `pose_validate_submit` hands the approved,
  digest-pinned `ApprovedValidationRequest` to a pluggable
  `HarnessExecutor` interface, wired via `Server.WithHarnessExecutor` —
  the same optional-dependency shape `WithReporter`/Conductor already
  uses. Without one configured, submission is a clear configuration
  error, never a silent no-op success. Submission is idempotent: the
  registry re-checks state under lock after the (potentially slow,
  external) `Submit` call returns, so a resubmit racing a first submit
  can never double-invoke the executor.
- **State is local and in-process in this repository**, explicitly not
  "centrally persisted" — a production deployment centralizes run state in
  Conductor and plugs execution in via `HarnessExecutor`, mirroring how
  `conductor_run_*` already delegates state to Conductor. The in-process
  registry exists so the state machine itself — the actual security
  surface — is real, local and directly testable without requiring
  Conductor or a Harness to exist.
- **Identity binding is HTTP-only** (the `X-MCP-Execution-Identity` header
  has no stdio equivalent), so orchestration approval is necessarily an
  HTTP-transport operation; local `pose validate` and stdio deployments are
  completely unaffected — a separate, unchanged command path.

## Consequences

- Positive: the demand for `pose_validate` is resolved without ever letting
  an MCP `tools/call` execute anything — the non-goal is upheld by
  construction, not by convention.
- Positive: substitution, replay, un-approved submission and unconfigured
  execution are all tested failure modes with typed, distinguishable
  errors, not just documented intentions.
- Trade-off: without a real `HarnessExecutor` wired in, the orchestration
  tools are fully exercisable up through approval but cannot actually run
  anything — expected and correct for this repository; a Harness
  implementation is out of scope here by design.
- Residual: cancellation of an already-submitted request is a local marker
  only; propagating it to a running Harness execution is the executor's
  responsibility, not pose-mcp's — documented, not silently assumed.
