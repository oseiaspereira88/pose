# POSE — Project Operating Standard for Engineering

**Spec-driven development that closes the loop.**

POSE is a governance framework for agentic software engineering. Where most
spec-driven-development (SDD) tools stop at scaffolding a spec, POSE governs the
full cycle — and enforces it with deterministic gates:

```
spec → execution → evidence → follow-ups → recurrence → knowledge
  ▲                                                        │
  └────────────── the loop closes back into planning ──────┘
```

- A spec cannot enter execution without passing a **Definition of Ready** gate
  (`pose lint-spec --ready-check`: intent, requirements with stable acceptance
  criteria IDs, technical plan).
- A spec cannot be marked done without a **closeout gate** (`pose lint-spec
  --strict`: completion date stamped, every follow-up explicitly dispositioned —
  spawned, covered, duplicate, done, or consciously discarded).
- Validation is **deterministic by contract**: a per-stack/per-module validation
  matrix (`pose validate`) runs real commands (`test`, `lint`, `typecheck`,
  `build`) and produces versionable reports with append-only JSONL history.
- **Recurrence detection** (`pose recurrence-check`) mines that history for
  repeated failures and escalates them into systemic fixes (new rules or
  workflows) instead of letting them be re-fixed forever.
- **Operational memory** (`.pose/knowledge/`) captures handoffs and decision
  logs with TTL governance, so context survives between executions and agents.
- Specs form a **dependency graph** (`depends_on`, `priority`) organized into
  governed **roadmaps** with milestone DAGs — validated for existence and
  acyclicity on every `pose check`.
- Everything is exposed to agents through a native **MCP server** (17 read
  tools: specs, readiness, roadmaps, knowledge, reports, changelogs, skills).

## What's in the box

| Path | Purpose |
|---|---|
| `pose` | CLI dispatcher: scaffolding, gates, discovery, metrics, housekeeping |
| `.pose/scripts/` | The engine behind the CLI (deterministic, offline, no network) |
| `.pose/workflows/` | Procedures per task type: feature, bugfix, review, refactor, docs, recurrence escalation |
| `.pose/rules/` | Cumulative domain rules: security, backend, frontend, docs style, delivery evidence, knowledge governance |
| `.pose/templates/` | Spec, roadmap, knowledge and changelog-fragment templates |
| `.pose/hooks/` | Git hooks (`pre-commit` runs `pose check`, `post-merge` reindexes) |
| `.pose/indexes/` | Machine-readable caches: repo map, validation matrix, spec graph, roadmaps, task map |
| `.agents/skills/` | 9 agent skills (Codex-native format; `.claude/skills/` symlinks for Claude Code) |
| `AGENTS.md` / `POSE.md` | The short agent contract and the full operating manual |
| `pose-mcp` | MCP server (Go) exposing the whole instance to agents |

## Quickstart

```bash
# from a clone of the POSE repository:
bash pose-dist/install.sh /path/to/your/repo
```

That's it. The installer:

- copies the machinery (CLI, engine scripts, workflows, rules, templates,
  hooks, skills) and creates the empty instance directories;
- substitutes `{{PROJECT_NAME}}`/`{{PROJECT_ID}}` in `AGENTS.md`/`POSE.md`
  (derived from the target directory name; override with `--project-name` /
  `--project-id`);
- builds the MCP server from source when a Go toolchain is available (or
  vendors a binary you pass via `--mcp-binary`; `--skip-mcp` to opt out) and
  generates a wrapper at `.pose/bin/pose-mcp-claude` that derives the project
  root — nothing is ever hardcoded;
- seeds `.mcp.json` if your repo has none;
- finishes by running `./pose init && ./pose check --strict` in your repo —
  the install is only reported successful if the gate is green.

Re-running the installer updates the machinery in place and **never touches
your instance content** (specs, ADRs, knowledge, reports, roadmaps). Your
edited `AGENTS.md`/`POSE.md` are preserved unless you pass `--force`.

Then start working: `./pose new-spec my-first-feature`, fill the spec, and let
the gates guide the rest (`./pose suggest feature` prints the canonical trail:
workflow + skill + rules).

Already use spec-kit or OpenSpec? Preview an offline migration with
`pose import spec-kit <path> --dry-run` or
`pose import openspec <path> --dry-run`, then rerun without `--dry-run` to
create POSE specs. Import validates the whole batch before writing, never
overwrites an existing spec, and prints a curation report for information the
source format could not supply. See the [CLI reference](docs-site/docs/cli.md#import-existing-sdd-specs)
for supported layouts and limits.

Teams standardized on pre-commit.com can enable `pose-check`,
`pose-lint-spec`, and `pose-history-check` directly from this repository. Pin
an immutable POSE release and see the [CI guide](docs-site/docs/ci.md#use-pose-from-pre-commitcom)
for the four-line configuration and pre-commit 4.4+ requirement.

Requirements: bash 4+, git, python3 (Go optional, only for the MCP server).
Platforms: Linux/macOS/WSL — native Windows support is on the roadmap, via the
single-binary Go CLI.

## How POSE compares

GitHub's spec-kit, OpenSpec and similar SDD tools generate well-structured
specs and prompts — they govern the *entry* of work. POSE governs entry **and
exit**: the Definition-of-Ready gate is matched by a closeout gate that refuses
"done" until evidence is recorded and every follow-up is triaged, and the
history those gates produce feeds recurrence detection and portfolio-level
readiness (dependency graph + roadmaps). If you only need spec templates,
lighter tools are fine; POSE is for teams that want the loop closed and
machine-checkable.

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
Contributions welcome: see [CONTRIBUTING.md](CONTRIBUTING.md).
Security reports: see [SECURITY.md](SECURITY.md).

POSE is developed as part of the **Crisol** platform (AI-native engineering:
orchestration, execution and visual operation on top of POSE governance).
POSE itself is free and runs entirely offline in your repository.
