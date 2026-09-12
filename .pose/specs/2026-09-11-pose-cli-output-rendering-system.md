---
slug: pose-cli-output-rendering-system
status: in-progress
created_at: 2026-09-11
completed_at:
supersedes:
depends_on: pose-cli-universal-help-and-subcommand-introspection
priority: 1
components: pose-mcp, cli
task_type: feature
delivers: governance:cli-output-contract
---

# Spec: One rendering layer for the CLI, and a contract for what it prints

## 1. Intent

### Goal
Every byte the POSE CLI prints comes from one rendering layer with one
vocabulary; a human watching a gate sees what is running, what it decided and
what to do next; a machine reads the same facts from `--json` on every command
that has a result; and the lines machines already depend on are enumerated and
pinned instead of being incidental.

### Business value
The CLI prints from 1138 call sites with nothing in between, and the
consequences are measurable (knowledge note `cli-output-design-taxonomy`,
measured on `main` at 2ff75f1):

- **long gates say nothing.** `pose validate` streams each check's raw output for
  minutes and prints no per-check outcome, duration or counter at all; the
  outcome exists only in the JSON. This is the complaint that started the work.
- **the same class of output goes to either channel.** 27 severity-prefixed
  lines reach stdout and 43 reach stderr, findings included, so `pose x
  2>/dev/null` drops real findings in some commands and not in others.
- **three severity dialects coexist** (`[ERROR]/[WARN]/[INFO]`, `Error:`,
  `[pose-install]`), five symbols exist in the whole tree, and nothing aligns:
  `text/tabwriter` is used twice.
- **`--json` means two different things** (boolean to stdout; a file path in
  `validate`), and eight commands — `check`, `history-check`, `knowledge-check`,
  `skills-check`, `recurrence-check`, `lint-spec`, `index`, `state` — have no
  machine channel at all. POSE is built for agents that must not parse prose.
- **28% of print sites are localisable**, so a pt-BR instance reads a mixed
  interface, and `POSE_LOCALE` is documented in no user-facing document.

The architecture, and the alternatives rejected, are in ADR
`2026-09-11-the-cli-has-one-rendering-layer-and-its-printed-lines-are-a-contract`.

### Constraints
- Standard library only, no CLI or TUI framework, and no new direct dependency —
  the constraint `pose-cli-universal-help-and-subcommand-introspection` set.
- Bilingual parity is a gate: every line the work touches exists in en and
  pt-BR.
- No escape sequence may reach a file, a JSON document, a report or any evidence
  record.
- The printed text is already consumed: `pose report` parses `validate`'s lines,
  and 144 assertions in 37 test files match literal output. Consumers move to
  structured input before the prose they read changes.
- Windows binaries ship: legacy consoles get the ASCII/no-colour profile.

### Non-goals
- Rich output — trees, panels, metric bars, drift diffs (category E of the
  note). A separate spec, after this layer exists.
- Interactive prompts, full-screen TUI, or concurrent step execution.
- Migrating the ~820 English-only print sites the work does not touch; tracked as
  a follow-up.
- Rewording what commands say. This spec changes where output is produced and how
  it is presented, plus the additions R6–R9 name.

---

## 2. Requirements

### Functional
- R1: A `cliout` package shall resolve a capability profile — TTY, colour,
  unicode, width, locale — **per output stream**, from `--color`, `POSE_COLOR`,
  `NO_COLOR`, `TERM`, `COLUMNS` and that stream's own `Stat`, and shall own every
  byte the CLI prints. A stream that is not a terminal shall receive no escape
  sequence even when the other one is.
- R2: Commands shall emit semantic events — `Verdict`, `Finding`, `Step`,
  `Field`, `Table`, `Section`, `Hint` — and shall not format severities,
  symbols, colours or alignment themselves. A guard test shall fail on a direct
  `fmt.Fprint*` in command code outside an allowlist, and the allowlist shall
  only shrink.
