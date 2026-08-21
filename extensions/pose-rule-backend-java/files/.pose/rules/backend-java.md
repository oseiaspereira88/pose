# Rule: Backend Java

## When to consult

Consult this guide for Spring Boot, Quarkus, Micronaut, Jakarta EE, or Kotlin backend services handling HTTP requests, messaging, domain logic, and persistence.

## Required patterns

- Enforce clear architectural layers: Controllers/Endpoints -> Application Services -> Domain Services -> Repositories.
- Use constructor injection for dependencies; avoid field injection with `@Autowired`.
- Ensure all closable resources (streams, connections, transactions) use `try-with-resources`.
- Explicitly declare transaction boundaries (`@Transactional`) with appropriate propagation and isolation levels.
- Annotate nullability contracts (`@NonNull`, `@Nullable`) or leverage Kotlin null-safety.
- Use structured logging via SLF4J without logging sensitive credentials or tokens.

## Blocking anti-patterns

- Catching `Throwable`, `Error`, or `Exception` generically and swallowing it without rethrowing or logging.
- Using `System.out.println` or `e.printStackTrace()` instead of structured logger instances.
- Performing unbounded database queries without pagination or slice limits.
- Holding database transactions open during long remote HTTP or third-party service calls.
- Sharing mutable state across concurrent threads without proper synchronization or concurrent collections.

## Minimum checks

- Run `./gradlew test` or `mvn test` in the affected modules.
- Run Checkstyle, SpotBugs, or Spotless linting according to module configuration.
- Ensure all builds pass with zero compilation errors or fatal warnings.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
