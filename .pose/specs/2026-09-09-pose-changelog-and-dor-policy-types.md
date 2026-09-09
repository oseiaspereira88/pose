---
slug: pose-changelog-and-dor-policy-types
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-policy-keys-and-release-surface-coverage
priority: 0
components: pose-mcp
delivers:
---

# Spec: changelog.json and dor.json are read through types of their own

## 1. Intent

### Goal
Give both policies a named type, make `changelog.json`'s `categories` the
setting it looks like, and say in `dor.json` that the gate is opt-in.

### Business value
`pose-policy-keys-and-release-surface-coverage` held seven policies to the keys
the engine reads and had to exempt these two, because neither had a type to
derive a key list from. The exemption recorded what the audit found while
looking: both files disagree with what is read.

`changelog.json` ships `categories` — the six valid fragment categories — and
nothing reads it. The set is written out three times instead: in the fragment
loader, in `pose check`, and implicitly in the notes renderer's order. A file
that promises a setting it does not have is worse than one that promises
nothing, and three copies of a list is how they come to disagree.

`dor.json` is the mirror image. `readiness.go` looks for `adopted_at`, which the
shipped policy does not contain, so the Definition of Ready reads as unadopted
on every fresh install. That is the intent — the gate applies to specs created
after a project adopts it — but nothing said so, and an omitted key is
indistinguishable from an oversight.

### Constraints
- An instance that never touches either file must behave exactly as before.
- The notes' section order is presentation, not the order anyone writes a list
  in. Making the set configurable must not reorder the output.

### Non-goals
- Changing when the Definition of Ready applies. The shipped value is empty,
  which is what an absent key already meant.

---

## 2. Requirements

### Functional
- R1: `changelog.json` shall be read through a named type, and `categories`
  shall govern which categories a fragment may declare, everywhere the set is
  consulted.
- R2: An absent policy, or one declaring no categories, shall keep the six
  defaults.
- R3: A category the policy adds shall render under a heading of its own, after
  the known ones, rather than being dropped.
- R4: `dor.json` shall be read through one named type covering both halves of
  the file, and shall ship an explicit empty `adopted_at`.
- R5: Both files shall be held to their keys by `pose doctor`, and the
  exemption list shall no longer name them.

