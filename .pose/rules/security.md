# Rule: Security

## When to consult

Consult this guide for authentication, authorization, sensitive data, external integrations, dependencies, and attack surfaces.

## Required patterns

- Apply least privilege to resource and credential access.
- Store secrets only in appropriate secret-management mechanisms, never in code.
- Validate and sanitize external input for its execution context.
- Keep personal and confidential data out of plaintext logs and metrics.
- Evaluate new dependencies for maintenance, licensing, and known vulnerabilities.
- Test both positive and negative authentication and authorization cases.

## Blocking anti-patterns

- Committing credentials, tokens, keys, or secrets in any versioned artifact.
- Disabling TLS or certificate verification without a formally documented mitigation.
- Trusting client input for authorization decisions.
- Executing dynamic commands without adequate validation and escaping.
- Ignoring critical security alerts without an approved exception record.

## Minimum checks

- Run secret scanning on the changed diff or scope.
- Run the module's applicable dependency vulnerability scanner.
- Pass the relevant authentication and authorization tests.
- Complete a review for sensitive-data exposure in logs and configuration.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
