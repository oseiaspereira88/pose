---
slug: pose-consolidation-batch-8-authoring
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on:
priority: 0
components: .pose
task_type: feature
surface: minimal
delivers:
---

# Spec: Escrever e validar o lote documental 8

## 1. Intent

### Goal

Formalizar continuidade de B16 na spec existente e criar somente os deltas de B22/B23, após merge do PR #166.

### Business value

Atribuir esta escrita separadamente da execução funcional das três unidades; continuar a sequência formalizada sem duplicar ledger ou saturar os lotes futuros.

### Constraints

Preservar byte a byte a spec reutilizada de B16, R1–R4, status e registros existentes. Preservar catálogo, aceites, DAG e oito specs reutilizadas. Não executar motor, testes funcionais dos deltas, policy, adoção, dispositions, releases ou closeouts.

## 2. Requirements

- R1: O lote deve formalizar o plano prospectivo de B16 na spec existente e materializar somente B22/B23 em draft, com aceites, artefatos, compatibilidade e validação planejados.
- R2: A matriz deve preservar 96 candidatos/44 unidades/17 lotes, distinguir materialização de entrega, registrar merge do lote 7 e manter lotes 9–16 sem novos artefatos.
- R3: Os roadmaps devem manter ownership única, separar ledger de proveniência e consumir resultados nos responsáveis sem fechar programas parciais.
- R4: O plano deve preservar B16 após B06/B14 e dispatch, B22 após B06 e B23 após B06/B22; as extensões ausentes de B16 devem ser explicitamente reconciliadas antes de seu start.
- R5: Os onze paths documentais devem passar conferência reproduzível, lint-spec, artifact-check e check --strict; roadmap-check deve recusar encerramento dos resultados pendentes sem alegar qualificação funcional.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-13-pose-consolidation-batch-8-authoring.md
- created: .pose/starts/pose-consolidation-batch-8-authoring.json
- created: .pose/changelogs/unreleased/pose-consolidation-batch-8-authoring.md
- created: .pose/specs/2026-10-13-pose-residual-material-coverage-and-identity.md
- created: .pose/specs/2026-10-13-pose-followup-metadata-and-routing-integrity.md
- modified: .pose/roadmaps/delegated-review.md
- modified: .pose/roadmaps/governance-efficiency.md
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/reports/2026-10-11-pose-analysis-index.md
- created: .pose/reports/2026-10-13-pose-batch-8-authoring.md

### Delivery targets

Nenhum target funcional desta escrita. Os targets nas specs novas são prospectivos.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5
- Decision: Preservar o contrato existente de B16 e formalizar sua extensão prospectiva no relatório, sob atribuição documental separada.
- Rationale: Reuso não exige duplicação ou alteração retrospectiva de R-IDs; cota/reserva/idempotência precisam de requisitos explícitos antes da execução.

## 6. Validation

### Strategy

Conferir digests/status, catálogo, aceites, DAG, ownership, refs e links. Executar lint-spec nas duas novas specs, na reutilizada e nesta spec administrativa; artifact-check da escrita e check --strict. Conferir recusa esperada de roadmap-check nos responsáveis/coordenador. CI/Security qualificam o head documental, sem comprovar requisitos funcionais futuros.

### Requirement trace

Vincular R1–R5 aos resultados executados no relatório. Manter in-progress até revisão e closeout nativo.

## 7. Final Report

### Delivered scope

Plano de continuidade, dois deltas draft, matriz e roadmaps escritos; conferência reproduzível, quatro lints, artifact-check e check --strict passaram, com avisos registrados no relatório. Conferir o head de CI no PR; implementação, revisão e closeout permanecem pendentes.

### Residual risks

Contrato ampliado e artefatos de B16 exigem reconciliação antes do start. Identidade/migração e semântica pública de cobertura/metadata exigem qualificação prospectiva. Merge não fecha specs nem unidades.

### Follow-ups

Manter execução e lotes futuros na matriz; nenhuma disposition aplicada nesta escrita.
