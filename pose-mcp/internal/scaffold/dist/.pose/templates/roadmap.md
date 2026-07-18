---
slug: <roadmap-slug>
status: draft        # draft | active | done | abandoned
created_at: <YYYY-MM-DD>
depends_on:          # prerequisite roadmaps, inline list: other-roadmap-a, other-roadmap-b
---

# Roadmap: <roadmap-slug>

> Governed roadmap. The frontmatter is flat (POSE contract); each milestone is
> a `## Milestone: <id>` section with flat bullets. Ordering between
> milestones comes from `- after:`; dates are PLANNING input (Gantt) — actuals
> derive from events and are never edited here. A spec belongs to at most ONE
> active roadmap (`pose check` validates).
>
> Free prose is welcome outside milestone sections: context, risks, release
> cut criteria.

## Milestone: <milestone-id>
- after:                       # ids of milestones in this roadmap and/or spec:<slug>, inline list
- target_start:                # optional, YYYY-MM-DD
- target_due:                  # optional, YYYY-MM-DD
- specs:                       # spec slugs, inline list: spec-a, spec-b
