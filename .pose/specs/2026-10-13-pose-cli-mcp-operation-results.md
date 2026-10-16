---
slug: pose-cli-mcp-operation-results
status: in-progress
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

Paths reconciliados na main 1a278aff antes do start. A escrita original
pertence ao lote documental 4; este delta implementa somente B12. As rotas
afetadas são `close --apply|--resume`, `continuous-closeout status`, os
parsers de `close`, `start` e `specs`, o adapter de subprocesso dos gates MCP
(`runCLIExit`/`runCLI`), `pose_list_specs` e o journal de uso. `--plan --json`
e os documentos de `start` mantêm o formato. O consumidor MCP é exercitado por
`callToolCtx` com prazo controlado e pelo transporte HTTP JSON-RPC.

- modified: .pose/specs/2026-10-13-pose-cli-mcp-operation-results.md
- created: .pose/starts/pose-cli-mcp-operation-results.json
- created: .pose/changelogs/unreleased/pose-cli-mcp-operation-results.md
- created: .pose/adr/2026-10-16-operation-results-separate-execution-from-verdict.md
- created: pose-mcp/internal/pose/operation_result.go
- created: pose-mcp/internal/pose/operation_result_test.go
- modified: pose-mcp/internal/pose/cli.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/cli/closeout_plan.go
- modified: pose-mcp/internal/cli/start.go
- modified: pose-mcp/internal/cli/specs_cmd.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/usage.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json
- created: pose-mcp/internal/cli/operation_results_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- created: pose-mcp/internal/mcpserver/operation_results_test.go
- modified: pose-mcp/internal/usage/usage.go
- modified: pose-mcp/internal/usage/usage_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:cli-mcp-operation-results module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/operation_result.go

### Approach and execution gates

Contrato no ADR `2026-10-16-operation-results-separate-execution-from-verdict`.
`internal/pose.OperationResult` (schema 1) carrega operation, scope, outcome,
step, steps_run, pending, next_action, effects confirmados, unconfirmed,
resumable, error tipado e data. Outcomes disjuntos: completed,
already-satisfied, waiting, blocked, conflict, failed, inconclusive. Rendering
e exit mapping ficam nos adapters; os exit codes documentados (0, 1, 2, 3) não
mudam. `close --apply|--resume` monta o resultado num laço sem escrita em
stdout e renderiza humano ou JSON a partir dele; um efeito só é confirmado
depois que o replanejamento o reflete, e um passo que falhou depois de poder
ter escrito vai para `unconfirmed`. `runCLIExit`/`runCLI` consultam o contexto
antes do `*exec.ExitError` e devolvem `*GateExecutionError` com causa, saída
parcial e limite de cleanup, com `WaitDelay` para não segurar a chamada; o MCP
o mapeia para `error_code: execution_inconclusive`. Parsers recusam com exit 2
antes de ler. O journal ganha `waiting`, `duration_unknown` e
`duration_samples`. Compatibilidade: recusas mais estritas e JSON novo onde
havia texto; classificado como minor, não patch (D2).

B12 é raiz; B13 consumirá o resultado para recovery, sem implementar publisher
CAS geral (B14) ou daemon (B35). Propagação de contexto dentro de chamadas de
domínio em processo do CLI fica fora: o processo do CLI termina pelo sinal.

## 4. Tasks

- [x] Context, readiness, assessment freshness e reconciliação de rotas e paths contra a main 1a278aff.
- [x] ADR prospectivo de schema, classificação e compatibilidade antes do código estrutural.
- [x] Native start com R1–R9/D1 preservados.
- [x] OperationResult no domínio; close apply/resume e continuous status sobre ele; parsers estritos de close, start, specs e pose_list_specs.
- [x] Gate subprocess classificado pelo contexto antes do exit status; MCP com erro inconclusivo tipado.
- [x] Journal de uso com waiting e duração desconhecida.
- [ ] V1–V8 nos consumidores CLI, domínio e MCP; probes P03/P06/P07/P08 antes e depois; validação estrita do módulo.
- [ ] Revisão por ator diferente, closeout nativo e PR.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R4, R5, R6, R7
- Decision: Separar outcome operacional, verdict, coverage e efeitos no resultado comum.
- Rationale: Espera, timeout e falha após publicação não são o mesmo fato nem permitem o mesmo retry.

### Decision D2
- Basis: R2, R3, R7
- Decision: Manter os documentos JSON que já têm contrato (`--plan --json`, plano e registro de `start`, `GateResult`) e introduzir o resultado comum só onde a chamada não tinha JSON ou imprimia texto no lugar do JSON que aceitava.
- Rationale: Trocar todos os JSON por um envelope quebraria consumidores sem acrescentar fato; o timeout vira erro tipado, não um campo novo em `GateResult`, para que nenhum consumidor leia `passed=false` como veredito.
- Consequences: Recusas mais estritas mudam a semântica pública de invocações antes toleradas; classificado como minor.

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
