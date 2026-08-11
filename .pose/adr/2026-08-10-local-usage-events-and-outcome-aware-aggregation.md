# ADR: Local usage events and outcome-aware aggregation

## Status
Accepted (2026-08-10) — spec `pose-usage-metrics`

## Context

POSE already has three partial signal systems: workflow report history, anonymous command telemetry and MCP OpenTelemetry. None can answer which tool produced useful project outcomes. Report history sees tasks rather than tool calls; anonymous telemetry carries only command names and is remote opt-in; MCP OTel treats a deterministic gate returning `passed=false` as a successful transport call. Asking agents to increment counters would be unverifiable and would make adoption depend on prompt compliance.

The solution must work offline, avoid dirtying the governed worktree on every command, remain project-scoped across Git worktrees, preserve existing exit/output contracts and never create an individual productivity score. Findings need stable comparison without persisting their raw IDs, which may contain module or file paths.

Alternatives considered:

1. **Write usage JSONL under tracked `.pose/events/`.** Reuses the DORA layout, but every POSE invocation dirties Git and usage noise becomes governance source content.
2. **Use only OTLP or anonymous telemetry.** Avoids local storage, but fails offline, requires operator infrastructure, cannot safely retain project-level finding lifecycles and conflates operational telemetry with product analytics.
3. **Store privacy-bounded events outside the worktree and aggregate locally.** Resolve the Git common directory (shared by worktrees), fall back to a user-cache directory for non-Git projects, HMAC every scope/finding identity with a local random salt, and expose only aggregate reads.
4. **Parse stdout/stderr centrally.** Covers commands quickly, but arbitrary output is not a stable contract and one verbose failure can fabricate hundreds of findings.

## Decision

Choose option 3, with typed result adapters instead of option 4.

- Record one schema-versioned terminal event at the CLI `Main` and MCP `callToolCtx` choke points. Recording is best-effort and cannot alter the wrapped result.
- Persist only tool, surface, timestamp, duration, bounded outcome enums, aggregate counts, version, HMAC scope and HMAC finding identities. Never persist raw arguments, output, paths, repository/project names, source content, principals or run IDs.
- Store monthly append-only JSONL under `<git-common-dir>/pose/usage/`; use an OS user-cache directory keyed by the SHA-256 of the absolute root only when no Git common directory is available. The storage location is local state, never installation machinery.
- Treat execution outcome and semantic outcome separately. A gate can complete successfully as a tool call while semantically failing because it found blockers.
- Count exact findings only from typed contracts. Structured validation contributes each failed/errored stable check ID; assessment/domain results contribute their own stable finding IDs. A generic failing gate contributes one conservative observation.
- Compare finding lifecycle only when both events declare complete sets and have the same tool, surface and HMAC scope. Missing identities from an incomplete result never imply resolution.
- Expose `pose usage` and read-only MCP `pose_usage` from the same aggregator. Exclude the query commands themselves to prevent recursive self-inflation.
- Keep remote anonymous telemetry and OTel separate. A future exporter may project aggregates only through explicit opt-in; this ADR authorizes no network transmission.
- Keep DORA and POSE usage as separate reports. Correlation can be studied later, but the engine never claims that tool use caused a delivery outcome.

Impacted modules: `pose-mcp/internal/{usage,cli,mcpserver}`, public CLI/MCP documentation, catalog golden fixture and embedded scaffold.

## Consequences

- Positive: tool counts and outcome/finding metrics require no agent-authored counters and work for CLI plus MCP.
- Positive: worktrees remain clean and multiple worktrees for one repository share the same project journal.
- Positive: HMAC identities support uniqueness and lifecycle comparison without disclosing raw finding or scope identifiers.
- Positive: transport errors, semantic failures, unavailable data and zero findings remain distinguishable.
- Negative: JSONL append is only atomic enough for one bounded write on ordinary local filesystems; malformed lines are counted and skipped rather than repaired silently.
- Negative: commands without typed results initially yield conservative failure observations, not detailed error counts.
- Negative: local cache state does not follow a clone to another machine; explicit aggregate export is future work.
- Neutral: no new dependency is introduced; SQLite durability and richer queries are rejected for now to preserve the standalone binary's dependency and cross-platform profile.
- Neutral: DORA correctness migration remains a separate decision because it changes delivery event semantics rather than usage instrumentation.

Rejected trade-offs: individual ranking is structurally absent; stdout line parsing is rejected as non-deterministic; automatic remote export is rejected by the existing opt-in/privacy boundary.
