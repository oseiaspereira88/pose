---
name: pose-knowledge
description: Use to create or update artifacts under .pose/knowledge, including cross-execution handoffs, decision logs with review triggers, and reusable technical notes. Trigger keywords - knowledge, handoff, decision-log, note, memory, context handoff, pose-maintainers.
when_to_use: Technical context must survive one execution and be resumed by another agent or cycle, especially after feature, bugfix, or review work when a spec or ADR is insufficient.
---

# Skill: pose-knowledge

## Required reading

1. [`.pose/rules/knowledge-governance.md`](../../../.pose/rules/knowledge-governance.md).
2. The knowledge-governance spec present in the installation.

## Artifact types

- **handoff:** partial state and next owner.
- **decision-log:** localized non-ADR decision with a review trigger.
- **note:** reusable technical context such as a debug recipe or curated link.

Use a 30-day default TTL and at most 90 days with justification.

## Steps

1. Create an artifact with `pose new-knowledge <type> <slug> --owner @<team> --ttl-days 30`.
2. Fill Context, Current state, Next checks, Risks, and Next owner; update `source_refs`.
3. Use `--restricted` for restricted content, while still excluding secrets and personal data.
4. Run `pose knowledge-check --strict`.
5. Search active knowledge before related work with `find .pose/knowledge -name '*<topic>*.md' -type f -not -path '*/archive/*'`.
6. Use `knowledge-housekeeping` to list expired artifacts, archive with `--apply`, and purge only after the retention window.

## Restrictions

- Never store secrets, tokens, personal data, or full restricted incident reports.
- Require an owner; reserve `@pose-maintainers` for institutional artifacts.
- Update `last_reviewed_at` only after a real review.

## Output requirements

- Complete artifact frontmatter and body under `.pose/knowledge/`.
- Successful strict knowledge check.
- Reference from the motivating spec or review.
