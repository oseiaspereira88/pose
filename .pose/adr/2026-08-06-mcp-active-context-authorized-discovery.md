# ADR: MCP active context uses authorized logical discovery

## Status
Accepted (2026-08-06) — spec `pose-mcp-active-context`

## Context

The project-scope contract already gives every project-capable MCP tool a
uniform `project_id`, typed selection errors and an opt-in strict mode. It does
not let a client distinguish the configuration currently stored in `.mcp.json`
from the registry loaded by the connected server process. A stale stdio
connection can therefore answer successfully from an old default project while
`pose doctor` reports the new repository's static configuration as healthy.

The diagnostic needs to identify the live server instance, its selection mode
and the logical project IDs the caller may use. It must remain useful when a
requested project is unknown, while preserving project authorization and never
exposing filesystem roots.

Alternatives considered:

1. Extend `pose doctor` to claim an end-to-end connection check — rejected
   because a local CLI process cannot inspect a client-owned stdio connection.
2. Return every registered project and root from a context tool — rejected
   because registry membership and host paths cross the authorization boundary.
3. Add a read-only context tool with per-project authorization filtering and
   keep the local doctor explicitly static — selected because each process
   reports only facts it can actually witness.

## Decision

Add the release-gated `pose_mcp_context` MCP tool. Handle it before project
store resolution so it remains callable when the default is absent, ambiguous
or stale. Return a versioned response containing server identity, process
instance identifier, start time, transport, registry refresh time, strict
selection state, effective selection mode and authorized logical project IDs.

Filter every returned project ID through the same `PolicyGate` and verified
Execution Identity used by ordinary tool calls. Record each discovery decision
through the existing auditor. Never return project roots, configuration file
content, environment values or unauthorized logical IDs. Return an optional
requested-project probe as `resolved` or `unknown`, with structured restart and
explicit-selection remediation for the latter.

Keep `pose doctor` as a local static diagnostic. Mark the `mcp.config` finding
with `diagnostic_scope: static-configuration` and
`connection_checked: false`, and direct operators to `pose_mcp_context` after
configuration changes or workspace switches.

Extend ordinary `project_unknown` tool results additively with the authorized
IDs visible to that caller plus structured remediation. Preserve existing
`error_code` and `project_id` fields.

Bound one discovery call to a fixed number of per-project authorization
decisions and report truncation to the caller rather than silently returning a
partial registry as if it were complete.

## Consequences

- Positive: agents can prove which server process and project registry they are
  actually using before reading governed state.
- Positive: static configuration health can no longer be mistaken for a live
  connection probe in machine-readable doctor output.
- Positive: discovery reuses the existing authorization and audit boundary and
  does not expose host topology.
- Trade-off: policy-backed discovery may evaluate one authorization decision
  per registered project. Since `project_unknown` remediation is an implicit
  error path, a client repeatedly guessing project IDs would otherwise amplify
  each mistake across the whole registry, so discovery carries an explicit
  probe ceiling and reports truncation.
- Trade-off: with no OPA endpoint configured, `PolicyGate` allows by default,
  so discovery returns every registered project. That is the pre-existing
  dev-mode posture of every other tool — the same caller could already read
  those projects directly — but `pose_mcp_context` is the first tool whose
  purpose is enumeration, so deployments that treat registry membership as
  confidential must configure a policy endpoint.
- Trade-off: a policy that requires an explicit project scope may require the
  caller to probe a known `project_id`; POSE does not bypass that policy to make
  discovery more convenient.
- Rejected trade-off: automatically reload environment-backed roots inside the
  server; a running process cannot safely infer that a client-side `.mcp.json`
  changed, so restart/reconnect remains explicit.

