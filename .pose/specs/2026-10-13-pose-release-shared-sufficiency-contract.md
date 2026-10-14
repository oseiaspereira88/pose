---
slug: pose-release-shared-sufficiency-contract
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-action-answers-reverify-on-read, pose-review-disclosure-requires-proof, pose-review-approval-domain-preflight, pose-action-target-and-node-scoping, pose-spec-facts-evidence-binding, pose-release-lifecycle-closure
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:release-shared-sufficiency-contract
---

# Spec: Compartilhar suficiência do cut e critérios tipados entre plan e apply

## 1. Intent

### Goal

Avaliar o mesmo recorte de release e evidência pertinente antes do freeze, diagnosticando critérios não reconhecidos e caminhos que não provam approval.

### Business value

B37 / D08, F06-01, F06-02 e F06-03. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de release](../reports/2026-10-12-pose-front-06-release-sufficiency.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Investigar compromisso de escopo e blockers derivados nos producers existentes e formalizar contrato compartilhado de suficiência. Plan e apply devem avaliar a mesma seleção, milestones/fragments/targets, profiles/criteria, policy/autoridade e source bindings; apply revalida inputs correntes antes de freeze, sem confiar em preview stale.
- R2: P01 deve expor blockers pertinentes de spec e escopo selecionado antes de freeze, com fonte, fase, causa, condição e coverage tipadas. Nenhum roadmap alheio cria bloqueio global e nenhum follow-up opcional se torna obrigatório pelo nome; compromisso vigente não pode desaparecer por simples exclusão silenciosa.
- R3: P13 deve recusar claim de delivery quando falta prova exigida pelo profile e provenance atuais para o target comprometido. Status done, path/entrypoint declarado, marcador satisfied ou artefato existente isoladamente não provam entrega. Consumir evidência adequada e bindings de B30/B02 sem gerar julgamento novo pelo facts reader.
- R4: P14 deve resolver refs apenas com evaluator conhecido e resultado adequado ao critério. Ref reconhecida lexicalmente mas sem evaluator, incluindo evidence:e2e quando sem avaliação disponível, é unsupported/unknown ou inválida conforme contrato, nunca sucesso; requisito obrigatório permanece não satisfeito com diagnóstico explicável.
- R5: P02 deve distinguir ausência intencional de critérios de parse incompleto/heading ou intenção não reconhecidos. Oferecer sintaxe canônica no template (## Cut criteria) e denunciar heading alternativo como ## Release cut criteria quando exprime intenção não lida; não aceitar zero critérios silenciosamente nem reescrever roadmaps históricos.
- R6: P03 deve distinguir resolução de manual-review:path de approval/judgment observado. Arquivo existente ou vazio prova no máximo path resolvido; condição de aprovação exige decisão, authority/assurance e binding adequados de B01/B02. Não converter texto/identidade declarada em confirmação humana observada.
- R7: Reusar evaluators de artifact/delivery/surface, closeout, milestone/federated acceptance e ActionRequest, com estados met/unmet/waived válido/not-applicable/unknown e coverage explícita. Compartilhar sem duplicar engine de gates; caro check externo consome resultado corrente adequado, sem rerun universal em cada query ou waiver cosmético.
- R8: Deferral continua não afirmando entrega e deve explicitar parcela/compromisso/limitação e decisão pertinente. Preservar ADRs de release, histórico selado, tags/manifests e distinção prepared/tagged/published/verified; checker de publicação observada não deve ocultar fato externo porque autorização prévia estava insuficiente.
- R9: Qualificar P01/P02/P03/P13/P14 com negativos e controles positivos em plan/apply CLI/MCP e compose/federated scopes. Decidir compatibilidade de descoberta/refs/contract version e eventual extensão ADR antes de start; lote documental não faz prepare/freeze/tag/publication/deferral/adoption nem fecha roadmap.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-12-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-release-shared-sufficiency-contract.md
- created: .pose/starts/pose-release-shared-sufficiency-contract.json
- created: .pose/changelogs/unreleased/pose-release-shared-sufficiency-contract.md
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/pose/release_lifecycle_test.go
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/pose/delivery_surface_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/roadmap_check_gate_test.go
- modified: .pose/templates/roadmap.md
- created: pose-mcp/internal/pose/release_sufficiency_test.go
- created: pose-mcp/internal/cli/release_sufficiency_test.go
- created: pose-mcp/internal/mcpserver/release_sufficiency_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:release-shared-sufficiency-contract module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/release_lifecycle.go

### Approach and execution gates

Executar após B01/B02/B08/B30 e reusar lifecycle done. Inventariar funções de release plan/prepare, evaluateCutCriteria/discovery, delivery proof, approval e federated acceptance; reconciliar todos handlers/template distribuídos/locales e actions antes do start. Reusar ADRs de release (manifest imutável/evidence append-only) e obligations; arquitetura exata de contrato/evaluator e schema/compatibilidade ainda prospectivos, exigindo ADR adicional somente para decisão estrutural não coberta antes do código. B38/B39/B40/B41 permanecem deltas distintos; não dependência artificial B38, não gate global nem nova engine duplicada.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8, R9
- Decision: Compartilhar avaliação de suficiência corrente e escopo selecionado, preservando semântica de prova, autoridade e história.
- Rationale: Associação e localização não provam intenção, compromisso ou resolução; preservar incerteza evita certificar ausência de trabalho sem evidência.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P01: spec/escopo bloqueado selecionado e roadmap alheio | Plan/apply recusam freeze pertinente; recorte alheio não bloqueia globalmente. |
| V2 — Preview adequado e inputs/profile/policy mudados antes de apply | Revalidação detecta stale e divulga diagnóstico/coverage. |
| V3 — P13: done/path/satisfied sem prova exigida e evidência adequada | Negativos recusados; positivo com binding/provenance atual passa. |
| V4 — P14: ref sem evaluator/unknown e evaluator suportado | Não avaliado não passa; controle adequado satisfaz critério. |
| V5 — P02: heading alternativo, canônico e ausência intencional | Intenção não reconhecida denunciada; ausência distinguível de parse incompleto. |
| V6 — P03: arquivo vazio versus aprovação adequada/stale | Path resolvido não approval; prova/authority pertinente exigida. |
| V7 — Deferral, waiver e histórico publicado com insuficiência prévia | Limites preservados; nenhum claim de delivery ou reescrita/ocultação de fato. |
| V8 — CLI/MCP, milestone/federation e scope parcial | Mesmo contrato/evaluators/inputs; nenhuma duplicação ou bloqueio alheio. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V2, V8 |
| R2 | V1, V8 |
| R3 | V3 |
| R4 | V4 |
| R5 | V5 |
| R6 | V6 |
| R7 | V4, V7, V8 |
| R8 | V7 |
| R9 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 12. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-12-authoring.md).

### Residual risks

Descoberta anteriormente silenciosa pode exigir diagnóstico novo; classificar compatibilidade antes do código. Refs sem evaluator e prova stale não podem virar met. Suficiência prévia e verificação de publicação efetivamente observada são tarefas distintas, sem reclassificar releases antigas.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
