---
slug: pose-bounded-isolated-graph-caches
status: in-progress
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

- modified: .pose/specs/2026-10-14-pose-bounded-isolated-graph-caches.md
- created: .pose/starts/pose-bounded-isolated-graph-caches.json
- created: .pose/changelogs/unreleased/pose-bounded-isolated-graph-caches.md
- created: pose-mcp/internal/pose/bounded_cache.go
- created: pose-mcp/internal/pose/cache_size.go
- created: pose-mcp/internal/pose/cache_accounting_test.go
- created: pose-mcp/internal/pose/cache_lifetime_test.go
- created: pose-mcp/internal/pose/bounded_cache_test.go
- created: pose-mcp/internal/pose/graph_view_isolation_test.go
- created: pose-mcp/internal/pose/graph_cache_measure_test.go
- created: pose-mcp/internal/pose/cache_invalidation_test.go
- modified: pose-mcp/internal/pose/delivery_integrity_cache.go
- modified: pose-mcp/internal/pose/delivery_integrity_cache_test.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/test_catalog.go
- modified: pose-mcp/internal/pose/delegated_verdicts.go
- modified: pose-mcp/internal/pose/design_delta.go
- modified: pose-mcp/internal/pose/git_spawn_cache.go
- modified: .pose/adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md
- modified: .pose/indexes/validation-matrix.json

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

Fixtures sintéticas e o índice real do repositório como corpus de medição. Os casos V1-V8 foram executados; a evidência de cada requisito está no rastreio e na check `bounded-isolated-graph-caches-integration`.

### Deterministic checks

`go test -race ./internal/pose` nos testes da check acima; a bateria completa de `internal/pose`, `internal/cli` e `internal/mcpserver`; a matriz estrita em clone limpo.

### Requirement trace
- R1 [satisfied] test:TestAViewOfTheGraphDoesNotCopyIt test:TestAViewSeesTheSameGraphEveryTimeAndTheCacheNeverWritesIt check:delivery-integrity-index-compaction-integration
- R2 [satisfied] test:TestADesignDeltaReportSharesNoReferenceWithItsCopies test:TestACachedGraphSharesNoReferenceWithItsCopies test:TestMutatingAReadNeverReachesTheNextRead test:TestDeliveryGraphCopyIsolatesProvenanceMaps test:TestDeliveryGraphCopyIsolatesMaterialInputs test:TestADesignDeltaReportIsIsolatedAndBounded
- R3 [satisfied] test:TestADesignDeltaIsAssessedOncePerKeyAndOnlyACompleteOneIsRetained test:TestTheGraphCacheKeyIsTheContentNotTheSizeOrTheOwner test:TestTheTestCatalogIsKeyedByTheMatrixAndMeteredPerRoot
- R4 [satisfied] test:TestASharedResultStaysChargedUntilItsLastWaiterHasCopiedIt test:TestTheTestCatalogDoesNotKeepTheFilesItReadItsNamesFrom test:TestAViewPinsTheEntryAndAnEvictedEntryStaysChargedUntilItLetsGo test:TestTheReservationIsHeldUntilTheResultIsPublished test:TestTheEstimateChargesEveryVariableSizedField test:TestTheDesignDeltaBudgetIsPerWorktree test:TestAValueNotRetainedIsNotCopiedForRetention test:TestProductionsAreAdmittedWithinTheInflightBudget test:TestBoundedCacheEvictsByBytesNotEntries test:TestBoundedCacheHoldsAnOwnerToItsBudget test:TestBoundedCacheDoesNotRetainWhatExceedsTheBudget test:TestRepeatedReadsAreHitsAndAlternationStaysUnderTheBudget test:TestADesignDeltaReportIsIsolatedAndBounded
- R5 [satisfied] test:TestAnOverloadedCatalogCacheIsReportedNotTakenForNoCatalog test:TestAQueueOfDistinctKeysIsBounded test:TestACancelledWaiterGivesItsPlaceBack test:TestOverloadIsTheSameRefusalOnEveryConsumer test:TestTheNumberOfProductionsIsCapped test:TestWaitersPerKeyAreCappedAndTheRefusedCallerIsToldToProduceForItself test:TestAReservationLargerThanTheBudgetRunsAloneInsteadOfStarving test:TestACopyDoesNotDelayAnotherKey test:TestADesignDeltaIsAssessedOncePerKeyAndOnlyACompleteOneIsRetained test:TestBoundedCacheProducesOncePerKeyUnderConcurrentMisses test:TestBoundedCacheDoesNotSerializeIndependentKeys test:TestBoundedCacheWaiterCancellationLeavesTheProducerAndOthers test:TestBoundedCacheDoesNotRememberAFailure test:TestConcurrentMissesOnTheSameIndexParseItOnce
- R6 [satisfied] test:TestCommitPathsAreReadOnceAndAFailureIsNotRemembered test:TestTheTestCatalogIsKeyedByTheMatrixAndMeteredPerRoot test:TestBoundedCacheExpiresByTTL
- R7 [satisfied] test:TestGraphByteEstimateTracksTheMeasuredHeap test:TestTheCostOfDeepIsolationIsMeasured test:TestAViewOfTheGraphDoesNotCopyIt test:TestRepeatedReadsAreHitsAndAlternationStaysUnderTheBudget
- R8 [satisfied] test:TestBoundedCacheSurvivesAlternatingOwnersUnderRace test:TestACancelledWaiterOnAGraphParseDoesNotAbandonTheOthers check:bounded-isolated-graph-caches-integration

