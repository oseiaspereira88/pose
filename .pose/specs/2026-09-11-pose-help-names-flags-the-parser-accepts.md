---
slug: pose-help-names-flags-the-parser-accepts
status: in-progress
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-report-help-names-every-flag
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: `pose <command> --help` names only flags the command accepts

## 1. Intent

### Goal
Five help entries shall stop advertising flags their commands refuse, and name
the ones they accept instead.

### Business value
Aligning the documentation with 5.0.2 meant checking every flag against the
parser, and the help catalog failed that check in the opposite direction from
`pose-report-help-names-every-flag`: not flags missing from help, but flags help
invents. Each was confirmed by running the command:

- `pose update --help` offered `--schema-only`; the command exits 2 with
  `Usage: pose update [--dry-run] [--force] [--no-self] [--locale tag]`, and help
  named neither `--no-self` nor `--locale`. POSE.md documented the same flag.
- `pose dora-metrics --help` offered `--app`, `--env` and `--since-days`; the
  command accepts `--application`, `--environment` and `--window-days`, with a
  30-day default rather than the 90 help stated.
- `pose adoption-metrics --help` offered `--since-days`; the command accepts only
  `--json`.
- `pose state --help` offered `--json`; the command refuses it.
- `pose release --help` showed `record` with `--provider-state`; the command
  takes `--event` and `--evidence`.

A help screen is where an operator learns a command. Each of these fails the
first time it is followed.

### Constraints
- The parser is the authority; help changes, the commands do not.

### Non-goals
- A guard holding every command's help to its parser. That is the open
  follow-up on `pose-report-help-names-every-flag`; extracting flag literals per
  command was too noisy to gate on, which is why these were confirmed by hand.

---

## 2. Requirements

### Functional
- R1: Each of the five help entries shall name only flags its command accepts.
- R2: `pose update --help` shall name `--no-self` and `--locale`, and
  `pose dora-metrics --help` the defaults the command applies.

### Non-functional
- Both locales stay complete (`TestBilingualHelpParity`).

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/help_catalog.go`

### Artifacts
- created: .pose/specs/2026-09-11-pose-help-names-flags-the-parser-accepts.md
- renamed: .pose/changelogs/unreleased/pose-help-names-flags-the-parser-accepts.md -> .pose/changelogs/v5.0.3/pose-help-names-flags-the-parser-accepts.md
- modified: pose-mcp/internal/cli/help_catalog.go

### Technical risks
- None: text only.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Correct the five entries in both languages (R1, R2)

### Validation
- [x] Run each command with the flags help used to name

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: `--force` on `update` read as "overwrite managed manuals wholesale",
  which looked wrong beside a plain update that already merges them.
- Decision: keep the claim and complete it.
- Rationale: measured, `--force` reruns `install --force`, which does reset the
  manuals wholesale; what help lacked was the scaffold, rule, workflow and MCP
  refresh it also performs.

---

## 6. Validation

### Strategy
Run each command with the flags its previous help named, and read the new help.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

### Execution log
- Date: 2026-09-11
- Environment: local, engine at main
- Notes: before the change, `update --schema-only`, `dora-metrics --since-days`,
  `adoption-metrics --since-days` and `state --json` each exited 2 with the
  command's own usage line, and `release record --provider-state` failed with
  `unexpected argument`. After it, each help entry matches that usage line, and
  the help and bilingual parity tests pass.

### Results summary
- Successes: R1, R2 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <the five entries now match the usage each command prints when it rejects an argument>
- R2 [satisfied] <`pose update --help` prints `pose update [--dry-run] [--force] [--no-self] [--locale <tag>]`; dora-metrics names production and 30 days>

### Known gaps
- Other entries are not held to their parsers.

---

## 7. Final Report

### Summary
Five help screens stop teaching flags that fail.

### Follow-ups