- R3: One severity vocabulary shall map each state (`pass`, `fail`, `error`,
  `warning`, `skipped`, `info`, `hint`) to one look, with an ASCII fallback
  profile, and every state shall also be named in words so colour is never the
  only carrier.
- R4: stdout shall carry the command's result — verdict, findings, data — and
  stderr its progress, usage and its own failures. The 43 misplaced sites shall
  move.
- R5: The lines machines depend on — verdict lines, `name.field=value`
  diagnostics, finding lines — shall be enumerated as contract lines and pinned
  by golden tests; `pose report` shall read `validate`'s structured result
  instead of parsing its printed output.
- R6: A command that runs work shall emit a step per unit, resolved with outcome
  and duration. On a TTY the renderer shall show one repainting status line —
  spinner frame, current step, counter, elapsed — replaced by the resolved line;
  without a TTY it shall print one line per event and no escape sequence. No
  animation shall appear without a TTY, or under `--quiet`, `NO_COLOR` or
  `TERM=dumb`.
- R7: `pose validate` shall capture each check's output rather than streaming it,
  print the tail on failure with a pointer to the full result, and restore
  streaming under `--verbose`; the per-check outcome, duration and exit code
  shall always be printed.
- R8: `--json` shall print one JSON document to stdout for every command with a
  result, including the eight that have none today; `--json-out <path>` shall
  write a file; `validate --json <path>` shall keep working as a deprecated
  alias; `--quiet` shall print the verdict only; and the exit-code contract
  (`0` success, `1` gate failure, `2` usage error) shall be documented and
  tested.
- R9: A finding shall carry the same fields in both channels — code, severity,
  path, message, remediation, message id — and the human rendering shall show
  the remediation rather than dropping it.
- R10: Messages the renderer emits shall come from a bilingual catalog keyed by
  id, with parity enforced by a test, as `help_catalog.go` already does for help.
- R11: Unknown-flag and unknown-command errors shall have one shape across
  commands, name the offending token alone, and suggest the nearest valid name
  when one is close.
- R12: `docs-site/docs/cli.md` and POSE.md (both locales) shall document
  `--json`, `--json-out`, `--quiet`, `--verbose`, `--color`, `POSE_COLOR`,
  `NO_COLOR`, `POSE_LOCALE` and the exit-code contract.

### Non-functional
- Deterministic bytes for a given profile and input: two runs of the same command
  differ only in durations.
- No new direct dependency; stdlib only.
- The plain profile — no TTY — is the profile agents and CI see, and carries every
  fact the decorated profile does.

### Compatibility
- `validate --json <path>` stays valid; the channel fix (R4) and the `validate`
  capture default (R7) are behaviour changes announced in the release notes.

### Security
- Secret redaction already applied to captured output stays applied; the renderer
  never re-reads a child's raw buffer for display outside the redacted tail.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/cliout/` — the new rendering layer (profile, emitters,
  palette, symbols, catalog, step model)
- `pose-mcp/internal/cli/validate.go` — steps, capture, `--verbose`
- `pose-mcp/internal/cli/report.go` — structured input instead of
  `parseValidationLines`
- `pose-mcp/internal/cli/check.go`, `artifact_integrity.go`, `surface_check.go`,
  `lintspec.go`, `knowledge*.go`, `doctor.go`, `release_lifecycle.go` — the
  lifecycle core migrates first
- `pose-mcp/internal/cli/help_catalog.go` — the flags R8 adds
- `docs-site/docs/cli.md`, `POSE.md`, `locales/pt-BR/POSE.md` and the scaffold
  copies

