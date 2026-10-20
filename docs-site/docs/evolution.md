# Releases and development status

**Doc type:** Explanation · **Reviewed:** 2026-10-20<br>
**Published baseline:** [POSE v7.4.1](https://github.com/oseiaspereira88/pose/releases/tag/v7.4.1), published 2026-10-10.

Use this page to distinguish what you can install today, what is present on the
development branch, and what remains a proposal. A changelog fragment is a
record of a change, including documentary work; it is not proof that a feature
has shipped. The engine version also does not tell you which optional contracts
your project has adopted.

## What the recent releases enable

| Release | Practical change | Where to begin |
|---|---|---|
| [6.0.0](https://github.com/oseiaspereira88/pose/releases/tag/v6.0.0) | Progressive review, R/A/D contract nodes, structural causality, atomic start and qualified project authority; optional contracts require adoption. | [Review and trust](review-and-trust.md), [Multiple projects](multiple-projects.md) |
| [6.1.0](https://github.com/oseiaspereira88/pose/releases/tag/v6.1.0) | More gates expose JSON on stdout; `--json-out` writes a file. `validate --json <path>` is deprecated. | [CLI reference](cli.md) |
| [6.2.0](https://github.com/oseiaspereira88/pose/releases/tag/v6.2.0) | Usage findings gain explicit verdicts; diagnostics explain governance gaps. Native package-manager verification remains deferred. | [Analytics](analytics.md), [Package channels](package-channels.md) |
| [6.3.0](https://github.com/oseiaspereira88/pose/releases/tag/v6.3.0) | Project-owned release version sources, compact delivery indexes and stricter CI/release completeness checks. | [Release notes](https://github.com/oseiaspereira88/pose/releases/tag/v6.3.0) |
| [7.0.0](https://github.com/oseiaspereira88/pose/releases/tag/v7.0.0) | Guided setup, a capability catalog, effective governance, phase-scoped action requests, signed human answers, Attention, typed diagnostics and adaptive assessment freshness. Review the release notes and effective policy before upgrading. | [Operate an instance](operate.md) |
| [7.1.0](https://github.com/oseiaspereira88/pose/releases/tag/v7.1.0) | Attention projects all registered source kinds; MCP can receive signed review attestations. Structural facts follow attributed commits, trace test references must resolve, and recurrence distinguishes settled failures from flapping. | [Review and trust](review-and-trust.md), [MCP](mcp.md) |
| [7.2.0](https://github.com/oseiaspereira88/pose/releases/tag/v7.2.0) | Local Ed25519 issuers let a standalone project sign attestations and prove reviewer authority. | [Native issuers](review-and-trust.md#sign-reviews-without-a-hosted-service) |
| [7.3.0](https://github.com/oseiaspereira88/pose/releases/tag/v7.3.0) | A signed legacy ledger supports adopting signing without rewriting earlier attestations. | [Adopt signing with existing history](review-and-trust.md#adopt-signing-with-existing-history) |
| [7.4.0–7.4.1](https://github.com/oseiaspereira88/pose/releases/tag/v7.4.1) | Bundle-derived review briefs and disposable-worktree dispatch; classification fixes allow the engine's own governance records to be sealed. Dispatch records an execution, never an approval. | [Delegate a review](review-and-trust.md#delegate-a-review) |

## On the development branch, not in v7.4.1

These changes have pending fragments under
[`.pose/changelogs/unreleased/`](https://github.com/oseiaspereira88/pose/tree/main/.pose/changelogs/unreleased).
No future version number or publication date is promised. The status below is
an editorial snapshot of the canonical specs on 2026-10-20, not a fresh
execution of their delivery gates.

| Area | Change recorded in the pending changelog | Spec status at review |
|---|---|---|
| Review trust | Authenticate proof before reporting verified identity; distinguish historical validity from current trust and verified source reuse from reaffirmation. | `done` |
| Approval writes | The shared domain writer refuses invalid, superseded or stale approving attestations across CLI, Store and MCP; negative decisions remain audit records. | `done` |
| Action answers | Reverify persisted answer proof before consuming assurance or satisfaction. | `done` |
| Source coverage | Unreadable specs, findings or review records yield incomplete coverage rather than an apparently empty backlog or a fallback approval. | `done` |
| Automation results | Typed operation results distinguish success, waiting, refusal, conflict, failure and inconclusive execution; parser and retry behavior become explicit. | `done` |
| Recovery | Diagnose invalid action journals and closeout checkpoints; recover a start lock only when its owner is proven dead. | `done` |
| Publication | Unique staging, compare-and-swap publication and shared lifecycle locks prevent concurrent writers from silently replacing each other's work. | `done` |
| Indexes | Publish and verify a coherent seven-file generation; consumers disclose unavailable or legacy-unversioned sets. | `done` |
| Git and dispatch | Refuse unavailable Git observations and hidden edits; verify sealed inputs and origin freshness around dispatch. | `in-progress` |

Read the fragment and its referenced spec before relying on development-only
behavior. The [CLI](cli.md#development-branch-behavior) and
[MCP](mcp.md#development-branch-behavior) references identify this boundary.
Installing `releases/latest` does not install these pending changes.

## Planned work is a separate category

The consolidation bootstrap and batch-authoring fragments document specs,
roadmap ownership, dependencies and acceptance plans. They do **not** establish
that the proposed runtime, caches, IPC, pilots or UX journeys are implemented.
The [product roadmaps](product-roadmaps.md) separate this active program from
the completed historical portfolio. Do not infer capability availability from
an authored spec, a roadmap milestone or the presence of a fragment.

## Upgrade deliberately

1. Read the target release's compatibility section and download its verified assets.
2. Update every CLI and MCP reader that must understand the adopted contracts.
3. Run `pose update`, then inspect `pose setup --json`, `pose adopt --list` and `pose doctor`.
4. Review capability decisions and the effective policy before enabling additional obligations.
5. Run the repository's structural and validation gates before resuming governed work.

Use the [quickstart](quickstart.md#verified-install) for the pinned installation
flow. Keep historical review records immutable. Downgrade is unsupported; a
version update and a policy adoption are separate operations.
