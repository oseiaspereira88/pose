# POSE — Project Operating Standard for Engineering

## 1) What it is

POSE is the operating standard for agent work in **{{PROJECT_NAME}}**.

Primary goals:

- reduce ambiguity in tasks
- improve execution predictability
- make validation and reporting consistent
- scale collaboration in a heterogeneous repository

POSE does **not** replace product architecture or existing security policies;
it organizes how agents execute technical work.

The short agent contract lives in [`AGENTS.md`](AGENTS.md); this document is
the operating manual (structure, CLI, per-type flows, CI, governance).

---

## 2) Principles

1. **Scope first**: read only the instructions and artifacts needed for the affected directories.
2. **Plan before implementing**: non-trivial changes go through a spec/plan.
3. **Incrementalism**: small, cohesive, auditable deliveries.
4. **Deterministic validation**: prefer reproducible commands (`test`, `lint`, `typecheck`, `build`, contract/security checks).
5. **Risk transparency**: always surface gaps and human-review points.

---

## 3) Structure

```text
.pose/
  workflows/     # procedure per work type
  templates/     # spec.md, roadmap.md, knowledge.md, changelog-fragment.md, doc-audit-report.md
  rules/         # domain rules (cumulative)
  knowledge/     # handoffs and notes with active governance
  adr/           # architectural decisions
  roadmaps/      # governed roadmaps (milestone DAGs)
  changelogs/    # user-facing fragments per spec (unreleased/ until the release cut)
  indexes/       # repo-map, services, packages, validation-matrix, module-metadata, task-map, spec-graph, roadmaps
  reports/       # versionable reports + history JSONL + archive/
  specs/         # living specs per feature
  schema-version # instance contract version (see `pose upgrade`)

.agents/skills/  # skills (source of truth; Codex-native format)
.claude/skills/  # Claude Code-compatible symlinks
pose             # native Go binary available on PATH
AGENTS.md        # short operating contract
POSE.md          # this manual
```

---

## 4) Key files

- [`AGENTS.md`](AGENTS.md): short contract, precedence and entry points.
- Subproject `AGENTS.md` (when present): local guidance, applied only to that directory's scope.
- [`.pose/workflows/*.md`](.pose/workflows/): procedure per work type (`feature`, `bugfix`, `review`, `refactor`, `documentation-update`, `recurrence-escalation`).
- [`.pose/rules/*.md`](.pose/rules/): domain rules; recurring content lives in [`.pose/rules/_base-recurrence.md`](.pose/rules/_base-recurrence.md).
- [`.pose/templates/spec.md`](.pose/templates/spec.md): the single per-feature spec template.
- [`.pose/templates/roadmap.md`](.pose/templates/roadmap.md): governed roadmap template.
- [`.pose/templates/changelog-fragment.md`](.pose/templates/changelog-fragment.md): user-facing fragment per spec (written at closeout).
- [`.pose/templates/doc-audit-report.md`](.pose/templates/doc-audit-report.md): template for editorial reviews and documentation audits.
- `pose` binary: native scaffold/check/validate/report automation and MCP server.
- [`.pose/specs/*/spec.md`](.pose/specs/): living specs per feature.
- [`.agents/skills/`](.agents/skills/): 9 skills in Codex-native format (`name`/`description` frontmatter, body with Required reading + Steps + Output requirements, optional metadata in `agents/openai.yaml`). Use `description` as the single routing source; Claude Code consumes the symlinks in [`.claude/skills/`](.claude/skills/) without requiring `when_to_use`.

---

## 5) Flows per task type

The operational step-by-step lives in the workflows. Each workflow also
includes the relevant "Execution — planner/implementer/reviewer mode" sections.

- Feature: [`.pose/workflows/feature.md`](.pose/workflows/feature.md)
- Bugfix: [`.pose/workflows/bugfix.md`](.pose/workflows/bugfix.md)
- Review: [`.pose/workflows/review.md`](.pose/workflows/review.md)
- Refactor: [`.pose/workflows/refactor.md`](.pose/workflows/refactor.md)
- Documentation: [`.pose/workflows/documentation-update.md`](.pose/workflows/documentation-update.md)
- Recurrence escalation: [`.pose/workflows/recurrence-escalation.md`](.pose/workflows/recurrence-escalation.md)