### Delivery targets
- governance:cli-output-contract module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### Artifacts
- created: .pose/specs/2026-09-11-pose-cli-output-rendering-system.md
- created: .pose/adr/2026-09-11-the-cli-has-one-rendering-layer-and-its-printed-lines-are-a-contract.md
- created: pose-mcp/internal/cli/cliout/profile.go
- created: pose-mcp/internal/cli/cliout/state.go
- created: pose-mcp/internal/cli/cliout/catalog.go
- created: pose-mcp/internal/cli/cliout/render.go
- created: pose-mcp/internal/cli/cliout/steps.go
- created: pose-mcp/internal/cli/cliout/cliout_test.go
- created: pose-mcp/internal/cli/print_guard_test.go
- created: pose-mcp/internal/cli/testdata/direct-print-sites.json
- created: pose-mcp/internal/cli/output.go
- created: pose-mcp/internal/cli/contract_lines_test.go
- created: pose-mcp/internal/cli/testdata/contract-lines.json
- modified: pose-mcp/internal/cli/report.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/knowledge_usage.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/amend_test.go
- modified: pose-mcp/internal/cli/knowledge_usage_test.go
- created: pose-mcp/internal/cli/validate_progress_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/cli/workspace_alias_test.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/cli.go
- created: pose-mcp/internal/cli/cliout/record.go
- modified: pose-mcp/internal/cli/check.go
- modified: docs-site/docs/cli.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

Implementation artifacts are declared as each increment lands; the list above is
what this spec creates before code.

The changelog fragment is **not** created here. `releaseInputs` consumes any
fragment whose spec merely exists, so a cut taken before the work lands would
publish a release note for a feature that does not exist yet. It is written at
closeout, and it carries `breaking: true`: R4 moves findings between stdout and
stderr and R7 stops streaming a check's output, both of which a script can
depend on today (Decision 7).

### Technical risks
- The Markdown report is assembled from the printed run, so its shape moves with
  the restyle. The validation JSON moves only if capture changes; the history
  JSONL stores no output and does not move.
- A spinner and a child process writing to one terminal corrupt each other, which
  is why R7 (capture) is a precondition for R6 on `validate`.
- 144 literal-output assertions are both the net and the bill: they migrate
  deliberately, per increment, never by regeneration.

---

## 4. Tasks

### Implementation
- [x] Increment 1 — the layer, invisible: profile, emitters, vocabulary, catalog
      skeleton, guard test with a full allowlist. Output stays byte-identical,
      proven by the existing assertions (R1, R2, R3, R10).
- [x] Increment 2 — the contract: enumerate contract lines with golden tests,
      move `pose report` to structured input, fix the 43 misplaced sites (R4, R5).
- [~] Increment 3 — the human channel: symbols with ASCII fallback, palette,
      `tabwriter` alignment, prose wrapping, summary blocks, findings with their
      remediation, one error shape with suggestions (R3, R9, R11).
- [x] Increment 4 — long runs: step model, TTY status line with spinner, elapsed
      and counter, `validate` capture with `--verbose` and per-check outcome
      lines (R6, R7).
- [~] Increment 5 — the machine channel and the docs: the renderer records what
      it would print, so `--json` is one document built from the same events;
      `pose check` gains `--json`, `--quiet` and `--color`, and the docs and
      manuals describe the channels, the flags, the environment variables and
      the exit codes. The other seven gates and `--json-out` remain (R8, R12).

### Validation
- [x] Profile matrix: NO_COLOR, POSE_COLOR, --color, TERM=dumb, COLUMNS, locale
- [x] A test that a non-terminal run writes no escape sequence
- [x] Bilingual parity over the catalog, format verbs included
- [x] The ratchet falls in every increment (1138 → 1094)
- [ ] A golden suite per profile for every command in the lifecycle core

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: the note left five decisions open; the work was authorised without
  them being answered individually.
- Decision: proceed with the defaults recorded in Decisions 2–6, each stated so
  it can be overridden before the increment that depends on it starts.
- Rationale: increments 1 and 2 depend on none of the five, so the work is not
  blocked; each later increment names the decision it rests on.

### Decision 2
- Date: 2026-09-11
- Context: `validate` streams a check's raw output (decision 1 of the note).
- Decision: capture by default, print the tail on failure, `--verbose` to stream.
- Rationale: a spinner cannot share the terminal with a child process, and the
  captured tail is already what evidence records. What changes is the terminal
  and the Markdown report, not the validation JSON.

