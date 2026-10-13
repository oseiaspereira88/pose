---
slug: pose-release-sufficiency-and-recovery
status: draft
created_at: 2026-10-13
depends_on:
---

# Roadmap: Suficiência e recuperação de releases verificáveis

**Resultado:** selecionar escopo suficiente, provar o cut/snapshot e revelar
publicação/recuperação com evidência fiel. Consultar a [matriz](../results/pose-implementation-plan.json)
e a [análise de release](../reports/2026-10-12-pose-front-06-release-sufficiency.md).

Materializar somente B38 no lote 6. Manter os outros deltas nos seus lotes e
não encerrar este recorte parcial como programa completo. Preservar a fundação
done de lifecycle e a composição/federação já existentes.

## Milestone: manifest-tag-integrity
- after: spec:pose-release-lifecycle-closure
- target_start:
- target_due:
- specs: pose-release-manifest-tag-integrity

**Exit gate:** B38 valida schema/projeções/digests e manifest/archive/notes/version
source no tree da tag, sem fallback ao worktree ou mover tag. Evidence posterior
ou novo ciclo não staleiam cut correto. Consultar o [lote 6](../reports/2026-10-13-pose-batch-6-authoring.md).

## Continuação planejada

| Unidade | Lote | Resultado a materializar |
|---|---|---|
| B37 | 12 | Escopo coerente e sufficiency da seleção. |
| B39/B40/B41 | 13 | Replay temporal, seleção/exclusividade e recovery de publicação conforme aceites da matriz. |

As linhas são planejamento; não criar refs para specs ausentes. Conferir títulos,
artefatos e dependências exatos da matriz ao escrever cada lote.

## Ativação e encerramento

- Manter draft durante a escrita; executar somente com escopo correspondente autorizado.
- Materializar/qualificar todas as unidades obrigatórias antes de encerrar o programa.
- Preservar contratos legacy, manifests congelados e tags; checker não corrige história.
- Distinguir prepared/tagged/published/verified e coverage, sem inventar rollback ou receipt.
- Decidir cuts/versões pela suficiência verificada, sem promessa de release por nome do roadmap.
