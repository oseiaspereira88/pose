---
slug: pose-state-consistency-and-recovery
status: draft
created_at: 2026-10-13
depends_on:
---

# Roadmap: Consistência de estado e recuperação verificável

**Resultado:** preservar verdade de leitura, snapshot, escrita e recuperação,
sem converter unknown em autorização ou geração parcial em complete.
Consultar a [matriz](../results/pose-implementation-plan.json) e a
[análise de consistência](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

Este responsável foi materializado no lote 2 para B07, antecipando somente o
arquivo previsto no lote 7. Os lotes 4 e 7 acrescentam B12/B13 e B14/B15; B34/B36 conservam seu lote.
Os sete members estão materializados em draft após o lote 9; existência não qualifica as sete unidades ou permite encerramento.

## Milestone: git-dispatch-snapshot
- after: spec:pose-recoverable-closeout-plan, spec:pose-delegated-review-dispatch
- target_start:
- target_due:
- specs: pose-git-and-dispatch-snapshot-integrity

**Exit gate:** B07 recusa clean/complete com Git desconhecido; verifica bytes e
inputs pertinentes antes/depois do dispatch, paths e outputs autorizados;
bundle superado conserva história sem efeito no scope vigente.

## Continuação prospectiva — lote 4

Consultar o [lote 4](../reports/2026-10-13-pose-batch-4-authoring.md); executar
somente quando o escopo funcional correspondente for autorizado.

## Milestone: operation-results
- after: spec:pose-recoverable-closeout-plan
- target_start:
- target_due:
- specs: pose-cli-mcp-operation-results

**Exit gate:** B12 compartilha outcome tipado entre CLI/MCP, com um JSON por
chamada, parsing estrito e distinção entre waiting/deadline/conflict, verdict,
coverage e efeitos confirmados. B12 é raiz independente de B07.

## Milestone: journal-checkpoint-lock-recovery
- after: operation-results, spec:pose-abm-atomic-start
- target_start:
- target_due:
- specs: pose-journal-checkpoint-and-lock-recovery

**Exit gate:** B13 exige EOF/schema/seq integrais; checkpoint corrupt/unavailable
não vira absent; resume revalida inputs/progresso e falha de persistência é visível.
Owner vivo/desconhecido nunca tem lock quebrado por idade. Runtime futuro deve
consumir o contrato; este milestone não declara daemon implementado.

## Continuação prospectiva — lote 7

Consultar o [lote 7](../reports/2026-10-13-pose-batch-7-authoring.md).

## Milestone: writer-publication
- after: operation-results, journal-checkpoint-lock-recovery, spec:pose-spec-authority-transfer, spec:pose-abm-atomic-start
- target_start:
- target_due:
- specs: pose-writer-coordination-and-atomic-publication

**Exit gate:** B14 usa staging exclusivo e coordenação/CAS pertinente; sem lost
update, overwrite de imutável ou dupla autoridade. Curated e efeitos antes de
erro têm preservação/receipt e recovery verificáveis; limites de writers externos explícitos.

## Milestone: index-generation
- after: writer-publication, spec:pose-delivery-integrity-index-compaction
- target_start:
- target_due:
- specs: pose-index-generation-integrity

**Exit gate:** B15 publica conjunto completo identificado ou unavailable;
reader fixa geração e retry/crash não mistura versões. Compatibilidade dos
consumers e retenção de geração viva qualificadas antes de encerramento.

## Continuação prospectiva — lote 9

Consultar o [lote 9](../reports/2026-10-13-pose-batch-9-authoring.md).
O catálogo deste responsável está materializado; implementar/qualificar cada
unidade e seus gates continua trabalho separado. Nenhum daemon é autorizado.

## Milestone: governance-read-view
- after: git-dispatch-snapshot, writer-publication, index-generation, spec:pose-source-coverage-integrity, spec:pose-obligation-projection, spec:pose-spec-authority-transfer
- target_start:
- target_due:
- specs: pose-governance-read-view-generation

**Exit gate:** B34 identifica bytes/closure efetivamente observados e revalida
inputs relevantes, incluindo journals/subjects/policy/tests/matriz/índices/refs;
interleaving causa retry limitado ou incoherence/unknown, nunca mistura ou relabel.
Mesmo mtime, dirty preexistente, rename/checkout e perda de watcher não certificam
current. Coverage perdida afeta Complete; gate/write exige revalidação/CAS no efeito,
com limites de writers externos explícitos. Qualificar antes de reuso entre operações.

## Milestone: generation-query-cursors
- after: governance-read-view, spec:pose-obligation-projection
- target_start:
- target_due:
- specs: pose-generation-bound-query-cursors

**Exit gate:** B36 vincula cursor à geração/query/filtros/ordem/escopo e oferece
snapshot identificado ou stale/restart explícito após mutação. Limitar retenção,
bytes e page size; limit reduz trabalho quando indexável e custo de total/scan é
honesto. Não truncar gates nem promover snapshot histórico a autorização atual.
Qualificar tokens, compatibilidade, empates, mutação e paridade entre consumidores.

## Ativação e encerramento

- Manter draft durante a escrita; executar somente com escopo funcional autorizado.
- Materializar e qualificar todos os deltas obrigatórios antes de encerrar o programa.
- Preservar identidade de worktree, histórico e compatibilidade; não alegar sandbox pelo check antes/depois.
- Consumir evidência real de corrida/crash/retry quando o delta correspondente existir.
- Não encerrar este roadmap pelo aceite isolado de parte dos sete members ou por resolução de suas refs.
