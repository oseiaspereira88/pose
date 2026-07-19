# ADR: MCP project-scope resolution and structured selection errors

## Status
Accepted (2026-07-19) — spec `pose-mcp-project-scope-contract`

## Context

Only 11 of the 20 `pose_*` tools advertised `project_id` in their schema
(`pose_check`, `pose_lint_spec`, `pose_suggest`, `pose_get_followups`,
`pose_get_knowledge`, `pose_get_rules`, `pose_get_skill`,
`pose_get_workflow`, `pose_list_knowledge` did not) even though `dispatch()`
already resolved every tool's store from an optional `project_id`
generically — a client following the published schema for those nine tools
would not know the argument existed. Resolution failures (`StoreFor`) were
untyped `fmt.Errorf` strings: "unknown project_id" and "no default project
root configured" were indistinguishable to a caller except by parsing
prose, and nothing separated "will never resolve" from "resolves once you
disambiguate." As POSE moves toward multi-repository operation, silent
implicit defaulting under multiple registered projects becomes a real
misrouting risk the roadmap's risk controls call out explicitly ("fail
closed on ambiguous project selection").

Alternatives considered:

1. **Leave resolution errors as opaque strings, document by convention** —
   already failed; nine tools drifted from the documented contract with no
   test catching it.
2. **Immediately fail closed on any implicit default under multi-project** —
   breaks the compatibility requirement (announced deprecation window) for
   deployments only just adopting multi-project.
3. **Typed errors + universal schema + opt-in strict mode.**

## Decision

Option 3:

- **Uniform schema (R1):** every `pose_*` tool (all 20; conductor tools
  excluded — they never touch a POSE store) declares the identical
  `project_id` property — same type, same description — pinned by
  `TestProjectIDSchemaConsistencyAcrossCatalog`, which also asserts
  `project_id` is never `required` (a default is convenience, not a
  mandate).
- **Typed resolution errors (R2):** `pose.ProjectUnknownError{ProjectID}`
  and `pose.ProjectAmbiguousError{Reason: "no-default"|"multi-project-implicit"}`
  replace the untyped `fmt.Errorf` calls in `Roots.StoreFor`. `callToolCtx`
  detects them via `errors.As` and returns `structuredContent.error_code`
  (`project_unknown` / `project_ambiguous`) alongside the existing
  human-readable text — additive to the tool-error contract, not a
  transport-level change. Authorization denial keeps its existing distinct
  path (JSON-RPC `-32004` with `decision.Metadata()`); the two mechanisms
  compose rather than collide.
- **No path leak (R3):** neither typed error nor its structured payload
  carries `Store.Root`; only the caller-supplied `project_id` or the
  resolution `reason` appears. Audit records already carried the logical
  `project_id` (never the resolved path) for every decision — unchanged.
- **Compatibility / deprecation window:** `RootsConfig.StrictSelection`
  (env `POSE_MCP_STRICT_PROJECT_SELECTION`) defaults to false. In that
  default mode, an empty `project_id` still resolves to `DefaultRoot`
  unconditionally — byte-identical to prior behavior — even once more than
  one project is registered. Setting the flag makes that same omission
  return `project_ambiguous`/`multi-project-implicit` whenever
  `len(registered projects) > 1`; a single-project deployment is
  unaffected either way, so stdio ergonomics are provably unchanged
  (`TestRoots_CompatModeKeepsImplicitDefaultUnderMultiProject` and the
  paired strict-mode test pin both directions).

## Consequences

- Positive: every client can discover and pass `project_id` on any tool,
  and can programmatically branch on `project_unknown` vs
  `project_ambiguous` vs policy-denied instead of pattern-matching prose.
- Positive: multi-project operators get an explicit, reversible knob to
  fail closed today, with single-project deployments provably immune to
  ever tripping it.
- Trade-off: the strict flag is opt-in, so the misrouting risk it prevents
  remains live by default until an operator adopts it — documented as the
  announced deprecation window the compatibility requirement calls for.
- Residual: authorization-level "unauthorized project" reuses the existing
  policy-deny path rather than a fourth `structuredContent.error_code`;
  revisit only if a client-side need for a unified error vocabulary across
  all three failure classes emerges.
