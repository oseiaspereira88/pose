---
slug: check-builds-the-delivery-graph-once
status: done
created_at: 2026-09-20
completed_at: 2026-09-20
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
delivers: surface:check-delivery-graph-reuse
---

# Spec: `check` builds the delivery graph once

## 1. Intent

### Goal

`pose check --strict` spent 600 of its 635 seconds in `checkDeliveryContracts`,
because `deliverySpecBlockers` rebuilt the entire delivery graph for every done
spec. Build it once for the loop, and stop `focusSurfaceGraph` from consuming the
graph it is handed, which is what made reuse impossible.

### Business value

`check` is this repository's primary structural gate and the one a pre-commit hook,
a CI job and every closeout run. At 635 seconds it exceeded a ten-minute budget twice
in one session and was recorded as "no verdict" rather than as a pass — a gate nobody
can afford to wait for stops being a gate.

The cost was quadratic in the repository's own success: one full graph build per done
spec, 117 of them here, about five seconds each. Every spec closed makes it worse.

### Constraints

The verdict, the findings and the exit code must be identical. A graph that cannot be
built must still be reported for every spec the loop would have checked, because that
error belonged to all of them.

### Non-goals

`checkReviewCloseout`, the remaining 34 seconds. It calls `GetCloseoutState` per spec
at about 0.25 seconds each, which is linear and not the reason the gate timed out.
Touching it here would mix a measured fix with an unmeasured one.

## 2. Requirements

- R1: `checkDeliveryContracts` builds the delivery graph once for the whole loop.
- R2: `focusSurfaceGraph` does not consume the graph it receives: the caller's
  deliveries and findings are unchanged, and focusing twice on one graph yields each
  spec's own result.
- R3: The verdict, the findings and the exit code are unchanged, compared by diffing
  the full output before and after.
- R4: An unbuildable graph is still reported once per spec in the loop.

## 3. Technical Plan

`focusSurfaceGraph` filtered into `graph.Deliveries[:0]` and `graph.Findings[:0]`.
The graph arrives by value, which reads as a copy and is not one for a slice, so
those writes went through the caller's backing array: focusing truncated the graph
the caller still held, and a second focus saw the residue of the first. Allocate
instead, add `deliverySpecBlockersFromGraph` for a caller that already has a graph,
and hoist the build in `checkDeliveryContracts`.

### Artifacts

- created: .pose/specs/2026-09-20-check-builds-the-delivery-graph-once.md
- created: .pose/changelogs/unreleased/check-builds-the-delivery-graph-once.md
- created: pose-mcp/internal/cli/delivery_graph_reuse_test.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/check.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- surface:check-delivery-graph-reuse module:pose-mcp/internal/cli profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

One allocation, one new entry point and one hoisted call. Reverting restores the
rebuild; the aliasing test would then fail, which is what keeps the two halves
together — the hoist is only safe because the filter no longer consumes its input.

## 4. Tasks

- [x] Localize the cost by measuring each checker phase rather than reading for it.
- [x] Pin the aliasing that made reuse unsafe, before reusing anything.
- [x] Build once, and prove the output is byte-identical.

## 5. Decisions

The phases were instrumented with temporary timing prints and the binary rebuilt,
rather than reasoned about from the source. The first hypothesis was
`checkReviewCloseout`, because it loops over specs calling `GetCloseoutState`; timing
one call at 0.25 seconds put it at about 29 seconds and ruled it out. Only the
instrumented run named `checkDeliveryContracts` at 600.89 seconds.

The aliasing is fixed rather than worked around with a deep copy at the call site.
A function that takes a struct by value and writes through its slices is a trap for
the next caller too, and the copy would have left the trap in place.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Filter does not consume its input | `go test ./internal/cli -run FocusSurfaceGraph -count=1` | Caller's slices unchanged; two focuses on one graph each see their own spec |
| Shared graph gives the same blockers | `go test ./internal/cli -run DeliverySpecBlockersFromShared -count=1` | Same blockers for repeated and interleaved slugs |
| Output unchanged | `pose check --strict` before and after, sorted and diffed | Zero diff lines; same 11 warnings; same exit code |
| Registered producer | `pose validate --strict --module pose-mcp` | The dedicated check runs the reuse corpus |

