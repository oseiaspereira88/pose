---
slug: pose-review-check-time-and-output-regression-test
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-governance-audit-skips-pure-git-reads
remediates:
priority: 3
components: pose-mcp
task_type: bugfix
---

# Spec: Teste de regressão para o tempo e a saída da vinculação das entradas

## 1. Intent

### Goal

O tempo e a saída de `review-check` num repositório com histórico real passam a ter um teste de regressão com limiar fixado, em vez de medição manual.

### Business value

As specs `pose-index-input-binding-reads-each-transfer-once` e `pose-governance-audit-skips-pure-git-reads` reduziram o `review-check` de cerca de 56 s para cerca de 25 s com a saída idêntica, mas só provaram o tempo e a saída por medição manual; qualquer regressão volta sem teste que a pegue.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Um teste opt-in mede o `review-check` de um repositório sintético de histórico e artefatos fixados e falha acima de um limiar declarado antes da medição.
- R2: O mesmo teste compara a saída com um golden.
- R3: O número de execuções do `git` por operação tem um teto testado.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.
