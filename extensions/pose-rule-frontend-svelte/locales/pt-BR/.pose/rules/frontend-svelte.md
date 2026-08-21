# Regra: Frontend Svelte

## Quando consultar

Consulte este guia para aplicações Svelte 5, SvelteKit, runes, server-side rendering e componentes de interface web em Svelte.

## Padrões obrigatórios

- Use Runes do Svelte 5 (`$state`, `$derived`, `$props`, `$effect`) para reatividade granular e contratos de componentes.
- Carregue dados com segurança via `+page.server.ts` ou `+layout.server.ts` do SvelteKit para operações autenticadas ou de banco, e `+page.ts` para fetch universal.
- Valide form actions recebidas com schemas (ex: Zod, Superforms) antes de aplicar mutações de estado.
- Isole APIs exclusivas do navegador (`window`, `navigator`) dentro de blocos `$effect` ou verifique `browser` de `$app/environment`.
- Use `$derived.by()` para computações memoizadas complexas a fim de evitar re-renderizações desnecessárias.

## Anti-padrões bloqueantes

- Usar `$effect` para alterar outro estado reativo (provocando atualizações em cascata ou loops infinitos de reatividade).
- Usar `{@html ...}` com conteúdo de usuário não sanitizado.
- Vazar variáveis de ambiente privadas em `$env/static/public` ou componentes de cliente.
- Usar estado mutável global em funções `load` universais, provocando vazamento de dados entre sessões no SSR.
- Alterar `$props` diretamente dentro de componentes filhos.

## Checagens mínimas

- Rodar `npm test` ou `pnpm test` (Vitest / Playwright).
- Rodar `svelte-check --tsconfig ./tsconfig.json` sem diagnósticos bloqueantes.
- Rodar `npm run lint` (ESLint com `eslint-plugin-svelte`).
- Rodar `npm run build` sem falhas de compilação.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
