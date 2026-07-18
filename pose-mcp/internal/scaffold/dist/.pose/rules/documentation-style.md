# Rule: Documentation Style

## When to consult

Consult this guide for process documentation, rules, specs, workflows, and operational instructions.

## Required conventions

- Write instructions in the imperative mood and begin them with action verbs.
- Keep bullets short and limited to one idea.
- Avoid duplicating sections with the same purpose across files.
- Link to the single source of truth instead of copying its content.
- Use `check`, `spec`, and `workflow` consistently.
- State each instruction's scope explicitly to reduce ambiguity.

## Examples: good and bad

### Redundancy

- **Good:** "Update review criteria in `.pose/workflows/review.md` and reference that workflow from the root AGENTS file."
- **Bad:** "Repeat review criteria in AGENTS, the workflow, and every related spec."

### Ambiguous reference

- **Good:** "Run the lint check described by the review workflow."
- **Bad:** "Run that standard validation before pushing."

## Quick editorial checklist

- Language is imperative and direct.
- Bullets are short and non-overlapping.
- Files do not duplicate sections.
- `check`, `spec`, and `workflow` are used consistently.
- References point to explicit files or paths.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
