---
slug: pose-worktree-runtime-qos-and-measurement
status: draft
created_at: 2026-11-09
completed_at:
supersedes:
depends_on: pose-optional-worktree-runtime, pose-worktree-runtime-resident-cache
remediates:
priority: 3
components: pose-mcp
task_type: feature
surface: minimal
delivers:
---

# Spec: QoS, budgets e medição do runtime opcional

## 1. Intent

### Goal

O runtime protege consultas curtas de checks pesados com budgets por categoria/raiz e globais, e publica medições com thresholds fixados de um corpus sintético.

### Business value

Fatia da B35 / PERF-08 e PERF-12 decidida em 2026-11-09 (emenda do ADR [Shared domain and optional worktree runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md)): a entrega foi dividida em três specs revisadas separadamente. A primeira é `pose-optional-worktree-runtime`.

### Constraints

Opt-in; o runtime nunca é obrigatório e o fallback standalone permanece. Nenhum caminho de autoridade novo.

## 2. Requirements

- R6: Budgets de in-flight evaluation/admission/CPU/memória/IO e filhos são por categoria/raiz e globais, considerando quota/afinidade e jobs simultâneos. Query curta não deve regredir por competir com check pesado; output conserva ordem determinística.
- R8: Medir RAM/CPU/latência/output bytes e queue/admission wait por cenário cold/standalone, operação-local, resident warm/churn/restart e carga pesada concorrente. Fixar amostras, thresholds/budgets e ambiente antes da qualificação, medidos neste ambiente com margem; divulgar supported/configured/effective e limitações.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start, depois que a fatia anterior estiver entregue.

## 6. Validation

Cenários V* herdados da spec `pose-optional-worktree-runtime`, com fixtures sintéticas e ambiente declarado.
