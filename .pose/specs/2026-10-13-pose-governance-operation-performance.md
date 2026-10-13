---
slug: pose-governance-operation-performance
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-check-spawns-fewer-git-processes, pose-attention-within-a-second
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:governance-operation-performance
---

# Spec: Medir e otimizar algoritmos de governança por operação com fidelidade

## 1. Intent

### Goal

Reduzir trabalho redundante em catálogo/bundles/Git/clustering por operação, com benchmark seguro e resultados semanticamente equivalentes.

### Business value

B32 / D12, PERF-01/02/03/05. Benchmark ignora exit codes e restaura arquivos locais; catálogo e Git são relidos por caminhos diversos e clustering escala em pares. Reutilizar as duas otimizações entregues, sem atribuir todo custo a filesystem. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise fonte](../reports/2026-10-12-pose-front-08-resident-performance.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Benchmark deve executar em fixture/checkout descartável com inventário/digests, corpus/revision/binary/engine/matriz/ambiente fixados; preservar verdict/exit/coverage e distinguir erro/timeout de amostra válida. Nunca usar checkout/reset para restaurar trabalho do operador ou || true para ocultar falha.
- R2: Perfil deve medir por produtor/etapa tempo, CPU, RSS/heap, leituras/parses/cópias, subprocessos Git/bytes e custo de saída, com metodologia, repetições e dispersão explícitas; não inferir percentual de CPU/I/O ou esforço humano a partir de bytes/latência sem evidência.
- R3: Catálogo/mapa de slug/scopes e bundles pertinentes devem ser produzidos/reusados no limite de uma operação, com identidade/binding e coverage honesta. Leituras devem ficar limitadas à visão necessária, sem limitar arbitrariamente últimas 100 specs ou excluir histórico requerido para passar gate.
- R4: Otimização deve preservar verdict, ordenação, IDs, conflitos e associações em layouts flat/nested/duplicados/foreign e inputs alterados. Reuso não deve converter ausência/erro em completeness ou autorização antiga em atual; alteração pertinente exige releitura/conflito explícito.
- R5: Normalização/clustering deve preservar candidatos exatos e semântica de similaridade/ordenamento, ou propor aproximação explicitamente contratada/qualificada antes da mudança; medir escala por entradas e tamanho de texto, sem fechamento/disposition por similaridade.
- R6: Atribuição Git em lote/reuso deve preservar claims/observed e histórico completo, incluindo refs alteradas, nomes/path bytes e revisões pertinentes; limitar bytes/tempo/framing e tratar falha parcial de subprocesso sem sucesso fabricado.
- R7: Memoização/reuso devem ser limitados à operação com orçamento e chaves pertinentes; não criar cache global/daemon, aliasing mutável entre consumers ou RAM ilimitada. Vistas/caches de B33 e runtime B35 conservam seus próprios contratos e lotes.
- R8: Qualificar corpus representativo e adversarial, controles positivos e paridade CLI/MCP, com resultados antes/depois e razões de rework/reuso somente quando observadas. Não alterar gate/policy ou descartar dados para melhorar número; queda de latência isolada não prova eficácia geral.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-7-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-governance-operation-performance.md
- created: .pose/starts/pose-governance-operation-performance.json
- created: .pose/changelogs/unreleased/pose-governance-operation-performance.md
- modified: scripts/bench-governance.sh
- modified: pose-mcp/internal/pose/followup_candidates.go
- modified: pose-mcp/internal/pose/git_batch.go
- created: pose-mcp/internal/pose/governance_operation_performance_test.go
- created: pose-mcp/internal/cli/governance_operation_performance_test.go
- created: pose-mcp/internal/mcpserver/governance_operation_performance_test.go
- created: .pose/results/pose-governance-operation-performance.json
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:governance-operation-performance module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/git_batch.go

### Approach and execution gates

Perfilar antes de escolher otimização e inventariar catálogo/checks/Bundle/Git consumers; reconciliar os paths exatos antes de start. Reutilizar primitives entregues e priorizar normalização única, mapas por operação e batch Git com limites. Registrar mudança de algoritmo/contrato aproximado em ADR quando necessária. Benchmark não deve rodar no checkout do usuário nem medir nova residência antes de B35 existir.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7
- Decision: Medir por produtor e otimizar reuso limitado por operação mantendo equivalência e benchmark isolado.
- Rationale: Melhor tempo sem verdict/coverage preservados não constitui melhoria de governança.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Benchmark com falha/timeout e arquivos locais modificados | Verdict/ambiente retidos; fixture isolada; nenhum arquivo local restaurado/descartado. |
| V2 — Perfil por produtor em corpus fixo e repetido | Tempo/CPU/RSS/parses/cópias/Git/bytes separados; limites/dispersão explicitados. |
| V3 — Flat/nested, slug duplicado, foreign e inputs alterados | Mesmo verdict/IDs/ordem/conflitos com coverage honesta. |
| V4 — Clustering em F e comprimento de texto crescentes | Mesmos candidatos exatos ou aproximação contratada; sem disposition automática. |
| V5 — Git histórico completo, refs mudadas, framing e falha parcial | Claims/observed equivalentes, limites e erro preservados. |
| V6 — CLI/MCP e operação repetida com orçamento | Paridade sem cache global/daemon ou alteração de gates para melhorar score. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Trace planejado; vincular os casos aos produtores reais na execução.
Não declarar requisitos satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1 |
| R2 | V2 |
| R3 | V3 |
| R4 | V3,V6 |
| R5 | V4 |
| R6 | V5 |
| R7 | V6 |
| R8 | V1–V6 |

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 7. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Fixtures mínimas não representam milhares de bundles reais; reportar limites sem extrapolar percentuais. Catálogo por operação não é snapshot universal para autorizar writes; a coerência dos inputs e geração continua nos contratos próprios. Não executar o benchmark destrutivo atual como validação deste lote documental.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
