---
type: note
slug: cli-output-design-taxonomy
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-09-12
last_reviewed_at: 2026-09-12
expires_at: 2026-12-11
source_refs:
  spec: ""
  workflow: documentation-update
  commands: ["pose check --strict", "pose knowledge-check --strict", "pose specs --json", "pose followups --json", "pose usage --json", "pose doctor --json", "pose review-plan spec:<slug> --json", "pose artifact-check --spec <slug> --json", "pose check --json", "pose lint-spec <slug> --json", "pose history-check --json", "pose knowledge-check --json", "pose skills-check --json", "pose recurrence-check --json", "pose index --json", "pose state --json"]
  external_sources: []
---

# note: cli-output-design-taxonomy

## Context

We want to raise the quality of what POSE's CLI prints, and to standardise it
across every command. This note groups the work into the categories the
discipline actually has, records what this engine does today — measured, not
assumed — and lists what each category would contain if we pursued it. It takes
no decisions: it exists so the spec and the ADRs that follow are written against
facts, and so the parts that need the maintainer's call are visible before code
is written.

**Why the 90-day TTL** (the knowledge rule allows it only with a recorded
reason): this note is the input to a spec and to a rendering-layer ADR, and it
stays the reference for the phases while they are implemented across releases. A
30-day expiry would retire it in the middle of that work. It should be reviewed
when the spec closes, and retired when the ADR and `docs-site/docs/cli.md` carry
the decisions.

The commands in `source_refs` are of two kinds, and are not interchangeable:
`pose check --strict` and `pose knowledge-check --strict` are the validation this
note ran; everything else was executed only to observe the surface it describes
(which commands accept `--json`, and what they print).

Two audiences read this output, and they are not the same reader:

- **humans**, who watch a gate run and need to know what is happening, what
  failed, and what to do next;
- **machines** — CI, scripts, and the agents POSE is built for — which need
  stable, parseable output.

A mature CLI treats those as separate channels. POSE currently has one and a
half: a human channel that machines already parse, and a `--json` channel that
covers some commands.

## Current state

Measured on `main` at 2ff75f1 (POSE 5.0.6), in `pose-mcp/internal/cli`
(70 non-test files, 35 top-level commands, 40 help-catalog entries).

**There is no presentation layer.**

- `fmt.Fprint*` is called from **1138 sites**; there is no intermediate writer,
  no severity type, no formatter.
- **0 ANSI escape sequences.** No colour, no bold, no dim.
- No TTY detection, no terminal-width or wrapping logic, no `--quiet`, no
  progress indicator of any kind.
- Symbols appear by accident: one `✓`, two `⚠`, two `💡` in the whole tree.
- `text/tabwriter` (stdlib) is used **twice**; every other aligned output is
  hand-spaced.

**Vocabularies coexist.** Three severity dialects are in use at once:

| Shape | Sites | Where it goes |
|---|---|---|
| `[ERROR]` / `[WARN]` / `[INFO]` | 37 / 14 / 26 | stdout **and** stderr (27 / 43 sites) — these are *findings*, the command's result |
| `Error: …` | 116 | stderr — the command itself failed |
| `[pose-install] …` | 6 | stderr — one command's private prefix |
| `Usage: …` | 134 | stderr |
| `Result: …` / `Resultado: …` | 54 / 22 | stdout — the verdict line |
| `name.field=value` | 50 | stdout — machine-oriented diagnostics |

**And the channel split is mixed, not merely undocumented.** Counting call
sites: 27 `[ERROR]`/`[WARN]`/`[INFO]` lines go to **stdout**, and 43 go to
**stderr** — including findings that are the command's result, such as
`lint-spec`'s DoR errors (`lintspec.go`) and `knowledge-usage`'s unresolved
references. So the same class of output lands on either channel depending on the
command, and `pose x 2>/dev/null` silently drops real findings in some commands
and not others. A rule is needed, and so is a migration: this is not a matter of
writing down what already holds.

