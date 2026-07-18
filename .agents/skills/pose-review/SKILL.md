---
name: pose-review
description: Use for POSE pull-request or code review to verify controlled scope, preserved contracts, security and observability impact, risk-proportional validation, and escalation where applicable. Trigger keywords - review, code review, PR review, review opinion, ultrareview.
when_to_use: Evaluating your own or another author's diff or PR under POSE. Use before commenting or approving to select rules, inspect validation evidence and prior decisions, and issue an actionable decision.
---

# Skill: pose-review

## Required reading

1. [AGENTS.md](../../../AGENTS.md).
2. [`.pose/workflows/review.md`](../../../.pose/workflows/review.md).
3. Applicable domain rules; security takes precedence in conflicts.

## Steps

1. Classify the change as feature, bugfix, refactor, documentation, or mixed.
2. Select rules with `pose suggest <type> --path <affected-dir>`.
3. Search `.pose/knowledge/` for prior module decisions, accepted risks, and pending follow-ups.
4. Require `pose validate` evidence proportional to risk.
5. Evaluate functional correctness, public contracts, security, observability, performance, and regression.
6. Classify findings as critical, high, medium, or low with evidence and expected action.
7. Run `pose recurrence-check --tolerant --window-days 14`; use recurrence escalation for a matching systemic signal.
8. Create a handoff for accepted residual risk, monitoring, or deferred action.
9. Decide: approved, approved with reservations, or rejected.

## Output requirements

- Completed Rules applied during review section.
- Severity-classified findings with expected actions.
- Clear and actionable final decision.
- Handoff when residual risk is accepted.
