# Rule: Backend Go

## Quando consultar

Consulte este guia em tarefas de handlers HTTP, serviços de domínio, persistência, concorrência e integração entre serviços Go.

## Padrões obrigatórios

- Erros devem ser tratados e propagados com contexto suficiente para diagnóstico.
- Handlers devem validar entrada e retornar códigos HTTP coerentes com o contrato.
- Regras de negócio devem ficar em camadas de serviço/domínio, não no transporte.
- Operações com contexto devem respeitar `context.Context` para timeout/cancelamento.
- Acesso a dados deve usar interfaces claras e facilitar testes.
- Logs devem ser estruturados e sem exposição de dados sensíveis.

## Anti-padrões bloqueadores

- Ignorar retornos de erro (`_` em erros críticos).
- `panic` em fluxo de requisição comum em vez de tratamento controlado.
- Acoplamento direto entre handler e implementação concreta de repositório sem abstração.
- Consultas sem limites/paginação em endpoints potencialmente volumosos.
- Concorrência sem sincronização adequada, sujeita a race conditions.

## Checks mínimos

- `go test ./...` no escopo afetado.
- `go test -race ./...` quando houver mudança de concorrência.
- `go vet ./...` sem achados bloqueadores.
- `check` de `lint` Go (ex.: `golangci-lint`) sem erros críticos.

## Precedência em conflito multi-domínio

- Em conflito com outras `rules`, aplique a alternativa mais restritiva para segurança, contrato e operação.
- Quando houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
