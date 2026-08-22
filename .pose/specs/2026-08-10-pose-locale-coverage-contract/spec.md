---
slug: pose-locale-coverage-contract
status: done
created_at: 2026-08-10
completed_at: 2026-08-10
supersedes:
depends_on: pose-skill-index-parity
priority: 1
components: pose-mcp, docs
delivers: governance:locale-coverage-contract
---

# Spec: Locale parity stops being a list of documents someone remembered

## 1. Intent

### Goal
Make the unit of translation coverage the locale tree rather than a named
document, so a translated file is guarded by default instead of guarded once
someone notices it drifted.

### Business value
Parity was extended three times and each extension was triggered the same way:
a human read a document and saw it was wrong. Manuals were guarded after
`POSE.md` drifted, skill entries after the skills drifted, the skill index after
a user reported two missing rows. Every fix guarded the document that had just
failed and left the next one uncovered.

The fourth report named six more files at once — `.pose/workflows/review.md`,
four `SKILL.md` files and `ui-surface.md` — which is the point at which the
pattern stops being a series of defects and becomes one defect about how
coverage is decided. Twenty-four translated files across rules, workflows and
templates had no gate at all.

Guarding those twenty-four would have been the fourth instance of the same
mistake. What must change is the default: a file under `locales/` that nobody
declared how to compare is a failure, not a silence.

### Constraints
- Prose cannot be compared. Languages differ in length and the skills differ in
  shape by design — terse English, example-rich pt-BR — so asserting textual
  sameness would produce noise that trains people to ignore the gate.
- What can be compared is what both sides must agree on: the heading tree, and
  the POSE commands each side teaches.

### Non-goals
- Machine translation or automated synchronisation. The gate reports drift; a
  human resolves it, in both directions.

---

## 2. Requirements

### Functional
- R1: Every file under `locales/` shall be discovered by walking the tree, not
  by being listed in a test.
- R2: A translated file with no English source shall fail — a translation of a
  document that no longer exists is stale by construction.
- R3: A translated file with no declared comparison shall fail, naming the
  declaration it needs. Coverage is declared, never assumed.
- R4: A declaration for a file no locale translates shall fail, so coverage
  cannot shrink silently.
- R5: The comparison shall run in both directions: what the source teaches and
  the translation does not, and what the translation teaches and the source
  does not.

### Non-functional
- The gate runs offline in the existing scaffold suite.

### Security
- None; the change is a test and documentation corrections.

### Compatibility
- Instances receive the corrected translations on the next `pose upgrade`.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/scaffold/locale_coverage_test.go` — the gate.
- `pose-mcp/internal/scaffold/skill_locale_parity_test.go` — the command
  extractor, made tolerant of wrapped code spans.
- Four translated documents, one orphan translation, two English workflows.

### Artifacts
- created: .pose/specs/pose-locale-coverage-contract/spec.md
- created: pose-mcp/internal/scaffold/locale_coverage_test.go
- modified: pose-mcp/internal/scaffold/skill_locale_parity_test.go
- modified: .pose/workflows/review.md
- modified: .pose/workflows/feature.md
- modified: locales/pt-BR/.pose/workflows/review.md
- modified: locales/pt-BR/.pose/workflows/feature.md
- modified: locales/pt-BR/.pose/workflows/recurrence-escalation.md
- modified: locales/pt-BR/.pose/rules/delivery-evidence.md
- removed: locales/pt-BR/.pose/rules/kubernetes.md

### Delivery targets
- governance:locale-coverage-contract module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- The declaration map is still written by a human. Its defence is R3: a new
  translated file cannot reach an instance without someone deciding how it is
  compared, because the build fails until they do.

---

## 4. Tasks

### Planning
- [x] Establish why three prior parity extensions did not prevent the fourth report
- [x] Choose comparisons that survive translation without asserting sameness of prose

### Implementation
- [x] R1: discover by walking the locale tree
- [x] R2: fail on a translation with no source
- [x] R3: fail on an undeclared translated file
- [x] R4: fail on a declaration nothing translates
- [x] R5: compare in both directions
- [x] Correct every disparity the gate reported

### Validation
- [x] Prove each failure mode by injection, with a positive control
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
A parity gate that only passes proves nothing — that is exactly how the previous
three passed while twenty-four files drifted. So each failure mode is proven by
injecting it and confirming the gate names it, followed by a positive control on
the clean tree.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/scaffold/... -count=1`
- Scope: locale coverage, skill parity, manual parity, embed drift
- Expected: pass, with injected drift failing

