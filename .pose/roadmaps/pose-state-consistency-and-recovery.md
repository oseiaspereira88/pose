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
arquivo previsto no lote 7. O lote 4 acrescenta B12/B13; os demais deltas conservam
seus lotes. Os três members atuais estão draft; não representar o recorte como
programa completo.

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

## Continuação planejada

| Unidades | Lote de escrita | Resultado a materializar |
|---|---|---|
| B14/B15 | 7 | Escritas concorrentes e recuperação de geração parcial. |
| B34/B36 | 9 | Read views/invalidação; cursor e consulta ligados à geração. |

As linhas são planejamento; não criar refs nativas para specs ausentes.
Conferir os aceites e dependências exatos na matriz em cada lote.

## Ativação e encerramento

- Manter draft durante a escrita; executar somente com escopo funcional autorizado.
- Materializar e qualificar todos os deltas obrigatórios antes de encerrar o programa.
- Preservar identidade de worktree, histórico e compatibilidade; não alegar sandbox pelo check antes/depois.
- Consumir evidência real de corrida/crash/retry quando o delta correspondente existir.
- Não encerrar este roadmap pelo aceite isolado de B07/B12/B13 ou por resolução de suas refs.