The agent contract (precedence, spec/ADR/check obligations, verification,
don'ts) lives in [`AGENTS.md`](AGENTS.md) and is **not** repeated here.

### 5.1 Spec lifecycle

Every spec created by `pose new-spec`
carries frontmatter with state and dates, preventing specs that linger "open"
after completion and follow-ups that rot into dead text.

```yaml
---
slug: <feature-slug>
status: draft        # draft → in-progress → done | blocked | superseded | abandoned
created_at: 2026-01-15   # stamped by pose new-spec
completed_at:            # filled on the transition to done
supersedes:              # slug of the superseded spec (when applicable)
depends_on:              # prerequisites: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority:                # integer >= 0 (lower = higher priority)
---
```

- **`status`** evolves `draft` → `in-progress` → `done`. Alternative terminal
  states: `blocked`, `superseded` (use `supersedes:` on the successor),
  `abandoned`.
- **`created_at`/`completed_at`** give the spec's real time window (file mtime
  is unreliable — it changes on every edit).
- **`depends_on`** declares prerequisites as an **inline comma-separated
  list** (POSE frontmatter is flat by contract — never a multi-line YAML
  list), with typed refs: a spec slug, `milestone:<roadmap>/<id>` or
  `roadmap:<slug>`. Spec refs are resolved by `check` (existence + graph
  acyclicity); `milestone:`/`roadmap:` refs resolve against the governed
  roadmaps in `.pose/roadmaps/` when they exist. `depends_on` expresses a real
  technical/logical prerequisite; scheduling preference is `priority`'s job.
  The aggregated graph lives in
  [`.pose/indexes/spec-graph.json`](.pose/indexes/) (generated by
  `pose index`; frontmatter stays authoritative) and a spec's eligibility is
  queryable via the pose-mcp tool `pose_spec_readiness`.
- **`priority`** (optional) orders attack preference among eligible specs; it
  creates no blocking.
- **Follow-ups with disposition:** the `Final Report > Follow-ups` section is
  not free text. Each item gets a bracketed disposition — `[open]`,
  `[spawned: <slug>]`, `[covered: <slug>]`, `[duplicate: <slug>]`, `[done]`,
  `[wont-do: <reason>]`. That answers, per follow-up, whether it seeded a new
  spec, is already covered elsewhere, was triaged before, or was discarded.
  Open items additionally declare ownership and a triage service level with a
  trailing `(owner:@alias crit:low|medium|high review:YYYY-MM-DD)` group —
  the SLA is a triage promise, not an implementation deadline. Legacy items
  without the group are reported as `unowned` (warning at closeout).
- **Requirement trace:** at closeout, the `Validation > Requirement trace`
  subsection maps every declared `R<N>` to its outcome — `[satisfied]` with
  evidence (free text plus structured refs `check:`, `test:`, `report:`,
  `commit:`), `[waived: <reason>]` or `[withdrawn: <reason>]`. Orphaned or
  missing IDs fail `lint-spec --strict`; the MCP tool
  `pose_requirement_trace` exposes the bidirectional projection.

Closeout is an explicit step (skill [`pose-spec-closeout`](.agents/skills/pose-spec-closeout/SKILL.md)):
set `status: done`, fill `completed_at`, triage every follow-up and pass the
gate `pose lint-spec <slug> --strict`,
which blocks "done without `completed_at`" and "done with an undispositioned
follow-up". The aggregated live backlog (`pose followups --open`) feeds the
planning of new specs.

Follow-up triage has **two layers**, by design, to keep the CLI deterministic
and avoid cascading drift:

1. **Deterministic (CLI):** `pose followups`
   proposes near-duplicate candidates by lexical similarity. Reproducible, no
   network, runs in CI.
2. **Semantic + confirmation (agent):** the `pose-spec-closeout` skill judges
   intent equivalence (what lexical heuristics miss) and **confirms with the
   user before writing** the consequential dispositions
   (`[spawned]`/`[covered]`/`[duplicate]`) — reusing a follow-up is a
   decision, not a default.

---

## 6) The `pose` CLI

