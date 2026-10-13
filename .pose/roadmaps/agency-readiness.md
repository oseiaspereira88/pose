---
slug: agency-readiness
status: draft
created_at: 2026-10-04
depends_on:
---

# Roadmap: Agency and explainable readiness (6.4)

**Program:** POSE agency and readiness, waves 1 and 2 of 4.
**Outcome:** one query answers what is still owed, by whom, from which source, restricting which phase of which scope, and what continues meanwhile; a material request to a person survives sessions, is resolved with adequate authority, and releases only the corresponding transition.

Source: [third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md),
items POSE-08 to POSE-19 and POSE-28; decisions D01–D16 in the
[ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md).

The central question the release must answer: *what is the state of this
delivery, which source supports the answer, what still depends on someone,
which phase is restricted and which work remains authorized?*

## Milestone: obligation-contract
- after: spec:pose-blocked-semantics-alignment, spec:pose-review-assurance-disclosure
- target_start:
- target_due:
- specs: pose-obligation-contract, pose-effective-governance-projection, pose-mechanization-adversarial-corpus

**Exit gate:** the ADR is Accepted with a versioned schema; effective governance distinguishes supported, configured, applicable and effective; the adversarial corpus fails against the defective tree.

## Milestone: queryable-obligations
- after: obligation-contract
- target_start:
- target_due:
- specs: pose-typed-producer-diagnostics, pose-obligation-projection, pose-state-attention

**Exit gate:** readiness, review, closeout and start obligations are aggregated read-only with snapshot and per-producer coverage; CLI and MCP return the same IDs for the same snapshot; a failed producer never yields an empty complete answer.

## Milestone: action-requests
- after: queryable-obligations, spec:pose-review-attribution-roles
- target_start:
- target_due:
- specs: pose-action-requests, pose-action-request-resolution

**Exit gate:** a material request survives sessions; wrong role, decline, agent-declared human confirmation, replay, conflict, stale digest and foreign project are refused or recorded without satisfying the request.

## Milestone: phase-readiness-enforcement
- after: action-requests
- target_start:
- target_due:
- specs: pose-phase-scoped-readiness, pose-governed-effect-enforcement, pose-transfer-preserves-obligations

**Exit gate:** the vertical slice of the analysis (section 18) passes end to end through the same domain code in CLI and MCP; enforcement is opt-in and a non-adopting instance is unchanged; interrupted transfer never duplicates authorization.

## Release cut criteria

The 6.4 cut needs the four milestones verified, the legacy `Ready` meaning
preserved across the fixture corpus, and the pilot stop/go from
`governance-efficiency` before any instance adopts enforcement. Adoption is a
separate, explicit decision per instance.

## Continuação prospectiva — lote 2

Preservar o programa histórico 6.4 e seus gates. O delta abaixo qualifica
perdas nas fontes existentes; não adota enforcement nem altera o cut histórico.

## Milestone: source-coverage-integrity
- after: spec:pose-obligation-projection, spec:pose-debt-marker-lexical-precision
- target_start:
- target_due:
- specs: pose-source-coverage-integrity

**Exit gate:** B06 expõe perda de leitura/parse/validação/scan por produtor e
recorte; negativa ilegível não permite aprovação antiga silenciosa; follow-ups
têm paridade CLI/MCP/Attention. Unknown não vira zero complete.
Consultar o [lote 2](../reports/2026-10-13-pose-batch-2-authoring.md).
B08/B09/B10 são materializados abaixo na continuação do lote 3; esta etapa
continua qualificando somente B06.

## Continuação prospectiva — lote 3

Consultar o [lote 3](../reports/2026-10-13-pose-batch-3-authoring.md). Manter os
members abaixo draft e preservar o cut histórico 6.4 e os contratos done.

## Milestone: action-target-scoping
- after: spec:pose-phase-scoped-readiness, spec:pose-governed-effect-enforcement
- target_start:
- target_due:
- specs: pose-action-target-and-node-scoping

**Exit gate:** B08 restringe apenas destino/fase corretos; roadmap não restringe
não-membros; requests de origem terminal continuam visíveis; artifact/kind/ID
resolvem exatamente o conteúdo material. B08 é raiz independente.

## Milestone: action-lifecycle-authority
- after: spec:pose-action-answers-reverify-on-read, spec:pose-review-disclosure-requires-proof, spec:pose-policy-scope-validation
- target_start:
- target_due:
- specs: pose-action-waiver-and-supersession-authority

**Exit gate:** B09 exige autoridade/prova adequadas para waiver e supersession;
terceiro não retira condição alheia; cancellation legítima não apaga obrigação
de outro producer. Qualificar/encerrar B01/B05 antes de executar.

## Milestone: action-recovery-preflight
- after: action-lifecycle-authority
- target_start:
- target_due:
- specs: pose-action-request-recovery-and-preflight

**Exit gate:** B10 apresenta próximo passo suportado para invalidated/subject
ausente/decline; preview read-only revela admissibilidade e prova pendente;
apply revalida e retry da mesma intenção completa supersession uma vez.
Auditoria de refusal deve corresponder a fatos realmente persistidos.

## Risk controls

- Derive, don't duplicate: no resolution journal for derived obligations.
- Unknown is never satisfied, cancelled or "no blocker".
- No top-level command per kind of pending item; no scheduler in the core.
- Every new gate adds its cheapest-formal-satisfaction case to the corpus.
