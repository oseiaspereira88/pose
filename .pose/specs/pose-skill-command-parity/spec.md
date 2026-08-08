---
slug: pose-skill-command-parity
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-skill-closeout-gate-parity
priority: 1
components: pose-mcp
delivers: governance:skill-command-parity
---

# Spec: Both language versions of a skill teach the same commands

## 1. Intent

### Goal
Fail the build when an English skill and its translation teach different POSE
commands, and settle the format question the previous cycle exposed by
restoring the commands the English rewrite dropped.

### Business value
`skills-check` validates a translated skill's contract and passed for the
entire life of a defect where the pt-BR `pose-spec-closeout` never mentioned
`pose review record`, `pose review-check` or `pose close` — the exact gate
`pose check --strict` enforces. Conformance and correctness are different
properties: a skill can satisfy every metadata rule and still walk an agent
through a workflow the engine rejects.

The manual parity check could not be reused. English skills were rewritten
terse (Codex-native) while translations kept an example-rich shape, so a
token-level comparison would report that difference as drift on every skill and
bury the real signal. The comparison that works is narrower: the POSE commands
and MCP tools each side tells an agent to run — ignoring flags, arguments,
wording and structure.

Applying it answered the open format question with evidence rather than taste.
It found **nine commands the translations teach and the English skills do not**,
across six skills. The rewrite had not merely condensed prose; it had removed
instruction. The starkest case: `pose-feature` in English never named
`pose new-spec` — the skill for building a feature did not mention the command
that creates its spec. `pose-spec-closeout` said "confirm strict deterministic
validation passed" without naming `pose validate`.

So the answer is not to condense the translations to match. The translations
were more complete, and the English skills got the missing commands back.

### Constraints
- Flags and arguments stay out of scope. A translation showing
  `--similarity 45` where the original does not is elaboration, not drift.
- The check must tolerate the format asymmetry it was written alongside, or it
  reproduces the problem it exists to avoid.

### Non-goals
- Reconciling skill formats. That difference is now understood and deliberate:
  terse English, example-rich translations, same commands.
- Comparing prose or structure, which the manual check does for manuals because
  manuals are structurally parallel and skills are not.

---

## 2. Requirements

### Functional
- R1: A deterministic check shall compare, per skill and per locale, the set of
  POSE commands and MCP tools each version teaches.
- R2: It shall fail in both directions, naming the commands — a command only in
  the translation is as much a defect as one only in the original.
- R3: It shall not report flag, argument, wording or format differences.
- R4: It shall fail when no translated skill is compared.
- R5: Every command a translation teaches and its English source dropped shall
  be restored to the English skill.

### Non-functional
- Runs inside the existing suite over 22 files.

### Security
- Read-only.

### Compatibility
- A locale translating only some skills stays valid; each pair is compared only
  where both sides exist.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/scaffold/skill_locale_parity_test.go` — the check.
- Six English skills that lost commands in the terse rewrite.

### Artifacts
- created: .pose/specs/pose-skill-command-parity/spec.md
- created: pose-mcp/internal/scaffold/skill_locale_parity_test.go
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: .agents/skills/pose-bugfix/SKILL.md
- modified: .agents/skills/pose-review/SKILL.md
- modified: .agents/skills/pose-doc-update/SKILL.md
- modified: .agents/skills/pose-knowledge/SKILL.md

### Delivery targets
- governance:skill-command-parity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- **Multi-word commands are enumerated by hand.** `review record` and
  `assess snapshot` are one command each, while `close spec:<slug>` is a command
  plus an argument, and no pattern separates them. A new two-word verb is
  compared by its first word only until the list is extended — under-reporting
  rather than false-alarming, which is the wrong direction for a guard and is
  recorded rather than hidden.
- A command taught only in prose, without the `pose ` prefix, is invisible. That
  was already true in `pose-knowledge`, which wrote `knowledge-housekeeping`
  bare; it was corrected here, but nothing prevents a recurrence.

---

## 4. Tasks

### Planning
- [x] Establish why the manual token comparison cannot be reused
- [x] Measure what each side teaches before deciding the format question

### Implementation
- [x] R1: compare taught command sets per skill and locale
- [x] R2: fail symmetrically, naming commands
- [x] R3: exclude flags, arguments and format
- [x] R4: fail on an empty comparison
- [x] R5: restore the nine dropped commands to the English skills

### Validation
- [x] Prove the check against the real defect
- [x] Prove the format asymmetry does not trip it
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Two properties must hold at once, and a check with only the first is worse than
none: it must catch a dropped command, and it must stay silent on the format
difference. Both are pinned by fixtures — a translation missing two commands,
and the same commands written terse against example-rich with extra flags.

The historical proof is the defect itself: pointed at the pt-BR
`pose-spec-closeout` as it stood before the previous fix, the check must name
the three missing commands.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/scaffold/... -run "SkillLocaleParity|SkillParityRejects" -count=1`
- Scope: 11 skill pairs plus the reject/allow fixtures
- Expected: pass

