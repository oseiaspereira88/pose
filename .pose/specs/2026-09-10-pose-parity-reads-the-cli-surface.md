---
slug: pose-parity-reads-the-cli-surface
status: in-progress
completed_at:
created_at: 2026-09-10
supersedes:
depends_on: pose-skill-command-parity
priority: 0
components: pose-mcp
task_type: refactor
delivers:
---

# Spec: Locale parity reads the commands from the CLI

## 1. Intent

### Goal
Both locale parity checks shall learn what a pose command is from the CLI's own
dispatch, instead of from a list in a test file or from how a manual happens to
be formatted.

### Business value
The skill parity check kept a hand-written list of the eight commands that take
a subcommand, and read any other `pose <cmd> <word>` as the command alone. The
CLI has thirteen. `contribute` was missing — so a translation teaching
`contribute submit` where the English skill teaches `contribute stage` read as
`contribute` on both sides and passed, while the skills use `pose contribute
stage` twenty-three times. `roadmap` was on the list and is not a command at all.

The manual parity check recognised a bare word as a command only if one of the
manuals listed it as a `- \`name\`` entry. Seven commands in AGENTS.md —
`assess`, `close`, `extension`, `install`, `suggest`, `update`, `validate` —
appear only in prose there, so a translation dropping any of them passed.

Both fail quiet, which is the wrong direction for a guard: it keeps passing
while it checks less.

### Constraints
- The parity checks live in `scaffold`, which the CLI imports, so they cannot
  import the CLI. The dispatch is read from its source instead.

### Non-goals
- Completing `pose help`. Thirty dispatched commands have no help catalog entry,
  which is why the catalog could not be the source; writing thirty bilingual
  entries is a documentation change, not this one.

---

## 2. Requirements

### Functional
- R1: The set of commands shall be the case labels of the CLI's dispatch.
- R2: A command shall be a group when its handler switches on its first
  argument, and its subcommands shall be that switch's word labels.
- R3: The skill parity check shall keep a second word only when the CLI
  dispatches on it.
- R4: The manual parity check shall count every CLI command as technical,
  without losing the list-entry identifiers it already counted.
- R5: An extraction that stops matching shall fail the test rather than empty
  the sets.

### Non-functional
- No shipped file changes; both checks pass against the current manuals and
  skills.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/scaffold/` — the two parity checks and the locale coverage
  check that shares the skill comparison

### Artifacts
- created: .pose/specs/2026-09-10-pose-parity-reads-the-cli-surface.md
- renamed: .pose/changelogs/unreleased/pose-parity-reads-the-cli-surface.md -> .pose/changelogs/v5.0.1/pose-parity-reads-the-cli-surface.md
- created: pose-mcp/internal/scaffold/cli_surface_test.go
- modified: pose-mcp/internal/scaffold/skill_locale_parity_test.go
- modified: pose-mcp/internal/scaffold/manual_locale_parity_test.go
- modified: pose-mcp/internal/scaffold/locale_coverage_test.go
- modified: .pose/specs/2026-08-08-pose-skill-command-parity.md
- modified: .pose/specs/2026-08-08-pose-manual-locale-parity.md

### Technical risks
- A handler that picks its subcommand some other way — `switch
  strings.ToLower(args[0])`, or a map lookup — drops out of the group set, and
  that command is under-reported again. The floor catches the extraction
  breaking wholesale, not one handler changing style.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Read commands and groups from the CLI's source (R1, R2, R5)
- [x] Increment 2: Skill parity keeps only dispatched second words (R3)
- [x] Increment 3: Manual parity joins CLI commands with list entries (R4)

### Validation
- [x] Measure what each check gained and lost against the current files

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the follow-up proposed deriving the list from "the CLI's own dispatch
  table". There is no table: `mainCommand` is a switch, and each group handles
  its subcommands in its own function. The help catalog declares subcommands,
  but for eight commands, and has no entry at all for thirty.
- Decision: parse the CLI's source — the case labels of the dispatch, and the
  labels of each handler's switch on its first argument.
- Rationale: it is the code that decides what runs, so it cannot disagree with
  the CLI the way a list or the catalog can. Parsing rather than importing is
  forced by `cli` importing `scaffold`.

### Decision 2
- Date: 2026-09-10
- Context: replacing the manuals' list entries with the CLI's commands gained
  seven commands in AGENTS.md and lost four tokens in POSE.md: `module`,
  `optional`, `pose`, `required`. The list-entry rule had been catching keys,
  not only commands.
- Decision: use the union.
- Rationale: the list entries were never only a command source, and dropping
  them would have traded one blind spot for another. Measured rather than
  assumed: the first version of this change replaced them and passed.

---

## 6. Validation

### Strategy
Assert both gaps closed, and measure the token sets before and after against
the shipped files.

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
- Notes: the extraction yields 68 commands and 13 groups; the groups match what
  each command prints when it rejects an unknown subcommand, including
  `release open-next`/`backfill` and `contribute submit`, which the help catalog
  omits. Against the shipped manuals, AGENTS.md went from 41 technical tokens to
  48 and POSE.md kept its 285. Every shipped skill and manual still passes, so
  nothing had drifted behind the old lists.

### Results summary
- Successes: R1–R5 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <loadCLISurface collects the case labels of every `switch cmd` in mainCommand, 68 commands>
- R2 [satisfied] <firstArgumentCases reads a handler's switch on args[0] or on a variable assigned from it; 13 groups>
- R3 [satisfied] <TestSkillParityRejectsAndAllows requires `contribute stage` against `contribute submit` to be reported, and `close spec-x` to stay `close`>
- R4 [satisfied] <TestManualParitySeesACommandDocumentedOnlyInProse requires a prose-only `suggest` to be reported and a `required` list entry to stay technical>
- R5 [satisfied] <loadCLISurface fails when `check`, `review` or `release` is absent, or when review or release yield no subcommands>

### Known gaps
- A handler that stops switching on its first argument drops its group from the
  set without failing, unless it is one of the two the floor names.

---

## 7. Final Report

### Summary
The parity checks ask the CLI what a command is. The skill check had been
missing six of the thirteen groups, and listed one command that does not exist.

### Follow-ups
