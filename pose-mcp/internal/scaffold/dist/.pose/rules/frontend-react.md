# Rule: Frontend React

## When to consult

Consult this guide for UI work, React components, client state, accessibility, forms, and frontend API integrations.

## Required patterns

- Keep components small, single-purpose, and explicit about typed props.
- Declare complete dependencies in effects (`useEffect`) and clean them up when needed.
- Compute derived state instead of duplicating it unnecessarily in `useState`.
- Handle loading, error, and success states explicitly in asynchronous flows.
- Preserve basic accessibility through semantic HTML, field labels, and keyboard navigation.
- Encapsulate backend communication in a reusable service or hook layer.

## Blocking anti-patterns

- Spreading business rules directly across visual components.
- Using incorrect effect dependencies that cause stale data or infinite loops.
- Using `any` broadly to bypass type errors.
- Hiding API failures from both users and observable logs.
- Breaking basic accessibility with unlabeled fields or controls without accessible names.

## Minimum checks

- Run frontend lint without errors.
- Run frontend type checking without errors.
- Run unit or integration tests for the changed flows.
- Complete the frontend build successfully.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
