# ADR: Materialize review gitlinks from the sealed local origin

## Status

Accepted on 2026-11-20 for `pose-review-dispatch-materializes-sealed-gitlinks`.

## Context

A governed subject already binds a submodule to a digest of its Git commit and verifies its origin worktree. Git leaves submodule directories empty when creating the review worktree. Enabling ordinary submodule update would introduce URL, configuration and hook inputs into an operation with a sealed local source.

## Decision

Walk only the gitlinks present in the sealed subject and the nested gitlinks required by their checkout integrity checks. Read their revisions from committed trees and transfer only the requested commit and its tree from each selected local repository. Keep network protocols, inherited Git configuration, templates and hooks outside this setup. Use standalone private Git administration so there is no object-store sharing or source registration. Keep its metadata outside the reviewer tree, with a relative marker that is stable across runs.

Use the shared setup deadline, bounded subprocess capture and process termination checks. Charge each child tree before transferring it. Inventory the private Git metadata under its captured directory identity and committed byte budget, separately from adapter output. Verify both origin and materialized checkout with the existing dispatch checks before allowing the adapter to execute.

## Consequences

Origin submodules must already be materialized and clean at the applicable pin. The reviewer can inspect pinned contents in place. The nested repositories expose shallow history at the pin, which suffices for the sealed review subject. Setup still fails if required local objects are unavailable or the work exceeds the setup bounds.

## Alternatives considered

Recursive submodule update depends on repository or global remotes. Local worktree registration would share mutable source administration. Outright bypass of submodule verification would leave the review input unwitnessed.
