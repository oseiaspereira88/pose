---
slug: pose-manual-locale-parity
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-machinery-distribution-contract
priority: 1
components: pose-mcp
delivers: governance:manual-locale-parity
---

# Spec: A translated manual cannot describe a POSE that no longer exists

## 1. Intent

### Goal
Bring `locales/pt-BR/POSE.md` back into parity with `POSE.md`, carry the
rationale the translation preserved back into the English original, and add a
check that fails when either side drifts again.

### Business value
`POSE.md` and `AGENTS.md` are shipped machinery: `refreshManagedDocs` delivers
the manual matching an instance's own locale on every `pose upgrade`. A stale
translation is therefore not a documentation inconvenience — it is distributed,
every release, to every instance installed with `--locale pt-BR`.

Measured rather than assumed: 24 commits touched `POSE.md`, 11 touched the
pt-BR mirror. Thirteen feature commits between 2026-07-19 and 2026-08-03 never
crossed — the release lifecycle, `pose state` and its automatic refresh, docs
governance, capability reassessment triggers, the validation platform
(`--emit-plan`, `--junit`, `--sarif`, `--changed-from`), `amend` with its
requirement dispositions, and the signed extension ecosystem. Both files shared
the same last-touched commit, which made them look synchronised.

The clearest symptom was not a line count: the subsection documenting the CLI
had diverged in purpose, titled `### Command reference` in English and
`### Estado atual` in pt-BR — a name from before the restructure. `validate` was
one line in pt-BR against three paragraphs in English.

**The drift runs both ways.** The translation was not merely behind: it carried
rationale the English original never had or had lost — why `history-check`
exists ("without it, `recurrence-check` and `stats` diverge between machines"),
what `stats` is for, that `followups`' candidates are mechanical hints while the
semantic judgement belongs to the agent layer. `AGENTS.md`, which looked
perfectly in parity by commit count and line count, documented three real MCP
tools (`pose_component_discover`, `pose_integration_check`,
`pose_tech_debt_check`) in pt-BR and in no English file at all.

The class was already closed for translated skills — `skills_check.go` says so
in its own comment: "leaving the locale mirrors unchecked let a pt-BR skill
drift out of contract while the English one stayed green"
(`pose-machinery-distribution-contract` R3). The manuals were simply outside
that guard.

### Constraints
- Prose cannot be diffed across languages. The contract has to rest on things
  that are identical in every language: heading structure and technical
  identifiers.
- The check must be symmetric. A token present only in the translation is as
  much a defect as one missing from it — that is how the three MCP tools were
  found.

### Non-goals
- Machine-translating content. Each side keeps its own voice; only the
  technical surface is required to match.
- Checking instances. This governs what the distribution ships, not what a
  given repository has installed.

---

## 2. Requirements

### Functional
- R1: `locales/pt-BR/POSE.md` shall document every command, flag, path and
  config key that `POSE.md` documents.
- R2: The rationale that exists only in the translation shall be carried into
  the English manual, and the MCP tools documented only in pt-BR shall be added
  to `AGENTS.md`.
- R3: A deterministic check shall fail when a translated manual's heading tree
  differs from its source's.
- R4: The same check shall fail on any technical token present on one side and
  absent from the other, in either direction, naming the tokens.
- R5: It shall fail when no translated manual is compared, so broken locale
  discovery reports itself instead of passing vacuously.

### Non-functional
- Runs inside the existing Go suite over four files; no measurable cost.

### Security
- Read-only over repository files.

### Compatibility
- A locale that translates only some manuals stays valid: each manual is
  compared only where a translation exists.

---

## 3. Technical Plan

### Affected areas
- `locales/pt-BR/POSE.md` — the command reference, the spec-lifecycle section,
  key files, the structure tree and the CI policy.
- `POSE.md` — the rationale recovered from the translation.
- `AGENTS.md` — the three MCP tools.
- `pose-mcp/internal/scaffold/manual_locale_parity_test.go` — the check, beside
  the embedded-dist drift guard that governs the same shipped material.

### Artifacts
- created: .pose/specs/pose-manual-locale-parity/spec.md
- created: pose-mcp/internal/scaffold/manual_locale_parity_test.go
- modified: POSE.md
- modified: AGENTS.md
- modified: locales/pt-BR/POSE.md

### Delivery targets
- governance:manual-locale-parity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- **Identifier extraction is heuristic.** A bare lowercase word cannot be told
  apart from prose by any pattern — `check` is a command, `when` is not — so
  single-word commands are recovered from the manuals' own list entries rather
  than from a maintained list. A command that stops being documented as a list
  entry silently leaves the comparison.
