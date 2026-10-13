---
slug: pose-action-request-recovery-and-preflight
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-action-waiver-and-supersession-authority, pose-action-requests, pose-action-request-presentation
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:action-request-recovery-and-preflight
---

# Spec: Recuperar requests e verificar admissibilidade antes de apply

## 1. Intent

### Goal

Oferecer próximos passos suportados para condições pendentes e convergir supersession parcial no retry da mesma intenção, com preview útil e read-only.

### Business value

B10 / T4-03, T4-05 e CONS-01. Presentation pede answer que writer recusa quando invalidated; subject ausente some da lista. Retry de open retorna nova request já existente antes de completar superseded na antiga. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de ações](../reports/2026-10-12-pose-front-04-actions-obligations.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Invalidated deve encaminhar nova request ligada à anterior ou operação realmente suportada pelo writer; missing subject deve revelar reparo e dados faltantes. Não assinar/responder digest antigo nem mostrar comando de resolução impossível como próximo passo executável.
- R2: Decline deve continuar não satisfeito quando sua consequência assim determina e oferecer remediação/substituição material ou waiver autorizado; não repetir pergunta idêntica para conseguir approve. Retorno de sessão não deve repetir satisfação válida nem ocultar condição pendente.
- R3: Presentation, Attention, readiness/closeout e CLI/MCP devem distinguir pergunta aguardando resposta de condição aguardando tratamento, usando next step tipado e capacidades/autoridade reais, sem novo comando top-level, state artificial ou auto-satisfação.
- R4: Preview deve compartilhar validação de admissibilidade do domínio para ator/role, opção, digest, revisão, projeto, estado e authority aplicáveis, com causas tipadas e zero writes/assinatura/toque; checks dependentes de prova ainda não produzida devem aparecer como pendentes, sem exit 0 certificar aceite futuro.
- R5: Apply deve revalidar precondições, autoridade/prova exigidas por B09 e revisão/digest no ponto protegido de escrita; preview admissível não autoriza apply stale. Retry com mesma idempotency key e payload deve manter identidade e não duplicar decisão; chave conflitante não deve ser aceita como retry equivalente.
- R6: Supersession entre journals deve expor fase parcial após lock/falha/crash e retomar o efeito pendente no retry da mesma intenção/ID; nova request já existente não deve fazer retornar antes do superseded faltante. Predecessor/substituta devem ficar visíveis até reconciliação, sem liberar condição por progresso parcial.
- R7: Retomada deve verificar lineage, identidade de intenção, revisões/digests e autoridade atuais pertinentes, com lock/order/limites de retry explícitos. Concorrência, erro de append/Sync e crash não devem produzir duas substitutas efetivas ou ocultar conflito; estado impossível/corrupto deve resultar em reparo/unknown, nunca completion fabricada.
- R8: Definir se refusal é resultado de comando ou auditoria persistida separadamente, reconciliando o compromisso da fundação. Não alegar evento refused que não foi gravado; eventual auditoria deve preservar revisão/idempotência da pergunta, privacidade e limites de volume, sem ruído automático no journal.
- R9: Controle positivo de resposta, supersession e sessão deve continuar válido; não alterar policy/adoção por inferência, apagar histórico ou criar retry infinito. Qualificar mudanças de contrato/compatibilidade e apontar evidência real das etapas recuperadas.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-3-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-action-request-recovery-and-preflight.md
- created: .pose/starts/pose-action-request-recovery-and-preflight.json
- created: .pose/changelogs/unreleased/pose-action-request-recovery-and-preflight.md
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/action_presentation.go
- created: pose-mcp/internal/pose/action_recovery_preflight_test.go
- created: pose-mcp/internal/cli/action_recovery_preflight_test.go
- created: pose-mcp/internal/mcpserver/action_recovery_preflight_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:action-request-recovery-and-preflight module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/action_request.go

### Approach and execution gates

Extrair preflight read-only reutilizado pelo writer e representar prova pendente sem acioná-la. Convergir open/supersede idempotente retomando append pendente, com estado transacional identificado/derivado ou checkpoint mínimo conforme decisão anterior ao código. Não generalizar para publisher CAS (B14) ou recovery de todos os journals (B13). Compartilhar next steps entre superfícies e distinguir observação de refusal de fato auditado.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5
- Decision: Compartilhar admissibilidade e próximos passos com writer, revelando checks de prova pendentes.
- Rationale: Um preview útil deve explicar o que ainda falta sem assinar ou prometer efeito futuro.

### Decision D2
- Basis: R6, R7, R8
- Decision: Retomar supersession parcial pela mesma intenção e divulgar somente auditoria realmente persistida.
- Rationale: Existência da substituta não comprova retirada da anterior; recusa de comando não é automaticamente evento auditado.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Invalidated, missing subject e sessão retomada | Próximo passo executável ou reparo explícito; digest antigo não é respondido. |
| V2 — Decline e satisfação válida | Remediação sem nova pergunta idêntica; satisfação válida não repetida. |
| V3 — Preview ator/answer/digest/revision/projeto/estado inválidos | Causas tipadas, nenhum write/toque; prova não produzida divulgada como pendente. |
| V4 — Preview válido seguido de stale/authority change e retry de resolução | Apply revalida; mesma intenção não duplica; payload conflitante recusado. |
| V5 — Lock no predecessor após escrita da substituta e retry mesmo ID | Fase parcial visível; retry completa superseded exatamente uma vez. |
| V6 — Crash/append/Sync failure e concorrência entre substitutas | Recovery limitado, sem duas substituições efetivas ou falso complete; conflito explícito. |
| V7 — Refusal sem registro e auditoria deliberadamente persistida | Disclosure corresponde aos fatos, sem mudar revisão/idempotência legítimas. |
| V8 — CLI/MCP/Attention e controles válidos | Paridade de next step/efeito, sem auto-adoption ou perda de histórico. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Trace planejado; vincular os casos aos produtores reais na implementação.
Não declarar requisitos funcionais satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1 |
| R2 | V2 |
| R3 | V1, V2, V8 |
| R4 | V3 |
| R5 | V4 |
| R6 | V5 |
| R7 | V5, V6 |
| R8 | V7 |
| R9 | V8 |

Consultar também a [análise de recovery](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 3. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Dois journals não são uma transação por estarem sequenciais. Determinar fase recuperável e tratamento de competing intent antes do código. Preview sem assinatura não pode certificar prova ainda ausente. Closeout de B09 precede execução; os deltas gerais de recovery conservam seus próprios lotes.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
