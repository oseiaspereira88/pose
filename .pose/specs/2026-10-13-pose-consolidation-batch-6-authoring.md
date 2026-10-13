---
slug: pose-consolidation-batch-6-authoring
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

# Spec: Escrever e validar o lote documental 6

## 1. Intent

### Goal

Materializar as três specs de implementação de B28/B44/B38 e seus vínculos, sem iniciar seus requisitos funcionais.

### Business value

Continuar o lote 5 mergeado em #164 com atribuição exata dos artefatos desta escrita.
Esta spec governa somente a tarefa documental; não é uma quarta unidade de
implementação nem altera o limite de três specs substanciais do lote.

### Constraints

Preservar o catálogo, os aceites e o DAG históricos, as oito specs reutilizadas e
suas baselines. Manter os três novos deltas em draft. Não mudar motor, policy,
adoção, versões, dispositions ou states das specs antigas.

## 2. Requirements

- R1: A escrita deve materializar somente as specs planejadas para B28/B44/B38, com requisitos estáveis, aceites, artefatos prospectivos, validação, compatibilidade e gates de execução explícitos.
- R2: A matriz deve distinguir artefato materializado de implementação entregue, preservar 96 candidatos/44 unidades/17 lotes e manter os lotes 7–16 sem novas specs materializadas nesta interação.
- R3: Os roadmaps devem materializar vínculos apenas para refs existentes, manter uma ownership por spec e separar a qualificação de escrita da qualificação dos resultados de confiança.
- R4: A escrita deve preservar B28/B44/B38 como raízes independentes e materializar somente o roadmap responsável por B38, mantendo seus demais deltas nos lotes futuros.
- R5: Os onze artefatos documentais devem passar a conferência reproduzível, lint-spec, artifact-check e check --strict; a checagem de encerramento dos resultados deve recusar os deltas draft, sem alegar eficácia funcional.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-13-pose-consolidation-batch-6-authoring.md
- created: .pose/starts/pose-consolidation-batch-6-authoring.json
- created: .pose/changelogs/unreleased/pose-consolidation-batch-6-authoring.md
- created: .pose/specs/2026-10-13-pose-distributed-instructions-parity.md
- created: .pose/specs/2026-10-13-pose-release-manifest-tag-integrity.md
- created: .pose/specs/2026-10-13-pose-planning-and-release-narrative-reconciliation.md
- created: .pose/roadmaps/pose-release-sufficiency-and-recovery.md
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/reports/2026-10-11-pose-analysis-index.md
- created: .pose/reports/2026-10-13-pose-batch-6-authoring.md

### Delivery targets

Nenhum target de runtime nesta escrita. Os targets nas três specs pertencem à execução futura.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R5
- Decision: Atribuir a escrita a uma spec documental separada e manter os três deltas funcionais draft.
- Rationale: Criar uma spec não executa seus artefatos prospectivos nem comprova seu target; a validação documental precisa de escopo próprio.

## 6. Validation

### Strategy

Confrontar a matriz e os arquivos, conferir cobertura/DAG/ownership, dependências,
counts e ausência de lotes futuros. Registrar a conferência reproduzível no relatório.
Executar lint-spec nas quatro specs, artifact-check da escrita e check --strict;
esperar recusa de roadmap-check enquanto os resultados estiverem draft/in-progress.
CI/Security verificarão o head documental; não apresentar suas suites como teste
de uma implementação que esta interação não realizou.

### Requirement trace

Mapear R1–R5 ao relatório, à conferência documental e aos checks efetivamente
executados. Manter a spec de escrita in-progress até sua revisão e closeout.

## 7. Final Report

### Delivered scope

Três specs draft, matriz e vínculos escritos; a conferência documental e os quatro lints passaram. Conferir os demais resultados e o head de CI no PR documental deste lote. Execução funcional, revisão e closeout correspondentes permanecem pendentes.

### Residual risks

Os artefatos de implementação são prospectivos. Decisões de instruções distribuídas, narrativa atual,
compatibilidade e integridade do snapshot de release precisam de qualificação antes do código.
Não ativar ou fechar o programa somente porque seus members existem.

### Follow-ups

Manter o trabalho funcional e os próximos lotes na matriz, sem criar novas dispositions.
