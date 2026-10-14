---
slug: pose-shared-runtime
status: draft
created_at: 2026-10-13
depends_on:
---

# Roadmap: Domínio e runtime comuns com desempenho qualificado

**Resultado:** operações CLI/MCP coerentes e custo controlado por produtor,
com reuso limitado e runtime opcional somente após qualificação de identidade,
coerência, autorização e recuperação. Consultar a [matriz](../results/pose-implementation-plan.json)
e a [análise de performance](../reports/2026-10-12-pose-front-08-resident-performance.md).

B32 foi materializado no lote 7 e B31/B33/B35 no lote 14, todos em draft.
A escrita dos quatro deltas não qualifica domínio/caches ou runtime; manter
standalone e opt-in, com resultados/limites demonstrados antes do closeout.

## Milestone: operation-performance
- after: spec:pose-check-spawns-fewer-git-processes, spec:pose-attention-within-a-second
- target_start:
- target_due:
- specs: pose-governance-operation-performance

**Exit gate:** B32 mede por produtor e reusa catálogo/bundles/Git por operação,
com clustering fiel e benchmark isolado que retém verdict/ambiente e não desfaz
arquivos do operador. Consultar o [lote 7](../reports/2026-10-13-pose-batch-7-authoring.md).
B32 é raiz independente de B14/B15; desempenho não autoriza gates sobre dados stale.


## Continuação prospectiva — lote 14

Consultar o [lote 14](../reports/2026-10-14-pose-batch-14-authoring.md) e o
[ADR de domínio/runtime Proposed](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md).
Resolver interface/IPC/ownership/compatibilidade e budgets antes de start.
B31/B33 não exigem B35; runtime consome ambos após seus demais pré-requisitos.

## Milestone: common-domain-handoff
- after: spec:pose-cli-mcp-operation-results, spec:pose-writer-coordination-and-atomic-publication, spec:pose-governance-read-view-generation, spec:pose-recoverable-closeout-plan, spec:pose-mcp-active-context, spec:pose-safe-validate-orchestration
- target_start:
- target_due:
- specs: pose-common-domain-and-cli-mcp-handoff

**Exit gate:** B31 preserva resultados/erros/authority por fonte de domínio comum;
wrappers extraídos não spawnam POSE. P04 tem route/context compatível ou limitação
explícita; MCP oferece plan read-only/handoff tipado, conservando P15. Writes
mantêm prova/coordenação e Harness não é substituído por execução arbitrária.

## Milestone: bounded-graph-caches
- after: spec:pose-index-generation-integrity, spec:pose-governance-read-view-generation, spec:pose-delivery-integrity-index-compaction
- target_start:
- target_due:
- specs: pose-bounded-isolated-graph-caches

**Exit gate:** B33 conserva grafo/filtragem com isolamento profundo, mutation/race
e redução medida de alocações/RAM; bytes/pins/waiters têm orçamento por worktree
e global. Misses coordenadas produzem uma vez por chave; nenhuma contaminação,
expansão total desnecessária ou cache de autorização stale. Git reader fica scoped.

## Milestone: optional-worktree-runtime
- after: operation-performance, common-domain-handoff, bounded-graph-caches, spec:pose-git-and-dispatch-snapshot-integrity, spec:pose-cli-mcp-operation-results, spec:pose-writer-coordination-and-atomic-publication, spec:pose-index-generation-integrity, spec:pose-governance-read-view-generation, spec:pose-attention-within-a-second
- target_start:
- target_due:
- specs: pose-optional-worktree-runtime

**Exit gate:** B35 qualifica handshake user/worktree/Git/engine/schema, endpoint
protegido, lifecycle/restart/update e fallback CI/standalone. Stdio não é IPC
compartilhável; watcher é hint, writers publicam geração e efeitos ambíguos exigem
receipt antes de retry. CPU/jobs simultâneos preservam query/QoS/output determinístico
com RAM/CPU/latência por cenário, sem scheduler/agents/workflows automáticos no core.

## Ativação e encerramento

- Manter draft durante a escrita; executar somente com escopo autorizado.
- Qualificar todos os deltas obrigatórios antes de encerrar o programa.
- Preservar verdict, coverage, bindings e limites de cada operação; latência menor não comprova eficácia universal.
- Conservar benchmarks isolados, resultados e erros; não restaurar trabalho do operador para medir.
- Adotar residência somente pelo contrato/qualificação correspondente, sem inferir que MCP stdio é IPC compartilhável.
