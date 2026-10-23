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
- B04 foi o lote inicial pela primeira camada/DAG; B01 encerra as duas specs em #179/#182. B02 aguarda o merge de #182; consultar os relatórios de qualificação e resultados nativos abaixo.
- #176 foi mergeado na main d55b8d41; B04 passou validação nativa strict 129/129 e global 132/132. O maintainer adotou as conclusões propostas e act-fa20ae0029b6ad14 está satisfeita, com identidade/separação declaradas. Consumir status nativo da spec e relatório B04 para o resultado do closeout. Atualização 2026-10-15: #178 publicou esse closeout; B01 está em qualificação no PR #179.
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

## Histórico da reconciliação de #179 — 2026-10-15

#178 mergeado na main a5016ce5 publica o closeout nativo de B04 e encerra a
espera do complemento. B01 é o próximo lote pelo DAG, com código parcial
mergeado em #157/#158. O PR #179 qualificou somente
`pose-action-answers-reverify-on-read`; preservar start/baseline e R1–R5/D1.
Consultar o [relatório de qualificação](../reports/2026-10-15-pose-b01-action-qualification.md).
A spec de disclosure permanece in-progress, para um PR próprio após este merge.
B01 continua delivery_complete false até qualificação e fechamento das duas
specs. B02 aguarda B01. Closeouts administrativos e B43 condicional continuam
separados; nenhuma policy/release é adotada.

Revisão de #179: o parecer independente de action-answer foi concluído por
`agent:independent-codex-pr179`, com separação declarada e sem confirmação
humana. O relatório e os resultados nativos `b01-action-*` vinculam a attestation
ao sujeito revalidado. O maintainer aceitou explicitamente em 2026-10-15 e a ação
`act-3ee9344e0652cc60` está answered/satisfied (revision 2, declared). A spec `pose-action-answers-reverify-on-read` foi encerrada nativamente em
2026-10-15 após verify/review-check aprovados; o admission closeout-check
confirma terminal=true e nenhum blocker. O resultado pós-publicação e seus
IDs estão em `b01-action-closeout-check.json` e `b01-action-review-verification.json`. B01 continua parcial/delivery_complete false;
disclosure e B02 não são iniciados por esta revisão. B04 permanece encerrado.


## Qualificação e closeout de B01 — PR #182, 2026-10-15

A main `1f324461` contém #179 (action-answer done) e #181 (verificação
pós-publicação e diagnóstico Security); B04 permanece encerrado. O PR #182
qualificou a spec irmã `pose-review-disclosure-requires-proof`, preservando
start e R1–R4/D1: native strict global132/132 e componente129/129, regressões
CLI/MCP e race, revisão independente e closeout nativo terminal=true/sem blockers.
Consultar o [relatório](../reports/2026-10-15-pose-b01-disclosure-qualification.md),
o [parecer](../reports/2026-10-15-pose-b01-disclosure-review.md) e os resultados
nativos `b01-disclosure-*` para admissão e verificação publicada atual.

Ambas as specs de B01 estão done e a matriz registra delivery_complete=true.
Esse resultado dos membros não encerra o roadmap/programa inteiro nem substitui
seus gates integrados. B02 (`pose-review-approval-domain-preflight`) é a próxima
unidade da cadeia preferencial após o merge de #182; B03 também exige B01,
enquanto B05/B06 e outras raízes continuam independentes. Nenhuma nova spec,
policy, release ou B43 é iniciada neste PR; closeouts administrativos continuam
separados de entrega funcional. CI e Security devem passar no head final.

B02 (`pose-review-approval-domain-preflight`) atingiu closeout nativo em
2026-10-16 UTC no PR #183: validação independente global strict 133/133 e módulo
130/130 no head e33496b1, F1–F4 resolvidos, review different-actor Codex/OpenAI
aprovada e close terminal. Bundle `rvb-045390915bbf6068`, atestação `rva-98ebc45a80704275`;
identidade/separação declaradas, sem assinatura ou confirmação humana. O
[parecer, Fourth round](../reports/2026-10-16-pose-b02-domain-preflight-review.md)
e os snapshots b02-domain-preflight-* registram os gates. A matriz agora marca
B02 delivery_complete=true. O programa/roadmap continua ativo: aguardar merge
de #183 pelo maintainer e depois reavaliar DAG/readiness nativa, sem escolher
uma nova implementação, policy, capability ou adoção neste closeout.

B12 (`pose-cli-mcp-operation-results`) atingiu closeout nativo em
2026-10-16 no PR #186. O resultado tipado, os parsers estritos, o erro MCP
inconclusivo e o journal foram qualificados por revisão different-actor
Codex/OpenAI frente a Claude/Anthropic: F1–F13 fechados, 26 contraprovas,
15 probes com controles preservados, vet, suíte completa, race e quickstart.
Validação de implementação no head 16151559: global strict 134/134 e módulo
131/131. Identidade e separação declaradas, sem assinatura ou confirmação humana.
O [parecer, Eighth round](../reports/2026-10-16-pose-b12-operation-results-review.md)
e a [evidência](../results/b12-independent-review.json) identificam os selos,
atestações e gates atuais. A matriz registra B12 delivery_complete=true;
o programa e este roadmap continuam ativos. Aguardar merge de #186 pelo
maintainer e reavaliar DAG/readiness nativa, sem selecionar nova implementação,
policy, capability ou adoção neste closeout.