### Decision 3
- Date: 2026-09-11
- Context: `--json` means two things (decision 2 of the note).
- Decision: `--json` prints to stdout everywhere; `--json-out <path>` writes
  files; `validate --json <path>` stays as a deprecated alias.
- Rationale: the majority spelling wins, the minority keeps working, and no
  script breaks on upgrade.

### Decision 4
- Date: 2026-09-11
- Context: localisation is at 28% (decision 3 of the note).
- Decision: 100% parity for lines the work touches; the remainder becomes a
  tracked follow-up, not a silent gap.
- Rationale: parity is already a gate for new work; migrating ~820 untouched
  sites inside this spec would bury the layer it exists to build.

### Decision 5
- Date: 2026-09-11
- Context: 35 commands (decision 4 of the note).
- Decision: the layer covers every command; the visible migration starts with the
  lifecycle core (`check`, `validate`, `artifact-check`, `surface-check`,
  `release`, `doctor`, `lint-spec`), and the guard allowlist is what carries the
  rest, shrinking per increment.
- Rationale: the enforcement point lands everywhere at once; the restyle lands
  where a human spends time first, and the allowlist makes the remainder
  visible instead of forgotten.

### Decision 7
- Date: 2026-09-12
- Context: review of pose#110 — the fragment written with the spec advertises the
  feature before it exists, and `breaking: false` would classify the release that
  ships R4 and R7 as a patch.
- Decision: the fragment is created at closeout, with `breaking: true`.
- Rationale: `releaseInputs` does not require a spec to be `done`, so the
  fragment is a release note waiting to be published by any cut; and moving
  findings between streams while changing what `validate` streams is a change a
  script can break on, which is what the field is for. If staying non-breaking
  matters more than the default, R7 can ship behind a flag — that is a scope
  decision, not a relabelling.

### Decision 6
- Date: 2026-09-11
- Context: terminal width and Windows ANSI (decision 5 of the note).
- Decision: stdlib only — `COLUMNS` with an 80-column fallback, TTY via
  `os.Stdout.Stat()`, ASCII/no-colour profile on legacy Windows consoles. No
  direct dependency on `golang.org/x/sys`.
- Rationale: the no-dependency constraint is inherited, and a wrong width
  degrades a layout while a new dependency changes what the binary is. Revisit as
  its own decision if the ASCII profile proves insufficient on Windows.

---

## 6. Validation

### Strategy
Render the same commands under each capability profile and compare against golden
files; assert that nothing written to disk carries an escape sequence; hold the
machine channel to a schema; and prove the two behaviour changes (channels,
capture) with tests that fail against today's code.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

#### Validation
- Command: `pose validate --report`
- Scope: this instance
- Expected: exit 0, strict mode

### Execution log
- Date: 2026-09-12
- Environment: local, Go 1.26
- Notes: five commits, one per increment, each with the suite green.
  Increment 1 landed the layer with output byte-identical, proven by the 144
  existing assertions. Increment 2 moved `pose report` onto the run's structured
  result and pinned the contract lines; `lint-spec`'s 29 findings and
  `knowledge-usage`'s references moved from stderr to stdout. Increment 4 gave
  `validate` a step per check — with the output of a failing one shown as a tail
  and `--verbose` to stream — measured by a fixture with one passing and one
  failing check. Increment 3 gave `artifact-check`/`surface-check` findings
  their remediation and gave an unknown token one shape. Increment 5 taught the
  renderer to record what it would print, so `pose check --json` prints one
  document built from the same events.

### Results summary
- Successes: R1, R3, R4 (for the commands migrated), R5, R6, R7, R9, R11, and
  R8/R12 for `check` and `validate`.
- Failures: none.
- Remaining: R2's allowlist still holds 1094 direct print sites, and R8's
  machine channel covers two commands of nine. Both are follow-ups below, and
  the spec stays in-progress until they close.

