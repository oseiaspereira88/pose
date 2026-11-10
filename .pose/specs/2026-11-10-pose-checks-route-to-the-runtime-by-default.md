---
slug: pose-checks-route-to-the-runtime-by-default
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-worktree-runtime-qos-and-measurement
remediates:
priority: 3
components: pose-mcp
task_type: feature
---

# Spec: Os checks vão para o runtime quando ele existe, sem pose runtime run

## 1. Intent

### Goal

Com o runtime do worktree rodando, compatível e disposto, `review-check`, `closeout-check`, `artifact-check` e `surface-check` rodam nele sem o chamador pedir, com a saída e o código de saída de sempre e o fallback standalone.

### Business value

O runtime só atende quem usa `pose runtime run`; o ganho (cerca de 25 s para cerca de 10 s no `review-check`) não chega a quem digita `pose review-check`. A decisão de rotear por padrão toca o contrato de cada comando e a saída com cor num terminal.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Um comando hospedado roteia ao runtime quando há um compatível e o digest de ambiente confere, e roda standalone caso contrário, com a mesma saída e o mesmo código.
- R2: A saída num terminal mantém a renderização com cor do standalone.
- R3: Um chamador pode desligar o roteamento (flag e ambiente) e nada inicia o runtime por conta própria.
- R4: ADR: decidir o roteamento por padrão antes do código.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.