B13 (`pose-journal-checkpoint-and-lock-recovery`) atingiu closeout nativo em
2026-10-16 no PR #187. Journal integral, checkpoint por efeitos observados e
recovery de trava por dono morto foram qualificados por revisão different-actor
Codex/OpenAI frente a Claude/Anthropic: F1–F4 resolvidos, 31 contraprovas por
asserção, P01–P17 com controles preservados, vet, suíte completa, race, builds
Windows/Darwin e jornada quickstart. Validação de implementação no head
8e8b1307: global strict 135/135 e módulo 132/132. Identidade e separação
declaradas, sem assinatura ou confirmação humana das conclusões.
O [parecer, Second round](../reports/2026-10-16-pose-b13-journal-checkpoint-lock-recovery-review.md)
e a [evidência](../results/b13-independent-review.json) identificam o bundle
`rvb-bbed7a49533bf1e3`, a atestação `rva-dfa2006f7d36184d` e os gates nativos. A matriz registra B13
delivery_complete=true; o programa continua ativo e este roadmap conserva seu
status. Aguardar merge de #187 pelo maintainer e reavaliar DAG/readiness nativa,
sem selecionar implementação, policy, capability ou adoção neste closeout.

B14 (`pose-writer-coordination-and-atomic-publication`) atingiu closeout
nativo em 2026-10-16 no PR #188. Publisher comum com staging exclusivo e
precondição explícita, trava de ciclo de vida compartilhada por start, cancel,
close, transfer e backfill, close vinculado aos bytes julgados e à autorização
revalidada dentro da trava, e state refresh com resultado parcial foram
qualificados em duas rodadas de revisão de ator diferente apenas por execução:
mesmo vendor e modelo do implementador (Claude Opus 5.5, Anthropic), em sessão
separada, não revisão de outro vendor. F1–F11 resolvidos ou declarados, 25/25
mutações versionadas reproduzidas e contraprovas independentes, P01–P04 iguais
ao registro, vet, suíte completa, race e builds Windows/Darwin. Validação no
head 00f743c3: global strict 136/136 e módulo 133/133. Quatro riscos baixos
foram aceitos por finding, com dono e data de revisão. Identidade declarada,
sem assinatura.
O [parecer, Second round](../reports/2026-10-16-pose-b14-writer-coordination-review.md)
e a [evidência](../results/b14-independent-review.json) identificam o bundle
`rvb-b3ff5896947dc94f`, a atestação `rva-7a84aea174407a89` e os gates nativos. A matriz registra B14
delivery_complete=true; o programa continua ativo e este roadmap conserva seu
status. Aguardar merge de #188 pelo maintainer e reavaliar DAG/readiness nativa,
sem selecionar implementação, policy, capability ou adoção neste closeout.

A correção transversal `pose-test-scripts-remove-their-temp-files`, fora de
B01–B44, atingiu closeout nativo em 2026-10-17 no PR #192. A revisão independente
Codex/OpenAI frente à implementação Claude/Anthropic aprovou a rodada 13:
F8/F9 corrigidos e reproduzidos, R3 reduzido por decisão do maintainer a forma
simples do helper ou pin SHA-256 revisado, e limites de shell/SIGKILL declarados.
Validação no head 534653c6: global strict 137/137 e módulo 134/134; medição Docker
real com pose 7.4.1 passou em 12,486 s sem resíduos no TMPDIR. Identidade e
separação declaradas, sem assinatura.
O [parecer](../reports/2026-10-17-pose-test-scripts-temp-cleanup-review.md) e a
[evidência](../results/test-scripts-temp-cleanup-independent-review.json)
registram o bundle `rvb-6167567bf3fa6ac4`, a atestação
`rva-5358c584e2a8bd4f`, os produtores e gates nativos. Esta entrega não cria
B-unit nem altera o plano de implementação; o programa e este roadmap conservam
seu status. Aguardar merge de #192 pelo maintainer.

B15 (`pose-index-generation-integrity`) foi iniciado nativamente em 2026-10-17
após o merge de #192: os sete arquivos de `pose index` são uma geração
identificada (bytes imutáveis por geração, nomes legados publicados dos mesmos
bytes e um descritor com o SHA-256 de cada arquivo e os digests dos inputs,
publicado por último pelo publisher da B14); uma falha antes do commit mantém a
geração anterior e o retry converge para a mesma identidade. Leitores da CLI e
do MCP conferem o conjunto: o que não confere é `unavailable` com motivo, nunca
um resultado vazio ou misto, e um repositório sem descritor continua legível
como `legacy-unversioned`. `pose index --verify` lê sem gravar. Probes P01–P03
com controle inalterado; 54/54 mutações mortas. Nesse registro de implementação a spec ainda estava in-progress,
com delivery_complete false; a revisão e o closeout são registrados abaixo.

