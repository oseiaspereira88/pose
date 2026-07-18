# Rule: Backend Go

## When to consult

Consult this guide for HTTP handlers, domain services, persistence, concurrency, and integrations between Go services.

## Required patterns

- Handle and propagate errors with enough context for diagnosis.
- Validate handler input and return HTTP status codes consistent with the contract.
- Keep business rules in service or domain layers, not in transport code.
- Honor `context.Context` for timeouts and cancellation.
- Put data access behind clear interfaces that support testing.
- Use structured logs without exposing sensitive data.

## Blocking anti-patterns

- Ignoring error returns, including `_` for critical errors.
- Using `panic` in ordinary request flows instead of controlled error handling.
- Coupling handlers directly to concrete repository implementations without an abstraction.
- Running unbounded or unpaginated queries in potentially large endpoints.
- Introducing concurrency without adequate synchronization or race protection.

## Minimum checks

- Run `go test ./...` in the affected scope.
- Run `go test -race ./...` when concurrency changes.
- Run `go vet ./...` without blocking findings.
- Run the applicable Go lint check, such as `golangci-lint`, without critical errors.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
