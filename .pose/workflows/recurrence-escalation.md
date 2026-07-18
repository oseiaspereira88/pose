# Workflow: Recurrence Escalation

## Objective

Trigger a systemic correction when recurring rework is not covered by current workflows.

## Preconditions

- Maintain period-based incident and rework records classified by domain and cause.
- Review existing workflows under `.pose/workflows/` to avoid duplication.
- Obtain area-owner confirmation that process escalation is necessary.

## Required recurrence metric

- **Name:** `recurrence_rework_uncovered`
- **Definition:** recurring incidents or rework during the period whose root cause is not covered by a current workflow.
- **Formula:** `uncovered_recurring_incidents / period`
- **Minimum dimensions:** domain (`frontend-react`, `backend-go`, `kubernetes`, `security`, `documentation-style`) and cause (`process`, `contract`, `implementation`, `validation`).

## Activation threshold

Activate this workflow when any condition is met in a rolling 30-day period:

- At least three uncovered recurring incidents in one domain.
- At least five uncovered recurring incidents across domains.
- Growth over two consecutive periods compared with the preceding 30 days.

## Execution checklist

1. Consolidate 30 days of `recurrence_rework_uncovered` evidence.
2. Confirm that no current workflow covers the pattern and record the gap.
3. Create `.pose/workflows/<name>.md` with scope, preconditions, checks, and outputs.
4. Link the new workflow from applicable domain rules and from the review workflow when relevant.
5. Update the related spec with rationale, acceptance criteria, and residual risks.
6. Define an owner, pilot window, and pilot success criteria.
7. Run deterministic checks for every changed file.
8. Record the post-pilot decision: keep, adjust, or discard the workflow.

## Required rule linkage

Select rules cumulatively for every affected domain:

- `.pose/rules/security.md`
- `.pose/rules/backend-go.md`
- `.pose/rules/frontend-react.md`
- `.pose/rules/kubernetes.md`
- `.pose/rules/documentation-style.md`
- `.pose/rules/knowledge-governance.md` when knowledge or process artifacts change

Apply the most restrictive rule when they conflict.

## Adoption review

Review the pilot after 45 days:

- Compare recurrence volume before and after activation by domain.
- Require at least a 30 percent reduction in the target domain.
- Evaluate operational cost, execution time, and evidence quality.
- Issue a formal `keep`, `adjust`, or `discard` decision.
- Open an owned, dated follow-up with an exit criterion for `adjust` or `discard`.

## Required outputs

- Evidence that the metric crossed its activation threshold.
- A published and referenced specialized workflow.
- An explicit map of applied rules.
- Pilot-review results and final decision.
- Residual risks and a mitigation plan.