```bash
pose help                          # show help

# Scaffold
pose init [--wizard [--yes]]       # ensure minimal structure; --wizard detects
                                      # stacks and seeds the validation matrix
pose new-spec <slug>               # create a spec at .pose/specs/<slug>/spec.md
pose new-roadmap <slug>            # create a governed roadmap in .pose/roadmaps/
pose new-adr "<title>"             # create a dated ADR
pose new-knowledge <type> <slug>   # create handoff/note/decision-log
                                      # (options: --owner @x --ttl-days N --restricted)

# Deterministic gates
pose check [--strict|--tolerant]   # structural integrity + matrix schema +
                                      # task-map sync + spec graph + schema version
pose validate [--strict|--tolerant] [--stack s] [--module path] [--report] [--json f] [--junit f] [--sarif f]
              [--changed-from rev [--changed-to rev]] [--explain] [--emit-plan f]
pose knowledge-check [--strict|--tolerant] [--max-overdue N]
pose recurrence-check [--strict|--tolerant] [--window-days N] [--threshold T] [--include-pass]
pose lint-spec <slug>|--all [--strict|--tolerant] [--required-only] [--ready-check]
pose followups [--open|--all] [--json]
pose history-check [--strict|--tolerant]

# Discovery and metrics
pose suggest [<type>] [--domain <d>] [--path <p>] [--json]
pose stats [workflows|tasks|contexts] [--since-days N] [--json]

# Artifact generation
pose index                         # regenerate indexes (incl. spec-graph, roadmaps)
pose report --task "..." [--outcome pass|fail|partial] [--since <ref>] [--git-stage] [...]

# Maintenance
pose upgrade [--dry-run]           # migrate the .pose/ contract to the engine version
pose knowledge-housekeeping <list-expired|archive-expired|purge-archived> [--dry-run|--apply]
pose reports-housekeeping <list-stale|archive-stale|purge-archived> [--older-than N] [--dry-run|--apply]
pose hooks <install|uninstall|status> [--force]
```

### Command reference

- `check` — validates POSE structural integrity (required paths and references in `AGENTS.md`/`POSE.md`) **plus** the [`validation-matrix.json`](.pose/indexes/validation-matrix.json) schema, [`task-map.json`](.pose/indexes/task-map.json) sync, the native spec dependency graph and the schema-version gate. It fails in `--strict` and warns where permitted in `--tolerant`.
- `new-spec` — generates a `spec.md` from [`.pose/templates/spec.md`](.pose/templates/spec.md).
- `new-adr` — creates an ADR with the standard template and a deterministic slug.
- `new-roadmap` — creates a governed roadmap from [`.pose/templates/roadmap.md`](.pose/templates/roadmap.md): flat frontmatter (`status: draft|active|done|abandoned`, `depends_on:` between roadmaps) + milestones as `## Milestone: <id>` sections with flat bullets (`- after:`, `- target_start:`, `- target_due:`, `- specs:`). `check` validates single membership in active roadmaps, milestone/roadmap DAGs, dates and typed-ref resolution; `pose_spec_readiness` resolves those refs for real (milestone satisfied = its specs done; roadmap satisfied = status done). Dates are planning input; actuals derive from events.
- `new-knowledge` — creates an artifact in [`.pose/knowledge/`](.pose/knowledge/) with mandatory frontmatter (`type`, `owner`, `sensitivity`, `created_at`, `last_reviewed_at`, `expires_at`). TTL default 30d, max 90d.
- `validate` — runs the declarative matrix in [`validation-matrix.json`](.pose/indexes/validation-matrix.json): per-stack checks, per-module overrides, severity (`required`/`optional`) and mode (`strict`/`tolerant`). `--json`/`--junit`/`--sarif <path>` emit the versioned structured result (schema 1) from one canonical model: stable check IDs (`<module>/<stack>/<name>`), command metadata, timing, severity, distinguishable outcomes (`pass|fail|error|skipped` — infra failures never masquerade as check failures), deterministic skip reasons, bounded captured output and secret redaction (configured env values only; inherited environment never enters the result). Text output stays authoritative; machine formats are additive. POSE-specific semantics survive the JUnit/SARIF projections via documented extensions (classname suffix / `pose/*` properties).
  **Runtime guardrails:** every check runs under a timeout (`timeoutSeconds` per check, `defaults.timeoutSeconds`, safe default 600s) and an output ceiling (`defaults.maxOutputBytes`, default 1 MiB); breaching either terminates the process group and records the explicit state (`limit_state: timeout|output-limit`). Checks marked `isolation: "required"` never run locally — they are skipped with a machine-readable reason and exported by `--emit-plan <file>`: an execution-plan envelope binding project, spec, check plan, matrix digest, git HEAD and an approval slot to be stamped with an expiring execution identity before the Harness may run it.
  **Changed scope:** `--changed-from <rev> [--changed-to <rev>]` selects the minimum safe module set deterministically — modules containing changed files (tracked and untracked), transitive dependents via `dependsOn` edges in [`module-metadata.json`](.pose/indexes/module-metadata.json), and policy widening (criticality `high` always runs). A change outside every module runs everything (uncertainty prefers safe execution); unselected checks are recorded as skipped with the selection reason and `--explain` prints every decision. Revisions are confined to a safe grammar; without the flags, full validation is unchanged.
