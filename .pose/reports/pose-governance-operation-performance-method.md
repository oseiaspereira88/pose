# Governance performance qualification method

Spec: `pose-governance-operation-performance`. Measurements recorded on
2026-11-03; structured observations and digests are in
`../results/pose-governance-operation-performance.json`.

## Scope and interpretation

The native corpus is commit `fdefae55b4fd6a501f4abbc6b4b20dd2beed198a`, with
the complete source ref inventory captured by the harness. The published binary
was installed from `4a3ccbb6`; the intervening corpus-evidence delivery changed
tests/governance, not that binary's runtime implementation. Each report pins
its actual binary digest, version output, corpus revision, refs, matrix,
platform, environment and command arguments.

These are observations on an interactive shared machine. Development and local
validation occurred during the measurement period. There is no claim of a
quiescent lab, causal speedup percentage, human-effort saving, physical I/O
share or general latency SLO. The three original `check --strict` runs hit the
600-second bound: their durations are retained as invalid timeout samples and
have no median. They are not equivalent to completed checks taking 600 seconds.

## Native command measurements

Use the current harness from the implementation checkout, passing the same
corpus root for both binaries. All output directories must be outside that
root. For example:

```sh
scripts/bench-governance.sh 3 /path/to/published-pose --root /path/to/fixed-corpus --timeout 600 --output-dir /external/published-results
scripts/bench-governance.sh 3 /path/to/candidate-pose --root /path/to/fixed-corpus --timeout 600 --output-dir /external/candidate-results
```

The harness mirrors every ref, dissociates external object alternates, and
creates a fresh detached worktree for each sample. It disables global/system
Git configuration and setup hooks/fsmonitor, and disables optional index
refresh locks. It copies/pins the selected binary. It never resets/restores an
operator checkout. Uncommitted operator files are recorded as dirty input
metadata but are excluded from the committed measurement corpus.

Unix `wait4` measures wall duration separately from user/system CPU and process
resource usage. `ru_maxrss` is reported in KiB on Linux, with platform-native
units identified elsewhere. Reaped descendants may contribute to resource
usage; this is not a complete profile of detached/unreaped background work.
Setup is measured and recorded separately, excluded from command durations.
Timeouts kill the sample's process group. Raw version/setup/command stdout and
stderr are retained; invalid observations are never included in medians.

The `index` samples also retain the eight generated index/descriptor files with
byte sizes and hashes. Capture opens every corpus directory component and file
with `O_NOFOLLOW`, accepts regular files only, and writes through an anchored
destination directory with exclusive creation. Capture errors invalidate the
sample. Symlink/file/parent, directory and FIFO controls exercise this boundary.

Three repetitions expose sample range and sample standard deviation. A later
one-repetition run captured generated files for semantic comparison; it is not
silently added to the earlier three-sample median. All eight files, including
generation and delivery integrity, were byte-identical between the published
binary and `ab8e2bea` on the same corpus. That comparison does not by itself
qualify a subsequent implementation change; later code still receives fresh
local validation and separate review.

## CPU and heap diagnostic

The partial diagnostic used an archived copy of `ab8e2bea`, leaving the active
and operator checkouts unchanged. A temporary `cmd/pose-profile` main called
the real `cli.Main`; it started `runtime/pprof.StartCPUProfile`, then at 45
seconds stopped the CPU profile, wrote a heap profile without forcing GC, and
recorded `runtime.ReadMemStats`. The CLI continued until an external 65-second
process-group deadline. Its eventual `-9` exit is explicitly an interrupted
diagnostic, not a successful command/verdict or native latency sample.

The temporary main's essential measurement sequence is:

```go
// Build inside an archived pose-mcp module so internal/cli is the real code.
pprof.StartCPUProfile(cpuFile)
var once sync.Once
finish := func() { once.Do(func() {
    pprof.StopCPUProfile()
    pprof.WriteHeapProfile(heapFile) // no runtime.GC call
    var stats runtime.MemStats
    runtime.ReadMemStats(&stats)
    // Persist stats, runtime.Version(), window and gc_forced:false.
}) }
timer := time.AfterFunc(45*time.Second, finish)
code := cli.Main(args, os.Stdout, os.Stderr)
timer.Stop()
finish()
os.Exit(code)
```

The archival source, helper and runner were retained under the local diagnostic
paths recorded in the structured result. Reproduction must create a fresh
archive/helper and record its own binary/profile hashes; profile bytes and
binary hashes are not promised to be deterministic across build paths or runs.

