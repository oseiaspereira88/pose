---
slug: pose-skill-index-parity
status: done
created_at: 2026-08-10
completed_at: 2026-08-10
supersedes:
depends_on: pose-skill-command-parity
priority: 1
components: pose-mcp
delivers: governance:skill-index-parity
---

# Spec: The translated skill index lists every skill

## 1. Intent

### Goal
Restore the two skills missing from the pt-BR skill index, and extend the
parity gate to cover the index itself.

### Business value
`.agents/skills/README.md` is the routing table an agent reads to discover which
skills exist. The pt-BR copy listed 11 of 13: `pose-surface-closeout` and
`pose-release-closeout` were absent, so a reader of the translated index had no
path to two skills that ship in the same directory.

`pose-skill-command-parity` was written for exactly this class and did not catch
it, because it compares `SKILL.md` files. A gate that checks the entries and
skips the index checks the wrong half — the entries can be perfect while nothing
points at them.

Reported from real use, not found by a gate: the reporter noticed the two rows
while reading both files.

### Constraints
- Compare the skills each index lists, not its prose. The tables carry
  translated descriptions and rule names by design.

### Non-goals
- Comparing the descriptions or rule columns.

---

## 2. Requirements

### Functional
- R1: The pt-BR skill index shall list every skill the English index lists.
- R2: A deterministic check shall fail, in both directions, when the two indexes
  list different skills.
- R3: It shall fail when no translated index is compared.

### Non-functional
- Runs in the existing suite over two files per locale.

### Security
- None.

### Compatibility
- Documentation only.

---

## 3. Technical Plan

### Affected areas
- `locales/pt-BR/.agents/skills/README.md` — the two rows.
- `pose-mcp/internal/scaffold/skill_locale_parity_test.go` — the index check.

### Artifacts
- created: .pose/specs/pose-skill-index-parity/spec.md
- modified: locales/pt-BR/.agents/skills/README.md
- modified: pose-mcp/internal/scaffold/skill_locale_parity_test.go

### Delivery targets
- governance:skill-index-parity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- The check compares linked skill slugs. An index that lists a skill with a
  broken link, or describes it wrongly, still passes — presence is not
  correctness.

---

## 4. Tasks

### Planning
- [x] Confirm the omission and establish why the existing gate missed it

### Implementation
- [x] R1: add the two rows to the translated index
- [x] R2: compare listed skills in both directions
- [x] R3: fail on an empty comparison

### Validation
- [x] Prove the check catches the omission that was reported
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
The proof is the reported defect itself: with the two rows removed again, the
check must name both skills. Anything less means it would have stayed silent
through the very report that prompted it.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/scaffold/ -run SkillIndexParity -count=1`
- Scope: the skill index per locale
- Expected: pass

### Execution log
- Date: 2026-08-10
- Environment: linux/amd64.
- Notes: both indexes now list 13 skills and the same set. Removing the two rows
  again makes the check report `omits 2 skill(s) … pose-release-closeout,
  pose-surface-closeout`, which is the defect as reported.

### Results summary
- Successes: index parity restored and guarded in both directions
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:skill-index-parity evidence:integration check:delivery-integration report:locales/pt-BR/.agents/skills/README.md — both indexes list the same 13 skills
- R2 [satisfied] test:pose-mcp/internal/scaffold/skill_locale_parity_test.go — omissions and extras are separate assertions; the omission case was proven against the reported defect
- R3 [satisfied] test:pose-mcp/internal/scaffold/skill_locale_parity_test.go — a run comparing nothing fails, stating the discovery is broken

### Known gaps
- Presence is not correctness: a listed skill with a broken link or a wrong
  description passes.
- Only `.agents/skills/README.md` is covered. Other translated index or routing
  documents are not compared.

---

## 7. Final Report

### Delivered scope
The pt-BR skill index lists every skill, and the parity gate covers the index in
addition to the entries.

### Files and modules changed
- locales/pt-BR/.agents/skills/README.md
- pose-mcp/internal/scaffold/skill_locale_parity_test.go

### Validation executed
- Command: the index parity test, plus the reported omission re-injected
- Result: pass; both skills named when omitted

### Residual risks
- The gate proves a skill is listed, not that the row is right.

### Follow-ups

- [open] The parity work has now been extended twice after a miss — first from manuals to skills, then from skill entries to the skill index — each time because someone noticed by reading. Rather than adding a check per document, consider enumerating the translated files under `locales/` and requiring each to have a declared comparison, so a new translated document is guarded by default instead of when it is missed. (owner:@pose-maintainers crit:medium review:2026-10-10)
