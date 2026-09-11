---
slug: pose-machinery-backs-up-only-local-edits
status: in-progress
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-machinery-distribution-contract
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: `pose update` backs up only machinery the instance edited

## 1. Intent

### Goal
A machinery file a release changed and the instance never touched shall be
refreshed without a backup and without being reported as customized; a file
the instance edited shall keep its backup.

### Business value
Adopting 5.0.3 in harne8, `pose update` wrote seven `.pose-backup` files and
reported each as "backed up customized". Every one was identical to the file
the repository had committed — nothing had been edited. The same happened on
the 5.0.2 and 4.0.1 adoptions, and each time the adoption had to diff the
backups by hand to prove no local change had been thrown away.

The cause: machinery delivery backed up any file whose content differed from
the new release, and the delivery manifest recorded only which paths had been
delivered, not what. With nothing to compare against, "the release changed
this file" and "the instance changed this file" were the same fact. Reported
upstream from `harne8-adopt-pose-v5-0-3`.

### Constraints
- A file the instance did edit still gets its backup, exactly as before.
- A manifest written before this change carries no digests; it must not be
  treated as proof that nothing was edited.

### Non-goals
- Merging an instance's edits into new engine content. The whole file is the
  ownership unit, as `pose-machinery-distribution-contract` decided.
- The managed manuals, which are merged by section and back up only content
  the merge could not keep.

---

## 2. Requirements

### Functional
- R1: The machinery manifest shall record the digest of the content delivered
  to each path.
- R2: A file still matching its recorded digest shall be refreshed without a
  backup and without a report.
- R3: A file differing from its recorded digest shall be backed up and reported
  as customized.
- R4: A file with no recorded digest shall be backed up, and the report shall
  say that a local edit could not be ruled out rather than call it customized.

### Non-functional
- The manifest stays readable by older engines: the field is additive.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/machinery.go` — the manifest and per-file delivery
- docs describing `pose update`

### Artifacts
- created: .pose/specs/2026-09-11-pose-machinery-backs-up-only-local-edits.md
- created: .pose/changelogs/unreleased/pose-machinery-backs-up-only-local-edits.md
- modified: pose-mcp/internal/cli/machinery.go
- modified: pose-mcp/internal/cli/machinery_test.go
- modified: docs-site/docs/cli.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Technical risks
- The first update after this change still backs up every changed file on an
  instance whose manifest predates digests. That is the honest answer for a
  file with no record, and the report now says so; the digests it records make
  the next update exact.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Record delivered digests (R1)
- [x] Increment 2: Decide the backup from them (R2, R3, R4)

### Validation
- [x] A release changing an untouched and an edited file, and a legacy manifest

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: with no recorded digest, the file could be refreshed quietly on the
  assumption that nobody edits machinery.
- Decision: back it up, and say why.
- Rationale: the assumption is exactly the kind a backup exists to avoid; one
  release of honest noise is cheaper than one lost edit.

---

## 6. Validation

### Strategy
Deliver one release, edit one file, deliver the next, and read what was backed
up and reported; repeat from a manifest with no digests.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-11
- Environment: local, Go 1.26
- Notes: observed before the change on harne8's 5.0.3 adoption: seven
  "customized" backups, each byte-identical to the committed file. With it,
  TestDeliverMachineryBacksUpOnlyWhatTheInstanceEdited refreshes the untouched
  file quietly and backs up the edited one as customized, and
  TestDeliverMachineryWithoutADigestSaysWhyItBacksUp keeps the backup for a
  legacy manifest with the new wording and records a digest. The existing
  delivery, deletion and locale tests still pass.

### Results summary
- Successes: R1–R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestDeliverMachineryBacksUpOnlyWhatTheInstanceEdited requires a digest for a delivered path after the first delivery>
- R2 [satisfied] <the same test requires the untouched file to take the new release with no backup and no report>
- R3 [satisfied] <the same test requires the edited file's content in .pose-backup and a "backed up customized" report>
- R4 [satisfied] <TestDeliverMachineryWithoutADigestSaysWhyItBacksUp requires the backup, the "no record of what POSE delivered" wording, no "customized", and a recorded digest>

### Known gaps
- Instances see one more round of backups on their first update after adopting
  this, until digests are recorded.

---

## 7. Final Report

### Summary
An update no longer reports a release's own changes as local edits it threw
away.

### Follow-ups
