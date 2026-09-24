# Multi-repository roadmap: autonomous review (2026-09-24)

Reviewer: `agent:codex`, in a separate execution from the implementation.
Scope: the four specs in `pose-multirepo-foundation` and the Harne8 consumer
preflight. No human sign-off is required by the current review policy.

## Decisions and evidence

| Spec | Decision | Sealed bundle / attestation | State |
| --- | --- | --- | --- |
| `pose-qualified-artifact-resolution` | Approved after repairing malformed R1–R8 trace syntax | `rvb-3b84788df18e48f7` / `rva-3265b3753d044c2f` | `review verify`: closed, fresh, approved; trace committed as `e75d0ef` |
| `pose-federated-roadmap-acceptance` | Approved | `rvb-8c75fd68d389c2fe` / `rva-e7b97ef69f04d903` | `pose close` completed; review resealed after validation |
| `pose-spec-authority-transfer` | Approved | `rvb-d15645c31f806991` / `rva-99b68371d28abfe5` | `pose close` completed; review resealed after validation |
| `pose-agent-project-context` | Changes requested | `rvb-b1cf811f6f3acf6d` / `rva-b178e83cb461ba31` | R8 deferred; review not approved |

The strict `pose-mcp` matrix passed 25/25 after the MCP tests were allowed to
bind loopback sockets. All four strict spec lints pass. The targeted
`surface-check --spec pose-agent-project-context --strict` passed one target
with zero findings, but it does not exercise old installed engines. The broad
repository-wide surface check still reports legacy attribution findings and
cannot substitute for a targeted result. `pose check --strict` passes with 11
existing warnings. `assess tech-debt` found zero markers; `assess integrate`
reports 56 unobserved provider references as static inventory.
The knowledge handoff passed `knowledge-check --strict` after the five
pre-existing expired artifacts were archived by `knowledge-housekeeping`.

## Rules applied during review

- Change type: mixed Go feature, MCP contract and governance documentation.
- Workflow: `.pose/workflows/review.md`.
- `.pose/rules/backend-go.md`: concurrency, bounded reads and error handling in
  the resolver and transfer journal.
- `.pose/rules/security.md`: explicit root authorization, digest/revision checks,
  path confinement and no local fallback.
- `.pose/rules/documentation-style.md`: spec traces, workflow, locales and CLI
  documentation.
- `.pose/rules/knowledge-governance.md`: no accepted risk or follow-up was
  silently disposed of.
- Frontend and Kubernetes rules are not applicable to these changes.

## Findings and remaining sequence

1. High, open (`old-engine-local-shadow`): the Harne8 adoption fixture builds
   the pinned old gitlink engine. It rejects `pose context`, but accepts
   `new-spec --task xref:...` as an old single-project command and creates a
   local draft. A direct probe also confirmed that `spec-transfer` is unknown
   and `check --strict` rejects the schema-4 review policy; those rejections do
   not prevent the old `new-spec` write. This can duplicate spec authority if
   that binary receives a qualified task. The real consumer binding remains
   inactive. Add a guard or
   distribution compatibility test that proves a deployed old engine cannot
   create a shadow during adoption, then reseal and review the agent spec.
2. Attribution reconciled: the late trace commit expands the qualified spec's
   commit interval across other work, but `artifact-check --strict` attributes
   only matching paths and passes. The source lifecycle edits and validation
   state were committed; the roadmap no longer reports
   `source-artifact-revision-mismatch`. Its remaining blocker is the unfinished
   agent spec. Resolution, composition and transfer milestone bundles are
   sealed, but their acceptance gate still observes that blocker, so no
   milestone was approved or closed.
3. The Harne8 adoption remains separate: pin the approved source, align CLI
   and MCP binaries, authorize project roots, exercise populated preview and
   rollback, then pass the consumer's own review and typed delivery gates.

Public compatibility remains opt-in at review-policy schema 3/4. No consumer
policy or `.mcp.json` was activated. The new MCP and CLI paths passed targeted
authorization and negative tests, but the old distribution gap above prevents
the agent-flow spec and roadmap from being approved. The review bundles seal
current test evidence by provenance; several results ran after individual
subject heads, which `review verify` reports as warnings, not as a claim that
those historical heads were retested in isolation.

Continuation handoff: `knowledge:multirepo-review-continuation`.
