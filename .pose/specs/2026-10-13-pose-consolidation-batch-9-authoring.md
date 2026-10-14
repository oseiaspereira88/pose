---
slug: pose-consolidation-batch-9-authoring
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

# Spec: Escrever e validar o lote documental 9

## 1. Intent

### Goal

Materializar B34/B36 e seus vínculos após merge do PR #167, sem iniciar requisitos funcionais.

### Business value

Formalizar leitura coerente e consultas ligadas à geração na sequência aprovada; atribuir a escrita separadamente das entregas de implementação.

### Constraints

Preservar catálogo, aceites, DAG, fundações e oito specs reutilizadas byte a byte. Manter os dois deltas draft; nenhuma mudança no motor, policy, adoção, dispositions, releases ou closeouts. Não criar daemon, executar benchmark ou adotar limite de histórico nesta escrita.

## 2. Requirements

- R1: Materializar somente as specs de B34/B36 com requisitos estáveis, aceites, artefatos prospectivos, validação, compatibilidade e gates de execução explícitos.
- R2: Preservar 96 candidatos/44 unidades/17 lotes, registrar merge do lote 8 sem fabricar closeout nativo e manter lotes 10–16 não materializados nesta interação.
- R3: O roadmap de consistência deve possuir B34/B36 em milestones distintos; o coordenador deve consumir os resultados sem ownership duplicada ou fechamento de programa parcial.
- R4: Preservar B34 após B06/B07/B14/B15 e B36 após B34; separar leitura coerente, completude, frescor e autorização atual, sem escolher implicitamente arquitetura do runtime futuro.
- R5: Os dez paths documentais devem passar conferência reproduzível, lint-spec, artifact-check e check --strict; roadmap-check deve recusar encerramento dos resultados ainda pendentes, sem alegar implementação funcional.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-13-pose-consolidation-batch-9-authoring.md
- created: .pose/starts/pose-consolidation-batch-9-authoring.json
- created: .pose/changelogs/unreleased/pose-consolidation-batch-9-authoring.md
- created: .pose/specs/2026-10-13-pose-governance-read-view-generation.md
- created: .pose/specs/2026-10-13-pose-generation-bound-query-cursors.md
- modified: .pose/roadmaps/pose-state-consistency-and-recovery.md
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/reports/2026-10-11-pose-analysis-index.md
- created: .pose/reports/2026-10-13-pose-batch-9-authoring.md

### Delivery targets

Nenhum target funcional nesta escrita; targets nas duas specs são prospectivos.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5
- Decision: Atribuir a escrita a esta spec administrativa e manter os dois deltas funcionais draft.
- Rationale: Definir read view e cursor não qualifica coerência ou custo; arquitetura/protocolo e evidências exigem reconciliação prospectiva antes da implementação.

## 6. Validation

### Strategy

Conferir catálogo, bytes/status das fundações/reutilizadas, contagens, coverage, DAG, deps, links e ownership. Executar lint-spec nas três specs, artifact-check da escrita e check --strict; roadmap-check deve recusar closeout enquanto members/resultados estiverem pendentes. CI/Security verificam o head documental, sem provar requisitos funcionais futuros.

### Requirement trace

Vincular R1–R5 ao relatório e checks efetivamente executados. Manter in-progress até revisão e closeout nativo.

## 7. Final Report

### Delivered scope

Dois deltas draft, matriz e roadmaps escritos; conferência reproduzível, três lints, artifact-check e check --strict passaram, com avisos registrados no relatório. CI verificará o head documental; implementação, revisão e closeout permanecem pendentes.

### Residual risks

Closure, protocolo de captura/revalidação e compatibilidade de cursores exigem decisão e qualificação antes do código. Snapshot coerente pode ser histórico; não o promover a autorização atual.

### Follow-ups

Manter execução e lotes futuros na matriz; nenhuma disposition aplicada.
