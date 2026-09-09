---
slug: pose-release-notes-name-contract-adoption
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-adoption-stamp-stays-readable
priority: 0
components: pose-mcp
delivers:
---

# Spec: A release that introduces a governance contract says so

## 1. Intent

### Goal
Have the release notes name the contracts a release introduces, and say what
adopting one costs, without anyone remembering to write it.

### Business value
v2.0.0 added `evidence_vocabulary_reconciled_at` to the review policy, and
v1.8.1 then refused the whole file: its decoder used `DisallowUnknownFields`, so
one key it did not know invalidated the policy. Dropping the strict decoder in
v2.0.2 stops it recurring for keys added from then on, and does nothing for the
adoption itself — the key names a contract the older engine does not have, and
no encoding avoids that. The remedy is to move every tool reading the repository
at once.

The notes for that release said none of this, which is how a repository spent a
day on it. This is a property of adopting a contract, so the version that
introduces one is exactly where it can be said.

Saying it per release by hand is the shape that fails the first time someone
forgets, and this repository has now been wrong three times that way in a month.
The registry already knows which contracts exist; it is the registry that says
it.

### Constraints
- A release that introduces no contract must not carry the warning, or it stops
  meaning anything.
- Notes already frozen must keep their digest. The prepared snapshot is
  immutable and `pose release check` compares against it.

### Non-goals
- Rewriting past releases' notes. They are frozen, and the manifest's digest is
  what makes them so.
- Detecting adoption cost beyond review-policy contracts.

---

## 2. Requirements

### Functional
- R1: Release notes shall carry a Compatibility section when the release
  introduces a governance contract, naming each contract and what it requires.
- R2: The section shall state the boundary each contract actually has: that an
  engine older than the release never applies it, and — only where adopting it
  writes a top-level policy key — that engines before the strict decoder was
  dropped stop reading the repository at all.
- R5: A recorded introducing version shall be found by the lookup in either
  spelling, enforced for every registry entry.
- R3: A release that introduces no contract shall carry no such section.
- R4: Every contract in the registry shall record the release that introduced
  it, enforced, so the next contract's release is not silent.

### Non-functional
- Notes prepared before this change keep their recorded digest.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_closeout.go` — the registry field and lookup
- `pose-mcp/internal/pose/release_lifecycle.go` — the rendered section

### Artifacts
- created: .pose/specs/2026-09-09-pose-release-notes-name-contract-adoption.md
- created: .pose/changelogs/unreleased/pose-release-notes-name-contract-adoption.md
- created: pose-mcp/internal/pose/contract_release_note_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: .pose/specs/2026-09-08-pose-adoption-stamp-stays-readable.md

### Technical risks
- `RenderReleaseNotes` feeds `NotesDigest`, so changing it changes what a future
  release freezes. Past releases are unaffected because nothing re-renders them:
  `checkRelease` and `release notes` both read the file and compare to the
  stored digest.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The registry records which release introduced each contract (R4)
- [x] Increment 2: The notes carry the section, and only when they should (R1, R3)
- [x] Increment 3: The section states each contract's real boundary (R2)
- [x] Increment 4: The lookup normalises both spellings, enforced (R5)

### Validation
- [x] Past releases still check out against their frozen notes

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: where the statement should live — the notes of each contract-adopting
  release, written by hand, or the engine.
- Decision: the engine, from the contract registry.
- Rationale: the registry is already the one place a contract is declared —
  adding one there is what makes `pose update` stamp it and `pose doctor` report
  it. Making it also what the notes say costs one field, and removes the only
  step that depended on someone remembering. A test fails if a contract is added
  without it.

### Decision 3
- Date: 2026-09-09
- Context: the first draft told every reader of every future contract to upgrade
  in lockstep. That is false for a contract recorded as an id inside
  `contract_adoptions`: the map is a key those engines already model, and the
  decoder stopped refusing unknown keys in 2.0.2 regardless.
- Decision: derive the claim from how the adoption is written.
  `stampContractAdoption` prefers a contract's legacy top-level field where it
  has one and otherwise records the id in the map, so `LegacyField != ""` is the
  test, and `StrictPolicyDecoderDroppedIn` names the boundary.
- Rationale: the false version is false in the direction that costs users work —
  it would tell a whole team to upgrade for a contract that breaks nothing they
  read. Raised in review on PR #73.

### Decision 2
- Date: 2026-09-09
- Context: which releases the existing three contracts belong to.
- Decision: read them from history rather than assign them.
- Rationale: `component-aware` and `review-bundles` first appear in a commit
  whose earliest containing tag is v1.1.0; `evidence-vocabulary` in one whose
  earliest is v2.0.0. Measured with `git merge-base --is-ancestor` against every
  tag in order, not inferred from when the specs were written.

---

## 6. Validation

### Strategy
Render the notes for a release that introduces a contract and one that does not,
and confirm the releases already frozen still validate.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose release check --version v2.0.0`
- Scope: this instance
- Expected: valid prepared snapshot

### Execution log
- Date: 2026-09-09
- Environment: local, Go 1.26
- Notes: v2.0.0 renders one contract, v1.1.0 renders two, v2.0.1 renders none.
  `pose release check` reports both v2.0.0 and v3.0.0 as valid prepared
  snapshots under the new binary, so nothing re-renders a frozen file. The first
  run of the new test failed on the wording, which is the assertion doing its
  job — the phrase had been edited after it was written. Review on PR #73 found
  the first draft's claim was false for any contract recorded in the map, and
  that the lookup normalised only its argument: writing `v2.0.0` in the registry
  and stripping the v from the argument alone fails three tests now and would
  have shipped a silent omission before.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <RenderReleaseNotes emits a Compatibility section from ContractsIntroducedIn(version), listing each contract's id and summary; TestReleaseNotesWarnWhenTheReleaseIntroducesAContract>
- R2 [satisfied] <TestTheWarningMatchesHowTheAdoptionIsWritten requires the stronger "move together" claim exactly for the contracts whose adoption writes a top-level key, and names that key; a contract carried in the map gets the weaker, true statement>
- R5 [satisfied] <TestEveryContractRoundTripsThroughTheLookup finds every registry entry under both spellings; reverting to stripping the argument alone fails it along with two others>
- R3 [satisfied] <TestReleaseNotesAreSilentWhenNoContractIsIntroduced renders v2.0.1 and requires no section>
- R4 [satisfied] <TestEveryContractRecordsTheReleaseThatIntroducedIt fails on any registry entry with an empty IntroducedIn>

### Known gaps
- The boundary is derived from whether the adoption writes a top-level key. A
  future contract that changes compatibility some other way — a value shape
  rather than a key — would need the registry to say so.
- Only review-policy contracts are covered. A future contract carried somewhere
  else would need its own registry to be named this way.

---

## 7. Final Report

### Summary
The release that introduces a governance contract now says what adopting it
costs, from the registry rather than from memory.

### Follow-ups

- [open] Say the same thing where an operator meets it rather than only in the notes: `pose update` stamps a contract adoption without mentioning that older engines stop reading the repository — owner:unowned crit:low review:2026-12-09
