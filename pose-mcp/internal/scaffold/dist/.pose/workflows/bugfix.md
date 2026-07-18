# Workflow: Bugfix

## Objective

Fix the root cause with the smallest possible impact, regression coverage, and operational safety.

## Preconditions

- The failure is reproduced (or objective evidence of the defect is recorded).
- The bug scope and affected components are identified.
- There is a testable root-cause hypothesis.
- There is a validation plan that prevents regression.

## Execution checklist

1. Reproduce the problem and define an observable failure mode.
2. **Consult `.pose/knowledge/`** for earlier incidents or handoffs in the same module or failure pattern; reuse an existing diagnosis when available.
3. Isolate the root cause and map collateral impact.
4. Define the smallest safe fix and a rollback plan.
5. Implement the fix as a cohesive change, without parallel refactoring.
6. Add or adjust a regression test when applicable.
7. Run relevant deterministic checks (`test`, `lint`, `typecheck`, `build`).
8. Validate that the defect is gone and adjacent behavior is preserved.
9. **Create a decision log** in `.pose/knowledge/` when the root cause reveals systemic debt or a trade-off with future impact (`pose new-knowledge decision-log <slug>`).
10. Record residual risks and post-fix monitoring.

## Required outputs

- Description of the defect, root cause, and fix approach.
- Evidence of regression coverage through a test or equivalent validation.
- Results of the executed checks.
- Residual risks, monitoring plan, and rollback plan when needed.

## Definition of done

- The defect no longer reproduces in the target scenario.
- Regression is covered by a suitable deterministic test or validation.
- No behavior outside the scope changed inadvertently.
- Relevant checks completed successfully.

## Execution — implementer mode

**Objective:** fix the root cause with minimal changes, without expanding scope.

- **Focus:** isolate the root cause before any fix; make a cohesive change without parallel refactoring; provide regression coverage before merge; clearly communicate the trade-off between a minimal fix and systemic prevention.
- **Anti-patterns:** fix a symptom without investigating the cause; mix a bugfix with an unrequested refactor; change a public contract to hide the defect; accumulate changes without validation checkpoints.
- **Handoff:** a surgical diff with rationale; executed regression test; residual risk and monitoring; review attention points, especially code near the fix.
