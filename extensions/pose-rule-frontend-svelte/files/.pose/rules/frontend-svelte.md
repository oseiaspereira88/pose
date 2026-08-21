# Rule: Frontend Svelte

## When to consult

Consult this guide for Svelte 5, SvelteKit, runes, server-side rendering, and UI component workflows in Svelte web applications.

## Required patterns

- Use Svelte 5 Runes (`$state`, `$derived`, `$props`, `$effect`) for fine-grained reactivity and component contracts.
- Load data securely via SvelteKit `+page.server.ts` or `+layout.server.ts` for database/authenticated operations, and `+page.ts` for universal fetching.
- Validate incoming form actions with schemas (e.g. Zod, Superforms) before applying state mutations.
- Isolate browser-only APIs (`window`, `navigator`) inside `$effect` blocks or check `browser` from `$app/environment`.
- Use `$derived.by()` for complex memoized computations to prevent unnecessary re-renders.

## Blocking anti-patterns

- Using `$effect` to mutate other reactive state (causing cascading updates or infinite reactivity loops).
- Using `{@html ...}` with unescaped or untrusted user content without sanitization.
- Leaking private environment variables into `$env/static/public` or client components.
- Relying on global mutable state across universal `load` functions, causing multi-tenant data leaks in SSR.
- Mutating `$props` directly inside child components.

## Minimum checks

- Run `npm test` or `pnpm test` (Vitest / Playwright).
- Run `svelte-check --tsconfig ./tsconfig.json` without blocking diagnostics.
- Run `npm run lint` (ESLint with `eslint-plugin-svelte`).
- Run `npm run build` without compilation failures.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
