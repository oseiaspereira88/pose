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


## Capacity recalibration of captured input bytes (D10) — 2026-11-13

Closing spec `pose-closeout-review-policy-by-cost-benefit`, `pose index` in a
fresh clone and then in the maintainer checkout refused the pose-dist corpus:
`index.delivery-integrity` exceeded the captured-input budget and
`index.generate` was left without a complete observation. Measured with a Git
shim on the real builder: the change sets of every spec read `git show
--binary` for 2,451 distinct trailer commits, 492 MB of Git output in the
capture (the final audit reads it again under its own budget), plus the file
inputs of the generation, so the operation needed more than 512 MiB. Nothing
was truncated; the refusal was correct. Raise the captured-input budget to
1 GiB, for capture and audit alike, and keep every other limit, including the
512 MiB executable-hash bound of D9. Observations retain their bytes, so the
operation may now hold up to 1 GiB of captured input in memory. This
recalibration buys headroom, not a scaling property: the corpus grows with
every attributed commit. The structural fix, retaining only a digest for pure
reads of immutable Git objects whose bytes a producer only hashes, is a
follow-up of that spec, and a further raise again requires a measured
reassessment.

## Streamed object reads restore the captured-input budget (D11) — 2026-11-14

Spec `pose-read-view-streamed-object-digests` supersedes D10. A Git object
read whose bytes a producer only hashes (the `git show --binary` of each
commit that feeds a change set's diff digest) is streamed into the producer:
the read view retains its digest and length, never its bytes, and charges a
separate streamed budget of 8 GiB per operation, and 8 GiB again for the
final audit, with at most 256 MiB per read. The final audit streams every
streamed read again: a patch's rendering depends on Git configuration and
attributes the operation does not capture, so no read is skipped on the
strength of its object id. The same bytes enter the
same hash, so every diff digest and change-set id is unchanged: on the
pose-dist corpus the 567 change sets and the whole `delivery-integrity.json`
were identical before and after. The captured-input budget returns to
512 MiB. Measured on the pose-dist corpus, `pose index` runs two operations:
the first retained 21.8 MB and streamed nothing, the second retained 32.2 MB
and streamed 1.26 GB in 2,511 reads, then 523 MB again in its audit.
Streaming bounds memory, not I/O: the bytes read still grow with the
attributed history.


## A coordinator checks the consumed index's inputs for its scopes (D12) — 2026-11-15

Spec `pose-check-shares-the-federated-material` refines D7. Under one
coordinator, every scope consumes the same pinned index generation, and its
inputs (the federated material among them) are the same for every scope.
D7 made each scope recompute them inside its own capture. With the
pose-dist corpus as a federated dependency, the federated material of one
harne8 check costs about 150 s and several hundred MiB of retained reads.
Each of 16 workers recomputed and retained it: `pose check --tolerant`
passed 10 GB within a minute, and four workers took 37 minutes.

A scope operation created by a coordinator delegates its check of the
consumed index's inputs to the coordinator operation. The bindings are
computed once per (root, generation, producer), through the coordinator's
Files and charged to its budget. The coordinator's final audit, which D7
already runs after every scope joins, covers those reads. A scope's own
closure keeps everything else it reads. An input that changes while the
scopes run fails the coordinator's audit, so the gate fails: no scope is
judged against a freshness check its closure does not cover. Measured on
harne8: 248 s at a 2.5 GB peak with 16 workers, against an abort past
10 GB.

This shares one immutable answer (whether the consumed generation's inputs
are current) in the binding-aware way D7 requires: scopes already prove the
coordinator's authorized root vector before a capture. It does not make the
scopes one snapshot.

## The index's federated material is kept with its read closure (D13) — 2026-11-16

Spec `pose-index-material-is-kept-per-repository-state` extends D12 across
operations. Every review operation checks that the index it reads is
current, and the federated material is one of the index's inputs. On harne8,
with pose-dist as a dependency, computing that material took about 90 s of a
140 s closeout, and a long-running MCP server paid it on every call.

A first design keyed a kept digest by each repository's HEAD, status and
dirty bytes. Its review refused it: the key missed what the material really
reads (nested repositories, ignored files), its reads ran outside the read
view, and a hit left the material's inputs out of the operation's closure
and final audit.

The material is now built in an isolated read-view operation over the
requested store's root, the same base binding and roots, with the resolver
bound to that operation, so a dependency's reads are in it too. Everything that operation observes is the material's
closure: file bytes, metadata, listings, links and captured Git commands.
The closure is kept with the digest only when that operation recorded
nothing unknown, streamed nothing, and passed its own final audit. Any later
operation of the process replays every read of the closure through its own
read view before reusing the digest. The replay records each read, so the
operation's closure and final audit cover every input of the material, and
any read that returns other bytes rebuilds the material. A build that cannot
be kept is followed by the operation's own build. The kept entry is named by
the engine, the project, the store root and the resolver's registry (its
configuration, each project's root or lookup error, its conflicts), which
decide outcomes no file read shows. Authorization is per-call policy: the
build records every decision it consulted, and a hit asks each one again
before reuse.

Measured on harne8: the second closeout call of one MCP server falls from
136 s to 43 s, with the same terminal verdict. The replay, not a rebuild,
is the remaining cost.

## The review ledger is read once per operation (D14) — 2026-11-16

Spec `pose-review-ledger-is-read-once-per-operation`. After D13, the first
closeout call of an MCP server on harne8 still took 140 to 200 s. Its CPU
profile put about 98 s in verifying the review bundles of federated
dependencies. Counted in one call, pose-dist's review ledger (318 runs,
80 MB of transcripts) was read 280 times, its attestations 304 times, and
transcript digests were checked 29,680 times, because every bundle
verification read and checked the whole ledger again.

