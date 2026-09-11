---
slug: pose-report-help-names-every-flag
status: in-progress
completed_at:
created_at: 2026-09-10
supersedes:
depends_on: pose-manual-locale-parity
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: `pose report --help` names every flag it accepts

## 1. Intent

### Goal
`pose report --help` shall document every flag `pose report` accepts.

### Business value
The follow-up this answers found `--validate-output` documented only inside a
fenced comment of the pt-BR manual, invisible to the parity check, and asked
whether to document it in both manuals or compare fence comments. Measured
again, the flag is now in no manual, no help and no reference page.

It is not a minor flag. It names the validation log the report reads its
commands, results and derived outcome from — the outcome a report carries when
none is given — and it is refused if it points outside the project.

It was one of eleven. `pose report` accepts sixteen flags and its help named
four, plus `--since` in the usage line; `--change-from` and `--change-to`, which
record a spec's immutable change set, were among the missing. The help also gave
`--outcome` as `pass|fail|partial`, while the parser accepts `skipped` and
`unknown` as well.

### Constraints
- The manuals abbreviate `pose report` with `[...]`, deferring to its help.

### Non-goals
- Documenting every flag of every command. This holds one command to its
  parser; the same gap elsewhere is not measured here.
- Comparing fence comments in the manual parity check. With the flag documented
  where the manuals defer to, there is nothing there to compare.

---

## 2. Requirements

### Functional
- R1: `pose report --help` shall name every flag the parser accepts.
- R2: The `--outcome` values shown shall be the ones the parser accepts.
- R3: Every flag description shall exist in English and Portuguese.

### Non-functional
- The flags the parser accepts are declared once, and the test reads them from
  there.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/report.go` — the accepted flags, lifted to a package
  variable
- `pose-mcp/internal/cli/help_catalog.go` — the `report` help entry

### Artifacts
- created: .pose/specs/2026-09-10-pose-report-help-names-every-flag.md
- renamed: .pose/changelogs/unreleased/pose-report-help-names-every-flag.md -> .pose/changelogs/v5.0.1/pose-report-help-names-every-flag.md
- modified: pose-mcp/internal/cli/report.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/help_test.go
- modified: .pose/specs/2026-08-08-pose-manual-locale-parity.md

### Technical risks
- None: the parser's behaviour is unchanged; only where its flag set is declared
  moved.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Declare the accepted flags once (R1)
- [x] Increment 2: Document all of them in both languages (R1, R2, R3)

### Validation
- [x] The test fails against the previous help

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the follow-up offered two answers — document the flag in both
  manuals, or compare fence comments.
- Decision: neither; document it in `pose report --help`.
- Rationale: the manuals list `pose report` with `[...]` and do not enumerate its
  flags, so adding one would be arbitrary and would start drifting from the
  parser the day it merged. The help is bilingual by construction, already held
  to parity, and is where `[...]` sends a reader.

---

## 6. Validation

### Strategy
Hold the help to the parser, and prove the test catches the previous help.

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
- Date: 2026-09-10
- Environment: local, Go 1.26
- Notes: against the previous `report` help entry,
  TestReportHelpNamesEveryFlag reports 11 flags; with this one it passes, and
  TestBilingualHelpParity passes. The descriptions were written from what
  `cmdReport` does with each value, not from the flag names.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestReportHelpNamesEveryFlag runs `pose report --help` and requires every key of reportValueFlags, plus --git-stage, to appear; it reports 11 against the previous help>
- R2 [satisfied] <the usage line and --outcome entry read pass|fail|partial|skipped|unknown, the set cmdReport validates>
- R3 [satisfied] <TestBilingualHelpParity passes with every new FlagHelp carrying both descriptions>

### Known gaps
- Other commands' help is not held to their parsers.

---

## 7. Final Report

### Summary
Every flag `pose report` accepts is in its help, in both languages, and a new one
cannot be added without it.

### Follow-ups

- [open] Hold every command's help to the flags its parser accepts, not only `pose report`'s — measure how many commands accept flags their help does not name before deciding whether it is worth one check (owner:unowned crit:low review:2027-03-10)
