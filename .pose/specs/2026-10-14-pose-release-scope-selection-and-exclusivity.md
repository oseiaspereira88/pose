---
slug: pose-release-scope-selection-and-exclusivity
status: draft
created_at: 2026-10-14
completed_at:
depends_on: pose-followup-metadata-and-routing-integrity, pose-residual-nature-commitment-and-effect, pose-release-shared-sufficiency-contract, pose-release-manifest-tag-integrity, pose-release-publication-evidence-replay, pose-release-prepare-record-recovery, pose-release-lifecycle-closure, pose-release-prepare-guards-claimed-fragments
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:release-scope-selection-and-exclusivity
---

# Spec: Selecionar cut coerente e recusar dupla atribuição antes do freeze

## 1. Intent

### Goal

Permitir subconjunto explícito com significado de exclusão/adiamento e recusar fragment assignment conflitante em plan/apply antes de mover a fila.

### Business value

B40 / F06-07 e F06-08. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de release](../reports/2026-10-12-pose-front-06-release-sufficiency.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: P11 deve admitir selecionar subconjunto coerente de specs/milestone/outcomes e fragments, com scope/refs/digests e critérios pertinentes explícitos no plan/manifest. Reusar milestone/federated acceptance e B37; patch de uma spec não exige roadmap novo ou conclusão do programa inteiro.
- R2: Distinguir item fora da intenção do cut, adiamento autorizado de parcela comprometida, outcome não atendido e residual divulgado. Exclusão editorial não dispensa obrigação vigente nem afirma entrega da parcela adiada; nature/commitment/effect de B24 e adequação/sucessores de B23 preservam a intenção. Exceção material usa request/policy e autoridade competentes, sem confirmação universal.
- R3: P10 deve ser recusado em plan e apply antes de qualquer movimento da fila quando a mesma assignment canônica viola exclusividade entre releases. Compartilhar descoberta e semântica com o check global; manifest/archive/pending/aliases e duplicatas da mesma spec não podem ter regras concorrentes. Flag para foreign moved claims não autoriza dupla assignment.
- R4: Leitura incompleta/ilegível ou manifest sem contrato reconhecido deve declarar coverage/gap e impedir concluir ausência de assignment no recorte necessário. Unknown não equivale a available; diagnóstico deve localizar fontes e permitir reconcile sem bloquear todo histórico por item alheio à seleção.
- R5: Plan e apply devem usar seleção/identity/inputs vinculados ao mesmo snapshot e o contrato de suficiência B37; apply revalida seleção, exclusividade, fontes/authority e critérios antes de freeze, sob coordenação B41/B14. Alteração concorrente da fila ou assignment recusa preview stale com efeitos conhecidos, sem mover outro item ou sobrescrever decisão.
- R6: Mover somente os fragments selecionados, preservando unselected/pending e seus vínculos. Manifest/notes/version evidence e recovery inventory contêm exatamente a seleção com refs/digests e limites; B38 verifica snapshot/tag, B39 verifica publicação/replay e B41 garante retomada sem duplicação. Sem promessa de que toda capability do programa ou canal foi entregue.
- R7: Preservar compatibilidade deliberada do fluxo legacy por fila, divulgando seu scope em vez de inferir promessa do produto. Seleção estruturada/schema/refs públicos, assignment identity e migration preview exigem contrato prospectivo e reconciliação dos ADRs antes do código; nunca redefinir manifests, tags ou subjects históricos.
- R8: Qualificar P10/P11 e controles de seleção válida, conjunto incoerente, destino done inadequado, fragment claim alheio, source loss e dois prepares concorrentes. CLI/MCP/produtores usam mesmo domínio; completar handlers/schemas/tests/distribuição antes de start e registrar provas de composição. Este lote não prepara cut, aplica deferral ou publica release.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-13-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-14-pose-release-scope-selection-and-exclusivity.md
- created: .pose/starts/pose-release-scope-selection-and-exclusivity.json
- created: .pose/changelogs/unreleased/pose-release-scope-selection-and-exclusivity.md
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/release_lifecycle_test.go
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/pose/release_archive.go
- modified: pose-mcp/internal/pose/release_claimed_fragments.go
- created: pose-mcp/internal/pose/release_selection_exclusivity_test.go
- created: pose-mcp/internal/cli/release_selection_exclusivity_test.go
- created: pose-mcp/internal/mcpserver/release_selection_exclusivity_test.go
- modified: .pose/adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:release-scope-selection-and-exclusivity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/release_lifecycle.go

### Approach and execution gates

Executar depois de B23/B24/B37/B38/B39/B41, integrando contratos já qualificados. Reusar appendReleasePolicyChecks, archive discovery, ForeignFragmentClaims, milestone/federation e sufficiency; mover descoberta compartilhada ao domínio se necessário, sem engine paralelo. Confrontar regra atual de exclusividade por spec e aliases antes de escolher outra identidade; não admitir segunda atribuição por novo registro de tentativa. Fixar sintaxe de seleção e contrato do manifest com compatibility/migration preview antes de start, conforme [ADR proposto](../adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md) e knowledge:adr-release-temporal-evidence-review. Não copiar requirements inteiros para outro ledger, nem impor policy/gates retrospectivos.

## 5. Decisions

### Decision D1
- Proposal: [Temporal evidence and attempt lineage](../adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md), status Proposed; knowledge:adr-release-temporal-evidence-review.
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Vincular seleção explícita e assignment exclusivo ao mesmo preflight e snapshot, distinguindo exclusão de deferral.
- Rationale: Fila de fragments não demonstra escopo suficiente; seleção, compromisso e exclusividade precisam de vínculos e cobertura antes do writer.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P11: duas pendentes e selecionar subconjunto coerente | Plan/manifest nomeiam seleção e refs; unselected permanece intacto. |
| V2 — Outcome obrigatório não atendido, opcional externo e deferral autorizado | Significados/efeitos próprios; não dispensar compromisso por exclusão. |
| V3 — P10: mesma assignment em duas releases e controle válido | Plan/apply recusam antes de mover; descoberta concorda com check global. |
| V4 — Manifest ilegível, perdas de leitura, aliases e fonte incompleta | Coverage/unknown explícitos; ausência de assignment não inventada. |
| V5 — Preview seguido de mudança da fila/authority ou prepare concorrente | Revalidação/CAS; conflito e efeitos divulgados, sem overwrite. |
| V6 — Foreign fragment claim e escape-hatch de moved claims | Limites da exceção preservados; dupla assignment continua recusada. |
| V7 — CLI/MCP, legacy queue e history/migration preview | Mesmo domínio/scope e compatibility deliberada, sem reescrita de cut. |
| V8 — Freeze selecionado, crash/resume e replay de publicação | Inventory/manifest/notes correspondem; coverage de delivery/canais limitada ao provado. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V2, V7 |
| R2 | V2, V8 |
| R3 | V3, V6 |
| R4 | V4 |
| R5 | V5 |
| R6 | V1, V8 |
| R7 | V4, V7 |
| R8 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 13. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-14-pose-batch-13-authoring.md).

### Residual risks

Seleção é promessa de scope, não certificado automático de outcome. Dupla assignment e source loss exigem preflight antes de mover fila. Compatibilidade de manifest/refs e proposta ADR devem estar resolvidas antes da implementação integrada.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
