---
slug: pose-consolidation-batch-12-authoring
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

# Spec: Escrever e validar o lote documental 12

## 1. Intent

### Goal

Formalizar B25/B26/B37 após merge #170, criando somente três specs draft e seus vínculos de planejamento.

### Business value

Preparar triagem autorizada, métricas por fatos e suficiência compartilhada do cut, sem confundir escrita com decisão aplicada ou entrega funcional.

### Constraints

Preservar as oito specs reutilizadas e fundações byte a byte, seus status/R-IDs/baselines, catálogo/aceites/DAG e planos de fases anteriores. Não executar agentes/runners, triagem/apply, mudança de policy/adoção, migração, release ou closeout.

## 2. Requirements

- R1: Materializar somente B25/B26/B37 em draft, com aceites, artefatos, compatibilidade, ADRs pertinentes e validação prospectivos; nenhuma implementação funcional ou decisão de risco aplicada.
- R2: Preservar 96 candidatos/44 unidades/17 lotes e oito reutilizadas, registrar merge do lote 11 sem fabricar closeout nativo e manter lotes 13–16 sem novas propostas materializadas.
- R3: Governance-efficiency deve possuir B25/B26 uma vez e release-sufficiency-and-recovery deve possuir B37; coordenador consome milestones qualificados sem duplicar ownership.
- R4: Preservar B25 após B01/B14/B24, B26 após B10/B24 e B37 após B01/B02/B08/B30, sem dependências temáticas artificiais. Distinguir candidato/decisão/aplicação, resposta/condição/restrição e path/prova/approval, conservando B19 → B20 → B21.
- R5: Os doze paths documentais devem passar conferência reproduzível, lint-spec, artifact-check e check --strict; roadmap-check deve recusar encerramento dos resultados pendentes, sem alegar entrega funcional.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-13-pose-consolidation-batch-12-authoring.md
- created: .pose/starts/pose-consolidation-batch-12-authoring.json
- created: .pose/changelogs/unreleased/pose-consolidation-batch-12-authoring.md
- created: .pose/specs/2026-10-13-pose-residual-triage-and-risk-acceptance.md
- created: .pose/specs/2026-10-13-pose-condition-response-and-work-metrics.md
- created: .pose/specs/2026-10-13-pose-release-shared-sufficiency-contract.md
- modified: .pose/roadmaps/governance-efficiency.md
- modified: .pose/roadmaps/pose-release-sufficiency-and-recovery.md
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/reports/2026-10-11-pose-analysis-index.md
- created: .pose/reports/2026-10-13-pose-batch-12-authoring.md

### Delivery targets

Nenhum target funcional desta escrita; targets das novas specs pertencem à execução futura.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5
- Decision: Reusar ADRs de obligations/release e fundações done; explicitar contratos e casos prospectivos sem selecionar schema/API novo nesta escrita.
- Rationale: Autoridade, satisfação, temporização e prova de entrega pertencem aos produtores existentes. Decisão estrutural adicional exige ADR antes do código, sem duplicar ledger/evaluator ou reinterpretar história selada.

## 6. Validation

### Strategy

Conferir catálogo/digests/status/R-IDs, contagens/coverage, DAG nativo/por unidade e fases, links e ownership. Executar lint-spec das quatro specs, artifact-check da escrita e check --strict; roadmap-check deve recusar closeout enquanto resultados/members estiverem pendentes. CI/Security verificam o head documental e não satisfazem casos funcionais prospectivos.

### Requirement trace

Vincular R1–R5 ao relatório/checks executados; manter in-progress até revisão e closeout nativo.

## 7. Final Report

### Delivered scope

Três deltas draft, matriz e roadmaps escritos. Conferência reproduzível, quatro lints, artifact-check e check --strict passaram, com avisos registrados no relatório; roadmap-check recusou closeout dos resultados pendentes. CI/Security verificarão o head documental. Implementação, revisão e closeout nativo permanecem pendentes.

### Residual risks

Risco/receipts e métricas exigem reconciliação de sintaxe/compatibilidade antes de start. Descoberta de critérios/ref evaluator requer contrato deliberado; ausência de evento não prova duração zero, e arquivo existente não prova aprovação.

### Follow-ups

Manter implementação e lotes futuros na matriz; nenhuma disposition, aceite de risco ou release aplicada.
