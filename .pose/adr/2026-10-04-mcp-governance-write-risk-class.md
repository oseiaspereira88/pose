# ADR: MCP governance-write risk class

## Status
Accepted (2026-10-04) — spec `pose-action-requests`. Extends
[MCP tool catalog is a release-gated contract](2026-07-19-mcp-tool-catalog-is-a-release-gated-contract.md).

## Context

`pose-action-requests` R1 asks for an MCP equivalent of `pose action open`, so an
agent working through MCP can raise a material question without shelling out to the
CLI. The catalog's three risk classes do not describe that tool. `read` and `gate`
both promise no writes, and `external-side-effect` means emitting events to another
system. Opening a request appends to a repository-owned journal under
`.pose/actions/`. That is a local write that reaches no network and that the CLI
already performs.

Alternatives considered:

1. **Classify it as `gate`.** Rejected: `gate` promises no writes, and reviewers and
   OPA policies rely on that promise.
2. **Keep MCP read-only and amend R1.** Rejected: agents are the usual requesters, and
   a requester that cannot ask is pushed back to free text in chat, which is what
   action requests replace.

## Decision

- Add the risk class `governance-write`: the tool appends to a repository-owned
  governance journal, previews unless `apply` is true, and never touches the network.
- `pose_action_open` is the first tool in the class. It calls the same domain function
  as the CLI (`PrepareActionRequest` for the preview, `OpenActionRequest` with
  `apply`), and returns the same id and request digest for the same input.
- Resolution stays CLI-only. An answer must name the digest and revision it answers
  and come from the recipient role, and no MCP tool records one.

## Consequences

- The catalog golden, the conformance test's accepted classes, the telemetry
  attribute vocabulary (`read`/`gate`/`governance-write`/`external-side-effect`) and
  `docs-site/docs/mcp.md` name the new class.
- Policy authors can deny `governance-write` tools as a class without denying gates.
- A later write tool must fit the same definition (local journal, preview by default,
  no network). Anything else needs its own decision.
