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

## Amendment: bounded, isolated graph caches (B33)

Spec `pose-bounded-isolated-graph-caches` fixes the cache accounting and view
ownership this ADR left to acceptance work. Rebuildable derived data (the parsed
delivery graph, the design-delta report, the test catalog) goes through one cache
in the domain package; the paths a commit changed are read by the governed capture,
bounded and not cached:

- **Bytes, not entries.** Each value reports its retained size; the cache evicts
  the least recently used entries under a global budget and a budget per owner (a
  worktree root). The parsed-graph budget is 192 MiB global and 96 MiB per
  worktree. The size is estimated by walking every field of the value (strings,
  slices, maps, pointers), so no variable-sized field is charged a constant; the
  estimate is calibrated against the measured heap of the real index (it errs
  high, about 2.3 times). A value larger than its budget, and a value its producer
  declares unretainable, is served and neither retained nor copied for retention.
  An entry a reader pins (a view in progress) stays charged after eviction until the
  reader lets go, and the production reservation is held until the result is
  published, so what the cache keeps alive is what it accounts for.
- **Work in flight.** A production reserves bytes before it runs, globally and per
  owner, and waits (or leaves with its context) until they fit; the number of
  concurrent productions, of queued keys (globally and per owner) and of waiters per
  key is capped, a waiter that leaves gives its place back, and a caller refused for
  lack of capacity gets `ErrCacheBusy` on every consumer: overload is an error the
  caller sees, never a production outside the cache. Copies run outside the lock, one of a
  few copy slots each, so a large copy never delays another key. The clone a caller
  keeps after it returns is the caller's memory and is outside the budget; a reader
  that only looks up fields uses a view and keeps nothing.
  The same holds for the input a caller read before asking and the clone being made
  for it: they are the caller's memory, and bounding them is a recorded follow-up. A
  shared result stays charged until its last waiter has copied it. A gate that needs
  the test catalog treats an overloaded cache as incomplete validation, not as an
  empty catalog.
  Every retained entry is also charged its metadata (the entry, its list node, the
  lookup slots, 256 bytes), its key and its owner, so many small entries cannot exceed the
  budget while the payload counters stay within it. Publication (the waiters a result has, the removal of the flight, the release of the
  reservation) is one critical section. The one read whose output is unbounded by
  nature, the paths of a commit, is bounded as above.
- **Key.** Parsed graphs are keyed by engine version, schema and the digest of the
  bytes: identical bytes cannot yield a different graph, so two worktrees holding
  the same index share one entry and none is served another's graph; the same size
  or modification time proves nothing. The test catalog is keyed by root and the
  digest of the validation matrix and expires after 30 seconds, because it reads
  the working tree; a long-lived server can serve a catalog built before a test
  changed for up to those 30 seconds when the matrix did not change (the accepted
  freshness window), never longer. The paths a commit changed are read by the governed capture, which audits every read and so is not served from a cache: the read is bounded instead, in output (4 MiB) and in paths (100,000), and a larger commit answers unknown and marks the governed read unknown.
  The design-delta report is keyed by root and the digest of its subject and
  options, metered per worktree, assessed once per key, and retained only when
  complete: a partial or unknown report comes from a bound or a failed read that a
  later call must recompute.
- **Misses.** A miss is produced once per key however many callers ask together;
  the production runs outside the lock, so one key never delays another. A waiter
  whose context ends leaves without disturbing the producer or the other waiters.
  A failure is never remembered (no negative caching): the next call produces
  again. Producers and waiters are bounded by the callers themselves; no pool or
  queue is created.
- **Isolation.** Every value handed out is a deep copy: the retained one is never
  shared. The design-delta report is copied by reflection, so a field added later is
  isolated without anyone remembering it; the graph keeps a hand-written copy
  because it is hot, and a reflection test holds it to the same standard. Read-only consumers that only look up a few fields use a view
  (`ViewDeliveryIntegrity`), which hands out the retained graph without a copy and
  forbids modifying or keeping it; the cache never writes a retained value. The
  aliasing the audit could not reproduce is reproduced and fixed: the attribute
  maps of the graph's nodes and the refs of the roadmap criteria were shared with
  the cache.
- **Not caches of authority.** Nothing here remembers a verdict, a satisfaction or
  an authority; gates and writes read current inputs. The Git batch reader of the
  earlier ADR stays scoped: no global pool or shared stream is introduced.

Cost, measured on the real 10.9 MB index: a deep copy per hit costs about 7 ms
and 18 MB of allocation where the shallow copy cost 1 ms and 5.7 MB; the view costs
about 39 KB. That is the price of isolation, paid only by consumers that need a
mutable graph.

## Amendment: common operations and the typed handoff (B31)

Spec `pose-common-domain-and-cli-mcp-handoff` fixes how an MCP route shares code
with the CLI, without a resident service:

- **Five route classes.** Every MCP tool is declared one of: domain-direct,
  cli-wrapper, read-only-planning, admitted-write, external-handoff. A test keeps
  the declaration equal to the advertised catalog, so a tool cannot ship without a
  class, and checks that the tools that may write are exactly the four governance
  write points (`pose_action_open`, `pose_action_resolve`, `pose_action_fulfil`,
  `pose_review_attest`), which revalidate authority, assurance, policy and subjects
  themselves. The MCP-Conductor-Harness boundary (`pose_validate_*`,
  `conductor_run_*`) is external-handoff: `tools/call` never becomes the execution
  of a project's checks.
- **No second binary for read-only operations the CLI package owns.** The domain
  wrappers run a registered command function in process when it is read-only and
  bounded (`suggest`, `followups`). The function is the single source: results,
  errors and coverage are the CLI's. The contract of the spawned run is kept: a
  non-zero exit is an error carrying the command's message, and a deadline or a
  cancellation is an execution failure with no verdict (the run cannot be killed, so
  it finishes in the background and writes nothing; a caller that has already
  cancelled never starts it, a completion that races a cancellation is discarded, and
  the runs alive at once, abandoned ones included, are capped so that repeated
  timeouts cannot accumulate work). The gates keep the process
  boundary: they are long, can be killed on a deadline, and must not leave a half
  verdict. Spawn attempts and in-process runs are counted so the extraction is
  evidenced by count, not by promise.
- **A typed handoff, not an execution.** A closeout plan with steps left carries an
  `OperationHandoff`: the operation and scope, the CLI invocation as structured
  arguments (no shell line), the plan, spec, revision, policy and generation digests
  it was computed from, the authority it needs, how the outcome comes back and when
  it stops being valid, and `executes: false`. The request id is derived from the
  operation, scope and plan digest, so a stale handoff is not mistaken for the
  current one. It names the project and root the caller selected and does not
  retarget itself. It carries no secret, grants no authority and infers no context:
  `pose close --apply` takes no context flag, its binding is the full plan digest it
  names (the digest the apply compares, with its `sha256:` prefix), and the performer
  verifies the authority and revision with a fresh `pose context` in the target root
  before applying. A plan that is terminal, has any blocked step or blocker, or whose
  read view is unknown has none.
- **Not shared by sharing code.** stdio is not an IPC channel shared between
  clients; an MCP transport gains no Execution Identity, signature or human presence
  of the CLI by calling the same function.

## Related specs

- [B31](../specs/2026-10-14-pose-common-domain-and-cli-mcp-handoff.md)
- [B33](../specs/2026-10-14-pose-bounded-isolated-graph-caches.md)
- [B35](../specs/2026-10-14-pose-optional-worktree-runtime.md)