### Execution log
- Date: 2026-08-10
- Environment: linux/amd64.
- The gate found eight disparities across four documents on its first run —
  fewer than the twenty-four uncovered files implied, which corrects the alarm
  in this spec's own framing: most translations were structurally sound and the
  size differences were prose.
- One finding was not drift but deletion. `locales/pt-BR/.pose/rules/kubernetes.md`
  had no English source: the rule was removed from the tree in `60a96f0` when it
  became a signed extension, and the translation stayed behind. It shipped to
  every instance for eight releases as a rule the engine no longer owns.
- Two findings ran the other way. The pt-BR review workflow named the MCP tools
  (`pose_tech_debt_check`, `pose_integration_check`) and the handoff command
  (`pose new-knowledge handoff`) that the English did not, and the pt-BR feature
  workflow taught `pose followups --all`. The English was corrected, not the
  translation — the reverse direction is a real source of findings, which is why
  R5 exists.
- The pt-BR review workflow also carried a duplicate section listing rules
  already listed above it, including the extension-only Kubernetes rule. Removed.
- The command extractor had a false negative that mattered more than any single
  document: it matched line by line, so a wrapped code span — `pose` ending one
  line and `surface-check` starting the next — read as untaught. A translator
  reflowing a paragraph disarmed the check silently. Normalising whitespace
  before matching fixed it and immediately surfaced a second real omission
  (`roadmap-check`) that the false negative had been hiding.

### Results summary
- Successes: coverage is now default-on for the locale tree; nine disparities
  corrected across six documents; one orphan translation withdrawn
- Failures: none outstanding
- Warnings: the declaration map is still authored by hand; R3 is what keeps it
  honest

### Requirement trace
- R1 [satisfied] governance:locale-coverage-contract evidence:integration check:delivery-integration test:pose-mcp/internal/scaffold/locale_coverage_test.go — the gate walks `locales/*` and fails if it compares nothing, so a broken discovery cannot masquerade as a clean tree
- R2 [satisfied] test:pose-mcp/internal/scaffold/locale_coverage_test.go — proven by injecting `locales/pt-BR/.pose/rules/orphan-probe.md` with no source; it is how the Kubernetes orphan was found
- R3 [satisfied] test:pose-mcp/internal/scaffold/locale_coverage_test.go — proven by adding a source and its translation and confirming the gate refuses the pair until a comparison is declared
- R4 [satisfied] test:pose-mcp/internal/scaffold/locale_coverage_test.go — the Kubernetes declaration had to be removed with the file, or the gate reported the stale declaration
- R5 [satisfied] test:pose-mcp/internal/scaffold/locale_coverage_test.go — proven by removing `pose review-check` from the translation; the reverse direction produced three of the nine real findings

### Known gaps
- Comparison is structural. A heading that keeps its level while its meaning
  drifts, or a command taught with the wrong flags, passes. The gate catches
  omission and divergence of shape, not mistranslation.
- Only `pt-BR` exists today. A second locale inherits the contract without
  further work, but has never been exercised.

---

## 7. Final Report

### Delivered scope
Translation coverage is now a property of the locale tree: every file is
discovered, must have an English source, and must have a declared comparison.
Nine disparities across six documents were corrected, in both directions, and an
orphan translation that had shipped for eight releases was withdrawn.

### Files and modules changed
- pose-mcp/internal/scaffold/locale_coverage_test.go
- pose-mcp/internal/scaffold/skill_locale_parity_test.go
- .pose/workflows/review.md, .pose/workflows/feature.md
- locales/pt-BR/.pose/workflows/{review,feature,recurrence-escalation}.md
- locales/pt-BR/.pose/rules/delivery-evidence.md
- locales/pt-BR/.pose/rules/kubernetes.md (deleted)

### Validation executed
- Command: the scaffold suite plus three injected failure modes
- Result: each injection reported; clean tree passes

### Residual risks
- Structural comparison does not read meaning. A wrong translation of a correct
  heading still passes.

### Follow-ups

- [open] The gate compares commands but ignores their flags, because a flag list differs legitimately between a terse source and an example-rich translation. `pose surface-check --strict` and `pose surface-check` are the same command to it, and only one of them is the doctrine. Decide whether flags that change a command's obligation — `--strict`, `--apply` — should be compared, and if so, how without reintroducing noise. (owner:@pose-maintainers crit:medium review:2026-10-10)
- [open] Only `pt-BR` exists, so the contract has never been exercised by a second locale. When one is added, confirm the declaration map does not need per-locale exceptions — if it does, the map is the wrong shape and should be per-locale rather than global. (owner:@pose-maintainers crit:low review:2026-12-10)
