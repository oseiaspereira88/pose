---
slug: pose-setup-offers-runtime-and-sandbox-dependencies
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-claude-sandboxed-profile
remediates:
priority: 3
components: pose-mcp
task_type: feature
---

# Spec: O pose setup oferece as dependências do runtime e do sandbox

## 1. Intent

### Goal

O `pose setup` trata as dependências opcionais (bubblewrap e socat para o perfil sandboxed do Claude, o runtime por worktree) como decisões de capacidade, mostra o que cada uma muda e como instalar.

### Business value

Hoje só o `pose doctor` avisa das dependências de um perfil (`delegation.requirements`); o onboarding do `pose setup` oferece só a capacidade `delegated-review`.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: O `pose setup` lista a capacidade com o que muda e o comando de instalação por plataforma.
- R2: Decidir, adiar e recusar ficam registrados como nas outras capacidades.
- R3: O `pose doctor` continua avisando quando a capacidade foi adotada e a dependência falta.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.
