# MCP server

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

The installer seeds `.mcp.json` when absent. It invokes the native binary
directly and records the installed project's root and project id in the server
environment; no wrapper or second executable is generated.

## Tools

| Tool | Returns |
|---|---|
| `pose_list_specs` / `pose_get_spec` | Spec inventory (lifecycle frontmatter) / full spec |
| `pose_spec_readiness` | Is a spec eligible? Resolves `depends_on` refs (specs, milestones, roadmaps) |
| `pose_list_roadmaps` / `pose_get_roadmap` | Governed roadmaps and their milestone DAGs |
| `pose_get_changelog` | User-facing changelog fragments |
| `pose_get_followups` | Aggregated follow-up backlog |
| `pose_check` / `pose_lint_spec` | Run the deterministic gates |
| `pose_suggest` | Canonical trail per task type |
| `pose_get_workflow` / `pose_get_rules` / `pose_get_skill` | Operating procedure content |
| `pose_list_knowledge` / `pose_get_knowledge` | Operational memory |
| `pose_list_reports` / `pose_get_report` | Validation evidence |
| `pose_insights` | Deterministic outcome aggregates by workflow, task or context |

## Security posture

- Default deny on OPA errors; policy decisions are audited
  (`policy.decided` / `policy.violation` structured logs).
- Shared-domain tools run in-process; CLI-backed tools invoke the current
  native executable. Every argument is validated and shell text is never
  evaluated.
- Multi-replica deployments need the Redis cursor store (enterprise hardening
  track); single-node dev needs nothing beyond the binary.
