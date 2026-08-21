# Rule: Backend Python

## When to consult

Consult this guide for API endpoints, domain models, persistence, async background tasks, and service integrations in Python backend applications (FastAPI, Django, Flask, Celery, Litestar).

## Required patterns

- Use explicit type annotations on public functions, classes, and schema boundaries (`typing`, Pydantic dataclasses/models).
- Isolate blocking I/O (file ops, legacy sync DB calls) from asynchronous event loops using threadpools or native async drivers.
- Propagate exceptions with clear domain context and map to standard HTTP status codes.
- Encapsulate database queries and transaction boundaries within dedicated repository or unit-of-work abstractions.
- Use structured logging and never log unredacted PII or sensitive tokens.

## Blocking anti-patterns

- Catching `Exception` or `BaseException` blindly with `pass` without logging or remediation.
- Using mutable default arguments (e.g. `def foo(bar=[])`).
- Performing raw SQL queries with string interpolation or formatting instead of parameterized statements.
- Invoking synchronous blocking calls directly inside `async def` endpoints.
- Leaving database sessions or connections unclosed across request lifecycles.

## Minimum checks

- Run `pytest` across affected test suites.
- Run `ruff check .` or `flake8` without critical findings.
- Run `mypy` or `pyright` type checks without unresolved blocking type errors.
- Run security scanning with `bandit` when modifying security-sensitive scopes.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
