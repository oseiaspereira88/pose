---
slug: pose-consolidation-program-bootstrap
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

# Spec: Formalizar o lote 0 do programa de consolidação do POSE

## 1. Intent

### Goal

Formalizar o coordenador, a matriz das 44 unidades e a reconciliação das entregas existentes, sem iniciar as implementações dos lotes futuros.

### Business value

Converter a consolidação da frente 10 em lotes de escrita explícitos e rastreáveis. Preservar as oito specs reutilizadas, as fundações concluídas e as correções mergeadas em #157/#158. A autorização do mantenedor nesta interação cobre a escrita do lote 0; a execução das implementações será tratada depois.

### Constraints

Preservar o catálogo histórico e suas dependências. Distinguir lote de escrita, ordem de execução e release. Não criar as specs futuras, alterar policy/adoção, fechar unidades por presença de código, reabrir ADRs aceitos ou mudar estados das specs existentes.

## 2. Requirements

- R1: A matriz deve mapear exatamente os 96 candidatos às 44 unidades canônicas, preservando fontes, aceites e dependências acíclicas.
- R2: Cada unidade deve pertencer a um lote de escrita entre 0 e 16 e a um roadmap responsável planejado, sem duplicar execução entre frentes.
- R3: O plano deve distinguir 36 novas specs, uma spec condicionada e oito specs existentes reutilizadas, sem criar referências governadas para artefatos futuros ausentes.
- R4: A reconciliação deve registrar os merges e heads testados de #157/#158 como entregas parciais de B01/B04, sem concluir unidades ou specs automaticamente.
- R5: O coordenador deve manter o bootstrap documental separado dos resultados de implementação e exigir materialização dos milestones futuros, qualificação e revisão antes do encerramento do programa.
- R6: A escrita deve declarar e reconciliar os artefatos deste lote e passar as verificações documentais aplicáveis, sem modificar o motor ou as fundações históricas.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-13-pose-consolidation-program-bootstrap.md
- created: .pose/starts/pose-consolidation-program-bootstrap.json
- created: .pose/changelogs/unreleased/pose-consolidation-program-bootstrap.md
- created: .pose/roadmaps/pose-consolidation-program.md
- created: .pose/results/pose-implementation-plan.json
- created: .pose/reports/2026-10-13-pose-plan-reconciliation.md
- modified: .pose/reports/2026-10-11-pose-analysis-index.md

### Delivery targets

Nenhum target de runtime. Entregar planejamento documental e referências verificáveis.

## 5. Decisions

### Decision D1
- Basis: R2, R3, R5
- Decision: Manter uma matriz autoritativa de planejamento; materializar milestones e referências executáveis apenas quando seus artefatos existirem.
- Rationale: Um milestone só com fundações done não comprova o delta futuro. O bootstrap qualifica a escrita do plano, não o encerramento das 44 unidades.

## 6. Validation

### Strategy

Conferir cobertura única de candidatos/unidades, DAG, limites dos lotes, reutilização, paths existentes e ausência dos futuros artefatos. Confrontar os PRs mergeados com seus heads e runs de CI. Executar lint-spec, artifact-check --strict e check --strict. Roadmap-check deverá recusar o encerramento enquanto o bootstrap estiver in-progress.

### Requirement trace

Mapear R1/R2/R3 ao validador documental reproduzível no relatório; R4 à reconciliação dos PRs e das specs; R5 ao roadmap e à recusa esperada de roadmap-check; R6 a lint-spec, artifact-check e check. Manter a spec in-progress até revisão e closeout.

## 7. Final Report

### Delivered scope

Artefatos do lote 0 escritos; cobertura, DAG e reutilização passaram na conferência documental reproduzível do relatório. Conferir os demais resultados de validação e o head testado no PR documental deste lote. Revisão independente e closeout permanecem pendentes.

### Residual risks

Check estrutural não prova completude de artefatos ainda não criados. Preservar a exigência documental de materialização e qualificação; revisão do programa deve confrontar a matriz completa.

### Follow-ups

Nenhum follow-up adicional: o trabalho futuro está explicitamente planejado na matriz, sem dispositions automáticas.
