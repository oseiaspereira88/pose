---
slug: pose-manual-distribution-merge
status: in-progress
created_at: 2026-08-07
completed_at:
supersedes:
depends_on: pose-command-reference-parity
priority: 1
components: pose-mcp
delivers: capability:manual-distribution
---

# Spec: Managed manuals reach installed repositories

## 1. Intent

### Goal
Let canonical POSE.md and AGENTS.md content reach an already-installed
repository without destroying the sections that repository owns.

### Business value
Today the manual has no distribution path at all. `pose upgrade` delivers only
the binary and the schema stamp; `pose upgrade --force` delivers everything but
overwrites the instance's own limitations, next steps and engine-feedback
sections. Every manual improvement therefore reaches new installs only — v0.16.6
shipped a POSE.md-only fix whose stated purpose was preventing scope loss during
`pose upgrade`, and no consumer could receive it.

### Constraints
- Never lose content an instance wrote, including sections the engine does not
  ship.
- Never reintroduce an unresolved `{{PROJECT_NAME}}`/`{{PROJECT_ID}}` placeholder
  into a rendered manual.
- Keep `--force` as an explicit full reset.

### Non-goals
- Distribute rules, workflows, templates or skills outside `--force`.
- Convert instance-owned sections into a separate file.

## 2. Requirements

### Functional
- R1: The canonical manuals shall mark instance-owned sections, and the merge
  shall refresh every other section from the shipped manual.
- R2: `pose install` on an existing manual shall merge rather than skip, and
  `--force` shall keep overwriting.
- R3: `pose upgrade` shall refresh managed manuals on every run, not only under
  `--force`.
- R4: The merge shall preserve local sections absent from the canonical manual
  and shall be idempotent.

### Non-functional
- Deterministic and offline: no network, no ordering dependence.

### Security
- No path outside the target root is read or written.

### Compatibility
- Additive marker comment; manuals without markers merge as fully engine-owned.
- Instances that never upgrade are unaffected.

## 3. Technical Plan

### Affected areas
- Installer and upgrade paths, plus the canonical manuals in both locales.

### Artifacts
- created: .pose/specs/pose-manual-distribution-merge/spec.md
- created: pose-mcp/internal/cli/managed_docs.go
- created: pose-mcp/internal/cli/managed_docs_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/maintenance.go

### Delivery targets
- capability:manual-distribution module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- Add the `<!-- pose:instance-owned -->` marker to the manual contract.
- `pose upgrade` gains manual refresh as normal behavior.

### Data/storage changes
- None.

### Technical risks
- Heading-based matching breaks if an instance renames an instance-owned
  heading; the renamed section is then preserved as an unknown local section
  rather than merged, which loses no content but leaves a duplicate.
- Placeholder recovery reads the instance's own rendered sentence; an instance
  that rewrote that sentence keeps the placeholder unresolved instead of
  guessing a project name.

## 4. Tasks

### Planning
- [x] Establish why the consumer received no manual content.

### Implementation
- [x] Mark instance-owned sections in both canonical manuals.
- [x] Add the section-aware merge.
- [x] Merge on install and refresh on every upgrade.

### Validation
- [x] Unit-test the merge and run an end-to-end upgrade on a real repository.

## 5. Decisions

- Markers live in the canonical manual only, and matching is by heading. An
  instance therefore needs no migration: the first upgrade after this change
  starts merging correctly against manuals that predate the marker.

## 6. Validation

### Strategy
Unit-test the merge for refresh, preservation, unknown sections, duplicate
headings and idempotence; then install a throwaway repository, age its manual,
run `pose upgrade` without `--force` and assert that engine prose refreshed
while the instance section, project name and absent placeholders held.

### Deterministic checks

| Class | Command | Expected evidence |
|---|---|---|
| Required unit | `go -C pose-mcp test ./internal/cli -run 'MergeManagedDoc\|RefreshManagedDocs' -count=1` | refresh, preservation, idempotence PASS |
| Required module | `go -C pose-mcp test ./... -count=1` | all packages PASS |
| Required static | `go -C pose-mcp vet ./...` | no findings |
| Required structure | `pose check --strict` | PASS |

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.17.0-dev.
- Notes: End-to-end run on a throwaway repo confirmed stale engine prose
  removed, instance section kept, `pose_mcp_context` guidance delivered, project
  name preserved and no placeholder reintroduced.

### Results summary
- Successes: unit, module, vet and structural checks; end-to-end upgrade.
- Failures: the first merge implementation joined section bodies as strings and
  dropped the blank line before each heading; reworked to carry line slices.
- Warnings: none.

### Requirement trace
- R1 [satisfied] capability:manual-distribution test:TestMergeManagedDocRefreshesEngineSectionsAndKeepsInstanceOnes
- R2 [satisfied] capability:manual-distribution test:TestRefreshManagedDocsUpdatesAnInstalledManual
- R3 [satisfied] capability:manual-distribution test:TestRefreshManagedDocsUpdatesAnInstalledManual test:TestRefreshManagedDocsIgnoresAbsentManual
- R4 [satisfied] capability:manual-distribution test:TestMergeManagedDocKeepsSectionsTheEngineDoesNotKnow test:TestMergeManagedDocIsIdempotent test:TestMergeManagedDocPrefersFirstDuplicateHeading

### Known gaps
- Only POSE.md and AGENTS.md are distributed outside `--force`; rules,
  workflows and skills still require it.

## 7. Final Report

### Delivered scope
A section-aware merge for managed manuals, instance-owned markers in both
locales, merge-on-install and manual refresh on every upgrade.

### Files and modules changed
- `pose-mcp` installer/upgrade paths and the canonical manuals.

### Validation executed
- Command: `go -C pose-mcp test ./... -count=1`, `pose check --strict`.
- Result: SUCCESS.

### Residual risks
- A consumer only benefits from the next upgrade onward; manuals already drifted
  stay drifted until then.

### Follow-ups

- [open] Rules, workflows, templates and skills still reach an instance only under `pose upgrade --force`, which overwrites local edits wholesale. Machinery needs the same distinction between engine-owned and instance-owned content that the manuals now have, so a plain upgrade can deliver them safely. (owner:@pose-maintainers crit:medium review:2026-10-07)
