# MCP server

**Doc type:** Reference &nbsp;·&nbsp; **Applies to:** POSE ≥ 0.9.0

`pose serve-mcp` exposes a read-heavy governance view of a POSE instance to
MCP-capable agents. Transports: stdio
(`--stdio`, ideal for agent runtimes) and Streamable HTTP (`POSE_MCP_ADDR`,
default `:8790`).

## Configuration

| Env var | Purpose |
|---|---|
| `POSE_PROJECT_ROOT` | Repository root of the default project (must contain `.pose/`) |
| `POSE_DEFAULT_PROJECT_ID` | Default project id (derived from the root dir name if empty) |
| `POSE_PROJECT_ROOTS` | JSON map of additional `project_id → root` entries |
| `POSE_MCP_TOKEN` | Bearer token for HTTP transport (empty = dev, auth off) |
| `POSE_MCP_OPA_URL` / `POSE_MCP_OPA_PATH` | OPA policy endpoint (empty = allow-all dev mode; failures deny) |
| `POSE_MCP_REQUIRE_PRINCIPAL` | Deny anonymous `tools/call` even without OPA |
| `POSE_MCP_IDENTITY_SECRET` | Verifies run-bound execution identities |
| `POSE_MCP_STRICT_PROJECT_SELECTION` | Non-empty = fail closed on empty `project_id` when more than one project is registered (see below) |

The installer seeds `.mcp.json` when absent. It invokes the native binary
directly and records the installed project's root and project id in the server
environment; no wrapper or second executable is generated.

## Observability

`pose serve-mcp` can emit OpenTelemetry traces, metrics and correlated logs
for every `tools/call` (spec `pose-otel-observability`). Off by default —
POSE stays fully offline unless **both** of the following are set:

| Env var | Purpose |
|---|---|
| `POSE_OTEL_ENABLED` | Must be `1`/`true` — POSE's own opt-in gate |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP/HTTP collector endpoint — no default is baked into the binary |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` to skip TLS (local/dev collectors) |
| `OTEL_EXPORTER_OTLP_HEADERS` | `key1=value1,key2=value2` — e.g. a collector auth header |
| `OTEL_TRACES_SAMPLER_ARG` | Trace sample ratio, `0.0`–`1.0` (default `1.0`) |
| `OTEL_METRIC_EXPORT_INTERVAL` | Metric export interval in milliseconds (default `15000`) |

Every span and metric carries only the tool name and its catalog risk
class (`read`/`gate`/`external-side-effect`) — never an argument, path,
repo name or user id. Metrics: `pose.mcp.tool.call.duration` (histogram),
`pose.mcp.policy.denial.count` (counter), `pose.mcp.tool.call.inflight`
(current concurrency). Logs are structured JSON on stderr, correlated to
the active span's `trace_id`/`span_id`, with paths and secret-shaped
content redacted before being written. A misconfigured or unreachable
collector never blocks server startup or a tool call — export failures
are logged and swallowed, bounded by the shutdown timeout.

## Tools

