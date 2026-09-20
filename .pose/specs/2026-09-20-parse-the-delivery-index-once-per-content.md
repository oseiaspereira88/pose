---
slug: parse-the-delivery-index-once-per-content
status: in-progress
created_at: 2026-09-20
completed_at:
supersedes:
depends_on: check-builds-the-delivery-graph-once
priority: 1
components: pose-mcp
task_type: refactor
delivers: capability:content-keyed-parse-memos, surface:parallel-gate-findings
---

# Spec: stop repeating gate work, then do the rest at the same time

## 1. Intent

### Goal

Close the follow-up `check-builds-the-delivery-graph-once` left open: profile what
remained of that gate, remove the repeated work whose result cannot differ, and then
run the independent per-spec work concurrently.

Two memos keyed on content rather than time, and one worker pool whose findings are
replayed in item order.

### Business value

After the graph hoist the gate still took 39 seconds, and profiling it named two
repetitions of identical work. `.pose/indexes/delivery-integrity.json` is 4.5 MB in
this repository and was read and unmarshalled 528 times in one run — 2.4 GB of JSON
for one gate. `AssessDesignDelta` ran 152 times for 116 distinct inputs, and already
computed a `CacheKey` from those inputs: the field named a cache nobody had written.

### Constraints

The verdict, the findings and the exit code must not move. A memo must not be able to
serve a stale value, and a cached value must never be shared: callers of both APIs
filter what they receive in place.

### Non-goals

Restructuring how freshness is verified. After the work below, the remaining 11.7
seconds is 116 distinct bundle preparations reading Git; making that cheaper means
changing what the gate checks, which is a design decision and not an optimisation.

`ListReviewBundles`, measured at 7.77 seconds over 339 calls. Its cost is 528 file
reads at about 44 microseconds each, so the cost is the reads and not a parse: a memo
that still reads every file saves nothing, and one that skips the reads would have to
key on modification time, which cannot be made exact. Left alone deliberately rather
than optimised on a weaker safety argument than the two taken here.

The intrinsic work: 116 distinct subjects each need a real bundle preparation with Git
reads. No cache removes that, and removing scopes from the gate would weaken it.

## 2. Requirements

- R1: The delivery index is parsed once per distinct content. The cache key is the raw
  bytes, so a rewrite inside one filesystem time tick cannot serve the previous graph.
- R2: The design delta is computed once per input digest, the value it already
  published as `CacheKey`.
- R3: Neither memo hands out a shared value. A caller that mutates what it received
  cannot affect the memo or any later caller.
- R4: A field added to either cached type without being copied fails a test, so an
  omission cannot become a silently shared slice.
- R5: The verdict, findings and exit code of `check --strict` are unchanged, compared
  by diffing the full output against the pre-optimisation baseline.
- R6: The per-spec gate loops run concurrently, and their findings are emitted in item
  order regardless of which worker finishes first. Repeated runs produce
  byte-identical output.
- R7: The concurrent path is free of data races under the race detector, and the
  escalation mode is applied to the replayed messages exactly as before.

## 3. Technical Plan

`GetDeliveryIntegrity` keeps reading the file — 0.44 ms against 10.7 ms to parse it —
and compares the bytes with the cached ones. `AssessDesignDelta` returns early on a
digest hit. Both return defensive copies, and both copies are guarded by a test that
walks the struct with reflection and fails on a field it does not handle.

### Artifacts

- created: .pose/specs/2026-09-20-parse-the-delivery-index-once-per-content.md
- created: .pose/changelogs/unreleased/parse-the-delivery-index-once-per-content.md
- created: pose-mcp/internal/pose/delivery_integrity_cache.go
- created: pose-mcp/internal/pose/delivery_integrity_cache_test.go
- created: pose-mcp/internal/cli/check_parallel.go
- created: pose-mcp/internal/cli/check_parallel_test.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/design_delta.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- capability:content-keyed-parse-memos module:pose-mcp/internal/pose profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- surface:parallel-gate-findings module:pose-mcp/internal/cli profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

Both memos are process-local and additive: reverting either restores the repeated
work without changing a result. The defensive copies are what make the memos safe, and
the guard tests are what keep the copies complete.

