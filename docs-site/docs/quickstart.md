# Quickstart

**Doc type:** Tutorial &nbsp;·&nbsp; **Applies to:** POSE 1.2.x

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

## Onboard your stacks

```bash
pose init --wizard        # interactive; --yes accepts all suggestions
```

The wizard detects modules by stack markers (`go.mod`, `package.json`,
`Cargo.toml`, `pom.xml`, `build.gradle`) and seeds them into the validation
matrix in `tolerant` mode — promote to `strict` when the checks stabilize.

## First spec

```bash
pose new-spec my-first-feature   # scaffold
pose suggest feature             # canonical trail: workflow + skill + rules
# fill Intent / Requirements (R-IDs!) / Technical Plan, then:
pose lint-spec my-first-feature --ready-check   # entry gate
# ... implement, validate ...
pose validate --strict --report                 # project checks + evidence
pose lint-spec my-first-feature --strict        # lifecycle closeout gate
```

The task trail is repository-owned: `pose suggest feature --path <module>`
resolves the applicable workflow, skill, cumulative rules and validation
command before an agent edits code. For medium/high-risk work, write the test
plan in the spec before implementation; after delivery, link each requirement
to current evidence and explicitly disposition every follow-up.

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
