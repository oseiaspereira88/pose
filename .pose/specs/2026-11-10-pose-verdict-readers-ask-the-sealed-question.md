---
slug: pose-verdict-readers-ask-the-sealed-question
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-delegated-approval-survives-routine-drift
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
---

# Spec: Os leitores de veredito respondem a pergunta selada, não a viva

## 1. Intent

### Goal

`pose review run-proof` e os demais leitores de veredito de uma run delegada usam a mesma pergunta que o fechamento usa para um escopo done (a prova como selada), e a expiração no caminho as-sealed é decidida e documentada.

### Business value

A spec `pose-delegated-approval-survives-routine-drift` reteve as aprovações no fechamento, mas `pose review run-proof` e leitores de veredito ainda perguntam a pergunta viva (a run ainda governa o escopo agora) e a expiração é ignorada no caminho as-sealed sem estar escrita no manual: dois leitores podem discordar sobre a mesma run.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Os leitores de veredito de uma run delegada de um escopo done respondem `ReviewRunRecordStands`/prova selada como o `closeout-check`, e não `ReviewRunStillGoverns`.
- R2: A expiração de uma aprovação no caminho as-sealed é decidida (vale ou não) e escrita no manual e na mensagem.
- R3: Um teste faz o `closeout-check` e cada leitor concordarem sobre a mesma run com deriva rotineira.

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