## 4. Tasks

- [x] Profile what remained after the graph hoist, per call and per distinct input.
- [x] Measure read against parse before choosing a key.
- [x] Memoize both, return copies, and guard the copies by reflection.
- [x] Prove the output is unchanged against the pre-optimisation baseline.
- [x] Run the independent per-spec loops concurrently with ordered replay, and prove
      the order, the determinism and the absence of races.

## 5. Decisions

The key is content, never time. An mtime has filesystem-dependent granularity, and two
writes inside one tick would serve a stale graph; comparing bytes cannot be wrong. That
standard is also why `ListReviewBundles` was left alone: the only way to speed it up is
to skip reads, and skipping reads needs a time-based key.

Both memos copy on the way in and on the way out. This repository has now found the
same aliasing defect three times — `focusSurfaceGraph`, the delivery index's own path
filter, and the report returned here — and a cache is precisely what turns that defect
from a local surprise into a shared one.

The duplicate ratio was counted before the design-delta memo was written: 152 calls,
116 distinct digests, 36 repeats. A cache for a ratio nobody measured is a guess.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Content key, not time | `go test ./internal/pose -run DeliveryGraphCacheKeyedOnContent -count=1` | Same bytes reuse; changed bytes reparse within the same second; the path filter does not narrow the next caller |
| Memo is keyed and copied | `go test ./internal/pose -run DesignDeltaMemoIsKeyedAndCopied -count=1` | A caller's edit does not reach the memo; an unknown digest is a miss |
| Copies are complete | `go test ./internal/pose -run 'DeliveryGraphCopyCoversEveryField\|DesignDeltaCopyCoversEveryField' -count=1` | Mutating a copy leaves the original byte-identical; an unhandled field fails |
| Output unchanged | `pose check --strict` diffed against the baseline | One line differs, the commits-behind counter, which moved because commits were made between runs |
| Registered producer | `pose validate --strict --module pose-mcp` | The dedicated check runs the memo corpus |

### Execution log

2026-09-20. Profiled with cumulative counters compiled into a scratch binary, per
function and per call count:

| call | time | calls |
| --- | --- | --- |
| `PrepareReviewBundle` | 30.40s | 189 |
| `AssessDesignDelta` | 9.41s | 152 |
| `ListReviewBundles` | 7.77s | 339 |
| `reviewBundleSubject` | 6.90s | 189 |
| `GetDeliveryIntegrity` | 5.96s | 528 |
| `ReviewPlan` | 5.17s | 251 |
| `ScopeDigest` | 0.17s | 504 |

Times nest, so they do not sum to the run. `GetDeliveryIntegrity` at 528 calls on a
4.5 MB file was the clearest repetition, and a micro-benchmark decided the key:
reading the file costs 0.44 ms and parsing it 10.7 ms, so the parse is 96% and a memo
that still reads is worth taking.

Then the duplicate ratio for the design delta: 152 calls, 116 distinct digests, 36
repeats at about 62 ms.

Measured result, each step the minimum of two runs, verdict identical throughout:

| step | time |
| --- | --- |
| before the graph hoist | 634.9s |
| graph built once | 39.3s |
| plus the index memo | 34.5s |
| plus the design-delta memo | 31.5s |

20x against the original. The full output was diffed against the pre-optimisation
baseline: one line differs, `baseline_commit is 323 commits behind HEAD` against 326,
which moved because three commits were made between the two runs. Eleven warnings and
exit 0 both times.

2026-09-20, second increment: concurrency. The memos were worth less than hoped — 8
seconds — and the reason is that they removed repetition while the remaining work is
not repeated: 116 distinct subjects, each needing a real bundle preparation. Per-spec
work is independent, so it runs on several cores.

`failOrWarnPerItem` resolves the candidate list first, cheaply, then runs the expensive
part across `runtime.NumCPU()` workers and replays each item's findings in item order.
The ordering is not incidental: a gate whose output order depends on scheduling cannot
be diffed between runs, and the diff is exactly how every step here was proven not to
change a verdict.

