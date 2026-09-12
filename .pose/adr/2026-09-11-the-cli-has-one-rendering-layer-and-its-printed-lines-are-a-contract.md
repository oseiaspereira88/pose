# ADR: The CLI has one rendering layer, and its printed lines are a contract

## Status
Accepted (2026-09-11) — implemented by spec `pose-cli-output-rendering-system`.
Baseline measurements in knowledge note `cli-output-design-taxonomy`.

## Context

POSE's CLI prints from **1138 `fmt.Fprint*` call sites** with no layer in
between: no writer wrapper, no severity type, no formatter. Measured on `main`
at 2ff75f1 (POSE 5.0.6):

- **zero ANSI escapes**, no TTY detection, no width or wrapping, no `--quiet`,
  no progress of any kind; five symbols in the whole tree; `text/tabwriter` used
  twice;
- **three severity dialects at once** — `[ERROR]/[WARN]/[INFO]` (77 sites),
  `Error:` (116), `[pose-install]` (6) — and the channel each uses is not a
  rule: 27 severity-prefixed lines go to stdout and **43 to stderr**, findings
  included (`lint-spec`'s DoR errors, `knowledge-usage`'s unresolved
  references), so `pose x 2>/dev/null` drops real findings in some commands and
  not others;
- **28% of print sites are localisable** (315/1138); the rest print English
  whatever `POSE_LOCALE` says;
- **`--json` means two things**: a boolean printing to stdout in most commands,
  a file path in `validate`; eight commands — `check`, `history-check`,
  `knowledge-check`, `skills-check`, `recurrence-check`, `lint-spec`, `index`,
  `state` — have no machine channel at all;
- **the printed text is already an interface**: `pose report` parses
  `validate`'s printed lines (`parseValidationLines`) to derive a recorded
  outcome, and **144 assertions across 37 test files** match literal output;
- `pose validate` streams each check's raw output for minutes and prints **no
  per-check outcome, duration or counter** — those exist only in the JSON.

Two readers consume all of this: humans watching a gate, and machines (CI,
scripts, and the agents POSE exists to govern). The engine ships linux, darwin
and windows binaries, and `pose-cli-universal-help-and-subcommand-introspection`
already settled that the CLI stays on the standard library with no framework and
with bilingual parity enforced by tests.

Standardising 1138 call sites by convention has no enforcement point, and
decorating strings in place would push escape sequences into evidence. The
question this ADR answers is therefore not "which colours" but **where output is
produced, and what about it is promised**.

## Decision

**1. One renderer owns every byte a command prints.** A `cliout` package holds
the writers; commands emit semantic events — `Verdict`, `Finding`, `Step`,
`Field`, `Table`, `Section`, `Hint` — and never format severities, symbols,
colours or alignment themselves. A guard test forbids direct `fmt.Fprint*` in
command code against an allowlist that may only shrink, which is how POSE
already keeps the help catalog and the tool catalog from eroding.

**2. Decoration happens at the edge; the data stays plain.** One capability
profile per process resolves TTY-ness, colour, unicode, width and locale from
`--color`/`POSE_COLOR`, `NO_COLOR`, `TERM`, `COLUMNS` and `os.Stdout.Stat()`.
Colour, symbols and animation are applied while writing to a terminal. Nothing
written to a file, a JSON document, a report or any evidence record ever carries
an escape sequence.

**3. Channels are a rule, not a habit.** stdout carries the command's result —
its verdict, findings and data. stderr carries progress, usage, and the failures
of the command itself. The 43 misplaced sites move, which is a visible change for
anyone redirecting one stream.

**4. Printed lines are a contract, and the contract is enumerated.** The lines
machines depend on — `Result:`/`Resultado:` verdicts, `name.field=value`
diagnostics, finding lines — are inventoried and pinned by golden tests.
Everything else is prose the renderer may restyle between releases. Internal
consumers move to structured input before the prose they read changes: `pose
report` reads `validate`'s structured result rather than parsing its output.

