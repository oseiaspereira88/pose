# ADR: Update preview records shared execution in an operation-scoped view

## Status

Accepted (2026-10-22) for implementation under spec `pose-update-dry-run-reports-the-whole-update`. This is a prospective engineering decision, not a declaration that the replacement is delivered. It supersedes the copy-based implementation strategy of spec D1, preserving that historical decision and its evidence.

## Context

B29 remains in-progress. The maintainer rejected delivery of the copy simulator after repeated fidelity failures, including executable-only directories where known children are readable but enumeration is not. Eagerly cloning the instance also omits source discovery and repeatedly reconstructs symlink/permission semantics incorrectly. The effect boundary includes plain update, force install, configuration seeding and migration, project identity, action requests, index generation, MCP configuration, skill links and final governance checks. Recording only manuals/machinery would not meet R1/R2. Separate design review is recorded in `../reports/2026-10-22-b29-update-preview-design-review.md`.

The existing publisher contract and the decision `2026-08-15-retired-machinery-files-stay-on-disk-never-auto-migrated-by-pose-update` remain authoritative: retain retired files, do not install extensions implicitly, preserve atomic/exclusive publication and observable partial-delivery behavior.

## Decision

Pass an explicit invocation-scoped execution context through the reachable update/install path. Both apply and preview run the same operation decisions. Apply uses existing native filesystem/publication mechanisms; preview reads original inputs lazily and records changes in an in-memory copy-on-write view. Subsequent governance reads observe staged writes. Capture one time/options/locale per invocation; no process-global dry-run flag, root registry, patched os functions or hidden concurrent state.

Preserve distinct operations for direct overwrite, exclusive creation, atomic publication, mkdir, removal, symlink and mode changes. A direct write through a link differs from replacing its directory entry. Preserve publication expectation/conflict/durability behavior in apply. Do not convert update/install into a new all-or-nothing transaction. Preserve order and explicit partial failure reports.

Run source-module discovery against the original source tree, with existing exclusions, ignore/submodule and alias semantics. Run governance consumers against the projected view, including seeded policies, request-origin validation and generation computation. Propagate the view through every newly constructed Store/resolver and relevant package-level reader. Any uncovered physical mutator or projected-state read is an implementation defect, not an acceptable fallback.

Preview the running engine's instance update, equivalent in scope to `--no-self`; do not download, self-update, create temporary copies or a fake Git repository, acquire publisher locks, or write through external links. Explain this self-update boundary. Preview request identities are observations at the captured instant, not reservations for a future apply. Emit per-capability request semantics and the same semantic operation wording in apply/preview, with only preview prefixes added by rendering.

Track an ordered semantic ledger separately from the final net filesystem difference. Count real content, directory, link, mode and request effects, without counting repeated writes as different files. Report known refusals and incomplete evaluation explicitly. Print no-change only when every applicable phase was evaluated and no effect remains. A preview describes intended operations under observed inputs; it cannot guarantee future write success under concurrent changes, ACL/security policy, quotas, mount/durability or provider failures. Such runtime uncertainty must not be presented as a proved failure or a fabricated successful application.

Bound read-only knowledge explicitly: do not infer ordinary content semantics solely from regular-file mode on virtual providers, or creation modes solely from umask when a default ACL is inherited. Unsupported provider, inheritance, inode alias, sticky-directory authority and effective-permission observations persist incomplete evaluation. Native apply remains responsible for actual kernel execution; platform compilation alone establishes no runtime parity.

Keep R3 warnings and existing flags compatible. Do not adopt policy, release, migrate projects, publish tags or change Harne8 adoption/gitlinks as part of this work.

## Alternatives and consequences

- Reject another eager-copy repair: it retains the demonstrated class of semantic mismatch and substitutes the discovery environment.
- Reject a second text-only planner: independently describing only selected writes recreates drift and omits reachable effects.
- Reject OS-specific overlay/FUSE as the baseline: external binaries, privileges and platform constraints contradict the standalone cross-platform engine.
- Reject a general POSIX emulator and a new atomic update transaction: excessive scope; model only operations the governed update actually uses, exposing unsupported cases rather than asserting completeness.

The change is broader than the original ten-file patch and requires explicit read/write dependency injection across CLI and model readers. Expand exact artifact declarations before each production edit. Prove no reachable preview mutator, source-discovery fidelity, read-after-write semantics, force and locale parity, and refusal/unknown/no-op behavior with deterministic tests. Reassess if future update phases introduce unmodeled effects or external integrations.