**Localisation is 28% covered.** 315 of the 1138 print sites pass through
`cliText`; the rest print English regardless of `POSE_LOCALE`. `POSE_LOCALE`
itself is documented in no user-facing document — only inside a spec.

**Unknown-flag errors are inconsistent**, though the exit code (2) is not:

```
pose check --json          → Error: invalid argument: --json
pose lint-spec X --json    → Error: unknown option: --json
pose knowledge-check --json→ Usage: pose knowledge-check [--strict|--tolerant] …
pose specs --json          → (valid JSON on stdout)
pose <garbage> --flag      → Unknown command: <the whole argv, joined>
```

**`--json` means two different things.** 55 parse sites. In most commands it is
a boolean that prints JSON to stdout (`specs`, `followups`, `usage`, `doctor`,
`artifact-check`, `review-plan`). In `pose validate` it takes a **path** and
writes a file (as do `--junit` and `--sarif`). Several gates have no machine
channel at all: `check`, `history-check`, `knowledge-check`, `skills-check`,
`recurrence-check`, `lint-spec`, `index`, `state`.

**The human output is already an API.** `pose report` parses `pose validate`'s
printed lines (`parseValidationLines`) to derive the recorded outcome. So text we
restyle can break an internal consumer. **144 assertions across 37 test files**
match literal output text — both the safety net and the migration bill.

Three persistence paths exist, and they carry different things — worth keeping
apart, because they size the evidence risk differently:

| Path | What it stores | Exposed to a restyle? |
|---|---|---|
| `.pose/results/*.json` (validation result) | each check's own captured `Output` (tail, secrets redacted), outcome, duration, exit code | only if we change what we capture from the child, not how we print |
| `.pose/reports/*.md` (Markdown report) | the subset `parseValidationLines` extracts from the printed run — commands and result lines | yes: directly coupled to the printed shape |
| `.pose/reports/history/*.jsonl` | report metadata only (`reportRecord`: task, outcome, hash, change set…) — **no captured output** | no |

**`pose validate` is the sharpest human gap.** It pipes every check's raw
stdout/stderr straight to the terminal (`io.MultiWriter`), and prints **no
per-check outcome line at all**: the outcome, duration and exit code exist only
in the JSON and the report. A human watching it sees raw `go test` noise, then
`Result: SUCCESS`. Nothing says which check is running, how many remain, or how
long any of them took — and these runs are minutes long.

**Constraints that already apply.** `pose-cli-universal-help-and-subcommand-introspection`
settled: stdlib only, no CLI framework, bilingual parity, `-h`/`--help` exit 0.
Its `help_catalog.go` is the precedent worth copying — one structured, bilingual
source of truth, with parity enforced by tests. Releases ship **linux, darwin
and windows** (amd64 + arm64), so Windows console capability is in scope.

## Categories

### A. CLI Output Formatting — the mechanics

Colour, bold/dim, indentation, alignment, wrapping, tables.

*Today:* none of it. Plain `Fprintf`, hand-spaced columns, no wrapping.

*What it would contain:*

- one indentation scale (findings indented under their subject, two spaces per
  level) and one bullet/label shape;
- alignment through `text/tabwriter` for every list command (`specs`,
  `followups`, `usage`, `reports`, `release status`);
- wrapping of prose to `min(terminal width, 100)`, and **never** of contract
  lines (`Result:`, `name.field=value`, paths) — a wrapped path is a broken
  path;
- terminal width from `COLUMNS`, falling back to 80. Real width needs an ioctl
  (`golang.org/x/sys` is already an indirect dependency; promoting it is a
  decision, not a given);
- colour applied by the writer at the edge, never baked into a message string.

### B. CLI Output Styling — the identity

A semantic palette, symbols, visual hierarchy.

*Today:* accidental (five symbols in total), and no palette.

*What it would contain:*

- a closed semantic palette mapped to POSE's own vocabulary, not to ad-hoc
  colours: `pass`, `fail`, `error`, `warning`, `skipped`, `info`, `hint`,
  `path`, `identifier`, `muted`. POSE already has closed vocabularies for
  severities and evidence classes; the palette should be a projection of them,
  so a new finding severity cannot appear without a defined look;