**5. The machine channel is uniform.** `--json` is a boolean that prints one JSON
document to stdout, for every command with a result, including the eight that
have none today. `--json-out <path>` writes a file, alongside the existing
`--junit` and `--sarif`; `validate --json <path>` keeps working as a deprecated
alias. `--quiet` prints the verdict only. Exit codes are documented and tested:
`0` success, `1` gate failure, `2` usage error. A finding carries the same
fields in both channels — code, severity, path, message, remediation — plus the
message id.

**6. Messages live in a bilingual catalog keyed by id**, the way help already
does, so parity is a test rather than a promise and machines have a stable key
that survives rewording. Lines the work touches reach 100% parity; the rest is
tracked, not silently left behind.

**7. The renderer owns a child process's output while a step is active.**
`pose validate` captures a check's output instead of streaming it, prints the
tail on failure with a pointer to the full result, and restores streaming under
`--verbose`. Per-check outcome, duration and exit code are always printed.

**8. Standard library only.** No new direct dependency: TTY detection via
`os.Stdout.Stat()`, width from `COLUMNS` with an 80-column fallback, and legacy
Windows consoles get the ASCII/no-colour profile rather than a syscall to enable
virtual terminal processing. Real terminal size and Windows ANSI enabling stay
open as a later decision, with `golang.org/x/sys` (already an indirect
dependency) as the candidate.

**9. Animation is a terminal affordance, never a data channel.** A spinner,
elapsed time and counters exist only on a TTY, and are suppressed by `--quiet`,
`NO_COLOR`, `TERM=dumb` and redirection. Outside a TTY the same facts arrive as
one line per event.

## Alternatives rejected

- **Adopt a CLI/TUI framework** (cobra, lipgloss, bubbletea) — rejected: the help
  spec already settled stdlib-only, and a TUI dependency would own the process's
  terminal, the thing gates must not depend on.
- **Standardise by convention, keeping per-command formatting** — rejected: 1138
  sites with no enforcement point drift back within a release; POSE's own
  precedent is that a contract without a test is a suggestion.
- **Colour and symbols inside message strings** — rejected: it puts escapes into
  reports, JSON and evidence, and makes a non-TTY profile impossible.
- **Treat all output as free-form and let consumers adapt** — rejected: `pose
  report` and CI parse it today; the failure would be silent.
- **Animate whenever it looks nice** — rejected: repainting lines corrupt CI logs
  and are noise to an agent; a fact that exists only in a repainted line is
  inaccessible to a screen reader.
- **JSON only, humans read `jq`** — rejected: the gates run for minutes and the
  human channel is where POSE explains itself; the note's whole premise is that
  both readers are first-class.
- **Keep `validate` streaming and add a spinner anyway** — rejected: two writers
  on one terminal corrupt each other. Whoever owns the child's output owns the
  progress line.

## Consequences

- Positive: presentation becomes one reviewable surface with one vocabulary; a
  new severity or state cannot ship without a defined look.
- Positive: the machine channel becomes usable without prose parsing, and the
  eight gates without JSON gain it.
- Positive: bilingual parity and no-ANSI-in-evidence become tests instead of
  review diligence.
- Positive: long runs report what they are doing, which is the complaint that
  started this.
- Trade-off: **144 output assertions move**, and they are the regression net;
  they migrate deliberately, not by regeneration.
- Trade-off: the channel fix and the `validate` default are behaviour changes for
  scripts that redirect streams or scrape streamed child output. They land in one
  release, announced in its notes.
- Trade-off: the Markdown report is assembled from the printed run, so its shape
  moves with the restyle; the validation JSON moves only if capture changes, and
  the history JSONL, which stores no output, does not move at all.
- Trade-off: a catalog plus a capability profile is more machinery than
  `Fprintf`, and every new message costs two strings.
- Neutral: `--json`'s file-path spelling in `validate` survives as a deprecated
  alias, so no script breaks on upgrade.

## Review triggers

Revisit this decision if the renderer starts needing to own the terminal
(interactive prompts, full-screen panels, concurrent step execution), if real
terminal width or Windows ANSI enabling becomes necessary enough to take a direct
dependency, if a third output channel appears (structured logs, OTLP events), or
if the contract-line inventory proves too coarse — a consumer breaking on a line
we believed was prose.
