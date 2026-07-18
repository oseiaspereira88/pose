# POSE skills index

Skills package recurring workflows in a format agents can consume. Claude Code
uses the symlinks under [`.claude/skills/`](../../.claude/skills/); other agents
read `.agents/skills/` directly.

Each `SKILL.md` uses YAML frontmatter with `name`, `description`, and
`when_to_use`, followed by Required reading, Steps, and Output requirements.

## Catalog

| Skill | Task type | Primary workflow | Base rules |
|---|---|---|---|
| [pose-feature](pose-feature/SKILL.md) | Feature or non-trivial refactor | [feature.md](../../.pose/workflows/feature.md) | security, documentation-style, and domain rules |
| [pose-spec-closeout](pose-spec-closeout/SKILL.md) | Close a completed spec | [feature.md](../../.pose/workflows/feature.md) | documentation-style |
| [pose-bugfix](pose-bugfix/SKILL.md) | Defect correction | [bugfix.md](../../.pose/workflows/bugfix.md) | security, documentation-style, and domain rules |
| [pose-review](pose-review/SKILL.md) | PR or code review | [review.md](../../.pose/workflows/review.md) | security, documentation-style, and domain rules |
| [pose-adr](pose-adr/SKILL.md) | Architectural decision | Motivating workflow | security and documentation-style |
| [pose-test-plan](pose-test-plan/SKILL.md) | Pre-implementation test plan | feature or bugfix | security, documentation-style, and domain rules |
| [pose-doc-update](pose-doc-update/SKILL.md) | Editorial documentation | [documentation-update.md](../../.pose/workflows/documentation-update.md) | documentation-style |
| [pose-knowledge](pose-knowledge/SKILL.md) | Handoff, decision log, or note | Any workflow closeout | knowledge-governance and documentation-style |
| [pose-recurrence-escalation](pose-recurrence-escalation/SKILL.md) | Escalation after recurrence detection | [recurrence-escalation.md](../../.pose/workflows/recurrence-escalation.md) | security and documentation-style |

## Machine-readable routing

```bash
pose suggest <task-type> [--path <dir>] [--json]
```

The source of truth is [`.pose/indexes/task-map.json`](../../.pose/indexes/task-map.json).
`pose check` validates referenced workflows, skills, and rules.

## Scope rule

Load only the skill for the current task type and the `AGENTS.md` files needed
for affected paths. Do not read the entire catalog by default.
