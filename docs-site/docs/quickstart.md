# Quickstart

**Doc type:** Tutorial &nbsp;·&nbsp; **Applies to:** POSE 5.x (current stable)

## Install

### Fast path — Linux and macOS

Run the installer from the root of the Git repository that should receive the
POSE contract:

```bash
curl -fsSLO https://github.com/oseiaspereira88/pose/releases/latest/download/install.sh && bash install.sh
```

The script resolves the latest release for the current OS and architecture,
installs `pose` into `~/.local/bin` and runs `pose install .` plus the final
strict gate when the current directory is a Git repository. Ensure
`~/.local/bin` is on `PATH`, then confirm the installation:

```bash
pose version
pose doctor
```

The one-liner tracks `latest` and optimizes first-time adoption. It relies on
HTTPS but does not independently verify the downloaded archive's checksum or
Sigstore identity. Use the pinned flow below when reproducibility or
supply-chain verification is required.

### Verified install

```bash
# with the native binary on PATH:
pose install /path/to/your/repo

# or from a release bundle containing install.sh beside the pose binary:
bash install.sh /path/to/your/repo
```

The installer copies workflows, rules, templates and skills, derives
`{{PROJECT_NAME}}`/`{{PROJECT_ID}}` from your directory name (override with
`--project-name` / `--project-id`), configures the same binary as the MCP
server, stamps the contract schema version and finishes with native `init`,
`index` and `check --strict` — installation only reports success if the gate
is green.

The v1 release archive is accompanied by SHA-256 checksums, keyless Sigstore
signatures, a CycloneDX SBOM and SLSA provenance. Use the release's own
`checksums.txt` before placing the binary on `PATH`; package-manager channel
support and its current limits are documented in [Package channels](package-channels.md).

Useful flags:

| Flag | Effect |
|---|---|
| `--locale pt-BR` | Install docs and templates in Brazilian Portuguese |
| `--force` | Overwrite an edited `AGENTS.md`/`POSE.md` on re-run |
| `--skip-mcp` | Skip the MCP server entirely |
| `--allow-non-git` | Install into a non-git directory (not recommended) |

Re-running the installer updates the machinery and **never touches your
instance content** (specs, ADRs, knowledge, reports, roadmaps). Custom rules,
workflows and templates you added are preserved.

## Detect your stacks

```bash
pose init --wizard        # interactive; --yes accepts all suggestions
```

The wizard detects modules by stack markers (`go.mod`, `package.json`,
`Cargo.toml`, `pom.xml`, `build.gradle`) and seeds them into the validation
matrix in `tolerant` mode — promote to `strict` when the checks stabilize.

## The first governed loop

This is the whole point of the quickstart: not to show you twenty commands,
but to make one thing happen — **a gate that blocks for a reason you
understand, and then passes**. Follow it in order; there is nothing to choose
between.

An all-green run would not show you anything. Any tool can agree with you.

### 1. Scaffold a spec

```bash
pose new-spec customer-export
```

```
Spec created: .pose/specs/2026-09-07-customer-export.md (status: draft)
```

### 2. Watch the entry gate refuse it

```bash
pose lint-spec customer-export --ready-check
```

```
[ERROR] customer-export: DoR: section Intent is missing, empty, or skeletal
spec.ready=false
spec.ready.failures=1
```

The spec exists, but nothing about it is decided yet. POSE will not let work
start against a spec that has not said what it is for. This is the *entry*
gate — Definition of Ready — and most tools do not have one.

### 3. Say what the work is

Open the spec and fill two things: the **Intent** section, and at least one
acceptance criterion with a stable ID under Requirements.

```markdown
### Goal
Export customer records as CSV for audit.

### Business value
Auditors currently request exports by hand.
```

```markdown
- R1: The exporter shall write customer records as CSV.
```

Run the entry gate again:

```bash
pose lint-spec customer-export --ready-check
```

```
spec.ready=true
spec.ready.failures=0
Resultado: SUCESSO
```

### 4. Find out what applies here

```bash
pose suggest feature
```

This resolves the workflow, skill, cumulative rules and validation commands
for this kind of work in this repository — before an agent edits anything. The
agent does not have to be told your engineering process in a prompt; it can
ask.

### 5. Implement, then prove it

```bash
pose validate --strict
```

POSE runs *your* repository's declared checks — the test, lint and build
commands in the validation matrix — not commands it invented.

### 6. Declare it done, and watch the exit gate refuse that too

Set `status: done` and `completed_at` in the spec frontmatter, then:

```bash
pose lint-spec customer-export --strict
```

```
[ERROR] customer-export: requirement trace: R1 has no trace entry
        (declare satisfied, waived or withdrawn)
```

Read that error slowly, because it is the product in one line.

You said the work is done. POSE is not disputing your code — the checks
passed. It is pointing out that **R1, the thing you promised, is not connected
to any evidence that it happened**. "Done" is a claim by whoever executed;
POSE requires it to be a property of the repository.

### 7. Connect the promise to the proof

Under `### Requirement trace`, say how R1 was satisfied:

```markdown
- R1 [satisfied] test:TestCustomerExportWritesCSV
```

```bash
pose lint-spec customer-export --strict
```

```
spec.trace.present=true
spec.trace.entries=1
spec.trace.missing=0
Resultado: SUCESSO
```

That is a governed delivery. An entry gate that refused to start
under-specified work, real repository checks, and an exit gate that refused to
close until every promise pointed at evidence.

If a requirement turned out not to apply, you say that instead — `[waived:
<reason>]` or `[withdrawn: <reason>]`. What you cannot do is stay silent,
which is the option most processes leave open.

## See the first analytics

Recognized CLI/MCP activity is observed automatically and locally. The query
does not count itself:

```bash
pose usage --since-days 30
pose adoption-metrics --json
```

These are product usage and adoption signals. They do not infer deployment
outcomes. To measure delivery, ingest explicit deployment/incident events and
run `pose dora-metrics`; see [Analytics and delivery metrics](analytics.md).

## Keep it healthy

```bash
pose check --strict       # structural integrity + graphs + schema version
pose validate --tolerant  # run the validation matrix
pose followups --open     # live backlog from all specs
pose update               # migrate the contract after engine updates
pose hooks install        # pre-commit check + post-merge reindex
```

Requirements: Git plus Bash for the one-liner, or the native `pose` binary for
the verified/manual path. The POSE runtime itself needs no Bash or Python.
Platforms: Linux, macOS and Windows.