- `upgrade` — migrates the instance contract through native, sequential and idempotent migrations; `--dry-run` lists the plan. Downgrade is always refused.
- `index` — generates `repo-map.json`, `services.json`, `packages.json`, `spec-graph.json` and `roadmaps.json` in `.pose/indexes/`, including per-module operational metadata from [`module-metadata.json`](.pose/indexes/module-metadata.json).
- `report` — generates a versionable report in `.pose/reports/` with execution metadata, minimal per-task history (`.pose/reports/history/`) and stable-field diffs.
- `knowledge-check` — double gate: (1) frontmatter schema of each knowledge artifact, and (2) overdue backlog against `--max-overdue`. In `--strict` both gates exit 1.
- `recurrence-check` — scans [`history JSONL`](.pose/reports/) for `task_slug`s with `≥ --threshold` occurrences in `--window-days` (default 3 in 14d). Ignores `outcome=pass` by default. When flagged, points to [`recurrence-escalation.md`](.pose/workflows/recurrence-escalation.md).
- `recurrence-effect` — closes the feedback edge: `--register` ties an escalation to its intervention (`rule:|workflow:|spec:<name>`) and observation window in append-only `interventions.jsonl`; the report compares recurrence rate (and optional `pose report --duration-seconds/--cost-usd` telemetry) before/after per intervention with data-quality warnings (sparse sample, incomplete window). `INEFFECTIVE` verdicts demand a governed follow-up; `--fail-ineffective` makes that blocking by policy. Aggregation is by task/context only — never individuals.
- `lint-spec` — verifies each `spec.md` section has real content, not placeholders. **`--ready-check`** applies the **Definition of Ready** (ENTRY gate): Intent/Requirements/Technical Plan filled, acceptance criteria with stable IDs (`- R<N>:`) and syntactically valid `depends_on` — without requiring Validation/Final Report. `check` applies the ready-check automatically on the `→ in-progress` transition. **Lifecycle gate:** `status: done` requires `completed_at` and a valid disposition on every follow-up; for `spawned`/`covered`/`duplicate` the target must be an **existing** spec (and not itself).
- `followups` — aggregates all specs' follow-ups, derives the live (`--open`) or full (`--all`) backlog, projects ownership (`--owner <alias>`) and expired reviews (`--overdue`), and proposes near-duplicate candidates by deterministic lexical similarity (stdlib only; threshold via `--similarity 0..100`, default 60). Exit 0 by default; `--fail-overdue` turns expired reviews into a blocking, risk-based policy gate.
- `history-check` — verifies every `.jsonl` under `reports/history/` is git-tracked. Strict blocks; tolerant warns.
- `suggest` — reads [`task-map.json`](.pose/indexes/task-map.json) and prints the canonical trail (workflow + skill + rules + spec/ADR + knowledge) for a task type. `--domain` adds domain rules; `--path` infers the domain via [`repo-map.json`](.pose/indexes/repo-map.json); `--json` for agents.
- `stats` — aggregates history JSONL outcomes by workflow, task or context. `--since-days N`; `--json`.
- `knowledge-housekeeping` / `reports-housekeeping` — idempotent maintenance (list/archive/purge). Mutations require `--apply`. Reports housekeeping **never touches `history/`**.
- `amend` — append-only spec amendment history (`.pose/specs/<slug>/amendments.jsonl`). `--baseline` snapshots every R-ID hash; `--ids R2 --change added|withdrawn|semantic|editorial --rationale <text> --author @alias [--reviewer @alias]` acknowledges a material change; `--list` renders history and pending acknowledgments. On `done` specs with a history, `lint-spec` rejects any requirement whose current text is not acknowledged by an event — specs cannot be silently rewritten after evidence.
- `knowledge-usage` — projects `knowledge:<slug>` citations from specs per artifact (owner, expiry, citing specs). Usage signals inform the owner's review; TTL is never extended automatically. Dangling `knowledge:` refs fail `knowledge-check`.
- `knowledge-suggest <query>` — deterministic, explainable lexical ranking over non-restricted knowledge (shared-term rationale exposed). Advisory only: suggestions never gate or auto-apply and require human confirmation before citing.
- `hooks` — links the native binary into `.git/hooks/`; invocation name selects `check --tolerant` for `pre-commit` and `index` for `post-merge`.

