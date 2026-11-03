---
slug: pose-consolidation-program-bootstrap
status: done
created_at: 2026-10-13
completed_at: 2026-11-03
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
- created: .pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md
- created: .pose/results/pose-consolidation-authoring-audit.json
- created: .pose/reports/2026-11-03-pose-consolidation-authoring-audit.py
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

- R1 [satisfied] report:.pose/reports/2026-10-13-pose-plan-reconciliation.md commit:e4ca6f572c6bb877c51f2f0a0b6109cc5db35862 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Cobertura dos 96 candidatos nas 44 unidades e DAGs conferidos no snapshot do bootstrap.
- R2 [satisfied] report:.pose/reports/2026-10-13-pose-plan-reconciliation.md commit:e4ca6f572c6bb877c51f2f0a0b6109cc5db35862 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Distribuição única das unidades entre lotes 0–16 e roadmaps planejados conferida.
- R3 [satisfied] report:.pose/reports/2026-10-13-pose-plan-reconciliation.md commit:e4ca6f572c6bb877c51f2f0a0b6109cc5db35862 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — 36 novas propostas, uma condicionada, oito reutilizadas e ausência das propostas futuras conferidas no snapshot.
- R4 [satisfied] report:.pose/reports/2026-10-13-pose-plan-reconciliation.md commit:e4ca6f572c6bb877c51f2f0a0b6109cc5db35862 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Registros de merge/head de #157/#158 conferidos também pelos segundos parents dos objetos Git, sem nova consulta ao CI.
- R5 [satisfied] report:.pose/reports/2026-10-13-pose-plan-reconciliation.md commit:e4ca6f572c6bb877c51f2f0a0b6109cc5db35862 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Coordenador separa autoria de implementação e mantém qualificação e revisão como condições de encerramento.
- R6 [satisfied] report:.pose/reports/2026-10-13-pose-plan-reconciliation.md commit:e4ca6f572c6bb877c51f2f0a0b6109cc5db35862 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Artefatos documentais reconciliados e reproduzidor histórico executado; gates atuais permanecem exigidos pelo fechamento nativo.

## 7. Final Report

### Delivered scope

Autoria documental concluída pelo fechamento nativo em 2026-11-03. Os reproduzidores dos snapshots de escrita passaram sem alteração do código histórico; consultar a [auditoria comum](../reports/2026-11-03-pose-consolidation-authoring-closeout.md).

Gates locais e critérios de saída julgados no [bundle rvb-681201947b36fa95](../review-bundles/rvb-681201947b36fa95.json), com [atestação rva-039030d7174daabd](../review-attestations/rva-039030d7174daabd.json) de uma execução separada do Codex, mesmo fornecedor/modelo herdado. Sem alegação de identidade autenticada ou revisão entre fornecedores.

O fechamento qualifica a escrita e seus vínculos; não conclui implementação, adoção, ADR proposto, piloto ou programa de consolidação. Nenhum CI remoto foi consultado.

### Residual risks

Check estrutural não prova completude de artefatos ainda não criados. Preservar a exigência documental de materialização e qualificação; revisão do programa deve confrontar a matriz completa.

### Follow-ups

Nenhum follow-up adicional: o trabalho futuro está explicitamente planejado na matriz, sem dispositions automáticas.