- a symbol set with an ASCII fallback profile (`✔/✖/!/-/→` → `[ok]/[x]/[!]/[-]/->`),
  chosen by capability detection, not by guessing;
- **colour is never the only carrier**: every state also says its word, so
  `NO_COLOR`, pipes, screen readers and colour-blind readers lose nothing;
- 4-bit ANSI as the baseline (dumb terminals, CI log viewers), 256-colour only
  as an enhancement.

### C. Terminal UX — the experience (the emphasis)

Messages, progress, prompts, actionable errors, consistency.

*Today:* the gates run silently for minutes behind raw child output; there is no
elapsed time, no counter, no phase, no `--quiet`, no `--verbose`.

*What it would contain:*

**Progress for work that takes time** (`validate`, `release check`, the compat
gate, `update`, `doctor`, `index` on large trees):

- a **step model**: each unit of work is started, then resolved
  (`pass|fail|skip|error`) with its duration. The renderer decides how that
  looks;
- in a TTY: a single repainting status line — spinner frame, current check,
  counter, elapsed seconds — replaced in place by the resolved line;
- outside a TTY: exactly one line per event, no repaint, no escapes. Same
  information, greppable;
- **no animation without a TTY**, and none when `--quiet`, `NO_COLOR`,
  `TERM=dumb`, or when output is redirected.

A sketch of what `pose validate` could look like in a TTY, with the child's
output captured rather than streamed:

```
  pose-mcp (go, mode=standard)
  ⠹ test  go test -count=1 ./...            12s
```

resolving to:

```
  pose-mcp (go, mode=standard)
  ✔ test        go test -count=1 ./...      18.4s
  ✔ lint        gofmt -l .                   0.3s
  ✖ typecheck   go vet ./...                 2.1s   exit 1
      internal/cli/x.go:42: unreachable code
      … 3 more lines — full output in .pose/results/validate.json

  3 checks · 2 passed · 1 failed · 20.8s
  Result: FAILURE (required check failed)
```

and, with no TTY, to the same facts one line at a time:

```
[module] pose-mcp (go, mode=standard)
  -> test go test -count=1 ./...
  <- test pass 18.4s
  -> typecheck go vet ./...
  <- typecheck fail 2.1s exit=1
Result: FAILURE (required check failed)
```

**Failure output that is quiet on success.** The modern default for a runner is
to capture a check's output and show it only when the check fails (with a tail,
and a pointer to the full result file), leaving `--verbose` to stream
everything. This is the single biggest change to how POSE *feels*, and also the
one with a real consequence: what the report captures as evidence changes shape.

**Actionable errors.** POSE is already ahead here and does not exploit it: every
delivery-integrity finding carries `code`, `severity`, `path`, `message` and
`remediation`, and the human channel flattens that into one line built by hand
at each site. A rendered finding should read:

```
  ✖ action-mismatch  .pose/changelogs/unreleased/x.md
      declared artifact action is absent from the attributed Git change sets
      fix: correct the action or record the exact attributed revisions
      docs: https://docs.harne8.com/POSE/cli/#artifact-check
```

The same rule for the command's own failures: what happened, what POSE expected,
and the exact next command. And one shape for every "unknown flag" and "unknown
command" error, with a suggestion when the input is close to a real name.

**Consistency across commands.** One verdict line, one summary block, one
diagnostics section, in the same order, for all 35 commands — so a reader who
learns `check` already knows how to read `surface-check`.

### D. CLI Output Rendering — the architecture

The layer that turns structured data into terminal bytes.

*Today:* there is none; every command formats its own strings.

*What it would contain:*

- a `cliout` package: a renderer holding the **capability profile** (is a TTY,
  colour allowed, unicode allowed, width, locale) plus semantic emitters —
  `Verdict`, `Finding`, `Step`, `Field`, `Table`, `Hint`, `Section`;
