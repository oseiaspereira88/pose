# Rule: Backend .NET

## When to consult

Consult this guide for ASP.NET Core Web APIs, minimal APIs, background workers, Entity Framework Core repositories, and gRPC services in C# / .NET.

## Required patterns

- Use async/await all the way down; propagate `CancellationToken` through HTTP endpoints and repository queries.
- Register dependencies with correct lifetimes (Transient, Scoped, Singleton) and avoid captive dependencies (e.g. Scoped injected into Singleton).
- Use `.AsNoTracking()` for read-only Entity Framework Core queries to reduce allocation overhead.
- Validate request models at controller or pipeline boundaries (FluentValidation, DataAnnotations).
- Use structured logging via `ILogger<T>` with message templates rather than string concatenation.

## Blocking anti-patterns

- Calling `.Result` or `.Wait()` on `Task` / `ValueTask` (sync-over-async causing threadpool starvation and deadlocks).
- Capturing `DbContext` instances inside Singleton services or long-lived background loops.
- Performing raw SQL execution with string interpolation rather than parameterized `FromSqlInterpolated`.
- Exposing internal domain or persistence entities directly over public API contracts without DTOs.
- Ignoring or discarding exceptions without proper diagnostic logging.

## Minimum checks

- Run `dotnet test` across affected solution test projects.
- Run `dotnet format --verify-no-changes` or IDE code analysis without fatal warnings.
- Run `dotnet build` with TreatWarningsAsErrors where enabled.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
