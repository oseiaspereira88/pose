# Rule: Backend Go

## When to consult

Consult this guide em tarefas de handlers HTTP, serviços de domínio, persistência, concorrência e integração entre serviços Go.

## Required patterns

- Erros must ser tratados e propagados with contexto suficiente para diagnóstico.
- Handlers must validar entrada e retornar códigos HTTP coerentes with o contrato.
- Regras de negócio must ficar em camadas de serviço/domínio, não no transporte.
- Operações with contexto must respeitar `context.Context` para timeout/cancelamento.
- Acesso a dados must usar interfaces claras e facilitar testes.
- Logs must ser estruturados e without exposição de dados sensíveis.

## Blocking anti-patterns

- Ignorar retornos de error (`_` em errors críticos).
- `panic` em fluxo de requisição comum em vez de tratamento controlado.
- Acoplamento direto entre handler e implementação concreta de repositório without abstração.
- Consultas without limites/paginação em endpoints potencialmente volumosos.
- Concorrência without sincronização adequada, sujeita a race conditions.

## Minimum checks

- `go test ./...` no escopo afetado.
- `go test -race ./...` when houver mudança de concorrência.
- `go vet ./...` without achados bloqueadores.
- `check` de `lint` Go (ex.: `golangci-lint`) without errors críticos.

## Precedência em conflito multi-domínio

- Em conflito with outras `rules`, apply a alternativa mais restritiva para security, contrato e operação.
- When houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
