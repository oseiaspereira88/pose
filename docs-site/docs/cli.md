# CLI reference

The `pose` CLI has two layers today (strangler migration in progress): a
unified Go binary (`pose`) for native commands and the script engine in
`.pose/scripts/` for commands still being migrated, with identical interface
and exit codes.

## Scaffold

| Command | Purpose |
|---|---|
| `pose init [--wizard [--yes]]` | Ensure the minimal structure; the wizard detects stacks and seeds the validation matrix |
| `pose new-spec <slug>` | Create `.pose/specs/<slug>/spec.md` from the template |
| `pose new-roadmap <slug>` | Create a governed roadmap in `.pose/roadmaps/` |
| `pose new-adr "<title>"` | Create a dated ADR |
| `pose new-knowledge <type> <slug>` | Create handoff/note/decision-log (`--owner`, `--ttl-days`, `--restricted`) |

## Deterministic gates

| Command | Purpose |
|---|---|
| `pose check [--strict\|--tolerant]` | Structural integrity + matrix schema + task-map sync + spec graph + schema version |
| `pose validate [--strict\|--tolerant] [--stack s] [--module p] [--report]` | Run the validation matrix |
| `pose lint-spec <slug>\|--all [--ready-check]` | Section content, DoR entry gate, done-lifecycle gate |
| `pose followups [--open\|--all] [--json]` | Aggregate follow-ups + near-duplicate candidates |
| `pose knowledge-check [--max-overdue N]` | Knowledge schema + overdue backlog |
| `pose recurrence-check [--window-days N] [--threshold T]` | Recurring failing task slugs |
| `pose history-check` | All history JSONL must be git-tracked |

## Discovery, metrics, artifacts

| Command | Purpose |
|---|---|
| `pose suggest [<type>] [--domain d] [--path p] [--json]` | Canonical trail: workflow + skill + rules |
| `pose stats [workflows\|tasks\|contexts] [--since-days N]` | Outcome aggregation from history |
| `pose index` | Regenerate all indexes (repo-map, spec-graph, roadmaps…) |
| `pose report --task "..." [--outcome ...] [--since ref]` | Versionable report + history JSONL |

## Import existing SDD specs

```bash
# Preview every spec-kit feature under the tree without writing files.
pose import spec-kit .specify/specs --dry-run

# Import an OpenSpec capability, specs tree, or change directory.
pose import openspec openspec/changes/add-2fa
```

The importer is native, deterministic, and offline. It accepts a single
`spec.md`, a feature/capability directory, or a supported specs tree. spec-kit
imports consume sibling `plan.md` and `tasks.md` when available; OpenSpec
change imports consume `proposal.md`, `design.md`, `tasks.md`, and capability
specs below `specs/`.

Every unit becomes `.pose/specs/<slug>/spec.md`. POSE validates the complete
batch before writing, never overwrites an existing destination, rejects
symlinks, and reports every source section that still needs human curation.
Use `pose lint-spec <slug> --ready-check` after reviewing that report. The
first version intentionally does not support force-overwrite, bidirectional
sync, custom spec-kit presets, or OpenSpec schemas outside the documented
behavioral/change layout.

## Maintenance

| Command | Purpose |
|---|---|
| `pose upgrade [--dry-run]` | Migrate the instance contract to the engine version |
| `pose knowledge-housekeeping <op> [--apply]` | List/archive/purge expired knowledge |
| `pose reports-housekeeping <op> [--apply]` | Same for reports (never touches `history/`) |
| `pose hooks <install\|uninstall\|status>` | Git hooks: pre-commit check, post-merge reindex |
| `pose serve-mcp [--stdio]` | Start the MCP server (unified binary) |
| `pose version` | Binary version + instance schema version |

Every gate is offline by design — no network calls, stdlib only. A gate
observed doing network I/O is a reportable bug (see SECURITY.md).