- the rule that **data stays plain and decoration happens at the edge**: nothing
  written to a file, a JSON document, a report or an evidence record ever
  carries an escape sequence;
- a **message catalog with IDs**, the way `help_catalog.go` already does it for
  help: bilingual parity becomes a test rather than a promise, and each message
  gains a stable identifier a machine can key on;
- channel discipline as code: results and findings to stdout, progress and the
  command's own failures to stderr, so `pose x --json | jq` is always safe;
- a guard test that scans command sources for direct `fmt.Fprint*` calls with an
  explicit allowlist — the mechanism POSE already uses to keep contracts from
  eroding.

This is the part that needs an ADR, because it changes where every command's
output comes from, and because output is a consumed contract.

### E. Rich Terminal Output — the optional depth

Tables, trees, panels, bars, syntax highlighting.

*Today:* absent; plausible places already exist.

*What it would contain:*

- **trees** for the spec graph (`depends_on`), roadmaps and milestones, and for
  the delivery-integrity graph's reverse lookups;
- **tables** for `specs`, `followups`, `usage`, `release status`,
  `adoption-metrics`;
- **panels/summary blocks** for a release cut and for `doctor`;
- **bars or sparklines** for DORA and adoption metrics;
- **diff-style** rendering for drift (machinery manifest, vendored sources,
  managed-manual merges), which is where a human currently reads a wall of
  paths.

Everything here is an enhancement, gated behind the capability profile, and none
of it may appear in a line a machine parses.

### F. Machine-readable vs human-readable — two channels

*Today:* mixed. `--json` is boolean in most commands and a path in `validate`;
seven gates have no machine channel; the human channel is parsed internally.

*What it would contain:*

- `--json` means **JSON on stdout**, everywhere, for every command that has a
  result; `--json-out <path>` (plus the existing `--junit`, `--sarif`) writes a
  file. `validate`'s current `--json <path>` stays as a deprecated alias,
  because scripts and CI use it;
- the machine document carries everything the human channel shows — including
  each finding's `code`, `severity`, `remediation`, and each step's duration and
  exit code — so nobody has to parse prose;
- `--quiet`: the verdict and the exit code, nothing else;
- a documented **exit-code contract** (0 success, 1 gate failure, 2 usage error)
  — today's behaviour, written down and tested;
- the internal consumers move first: `pose report` should read `validate`'s
  structured result, not its prose, before the prose is allowed to change;
- `NO_COLOR`, `POSE_COLOR=auto|always|never` (and `--color`), `POSE_LOCALE` —
  all documented in the CLI reference and the manuals, which today mention none
  of them.

## Cross-cutting constraints

1. **Output is a contract.** Internal parsers, CI greps, 144 test assertions and
   agents read it. Sequence: identify the contract lines, give the consumers a
   structured source, then restyle.
2. **Evidence must stay clean.** No escape sequences in JSON, JUnit, SARIF or
   reports. The exposure is uneven (see the table above): the Markdown report is
   built from the printed run, so it moves with any restyle, while the validation
   JSON only moves if we change what we capture from a child process, and the
   history JSONL stores no output at all.
3. **Determinism.** Same inputs, same bytes, with timings the only variable —
   otherwise golden tests and reproducible reports suffer. Spinner frames never
   reach a non-TTY.
4. **No new dependencies**, per the help spec. Everything here is reachable with
   stdlib (`text/tabwriter`, `os.Stdout.Stat()` for TTY detection); only real
   terminal-size and Windows virtual-terminal enabling would want
   `golang.org/x/sys`, already present as an indirect dependency.
5. **Bilingual parity is a gate**, so every string doubles. A catalog makes that
   testable; raising coverage from 28% to 100% (~820 sites) is a decision of its
   own.
6. **Windows is shipped.** Legacy consoles need virtual-terminal enabling and
   may not render box drawing or emoji; the ASCII profile is not optional.
7. **Accessibility.** Never colour alone; no meaning that exists only in a
   repainted line; UTF-8 assumed only when detected.
