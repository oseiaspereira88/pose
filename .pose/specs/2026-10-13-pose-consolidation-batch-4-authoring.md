---
slug: pose-consolidation-batch-4-authoring
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

# Spec: Escrever e validar o lote documental 4

## 1. Intent

### Goal

Materializar as três specs de implementação de B12/B13/B30 e seus vínculos, sem iniciar seus requisitos funcionais.

### Business value

Continuar o lote 3 mergeado em #162 com atribuição exata dos artefatos desta escrita.
Esta spec governa somente a tarefa documental; não é uma quarta unidade de
implementação nem altera o limite de três specs substanciais do lote.

### Constraints

Preservar o catálogo, os aceites e o DAG históricos, as oito specs reutilizadas e
suas baselines. Manter os três novos deltas em draft. Não mudar motor, policy,
adoção, versões, dispositions ou states das specs antigas.

## 2. Requirements

- R1: A escrita deve materializar somente as specs planejadas para B12/B13/B30, com requisitos estáveis, aceites, artefatos prospectivos, validação, compatibilidade e gates de execução explícitos.
- R2: A matriz deve distinguir artefato materializado de implementação entregue, preservar 96 candidatos/44 unidades/17 lotes e manter os lotes 5–16 sem novas specs materializadas nesta interação.
- R3: Os roadmaps devem materializar vínculos apenas para refs existentes, manter uma ownership por spec e separar a qualificação de escrita da qualificação dos resultados de confiança.
- R4: A escrita deve preservar B12/B30 independentes e B13 após B12, com dependências nativas reais e sem contornar specs não terminais.
- R5: Os doze artefatos documentais devem passar a conferência reproduzível, lint-spec, artifact-check e check --strict; a checagem de encerramento dos resultados deve recusar os deltas draft, sem alegar eficácia funcional.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-13-pose-consolidation-batch-4-authoring.md
- created: .pose/starts/pose-consolidation-batch-4-authoring.json
- created: .pose/changelogs/unreleased/pose-consolidation-batch-4-authoring.md
- created: .pose/specs/2026-10-13-pose-cli-mcp-operation-results.md
- created: .pose/specs/2026-10-13-pose-journal-checkpoint-and-lock-recovery.md
- created: .pose/specs/2026-10-13-pose-spec-facts-evidence-binding.md
- modified: .pose/roadmaps/pose-state-consistency-and-recovery.md
- modified: .pose/roadmaps/governance-efficiency.md
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/reports/2026-10-11-pose-analysis-index.md
- created: .pose/reports/2026-10-13-pose-batch-4-authoring.md

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

Os artefatos de implementação são prospectivos. Decisões de resultados operacionais, recovery de locks/checkpoints,
compatibilidade e vínculo de evidência precisam de qualificação antes do código.
Não ativar ou fechar o programa somente porque seus members existem.

### Follow-ups

Manter o trabalho funcional e os próximos lotes na matriz, sem criar novas dispositions.
