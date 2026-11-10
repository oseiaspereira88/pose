---
slug: pose-runtime-memo-hits-and-git-config-key
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-worktree-runtime-resident-cache
remediates:
priority: 3
components: pose-mcp
task_type: bugfix
---

# Spec: Acertos do memo por comando e configuração do Git na chave

## 1. Intent

### Goal

O envelope de `pose runtime run` traz os acertos do memo do próprio comando mesmo com faixas concorrentes, e a chave do memo inclui a configuração global e de usuário do Git.

### Business value

`memo_hits` é a diferença de um contador global e inclui os de comandos concorrentes; a chave usa o `.git/config` do repositório mas não o config global e de usuário, que também mudam o que um `git log` responde (`log.follow`); a admissão dos bytes de um acerto contra o orçamento de uma operação quase esgotado não tem teste.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: `memo_hits` é contado por operação.
- R2: A chave do memo inclui o digest do config global e de usuário do Git.
- R3: Um teste admite um acerto contra um orçamento quase esgotado e vê o recuso do budget.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.
