---
slug: pose-consolidation-program
status: draft
created_at: 2026-10-13
depends_on:
---

# Roadmap: Consolidar as dez frentes do POSE por entregas verificáveis

**Resultado:** completar e qualificar as unidades do escopo aprovado, preservando
história, autorização e compatibilidade, e demonstrar eficiência e utilidade nas
jornadas integradas. Planejamento de migração condicionada não equivale a apply.

Consultar a [matriz autoritativa](../results/pose-implementation-plan.json) para
os 44 Bxx, candidatos, lotes, specs propostas, dependências e responsáveis.
Consultar a [reconciliação do lote 0](../reports/2026-10-13-pose-plan-reconciliation.md)
para os PRs existentes, regras de reaproveitamento, aceite por frente e validação.

## Milestone: planning-bootstrap
- after:
- target_start:
- target_due:
- specs: pose-consolidation-program-bootstrap

**Exit gate:** matriz com 96 candidatos e 44 unidades, DAG e lotes íntegros,
evidências de #157/#158 reconciliadas e planejamento documental revisado.
Este milestone entrega somente o lote 0, não a implementação do programa.

## Etapas planejadas para materialização incremental

| Resultado | Roadmaps a continuar ou criar | Condição para qualificar |
|---|---|---|
| Verdade dos resultados | record-truthfulness, agency-readiness | Prova/coverage/authority/terminal não excedem a evidência verificada. |
| Consistência e recuperação | pose-state-consistency-and-recovery | Corrupção, lost update, geração parcial e crash/retry têm resultado fiel. |
| Delegação completa | delegated-review | Review, adjudication e smoke produzem conclusão governada, prova e fulfillment. |
| Residual, UX e releases | governance-efficiency, pose-release-sufficiency-and-recovery | Natureza/compromisso/efeito distinguíveis; cut, publicação e recovery verificáveis. |
| Domínio e runtime | pose-shared-runtime | Read views coerentes, caches limitados, IPC e fallback qualificados por worktree. |
| Qualificação integrada | governance-efficiency | Piloto B42 demonstra jornadas, utilidade, compatibilidade e rollback no escopo completo. |
| Migração condicionada | epistemic-lifecycle | Decisão por benefício e prova antes de criar e executar B43; manter exclusão explícita se adiado. |

Essas linhas são etapas de planejamento, não milestones executáveis. Não criar
referências nativas para arquivos ausentes. Materializar os milestones com
members e dependências reais quando os respectivos lotes forem escritos.

## Ativação e encerramento

- Manter o programa draft nesta fase de escrita; não iniciar implementação por inferência.
- Preservar specs done e os ADRs aceitos; abrir deltas prospectivos e continuar drafts adequados.
- Conferir uma autoridade executável por spec e um responsável planejado por unidade.
- Não alterar adoção/policy nem criar manifest de release a partir desta matriz.
- Materializar os milestones restantes e qualificar todas as unidades obrigatórias antes do encerramento.
- Revisar a matriz completa, os gates dos programas filhos e o piloto integrado; só então aplicar closeout nativo.
- Tratar o bootstrap encerrado como planejamento entregue; ele isoladamente não encerra este programa.
- Confrontar B43 com sua condição e disposition explícita; não computar migração adiada como implementação entregue.

## Controles de risco

Não fabricar datas, versões ou donos individuais. Não impor limite de 100 specs
para certificar um gate completo. Não confundir sucesso de consulta com coverage,
assinatura com aprovação semântica, run com conclusão ou MCP com runtime comum.
Registrar stop/go e rollback de adoção por instância. A revisão deve recusar
encerramento enquanto etapas planejadas continuarem sem materialização/aceite:
o check estrutural, sozinho, não valida essa completude futura.