```sh
go -C /external/archive/pose-mcp build -buildvcs=false -o /external/pose-profile ./cmd/pose-profile
go tool pprof -top -cum -nodecount=25 /external/diagnostic/cpu.pprof
go tool pprof -top -nodecount=20 /external/diagnostic/cpu.pprof
```

The 45-second window contained 207.49 seconds of sampled CPU across goroutines.
`ListSpecs` accumulated 68.25 seconds (32.89% of those samples), largely inside
`GetSpec`; these nested cumulative percentages overlap and must not be added.
The runtime snapshot recorded 102,572,626,992 total allocated bytes and
533,311,136 live heap bytes, with 364 GCs. These are different quantities, not
RSS or physical disk I/O. The window preceded spec-metadata reuse and motivated
that next optimization; it does not describe every stage of the full command.

## Producer fixtures and function-entry instrumentation

The small fixtures run actual readers, real Git commands and native CLI/MCP
entrypoints. They do not replace a completed native command benchmark.

```sh
go -C pose-mcp test ./internal/pose -run '^TestReviewOperationViewSpecCatalogueReuseAndBodiesStayLive$' -v -covermode=atomic -coverprofile /external/spec.cover -count=1
go -C pose-mcp test ./internal/pose -run '^TestReviewOperationViewCapturesEachHeaderOnce$' -v -covermode=atomic -coverprofile /external/headers.cover -count=1
go -C pose-mcp test ./internal/cli -run '^TestGovernanceOperationPerformanceCapturesTrailersOnce$' -v -count=1
```

For each named function, its first executable coverage block supplies an
actual entry count; the profile is atomic so concurrent increments are not
lost. The result records source locations and profile digests. In the six-spec
fixture: 12 successful body queries plus one missing-slug query, 36 spec-file
parses, 24 `cloneSpec` calls and three canonical catalogue captures (including
final audits). `ReadFile` counters recorded 36 calls and 3,204 logical returned
bytes. Metadata lookup does not cache bodies or remove final input checks.

For six unrelated valid bundle records queried four times, there were six full
header classifications and 12 `ReadFile` calls including the final byte audit,
returning 504 logical bytes. These bytes are API-returned bytes, not physical
I/O, and do not pretend to cover every `Open`/`Read` path.

The Git fixture wrapper records each real Git launch and its generated stdout
before forwarding the exact bytes/exit status. It runs only after fixture
setup, outside measured native timings. For 1/3/6 specs it recorded 11/23/41 Git
launches and one complete trailer log per collection. Root-parent resolution
has expected nonzero exits followed by the raw root-object proof; generated
stdout counts include those attempts. Stderr and physical I/O are not inferred.

`cloneSpec` entries count semantic copying calls, not all runtime memory copies.
The CPU profile separately observes runtime copying/GC work. No I/O or CPU
percentage is calculated from the fixture byte counts.

## Exact clustering and output serialization

```sh
go -C pose-mcp test ./internal/cli -run '^TestGovernanceOperationPerformanceClustering$' -bench '^BenchmarkGovernanceOperationPerformanceClustering$' -benchtime=1x -count=3
go -C pose-mcp test ./internal/cli -run '^TestGovernanceOperationPerformanceClustering$' -bench '^BenchmarkGovernanceOperationPerformanceGraphOutput$' -benchtime=1x -count=3
```

Clustering compares prepared inputs to the independent pre-change rule, keeping
normalization, similarity, ordering and candidates exact. Measurements vary
both entry count (8/32) and text length (32/128). Report time, allocated bytes
and allocation count separately. Above the one-MiB input-preparation budget,
the complete original algorithm remains available; an adversarial fixture
forces that fallback and still compares real candidates at five thresholds.

Output measurements call the actual `DeliveryIntegrityGraph.IndexJSON` for
16/128 claims and record logical JSON byte count, nanoseconds, allocated bytes
and allocations. Throughput/bytes here are serialization observations, not
storage/network I/O or an estimate of human effort.

## Remaining cost

The observations justify reducing repeated discovery and unrelated-header
classification. They do not make review verification free: subject hashing,
strict selected-record loads, archived-fragment discovery, trust and recursive
verification remain live. Persistent views/caches and optional runtime work
belong to their existing B33/B35 scopes; no new daemon or global authority cache
was introduced by this spec.