### Non-functional
- All nine policy-key findings report ok on a fresh install.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/changelog_policy.go` — the type and the category rules
- `pose-mcp/internal/pose/dor_policy.go` — the type and the adoption test
- `pose-mcp/internal/pose/release_lifecycle.go` — the loader and the renderer
- `pose-mcp/internal/cli/check.go` — the third copy of the list
- `pose-mcp/internal/pose/readiness.go` — the anonymous struct it used

### Artifacts
- created: .pose/specs/2026-09-09-pose-changelog-and-dor-policy-types.md
- created: .pose/changelogs/unreleased/pose-changelog-and-dor-policy-types.md
- created: pose-mcp/internal/pose/changelog_policy.go
- created: pose-mcp/internal/pose/dor_policy.go
- created: pose-mcp/internal/pose/changelog_dor_policy_test.go
- modified: .pose/policy/dor.json
- modified: pose-mcp/internal/scaffold/dist/.pose/policy/dor.json
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/pose/readiness.go
- modified: pose-mcp/internal/pose/release_lifecycle_test.go
- modified: pose-mcp/internal/pose/contract_release_note_test.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/policy_keys.go
- modified: pose-mcp/internal/cli/policy_keys_test.go
- modified: .pose/specs/2026-09-09-pose-policy-keys-and-release-surface-coverage.md

### Technical risks
- `LoadReleaseFragments`, `RenderReleaseNotes` and `NewReleaseManifest` gained a
  parameter, so every call site had to be found rather than defaulted. That is
  the point: a default would have left a caller validating against the six while
  another validated against the policy.

---

## 4. Tasks

### Implementation
- [x] Increment 1: A named changelog type, and one set of categories (R1, R2)
- [x] Increment 2: A configured category renders (R3)
- [x] Increment 3: One DoR type, and an explicit empty adoption date (R4)
- [x] Increment 4: Both held to their keys, exemption list emptied (R5)

### Validation
- [x] The refusal message shown quoting the policy rather than a fixed list

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: `categories` could govern which categories are valid, or the order
  the notes present them in. The shipped list is in neither the render order nor
  alphabetical.
- Decision: it governs validity. Presentation order stays in code, with anything
  the policy adds rendered after the known categories, sorted.
- Rationale: the shipped list is exactly the six valid ones, which is what the
  key means. Rendering in policy order would change every instance's notes for
  no reason the operator asked for, and the order that puts `security` first is
  a reading decision, not a configuration one.

### Decision 2
- Date: 2026-09-09
- Context: `dor.json` could gain `adopted_at` with today's date, or with an
  empty value.
- Decision: empty.
- Rationale: an empty value is exactly what the absent key already meant, so no
  instance changes behaviour. Writing a date would silently adopt a gate on
  every existing repository at its next update.

### Decision 3
- Date: 2026-09-09
- Context: the three functions that needed the policy could take it as a
  parameter, or read it from a root they derive.
- Decision: take it as a parameter.
- Rationale: `LoadReleaseFragments` receives a directory, not a root, and
  deriving one by walking up three levels is the kind of inference that is right
  until a caller passes something else. Adding the parameter made the compiler
  name all eleven call sites.

---

## 6. Validation

### Strategy
Configure a category set and require the engine to honour it, including in the
message an author reads; and require an untouched instance to be unchanged.

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
- Date: 2026-09-09
- Environment: local, Go 1.26
- Notes: a policy declaring `["added","fixed"]` makes the loader refuse a
  `security` fragment with `want added|fixed`, quoting the policy rather than a
  fixed list. A policy adding `performance` renders `## Performance` after the
  known headings. An absent or empty policy keeps the six. On a fresh `pose
  install` all nine `*.policy-keys` findings report ok, including the two this
  spec adds.

### Results summary
- Successes: R1 through R5 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <ChangelogPolicy.ValidCategories backs the fragment loader and pose check, and CategoryList backs the message; TestChangelogCategoriesGovernWhatAFragmentMayDeclare requires the refusal to quote the configured set and not the hard-coded one>
- R2 [satisfied] <TestChangelogCategoriesFallBackToTheDefaults covers an absent policy and an empty list>
- R3 [satisfied] <TestRenderOrderIsPresentationAndStillCarriesAnAddedCategory asserts the heading exists and comes after the known ones>
- R4 [satisfied] <DoRPolicy carries schemaVersion, adopted_at, defaultTaskType and taskTypes, and both readiness.go and check.go read it; the shipped dor.json now carries adopted_at:"" with a comment saying the gate is opt-in>
- R5 [satisfied] <policyKeyChecks holds changelog.json and dor.json; policyFilesWithoutAModelledStruct is empty, and TestEveryShippedPolicyIsHeldToItsKeysOrExempted fails on any shipped policy that is neither>

### Known gaps
- The shipped `changelog.json` carries `adopted_at: "2026-08-03"`, which is this
  repository's own adoption date travelling to every instance. Same family as
  the self-referential policy roots of issue #17, in a file that was not part of
  that fix.
- A configured category renders under a capitalised form of its own name. A
  project wanting a different heading has no way to say so.

---

## 7. Final Report

### Summary
Both files are read through types that describe them, and `categories` is the
setting it always looked like.

### Follow-ups

- [open] Ship `changelog.json` with an empty `adopted_at` rather than this repository's own date, which every instance currently inherits — same family as the self-referential policy roots of issue #17 — owner:unowned crit:medium review:2026-12-09
- [open] Let a configured changelog category declare its heading, instead of capitalising its own name — owner:unowned crit:low review:2027-03-09
