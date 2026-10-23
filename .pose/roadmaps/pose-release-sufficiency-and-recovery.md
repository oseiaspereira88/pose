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

B38 foi materializado no lote 6, B37 no lote 12 e B39/B41/B40 no lote 13,
todos em draft. A escrita dos cinco deltas não encerra o programa; qualificar
separadamente cada resultado antes do closeout. Preservar a fundação
done de lifecycle e a composição/federação já existentes.

## Milestone: manifest-tag-integrity
- after: spec:pose-release-lifecycle-closure
- target_start:
- target_due:
- specs: pose-release-manifest-tag-integrity

**Exit gate:** B38 valida schema/projeções/digests e manifest/archive/notes/version
source no tree da tag, sem fallback ao worktree ou mover tag. Evidence posterior
ou novo ciclo não staleiam cut correto. Consultar o [lote 6](../reports/2026-10-13-pose-batch-6-authoring.md).


## Milestone: cut-sufficiency
- after: spec:pose-action-answers-reverify-on-read, spec:pose-review-disclosure-requires-proof, spec:pose-review-approval-domain-preflight, spec:pose-action-target-and-node-scoping, spec:pose-spec-facts-evidence-binding, spec:pose-release-lifecycle-closure
- target_start:
- target_due:
- specs: pose-release-shared-sufficiency-contract

**Exit gate:** B37 faz plan/apply avaliar o mesmo recorte e revalidar inputs antes
do freeze; blocker alheio não vira global. Delivery exige prova/profile/provenance
adequados; ref sem evaluator e heading/intenção não reconhecidos têm diagnóstico,
e path existente não prova approval. Reusar evaluators e ADRs existentes; deferral
não afirma entrega e história/publicação observada não é reescrita por preflight.
Consultar o [lote 12](../reports/2026-10-13-pose-batch-12-authoring.md).


## Continuação prospectiva — lote 13

Consultar o [lote 13](../reports/2026-10-14-pose-batch-13-authoring.md) e o
[ADR temporal/lineage Proposed](../adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md).
Reconciliar e resolver schema/compatibilidade antes do start funcional; preservar
ADRs aceitos e história. B39/B41 não dependem um do outro; B40 integra ambos.

## Milestone: publication-evidence-replay
- after: manifest-tag-integrity, spec:pose-historical-trust-and-review-reuse, spec:pose-release-lifecycle-closure, spec:pose-reproducible-release-verification
- target_start:
- target_due:
- specs: pose-release-publication-evidence-replay

**Exit gate:** B39 revalida hashes, identidade por papel, assets/publication binding
e cobertura de plataforma/canal. P05 separa candidato stale de história íntegra;
P07–P09 não elevam prova alterada/parcial a verified total. Knobs têm suporte/efeito
observados, retries preservam lineage e proof superseded não autoriza open-next.
Import factual validado não depende de autorização retrospectiva do cut.

## Milestone: prepare-record-recovery
- after: manifest-tag-integrity, spec:pose-cli-mcp-operation-results, spec:pose-journal-checkpoint-and-lock-recovery, spec:pose-writer-coordination-and-atomic-publication, spec:pose-release-lifecycle-closure
- target_start:
- target_due:
- specs: pose-release-prepare-record-recovery

**Exit gate:** B41 persiste intenção/fases/inventário e efeitos confirmados; crash,
concorrência e retry convergem sem duplicar evento ou admitir transição stale.
Compensação falha preserva material e erro/receipt, sem limpeza indiscriminada
ou manifest inventado por backfill; durabilidade e power loss têm limites claros.

## Milestone: scope-selection-exclusivity
- after: cut-sufficiency, manifest-tag-integrity, publication-evidence-replay, prepare-record-recovery, spec:pose-followup-metadata-and-routing-integrity, spec:pose-residual-nature-commitment-and-effect, spec:pose-release-lifecycle-closure, spec:pose-release-prepare-guards-claimed-fragments
- target_start:
- target_due:
- specs: pose-release-scope-selection-and-exclusivity

**Exit gate:** B40 seleciona subconjunto coerente com scope/refs/digests, mantendo
exclusão, deferral e outcome distintos. P10 recusa assignment duplicada em plan/apply
antes de mover fila; descoberta compartilhada e source loss não viram ausência.
Preflight revalida inputs/authority e recovery move somente seleção comprovada,
sem dispensar compromisso por renomeação ou exigir concluir roadmap alheio.

## Ativação e encerramento

- Manter draft durante a escrita; executar somente com escopo correspondente autorizado.
- Materializar/qualificar todas as unidades obrigatórias antes de encerrar o programa.
- Preservar contratos legacy, manifests congelados e tags; checker não corrige história.
- Distinguir prepared/tagged/published/verified e coverage, sem inventar rollback ou receipt.
- Decidir cuts/versões pela suficiência verificada, sem promessa de release por nome do roadmap.

### B38 — closeout independente (2026-10-19)

PR #198 conclui B38 após a décima terceira revisão independente no HEAD `07e77f89`.
Revisor `agent:independent-claude-opus-5-5-review`, Claude/Anthropic, o mesmo vendor
do implementador: não há separação de vendor, só de execução e de contexto, porque o
Codex está indisponível (limite de uso até 2026-10-24). Identidade e separação são
declaradas, e a atestação no pose-dist não tem envelope. Nenhuma policy foi alterada.

Validação estrita global 140/140 e módulo 137/137. Bundle `rvb-7a5a4b8cac65cf0b`,
atestação `rva-fc93233a05b7e708`. Review verify fresh/approved e close nativo aprovados.
Spec done e B38 delivery_complete no [plano](../results/pose-implementation-plan.json).
Os vinte follow-ups low permanecem `[open]` na spec (owner @core, revisão 2026-12-01).
Nenhum deles permite aprovação indevida. Parecer em
[review B38r13](../reports/2026-10-19-b38r13-claude-independent-review.md) e
[evidência compacta](../results/b38r13-claude-independent-review.json). Aguardar o merge
pelo maintainer. Os demais milestones deste roadmap continuam em draft.
