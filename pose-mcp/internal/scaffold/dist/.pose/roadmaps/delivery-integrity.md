---
slug: delivery-integrity
status: active
created_at: 2026-08-02
depends_on:
---

# Roadmap: Delivery integrity

**Outcome:** replace spec-local completion with a falsifiable chain from
declared scope to reviewed closure, observed Git change, production composition,
user-reachable surface, roadmap acceptance and verified published release.

This roadmap originated from sibling issues
[#8](https://github.com/oseiaspereira88/pose/issues/8) and
[#7](https://github.com/oseiaspereira88/pose/issues/7) and now also closes the
review and release-lifecycle gaps exposed while planning their implementation.
It keeps four delivery units because reviewed closure, provenance,
reachability and publication have different witnesses, while binding their
closeout through current review attestations, one schema-versioned
`delivery-integrity.json` graph and evidence-backed release records.

The implementation order is intentional. Reviewed closeout must exist before it
can govern the following specs. Git-verifiable artifact identity must then exist
before surface assurance can prove whether those artifacts participate in the
product. A release lifecycle then consumes the reviewed delivery without
leaving its fragments permanently unreleased. No milestone may claim the
roadmap outcome independently.

## Milestone: governed-closeout
- after:
- target_start:
- target_due:
- specs: pose-hierarchical-review-closeout

**Exit gate:** immutable, digest-bound reviews gate spec, milestone and roadmap
closure; findings force bounded remediation and re-review; continuous-closeout
exposes an objective terminal state without expanding authorization.

## Milestone: provenance-foundation
- after: governed-closeout
- target_start:
- target_due:
- specs: pose-artifact-provenance-ledger

**Exit gate:** structured artifact claims reconcile against explicit Git change
sets; reverse provenance and orphan findings are deterministic; the shared graph
schema, migration policy and architecture decision are frozen.

## Milestone: composed-delivery
- after: provenance-foundation
- target_start:
- target_due:
- specs: pose-delivery-surface-assurance

**Exit gate:** delivery targets, production entrypoints, evidence classes and
roadmap cut criteria consume the same provenance graph; green build checks can
no longer mask unreachable surfaces or uncomposed capabilities in strict mode.

## Milestone: release-closure
- after: composed-delivery
- target_start:
- target_due:
- specs: pose-release-lifecycle-closure

**Exit gate:** a prepared release consumes selected fragments into an immutable
snapshot; tag, publication and verification remain distinct evidence-backed
states; the next development cycle starts without prior work remaining in the
unreleased queue.

## Cut criteria

- C1: All four specs are `done`, each requirement has current trace evidence,
  current scope-appropriate review approval, and strict spec lint passing.
- C2: `pose check --strict`, the full Go suite, embedded-scaffold parity and
  `pose validate --strict --module pose-mcp --report` pass on the final tree.
- C3: A real Git fixture rejects a false artifact claim, an undeclared changed
  path, an unsafe revision and an ambiguous legacy attribution.
- C4: A composed-delivery fixture keeps artifact and build checks green while
  rejecting a tested-but-unreachable UI surface and a service absent from the
  production composition root.
- C5: The corrected fixture exposes an explainable path from spec to artifact to
  capability or surface to entrypoint to fresh check result, and both
  `surface-check` and `roadmap-check --strict` pass.
- C6: Legacy specs and roadmaps remain readable, migration is observable before
  enforcement, and dry-run backfill performs no mutation.
- C7: CLI JSON and project-scoped MCP projections match golden schemas and do
  not expose source contents, secrets or unrestricted absolute paths.
- C8: A review fixture keeps a spec, milestone and roadmap open when approval is
  missing, rejected or stale, then reaches terminal success only after bounded
  remediation, revalidation and fresh hierarchical review.
- C9: A release fixture progresses from unreleased fragments through prepared,
  tagged, published and verified states using immutable manifests and evidence;
  new fragments cannot alter prior notes.
- C10: Historical backfill reports the real `v0.9.0` directory and later tags
  without fabricating manifests, notes or publication proof, and the release
  script contains no broad staging, tag overwrite or force-push path.

## Architectural boundaries

- Keep declaration, observation and verification as distinct graph facts.
- Keep deterministic validation and technical review as distinct closeout
  evidence.
- Generate one combined index; expose focused artifact and surface views from
  that index.
- Use Git as the witness for repository changes and registered checks as the
  witness for composition and reachability.
- Reject raw executable text in specs and roadmap criteria.
- Keep operational ownership in module metadata and historical provenance in
  the delivery graph.
- Treat follow-ups as debt tracking, never as primary gap detection.
- Treat unreleased fragments as a pending queue, tags as source identities and
  provider publication/verification as separate evidence-backed facts.

## Rollout

1. Ship parsers, projections and findings in observability mode.
2. Enable artifact and delivery gates per governed root and new spec.
3. Backfill or explicitly baseline legacy history with dry-run review.
4. Activate roadmap cut enforcement after its member specs use the new
   contract.
5. Make the contract the scaffold default in a repository schema bump while
   preserving warnings for legacy completed work.

## Risk controls

- Bind validation freshness to the observed change-set digest.
- Bind review freshness to canonical spec, milestone and roadmap scope digests.
- Fail closed on ambiguous attribution, unsafe revision or path escape.
- Scope orphan detection with reviewed roots and exact exclusions.
- Require reference fixtures for UI and non-UI composition without embedding
  framework logic in the engine.
- Keep this roadmap `active` until every cut criterion has deterministic
  evidence; completed spec counts alone are insufficient.
- Keep release publication state honest when provider evidence is missing or a
  tag-triggered workflow fails.
