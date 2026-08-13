---
name: pose-review
description: Use for POSE pull-request or code review to verify controlled scope, preserved contracts, security and observability impact, risk-proportional validation, and escalation where applicable. Trigger keywords - review, code review, PR review, review opinion, ultrareview.
when_to_use: Evaluating your own or another author's diff or PR under POSE. Use before commenting or approving to select rules, inspect validation evidence and prior decisions, and issue an actionable decision.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read
---

# Skill: pose-review

## Required reading

1. [AGENTS.md](../../../AGENTS.md).
2. [`.pose/workflows/review.md`](../../../.pose/workflows/review.md).
3. Applicable domain rules; security takes precedence in conflicts.

## Steps

1. Classify the change as feature, bugfix, refactor, documentation, or mixed.
2. Resolve `pose review-plan <scope> --explain`; stop on blockers and retain its `plan_digest`.
3. Run required tools in plan order and record why each recommended tool was used or skipped.
4. Select rules with `pose suggest review --path <affected-dir>` for every mapped component.
5. Search `.pose/knowledge/` for prior module decisions, accepted risks, and pending follow-ups.
6. Require `pose validate` evidence proportional to risk.
7. Evaluate every required plan criterion, including cross-component boundaries.
8. Classify findings as critical, high, medium, or low with evidence and expected action.
9. Run `pose recurrence-check --tolerant --window-days 14`; use recurrence escalation for a matching systemic signal.
10. Create a handoff with `pose new-knowledge handoff <slug>` for accepted residual risk, monitoring, or deferred action.
11. Record with `pose review record <scope> ... --plan-digest <sha256> --apply`, then run `pose review-check <scope>`.
12. Decide: approved, approved with reservations, or rejected.

## Output requirements

- Completed Rules applied during review section.
- Effective plan digest and dispositions for required and recommended tools.
- Severity-classified findings with expected actions.
- Clear and actionable final decision.
- Handoff when residual risk is accepted.
