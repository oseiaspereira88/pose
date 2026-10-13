---
slug: pose-cli-mcp-operation-results
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-recoverable-closeout-plan
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:cli-mcp-operation-results
---

# Spec: Compartilhar resultados operacionais e parsing entre CLI e MCP

## 1. Intent

### Goal

Produzir resultados tipados que distingam execução, veredito de domínio, espera, coverage e efeitos confirmados, com parsing estrito e uma saída JSON válida por chamada solicitada.

### Business value

B12 / F07-02, F07-03, F07-09, PERF-07 e CONS-13. Close apply/resume pode aceitar --json e emitir texto; flags inválidos ampliam consultas; timeout do subprocesso é classificado como gate negativo. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de UX](../reports/2026-10-12-pose-front-07-operational-ux.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Definir resultado versionado compartilhado que separe sucesso de consulta/execução, verdict de domínio, readiness/coverage, espera, deadline/cancelamento, conflito e falha operacional; transporte bem-sucedido não deve promover verdict nem complete.
- R2: Quando JSON for solicitado numa rota suportada, success/wait/failure deve emitir exatamente um documento JSON válido com operation/step/result/pending/next action e efeitos confirmados pertinentes; humano e máquina devem derivar do mesmo resultado, sem texto de progresso contaminando stdout JSON.
- R3: Start/close apply/status/plan/resume e continuous status devem aceitar modos compatíveis ou recusar combinações/flags desconhecidos explicitamente, independentemente da ordem dos argumentos. Datas/números/limites inválidos devem falhar antes da operação, sem ampliar silenciosamente o conjunto consultado; preservar aliases documentados.
- R4: Waiting, inclusive exit 3 previsto para judgment, deve conservar causa, pending IDs e próximo comando suportado, sem failed genérico. No-op/already satisfied deve ser separado de verificação atual de gates, sem reinterpretação retrospectiva de contratos legacy.
- R5: Contexto/deadline/cancelamento deve ser propagado pelo domínio e filhos pertinentes; verificar expiração antes de classificar ExitError como verdict negativo. Timeout/interrupção deve retornar execução inconclusiva e identificar output parcial e limites de cleanup, sem passed=false apresentado como reprovação concluída.
- R6: Erro após escrita/publicação deve revelar committed effects e fase comprovados, erro e possibilidade de resume/reconcile; output ausente não prova rollback e resultado parcial não autoriza retry cego. Não alegar atomicidade geral por padronizar o envelope.
- R7: CLI e MCP devem preservar os mesmos outcomes e causas tipadas no domínio, mantendo compatibilidade deliberada de exit codes, JSON-RPC, campos e aliases. Invalid args não devem gerar side effects ou payload falsamente complete; flag JSON não suportado deve ser recusado explicitamente.
- R8: Observabilidade existente deve classificar waiting/rework gerado/reusado por fatos do resultado e causa conhecida, sem métricas sintéticas: ausência de evento/intervalo não deve virar tempo zero. Não construir nova plataforma de telemetria ou inferir esforço humano de bytes.
- R9: Qualificar probes P03/P06/P07/P08, timeout controlado PERF-07 e erro com efeito parcial CONS-13, além de controles positivos, usando mesmo resultado para renderers e consumers; não alterar policy/adoção nem automatizar julgamento por sucesso de execução.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-4-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-cli-mcp-operation-results.md
- created: .pose/starts/pose-cli-mcp-operation-results.json
- created: .pose/changelogs/unreleased/pose-cli-mcp-operation-results.md
- modified: pose-mcp/internal/cli/closeout_plan.go
- modified: pose-mcp/internal/cli/start.go
- created: pose-mcp/internal/pose/operation_result.go
- created: pose-mcp/internal/pose/operation_result_test.go
- created: pose-mcp/internal/cli/operation_results_test.go
- created: pose-mcp/internal/mcpserver/operation_results_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:cli-mcp-operation-results module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/closeout_plan.go

### Approach and execution gates

Mapear rotas/parsers e adapter subprocess no source atual e reconciliar seus paths antes de start. Definir schema, classificação e compatibilidade em ADR prospectivo antes do código estrutural. Reutilizar diagnostics e closeout plan; propagar context em vez de traduzir timeout em verdict. Manter result rendering e exit mapping nos adapters, com efeitos observados pelo domínio. B12 é raiz; B13 consumirá o resultado para recovery, sem implementar publisher CAS geral (B14) ou daemon (B35).

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R4, R5, R6, R7
- Decision: Separar outcome operacional, verdict, coverage e efeitos no resultado comum.
- Rationale: Espera, timeout e falha após publicação não são o mesmo fato nem permitem o mesmo retry.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P03 success/wait/failure JSON e output humano | Um JSON válido; mesmo resultado/step/pending/next action, stdout sem texto extra. |
| V2 — P06 unknown flags/datas/números e P07 modos em ordens diferentes | Recusa sem writes ou ampliação; aliases válidos preservados. |
| V3 — P08 continuous JSON suportado/não suportado | Documento válido ou erro explícito, nunca flag ignorado. |
| V4 — Waiting exit 3, no-op e coverage incomplete | Espera distinta de failure/verificação; consulta válida não é ready/complete. |
| V5 — Processo lento, cancelamento e output parcial | Contexto propagado e inconclusivo tipado, sem reprovação de gate fabricada. |
| V6 — Falha após state/history ou outra publicação parcial | Efeitos confirmados/fase/erro expostos, sem rollback presumido. |
| V7 — CLI/MCP e eventos waiting/reuse/rework | Outcomes equivalentes; métricas só com fatos e intervalos conhecidos. |
| V8 — Controles positivos e contratos legacy | Sucesso/aliases preservados; sem policy nova retrospectiva. |

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
| R1 | V1, V4–V6 |
| R2 | V1 |
| R3 | V2, V3 |
| R4 | V4, V8 |
| R5 | V5 |
| R6 | V6 |
| R7 | V1–V3, V7 |
| R8 | V7 |
| R9 | V1–V8 |

Consultar também [consistência/recovery](../reports/2026-10-12-pose-front-09-consistency-recovery.md) e [performance](../reports/2026-10-12-pose-front-08-resident-performance.md).

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 4. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Schema, exit mapping e modes são contratos públicos. Enumerar rotas afetadas para evitar envelope de sucesso que esconda falha de transporte/execução. Context cancellation não prova ausência de efeito; o domínio precisa informar o que confirmou. Sem instrumentação suficiente, duração/efeito permanece unknown.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
