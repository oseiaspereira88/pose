# Regra: Serverless Cloudflare Workers

## Quando consultar

Consulte este guia para implementações de Cloudflare Workers, Pages Functions, Durable Objects, KV, D1 SQL e armazenamento de objetos R2.

## Padrões obrigatórios

- Respeite os limites do runtime de borda: mantenha o consumo de memória dentro das cotas e use responses em streaming para grandes payloads.
- Proteja bindings de ambiente: acesse segredos exclusivamente através do objeto `env` em vez de variáveis globais.
- Defina headers de cache explícitos (`Cache-Control`, `CF-Cache-TTL`) em respostas estáticas ou passíveis de cache.
- Trate rejeições assíncronas e envolva handlers de fetch em blocos try/catch.
- Use `ctx.waitUntil()` para efeitos colaterais assíncronos que precisam continuar após o envio da resposta.

## Anti-padrões bloqueantes

- Depender de módulos binários nativos do Node.js ou APIs de sistema operacional não suportadas no runtime de borda.
- Armazenar segredos ou credenciais sensíveis diretamente no `wrangler.toml`, `wrangler.json` ou no código-fonte.
- Acumular estado em memória não delimitado ou caches globais que vazam entre instâncias reaproveitadas (warm isolates).
- Executar processamento síncrono pesado que exceda o limite de CPU do runtime de borda.
- Não tratar falhas transitórias do D1 ou KV com estratégia adequada de retry ou fallback.

## Checagens mínimas

- Rodar `wrangler check` ou `npx wrangler check` no módulo do worker.
- Rodar testes unitários (Vitest / Jest com `@cloudflare/workers-types` ou `miniflare`).
- Rodar `wrangler types` para garantir tipos TypeScript atualizados para os bindings.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