### Requirement trace
- R1 [satisfied] <TestProfileIsResolvedPerStream and TestColourPrecedenceAndCapabilities: a buffer is never a terminal, and NO_COLOR/POSE_COLOR/--color/TERM/COLUMNS/POSE_LOCALE each decide what they own>
- R3 [satisfied] <TestStateVocabularyIsClosedAndAsciiCapable and TestColourNeverCarriesMeaningAlone: every state has a word, an ASCII symbol and a round-tripping key, and the word survives stripping the colour>
- R4 [satisfied] <TestFindingCarriesItsRemediationAndStaysOnStdout, TestHintAndFailureGoToStderr, and the migrated commands' tests now read findings from stdout>
- R5 [satisfied] <testdata/contract-lines.json with TestRendererKeepsTheContractLines, TestLegacyValidationLogsStillParse and TestReportReadsTheRunNotItsProse>
- R6 [satisfied] <TestStepsArePlainLinesWithoutATerminal, TestStepsRepaintOnlyOnATerminal, TestQuietSuppressesProgressEntirely, TestValidatePrintsAStepPerCheckAndCapturesTheirOutput>
- R7 [satisfied] <TestValidatePrintsAStepPerCheckAndCapturesTheirOutput and TestValidateVerboseStreamsTheCheckOutputAgain: captured by default, tail on failure, streamed on request, and the result file unchanged>
- R9 [satisfied] <TestFindingCarriesItsRemediationAndStaysOnStdout; artifact-check and surface-check pass the graph's fields through unflattened>
- R10 [satisfied] <TestCatalogParity: both languages and the same format verbs for every id>
- R11 [satisfied] <TestUnknownTokenSuggestsOnlyANeighbour and the unknown-command tests: the token alone, with a suggestion only when one is within a typo's distance>

### Known gaps
- R2, R8 and R12 are not traced: each is partly implemented, and the trace
  vocabulary is closed (`satisfied|waived|withdrawn|deferred-integration`) with
  no partial disposition. They are traced when the follow-ups below close, which
  is also when this spec can reach `done`.
- Seven gates still have no machine channel, and their findings still print
  directly: `history-check`, `knowledge-check`, `skills-check`,
  `recurrence-check`, `lint-spec`, `index` and `state`. `--json-out <path>` and
  the deprecation of `validate --json <path>` wait with them.
- 1094 direct print sites remain in the allowlist. The ratchet keeps them from
  growing; migrating them is mechanical and unfinished.
- The `[ERRO]`/`[AVISO]` anchor in `pose check` is gone from the human line.
  That was deliberate and is a breaking change: the untranslated anchor is the
  `severity` field of `--json`, and the human line now reads in the reader's
  language.

---

## 7. Final Report

### Summary
The CLI has a rendering layer, its printed lines have an enumerated contract,
long runs report what they are doing, and the machine channel is one document
built from the same events the human channel shows. Two commands of nine use
that channel so far.

### Follow-ups

- [open] Give the remaining seven gates a machine channel — `history-check`, `knowledge-check`, `skills-check`, `recurrence-check`, `lint-spec`, `index`, `state` — which means migrating their metric and finding lines to the renderer first, since a `--json` that prints an empty findings list would be worse than none (owner:unowned crit:medium review:2026-12-12)
- [open] Add `--json-out <path>` and deprecate `validate --json <path>` in the same release, so the file-writing spelling stays available while `--json` means stdout everywhere (owner:unowned crit:medium review:2026-12-12)
- [open] Migrate the 1094 direct print sites the allowlist still holds; the ratchet prevents growth but the layer is not the only writer until they are gone (owner:unowned crit:low review:2027-03-12)
- [open] Raise localisation parity beyond the lines this work touched: 28% of print sites were localisable when it started, and the catalog only covers what the renderer emits (owner:unowned crit:low review:2027-03-12)
