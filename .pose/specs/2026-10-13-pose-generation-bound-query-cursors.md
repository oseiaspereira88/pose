---
slug: pose-generation-bound-query-cursors
status: done
created_at: 2026-10-13
completed_at: 2026-11-08
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

- modified: .pose/specs/2026-10-13-pose-generation-bound-query-cursors.md
- created: .pose/starts/pose-generation-bound-query-cursors.json
- created: .pose/changelogs/unreleased/pose-generation-bound-query-cursors.md
- created: .pose/adr/2026-11-08-generation-bound-query-cursors.md
- modified: pose-mcp/internal/mcpserver/pagination.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- created: pose-mcp/internal/pose/generation_query.go
- created: pose-mcp/internal/pose/generation_query_test.go
- created: pose-mcp/internal/mcpserver/generation_cursor_test.go
- created: pose-mcp/internal/mcpserver/generation_cursor_linux_test.go
- created: pose-mcp/internal/mcpserver/pagination_identity_linux.go
- created: pose-mcp/internal/mcpserver/pagination_identity_other.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:generation-bound-query-cursors module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/mcpserver/pagination.go

### Approach and execution gates

Executar depois de B34. Inventariar ferramentas paginadas e queries de domínio antes de start; registrar todos os handlers/consumers alterados no plano exato. Reconciliar schema de paginação existente e decidir snapshot-retained versus restart, binding/proteção do token, retention e total/custo por query. Usar read view de domínio, sem fatiar uma lista recomposta a cada página. Gates continuam completos no recorte solicitado; paginação de exibição não impõe limite de 100 specs ao aceite. A decisão de retenção não implementa B35/IPC/scheduler, e cursores de eventos HTTP/Redis conservam contrato próprio.

## 5. Decisions

ADR: `.pose/adr/2026-11-08-generation-bound-query-cursors.md`.

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Continuar somente na geração e query vinculadas ao cursor, com obsolescência e custo explícitos.
- Rationale: Existência de dados ou sucesso de transporte não prova coerência, frescor ou autoridade; preservar bindings e incerteza impede autorizar com leitura obsoleta.

## 6. Validation

### Strategy

Fixtures sintéticas e coleção fixa por teste. Os casos abaixo foram executados; a evidência de cada requisito está no rastreio de requisitos e na check `generation-bound-query-cursors-integration`.

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

Aplicar os gates do `review-plan` e as rules vigentes. Durante a correção, executar regressões focadas no delta e nos controles afetados; no fechamento, registrar evidência canônica atual que cubra o módulo. Uma validação global que já contenha seus resultados pode responder também ao componente quando coverage/proveniência nativas comprovarem isso, sem repetir a suíte apenas por haver dois IDs de ferramenta. Preservar todos os checks obrigatórios e repetir o que uma mudança material invalidar.

Qualificar as consultas alteradas, cursor stale e mudança de geração com fixtures de paginação e custo de leitura. Reusar B34 sem repetir sua campanha completa de falhas de writer/publicação.

Os casos da tabela são objetivos de cobertura e podem compartilhar fixtures e execuções. Reusar corpus e controles das fundações sem recriar sua qualificação inteira. Qualquer ensaio adicional deve responder a risco, contrato ou finding identificado, com escopo e limite definidos.

