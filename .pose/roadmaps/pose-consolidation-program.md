---
slug: pose-consolidation-program
status: active
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

## Milestone: batch-one-authoring
- after: planning-bootstrap
- target_start:
- target_due:
- specs: pose-consolidation-batch-1-authoring

**Exit gate:** três deltas escritos em draft com aceite, artefatos prospectivos,
dependências e vínculos atuais validados. Este milestone qualifica a escrita,
não as implementações de B04/B02/B05.

## Milestone: batch-two-authoring
- after: batch-one-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-2-authoring

**Exit gate:** B03/B06/B07 escritos em draft; matriz, coverage de aceites,
ownership e refs válidos. Qualificar a escrita separadamente do comportamento.

## Milestone: trust-foundations
- after: batch-two-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:record-truthfulness/temporal-trust, xref:proj.pose-dist/milestone:record-truthfulness/verified-proof, xref:proj.pose-dist/milestone:record-truthfulness/approval-preflight, xref:proj.pose-dist/milestone:record-truthfulness/policy-scopes, xref:proj.pose-dist/milestone:record-truthfulness/presence-channel-parity

**Exit gate:** consumir o aceite verificado dos cinco milestones de confiança
do roadmap responsável, sem copiar a ownership das specs. Não encerrar esta
etapa quando apenas a escrita passou. Os demais resultados ainda precisam de
materialização e qualificação nos seus lotes.

## Milestone: source-and-snapshot-integrity
- after: batch-two-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:agency-readiness/source-coverage-integrity, xref:proj.pose-dist/milestone:pose-state-consistency-and-recovery/git-dispatch-snapshot

**Exit gate:** consumir coverage por fonte de B06 e integridade Git/dispatch de
B07 nos responsáveis. Ambos são raízes independentes da cadeia B01/B04/B03;
existência de specs e sucesso documental não qualificam os resultados.

## Milestone: batch-three-authoring
- after: batch-two-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-3-authoring

**Exit gate:** B08/B09/B10 escritos em draft, com aceites, artefatos prospectivos,
casos, dependências e vínculos válidos; nenhuma eficácia funcional inferida.

## Milestone: action-condition-integrity
- after: batch-three-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:agency-readiness/action-target-scoping, xref:proj.pose-dist/milestone:agency-readiness/action-lifecycle-authority, xref:proj.pose-dist/milestone:agency-readiness/action-recovery-preflight

**Exit gate:** consumir os três resultados de actions no responsável, sem
duplicar ownership. B08 é independente; B09 exige B01/B05 e B10 exige B09.
Escrita, aprovação do PR e refs resolvidas não qualificam esses comportamentos.

## Milestone: batch-four-authoring
- after: batch-three-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-4-authoring

**Exit gate:** B12/B13/B30 escritos em draft, com aceites, artefatos prospectivos,
casos, dependências e vínculos válidos; nenhuma eficácia funcional inferida.

## Milestone: operational-recovery-and-facts
- after: batch-four-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:pose-state-consistency-and-recovery/operation-results, xref:proj.pose-dist/milestone:pose-state-consistency-and-recovery/journal-checkpoint-lock-recovery, xref:proj.pose-dist/milestone:governance-efficiency/spec-facts-binding

**Exit gate:** consumir resultados operacionais, recovery e facts nos responsáveis,
sem duplicar ownership. B12/B30 são raízes; B13 exige B12. Refs resolvidas e merge
da escrita não qualificam os comportamentos nem encerram os programas parciais.

## Milestone: batch-five-authoring
- after: batch-four-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-5-authoring

**Exit gate:** um delta draft de B11 e planos de continuidade nas duas specs
existentes de B11/B29; aceites, dependências, ownership e baselines conferidos.

## Milestone: semantic-closeout-and-update-fidelity
- after: batch-five-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:governance-efficiency/amendment-history-preclose, xref:proj.pose-dist/milestone:governance-efficiency/terminal-closeout-verification, xref:proj.pose-dist/milestone:governance-efficiency/update-dry-run-qualification

**Exit gate:** consumir resultados de amendments, verificação terminal e
fidelidade de update no responsável, sem duplicar ownership. Escrita e merge
não qualificam implementações nem mudam status de specs reutilizadas.

## Milestone: batch-six-authoring
- after: batch-five-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-6-authoring

**Exit gate:** B28/B44/B38 escritos em draft e responsável de B38 criado no recorte
correto; fontes, aceites, dependências e ownership conferidos.

## Milestone: distributed-instructions
- after: spec:pose-adaptive-assessment-freshness, spec:pose-v7-legacy-cleanup-plan
- target_start:
- target_due:
- specs: pose-distributed-instructions-parity

**Exit gate:** B28 alinha freshness em root/scaffold/pt-BR; suggest divulga
availability/extension-needed e migrate explica apply indisponível sem promessa.

## Milestone: planning-release-narrative
- after: spec:pose-delegated-review-contract, spec:pose-v7-legacy-cleanup-plan
- target_start:
- target_due:
- specs: pose-planning-and-release-narrative-reconciliation

