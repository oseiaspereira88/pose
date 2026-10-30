---
slug: pose-delivery-later-removal
status: in-progress
created_at: 2026-10-30
completed_at:
supersedes:
depends_on: pose-delegation-parity-gate
remediates:
priority: 3
components: pose-mcp
task_type: feature
delivers: capability:delivery-later-removal
---

# Spec: A file a later spec removes no longer has to exist for the specs that created it

## 1. Intent

### Goal

Let a spec remove a file that earlier done specs declare as `created` or `modified`: the earlier claims stay true of their time, and their existence check is satisfied by the later, observed removal — only when the removal comes after the change it supersedes.

### Business value

Follow-up F5 of `pose-delegation-parity-gate`: removing `pose review dispatch` meant deleting `pose-mcp/internal/cli/review_dispatch.go`, which five done specs declare as current; `check --strict` refused, and the file was kept only to hold the removed aliases' answer. POSE knew how a release archives a file, not how a later spec removes one, so any removal of a file an earlier spec created forced either keeping it or rewriting a closed spec's history.

### Constraints

Done specs are not edited: their claims remain as written. The rule is decided from Git history, not from dates or spec order in the filesystem. An archival by a release keeps working as it does.

### Non-goals

Inferring removals no spec declares; removals outside attributed change sets; renames (already modelled).

## 2. Requirements

### Functional

- R1: When a path a spec declares `created` or `modified` is not tracked at the selected head, its existence shall be satisfied when another spec declares `removed:` for that path and its attributed change sets contain a commit that deletes the path — absent there, present in a parent that does not rename it away — of which every attributed commit of the declaring spec that changes the path's entry against any of its parents (a root creation, a type change, a deletion and a merge included) is a Git ancestor; commits are gathered from all of a spec's change sets, an explicit range standing for that spec's, and judged by ancestry only.
- R2: A satisfied claim shall be recorded in the delivery-integrity graph as an edge `removed-by` from the artifact to the removing spec, naming the deleting commit, and the removal witnesses shall be part of the graph's inputs.
- R3: The existence finding shall remain when a change of the declaring spec to the path follows the removal or is not its ancestor, when the declaring spec is the removing spec, when the removal claim is not observed as a deletion in the removing spec's change sets, when a change set names a commit that is not an immutable object name, when Git cannot read the history it needs, or when no spec declares the removal; paths are read literally, never as pathspecs, and the message shall say which of these kept the removal from applying when a removal claim exists.
- R4: `pose index`, `pose check --strict`, `pose artifact-check` and `pose surface-check` shall apply the same rule from one computation.
- R5: `pose-mcp/internal/cli/review_dispatch.go` shall be removed, the removed aliases' answer moved back into the review command, and the five done specs that declare it shall pass `check --strict` through R1.
- R6: The operating manual shall describe how a later spec removes a file earlier specs declare.

### Non-functional

- Git is consulted only for paths that are absent at the head and that some spec declares removed.

### Security

- None beyond reading Git history already read for change sets.

### Compatibility

- A repository with no `removed:` claim for an absent path sees no change; an existence finding that a removal now satisfies disappears.

## 3. Technical Plan

### Affected areas

Delivery-integrity graph (existence rule, removal witnesses, edges), the CLI that builds its inputs from Git, the review command, the manuals.

### Artifacts

- created: .pose/specs/2026-10-30-pose-delivery-later-removal.md
- created: .pose/starts/pose-delivery-later-removal.json
- created: .pose/changelogs/unreleased/pose-delivery-later-removal.md
- modified: .pose/specs/2026-10-26-pose-delegation-parity-gate.md
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/delivery_integrity_compact.go
- modified: pose-mcp/internal/pose/delivery_integrity_cache.go
- modified: pose-mcp/internal/pose/delivery_integrity_compact_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/surface_check.go
- created: pose-mcp/internal/cli/delivery_later_removal_test.go
- created: pose-mcp/internal/cli/delivery_later_removal_history_test.go
- created: .pose/specs/2026-10-30-pose-delivery-later-removal.amendments.jsonl
- modified: pose-mcp/internal/cli/review_closeout.go
- removed: pose-mcp/internal/cli/review_dispatch.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delivery-later-removal module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/artifact_integrity.go

### API/contract changes

Edge type `removed-by` and removal witnesses in `delivery-integrity.json`; the existence rule of R1–R3.

## 4. Tasks

- [ ] Removal witnesses computed from Git
- [ ] Existence rule and graph edges
- [ ] Same rule in index, check, artifact-check, surface-check
- [ ] Remove review_dispatch.go
- [ ] Manual
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R3
- Minimal option: accept any `removed:` claim for the path from any spec
- Selected option: accept a removal only when it is observed as a deletion and the superseded change is its Git ancestor
- Rationale: an unordered rule would excuse a claim made after the removal — a spec that declares it modified a file that no longer exists — which is exactly the error the existence check catches (maintainer's question, 2026-10-30)
- Consequences: Git ancestry is read for the absent paths a spec removes; a removal on a branch that does not contain the earlier change satisfies nothing
- Falsifier: a claim made after an observed removal that the check accepts, or a claim before it that the check still refuses

## 6. Validation

### Strategy

A Git fixture with spec A creating a file, spec B deleting it later and declaring `removed:`: A's existence passes and the graph has the edge; spec C declaring `modified:` after the removal still fails; B's removal unobserved, B on a branch not containing A's change, and A removing its own file all keep the finding. Then the real removal of `review_dispatch.go` passes `check --strict`. Each rule is shown failing on revert.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: follow-up F5 of `pose-delegation-parity-gate`.

### Residual risks

None yet.

### Follow-ups