8. **Agents are first-class readers.** For them the plain profile plus `--json`
   *is* the UX: a spinner is noise, an unstable line is a bug.

## Candidate work, in dependency order

| Phase | Work | Why it is where it is |
|---|---|---|
| 0 | Inventory the parsed contract: every line a consumer (internal, CI, tests) depends on, listed and pinned by golden tests | Nothing can be restyled safely before this exists |
| 1 | `cliout` rendering layer: capability profile, semantic emitters, single severity vocabulary, channel rules, message catalog, guard test | Everything else is a consumer of this |
| 2 | Human polish: one vocabulary, symbols with ASCII fallback, palette, tabwriter alignment, wrapping, summary blocks | Mechanical once the layer exists |
| 3 | Long-running UX: step model, spinner + elapsed + counter in a TTY, per-check outcome lines everywhere, quiet-on-success with `--verbose` | Depends on the decision about who owns a check's output |
| 4 | Machine channel: `--json` on stdout everywhere, `--json-out` for files, `--quiet`, exit-code contract, gates that have no JSON get it, internal parsers move to structured input | Must land before or with the restyle of the lines they parse |
| 5 | Rich output: trees, panels, metric bars, drift diffs | Pure enhancement; last |

## Decisions that need the maintainer

1. **Quiet-on-success for `validate`?** Capturing a check's output instead of
   streaming it is the biggest visible change, and it changes what reports
   capture as evidence. Keep streaming, capture-and-summarise, or make it a
   flag with a default to choose?
2. **`--json` semantics.** Standardise on stdout and demote `validate --json <path>`
   to a deprecated alias, or keep two spellings permanently?
3. **Localisation scope.** Raise parity to 100% as part of this work, or hold
   parity only for lines the work touches?
4. **Blast radius.** All 35 commands, or the lifecycle core first (`check`,
   `validate`, `artifact-check`, `surface-check`, `release`, `doctor`)?
5. **Terminal size and Windows.** Promote `golang.org/x/sys` to a direct
   dependency for real width and virtual-terminal enabling, or stay on
   `COLUMNS` + 80 columns + an ASCII profile?

## Next checks

- None yet: this note is the input to a spec. Once the spec exists, its own
  gates apply (`go test -count=1 ./...`, `pose check --strict`,
  `pose validate --report`).

## Risks

- **Contract erosion**: restyling a line an internal parser reads
  (`parseValidationLines`) or CI greps would break a gate silently. Mitigation:
  phase 0 and phase 4 before phase 2.
- **Test churn**: 144 literal-output assertions will move. They are also the
  regression net, so they should be migrated deliberately, not regenerated
  wholesale.
- **Evidence noise**: the Markdown report is assembled from the printed run, so
  restyling moves it; the validation JSON moves only if capture changes. Expected
  either way, but it should land in one release, announced.
- **Interleaving**: a spinner and a child process writing to the same terminal
  corrupt each other. Whoever owns the child's output owns the spinner.
- **Scope creep into a framework**: the goal is a thin renderer, not a TUI.
  Panels and trees are last for that reason.

## Next owner

@pose-maintainers — the five decisions above, then the spec and the ADR for the
rendering layer.

## References

- `pose-mcp/internal/cli/help_catalog.go` — the precedent: one structured,
  bilingual source of truth with parity enforced by tests.
- `.pose/specs/2026-08-21-pose-cli-universal-help-and-subcommand-introspection.md`
  — stdlib only, no CLI framework, bilingual parity.
- `pose-mcp/internal/cli/validate.go` — the streaming loop and the missing
  per-check outcome line.
- `pose-mcp/internal/cli/report.go` (`parseValidationLines`) — the internal
  consumer of human output.
- `pose-mcp/internal/pose/delivery_integrity.go` — findings already carry
  `code`/`severity`/`message`/`remediation`; the human channel flattens them.
- `docs-site/docs/cli.md` — the reference that would document `--json`,
  `--quiet`, `POSE_COLOR`, `NO_COLOR` and `POSE_LOCALE`.
