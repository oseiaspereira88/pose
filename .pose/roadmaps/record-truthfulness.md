---
slug: record-truthfulness
status: active
created_at: 2026-10-04
depends_on:
---

# Roadmap: Record truthfulness (6.3.x)

**Program:** POSE agency and readiness, wave 0 of 4.
**Outcome:** POSE's records no longer imply more than the engine observed — adoption cutoffs, publication, review assurance, attribution and `blocked` mean what they say — and the existing backlog is reconciled before new features repeat delivered work.

Source: [third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md)
(backlog [JSON](../reports/2026-10-03-pose-consolidated-backlog.json), items POSE-01 to POSE-07)
and the transversal [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md).
This roadmap plans work; it declares no capability delivered.

Compatible fixes and diagnostics fit the 6.3.x maintenance line. Outputs whose
meaning changes (public-claims fields, attribution contract, metric v2) are
versioned and may ship in the next minor. Publication of the prepared 6.3.0
candidate stays under the existing release lifecycle and is not part of this roadmap.

## Milestone: compatibility-truth
- after:
- target_start:
- target_due:
- specs: pose-legacy-contract-cutoffs, pose-public-claims-publication-provenance, pose-flat-spec-amendments

**Exit gate:** every registered contract's legacy cutoff is consumed or reported; candidate, prepared and published versions are distinct; flat specs amend like folder specs.

## Milestone: review-truth
- after:
- target_start:
- target_due:
- specs: pose-review-assurance-disclosure, pose-review-attribution-roles

**Exit gate:** the 6.3.0 `human:oseias` attestations no longer read as human review; declared and verified separation render differently; attribution roles are prospective and legacy records keep their digests.

## Milestone: lifecycle-truth
- after: review-truth
- target_start:
- target_due:
- specs: pose-blocked-semantics-alignment, pose-open-backlog-reconciliation

**Exit gate:** `blocked` has one non-terminal meaning everywhere with transfer protection intact and metric v1 unchanged; every remaining requirement of the non-terminal specs is classified and only evidence-backed dispositions are applied.

## Continuação prospectiva — programa de consolidação, lote 1

Preservar os milestones e a baseline 6.3.x acima como programa histórico.
Os deltas abaixo pertencem à consolidação atual; não renomear o cut histórico
nem anunciar uma nova versão. Consultar a [matriz](../results/pose-implementation-plan.json)
e o [relatório do lote 1](../reports/2026-10-13-pose-batch-1-authoring.md).
B04 iniciou implementação sob autorização explícita em 2026-10-14. Os demais
members novos permanecem draft; B01 conserva código mergeado e closeout pendente.

## Milestone: temporal-trust
- after: spec:pose-material-equivalence-reuse, spec:pose-native-attestation-issuer
- target_start:
- target_due:
- specs: pose-historical-trust-and-review-reuse

**Exit gate:** B04 distingue validade temporal histórica, confiança atual e base
temporal; reuso comprova a fonte ou declara reassunção atual, com orçamento de
travessia. Claimed timestamp não inventa autorização histórica. Preservar #157.

Execução atual: [relatório B04](../reports/2026-10-14-pose-implementation-b04.md).
Readiness/start/assessment executados, R1–R8 preservados; qualificação global
strict 132/132. O maintainer adotou expressamente as conclusões em 2026-10-14 e
action act-fa20ae0029b6ad14 está satisfeita. A atribuição é adoção humana de
conclusões preparadas pelo agente, com identidade/separação declaradas. O status
nativo da spec e seu relatório registram o resultado do closeout; consumir esse
resultado, sem inferir entrega pelo merge ou pelo código. Atualização 2026-10-15:
#178 publicou esse closeout; B01 iniciou a qualificação no PR #179.

## Milestone: verified-proof
- after: temporal-trust
- target_start:
- target_due:
- specs: pose-action-answers-reverify-on-read, pose-review-disclosure-requires-proof

**Exit gate:** qualificar as correções mergeadas de #157/#158 contra o B01 completo,
o contexto temporal de B04 e os consumidores; fechar as duas specs pelos gates.
Merge e assinatura não substituem qualificação ou julgamento semântico.

Retomada em 2026-10-14: #176 mergeado (d55b8d41); B04 passou qualificação nativa
strict 129/129 na main e 132/132 global. O aceite do maintainer foi registrado
em act-fa20ae0029b6ad14. Consumir o status nativo de temporal-trust e o relatório
B04. Atualização 2026-10-15: a spec action-answer de verified-proof está em
qualificação no PR #179; disclosure ainda aguarda seu próprio PR.

## Milestone: approval-preflight
- after: verified-proof
- target_start:
- target_due:
- specs: pose-review-approval-domain-preflight

**Exit gate:** nova approval inválida grava zero artefatos de review em Store,
CLI, import e MCP; negativa legítima e aprovação válida mantêm seus contratos.

## Milestone: policy-scopes
- after: spec:pose-effective-governance-projection
- target_start:
- target_due:
- specs: pose-policy-scope-validation

**Exit gate:** scope desconhecido ou inválido não aparece applicable/effective;
domínio, preparação, projeção, CLI e MCP compartilham a taxonomia admitida.
B05 é uma raiz independente; não depende artificialmente de concluir B04/B01/B02.

## Milestone: presence-channel-parity
- after: verified-proof
- target_start:
- target_due:
- specs: pose-proof-presence-channel-parity

**Exit gate:** B03 declara escopo de presença e prova admitida; SSH/SK/issuer
obedecem ao mesmo contrato de canal, sem dispensa silenciosa ou touch fabricado.
Consultar o [lote 2](../reports/2026-10-13-pose-batch-2-authoring.md).

## Risk controls

- Correct the effective data; never disable a gate to make history pass.
- Sealed bundles keep their stamped contracts; no retroactive governance.
- No auto-close from code existence; external operations are not performed by the program.

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

Revisão independente de #179: os oito julgamentos de action-answer foram
avaliados por `agent:independent-codex-pr179` (separação declarada, sem adoção
humana). Consultar o relatório e os resultados nativos `b01-action-*` para
attestation/verify. O maintainer aceitou explicitamente as conclusões em 2026-10-15; a ação
`act-3ee9344e0652cc60` está answered/satisfied (revision 2, declared). A spec
`pose-action-answers-reverify-on-read` foi encerrada nativamente em 2026-10-15,
com verify/review-check aprovados e admission closeout-check terminal=true/sem
blockers. Os IDs e o resultado pós-publicação estão em
`b01-action-review-verification.json` e `b01-action-closeout-check.json`. B01 continua parcial/delivery_complete
false e B02 aguardando a qualificação e o closeout da spec irmã.


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
