---
slug: pose-abm-review-tool-deferred-without-delivery
status: done
created_at: 2026-09-19
completed_at: 2026-09-19
supersedes:
depends_on: pose-abm-review-soundness
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:review-tool-deferred
---

# Spec: Review tools delivery-gated podem ser deferidas honestamente

## 1. Intent

### Goal
Permitir que a CLI registre a disposição `deferred` para uma ferramenta
delivery-gated quando o scope não declara delivery target, deixando a validação
final comparar a disposição com o contexto real do bundle.

### Business value
Evitar que reviews de specs sem delivery target sejam forçados a fabricar
evidência de `validate` ou a marcar uma ferramenta obrigatória como não usada.

### Constraints
- O evaluator continua sendo a autoridade sobre a existência de delivery
  target; a CLI apenas não bloqueia a disposição antes dessa avaliação.
- Scopes com target continuam recusando `deferred` no gate final.
- Nenhuma evidência sintética é criada.

### Non-goals
Não alterar a matriz de validação, profiles de delivery ou critérios de
review; não relaxar `review-complete`.

## 2. Requirements

- R1: A disposição CLI `deferred` é aceita para ferramenta com precondition
  `delivery-target-declared` quando contém rationale.
- R2: A avaliação final continua recusando a disposição quando o scope possui
  delivery target e não há conclusão válida.
- R3: A mudança não gera evidência nem altera o contrato de `pose validate`.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/cli/review_closeout.go` e teste de contrato CLI.

### Delivery targets
- governance:review-tool-deferred module:pose-mcp profile:structural-delta entrypoint:pose-mcp/internal/cli/review_closeout.go

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-review-tool-deferred-without-delivery.md
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout_test.go

### Rollout and reversal
Mudança aditiva no parser de disposição; revert simples, sem migração.

## 4. Tasks

- [x] Reproduzir a impossibilidade de atestar scope sem delivery target.
- [x] Permitir defer explícito delivery-gated com rationale.
- [x] Cobrir o parser com teste determinístico.
- [x] Rodar suíte e review final.

## 5. Decisions

### Decision D1 — avaliar contexto no evaluator, não no parser

- Context: o parser não recebe o grafo completo de delivery; bloquear ali
  confundia ausência de target com falha de validação.
- Options considered: fabricar `unit:auto`; aceitar `not-used`; permitir
  `deferred` apenas para precondition delivery-gated e deixar o evaluator
  decidir.
- Decision: terceira opção.
- Rationale: preserva evidência e mantém a regra de target-bearing scope no
  caminho que conhece o bundle.
- Consequences: uma disposição inválida ainda falha em `review verify`.

## 6. Validation

| Scenario / requirements | Command | Expected evidence |
| --- | --- | --- |
| Parser accepts delivery-gated deferred; R1 | `go test ./internal/cli -run TestRequiredDeliveryToolMayBeDeferredWithoutDeliveryTarget -count=1` | pass with rationale |
| Full CLI/review behavior; R2/R3 | `go test ./internal/cli ./internal/pose -run 'Review|Closeout' -count=1` | target-bearing gates remain strict |
| Full module | `go test ./...` | no regression |

### Execution log
2026-09-19: discovered while sealing the date-independent authority fixture
bugfix; the evaluator already understood no-target deferral, but the CLI
rejected the disposition before it could reach that evaluator.
2026-09-19: focused parser regression, full Go suite, vet, build and review
convergence validation passed; target-bearing tool coverage remained strict.
2026-09-19: sealed review bundle `rvb-fa979be05b5ed2ad` and approved
attestation `rva-64d05201a93ae700` verified fresh.

### Requirement trace
- R1 [satisfied] <TestRequiredDeliveryToolMayBeDeferredWithoutDeliveryTarget accepts a rationale-bearing delivery-gated disposition> evidence:unit:pose-mcp/go/test
- R2 [satisfied] <review evaluator keeps target-bearing required tools strict> evidence:integration:pose-mcp/go/review-bundle-convergence
- R3 [satisfied] <the implementation changes only CLI disposition parsing and preserves validation behavior> evidence:build:pose-mcp/go/build evidence:unit:pose-mcp/go/test

## 7. Final Report

### Delivered scope
Parser-only governance fix delivered. Delivery-gated tools may be recorded as
deferred before scope evaluation, while final review verification remains the
authority and rejects deferral when a delivery target exists.

### Residual risks
The CLI intentionally does not inspect the delivery graph; callers must still
run the final review verification gate. This boundary is covered by the
existing target-bearing review tests and the sealed validation evidence.

### Follow-ups
None.