### Known gaps
- Measured on the real index (10.9 MB, 6,937 nodes), heap and allocation only, one machine, one corpus, `go test` timings of single runs (no percentile is claimed): the first parse takes 190 ms and a hit through the deep copy 11.5 ms; the deep copy allocates 18.5 MB where the shallow copy it replaces allocated 5.7 MB (7.2 ms against 1.0 ms), which is the price of isolation; a read of one scope's digest through the view allocates 30 KB against 18.5 MB through the copy (4.2 ms against 11.2 ms, most of it hashing the index bytes for the key). Twelve concurrent first reads produce one parse (274 ms wall, eleven waiters coalesced). The retained size is estimated at 54 MB against 23 MB measured heap (walking every field), an estimate that errs high on purpose. RSS was not measured and no speedup of a whole command is claimed.
- Only two read-only consumers were moved to the view; the other callers of `GetDeliveryIntegrity` keep the deep copy.
- The view's contract (do not modify or keep the graph) is documented and tested on the cache side, not enforced by the compiler.

---

## 7. Final Report

### Delivered scope
One bounded cache (bytes, global and per worktree, least recently used, optional expiry) carries the parsed delivery graph, the design-delta report, the test catalog and the paths of a commit. A missed key is produced once however many callers ask, after reserving the bytes it expects to hold against global and per-worktree in-flight budgets, with a cap on productions and on waiters per key; a waiter that leaves does not disturb the producer or the others; a failure is never remembered; a value over its budget, or declared unretainable, is served and neither retained nor copied for retention; copies run outside the lock. Parsed graphs are keyed by engine, schema and the digest of the bytes, so identical bytes share an entry and nothing is served another worktree's graph. The test catalog is keyed by root and the digest of the validation matrix and expires after 30 seconds. The deep isolation is structural: a reflection test fills every slice, map and pointer of the graph and finds the references a copy shares with the original, which reproduced the aliasing the audit had not reproduced (the node attribute maps and the roadmap criteria refs) and fixed it. Read-only consumers can view the graph without a copy (`ViewDeliveryIntegrity`). Left out: moving every consumer to the view, RSS measurement, any persistent or cross-process cache, any global Git pool.

### Files and modules changed
- pose-mcp/internal/pose (bounded_cache.go, delivery_integrity_cache.go, delivery_integrity.go, delegated_verdicts.go, design_delta.go, test_catalog.go, git_spawn_cache.go and their tests)
- ADR shared-domain-and-optional-worktree-runtime (amendment); the validation matrix gains `bounded-isolated-graph-caches-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head, and the suites of `internal/pose`, `internal/cli` and `internal/mcpserver`
- Result: recorded by the sealed validation evidence; independent delegated review by a different vendor, concluded with `pose review conclude`.

### Residual risks
- The clone a caller keeps is the caller's memory and is outside the budget; the view exists for readers that keep nothing. So are the inputs a caller read before asking the cache (the raw index bytes a waiter holds) and the clone being made for a caller: the cache charges what it keeps alive for itself, retained entries, productions, shared results until their last waiter copied them, and pinned entries. Bounding what callers allocate before and after the call is not done here; reading a smaller input is the follow-up below.
- An overloaded catalog cache is reported: lint-spec and the closeout gate fail with "test references could not be verified" instead of treating the missing catalog as an empty one; callers that only skip resolution keep the lenient form.
- A consumer that needs a mutable graph pays about 7 ms and 18 MB per read; moving more consumers to the view lowers it.
- The estimate of retained bytes errs high, so the cache holds less than its budget in practice.
- The 30-second expiry of the test catalog is a policy, not a proof of freshness; gates read the matrix digest in the key.

### Follow-ups

- [open] Move the remaining read-only consumers of the delivery graph to `ViewDeliveryIntegrity` and measure the allocation saved per command. (owner:@pose-maintainers crit:low review:2027-01-08)
- [open] Bound what callers hold around the cache: read the delivery index in a bounded stream or by a size-gated admission before the bytes are allocated, and reserve the bytes of the clone a caller is about to receive. (owner:@pose-maintainers crit:medium review:2027-01-08)
- [open] Measure RSS and the retained heap of a long-lived MCP server that alternates between projects, to settle the budgets from more than one corpus. (owner:@pose-maintainers crit:low review:2027-01-08)
