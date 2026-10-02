---
slug: pose-delivery-integrity-index-compaction
status: draft
created_at: 2026-10-02
completed_at:
delivers: capability:delivery-integrity-index-compaction
components: pose-mcp
task_type: refactor
priority: 1
---

# Spec: Keep the delivery-integrity index linear in the work it describes

## 1. Intent

### Goal

Stop `.pose/indexes/delivery-integrity.json` from growing with the product of
deliveries and validation results, without changing what any gate concludes.

### Business value

The index went from 3.2 MB at v5.0.8 to 9.1 MB at v6.2.0 (compact JSON; the file
on disk is about 20% larger because it is indented) while specs grew from 209 to
283. Every regeneration produces diffs of tens of thousands of lines, every
release adds more, and the file is parsed and sealed on each `check`, `index` and
`validate`. Review noise and cost rise faster than delivery does.

### Measurements this spec is based on

Taken on 2026-10-02 from the v6.2.0 index and the working tree, not from the
code alone.

Three structures grow as a product instead of as a sum.

1. `validated-by` edges: 19,472 of 35,455 edges, about 4.6 MB. Each delivery
   target is linked to every passing result whose module and evidence class
   match, and the same edge is written a second time for the entrypoint node
   (`delivery_surface.go:447-448`). The entrypoint node is shared by many
   targets (14 entrypoints for 276 targets), so the same entrypoint edge is
   appended once per target: 19,472 entries hold only 9,866 distinct edges. In
   the whole edge list 9,638 of 35,455 entries (27%) are exact duplicates,
   9,606 of them `validated-by`. The remaining 9,274 target-to-result edges
   are genuine pairs, but of the 46 results, 42 link to the same 215 targets
   and 4 to the same 61; there are two distinct result sets.
2. `scope_provenance` inside each `validation_results` entry: 53 results, each
   carrying the same 247-entry spec-to-digest map, about 1.35 MB. The map is
   computed once per run (`validate.go:536-544`) and copied into every result
   (`surface_check.go:78`). All 53 copies are identical.
3. `paths`: 9,274 of 13,565 entries are `validation-result:` references repeated
   for every delivery path, which duplicates the edges of item 1.

`nodes`, `claims`, `change_sets` and `reverse` grow with specs and commits and
are not touched here. The graph is a public compatibility contract (ADR
2026-08-02), so the change keeps every conclusion derivable and adds a schema
version instead of reinterpreting fields in place.

### Constraints

The provenance digest (claims and change sets) and each scoped digest must stay
byte-identical, because sealed review bundles and `deliveryEvidenceCurrent`
depend on them. The graph `input_digest` is recomputed over the whole graph and is
not consumed by bundles, so it may change. No change to `pose-mcp/internal/cli`:
that root is a governed surface and the writers there already marshal the graph.

### Non-goals

Untracking the index or regenerating it only in CI; changing which results count
as current evidence; changing `nodes`, `claims`, `change_sets` or `reverse`; the
indentation style of the written JSON; the equivalent growth in
`.pose/results/delivery-validation.json`, which stores the run-level map once.

## 2. Requirements

### Functional

- R1: The index shall record each validation run's `scope_provenance` once, and
  results shall reference their run; reading the index shall give every result the
  same `ScopeProvenance` it had before.
- R2: The index shall record the set of passing results that validates a delivery
  as one shared `validation-set` node per distinct set of results, linked from each
  delivery target with a single `validated-by` edge and to its member results with
  `contains` edges, instead of one edge per delivery and result.
- R3: The edge list shall contain no duplicate edge. An entrypoint node shared by
  several targets shall carry each of its `validated-by` links once.
- R4: A delivery path shall reference its validation set once instead of listing
  every result.
- R5: The expanded form, rebuilt from the compact one, shall contain the same
  delivery-to-result pairs, the same findings and the same `ScopeProvenance` per
  result as the form produced by the current writer on the same inputs.
- R6: Index size shall be linear in deliveries and results: adding one delivery
  target that passes the same results shall add a constant number of edges, and
  adding one passing result shall add one `contains` edge per distinct set that
  includes it.
- R7: The provenance digest and every scoped digest computed on this repository
  shall be identical before and after the change.
- R8: A reader shall accept schema 1 and schema 2 indexes. A schema 1 index found
  on disk shall be rebuilt, never reinterpreted, and an engine that does not know
  schema 2 shall rebuild instead of trusting it.

### Non-functional

- On this repository, the compact index shall be at most half the size of the
  schema 1 index regenerated from the same inputs, measured in bytes of the
  written file.
- Building the graph shall not take measurably longer: compare `pose index` wall
  time on this repository before and after, and report both.

### Security

The index continues to hold only repository paths, digests and identifiers. No
result output, environment value or absolute path may be added to make the
sets self-describing.

### Compatibility