**Exit gate:** B44 preserva história 6.x/7.0/ADRs aceitos e publica reconciliação
atual de entrega/adoção/eficácia/planejamento com fontes, sem capabilities ausentes.
Este delta documental é distinto da escrita administrativa do lote.

## Milestone: release-snapshot-integrity
- after: batch-six-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:pose-release-sufficiency-and-recovery/manifest-tag-integrity

**Exit gate:** consumir integridade do snapshot/tag de B38 no responsável. O
coordenador possui B28/B44 e as specs de escrita; não duplica a ownership de B38.
Resultados resolvidos ou merge de docs não encerram programas parciais.

## Milestone: batch-seven-authoring
- after: batch-six-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-7-authoring

**Exit gate:** B14/B15/B32 escritos em draft e roadmap de runtime criado apenas
no recorte B32; aceites/dependências/baselines/ownership conferidos.

## Milestone: publication-generation-and-performance
- after: batch-seven-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:pose-state-consistency-and-recovery/writer-publication, xref:proj.pose-dist/milestone:pose-state-consistency-and-recovery/index-generation, xref:proj.pose-dist/milestone:pose-shared-runtime/operation-performance

**Exit gate:** consumir coordenação/publicação, geração e performance nos
responsáveis sem duplicar ownership. B14 exige B12/B13, B15 exige B14 e B32 é
independente. Escrita e merge não qualificam os comportamentos ou daemon futuro.

## Milestone: batch-eight-authoring
- after: batch-seven-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-8-authoring

**Exit gate:** B16 reutilizado sem alteração de R1–R4/status e B22/B23 escritos
em draft; extensão prospectiva, aceites, DAG, digests e ownership conferidos.

## Milestone: ledger-and-residual-integrity
- after: batch-eight-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:delegated-review/attempt-ledger, xref:proj.pose-dist/milestone:governance-efficiency/residual-material-identity, xref:proj.pose-dist/milestone:governance-efficiency/followup-metadata-routing

**Exit gate:** consumir ledger no responsável e identidade/metadata/roteamento
em governance-efficiency sem duplicar ownership. B16 exige B06/B14, B22 exige B06
e B23 exige B06/B22. Merge/escrita não certificam cota, integridade ou resolução;
programas parciais e extensão contratual de B16 continuam sujeitos a qualificação.

## Milestone: batch-nine-authoring
- after: batch-eight-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-9-authoring

**Exit gate:** B34/B36 escritos em draft, com aceites, closure/cursor prospectivos,
DAG, digests e ownership conferidos; não inferir implementação ou residência.

## Milestone: coherent-reading-and-queries
- after: batch-nine-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:pose-state-consistency-and-recovery/governance-read-view, xref:proj.pose-dist/milestone:pose-state-consistency-and-recovery/generation-query-cursors

**Exit gate:** consumir os resultados de read view e consultas no responsável,
sem duplicar ownership. B34 exige B06/B07/B14/B15 e B36 exige B34; materialização,
merge e refs resolvidas não qualificam coerência, frescor, cobertura ou custo de queries.
Gates permanecem completos no recorte e revalidam autorização atual.

## Milestone: batch-ten-authoring
- after: batch-nine-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-10-authoring

**Exit gate:** B18 draft e planos de continuidade de B17/B19/B21 nas duas specs
existentes; R-IDs/bytes/status, aceites/DAG/fases e ownership conferidos.

## Milestone: delegated-trust-and-capability
- after: batch-ten-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:delegated-review/trust, xref:proj.pose-dist/milestone:delegated-review/adapter-containment, xref:proj.pose-dist/milestone:delegated-review/adoption

**Exit gate:** consumir prova/containment/capability no responsável sem duplicar
ownership. B17 exige B01/B04/B16, B18 exige B07/B12, B19 exige B02/B16/B17/B18;
B21 depende de B19/B20 e conserva qualificação posterior ao fulfillment do lote 11.
A spec compartilhada exige fase B19 verificável antes de B20 e closeout integrado
somente em B21; merge/escrita não qualificam qualquer kind, adoção ou runner remoto.

## Milestone: batch-eleven-authoring
- after: batch-ten-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-11-authoring

**Exit gate:** B20 reutilizado com R1–R3/status intactos e B27/B24 escritos em draft;
aceites, fontes/ADRs, dependências/fases, digests e ownership conferidos.

## Milestone: fulfillment-and-proportional-residual
- after: batch-eleven-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:delegated-review/fulfillment, xref:proj.pose-dist/milestone:governance-efficiency/proportional-next-action, xref:proj.pose-dist/milestone:governance-efficiency/residual-nature-commitment

**Exit gate:** consumir fulfillment, próxima ação e dimensões do residual nos
responsáveis, sem copiar ownership. B20 exige B08/B09/B10 e fase B19; B27 exige
B08/B10/B12 e B24 exige B22/B23. B21 qualifica capability integrada após B20.
Merge/escrita não entregam gate de fase, runner, triagem ou autorização nova.

