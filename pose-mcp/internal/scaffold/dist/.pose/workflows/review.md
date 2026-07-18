# Workflow: Review

## Objective

Verify that a change is correct, production-safe, and aligned with its approved scope and specs.

## Preconditions

- Make the final diff readable through cohesive commits or batches.
- Provide requirement and spec context.
- Attach implementer validation evidence, including `./pose validate` output.
- Define acceptance criteria and expected risk.

## Execution checklist

1. Confirm the objective and approved scope.
2. Select applicable rules explicitly and record them in the review.
3. Resolve rule conflicts with the most restrictive option, prioritizing security for exposure, authorization, and integrity risks.
4. Consult `.pose/knowledge/` for relevant handoffs, accepted risks, follow-ups, and decision logs.
5. Check compliance with specs, contracts, and local instructions.
6. Review functional correctness and edge cases.
7. Evaluate security, observability, and performance risks.
8. Require validation evidence from `.pose/indexes/validation-matrix.json` proportional to risk.
9. Identify regression and rollout or rollback risks.
10. Classify findings by severity and propose objective actions.
11. Create and link a handoff when findings result in accepted risk, post-merge monitoring, or deferred work.
12. Issue a final decision: approved, approved with reservations, or rejected.

## Required rule selection

Attach this section to the review:

```md
## Rules applied during review
- Change type: <feature|bugfix|refactor|documentation-update|mixed>
- Workflow consulted: `.pose/workflows/<file>.md`
- Rules selected:
  - [ ] `.pose/rules/security.md`
  - [ ] `.pose/rules/backend-go.md`
  - [ ] `.pose/rules/frontend-react.md`
  - [ ] `.pose/rules/kubernetes.md`
  - [ ] `.pose/rules/documentation-style.md`
  - [ ] `.pose/rules/knowledge-governance.md` when knowledge or process changes
- Rationale for each selected rule: <one line per item>
- Rules not applicable: <list and explain>
```

## Recurrence escalation

- Trigger `.pose/workflows/recurrence-escalation.md` when recurring rework is uncovered and the threshold is met.
- Require `recurrence_rework_uncovered` evidence for the 30-day period.
- Require explicit links from the specialized workflow to applicable domain rules.
- Require a formal keep, adjust, or discard decision after the 45-day pilot.

## Minimum domain coverage

- React UI: apply `frontend-react`, `security`, and `documentation-style`.
- Go services: apply `backend-go`, `security`, and `documentation-style`.
- Cluster deployment or infrastructure: apply `kubernetes`, `security`, and `documentation-style`.
- Process, spec, workflow, rule, or report: apply `documentation-style`, `knowledge-governance`, and security when sensitive data is involved.
- Cross-stack changes: apply every touched domain rule cumulatively.

## Domain review checklist

### Security

- Confirm authentication, authorization, and least privilege when applicable.
- Verify that code, configuration, manifests, docs, and logs contain no secrets.
- Require applicable vulnerability and secret-scanning evidence.

### Contracts

- Confirm compatibility of public HTTP, event, schema, CLI, and file contracts.
- Validate forward and backward compatibility for rollout and rollback.
- Require a spec update for contract changes.

### Observability

- Verify structured logs and metrics without sensitive data.
- Confirm that probes, health checks, and alerts reflect real behavior.
- Preserve enough traceability for post-deployment diagnosis.

### Validation

- Require lint, typecheck, test, and build coverage proportional to risk.
- Require executed `./pose validate` evidence and relevant results.
- Record environment limitations and residual validation risks.

## Editorial checklist

- Use imperative, actionable instructions.
- Keep bullets short and avoid duplicated sections.
- Use `check`, `spec`, and `workflow` consistently.
- Reference explicit files and paths.

## Required outputs

- A review decision with rationale.
- A completed rule-selection section.
- Findings with severity, evidence, and recommendation.
- Recurrence classification by domain and cause with preventive actions.
- An explicit statement about public contracts and compatibility.
- Residual risks and safe-deployment conditions.
- References to executed checks and collected evidence.

## Complete multi-rule review example

```md
## Review Summary
- Decision: approved with reservations
- Change type: feature (Go API, React UI, and Helm)
- Workflow: `.pose/workflows/feature.md`

## Rules applied during review
- `.pose/rules/backend-go.md`: verified handlers, context, and error handling.
- `.pose/rules/frontend-react.md`: verified accessibility and explicit loading and error states.
- `.pose/rules/kubernetes.md`: verified resources, probes, and immutable images.
- `.pose/rules/security.md`: verified authorization and absence of secrets.
- `.pose/rules/documentation-style.md`: verified editorial consistency.

## Checks and evidence
- `./pose validate`: passed
- `go test ./...`: passed in the backend module
- `pnpm lint && pnpm test`: passed in the frontend module
- `helm template` and `kubectl apply --dry-run=client`: passed

## Contracts and compatibility
- Preserved `POST /v1/storage`.
- Added optional `retentionDays`, preserving backward compatibility.

## Findings
- Medium: missing queue-saturation alert; add a metric and alert before production.
- Low: frontend error lacks request-id context; improve observable UX.

## Residual risks
- Review did not simulate real cluster load.
- Increase monitoring during the first 24 hours.
```

## CI failure interpretation

- Treat `POSE check (strict)` as merge-blocking on protected branches.
- Treat failed required checks in `POSE validate (strict)` as merge-blocking.
- Attach pipeline artifacts: `pose-check.log`, `pose-validate.latest.log`, and the POSE report.
- Classify optional-check failures as quality reservations with an owned remediation plan.
- Reject changes that violate specs, public contracts, security, or safe rollout.

## Phased rollout for unready modules

- Enforce the current required checks immediately.
- Use `moduleOverrides` to phase adoption without weakening global structural and required gates.
- Promote optional checks to required by module with an owner and agreed window.

### Optional-to-required promotion protocol

- Select a pilot domain and map candidate checks with owners and explicit risks.
- Measure success for four weeks and require a baseline of at least 95 percent.
- Promote only eligible domains through `moduleOverrides`.
- Update the validation matrix and quality policy in the same change set.
- Monitor regressions and adjust rollout without removing established required gates.
- Update specs and rules when rollout changes merge acceptance criteria.

## Definition of done

- Resolve or formally accept every critical and high finding.
- Make the final decision clear and actionable.
- Support quality and security conclusions with evidence.
- Preserve approved scope without unjustified drift.

## Reviewer mode

**Objective:** evaluate technical quality and production readiness with emphasis on correctness, risk, and scope.

- **Focus:** functional correctness, spec consistency, regression and security risk, operability, validation quality, and actionable feedback.
- **Anti-patterns:** approve without sufficient evidence; review style while ignoring functional risk; request unrelated changes; block progress on subjective preference.
- **Handoff:** state the decision, severity and evidence for each finding, safe merge or deployment conditions, accepted risks, and recommended monitoring.