| Tool | Returns |
|---|---|
| `pose_list_specs` / `pose_get_spec` | Spec inventory (lifecycle frontmatter) / full spec |
| `pose_requirement_trace` | Bidirectional requirement↔evidence trace of one spec (dispositions, refs, missing/orphans) |
| `pose_capability_state` | Current capability assessment: mechanisms with scores/targets, typed evidence, gaps, evidence-resolution issues and age |
| `pose_capability_history` | Append-only assessment snapshots (score vectors), supersede-aware and paginated |
| `pose_spec_amendments` | Append-only amendment history of one spec plus unacknowledged requirement changes |
| `pose_spec_readiness` | Is a spec eligible? Resolves `depends_on` refs (specs, milestones, roadmaps) |
| `pose_list_roadmaps` / `pose_get_roadmap` | Governed roadmaps and their milestone DAGs |
| `pose_get_changelog` | User-facing changelog fragments |
| `pose_get_followups` | Aggregated follow-up backlog |
| `pose_check` / `pose_lint_spec` / `pose_skills_check` | Run the deterministic gates |
| `pose_suggest` | Canonical trail per task type |
| `pose_get_workflow` / `pose_get_rules` / `pose_get_skill` | Operating procedure content |
| `pose_list_knowledge` / `pose_get_knowledge` | Operational memory |
| `pose_list_reports` / `pose_get_report` | Validation evidence |
| `pose_insights` | Deterministic outcome aggregates by workflow, task or context |
| `pose_extension_list` | List installed extensions (id, version, kind, digest, signature status) |
| `pose_validate_request` | Resolve an immutable, digest-pinned validation plan (no execution) |
| `pose_validate_approve` | Approve/reject a plan, bound to its digest, requiring an Execution Identity |
| `pose_validate_submit` | Hand an approved plan to the configured Harness executor |
| `pose_validate_status` | Read a validation request's current state and plan |
| `pose_validate_cancel` | Cancel a non-terminal validation request |

Every tool above is classified `read` (repository-owned governance state only)
except `pose_check` and `pose_lint_spec`, classified `gate` (deterministic
local gates — no writes, no network). The advertised catalog is a release-gated
public contract frozen by a golden fixture
(`pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json`); removals or
incompatible schema changes require an ADR and a release note.

## Optional tools

Three `external-side-effect` tools report externally observed runs to a
Harne8 Conductor control plane. They are always advertised in `tools/list`,
but calls only succeed when the reporter is activated via `CONDUCTOR_URL`,
`CONDUCTOR_RUN_TOKEN` and `CONDUCTOR_PROJECT_ID`; without activation they
return an `isError` result with configuration guidance.

| Tool | Effect |
|---|---|
| `conductor_run_open` | Open an observed external run (returns `run_id`, `task_id`) |
| `conductor_run_event` | Append a progress/checkpoint event to an open run |
| `conductor_run_close` | Close a run with its outcome and cost |

## Project scope contract

Every `pose_*` tool advertises the same `project_id` schema and resolution
rule — a default is convenience only, never a silent guess. `tools/call`
resolves the project from, in order: the `project_id` argument, then the
`X-MCP-Project`/`X-Project-Id` header, then the configured default root.
Resolution failures are distinct, structured errors (`isError: true`,
`structuredContent.error_code`) that never include the resolved filesystem
root — only the caller-supplied logical identifier:

| `error_code` | Meaning | `structuredContent` |
|---|---|---|
| `project_unknown` | `project_id` does not resolve to any registered root, even after a rescan | `project_id` |
| `project_ambiguous` | `project_id` was omitted and the server cannot pick one unambiguously | `reason`: `no-default` or `multi-project-implicit` |

A third case — the resolved project exists but policy denies it — surfaces
through the existing JSON-RPC error `-32004` with `decision.Metadata()`
(`policy denied`), not through `structuredContent`.

**Compatibility / deprecation window:** with `POSE_MCP_STRICT_PROJECT_SELECTION`
unset (default), an empty `project_id` always resolves to the configured
default root, even once a deployment has onboarded more than one project —
existing single-project stdio ergonomics are exactly unchanged. Setting the
variable makes that same omission fail closed with
`project_ambiguous`/`multi-project-implicit` whenever more than one project
is registered; a genuinely single-project deployment is never affected by
the flag. Multi-project operators should plan to adopt it as project count
grows; it is expected to become the default in a future release.

## Pagination and catalog stability

`pose_list_specs`, `pose_list_roadmaps`, `pose_list_knowledge` and
`pose_list_reports` accept optional `cursor`/`limit` arguments and return an
additive `next_cursor` field (empty when exhausted). Cursors are opaque,
versioned position tokens over each list's fixed deterministic order (spec
slug, roadmap slug, knowledge slug, or `generated_at` descending) — never
parse or construct one client-side; a malformed or wrong-version cursor is a
tool error, not silently coerced to page 1. Omitting both arguments returns
every item in a single page — the exact response shape from before
pagination existed.

