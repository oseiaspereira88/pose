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
Readiness/start/assessment executados, R1–R8 preservados. Evidência funcional é
registrada após os testes; review different-actor e closeout nativo são gates
pendentes. Não declarar temporal-trust encerrado nem liberar verified-proof
somente pela presença do código ou pelo merge do PR.

## Milestone: verified-proof
- after: temporal-trust
- target_start:
- target_due:
- specs: pose-action-answers-reverify-on-read, pose-review-disclosure-requires-proof

**Exit gate:** qualificar as correções mergeadas de #157/#158 contra o B01 completo,
o contexto temporal de B04 e os consumidores; fechar as duas specs pelos gates.
Merge e assinatura não substituem qualificação ou julgamento semântico.

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
