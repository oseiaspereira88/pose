---
name: pose-recurrence-escalation
description: Use when recurrence-check flags a recurring task above threshold to investigate systemic cause, propose a rule or workflow change, document the decision, and close the loop. Trigger keywords - recurrence, recurring pattern, recurrence-escalation, escalation, systemic debt.
when_to_use: Manual or CI recurrence-check flags at least one key above threshold. Use before labeling the problem intermittent or suppressing the signal.
---

# Skill: pose-recurrence-escalation

## Required reading

1. [`.pose/workflows/recurrence-escalation.md`](../../../.pose/workflows/recurrence-escalation.md).
2. [`.pose/rules/_base-recurrence.md`](../../../.pose/rules/_base-recurrence.md).
3. Matching JSONL history under `.pose/reports/history/`.

## Steps

1. Confirm the signal with `pose recurrence-check --tolerant --window-days 30 --threshold 3`.
2. Aggregate workflow and task outcomes with `pose stats workflows --since-days 30` and `pose stats tasks --since-days 30 --json`.
3. Investigate common modules, root causes, repeatedly violated rules, and missing workflow prevention.
4. Add or adjust the cheapest effective rule or workflow; promote an optional check only with at least 95 percent success over four weeks.
5. Record the decision with `pose new-knowledge decision-log escalation-<task-slug> --owner @<owner> --ttl-days 90`.
6. Update the related spec and link the decision log.

## Output requirements

- Decision log linked to historical outcomes.
- Rule, workflow, or matrix change that addresses the systemic cause.
- Expected successful strict recurrence check after the next cycle.
- Updated escalation workflow when the escalation pattern is new.
