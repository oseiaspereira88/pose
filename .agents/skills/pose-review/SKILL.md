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
2. Resolve `pose review bundle <scope> --explain` to inspect effective criteria, components and tool requirements.
3. Run active required tools first, record why each recommended tool was used or skipped, and keep completion tools deferred until attestation.
4. Select rules with `pose suggest review --path <affected-dir>` for every mapped component.
5. Search `.pose/knowledge/` for prior module decisions, accepted risks, and pending follow-ups.
6. Require deterministic `pose validate --strict` evidence matching delivery targets.
7. Evaluate every required plan criterion, including cross-component boundaries.
8. Classify findings as critical, high, medium, or low with evidence and expected action.
9. Run `pose recurrence-check --tolerant --window-days 14`; use recurrence escalation for a matching systemic signal.
10. Create a handoff with `pose new-knowledge handoff <slug>` for accepted residual risk, monitoring, or deferred action.
11. Seal the review subject with `pose review bundle <scope> --seal`.
12. Attest the bundle automatically using `pose review auto-attest <bundle-id> --reviewer agent:reviewer-subagent --apply` (or `pose review attest` with explicit findings when requesting changes).
13. Run `pose review verify <scope>` and `pose review-check <scope>`; close only when both confirm a valid, approved attestation.
14. Decide: approved, approved with reservations, changes requested, or rejected.
15. When Contributor Mode is active, if review uncovers POSE linter false-positives, diagnostic frictions, or rule gaps, stage an improvement proposal with `pose contribute stage --type enhancement --title "<summary>"`.

## Output requirements

- Completed Rules applied during review section.
- Bundle/plan digests and dispositions for required, recommended and deferred completion tools.
- Severity-classified findings with expected actions.
- Clear and actionable final decision.
- Handoff when residual risk is accepted.