It is safe because the per-item function touches no shared mutable state, which was
verified rather than assumed: `Store` carries only a root, the lookup tables in
`internal/pose` are read-only, the only mutable package state in the read path is the
two memos above and both are mutex-guarded, and `focusSurfaceGraph` no longer writes
through the graph it receives — which is what lets one prebuilt graph serve every
worker. The aliasing fix from the previous spec is what made this possible at all.

Measured, minimum of two runs each, verdict identical throughout:

| step | time |
| --- | --- |
| before the graph hoist | 634.9s |
| graph built once | 39.3s |
| plus the two content memos | 30.8s |
| plus concurrent per-spec work | 16.6s |

38x against the original. Per phase, against the first profile:

| phase | before | after |
| --- | --- | --- |
| `checkDeliveryContracts` | 600.89s | 5.78s |
| `checkReviewCloseout` | 33.77s | 11.68s |

Equivalence, proven three ways: the unsorted output of the sequential and concurrent
binaries is identical, so the order is preserved and not merely the set; three
consecutive concurrent runs hash to one value; and the race detector is clean on
`internal/cli` and `internal/pose`.

Defect injection for the ordering: replaying the buffered findings in reverse item
order fails `TestFailOrWarnPerItemKeepsItemOrder`, which skews each item's work so a
later item finishes first whenever the pool has room.

### Requirement trace

- R1 [satisfied] capability:content-keyed-parse-memos evidence:integration
  check:parse-memo-integration test:TestDeliveryGraphCacheKeyedOnContent — identical
  bytes are reused, changed bytes are reparsed within the same second
- R2 [satisfied] capability:content-keyed-parse-memos evidence:integration
  check:parse-memo-integration test:TestDesignDeltaMemoIsKeyedAndCopied — a hit is
  keyed on the digest the report already published
- R3 [satisfied] capability:content-keyed-parse-memos evidence:integration
  check:parse-memo-integration test:TestDeliveryGraphCacheKeyedOnContent
  test:TestDesignDeltaMemoIsKeyedAndCopied — a caller's mutation reaches neither the
  memo nor the next caller
- R4 [satisfied] capability:content-keyed-parse-memos evidence:integration
  check:parse-memo-integration test:TestDeliveryGraphCopyCoversEveryField
  test:TestDesignDeltaCopyCoversEveryField — both guards walk the struct and fail on a
  field they do not handle
- R5 [satisfied] capability:content-keyed-parse-memos evidence:integration
  check:parse-memo-integration — full output diffed against the baseline: one line, the
  commits-behind counter, with eleven warnings and exit 0 on both sides
- R6 [satisfied] surface:parallel-gate-findings evidence:integration
  check:parallel-gate-integration test:TestFailOrWarnPerItemKeepsItemOrder — findings
  come out in item order with the work skewed against completion order, an item with
  nothing to say does not shift the rest, and three consecutive runs of the real gate
  hash to one value
- R7 [satisfied] surface:parallel-gate-findings evidence:integration
  check:parallel-gate-integration test:TestFailOrWarnPerItemRespectsMode — the race
  detector is clean on both affected packages, and strict mode escalates the replayed
  messages exactly as before

## 7. Final Report

### Scope delivered

`check --strict` on this repository goes from 39.3 seconds to 16.6, and from 634.9
across both specs — 38x — with the findings and their order unchanged. Two repetitions
of provably identical work are gone, both memos are keyed on content so neither can
serve a stale value, and the independent per-spec work now runs on every core with its
findings replayed in item order.

### Residual risks

The remaining 16.6 seconds is intrinsic to what the gate checks: `checkReviewCloseout`
is 11.7 of it, and that is 116 distinct bundle preparations reading Git. Concurrency
returned 2.6x on sixteen cores rather than something closer to the core count, which
says the floor is process spawning and I/O rather than CPU — not profiled further,
because the next step past it is changing what the gate verifies.

`ListReviewBundles` is 7.77 seconds of file reads, named above as deliberately
untouched. The two memos are process-local and unbounded in entry count; a single
command reads a bounded set of scopes, so this is stated rather than fixed. The worker
count is `runtime.NumCPU()` with no flag: a machine where that is wrong has no way to
say so yet.

### Follow-ups

None. The follow-up from `check-builds-the-delivery-graph-once` is closed by this
spec.
