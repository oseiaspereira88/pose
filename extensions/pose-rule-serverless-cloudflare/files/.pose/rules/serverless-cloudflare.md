# Rule: Serverless Cloudflare Workers

## When to consult

Consult this guide for Cloudflare Workers, Pages Functions, Durable Objects, KV, D1 SQL, and R2 object storage implementations.

## Required patterns

- Respect edge runtime limits: keep request handler memory within plan constraints and use streaming responses for large payloads.
- Secure environment bindings: access secrets exclusively through `env` parameters rather than global variables.
- Set explicit caching headers (`Cache-Control`, `CF-Cache-TTL`) on static or cacheable responses.
- Handle unhandled promise rejections and wrap asynchronous fetch event handlers in try/catch blocks.
- Use `ctx.waitUntil()` for non-blocking asynchronous side effects that must outlive the response cycle.

## Blocking anti-patterns

- Relying on Node.js native binary modules or unsupported OS-level APIs in edge workers.
- Storing sensitive credentials directly inside `wrangler.toml`, `wrangler.json`, or source code.
- Accumulating unbounded in-memory state or global caches that leak across warm isolate instances.
- Performing synchronous blocking computations that exceed the edge CPU execution limit.
- Failing to handle D1 or KV transient errors with appropriate retry or fallback strategies.

## Minimum checks

- Run `wrangler check` or `npx wrangler check` in the worker module.
- Run unit tests (Vitest / Jest with `@cloudflare/workers-types` or `miniflare`).
- Run `wrangler types` to ensure TypeScript binding definitions are up to date.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
