---
slug: pose-bounded-isolated-graph-caches
status: draft
created_at: 2026-10-14
completed_at:
depends_on: pose-index-generation-integrity, pose-governance-read-view-generation, pose-delivery-integrity-index-compaction
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:bounded-isolated-graph-caches
---

# Spec: Isolar vistas de grafo e limitar caches por bytes e geração

## 1. Intent

### Goal

Preservar grafo/filtragem e isolamento profundo entre consumidores com menor custo medido de cópia/expansão e misses coordenadas.

### Business value

B33 / PERF-04, PERF-10 e CONS-11. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de performance](../reports/2026-10-12-pose-front-08-resident-performance.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Preservar semântica de grafo, filtragem, identity/refs/findings e coverage para schemas legados/compactos. Consultas pequenas não devem expandir/copiar o grafo inteiro desnecessariamente; representar vistas/projeções imutáveis por API ou cópias defensivas profundas onde o contrato público ainda permite mutação, sem omitir dados para ganhar desempenho.
- R2: Qualificar isolamento profundo de todos os campos mutáveis, incluindo Nodes.Attributes, maps/slices aninhados, sets/resultados/ChangeSets e índices reverse/paths. Mutação de uma leitura/filtro não altera consumidor concorrente ou próxima leitura; testes de campos externos/copied slices isoladamente não bastam. Tratar a hipótese de aliasing observada no código como não reproduzida até fixture/race.
- R3: Key de cache deve refletir worktree/raiz autorizada, engine/parser/schema, conteúdo e dependências/generation relevantes de B34/B15. Duas worktrees/clones/raízes federadas ou revisões não se contaminam; same size/mtime não prova equivalência. Cache de dado reconstruível não é cache universal de verdict, satisfaction ou authority atual.
- R4: Limitar bytes retidos por worktree e globalmente, incluindo raw bytes, estruturas expandidas, índices, vistas/cópias vivas e filas/pin de produção. Fixar accounting/eviction/TTL e limites antes do código; limite de número de entradas não prova RAM limitada. Alternância de projetos/gerações não cresce sem orçamento ou causa retenção eterna.
- R5: Coordenar misses por chave para apenas uma produção concorrente, sem mutex global cobrindo parsing/cópia longa de chaves independentes. Cancelamento de um waiter não abandona outros indevidamente; erros têm coverage e política explícita de retry/negative caching, sem memorizar falha como ausência ou sucesso. Produções e waiters também têm orçamento.
- R6: Revisar caches designDelta/testCatalog/Git e lifetime de CLI curta antes de torná-los duradouros; invalidação considera testes/matriz/módulos/policies/refs/raízes usados. Preservar o ADR de Git batch scoped: este delta não cria pool global ou stream compartilhado. Dados imutáveis podem ter reuso admitido, enquanto working tree/gates dependem de generation validada.
- R7: Medir alocações, heap/RSS/retained bytes, hits/misses/evicções/produção por chave e latência no mesmo corpus/ambiente com operação/filtro identificados. Mostrar redução real de RAM/alocações contra baseline e trade-offs sem tomar compactação em disco como ganho de heap, máximo RSS de filho como soma de processos ou três amostras como p95.
- R8: Qualificar mutation tests profundos e race detector com alternância de projetos/gerações, misses simultâneas e cursor/filtro stale, preservando outputs determinísticos e controles válidos. Resolver proposal/schema/API/compatibility e ownership da vista antes de start; não iniciar runtime, carregar toda .pose em RAM ou afirmar speedup sem medição.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-14-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-14-pose-bounded-isolated-graph-caches.md
- created: .pose/starts/pose-bounded-isolated-graph-caches.json
- created: .pose/changelogs/unreleased/pose-bounded-isolated-graph-caches.md
- modified: pose-mcp/internal/pose/delivery_integrity_cache.go
- modified: pose-mcp/internal/pose/delivery_integrity_cache_test.go
- modified: pose-mcp/internal/pose/delivery_integrity_compact.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/design_delta.go
- modified: pose-mcp/internal/pose/test_catalog.go
- modified: pose-mcp/internal/pose/git_spawn_cache.go
- created: pose-mcp/internal/pose/graph_view_isolation_test.go
- created: pose-mcp/internal/pose/cache_budget_concurrency_test.go
- created: pose-mcp/internal/pose/graph_view_benchmark_test.go
- modified: .pose/adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:bounded-isolated-graph-caches module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/delivery_integrity_cache.go

### Approach and execution gates

Executar após B15/B34. O cache atual lê conteúdo exato, guarda última entrada, permite decode de misses concorrentes e copia Nodes superficialmente; compactação schema2 ainda expande para grafo. Reusar a fundação done sem repetir seu ganho histórico. Definir API de vistas que não exponha coleções mutáveis compartilhadas, migração dos consumers e accounting/budgets com fixtures. A [proposta de runtime/cache](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md) e knowledge:adr-shared-worktree-runtime-review são prospectivos; completar todas estruturas/cache owners/consumers e paths antes do start. Não prometer zero-copy genérico nem bound de RAM apenas por LRU/TTL sem contabilizar pins/cópias. B35 só consome residência após qualificação; não dependência B35 para corrigir isolamento.

## 5. Decisions

### Decision D1
- Proposal: [Shared domain and optional worktree runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md), status Proposed; knowledge:adr-shared-worktree-runtime-review.
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Compartilhar somente vistas efetivamente isoladas, com budget em bytes e produção coordenada por chave validada.
- Rationale: Cópia superficial e contagem de entradas não provam isolamento profundo ou RAM limitada; cada otimização deve preservar grafo e cobertura.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Grafo legado/compacto e filtros pequenos/grandes | Mesma semântica/coverage, sem expansão total desnecessária para query pequena. |
| V2 — Mutation de Attributes/maps/slices e dois consumidores | Isolamento profundo e próxima leitura íntegra; controles positivos. |
| V3 — Worktrees/projetos/raízes federadas e mesmo mtime/tamanho | Keys/generation diferentes, sem contaminação ou equivalência inferida. |
| V4 — Muitas gerações, raw+expanded+pins+vistas e eviction | Bytes globais/por worktree limitados, métricas explicam retenção. |
| V5 — Misses mesma chave/diferentes, waiter cancel e erro de producer | Uma produção por chave; orçamento/coverage e concorrência independentes. |
| V6 — Testes/matriz/policy/refs mudados e Git reader scoped | Invalidação pertinente; nenhum pool global ou cache de autorização stale. |
| V7 — Benchmark fixo, alocações/RSS/heap e corpus histórico | Ganho/trade-offs medidos com limites estatísticos; disco não prova heap. |
| V8 — Race, filtros/cursor stale e read legacy | Determinismo/compatibility, sem write ou runtime iniciado. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V8 |
| R2 | V2, V8 |
| R3 | V3, V6 |
| R4 | V4, V5 |
| R5 | V5 |
| R6 | V3, V6 |
| R7 | V7 |
| R8 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 14. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-14-pose-batch-14-authoring.md).

### Residual risks

A amostra de auditoria não reproduziu corrupção por consumidor nem data race nativa; a implementação precisa mostrar fixture falhando/controle e race detector. Orçamento inclui objetos ainda vivos, não apenas o map de cache. Dados cacheados não dispensam revalidação de authority ou geração de gate.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