Within one operation the read view already returns the same bytes for the
same read. A value derived only from those reads is therefore the same on
every request of the operation. The operation now keeps, for its own life,
each project's parsed review runs, its review attestations per bundle filter,
each review bundle's header, and each transcript check per path and expected
digest. The first
computation reads through the view, so the closure and the final audit cover
every input as before. Callers receive copies of the kept slices. A store
without an operation reads and checks on every call. Nothing is kept across
operations.

Measured on harne8: the first call's CPU falls from 130 s to 53 s, and its
wall time from about 197 s to 136 s. The rest is Git: 1,116
`git log -1 -- <path>` commands, 81 s. A commit-graph with changed-path Bloom
filters takes each from 47 ms to 2 ms and the first call to 59 s; it is
repository maintenance, not part of this decision.

## A federated dependency is proved once per operation (D15) — 2026-11-17

Spec `pose-federated-source-proof-is-computed-once-per-operation`. After D14
the first closeout call on harne8 still spent about 37 s of CPU in
`federatedSourceProof`: 64 proofs in one call, 39 of them distinct, because
several roadmaps consume the same pose-dist dependency. The proof reads
through the operation's view, so within the operation it is a function of
those reads; it is now kept per source root, identity and pinned revision,
for each producer, by the same per-operation memo as the review ledger, and
callers receive deep copies. The first call fell from about 80 s to 60 s.


## Accepted amendment — 2026-11-18: observation capacity for federated closeout

Set the shared capture and audit limit to 50,000 observations, including
retained directory entries. Retain the 16 MiB metadata, 512 MiB captured
bytes and 8 GiB streamed bytes limits. An observation over any bound leaves
coverage unknown. Root admission, retry and guarded write rules still apply.

The Harne8 corpus at `dacd8e8e`, with the engine pinned to `903b1b10`, needs
41,128 to 41,208 observations across its three outstanding closeout plans.
The hooks plan reaches 40,000 during federated material verification after
the worktree snapshot has consumed 13,440 observations. The higher count
leaves room for the measured corpus. It can increase memory and replay work by
up to 10,000 observations per operation within the other bounds.

Keep the complete worktree comparison and federated input bindings. Scoping
out either check would remove evidence used when a plan is later applied.
The measurements are in the closeout-read-view-federated-corpus report.
