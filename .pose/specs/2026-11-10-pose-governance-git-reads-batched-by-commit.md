---
slug: pose-governance-git-reads-batched-by-commit
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-governance-audit-skips-pure-git-reads
remediates:
priority: 3
components: pose-mcp
task_type: feature
---

# Spec: Leituras do Git da governança agrupadas por commit

## 1. Intent

### Goal

As cerca de 240 execuções de `git log -1 -- <path>` de um `review-check` viram uma leitura agrupada que dá, para cada caminho, o mesmo commit que a execução individual dá.

### Business value

Depois das specs de engine, o `review-check` leva cerca de 25 s e cerca de 14 s são esses `git log -1 -- <path>` (cerca de 57 ms cada num histórico de 2 800 commits). Agrupá-los muda a semântica da simplificação de histórico de merges, então a equivalência tem de ser provada, não suposta.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Uma leitura agrupada devolve, para cada caminho pedido, o commit que `git log -1 -- <path>` devolveria, provado por um teste de equivalência sobre um histórico com merges que tocam e revertem o caminho.
- R2: A leitura agrupada só é usada onde a equivalência vale; onde não, a individual.
- R3: A medição mostra o ganho no `review-check` com a saída idêntica.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.
