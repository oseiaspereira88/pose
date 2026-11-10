---
slug: pose-delegated-output-redaction-root-cause
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-delegate-picker-and-claude-runs
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
---

# Spec: Causa raiz do 'agent's output could not be redacted' nas tarefas Claude

## 1. Intent

### Goal

Uma tarefa delegada ao Claude no harne8 não falha mais com 'agent's output could not be redacted', e a falha, quando ocorre, nomeia o perfil e o motivo.

### Business value

No harne8 o perfil antigo do Claude produzia essa falha e foi contornado trocando-o pelo do repositório do motor; a causa raiz na redação nunca foi isolada, e a mensagem não diz qual perfil falhou.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Reproduzir a falha com o perfil antigo e isolar o ponto da redação que a produz.
- R2: A redação trata a saída que a causou sem descartar o registro; ou a falha nomeia o perfil, o evento e o motivo.
- R3: Um teste com a saída que a causou falha antes e passa depois.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 4. Tasks

- [ ] Completar a spec (artefatos, alvos de entrega, estratégia de validação) e iniciar
- [ ] Implementar e testar
- [ ] Revisão independente e fechamento

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.

---

## 7. Final Report

### Delivered scope
Rascunho: nada entregue ainda.