## Milestone: batch-twelve-authoring
- after: batch-eleven-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-12-authoring

**Exit gate:** B25/B26/B37 escritos em draft, fontes/aceites/ADRs, digests,
contagens, dependências e ownership conferidos; nenhuma execução funcional.

## Milestone: triage-metrics-and-cut-sufficiency
- after: batch-twelve-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:governance-efficiency/residual-triage-risk, xref:proj.pose-dist/milestone:governance-efficiency/condition-work-metrics, xref:proj.pose-dist/milestone:pose-release-sufficiency-and-recovery/cut-sufficiency

**Exit gate:** consumir os três resultados nos responsáveis. B25 exige B01/B14/B24,
B26 exige B10/B24 e B37 exige B01/B02/B08/B30, sem dependências temáticas artificiais.
Candidato não é decisão, resposta não é satisfação e path não é approval.
Merge documental não aceita risco nem qualifica métricas ou prepara release.

## Milestone: batch-thirteen-authoring
- after: batch-twelve-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-13-authoring

**Exit gate:** B39/B41/B40 escritos em draft, ADR suplementar Proposed e nota de
revisão, coverage/digests/DAG/ownership conferidos. Nenhuma operação de release.

## Milestone: publication-recovery-and-selected-cut
- after: batch-thirteen-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:pose-release-sufficiency-and-recovery/publication-evidence-replay, xref:proj.pose-dist/milestone:pose-release-sufficiency-and-recovery/prepare-record-recovery, xref:proj.pose-dist/milestone:pose-release-sufficiency-and-recovery/scope-selection-exclusivity

**Exit gate:** consumir resultados do roadmap responsável. B39 exige B04/B38;
B41 exige B12/B13/B14/B38; B40 integra ambos após B23/B24/B37/B38/B39/B41.
ADR Proposed exige resolução antes de start; não há ciclo B39/B41, estado
verified fabricado, recovery por backfill ou encerramento por escrita/merge.

## Milestone: batch-fourteen-authoring
- after: batch-thirteen-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-14-authoring

**Exit gate:** B31/B33/B35 escritos em draft, ADR Proposed e nota de revisão,
coverage/digests/DAG/ownership conferidos. Nenhum runtime ou benchmark funcional.

## Milestone: common-domain-caches-and-runtime
- after: batch-fourteen-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:pose-shared-runtime/common-domain-handoff, xref:proj.pose-dist/milestone:pose-shared-runtime/bounded-graph-caches, xref:proj.pose-dist/milestone:pose-shared-runtime/optional-worktree-runtime

**Exit gate:** consumir resultados do roadmap responsável. B31 exige B12/B14/B34;
B33 exige B15/B34; B35 exige B07/B12/B14/B15/B31/B32/B33/B34. Fonte única,
isolamento/generation/authority e QoS têm prova; stdio não é IPC ou grant.
Resolver ADR antes de start e preservar fronteira Harness, sem adoção por merge.

## Milestone: batch-fifteen-authoring
- after: batch-fourteen-authoring
- target_start:
- target_due:
- specs: pose-consolidation-batch-15-authoring

**Exit gate:** B42 escrito em draft com oito percursos, trilhas pessoa/agente,
baseline/métricas/stop-go/rollback e dependencies/coverage conferidos. Escrita
materializa todas as propostas obrigatórias; implementações e B43 condicional
continuam pendentes, sem closeout nativo implícito pelo merge.

## Milestone: integrated-pilot-qualification
- after: batch-fifteen-authoring
- target_start:
- target_due:
- specs:
- consumes: xref:proj.pose-dist/milestone:governance-efficiency/integrated-agency-runtime-pilot

**Exit gate:** consumir a qualificação integrada do roadmap proprietário, com
seis pré-requisitos técnicos e B31/B33/B35 finais aceitos. Pessoa real e agente
têm evidência separada nos oito percursos; prova de utilidade/compatibilidade,
coverage, stop/go e rollback não pode ser substituída pelo pipeline documental.

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

A tabela resume resultados cujo aceite permanece pendente; as etapas obrigatórias
já possuem milestones e referências materializadas nos lotes 0–15. B43 permanece
condicional e ausente: adjudicar benefício/prova antes de criar sua spec/milestone,
ou registrar adiamento explícito. Não encerrar o programa só pela materialização.

## Ativação e encerramento

- A retomada explícita de 2026-10-14 autoriza implementação incremental, uma spec por PR e espera de merge.
- B04 é o lote atual pela primeira camada/DAG e cadeia preferencial; consultar o [relatório](../reports/2026-10-14-pose-implementation-b04.md).
- #176 foi mergeado na main d55b8d41; a retomada qualifica B04, com validação nativa strict 129/129 e preparação de evidências/revisão. Different-actor, aceite act-fa20ae0029b6ad14 e closeout continuam pendentes; B01 não foi iniciado.
- Separar implementação validada de review/closeout; não promover a entrega pelo merge dos lotes documentais.
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
