---
slug: pose-consolidation-batch-14-authoring
status: in-progress
created_at: 2026-10-14
completed_at:
depends_on:
priority: 0
components: .pose
task_type: feature
surface: minimal
delivers:
---

# Spec: Escrever e validar o lote documental 14

## 1. Intent

### Goal

Formalizar B31/B33/B35 após merge #173, criando somente três specs draft e proposta ADR/nota de revisão.

### Business value

Preparar fonte de domínio comum, vistas/caches isolados e runtime opcional com fallback e QoS, preservando autoridade, geração e fronteira de execução externa.

### Constraints

Preservar oito reutilizadas/fundações e ADRs aceitos byte a byte, status/R-IDs/start baselines, catálogo/aceites/DAG e B19 → B20 → B21. Não executar daemon/IPC/host MCP, agentes/runners, benchmark de runtime, implementação funcional, policy/adoption, migração, release ou closeout. ADR fica Proposed.

## 2. Requirements

- R1: Materializar somente B31/B33/B35 em draft, com mapping dos sete candidatos, artefatos/compatibilidade/validação prospectivos e proposta de domínio/IPC/cache vinculada; nenhuma implementação funcional.
- R2: Preservar 96 candidatos/44 unidades/17 lotes, oito reutilizadas e fundações; registrar merge do lote 13 sem inventar closeout nativo e manter lotes 15–16 sem novas propostas materializadas.
- R3: Shared-runtime deve possuir os três novos deltas uma vez, mantendo B32; coordenador consome milestones qualificados sem duplicar ownership ou encerrar o programa pela escrita.
- R4: Preservar B31 após B12/B14/B34, B33 após B15/B34 e B35 após B07/B12/B14/B15/B31/B32/B33/B34. Não exigir B35 para extrair API ou isolar caches; conservar stdio do cliente, authority dos writers e fronteira MCP-Conductor-Harness.
- R5: Os treze paths documentais devem passar conferência reproduzível, lint-spec, knowledge-check, artifact-check e check --strict; roadmap-check deve recusar closeout de entregas pendentes. Proposta/nota não adota runtime, schema, cache policy ou IPC.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-14-pose-consolidation-batch-14-authoring.md
- created: .pose/starts/pose-consolidation-batch-14-authoring.json
- created: .pose/changelogs/unreleased/pose-consolidation-batch-14-authoring.md
- created: .pose/specs/2026-10-14-pose-common-domain-and-cli-mcp-handoff.md
- created: .pose/specs/2026-10-14-pose-bounded-isolated-graph-caches.md
- created: .pose/specs/2026-10-14-pose-optional-worktree-runtime.md
- modified: .pose/roadmaps/pose-shared-runtime.md
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/reports/2026-10-11-pose-analysis-index.md
- created: .pose/reports/2026-10-14-pose-batch-14-authoring.md
- created: .pose/adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md
- created: .pose/knowledge/2026-10-14-decision-log-adr-shared-worktree-runtime-review.md

### Delivery targets

Nenhum target funcional nesta escrita; targets das specs draft pertencem à execução futura.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5
- Decision: Registrar proposta de domínio/runtime com stdio não compartilhável, cache isolado/bounded e chamadas admitidas, preservando scoped Git, active-context e Harness.
- Rationale: Residência sem geração/authority/budgets amplia riscos; compatibilidade, IPC e métricas precisam de resolução antes do código, sem implementar scheduler ou duplicar regras.

## 6. Validation

### Strategy

Conferir catálogo/digests/status/R-IDs, contagens/coverage, DAG/overlays/fases, links e ownership. Executar quatro lint-spec, knowledge-check e artifact-check da escrita, check --strict e roadmap-check que deve recusar terminal. CI/Security verificam head documental, não casos funcionais prospectivos ou eficácia de runtime.

### Requirement trace

Vincular R1–R5 ao relatório e checks; manter in-progress até review/closeout nativo.

## 7. Final Report

### Delivered scope

Três drafts, ADR Proposed, nota de revisão, matriz e roadmaps escritos. Conferência reproduzível, quatro lints, knowledge-check, artifact-check e check --strict passaram, com avisos registrados; roadmap-check recusou closeout pendente. CI/Security verificarão o head documental. Implementação, revisão e closeout nativo permanecem pendentes.

### Residual risks

IPC/compatibilidade/ownership e budgets ainda exigem resolução. Mutation/race e QoS precisam de provas atuais; stdio, watcher/cache hit e processo aberto não comprovam autoridade, frescor ou residência compartilhável.

### Follow-ups

Manter execução funcional e lotes futuros na matriz. Consultar knowledge:adr-shared-worktree-runtime-review; nenhuma adoção ou runtime iniciado.
