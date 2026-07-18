# Frontmatter contracts

POSE frontmatter is **flat by contract** — inline comma-separated lists, never
multi-line YAML lists. This keeps every artifact parseable by simple
deterministic tooling (and by agents) without a YAML edge-case zoo.

## Spec (`.pose/specs/<slug>/spec.md`)

```yaml
---
slug: my-feature
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-01-15
completed_at:        # stamped on the transition to done
supersedes:          # slug of the superseded spec
depends_on: other-spec, milestone:my-roadmap/m1, roadmap:other-roadmap
priority: 1          # integer >= 0; lower = attack first; never blocks
---
```

Rules enforced by `pose check` / `pose lint-spec`:

- `depends_on` refs must exist; the graph must be acyclic.
- `status: done` requires `completed_at` + a disposition on every follow-up.
- Entering `in-progress` requires the Definition of Ready (`--ready-check`).
- Acceptance criteria use stable IDs (`- R<N>:`); published IDs are never
  renumbered — a withdrawn criterion is marked as withdrawn.

## Roadmap (`.pose/roadmaps/<slug>.md`)

```yaml
---
slug: my-roadmap
status: draft        # draft | active | done | abandoned
created_at: 2026-01-15
depends_on:          # other roadmaps, inline list
---
```

Milestones are sections, not frontmatter:

```markdown
## Milestone: m1
- after:              # milestone ids and/or spec:<slug>, inline list
- target_start: 2026-02-01
- target_due: 2026-02-15
- specs: spec-a, spec-b
```

Enforced: unique spec membership across active roadmaps, milestone/roadmap
DAGs, date sanity, resolvable refs.

## Knowledge (`.pose/knowledge/*.md`)

```yaml
---
type: handoff        # handoff | decision-log | note
owner: "@team-or-person"
sensitivity: normal  # normal | restricted
created_at: 2026-01-15
last_reviewed_at: 2026-01-15
expires_at: 2026-02-14   # TTL <= 90 days, default 30
---
```

## Changelog fragment (`.pose/changelogs/unreleased/<spec>.md`)

```yaml
---
spec: my-feature
category: added      # added | changed | fixed | removed | deprecated | security
breaking: false
refs: PR#123
---
```

The body is 1–3 user-facing sentences. At release time,
`pose release-notes` consolidates fragments into grouped release notes
(breaking changes first).

## Schema version (`.pose/schema-version`)

A single integer line. The engine's `POSE_SCHEMA_VERSION` must be ≥ the
instance's; `pose upgrade` migrates forward, never backward.
