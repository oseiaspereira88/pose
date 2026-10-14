---
slug: pose-generation-bound-query-cursors
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-governance-read-view-generation, pose-obligation-projection
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:generation-bound-query-cursors
---

# Spec: Vincular paginação e consulta à geração observada

## 1. Intent

### Goal

Evitar duplicação ou omissão silenciosa entre páginas e reduzir trabalho quando a consulta for indexável sem truncar gates completos.

### Business value

B36 / PERF-11. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de consistência](../reports/2026-10-12-pose-front-09-consistency-recovery.md) e a [análise de performance](../reports/2026-10-12-pose-front-08-resident-performance.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Definir cursor opaco/versionado ligado à read view/geração de B34, ferramenta/query, escopo de projeto/worktree/root autorizado, filtros normalizados, ordem total determinística com desempate e posição de continuação. Offset ou versão de transporte isolados não devem identificar a sequência de resultados.
- R2: Paginar a mesma geração retida e válida ou recusar cursor stale com diagnóstico/restart explícito quando a fonte muda; declarar estratégia por superfície antes da implementação. Não continuar sobre coleção corrente alterada nem reiniciar silenciosamente na página zero; preservar a geração histórica identificada quando essa navegação for admitida.
- R3: Validar token malformado/truncado, versão unsupported, bindings de query/filtros/ordem/projeto/tool divergentes, expiração e posição inválida. Recusar adulteração ou replay fora do escopo conforme modelo de autoridade; cursor nunca amplia autorização e cada página revalida acesso/contexto. Definir proteção dos tokens sem tratar base64 como garantia de integridade.
- R4: Snapshot de navegação pode ser histórico e coerente, mas não deve alimentar gate ou efeito como current sem revalidação de B34. Divulgar geração/freshness/coverage/limitações por página; end-of-page/next_cursor ausente não deve certificar cobertura de todo histórico ou ausência de pendências.
- R5: Limit deve limitar tamanho da página e, com índice/ordem seletivos disponíveis, leitura/parse/cópias de dados; aplicar filtros/ordem no domínio antes do recorte. Total exato é opcional ou explicitamente custa scan completo; não materializar tudo apenas para calcular total ou emitir notice. Quando o scan integral for necessário, divulgar custo/limite sem alegar otimização inexistente.
- R6: Definir bounds de page size/token/bytes, retention/TTL, número de gerações/consultas retidas, custo e eviction de snapshots; expiração/eviction deve produzir cursor stale/restart, nunca outra sequência. Ausência de estado residente deve admitir cursor verificável ou restart/fallback explícito; não exigir daemon para consultas CLI one-shot.
- R7: Preservar compatibilidade das chamadas legadas sem cursor/limit e declarar migração/versionamento de tokens antigos sem reinterpretar ordinal como geração. CLI/MCP consumidores devem observar a mesma consulta/read view quando aplicável; documentar diferenças legítimas de superfície e preservar content/structuredContent sem divergência.
- R8: Qualificar mutação entre páginas por inserção/remoção/rename/reordenação e empate de ordenação, filtros alterados, token hostil, geração expirada e contexto não autorizado. Demonstrar snapshot consistente ou stale explícito e ausência de duplicação/omissão silenciosa; medir leitura/parse/bytes/latência para limit em corpus indexável e não indexável, sem confundir cursor de eventos com cursor de catálogo.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-9-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-generation-bound-query-cursors.md
- created: .pose/starts/pose-generation-bound-query-cursors.json
- created: .pose/changelogs/unreleased/pose-generation-bound-query-cursors.md
- modified: pose-mcp/internal/mcpserver/pagination.go
- modified: pose-mcp/internal/mcpserver/pagination_test.go
- modified: pose-mcp/internal/pose/spec.go
- created: pose-mcp/internal/pose/generation_query.go
- created: pose-mcp/internal/pose/generation_query_test.go
- created: pose-mcp/internal/mcpserver/generation_cursor_test.go
- created: pose-mcp/internal/cli/generation_query_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:generation-bound-query-cursors module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/mcpserver/pagination.go

### Approach and execution gates

Executar depois de B34. Inventariar ferramentas paginadas e queries de domínio antes de start; registrar todos os handlers/consumers alterados no plano exato. Reconciliar schema de paginação existente e decidir snapshot-retained versus restart, binding/proteção do token, retention e total/custo por query. Usar read view de domínio, sem fatiar uma lista recomposta a cada página. Gates continuam completos no recorte solicitado; paginação de exibição não impõe limite de 100 specs ao aceite. A decisão de retenção não implementa B35/IPC/scheduler, e cursores de eventos HTTP/Redis conservam contrato próprio.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Continuar somente na geração e query vinculadas ao cursor, com obsolescência e custo explícitos.
- Rationale: Existência de dados ou sucesso de transporte não prova coerência, frescor ou autoridade; preservar bindings e incerteza impede autorizar com leitura obsoleta.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Inserir/remover/rename/reordenar entre páginas | Mesma geração identificada ou stale/restart explícito; nenhum salto ou duplicata silenciosa. |
| V2 — Empates, filtros/ordem/tool/projeto/root divergentes | Ordem total e binding verificados; cursor não muda de consulta ou autoridade. |
| V3 — Token malformed/unsupported/adulterado, posição inválida e replay | Recusa útil; nunca coerção a zero nem ampliação de acesso. |
| V4 — Retention/TTL/eviction e fonte indisponível | Stale explícito, recursos limitados; coverage/unknown preservados. |
| V5 — Limit e total em corpus indexável versus scan necessário | Trabalho reduzido quando demonstrável; custo de total/scan divulgado, sem truncar gate. |
| V6 — Legado sem paginação/token antigo e CLI/MCP | Compatibilidade deliberada; generation/freshness/content alinhados; tokens antigos não promovidos. |
| V7 — Snapshot histórico usado por navegação e tentativa de gate/write | Histórico divulgado; autorização exige revalidação atual de B34. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1–V3 |
| R2 | V1, V4 |
| R3 | V2, V3 |
| R4 | V4, V7 |
| R5 | V5 |
| R6 | V4, V5 |
| R7 | V6 |
| R8 | V1–V7 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 9. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-9-authoring.md).

### Residual risks

Compatibilidade dos tokens e garantia de snapshot precisam de decisão por tool/query; retenção pode consumir memória ou storage, por isso TTL e bytes são obrigatórios. Total exato pode exigir leitura integral. Não prometer menor custo em consultas não indexáveis nem depender de processo residente não entregue.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
