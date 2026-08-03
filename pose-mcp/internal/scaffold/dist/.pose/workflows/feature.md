# Workflow: Feature

## Objective

Deliver a production feature with clear scope, incremental implementation, and deterministic validation.

## Preconditions

- Make the business requirement and acceptance criteria explicit.
- Identify the affected directories.
- Create or update the related spec under `.pose/specs/`.
- Map technical dependencies and initial risks.

## Execution checklist

1. Confirm the objective, constraints, and affected public contracts.
2. Map affected modules and read relevant local instructions.
3. Search `.pose/knowledge/` for relevant handoffs, notes, and decision logs; cite consulted artifacts in the spec.
4. Review or create the spec with intent, requirements, and tasks.
5. Declare exact source-tree actions under `### Artifacts`; keep declaration separate from Git-observed evidence.
6. Plan small, reversible delivery increments.
7. Implement incrementally and validate each meaningful step.
8. Run `pose artifact-check --spec <slug> --from <base> --to <head> --strict` against an immutable attributed change set.
9. When the spec declares `delivers`, run validation to a structured result and require `pose surface-check --spec <slug> --strict`; build/unit alone never prove composition or reachability.
10. Run applicable deterministic checks: test, lint, typecheck, and build.
11. Review security, observability, and operational-documentation impact.
12. Create a reusable handoff with `pose new-knowledge handoff <slug>` when another execution needs partial state, a pending decision, or a follow-up; link the spec through `source_refs`.
13. Summarize the result, residual risks, and next steps.
14. Run a separate review pass and record its immutable attempt with `pose review record spec:<slug> ... --apply`.
15. Require `pose closeout-check spec:<slug>` before applying `pose close spec:<slug>`; remediate and re-review when the digest or findings block closure.
16. Complete follow-up and changelog disposition, then pass `pose lint-spec <slug> --strict`.

## Required outputs

- Summarize changes by module and file.
- Attach commands and status for executed validation.
- Update specs and documentation when behavior changes.
- List residual risks with mitigation or a follow-up plan.

## Definition of done

- Meet all acceptance criteria with verifiable evidence.
- Preserve public contracts or document intentional changes.
- Pass every relevant deterministic check.
- Keep scope controlled and exclude unrelated refactors.
- Close the spec with a current approved review, `status: done`, `completed_at`, and dispositioned follow-ups.

## Planner mode

**Objective:** turn intent into an executable plan with controlled scope, explicit risks, and defined validation.

- **Focus:** understand the problem precisely; delimit modules and contracts; sequence verifiable increments; define deterministic checks early.
- **Anti-patterns:** omit constraints or dependencies; create a plan too large for incremental validation; ignore existing specs and workflows; assume risk is absent without evidence.
- **Handoff:** prioritize small steps, identify target files and boundaries, assign mandatory checks per step, and highlight residual implementation risks.

## Implementer mode

**Objective:** execute the plan through cohesive, production-safe changes with continuous validation.

- **Focus:** make the smallest high-impact changes; follow scope and local conventions; validate every relevant increment; communicate trade-offs and residual risks.
- **Anti-patterns:** expand scope with unsolicited refactors; change public contracts without specs or docs; accumulate large unvalidated changes; fix symptoms without investigating root cause.
- **Handoff:** summarize the diff and rationale, executed commands and results, limitations and follow-ups, and review-sensitive areas.
