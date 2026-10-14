# ADR: Shared domain and optional worktree runtime

## Status

Proposed — 2026-10-14, for B31/B33/B35. No IPC, cache API/schema, daemon lifecycle,
platform default or policy is implemented/adopted by this proposal. Reconcile and
resolve the structural contract before functional start. Preserve accepted ADRs
for active-context discovery, scoped Git readers, obligations and the
MCP-Conductor-Harness validation boundary.

## Context

The [performance analysis](../reports/2026-10-12-pose-front-08-resident-performance.md)
separates computation, transport and freshness costs. CLI-backed Store adapters
spawn the binary with a fixed deadline; MCP persistence alone does not remove
repeated algorithms or graph copies. The [UX analysis](../reports/2026-10-12-pose-front-07-operational-ux.md)
shows plan/context composition and planning handoff gaps while P15 read parity
already works. The [consistency analysis](../reports/2026-10-12-pose-front-09-consistency-recovery.md)
identifies nested mutable aliasing and generation/coordination boundaries.

Cached graph copies include shallow Nodes.Attributes, misses are not coordinated
per key, and the single last graph can churn across projects. These are code-founded
observations, not measured corruption by a current consumer or a native race.
A client-owned MCP stdio stream cannot be attached by an unrelated CLI. Existing
sessions/cursors/registry mutexes do not prove worktree IPC or write serialization.

## Decision

Propose these boundaries for review, without choosing wire/schema details yet:

- Extract one typed domain interface for admitted operations. CLI/MCP adapt syntax
  and rendering, with common results/coverage/generation/error semantics. Preserve
  the adapter-not-fork principle; remove the POSE subprocess only where extraction
  is qualified. Legitimate scoped Git and external Harness execution remain separate.
- Preserve project selection and actual authority/assurance. Read-only planning or
  typed CLI handoff does not authorize writes; sharing code, process UID or stdio
  does not create Execution Identity, signature or human presence. Active-context
  discovery keeps authorized logical IDs and never reveals unauthorized roots.
- Keep validation orchestration outside arbitrary tools/call execution. Native
  structural/read-only evaluations and explicitly admitted operations use the
  domain; project check execution still requires its existing request/approval/
  Harness contract. Scheduling, agents, workflow retries and notifications stay
  outside the POSE core. Bounded admission/in-flight work here is evaluation of
  requested calls, not a durable workflow queue or an autonomous scheduler.
- Expose effectively immutable graph views or deep defensive copies at mutable
  compatibility boundaries. Keys identify worktree, content, parser/engine/schema
  and complete dependencies/generation; current authorization/verdict is not
  inferred from a cache hit. Coordinate one production per key, budget retained
  bytes/pins/waiters per worktree and globally and measure allocations/RAM.
- Preserve the scoped Git batch-reader decision. Residence does not authorize
  a shared cat-file stream, persistent/global reader pool or shell expansion.
  Any future change of its ownership/lifetime needs a separate structural decision.
- Make runtime optional, identified by user + canonical worktree/Git identity +
  engine/build/protocol/schema/capabilities. project_id/common Git directory alone
  cannot distinguish mutable worktrees. A separate protected local IPC endpoint
  may host this domain; platform transport (Unix socket/named pipe or equivalent),
  authentication/permissions and compatibility require resolution before code.
  Never take a client's MCP stdio connection or claim a static doctor sees it.
- Support explicit attach/start/status/stop/restart and standalone/CI fallback.
  Restart/update must not inherit stale generation/authority. An ambiguous committed
  write requires receipt/reconcile before retry through another process; query
  fallback validates inputs. Watch events are hints; B14/B15/B34 own publication,
  coordination and coherent read generations, including external edits.
- Bound requested evaluation concurrency by category/worktree and globally, with
  CPU quota/affinity, memory/IO/child-process and admission metrics. Protect short
  queries during heavy work and preserve deterministic output. Fix budgets and
  benchmark thresholds on comparable corpora before adoption, rather than changing
  defaults from one historical sample or claiming p95 from three observations.

## Consequences

B31 yields standalone semantic reuse, B33 supplies isolated/bounded graph views,
and B35 consumes both after B07/B12/B14/B15/B32/B34 prerequisites. API/caches can
be qualified without requiring a resident service; B35 stays opt-in after delivery.
The exact IPC/protocol, compatible GateResult/envelopes, platform support, cache
accounting/view ownership and lifecycle/QoS thresholds remain acceptance work.
No new root enumeration, command, default, workflow queue or policy is adopted.
Historical benchmark numbers and the duration of documentary checks do not prove
runtime speedup. Consumers retain existing mutable contracts until qualified
migration, and the source/effects/assurance remain authoritative.

## Alternatives rejected

- Mirror all .pose into RAM at startup: loads cold history and retains repeated
  parsing/copying/algorithms without establishing freshness or a memory budget.
- Reuse the MCP client's stdio as shared IPC: the stream belongs to its initiating
  client and supplies neither attach compatibility nor authority for another CLI.
- Fork CLI rules in MCP/runtime: improves transport at the cost of divergent
  result, validation and authorization semantics.
- Treat watcher events or mtime/clean as proof: misses external changes and cannot
  bind a gate/write to complete current inputs.
- Run arbitrary validation checks in a new in-process MCP helper: bypasses the
  established execution-identity/approval/Harness boundary.
- Mandatory autostart and unlimited work/cache: removes CI/standalone fallback,
  risks memory growth and query starvation without a measured benefit.

## Review triggers

Before B31/B33/B35 start, settle interface/compatibility, caller mapping, local IPC
platform/identity, budgets and benchmark thresholds. Revisit if a scoped reader
lifetime changes or a new external execution capability is proposed. Track the
expiring operational review in knowledge:adr-shared-worktree-runtime-review.

## Related specs

- [B31](../specs/2026-10-14-pose-common-domain-and-cli-mcp-handoff.md)
- [B33](../specs/2026-10-14-pose-bounded-isolated-graph-caches.md)
- [B35](../specs/2026-10-14-pose-optional-worktree-runtime.md)
