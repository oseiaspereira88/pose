# ADR: Qualified artifact authority and explicit spec transfer

## Status
Accepted — 2026-09-21. The user authorized implementation of this sequence.
Runtime adoption in consumer projects remains a separate, evidence-gated action.
Implemented by [resolution](../specs/2026-09-21-pose-qualified-artifact-resolution.md),
[transfer](../specs/2026-09-21-pose-spec-authority-transfer.md) and
[agent context](../specs/2026-09-21-pose-agent-project-context.md).

## Context
A coordinator and executor can contain the same slug with different requirements
and lifecycle states. Store and Git attribution are project-local. Directory
ancestry is not authority; slug equality does not prove duplicate intent.
Extend the accepted [authorization ADR](2026-07-19-cross-repo-portfolio-reuses-mcp-project-authorization.md)
and [selection ADR](2026-07-19-mcp-project-scope-resolution-and-structured-selection-errors.md).
Keep their authorization boundary; do not create another project registry.

## Decision
- Identify artifacts by stable project ID, kind and local slug. Revision and
  digest identify evidence, not the artifact. Bind explicit IDs through existing
  project configuration; directory rename must not change adopted identity.
- Keep local refs and legacy `xref:<project_id>/<slug>` (spec) compatible.
  Proposed typed forms are `xref:<project_id>/spec:<slug>`,
  `xref:<project_id>/roadmap:<slug>` and
  `xref:<project_id>/milestone:<roadmap>/<milestone>`. These are design syntax,
  not inputs supported by the installed engine. Negotiate adoption before use.
- Share parsing/resolution across CLI, MCP, checks, readiness and projection.
  Use canonical Store readers and existing authorized roots. Never fall back
  from failed external resolution to a same-slug local spec.
- Give each executable spec one authority. Redirects/projections own no second
  lifecycle, requirements, review or delivery claim. Separate transfer from
  decomposition into independently named scopes with requirement mappings.
- Implement explicit transfer as an idempotent journaled operation: immutable
  plan digest, operation ID and expected revisions in both projects. Prepare the
  destination as non-executable, retire the source with a handoff receipt, then
  activate the destination after verifying that receipt. Interrupted transfer
  blocks execution of that scope. Do not promise a distributed Git transaction.
- Authorize writes in each project separately. Preserve source documents,
  amendments, commits and bundles in history; migration never rebinds an old
  attestation to a different subject. Support binding an existing executor only
  after explicit requirement reconciliation, preserving its approved evidence.
- Keep `POSE-Spec:` repository-scoped. A consumer gitlink update proves adoption
  of a revision, not ownership of the child's implementation commits.

Rejected: editable mirrors (permanent drift), implicit ancestry (cwd changes
meaning), global slug uniqueness (rejects unrelated projects), global database
(moves authority), symlinks (portability and confinement failures).

## Consequences
Add schemas/capability negotiation for transfer and qualified references.
Unsupported adopted metadata must fail explicitly. Preserve single-project
behavior and offline operation with authorized local revisions/evidence. Different
projects may legitimately reuse a slug; warn on declared competing authority,
not similarity alone. A missing project never authorizes fallback or scanning.

## Implementation amendment — 2026-09-21

Reuse the existing explicit project-root map as the selected checkout. Reject
conflicting implicit bindings and duplicate JSON keys instead of silently choosing
one. Directory-derived IDs remain legacy-only; adoption binds stable IDs explicitly.

Negotiate required reference metadata with review policy schema 3 and
`qualified_artifact_refs_version: 1`, explicitly adopted by the consumer. Inspection
showed that the current schema-2 decoder ignores unknown fields, so an additive
field alone cannot make an older binary refuse the contract. Preserve schema 1/2
and historical bundles; do not automatically stamp schema 3 during `pose update`.

## Compatibility amendment — 2026-09-24

Cross-project spec creation uses a distinct `new-spec-qualified` CLI verb with
mandatory `--task xref:<project>/spec:<slug>` and `--expect-context <digest>`.
The pre-contract `new-spec` parser ignores unknown flags and can create a local
shadow when given `--task`; it cannot be changed after distribution. The new
verb makes that binary reject the operation before writing, while the current
engine reuses its existing authority resolver and freshness checks. Consumers
must keep old bindings inactive until they install the compatible engine.
