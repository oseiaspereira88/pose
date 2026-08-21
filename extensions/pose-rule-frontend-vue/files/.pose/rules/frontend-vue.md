# Rule: Frontend Vue

## When to consult

Consult this guide for Vue 3, Nuxt, Vite, Pinia, Vue Router, and UI components handling reactive state, props, events, and rendering.

## Required patterns

- Use `<script setup lang="ts">` and Vue 3 Composition API for idiomatic reactivity and type inference.
- Define explicit typed props and events using `defineProps<Props>()` and `defineEmits<Emits>()`.
- Manage shared application state with Pinia stores rather than mutable global objects.
- Keep business logic in composables (`use*`), keeping template components focused on presentation.
- Clean up event listeners, timers, and external subscriptions in `onUnmounted`.
- Ensure SSR / Nuxt hydration safety: guard browser-only APIs (`window`, `localStorage`) inside `onMounted` or `<ClientOnly>`.

## Blocking anti-patterns

- Mutating props directly inside child components instead of emitting update events.
- Destructuring reactive objects (`reactive()` or Pinia stores) without `toRefs()` or `storeToRefs()`, breaking reactivity.
- Using `v-html` with untrusted user input without sanitization (DOMPurify).
- Manipulating the DOM directly with `document.querySelector` instead of template refs (`ref()`).
- Committing secrets or unredacted API credentials into client-side bundles.

## Minimum checks

- Run `npm test` or `pnpm test` (Vitest).
- Run `vue-tsc --noEmit` or `npm run typecheck` without blocking type errors.
- Run `npm run lint` (ESLint with `plugin:vue/vue3-recommended`).
- Run `npm run build` without compilation failures.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