#### Security / Contract
- Command: `pose skills-check --strict`
- Scope: contract conformance, unaffected by this change
- Expected: pass

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64.
- Notes: the first run reported nine commands taught only in translation —
  `new-knowledge` (pose-bugfix, pose-feature, pose-review), `report`
  (pose-doc-update), `followups` and `new-spec` (pose-feature),
  `knowledge-housekeeping` (pose-knowledge), `validate` and `new-spec`
  (pose-spec-closeout). Each was verified against both files before being
  restored. Pointed at `95797d6:locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md`
  — the skill before the previous cycle's fix — it names `close`,
  `review record` and `review-check`. `skills-check --strict` reports 0/0.

### Results summary
- Successes: the real defect is detected; the format asymmetry is not; nine
  dropped commands restored
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:skill-command-parity evidence:integration check:delivery-integration test:pose-mcp/internal/scaffold/skill_locale_parity_test.go — command and MCP-tool sets are extracted per pair and compared
- R2 [satisfied] test:pose-mcp/internal/scaffold/skill_locale_parity_test.go — both directions are separate assertions with distinct messages; the reverse direction is what surfaced the nine dropped commands
- R3 [satisfied] test:pose-mcp/internal/scaffold/skill_locale_parity_test.go — the allow fixture pairs a terse English form against an example-rich translation with extra flags and must produce no finding
- R4 [satisfied] test:pose-mcp/internal/scaffold/skill_locale_parity_test.go — a run comparing nothing fails, stating the discovery is broken
- R5 [satisfied] report:.agents/skills/pose-feature/SKILL.md — nine commands restored across six skills, including `pose new-spec` in the feature skill and `pose validate` in the closeout skill

### Known gaps
- Multi-word commands come from a hand-maintained list; a new two-word verb is
  compared by its first word until added.
- A command written without its `pose ` prefix is invisible to the check.
- Command parity is not workflow parity: both sides can name the same commands
  in a different order or for different reasons.

---

## 7. Final Report

### Delivered scope
A symmetric command-parity check across every translated skill, and the nine
commands the English rewrite had dropped restored to their skills.

### Files and modules changed
- pose-mcp/internal/scaffold/skill_locale_parity_test.go
- six skills under .agents/skills/

### Validation executed
- Command: the parity test plus its reject/allow fixtures; `skills-check --strict`
- Result: pass; three commands named when pointed at the pre-fix skill

### Residual risks
- The check governs which commands are taught, not whether the surrounding
  guidance is still correct.

### Follow-ups

- [open] The multi-word command list (`review record`, `assess snapshot`, …) is maintained by hand and under-reports when a new two-word verb appears — a guard failing quiet is the wrong direction. Consider deriving it from the CLI's own dispatch table at test time, which would also fix the same weakness in the manual parity check's command discovery. (owner:@pose-maintainers crit:low review:2026-11-06)