The tool catalog itself never changes within one server process: `tools/list`
is a pure function of the binary and is byte-identical across calls in the
same session, so `capabilities.tools.listChanged: false` is verified, not
aspirational. A catalog change only happens across a release (a new
`pose-mcp` binary, a new `serverInfo.version`) — clients should reconnect
(re-`initialize`) after observing that version change; POSE does not (and,
given a static per-process catalog, has no reason to) emit
`notifications/tools/list_changed` events.

**Resources and prompts are deliberately not implemented.** Every governed
read POSE exposes — specs, roadmaps, knowledge, reports, workflows, rules,
skills — is already served through typed, schema-validated, project-scoped,
policy-gated tools. A generic MCP `resources` primitive would let a client
address arbitrary repository content by URI, which is exactly "expose
repository files wholesale" — explicitly out of scope. A generic `prompts`
primitive risks encoding procedure outside the reviewable
`.pose/workflows/*.md` files it is meant to expose, which is "turn prompts
into hidden policy" — also out of scope. `capabilities` therefore advertises
only `tools`; a tools-only client sees no unimplemented primitive to
misconfigure.

## Safe validation orchestration

`pose validate` stays a local, unrestricted CLI command. Agents that need to
*request* validation through MCP go through a separate, deliberately narrow
state machine instead of an unsafe direct-execution passthrough:

```
pose_validate_request → pending_approval
       │ (plan digest pins matrix + git HEAD + filters)
       ▼
pose_validate_approve → approved | rejected
       │ (requires a bound Execution Identity; digest must match exactly)
       ▼
pose_validate_submit  → submitted (idempotent; requires a Harness executor)
```

- **The plan is immutable and digest-pinned.** `pose_validate_request` hashes
  the exact validation matrix bytes plus git HEAD and the requested filters;
  `pose_validate_approve` must echo that exact `plan_digest`, or the call is
  rejected as plan substitution — the request cannot be silently widened
  between resolution and approval.
- **Approval is never anonymous.** `pose_validate_approve` requires a valid,
  unexpired `X-MCP-Execution-Identity` token (ADR-007) regardless of the
  server's default policy mode — a deployment running OPA in dev/allow-all
  still cannot approve orchestrated validation anonymously. Because the
  identity header only exists on the HTTP transport, orchestration approval
  is an HTTP-transport operation; stdio deployments inherit trust from the
  spawning client for local `pose validate`, which is unaffected by any of
  this.
- **pose-mcp never executes the plan.** `pose_validate_submit` hands the
  approved, digest-pinned plan to a pluggable `HarnessExecutor` — wired with
  `Server.WithHarnessExecutor`, the same optional-dependency pattern as
  `WithReporter` for Conductor. Without one configured, submission returns a
  clear configuration error; nothing runs. Submission is idempotent:
  resubmitting an already-submitted request returns the same `execution_id`
  without invoking the executor again.
- **State is local and in-process**, not "centrally persisted" — a
  production deployment centralizes run state in Conductor and plugs
  execution in via `HarnessExecutor`, the same relationship the Conductor
  run reporter tools already have to the Conductor board. `pose_validate_status` reads a
  request's current state; `pose_validate_cancel` marks a non-terminal
  request cancelled (a submitted request's cancellation reaching a running
  Harness execution is the executor's own responsibility, not pose-mcp's).

## Security posture

- Default deny on OPA errors; policy decisions are audited
  (`policy.decided` / `policy.violation` structured logs).
- Shared-domain tools run in-process; CLI-backed tools invoke the current
  native executable. Every argument is validated and shell text is never
  evaluated.
- Multi-replica deployments need the Redis cursor store (enterprise hardening
  track); single-node dev needs nothing beyond the binary.