### Execution log

2026-09-20. Measured, in this order.

The corpus first: 528 review bundles, 515 attestations, 225 specs, 1077 markdown
files. Then each checker phase, with temporary timing prints compiled into a
scratch binary:

| Phase | Time |
| --- | --- |
| `checkDeliveryContracts` | 600.89s |
| `checkReviewCloseout` | 33.77s |
| everything else, summed | 0.28s |
| total | 634.9s |

`deliverySpecBlockers(root, slug)` calls `buildCurrentDeliveryGraph(root)`, whose only
argument is the root: the graph never depended on the slug. One build measured at
5.43 seconds through the CLI, and 117 done specs at that cost projects 636 seconds,
which is the 600.89 the instrumented run recorded.

Two hypotheses were wrong before that and are recorded because each was plausible:
`checkReviewCloseout` looked quadratic in bundles and is linear in specs at 0.25s
each; and my first attempt to time the whole run used `/usr/bin/time`, which does not
exist here, so it produced exit 127 and no measurement at all.

After the fix: 39.0 seconds, exit 0, verdict
`SUCCESS — (strict mode) with 11 warning(s)`. The full outputs before and after were
sorted and diffed: zero differing lines, same eleven warnings. 16x, with the same
result rather than a faster different one.

Defect injection: restoring `graph.Deliveries[:0]` and `graph.Findings[:0]` fails both
reuse tests, one reporting that the first focus was rewritten by the second and the
other that the shared graph yielded no blockers at all.

### Closeout

2026-09-20 UTC. Bundle `rvb-a5019a214fa3ddd1`, twelve evidence items, under the five
sealed contracts. Attestation `rva-fe1db85d60c221b7`, `agent:claude-opus-5`,
approved; `review-check` fresh and approved; `closeout-check` terminal.
`surface-check --strict` exits 0 with one inferred-coverage warning kept.

### Requirement trace

- R1 [satisfied] surface:check-delivery-graph-reuse evidence:integration
  check:delivery-graph-reuse-integration test:TestDeliverySpecBlockersFromSharedGraph
  — the loop takes a prebuilt graph through `deliverySpecBlockersFromGraph`
- R2 [satisfied] surface:check-delivery-graph-reuse evidence:integration
  check:delivery-graph-reuse-integration test:TestFocusSurfaceGraphDoesNotConsumeItsInput
  — the caller's slices are unchanged and two focuses are independent
- R3 [satisfied] surface:check-delivery-graph-reuse evidence:integration
  check:delivery-graph-reuse-integration — `check --strict` output sorted and diffed
  before and after: zero diff lines, eleven warnings both times, exit 0 both times
- R4 [satisfied] surface:check-delivery-graph-reuse evidence:integration
  check:delivery-graph-reuse-integration — the build error is carried into the loop
  and reported per spec, so an unreadable graph cannot present as a clean run

## 7. Final Report

### Scope delivered

`check --strict` on this repository goes from 635 seconds to 39, with byte-identical
findings. The gate is affordable again, and the aliasing that forced the rebuild is
gone rather than avoided.

### Residual risks

`checkReviewCloseout` is now the largest phase at 34 seconds and is linear in done
specs, so it will grow. It is left alone deliberately: it was measured and is not
what made the gate time out. The graph build itself is still 5.4 seconds and was not
profiled further — the fix removed 116 of 117 calls to it rather than making it
faster.

### Follow-ups

- [covered] Profile the single graph build and the 34s closeout phase — profiled and
  reduced by parse-the-delivery-index-once-per-content, which removed two repetitions
  of identical work (owner:@pose-maintainers crit:low review:2026-11-20)
