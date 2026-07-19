# Concepts

## The closed loop

POSE's central idea: **work that leaves no machine-checkable trace didn't
finish.** Every stage of the cycle emits an artifact the next stage consumes:

1. **Spec** — a living document with flat frontmatter (status, dates,
   dependencies, priority) and seven sections (Intent → Final Report).
2. **Execution** — governed by workflows per task type (feature, bugfix,
   review, refactor, docs, recurrence escalation) and skills per recurring
   task.
3. **Evidence** — `pose validate` runs the deterministic matrix; `pose report`
   persists versionable reports plus append-only JSONL history.
4. **Follow-ups** — everything discovered but not done is recorded with a
   disposition; `pose followups --open` is the live backlog.
5. **Recurrence** — `pose recurrence-check` flags task slugs that keep
   failing; the escalation workflow turns them into new rules/workflows.
6. **Knowledge** — handoffs and decision logs with TTL governance carry
   context to the next execution — then the loop feeds planning again.

## Spec lifecycle

```
draft ──(DoR gate)──► in-progress ──(closeout gate)──► done
                │                          │
                └── blocked / superseded / abandoned
```

- **Entry (Definition of Ready):** Intent/Requirements/Technical Plan filled,
  acceptance criteria with stable IDs (`- R<N>:`). `pose check` enforces it
  automatically on the `→ in-progress` transition.
- **Exit (closeout):** `completed_at` stamped and every follow-up
  dispositioned — `[open]`, `[spawned: slug]`, `[covered: slug]`,
  `[duplicate: slug]`, `[done]`, `[wont-do: reason]`. For
  spawned/covered/duplicate the target spec must exist (no "covered" by a
  typo). Open follow-ups declare ownership and a triage SLA —
  `(owner:@alias crit:low|medium|high review:YYYY-MM-DD)` — and every
  declared `R<N>` gets a trace entry (`[satisfied]` with evidence refs,
  `[waived: reason]` or `[withdrawn: reason]`) in the
  `Requirement trace` subsection; `pose followups --overdue` and the MCP
  tool `pose_requirement_trace` project both sides.

## Dependency graph and roadmaps

Specs declare `depends_on` (typed refs: spec slug, `milestone:<roadmap>/<id>`,
`roadmap:<slug>`) and `priority`. `pose check` validates existence and
acyclicity; `pose index` caches the graph (`spec-graph.json`); the MCP tool
`pose_spec_readiness` answers "is this spec eligible to start?" by resolving
the refs for real.

Roadmaps are governed artifacts: milestones form a DAG (`after:`), carry
planned dates (Gantt input — actuals derive from events) and own specs
exclusively (one active roadmap per spec).

## Validation matrix

`.pose/indexes/validation-matrix.json` declares checks per stack (node, go,
rust, java) with per-module overrides and two severities: `required` failures
block; `optional` failures inform. Modes `strict`/`tolerant` decide whether
structural warnings block. `pose init --wizard` seeds modules from a
repository scan.

## Operational memory

`.pose/knowledge/` holds three artifact types — **handoff** (context between
executions), **decision-log** (decisions with a review trigger), **note**
(reusable context) — all with mandatory frontmatter and TTL (max 90 days).
`pose knowledge-check` gates schema and overdue backlog; housekeeping
archives/purges expired entries.

## Schema versioning

The `.pose/` contract itself is versioned (`.pose/schema-version`). The engine
declares `POSE_SCHEMA_VERSION`; `pose check` detects drift and `pose upgrade`
applies sequential idempotent migrations. An instance newer than its engine is
always an error — upgrade the engine, never downgrade the instance.
