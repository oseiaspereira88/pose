---
slug: pose-worktree-runtime-resident-cache
status: draft
created_at: 2026-11-09
completed_at:
supersedes:
depends_on: pose-optional-worktree-runtime
remediates:
priority: 2
components: pose-mcp
task_type: feature
surface: minimal
delivers:
---

# Spec: Reuso residente das views do domínio e resultado tipado

## 1. Intent

### Goal

O runtime opcional reusa as views imutáveis da B33 entre chamadas admitidas, reconcilia mudanças externas e devolve resultado tipado B12.

### Business value

Fatia da B35 / PERF-08 e PERF-12 decidida em 2026-11-09 (emenda do ADR [Shared domain and optional worktree runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md)): a entrega foi dividida em três specs revisadas separadamente. A primeira é `pose-optional-worktree-runtime`.

### Constraints

Opt-in; o runtime nunca é obrigatório e o fallback standalone permanece. Nenhum caminho de autoridade novo.

## 2. Requirements

- R5: Cache/catalogue resident reuse depende de B15/B33/B34 e inputs completos, bodies sob demanda e budgets em bytes por worktree/global. Watch events são hints; perdas, rename/replace, checkout/reset e mudanças Git/refs exigem reconciliação. Writers B14 publicam e geração/verified/cache não são herdados após restart.
- R7: Resultado tipado B12 distingue domínio concluído de erro, cancel/deadline/conflict/partial effects e traz generation/input bindings/coverage/receipt. Preservar a fronteira MCP-Conductor-Harness: serviços avaliam chamadas admitidas, não executam arbitrariamente.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start, depois que a fatia anterior estiver entregue.

## 6. Validation

Cenários V* herdados da spec `pose-optional-worktree-runtime`, com fixtures sintéticas e ambiente declarado.
