---
slug: pose-closeout-terminal-verification
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-action-target-and-node-scoping, pose-amendment-history-holds-before-close, pose-recoverable-closeout-plan
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:closeout-terminal-verification
---

# Spec: Separar lifecycle done de fechamento semanticamente verificado

## 1. Intent

### Goal

Fazer plano, execução e leitura de closeout compartilharem o verificador aplicável, distinguindo done/no-op de verificação terminal e preservando a continuidade da spec existente de amendments.

### Business value

B11 / D09 e F07-01. O ramo done de PlanCloseout marca todos os steps done e Terminal=true pelo status bruto. D09 permanece responsabilidade da spec aberta pose-amendment-history-holds-before-close; este delta consome seu gate, sem duplicar baseline/append de amendments. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de UX](../reports/2026-10-12-pose-front-07-operational-ux.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Status done e operação idempotente/no-op devem ser distintos de verificação semântica terminal. O plano não deve marcar review/judgment/evidence/verify executados ou satisfeitos apenas porque o lifecycle é done.
- R2: Plan, closeout-check, apply/resume e consumidores CLI/MCP devem usar o mesmo verificador do contrato aplicável ao scope, com blockers, limitações e próximos passos suportados; P02 com review devido ausente não deve aparecer terminal em uma superfície e recusado na outra.
- R3: O gate de amendment history deve ser consumido antes da transição final, usando a implementação e requisitos R1–R3 da spec reutilizada. Não criar segundo seeding de baseline ou segunda taxonomia/log para resolver D09.
- R4: Done com prova suficiente deve manter idempotência e retorno no-op fiel; done com verificação insuficiente deve revelar lacuna/reparo sem reabrir, sobrescrever ou fabricar review automaticamente. Contexto desconhecido/ilegível não deve certificar terminal.
- R5: Contracts selados, cutoffs de adoção e semântica legacy devem ser preservados: policy atual nova não deve impor enforcement retrospectivo indevido. Explicitar o fundamento de verificação e a limitação histórica quando não houver prova, sem inventar receipt antigo.
- R6: Baselines/start records/amendments e requisitos históricos devem permanecer íntegros e append-only. Reconciliar plano de artefatos e dependências da spec draft reutilizada antes de start, conservando os seus R1–R3; qualificar primeiro a cadeia B08 e o gate de amendments.
- R7: Mensagem, step state e payload devem corresponder a operações/verificações realmente feitas ou reutilizadas com prova; sucesso de consulta ou status bruto não deve virar delivery claim. Usar o contrato de resultados de B12 quando disponível, sem torná-lo dependência artificial do DAG deste delta.
- R8: Qualificar P02, done/no-op válido, prova ausente/ilegível, legacy/cutoffs e gate de amendments antes do write, com paridade CLI/MCP e controles positivos. Não alterar policy, adoção, release ou disposition por inferência.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-5-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-closeout-terminal-verification.md
- created: .pose/starts/pose-closeout-terminal-verification.json
- created: .pose/changelogs/unreleased/pose-closeout-terminal-verification.md
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/closeout_plan.go
- created: pose-mcp/internal/pose/closeout_terminal_verification_test.go
- created: pose-mcp/internal/cli/closeout_terminal_verification_test.go
- created: pose-mcp/internal/mcpserver/closeout_terminal_verification_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:closeout-terminal-verification module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/closeout_plan.go

### Approach and execution gates

Reutilizar verificação aplicável de closeout e o gate implementado na spec de amendments existente. Remover o atalho semântico do status done conservando no-op operacional explícito. Definir compatibilidade do plano e payload antes do código; registrar decisões estruturais adicionais necessárias. A spec de amendments continua dona de seeding/gate/warning R1–R3; declarar e iniciar seus artefatos exatos na execução futura e qualificar seu closeout antes deste delta.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R5, R6
- Decision: Compartilhar verificação semântica aplicável e consumir o gate da spec existente de amendments.
- Rationale: Status bruto e duplicação de baseline não comprovam closeout nem preservam a história.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P02 done sem review devido após adoção | Plano não certifica terminal/steps verificados; mesmo blocker do closeout-check. |
| V2 — Done válido/no-op e consulta bem-sucedida | Idempotência preservada, verificação/fonte distintas de operação repetida. |
| V3 — História inválida de amendments antes do close | Gate compartilhado recusa antes da transição; sem segundo seeding/log. |
| V4 — Primeira emenda com start baseline e log histórico | R1–R3 da spec existente qualificados; baseline append-only preservada. |
| V5 — Legacy, cutoff e mudança de policy atual | Contrato histórico aplicável preservado; sem enforcement retrospectivo indevido. |
| V6 — Prova/contexto ilegíveis e paridade CLI/MCP | Limitação/blocker fiel; não fabricar terminal, receipt ou review. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Trace planejado; vincular casos aos produtores reais na implementação.
Não declarar requisitos funcionais satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V2 |
| R2 | V1, V6 |
| R3 | V3, V4 |
| R4 | V2, V6 |
| R5 | V5 |
| R6 | V3, V4 |
| R7 | V1, V2, V6 |
| R8 | V1–V6 |

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 5. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

No-op de lifecycle não garante conclusão atual. A implementação precisa distinguir contrato histórico aplicável de exigência nova e não reescrever status/review para sanar lacuna. A spec de amendments está draft e precisa de plano/start próprios; este lote documenta sua continuidade sem mudar baseline ou declarar execução.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