### B15 — closeout independente (2026-10-17)

PR #193 conclui B15 após três rodadas de revisão: F01–F16 resolvidos na causa e
nenhum finding novo em ff50b4eb. Revisor `agent:independent-gpt-6.1-sol-review`,
Codex/OpenAI, outro vendor em relação ao implementador Claude/Anthropic; identidade
declarada e atestação não assinada. Validação estrita global 138/138 e explícita
do módulo 135/135; vet, suíte completa, race, builds Windows/Darwin, jornada de
install/upgrade, 54/54 mutações e contrafactuais dos 48 testes novos verificados.
Probes P01–P03 reproduzidos e seis probes independentes das correções finais passaram.

Bundle `rvb-68d1469c0679927b`, atestação `rva-ce3732f9185f01c8`; review verify fresh/approved,
review-check, closeout-check, close nativo, lint-spec e check estritos aprovados.
Surface-check da B15 aprovado com evidência atual; action-mismatch de outra spec
de release foi reproduzido na main 18e4342b. Spec
`pose-index-generation-integrity` done e B15 `delivery_complete: true` no
[plano](../results/pose-implementation-plan.json). Parecer e evidência em
[review B15](../reports/2026-10-17-pose-b15-index-generation-review.md) e
[resultado compacto](../results/b15-independent-review.json).

O programa permanece ativo. Aguardar merge pelo maintainer de #193 e reavaliar
DAG/readiness nativa; este closeout não seleciona outra implementação. Os limites
Git/read view global B34/B35 e recuperação de release B41 permanecem declarados.

B06 (`pose-source-coverage-integrity`) foi iniciado nativamente em 2026-10-17 após
o merge de #193: toda leitura que descartava em silêncio o que não conseguia ler
(spec de follow-ups, obrigação refutada pela validação, arquivo do scan de marcadores,
registro de review) passa a devolver uma cobertura com as perdas (produtor, artefato,
tipo e escopo) e as exclusões declaradas; o Complete deixa de significar "a lista veio
vazia" e passa a significar cobertura do recorte pedido. Backlog, candidatos, estado
do projeto e Attention contam o mesmo conjunto não resolvido, e um registro de review
ilegível que pode ser posterior não deixa a aprovação anterior valer. Probes com
controle inalterado. Nesse registro de implementação, a spec ainda estava in-progress e delivery_complete
false; o closeout independente está registrado abaixo.

### B06 — closeout independente (2026-10-18)

PR #194 conclui B06 após a oitava revisão independente no HEAD 67f25e79.
Revisor `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI, outro vendor em
relação ao implementador Claude/Anthropic; identidade e separação declaradas,
atestação sem envelope no pose-dist. Nenhuma policy foi alterada.

Validação estrita global 139/139 e módulo 136/136; 115/115 mutações mortas por
asserção, 83 testes selecionados com contrafactuais, vet, suíte completa, race,
builds Windows/Darwin, probes front05 e jornada install/upgrade passaram.
Bundle `rvb-d4e49da25f48b9ae`, atestação `rva-eb2a3da043e02b89`; review verify
fresh/approved e close nativo aprovados. Spec done e B06 delivery_complete no
[plano](../results/pose-implementation-plan.json).

R6-F04 e R7-F04 permanecem open na spec, owner @core, low, revisão 2026-12-01;
são limitações diagnósticas conservadoras, sem aprovação indevida demonstrada.
Parecer em [review B06](../reports/2026-10-17-pose-b06-source-coverage-review.md)
e [evidência compacta](../results/b06-independent-review.json). Aguardar merge
pelo maintainer e reavaliar readiness; nenhuma próxima implementação selecionada.


## B03 — execução individual iniciada em 2026-10-23

Managed start de `pose-proof-presence-channel-parity` admitido após B01/B04.
ADR prospectivo escolhe schema 2 e migração explícita; a implementação inicial
4dc8277 inclui regressões de domínio/CLI/MCP. Naquela fase B03 permanecia delivery_complete=false; validação, revisão,
closeout e publicação eram próximos passos, não resultados já obtidos. As fundações,
policy da instância, release e gitlink Harne8 são preservados.


## B03 — fechamento funcional nativo em 2026-10-23

Spec `pose-proof-presence-channel-parity` done após147/147 e revisão real
dos15critérios, bundle `rvb-343f8deb0f9be466`, atestação
`rva-eeaa489d1572aa48`, terminal=true/sem blockers. A matriz marca B03
delivery_complete=true conforme sua closure_policy funcional; publicação na main
é uma etapa separada, pendente do head final e CI/merge do PR #203.
Consultar os relatórios B03 de implementação/revisão. O programa continua ativo;
nenhuma outra unidade, policy da instância ou release é adotada por esse registro.
