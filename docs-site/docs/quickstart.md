# Quickstart

## Install

```bash
# from a clone of the POSE repository:
bash pose-dist/install.sh /path/to/your/repo
```

The installer copies the machinery (CLI, engine, workflows, rules, templates,
hooks, skills), derives `{{PROJECT_NAME}}`/`{{PROJECT_ID}}` from your
directory name (override with `--project-name` / `--project-id`), builds the
MCP server when a Go toolchain is available (`--mcp-binary` / `--skip-mcp` to
control this), stamps the contract schema version and finishes by running
`./pose init && ./pose check --strict` in your repo — the install only
reports success if the gate is green.

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
./pose init --wizard        # interactive; --yes accepts all suggestions
```

The wizard detects modules by stack markers (`go.mod`, `package.json`,
`Cargo.toml`, `pom.xml`, `build.gradle`) and seeds them into the validation
matrix in `tolerant` mode — promote to `strict` when the checks stabilize.

## First spec

```bash
./pose new-spec my-first-feature   # scaffold
./pose suggest feature             # canonical trail: workflow + skill + rules
# fill Intent / Requirements (R-IDs!) / Technical Plan, then:
./pose lint-spec my-first-feature --ready-check   # entry gate
# ... implement, validate ...
./pose lint-spec my-first-feature --strict        # closeout gate
```

## Keep it healthy

```bash
./pose check --strict       # structural integrity + graphs + schema version
./pose validate --tolerant  # run the validation matrix
./pose followups --open     # live backlog from all specs
./pose upgrade              # migrate the contract after engine updates
./pose hooks install        # pre-commit check + post-merge reindex
```

Requirements: bash 4+, git, python3 (Go optional, only for the MCP server).
Platforms: Linux/macOS/WSL; native Windows arrives with the fully-native Go
CLI (in progress — see the roadmap).
