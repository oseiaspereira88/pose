# Regra: Frontend Vue

## Quando consultar

Consulte este guia para aplicações Vue 3, Nuxt, Vite, Pinia, Vue Router e componentes de interface que tratam estado reativo, props, eventos e renderização.

## Padrões obrigatórios

- Use `<script setup lang="ts">` e Composition API do Vue 3 para reatividade idiomática e inferência de tipos.
- Defina props e eventos tipados explicitamente usando `defineProps<Props>()` e `defineEmits<Emits>()`.
- Gerencie o estado compartilhado da aplicação com stores Pinia em vez de objetos globais mutáveis.
- Mantenha a lógica de negócio em composables (`use*`), focando componentes de template na apresentação.
- Limpe event listeners, timers e assinaturas externas no hook `onUnmounted`.
- Garanta segurança de hidratação SSR / Nuxt: proteja APIs exclusivas do navegador (`window`, `localStorage`) dentro de `onMounted` ou `<ClientOnly>`.

## Anti-padrões bloqueantes

- Modificar props diretamente dentro de componentes filhos em vez de emitir eventos de atualização.
- Desestruturar objetos reativos (`reactive()` ou stores Pinia) sem `toRefs()` ou `storeToRefs()`, quebrando a reatividade.
- Usar `v-html` com dados não confiáveis de usuários sem sanitização prévia (DOMPurify).
- Manipular o DOM diretamente com `document.querySelector` em vez de template refs (`ref()`).
- Commitar segredos ou credenciais de API em bundles que são servidos no lado do cliente.

## Checagens mínimas

- Rodar `npm test` ou `pnpm test` (Vitest).
- Rodar `vue-tsc --noEmit` ou `npm run typecheck` sem erros bloqueantes de tipagem.
- Rodar `npm run lint` (ESLint com `plugin:vue/vue3-recommended`).
- Rodar `npm run build` sem falhas de compilação.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
