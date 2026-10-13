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

Materializar no lote 7 apenas B32. O member atual está draft e não implementa
daemon/cache residente; os outros deltas conservam seus lotes.

## Milestone: operation-performance
- after: spec:pose-check-spawns-fewer-git-processes, spec:pose-attention-within-a-second
- target_start:
- target_due:
- specs: pose-governance-operation-performance

**Exit gate:** B32 mede por produtor e reusa catálogo/bundles/Git por operação,
com clustering fiel e benchmark isolado que retém verdict/ambiente e não desfaz
arquivos do operador. Consultar o [lote 7](../reports/2026-10-13-pose-batch-7-authoring.md).
B32 é raiz independente de B14/B15; desempenho não autoriza gates sobre dados stale.

## Continuação planejada

| Unidades | Lote | Resultado a materializar |
|---|---|---|
| B31/B33/B35 | 14 | API/handoff comuns; vistas/caches isolados e limitados; runtime opcional por worktree e budgets. |

As linhas são planejamento; não criar refs nativas para specs ausentes.

## Ativação e encerramento

- Manter draft durante a escrita; executar somente com escopo autorizado.
- Qualificar todos os deltas obrigatórios antes de encerrar o programa.
- Preservar verdict, coverage, bindings e limites de cada operação; latência menor não comprova eficácia universal.
- Conservar benchmarks isolados, resultados e erros; não restaurar trabalho do operador para medir.
- Adotar residência somente pelo contrato/qualificação correspondente, sem inferir que MCP stdio é IPC compartilhável.
