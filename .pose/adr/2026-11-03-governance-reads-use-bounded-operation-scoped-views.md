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

Use these recalibrated hard budgets: 25,000 observed records including retained
directory entries, 512 MiB captured input bytes, 8 MiB retained metadata,
one retry of a complete read-only operation,
at most 16 authorized federated roots and depth 8 with cycle detection. A
budget overflow produces `unknown`/`incomplete`, never truncation. Reconcile
observed paths even when no watcher is present or its events were lost; a
watcher is only a hint. Do not add a watcher, daemon/IPC, global cache, B31
retention policy, B33 graph cache or B35 worktree runtime in this ADR.

Clarification recorded on 2026-11-05: the depth-8 budget bounds project
crossings along a dependency path, including authority-transfer redirects;
it does not replace the existing total dependency-path limit of 64 edges.
Retain the existing limits of 256 distinct graph identities and 256 input
dependency entries per artifact. A shared DAG suffix is visited once, but its
maximum remaining path cost must be checked at each incoming prefix. Entries
already visited do not consume another distinct-node slot. Redirect chains
also retain their independent limit; they cannot conceal project crossings
when composed with dependency edges. The operation still authorizes at most
16 roots and retains the same byte, metadata, observation and retry budgets.

This corrects the scope of the existing bound rather than raising it. The
native spec catalog at implementation checkpoint `f54e9285` has 482 specs and
553 local dependency edges, no cycles or foreign spec edges, a longest local
path of 27 edges and at most 111 reachable spec identities. The index preflight
refused this legitimate corpus when depth 8 was applied to local edges. The
measurement and its exclusions are recorded in the B34 budget report; actual
index publication and composed-bound fixtures still require current validation.

Capacity recalibration D6, recorded on 2026-11-05, supersedes the initial
10,000-record / 128-MiB values prospectively. A complete read-only compiler
calibration of the real builder and its final audit retained 18,905 combined
records, 370,690,278 input bytes and 4,479,003 metadata bytes for seven outputs.
The capacity report records the production refusals, diagnostic overlays,
counter measurements and limits of that evidence. The chosen caps provide
bounded headroom without changing input coverage, truncating history or
altering other limits. Keep budget/unknown causes when a callback fails;
a missing closure must not be classified solely as an invalid policy path.
Calibration does not qualify the normal build or authorize index publication.

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


## Accepted amendment — 2026-11-06: bounded multi-scope coordination (D7)

The real global strict check exhausted the shared governance record/metadata
budget across independent historical closeout evaluations. Separate the global
coordinator from scope operations: capture policy, candidate enumeration and the
bounded immutable catalogue under an explicitly bound coordinator; audit those
selection inputs after all workers join, including disabled-policy and empty
candidate paths. Each scope, recursive children and exactly authorized federated
roots use their own bounded capture and final audit. Bound worker concurrency
and retain evaluated results, not every scope's captured closure.

Catalogue reuse must expose an explicit binding-aware interface; do not bypass
root, project, resolver or Files identity checks by unwrapping a live view. Buffer
findings until the relevant scope and coordinator audits complete. A localized
loss must remain unknown and make the global gate fail without exhausting an
unrelated sibling. Preserve complete historical review and attestation reads,
later negative decisions, replacements, child validity and current trust checks.

This composes coherent global selection and individually audited scope results.
It does not certify one simultaneous filesystem snapshot across every scope.
The existing per-operation caps remain unchanged. Keep real domain errors with
closure audit errors; unavailable metadata and exhausted budgets are not proof
of an escaping path or observed content mutation. Regression tests must exercise
the real global orchestration under injected small budgets and serial/parallel
workers, selection interleavings, localized exhaustion, retention negatives and
exact binding checks. Native whole-corpus verification is still required.


## Instance execution configuration boundary (D8) — 2026-11-06

The user-authorized native review uses a local delegation policy with concrete
vendors, executables, access and executor issuer. Keep that live policy out of the
embedded scaffold through the existing instance-policy exclusion mechanism.
The target instance uses its native delegation setup and its own configuration;
no engine-local agent profile or issuer becomes a target default. This preserves
the same authority boundary as neutral review policy and changes no capability
adoption, historical record or review verdict.


## Bounded executable identity for the selected review agent (D9) — 2026-11-06

The installed native Claude Code 2.1.318 executable is 337,814,032 bytes. Its
provenance setup failed before the vendor ran because executable hashing reused
the 256 MiB transcript-retention bound. Hash executables as a stream bounded to
512 MiB, separately from evidence retention. Keep regular-file and identity
checks, bounded reads and rejection above that limit. Keep transcript and B34
read-view budgets unchanged. Do not substitute a launcher to evade the selected
executable's identity proof. This accepts the measured distribution with bounded
headroom; larger future distributions still require an explicit reassessment.


### Provider framing for the instance-owned review profile

Claude's stream repeats session_id in init and result events. The instance
profile selects the unique system/init event so the existing ambiguity guard
and private-session redaction remain intact. A restricted, noninteractive review
uses Read/Grep/Glob/Bash without Edit/Write/Agent tools or plan approval mode.
This changes neither independence nor stream/output/run limits; the actual
working tree remains checked for mutation. No vendor profile becomes a scaffold
default.
