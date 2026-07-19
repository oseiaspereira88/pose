# ADR: Polyglot stack catalog with priority resolution

## Status
Accepted (2026-07-19) — spec `pose-stack-catalog-expansion`

## Context

The baseline validation matrix covered Node.js, Go, Rust and Java; Python and
.NET repositories had no low-level setup path. Python specifically has
several competing package managers (poetry, pipenv, pip, setuptools,
PEP 517) whose markers can coexist in one directory — detection must resolve
that deterministically, never execute project files (setup.py, pyproject.toml
build backends) and never download dependencies implicitly. The existing
`when.fileExists`/`when.fileNotExists` predicates support only single-file
conditions, insufficient to express "yield to any higher-priority manager."

Alternatives considered:

1. **Execute the project's own tooling to detect the manager** (e.g. shell
   out to `python -c "import tomllib..."`) — violates the non-goal and the
   security requirement to never execute project files during detection.
2. **One matrix check per manager with no conflict handling** — silently
   runs redundant or wrong checks when multiple markers coexist.
3. **Marker-based profile catalog with declared priority, resolved via an
   additive `when` predicate extension.**

## Decision

Option 3:

- **Profile catalog** (`stackCatalog`, `internal/cli/stack_catalog.go`):
  each profile declares stack, manager, marker (exact filename or `*.ext`
  suffix), prerequisite native tool and a priority. Node/Go/Rust/Java are
  included unchanged (single marker, existing behavior preserved) so
  `pose stacks` reports one complete catalog, not a partial one.
- **Python profiles** (priority order): poetry (`poetry.lock`) → pipenv
  (`Pipfile`) → pip (`requirements.txt`) → setuptools (`setup.py`) →
  generic PEP 517 (`pyproject.toml`, lowest confidence, `optional`
  severity since it is the least specific signal).
- **.NET profile:** single manager (`dotnet` CLI), markers are suffix
  patterns (`*.sln`, `*.csproj`, `*.fsproj`, `*.vbproj`); `dotnet test` runs
  unconditionally when any marker is present (multi-project directories
  without a solution file are a documented dotnet-CLI limitation, not a
  POSE gap).
- **Matrix extension:** `validationWhen.FileNotExistsAny []string`
  (additive to the existing single-field predicates) lets a lower-priority
  check declare "skip if any of these higher-priority markers exist" —
  `pip-test` yields to `poetry.lock`/`Pipfile` this way, mirroring the
  existing Java gradle/maven `fileExists`/`fileNotExists` pattern.
- **Detection command (R1/R2):** `pose stacks [--path dir] [--json]` is
  read-only and offline — it lists directory entries and matches markers
  (`os.ReadDir` + suffix/name comparison), checks the prerequisite with
  `exec.LookPath` (never runs it), and reports `winner`/`shadowed` with
  `confidence: medium` whenever more than one profile matches the same
  stack. It never mutates the matrix; the override path
  (`moduleOverrides`) is printed as the documented escape hatch.
- **Fixtures (R3):** absent-tool (`LookPath` false), multiple-manager
  conflict (poetry + pip markers together resolve to poetry, confidence
  medium) and the matrix-level exclusion behavior are covered by tests.

## Consequences

- Positive: Python and .NET repositories get maintained, offline,
  deterministic detection without executing untrusted project code or
  implicit installs; manager conflicts are visible and resolved, not
  silently guessed.
- Positive: the `FileNotExistsAny` extension is reusable for any future
  stack with competing markers, without repeating single-field predicates.
- Trade-off: PEP 517-only repositories (no lockfile) get the lowest
  confidence and an `optional` default check — expensive or wrong defaults
  in large repos are mitigated by keeping this tier non-blocking and
  override-friendly, per the spec's stated risk.
- Residual: framework-level certification is explicitly out of scope; the
  catalog covers marker-level manager detection only.
