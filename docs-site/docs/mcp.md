# MCP server

`pose-mcp` (also available as `pose serve-mcp`) exposes the whole POSE
instance to MCP-capable agents — read-heavy by design. Transports: stdio
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

The installer generates `.pose/bin/pose-mcp-claude`, a wrapper that derives
`POSE_PROJECT_ROOT` from its own location — nothing is hardcoded — and seeds
`.mcp.json` when the repo has none.

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

## Security posture

- Default deny on OPA errors; policy decisions are audited
  (`policy.decided` / `policy.violation` structured logs).
- `pose_suggest`-style tools that shell into the CLI validate every argument
  and never do PATH lookups.
- Multi-replica deployments need the Redis cursor store (enterprise hardening
  track); single-node dev needs nothing beyond the binary.