---

## 7) CI policy

- Run `pose check --strict` on every `pull_request` to `main`; treat failure as blocking.
- Run `pose validate --strict` on every `pull_request` to `main`; treat `required` check failure as blocking.
- Run the same workflow on `push` to `main` to detect post-merge drift.
- Publish versionable artifacts per run: `pose-check.log`, `pose-validate.latest.log` and the `pose report` output.
- Consume those artifacts during review instead of ephemeral job logs.

A ready-made GitHub Action wrapping these gates ships with the POSE
distribution (`pose-action/`).

### Interpreting failures

- `POSE check (strict)` failure = structural break of the standard.
- `POSE validate (strict, required gate)` failure = objective quality block.
- `optional`-only failures = flagged technical risk; prioritize by criticality.

### Phased rollout (recommended)

1. Observability: PR workflow with artifacts, no new gates raised.
2. Enforcement on `main`: strict `check` and `validate` as blocking gates; adjust `moduleOverrides` for modules not ready.
3. Gradual expansion: promote mature checks from `optional` to `required` per domain.
4. Hardening: review the matrix periodically, remove temporary exceptions.

### Validation matrix per stack/module

- Single source: [`validation-matrix.json`](.pose/indexes/validation-matrix.json).
- Base stacks: `node`, `go`, `rust`, `java` (Maven/Gradle).
- `moduleOverrides` adjusts stack, mode and extra checks per module
  (`pose init --wizard` seeds them from repository scan).
- `required` in a `strict` or `tolerant` module → exit 1; `optional` failures don't block.

---

## 8) `.pose/knowledge/` governance

The full loop (create → consult in workflows → schema validation → CI gate →
housekeeping) is available from installation; maturity comes from usage.

Write path: `pose new-knowledge <type> <slug>`.
Read path: the feature/bugfix/review workflows include "consult
`.pose/knowledge/`" as a mandatory checklist step.
Gate: `pose knowledge-check --strict`, used in CI.

Health criteria: a dedicated governance spec when activating the subsystem;
the [`knowledge-governance.md`](.pose/rules/knowledge-governance.md) rule;
defined ownership with biweekly/monthly review; minimal housekeeping.
On repeated neglect (overdue items untreated for 2 cycles), treat `knowledge`
as degraded and block functional expansion until it recovers.

---

## 9) Instance limitations

<!-- Keep the REAL limitations of your instance here, with evidence:
     modules missing from module-metadata.json, stacks outside the matrix,
     gates still tolerant and why. -->

- Document limitations as the instance evolves.

---

## 10) Instance next steps

<!-- The operational backlog of POSE IN THIS repository (not product
     features). Each item with an owner and a done criterion. -->

1. Fill `.pose/indexes/module-metadata.json` for critical modules.
2. Enable strict `check`/`validate` in CI (see §7).
3. Run knowledge housekeeping on a recurring cycle.

---

## 11) Executive summary

POSE is the operational layer that makes agent work reliable in the repository:

- short instructions in [`AGENTS.md`](AGENTS.md)
- operational depth in [`.pose/`](.pose/)
- assisted execution via [`pose`](pose) (CLI)
- progressive maturity with skills in [`.agents/skills/`](.agents/skills/)
