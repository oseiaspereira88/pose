---
slug: pose-skill-closeout-gate-parity
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-manual-locale-parity
priority: 1
components: pose-mcp
delivers: governance:skill-closeout-gate-parity
---

# Spec: The translated closeout skill teaches the gate it must pass

## 1. Intent

### Goal
Restore the hierarchical review gate to `locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md`,
which instructs agents through a closeout that `pose check --strict` then rejects.

### Business value
`pose-spec-closeout` is the skill an agent loads to close a spec. The English
version routes the closeout through the review gate — `pose review record`,
`pose review-check`, `pose close`. The pt-BR version did not mention any of the
three: it went from follow-up triage straight to hand-editing `status: done` in
the frontmatter.

That is not a cosmetic gap. The review gate is exactly what `pose check
--strict` enforces, so an agent following the translated skill produces a spec
that fails the structural gate with `review closeout: record or remediate a
fresh review` — an error whose cause is not visible from its text. This session
hit that failure repeatedly while closing specs, which is what makes the
consequence concrete rather than theoretical.

The gap came from one commit: `feat: enforce hierarchical review closeout`
(2026-08-02) updated the English skill and never reached the translation.

Measured rather than assumed: of the 6–9 commits per skill that touched only
the English side, almost all are the `feat(i18n)` commits of 2026-07-17/18 that
*created* the English versions from the Portuguese originals — this repository
started in pt-BR, so those are not drift. Filtering those out leaves three
candidates across eleven skills, and two dissolve under inspection: `remove
legacy runtime and fallbacks` replaced `./pose` with `pose`, and neither side
uses the legacy form today; `add native project-state artifact` is present in
the pt-BR `pose-feature`. Only this one was real.

### Constraints
- Keep the translation's own format. The English skills were rewritten into a
  terse Codex-native shape while the translations kept an earlier, example-rich
  one; changing that is an editorial decision, not part of fixing a gate.

### Non-goals
- Reconciling the format difference between English and translated skills.
- A parity check for skills: the existing manual-parity check compares
  technical tokens symmetrically, which would fight the deliberate format
  asymmetry rather than catch drift. Recorded as a follow-up.

---

## 2. Requirements

### Functional
- R1: The translated skill shall instruct recording an immutable review
  attempt with `pose review record ... --apply`.
- R2: It shall require `pose review-check spec:<slug>` to report a fresh,
  approved review before any lifecycle transition.
- R3: It shall apply the transition through `pose close spec:<slug>`, with the
  manual frontmatter edit as the exception that still preserves the gate.
- R4: Its output requirements and anti-patterns shall name the gate, so the
  failure mode is recognisable from the skill itself.

### Non-functional
- No format change: the translation keeps its numbered steps with bash blocks.

### Security
- None; documentation of an existing gate.

### Compatibility
- `skills-check --strict` must keep passing for the translated mirror.

---

## 3. Technical Plan

### Affected areas
- `locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md` — Steps, Output
  requirements, Anti-patterns.

### Artifacts
- created: .pose/specs/pose-skill-closeout-gate-parity/spec.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md

### Delivery targets
- governance:skill-closeout-gate-parity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- Nothing detects the next divergence of this kind. `skills-check` validates
  the translated skill's contract — metadata, links, unsafe instructions — and
  passed throughout, because a skill can be perfectly conformant and still teach
  a workflow that no longer exists. That is the same shape as the manuals, and
  it is left open deliberately rather than solved with a check that would fight
  the format asymmetry.

---

## 4. Tasks

### Planning
- [x] Separate i18n-origin commits from real content drift across all 11 skills
- [x] Verify each candidate against the files rather than trusting the log

### Implementation
- [x] R1: record the review attempt
- [x] R2: require the gate before transition
- [x] R3: transition through `pose close`
- [x] R4: name the gate in outputs and anti-patterns

### Validation
- [x] `skills-check --strict`
- [x] Command parity across all 11 skill pairs
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Prose and format differ between the two versions by design, so the comparison
that matters is which POSE commands each side teaches. That ignores wording and
structure and isolates the thing an agent actually executes — and it is the
measure that found this gap and cleared the other ten skills.

### Deterministic checks

#### Security / Contract
- Command: `pose skills-check --strict`
- Scope: contract conformance of every skill including translated mirrors
- Expected: pass

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64.
- Notes: before the fix, the English `pose-spec-closeout` referenced
  `review record`, `review-check` and `close`; the translation referenced none.
  After it, the command-set comparison across all eleven pairs reports zero
  commands documented in English and absent from the translation.
  `skills-check --strict` reports 0 errors and 0 warnings.

### Results summary
- Successes: the gate is taught in both versions; all 11 skill pairs clear the
  command-parity comparison
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:skill-closeout-gate-parity evidence:integration check:delivery-integration report:locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md — step 2 records the attempt with `--apply` and states that the default is dry-run and that independence is `same-actor-separate-execution`
- R2 [satisfied] check:skills-check — step 3 requires `pose review-check spec:<slug>` with `fresh` and `approved`, and says stale attempts are superseded rather than edited
- R3 [satisfied] check:skills-check — step 5 applies `pose close spec:<slug>`, with the manual edit confined to the Git-workflow exception
- R4 [satisfied] report:locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md — the output requirements demand a fresh approved review, and the anti-patterns name the exact `pose check --strict` error that hand-editing produces

### Known gaps
- No check prevents the next skill from drifting the same way. `skills-check`
  passed throughout this defect.
- The format divergence between English and translated skills is untouched and
  undecided.

---

## 7. Final Report

### Delivered scope
The translated closeout skill routes through the review gate it must pass, and
names the failure that skipping it produces.

### Files and modules changed
- locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md

### Validation executed
- Command: `pose skills-check --strict`; command-set comparison over 11 skill pairs
- Result: pass; zero English commands missing from any translation

### Residual risks
- The class is closed for this instance, not for the next one.

### Follow-ups

- [open] Nothing detects a translated skill that teaches a workflow the engine no longer has: `skills-check` validates contract conformance and passed through this entire defect. A parity check would have to compare only the POSE commands each side teaches — the token-level comparison used for the manuals would fight the deliberate format asymmetry between terse English skills and example-rich translations. (owner:@pose-maintainers crit:medium review:2026-10-08)
- [open] Decide the format question the drift exposed: English skills were rewritten terse (Codex-native, no code blocks) while translations kept an earlier example-rich shape, so the translations carry more concrete commands than the originals. Either the translations should be condensed to match, or the English skills lost detail worth restoring — but the two should not stay divergent by accident. (owner:@pose-maintainers crit:low review:2026-11-06)
