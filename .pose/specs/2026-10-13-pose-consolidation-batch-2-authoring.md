---
slug: pose-consolidation-batch-2-authoring
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

# Spec: Escrever e validar o lote documental 2

## 1. Intent

### Goal

Materializar as três specs de implementação de B03/B06/B07 e seus vínculos, sem iniciar seus requisitos funcionais.

### Business value

Continuar o lote 1 mergeado em #160 com atribuição exata dos artefatos desta escrita.
Esta spec governa somente a tarefa documental; não é uma quarta unidade de
implementação nem altera o limite de três specs substanciais do lote.

### Constraints

Preservar o catálogo, os aceites e o DAG históricos, as oito specs reutilizadas e
suas baselines. Manter os três novos deltas em draft. Não mudar motor, policy,
adoção, versões, dispositions ou states das specs antigas.

## 2. Requirements

- R1: A escrita deve materializar somente as specs planejadas para B03/B06/B07, com requisitos estáveis, aceites, artefatos prospectivos, validação, compatibilidade e gates de execução explícitos.
- R2: A matriz deve distinguir artefato materializado de implementação entregue, preservar 96 candidatos/44 unidades/17 lotes e manter os lotes 3–16 sem novas specs materializadas nesta interação.
- R3: Os roadmaps devem materializar vínculos apenas para refs existentes, manter uma ownership por spec e separar a qualificação de escrita da qualificação dos resultados de confiança.
- R4: A escrita deve preservar B03 após B01/B04 e B06/B07 como raízes independentes; antecipar somente o roadmap responsável por B07, sem materializar suas unidades futuras ou contornar dependências não terminais.
- R5: Os treze artefatos documentais devem passar a conferência reproduzível, lint-spec, artifact-check e check --strict; a checagem de encerramento dos resultados deve recusar os deltas draft, sem alegar eficácia funcional.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-13-pose-consolidation-batch-2-authoring.md
- created: .pose/starts/pose-consolidation-batch-2-authoring.json
- created: .pose/changelogs/unreleased/pose-consolidation-batch-2-authoring.md
- created: .pose/specs/2026-10-13-pose-proof-presence-channel-parity.md
- created: .pose/specs/2026-10-13-pose-source-coverage-integrity.md
- created: .pose/specs/2026-10-13-pose-git-and-dispatch-snapshot-integrity.md
- modified: .pose/roadmaps/record-truthfulness.md
- modified: .pose/roadmaps/agency-readiness.md
- created: .pose/roadmaps/pose-state-consistency-and-recovery.md
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/reports/2026-10-11-pose-analysis-index.md
- created: .pose/reports/2026-10-13-pose-batch-2-authoring.md

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

- R1 [satisfied] report:.pose/reports/2026-10-13-pose-batch-2-authoring.md commit:4c22c3d608f570c18b66fed82701db23943c5c48 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Materialização e cobertura documental conferidas no snapshot do lote, sem afirmar execução funcional.
- R2 [satisfied] report:.pose/reports/2026-10-13-pose-batch-2-authoring.md commit:4c22c3d608f570c18b66fed82701db23943c5c48 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Contagens, lotes, reutilização e ausência dos artefatos futuros conferidas no snapshot de escrita.
- R3 [satisfied] report:.pose/reports/2026-10-13-pose-batch-2-authoring.md commit:4c22c3d608f570c18b66fed82701db23943c5c48 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Ownership e referências materializadas conferidas; consumo pelo coordenador separado de entrega funcional.
- R4 [satisfied] report:.pose/reports/2026-10-13-pose-batch-2-authoring.md commit:4c22c3d608f570c18b66fed82701db23943c5c48 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Dependências e limites da execução prospectiva conferidos no plano e nos artefatos daquele lote.
- R5 [satisfied] report:.pose/reports/2026-10-13-pose-batch-2-authoring.md commit:4c22c3d608f570c18b66fed82701db23943c5c48 report:.pose/reports/2026-11-03-pose-consolidation-authoring-closeout.md — Conferência histórica e checks documentais registrados no relatório; gates atuais são exigidos novamente pelo fechamento nativo.

## 7. Final Report

### Delivered scope

Autoria documental concluída pelo fechamento nativo em 2026-11-03. Os reproduzidores dos snapshots de escrita passaram sem alteração do código histórico; consultar a [auditoria comum](../reports/2026-11-03-pose-consolidation-authoring-closeout.md).

Gates locais e critérios de saída julgados no [bundle rvb-600e9e49de40e63c](../review-bundles/rvb-600e9e49de40e63c.json), com [atestação rva-08d7378a329f5dca](../review-attestations/rva-08d7378a329f5dca.json) de uma execução separada do Codex, mesmo fornecedor/modelo herdado. Sem alegação de identidade autenticada ou revisão entre fornecedores.

O fechamento qualifica a escrita e seus vínculos; não conclui implementação, adoção, ADR proposto, piloto ou programa de consolidação. Nenhum CI remoto foi consultado.

### Residual risks

Os artefatos de implementação são prospectivos. Decisões de presença, cobertura por recorte,
compatibilidade e autorização dos outputs precisam de qualificação antes do código.
Não ativar ou fechar o programa somente porque seus members existem.

### Follow-ups

Manter o trabalho funcional e os próximos lotes na matriz, sem criar novas dispositions.
