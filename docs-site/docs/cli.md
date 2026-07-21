# CLI reference

**Doc type:** Reference &nbsp;·&nbsp; **Applies to:** POSE ≥ 0.9.0

The `pose` CLI is a single native Go binary. Every command below executes
without Bash or Python fallbacks and works offline.

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

## DORA and adoption metrics

| Command | Purpose |
|---|---|
| `pose record-deployment --application A --environment E --status success\|failure --source manual\|ci\|webhook [--deployed-at RFC3339] [--lead-time-seconds N] [--change-ref R]` | Ingest one deployment event |
| `pose record-incident --application A --started-at RFC3339 --severity minor\|major\|critical --source manual\|ci\|webhook [--resolved-at RFC3339] [--caused-by-deployment]` | Ingest one incident event |
| `pose dora-metrics [--application A] [--window-days N] [--json]` | The 5 DORA metrics; each reports `unavailable` (never a fabricated zero) without real data |
| `pose adoption-metrics [--json]` | Activation, time-to-first-gate, retention, task success — derived from specs/history POSE already owns |
| `pose events-housekeeping <list-expired\|purge> [--older-than-days N] [--apply]` | Retention/deletion for stored deployment/incident events |

Deployment and incident events are explicit input only — POSE never infers
them from commits — and carry no identity field beyond `application` and
`source`; every metric is a team/application aggregate, never an
individual score. See [DORA metrics guide](https://dora.dev/guides/dora-metrics/).

## Semantic governance assist

| Command | Purpose |
|---|---|
| `pose semantic-suggest (--for <spec-slug>\|--query "text") [--top N] [--provider lexical] [--json]` | Advisory suggestions: related follow-ups, recurrence patterns and knowledge, each cited with score/rationale/provider |
| `pose suggest-feedback --for <spec-slug> --ref <artifact-ref> --kind knowledge\|followup\|recurrence --decision accept\|reject [--score N]` | Record a minimized accept/reject decision (never the candidate's content) |

Suggestions are advisory only — they never gate a check or mutate a spec.
`lexical` (deterministic, offline token/sequence similarity) is the only
approved provider today; sensitivity-restricted knowledge is filtered
before any retrieval, never suggested.

## Capability assessment

| Command | Purpose |
|---|---|
| `pose assess` | Validate `.pose/capabilities/assessment.md`: schema, typed evidence resolution, stable mechanism ids, staleness vs. policy |
| `pose assess init` | Scaffold the artifact with the method's 16 default mechanisms |
| `pose assess snapshot` | Append the current score vector to `history.jsonl` (append-only; no-op when unchanged) |
| `pose assess diff [--from <ts>] [--to <ts>] [--json]` | Mechanical comparison between two snapshots (raised/lowered/added/retired) |

Scores are human judgment (0-5; the target is not always 5) — the mechanism
validates structure and evidence, it never computes a score. Evidence uses
typed references (`spec:`/`report:`/`adr:`/`knowledge:`/`doc:`/`commit:`/
`check:`/`url:`); local types must resolve, the rest are syntactic
(offline contract). `pose check --strict` runs the same validation when the
artifact exists (opt-in by presence). Staleness thresholds live in
`.pose/policy/capabilities.json` (defaults: 30 days / 200 commits).

## Cross-repository portfolio

| Command | Purpose |
|---|---|
| `pose portfolio-projection [--projects-dir DIR] [--max-staleness-days N] [--json]` | Reconcile dependencies, readiness, ownership and criticality across authorized repositories |

Only repositories registered via `HARNE8_PROJECTS_DIR` (or explicit
`POSE_PROJECT_ROOTS`) — the same allowlist the MCP server already uses —
ever enter a projection; nothing is discovered by an open filesystem
walk. Add `depends_on: xref:<project_id>/<spec-slug>` to a spec to
declare a cross-repository dependency (additive to the existing
`other-spec` / `milestone:...` / `roadmap:...` forms). The projection is
persisted to `.pose/reports/portfolio-projection.json`, explains every
blocked, stale or unauthorized/unknown cross-reference explicitly, and
tombstones artifacts that disappeared since the last run rather than
silently dropping them. Repositories remain authoritative; the
projection is a reconciled read, never a write back to another
repository.

## Harness evidence reconciliation

| Command | Purpose |
|---|---|
| `pose reconcile-evidence record --run-id ID --request-id ID --execution-id ID --plan-digest SHA --status success\|failure --source harness\|manual [--result-digest SHA] [--allow-supersede]` | Reconcile a Harness execution result into local evidence, identity-bound to the submitting Execution Identity |
| `pose reconcile-evidence list [--request-id ID] [--json]` | List recorded evidence |
| `pose reconcile-evidence housekeeping <list-expired\|purge> [--older-than-days N] [--apply]` | Retention for evidence records |

A second record for a `request_id` that already has evidence is rejected
unless `--allow-supersede` is passed — and even then the prior record is
never edited or removed, only superseded by a new, explicitly-linked one.
See [architecture: Harne8 control-plane composition](architecture.md#mechanism-15-harne8-control-plane-composition).

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
| `pose doctor [--json] [--fix [--yes] [--only <check>]]` | Read-only diagnostics; `--fix` previews confined remediation, `--fix --yes` applies and rechecks it |
| `pose knowledge-housekeeping <op> [--apply]` | List/archive/purge expired knowledge |
| `pose reports-housekeeping <op> [--apply]` | Same for reports (never touches `history/`) |
| `pose hooks <install\|uninstall\|status>` | Git hooks: pre-commit check, post-merge reindex |
| `pose serve-mcp [--stdio]` | Start the MCP server (unified binary) |
| `pose version` | Binary version + instance schema version |

Every gate is offline by design — no network calls, stdlib only. A gate
observed doing network I/O is a reportable bug (see SECURITY.md).