- Placeholders inside `<...>` are prose in syntax (`<reason>` / `<motivo>`) and
  are excluded on both sides. That set is maintained by hand, and a new
  placeholder pair shows up as a false positive until it is added — noisy
  rather than silent, which is the correct direction, but still maintenance.

---

## 4. Tasks

### Planning
- [x] Measure the drift and its direction from the two files' commit histories
- [x] Identify what exists only in the translation before overwriting anything

### Implementation
- [x] R1: bring the pt-BR manual up to the English one
- [x] R2: carry the translation-only rationale and MCP tools into English
- [x] R3: compare heading trees
- [x] R4: compare technical tokens symmetrically
- [x] R5: fail on empty comparison

### Validation
- [x] Prove the check against the real drift, in both directions
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
A parity check written after a drift has to demonstrate it would have caught
that drift, so it is run against the actual pre-sync files rather than against
fixtures — in both directions, because the drift itself ran both ways.

The check's own construction needed the same scepticism. An early version
passed a manual it should have failed: fenced code blocks are three backticks
each, so pairing inline spans without removing the fences first silently offsets
every match, and the two manuals had different numbers of fences. A check that
passes for the wrong reason is worse than no check, and it was caught only by
running it against a file already known to be broken.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/scaffold/... -run ManualLocaleParity -count=1`
- Scope: heading trees and technical tokens across every translated manual
- Expected: pass

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64.
- Notes: against `HEAD:locales/pt-BR/POSE.md` — the manual before this sync —
  the check reports **76** technical tokens documented in English and absent
  from the translation. Against `HEAD:AGENTS.md` it reports the **3** MCP tools
  present only in pt-BR, proving the reverse direction. After the sync both
  manuals pass. pt-BR's technical-token count went from 235 to 403 against the
  English 399; heading trees were already identical and remain so.

### Results summary
- Successes: both directions of the real drift detected; both manuals in parity
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:manual-locale-parity evidence:integration check:delivery-integration test:pose-mcp/internal/scaffold/manual_locale_parity_test.go — the pt-BR manual carries every technical token the English one does, verified by the check passing where it previously reported 76 missing
- R2 [satisfied] report:POSE.md report:AGENTS.md — ten rationale passages recovered into English (`history-check`'s reason to exist, `stats`' purpose, `followups`' agent boundary, `hooks`' backup, `suggest`'s domains, `index`'s graph, `knowledge-check`'s rule, `new-knowledge`'s template, `recurrence-check`'s pass filter, `reports-housekeeping`'s defaults) and the three MCP tools added to `AGENTS.md`
- R3 [satisfied] test:pose-mcp/internal/scaffold/manual_locale_parity_test.go — heading shape is compared level by level and reports the counts on mismatch
- R4 [satisfied] test:pose-mcp/internal/scaffold/manual_locale_parity_test.go — both directions are asserted separately with distinct messages; the reverse direction is what surfaced the three MCP tools
- R5 [satisfied] test:pose-mcp/internal/scaffold/manual_locale_parity_test.go — a run that compares nothing fails, stating the locale discovery is broken rather than the manuals

### Known gaps
- Comments inside fenced usage blocks are stripped before comparison, because
  they are prose in the manual's own language. A flag documented *only* in such
  a comment is therefore invisible to the check — `--validate-output` is exactly
  that case in pt-BR today.
- The check governs `POSE.md` and `AGENTS.md`. Other translated material under
  `locales/` is not compared.
- Parity of tokens is not parity of meaning: a translation can name every
  identifier and still describe it wrongly.

---

## 7. Final Report

### Delivered scope
The pt-BR manual documents everything the English one does; the English manual
recovered the rationale and MCP tools that existed only in translation; and a
symmetric check fails on either kind of drift.

### Files and modules changed
- locales/pt-BR/POSE.md
- POSE.md
- AGENTS.md
- pose-mcp/internal/scaffold/manual_locale_parity_test.go

### Validation executed
- Command: `go -C pose-mcp test ./internal/scaffold/... -run ManualLocaleParity`
- Result: pass; 76 and 3 findings respectively when pointed at the pre-sync files

### Residual risks
- Token parity does not imply the translation is correct, only that it is
  complete on the technical surface.

### Follow-ups

- [open] A flag documented only inside a fenced usage comment is invisible to the parity check — `--validate-output` is that case in pt-BR and appears in no English file. Decide whether to document it properly in both manuals or to compare fence comments too, accepting the prose false positives that motivated stripping them. (owner:@pose-maintainers crit:low review:2026-11-06)
- [open] Single-word commands are recognised only while they appear as `- \`name\`` list entries in one of the manuals. A command documented solely in prose leaves the comparison without any signal. Consider deriving the command set from the CLI itself at test time rather than from the manuals' formatting. (owner:@pose-maintainers crit:low review:2026-11-06)
