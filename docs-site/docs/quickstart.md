# Quickstart

**Doc type:** Tutorial &nbsp;·&nbsp; **Applies to:** POSE ≥ 0.9.0

## Install

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
pose lint-spec my-first-feature --strict        # closeout gate
```

## Keep it healthy

```bash
pose check --strict       # structural integrity + graphs + schema version
pose validate --tolerant  # run the validation matrix
pose followups --open     # live backlog from all specs
pose upgrade              # migrate the contract after engine updates
pose hooks install        # pre-commit check + post-merge reindex
```

Requirements: the native `pose` binary and git. No Bash or Python runtime is
needed. Platforms: Linux, macOS and Windows.