### Requirement trace
- R1 [satisfied] test:TestFiltersThatSerializeAlikeBindDifferentCursors test:TestAnOmittedSortAndSlugSortAreTheSameQuery test:TestPageByGenerationWalksTheCollectionExactlyOnce test:TestACursorStaysWithItsQuery test:TestEquivalentFiltersShareACursor check:generation-bound-query-cursors-integration
- R2 [satisfied] test:TestACursorIsRefusedWhenTheCollectionChanged test:TestACursorIsRefusedAfterTheCollectionChangesBetweenPages test:TestACursorIsRefusedWhenAReturnedFieldChanges
- R3 [satisfied] test:TestANullCursorIsRefusedOnEveryPaginatedTool test:TestACursorAliasCannotRestartAWalk test:TestAnIssuedCursorStaysWithinTheSizeBoundForHugeFilters test:TestHostileCursorsAreRefusedNotCoerced test:TestAnExpiredCursorIsRefused test:TestAPositionOutsideTheCollectionIsRefused test:TestAHostileCursorIsRefusedOverMCP
- R4 [satisfied] test:TestAReusedGenerationSeesAnEditThatPreservesSizeAndModTime test:TestAReusedGenerationSeesASymlinkTargetChange test:TestAPageCarriesTheIndexReadersStatusReasonAndDivergedFiles test:TestAPageStatesItsCostAndGenerationStatus test:TestEveryPageDisclosesItsGenerationAndCollection test:TestPageByGenerationWalksTheCollectionExactlyOnce
- R5 [satisfied] test:TestAPageDoesNotReverifyAnUnchangedGeneration test:TestEveryPageStatesItsCost test:TestPageByGenerationClampsAndDisclosesTheLimit test:TestPageLimitIsBoundedAndDisclosed
- R6 [satisfied] test:TestAnAliasOfLimitOrCursorIsRefused test:TestAnExplicitNonPositiveLimitIsRefusedOnEveryPaginatedTool test:TestAContinuationWithoutALimitIsStillABoundedPage test:TestAContinuationWithoutALimitIsBoundedOverMCP test:TestACursorDoesNotSurviveAnotherServerProcess test:TestAnExpiredCursorIsRefused
- R7 [satisfied] test:TestPageByGenerationWithoutCursorOrLimitIsTheLegacyCall test:TestPaginationOmittedIsUnpaginatedAndBackwardCompatible test:TestHostileCursorsAreRefusedNotCoerced
- R8 [satisfied] test:TestPageCostIsMeasuredAndBounded test:TestACursorIsRefusedWhenTheCollectionChanged test:TestACursorStaysWithItsToolAndFilters test:TestACursorDoesNotSurviveAnotherServerProcess

### Known gaps
- Measured (R8, `TestPageCostIsMeasuredAndBounded`, Linux `rchar` of the process, synthetic 4 KB specs): 20 specs read 95 KB for the whole call and 89 KB for `limit=5`; 80 specs read 374 KB and 340 KB. The bytes read grow linearly with the collection, a page costs about the whole call (limit bounds what is returned, not what is read), and the response follows the page (2 KB for `limit=5` against 4.6 KB and 16.5 KB for the whole call). Latency was 0.4-1.6 ms. With a published generation the first call also verifies it (14.7 MB read for seven 1 MB files) and the next page reads 31 KB, because the verification is reused while the index files are unchanged. The indexed case is dispositioned: none of the five list collections is indexed, so no reduction of reads is claimed; giving them an index is the open follow-up below.
- The list tools load their whole collection, so a page costs one recomputation and the total is a full scan; this is disclosed, and no index-backed limit was introduced because none of the five collections is indexed.
- The CLI has no cursor-paginated surface; only the MCP list tools paginate.

---

## 7. Final Report

### Delivered scope
Paginated MCP list tools continue a query over the collection the cursor was issued for. A cursor is signed (HMAC with a process-held key), expires after 15 minutes, and is bound to its tool, project, filters and the ordered collection with its index generation. A changed collection, another query, an expired token, a version 1 token, an altered token or a server restart is refused with a `cursor-*` code and the caller restarts from the first page; nothing is continued over a changed collection or restarted silently. Pages are bounded to 500 items, and each discloses its generation, collection digest, position and total, with the statement that a missing `next_cursor` certifies nothing beyond that collection. Calls without cursor and limit are unchanged. The five tools `pose_list_specs`, `pose_list_roadmaps`, `pose_list_knowledge`, `pose_list_reports` and `pose_capability_history` share one helper. Left out: index-backed limits, a durable key and retained snapshots (ADR rejects them).

### Files and modules changed
- pose-mcp/internal/pose (generation_query.go and its tests)
- pose-mcp/internal/mcpserver (pagination.go, server.go, generation_cursor_test.go, tool catalog golden)
- ADR generation-bound-query-cursors; POSE.md and its scaffold and pt-BR copies; the validation matrix gains `generation-bound-query-cursors-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head
- Result: recorded by the sealed validation evidence; independent delegated review by a different vendor, concluded with `pose review conclude`.

### Residual risks
- A cursor never outlives the server process; clients must restart a walk after a restart.
- A page discloses the index reader's status, reason and the legacy files that diverge from the verified generation.
- A publication of the index generation between reading the collection and labelling the page is refused (retry), not mislabelled; an unavailable or diverged generation is disclosed in `generation_status`.
- The total is a full scan of the recomputed collection for these tools.
- A page is a navigation view: gates and effects read current inputs and must not consume a page.

### Follow-ups

- [open] Give the list collections an index so that `limit` bounds the read, parse and copy work instead of only the page size, then measure it against the corpus. (owner:@pose-maintainers crit:low review:2027-01-08)