`pose surface-check`, `pose review bundle --seal`, `pose closeout-check`,
`pose artifact-check`, `pose check --strict`, the MCP delivery-integrity and
surface-assurance tools and the roadmap criterion gates shall produce the same
verdicts on this repository's current tree. Sealed bundles that are fresh before
the change remain fresh after it.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/pose`: graph construction in `delivery_surface.go`, the graph
type and schema constant in `delivery_integrity.go`, and the cache reader in
`delivery_integrity_cache.go`. The writers in `cli/index.go` and
`cli/artifact_integrity.go` call the graph's JSON encoding and stay unchanged. The
MCP tools read the index through the same types.

### Artifacts

- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/delivery_integrity_cache.go
- modified: pose-mcp/internal/pose/module_scope_direction_test.go
- created: pose-mcp/internal/pose/delivery_integrity_compact.go
- created: pose-mcp/internal/pose/delivery_integrity_compact_test.go
- modified: .pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
- created: .pose/specs/2026-10-02-pose-delivery-integrity-index-compaction.md
- created: .pose/changelogs/unreleased/pose-delivery-integrity-index-compaction.md

### Delivery targets

- capability:delivery-integrity-index-compaction module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

This capability changes the serialized form of an existing engine artifact. It is
exercised through the same commands that already read the index and adds no new
command, flag or MCP tool.

### API/contract changes

The index gains `schema_version: 2`, a `validation_runs` table, `validation-set`
nodes and `contains` edges. Results carry a run reference in place of their own map.
`validated-by` now points from a delivery target to a `validation-set`. Consumers
that walk the graph for `validation-result` nodes follow one more hop. The
amendment to the ADR records this and the reason the pairs are lossless.

### Data/storage changes

None outside the index file. The results file already stores the run-level map once
and is not changed.

### Technical risks

A set that differs by one stale result per delivery would defeat the sharing and
return the product shape. The current index has two sets, but the plan tests the
degenerate case, where every delivery has a distinct set, to confirm size is still
bounded by deliveries times set size and to name that bound in the ADR.

Sharing relies on set identity being a stable digest of sorted member IDs. A
reordering bug would change the node ID between runs and put noise back into diffs.

## 4. Tasks

### Planning

- [ ] Confirm the measurements above against a fresh `pose index` on v6.2.0.
- [ ] Decide the exact `validation-set` ID format and the `validation_runs` shape.

### Implementation

- [ ] Compute sets from the existing per-target result loop, preserving the
      currency check per target and spec.
- [ ] Write schema 2 and read schema 1 and 2.
- [ ] Amend the ADR and add the changelog fragment.

### Validation

- [ ] Equivalence test against a graph built with the schema 1 writer.
- [ ] Digest-identity test against this repository's change sets and claims.
- [ ] Growth test with synthetic deliveries and results.
- [ ] Run the full canonical matrix and compare sealed-bundle freshness.

## 5. Decisions

### Decision 1

- Date: 2026-10-02
- Context: the product shape comes from linking every delivery to every matching
  result. The links are exact, since currency is checked per target and spec, so a
  coarser edge such as delivery to module and class would lose that.
- Options considered: remove only the duplicate edges, which needs no schema
  change and removes 27% of the edges but leaves one edge per target and result;
  link deliveries to module and class groups, which is linear but changes what the
  edge asserts; share one node per distinct result set, which is exact and linear
  when sets repeat.
- Decision: share one `validation-set` node per distinct result set, and write
  every edge once. Deduplication does not depend on the schema bump and lands
  first inside the same scope, so it is measured on its own.
- Rationale: it is the only option that is exact and keeps growth bounded by what
  differs between deliveries. The index shows two distinct sets for 276 delivery
  targets.
- Consequences: graph consumers follow one more hop. Worst case, every delivery has
  a distinct set, grows as before; the ADR states that bound.
- Falsifier: a repository whose deliveries each have a distinct set after the change
  would show no size reduction, and the growth test would fail on its own fixture.

## 6. Validation

### Strategy

Build the graph with the current writer and with the compact writer on the same
fixture, expand the compact form and compare delivery-result pairs, findings and
per-result `ScopeProvenance`. Compute the provenance digest and every scoped digest
on this repository before and after. Regenerate the index here and measure bytes.
Seal a bundle on a spec before the change, apply it, and check the bundle stays
fresh.

### Deterministic checks

- Command: go -C pose-mcp test ./internal/pose ./internal/mcpserver -run 'DeliveryIntegrity|DeliverySurface|ModuleScope' -count=1
- Scope: graph construction, cache reader and MCP readers
- Expected: all schema 1 and schema 2 cases pass

### Execution log

Not started. The measurements in section 1 were taken on 2026-10-02 from the v6.2.0
index and the working tree.

### Requirement trace

### Known gaps

Untracking the index and regenerating it only in CI is not decided here.

## 7. Final Report

### Delivered scope

Pending.

### Follow-ups

Pending.
