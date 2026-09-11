---
slug: pose-manual-merge-backs-up-only-local-edits
status: in-progress
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-machinery-backs-up-only-local-edits
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: The manual merge backs up only what the instance edited

## 1. Intent

### Goal
When a release rewords an engine-owned section of POSE.md or AGENTS.md that the
instance never touched, the merge shall replace it without a backup and without
a warning; an edit the instance made that the merge cannot keep shall still be
backed up.

### Business value
Adopting 5.0.4 in harne8, machinery delivery wrote no backup — the fix 5.0.4
shipped — but the manual merge still backed up POSE.md with "content outside
instance-owned sections was not preserved". The backup was identical to the
committed file; the only difference was 5.0.4's own rewording of the `update`
entry.

The merge decides what it lost by comparing lines: any non-blank line of the
local manual absent from the result counts as dropped. That catches a note an
instance wrote inside an engine-owned section, which is what it is for. It
equally catches every line a release rewrote, so each release that touches the
manuals reads as discarding local content. It is the defect 5.0.4 fixed for
machinery, one path over. Reported upstream by `harne8-adopt-pose-v5-0-4`.

### Constraints
- A note the instance wrote inside an engine-owned section is still backed up.
- Instance-owned sections are untouched: the merge already keeps them whole.
- A manual with no record is not proof that nothing was edited.

### Non-goals
- Keeping an instance's edit inside an engine-owned section in place; the
  section is the unit, and the backup is how the edit survives.

---

## 2. Requirements

### Functional
- R1: The delivery manifest shall record, for each managed manual, the digest of
  every section POSE wrote, and of the preamble.
- R2: A section still matching its recorded digest shall be left out of the
  lost-content comparison.
- R3: A section differing from its record shall be compared as before, and a
  lost line backed up and reported as customized.
- R4: A manual with no record shall be compared as before, backed up with a
  report saying a local edit could not be ruled out, and then recorded.
- R5: An update that changes nothing shall not rewrite the record.
- R6: Only sections POSE writes from the canonical manual shall be recorded —
  never an instance-owned section or one the instance invented.

### Non-functional
- `install` and `update` apply the same rule.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/managed_docs.go` — the refresh and the comparison
- `pose-mcp/internal/cli/install.go` — the install merge
- `pose-mcp/internal/cli/machinery.go` — the manifest's `manuals` field

### Artifacts
- created: .pose/specs/2026-09-11-pose-manual-merge-backs-up-only-local-edits.md
- renamed: .pose/changelogs/unreleased/pose-manual-merge-backs-up-only-local-edits.md -> .pose/changelogs/v5.0.5/pose-manual-merge-backs-up-only-local-edits.md
- modified: pose-mcp/internal/cli/managed_docs.go
- modified: pose-mcp/internal/cli/managed_docs_test.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/machinery.go
- modified: docs-site/docs/cli.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Technical risks
- Machinery delivery rewrites the manifest after the manual merge on the same
  update; it now reads the manifest and keeps the manual records instead of
  replacing the whole file.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Record each manual's sections as written (R1, R5)
- [x] Increment 2: Compare only sections that differ from the record (R2, R3, R4)
- [x] Increment 3: Record from the canonical manual, not the merged one (R6)

### Validation
- [x] The main test fails with the record ignored, with the reported symptom

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: the record could be one digest per manual, which is simpler.
- Decision: one digest per section, plus the preamble.
- Rationale: instances routinely write in their own sections, so a whole-file
  digest would almost never match and the false positive would stay. The merge
  already works section by section; the record matches its unit.

### Decision 2
- Date: 2026-09-11
- Context: a no-op merge could refresh the record to the current file.
- Decision: record only when no record exists.
- Rationale: the record means "what POSE wrote". Refreshing it on a no-op
  changed the manifest on an update that changed nothing, which the upgrade
  idempotency test caught.

### Decision 3
- Date: 2026-09-11
- Context: review of pose#103 — the first version recorded every section of the
  merged manual, including instance-owned sections and sections the instance
  invented. A later release shipping an engine section under an invented
  heading would then replace the instance's text, and the comparison would skip
  it because its body matched the record: a silent loss, the opposite of what
  the change is for.
- Decision: record digests from the canonical manual, engine-owned sections and
  preamble only.
- Rationale: the record means "what POSE wrote". Every engine-owned section of
  the merged manual is the canonical body verbatim, so recording the canonical
  describes the file exactly where POSE wrote it and nowhere else.

---

## 6. Validation

### Strategy
Simulate a manual an older release wrote, one an instance edited, and one with
no record, and read what the merge backs up and reports.

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
- Notes: observed on harne8's 5.0.4 adoption: a POSE.md backup identical to the
  committed file. TestRefreshManagedDocsReplacesAnOlderReleaseSectionWithoutABackup
  replaces an older release's section quietly; with the record ignored it fails
  with the backup and the warning. TestRefreshManagedDocsWithoutARecordSaysWhyItBacksUp
  keeps the backup for a manual with no record, with the new wording, and records
  it. The existing edited-section test still reports "backed up customized", and
  TestUpgradeApplyIsIdempotentAndPreservesInstanceContent still sees only
  `schema-version` change.

### Results summary
- Successes: R1–R5 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <recordDeliveredManual stores a digest per section and the preamble; the no-record test requires a record after the refresh>
- R2 [satisfied] <TestRefreshManagedDocsReplacesAnOlderReleaseSectionWithoutABackup requires no backup, no report, and the older text replaced>
- R3 [satisfied] <TestRefreshManagedDocsWarnsAndBacksUpDroppedContent still requires "backed up customized" and the note in the backup>
- R4 [satisfied] <TestRefreshManagedDocsWithoutARecordSaysWhyItBacksUp requires the backup, "no record of what POSE delivered", no "customized", and a record>
- R5 [satisfied] <TestUpgradeApplyIsIdempotentAndPreservesInstanceContent requires only schema-version to change on a no-op update>
- R6 [satisfied] <TestAnInventedSectionIsNeverRecordedAsDelivered requires neither an invented nor an instance-owned heading in the record, and fails against the first version; TestAReleaseClaimingAnInventedHeadingStillBacksItUp requires the replaced invented section to count as lost>

### Known gaps
- An instance's first update after adopting this has no manual record, so a
  reworded manual is backed up once more — with the report saying why — and
  recorded.

---

## 7. Final Report

### Summary
A release rewording its own sections of the manuals no longer reads as local
content being thrown away.

### Follow-ups
