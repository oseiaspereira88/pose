# ADR: Governance reads use bounded operation-scoped views

## Status
Accepted

## Context

Spec `pose-governance-read-view-generation` (B34) finds a concrete gap in the
current obligation projection: `CurrentObligationSnapshot` runs before its
producers, and its digest binds revision, dirty boolean, policy, contract,
authority context and one index digest, but not the action journals, subjects,
review records and other bytes actually consumed by those producers. A tree
that is already dirty can change without changing the dirty bit. The projection
does not validate those inputs after the producers finish.

Four accepted contracts already cover parts of this problem:

- B06 (`2026-10-17-a-source-that-lost-something-is-incomplete-for-what-depends-on-it`)
  requires localized, per-source coverage when a read loses an input.
- B07 (`2026-10-18-git-state-that-cannot-be-read-is-unknown-and-a-worktree-is-judged-by-its-content`)
  requires unknown Git/worktree state to stay unknown and judges a worktree by
  observed content, not only status flags.
- B14 (`2026-10-16-writers-publish-through-one-exclusive-cas-publisher`)
  provides per-artifact CAS and lifecycle locks, while explicitly excluding
  transactions across files and the final check-to-rename race with a writer
  that ignores POSE locks.
- B15 (`2026-10-17-index-sets-are-published-as-verified-generations`) pins and
  verifies one immutable index set per root, not a global snapshot across
  independent roots.

`reviewOperationView` already captures bounded spec metadata and review bundle
headers for one operation, serves reusable observations, and audits their
inputs before a green result. It is intentionally not an atomic filesystem
snapshot, does not cover all producers, and is not a write authorization. The
new contract should extend that seam and the existing `Files` abstraction;
another parser, global hash, daemon, watcher-based proof or unbounded cache
would duplicate or overstate the existing guarantees.

## Decision

Add an operation-scoped, read-only view over the
existing `Files` abstraction. Capture only the paths, directory listings,
errors and absences actually observed. `ReadFile` and `Open` must share the
captured bytes; metadata and link resolution participate in the read set.
Serve immutable captured bytes to all producer clones and record attribution
by an explicit producer/scope carried with each clone. Audit the observed
closure against the original `Files` at operation end. Use a bounded retry for
read-only operations; discard the attempt before retrying. If the closure,
budget, producer attribution, parser/schema version or authorized root binding
cannot be proved, preserve readable data where useful but return
`incomplete`/`unknown`; never certify a truncated or mixed result.

Start with `ProjectObligations`, which is the shared domain projection for CLI
`state attention` and MCP `pose_obligations`. Pin indexes once per root and
pass that store/view through all producers. Then share the capture/audit
primitive with `WithReviewOperationView` and integrate review/closeout. Producers
that read through native `os` or accept only an uninstrumented `root` path must
be moved to `Files` or explicitly remain uncertified. Do not change the
meaning of existing sources or discard history to fit the budget.

Expose an additive `read_view` object on the obligation report, with its own
`schema_version: 1`, a digest of observed inputs and a vector of
root/project/worktree/ref/index bindings plus known engine/parser/schema
versions. Keep `ObligationReport.schema_version` at 1, keep `Obligation` and
historical bundles at their existing versions, and document the field in
`POSE.md` and the MCP catalog. Do not expose absolute host paths. A legacy
report or historical record without a binding remains readable with a
limitation, but is `unknown` when a current gate requires proof. Existing
`coherent`, `complete` and producer coverage remain distinct; freshness is
evaluated at final validation, not promised beyond it.

For effectful operations, do not accept a read-view ID as authorization.
Acquire an explicit cooperative write guard for each authorized root before
recomputing the complete evaluation and retain it through publication and the
receipt. Acquire roots in canonical order, then resource locks. Reuse the
existing per-root transfer lock and cross-platform advisory-lock primitives;
carry the lease explicitly through the operation's `Store`/`Files`, never a
process-wide or goroutine-local reentrancy flag. Each participating publisher
must use the same guard protocol. A CAS on output B does not fence input A.
Migrate and test direct journal and immutable-record writes as well as B14
publishers before claiming that an effect is protected. The index guard must
cover construction of outputs and capture of their input bindings, not only
the final descriptor publication. Independent roots keep separate guards and
generations. Unavailable coordination is an explicit refusal, not permission
to reuse an earlier authorization.

Revalidate closure and authority at the actual effect point under the existing
resource/lifecycle locks and B14 CAS or B15 descriptor commit where applicable.
Cover start/action-request/review seal, attestation and closeout paths. A
conflict does not produce an effect. A `publish-unsynced` result continues to
mean the write committed but durability is unconfirmed; do not report it as a
plain failure without effect. Publish an index generation only after its B15
descriptor commit. Because B34 adds no cross-operation cache or persistent
global generation, a write retires the prior operation view rather than
publishing a new global generation.

Use these initial hard budgets: 10,000 observed records, 128 MiB captured input
bytes, 8 MiB retained metadata, one retry of a complete read-only operation,
at most 16 authorized federated roots and depth 8 with cycle detection. A
budget overflow produces `unknown`/`incomplete`, never truncation. Reconcile
observed paths even when no watcher is present or its events were lost; a
watcher is only a hint. Do not add a watcher, daemon/IPC, global cache, B31
retention policy, B33 graph cache or B35 worktree runtime in this ADR.

An external writer that changes and restores bytes between audits without
participating in a fencing protocol leaves no proof for hashes or `lstat` to
observe. A view may claim content equivalence at final validation, but cannot
claim that no transition occurred. Where a gate depends on that transition,
require a participating writer proof/lock; otherwise return `unknown` and do
not execute the effect. This is not a transaction across the filesystem.

## Alternatives rejected

- **Hash the whole repository or `.pose` on every call:** hashes are not the
  actual read closure, read unrelated history, and can miss an ABA transition.
- **Reuse a report or cache as authorization:** a read-only observation does
  not reserve inputs or authority for a later effect.
- **Add a process-wide generation registry or watcher proof:** it cannot cover
  restarts or external writers and belongs to separately governed runtime
  work.
- **Copy all source files into a snapshot:** unbounded eager history and new
  filesystem semantics duplicate the operation view rather than extending it.

## Consequences

CLI and MCP can expose the same operation identity through their shared
projection. Consumers see an additive report schema version; old reports stay
readable but lack proof for current authorization. Operation memory and final
audit work are bounded; overflow or uninstrumented reads can make a producer
unknown. Validating observed files twice adds I/O, and one retry can rerun the
read-only producers. Existing B14/B15 writers remain the source of truth for
effects and index commits.

The engine does not promise an atomic filesystem snapshot, detect an external
ABA that leaves identical final bytes, or defend against writers that bypass
POSE fencing. Such inputs cannot authorize an effect whose contract requires
transition evidence. More producers become current only when their read paths,
coverage and writer boundaries are explicitly brought under the view.
