---
slug: pose-brief-ties-every-conclusion-refusal
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-review-brief-states-the-conclusion-rules
remediates:
priority: 3
components: pose-mcp
task_type: bugfix
---

# Spec: O brief da revisão declara toda regra que recusa uma conclusão

## 1. Intent

### Goal

O teste de amarração entre as mensagens de recusa e o brief cobre todas as fontes de recusa de uma conclusão, e o brief declara as regras que faltam.

### Business value

A spec do brief amarrou só as quatro funções do preflight; ficaram de fora as regras de mapeamento de causalidade, de decisão de finding, de reuso, o protocolo de rodada e `ParseReviewDraft`, entre elas três ou mais mapeamentos numa base com a mesma justificativa, uma base que não alcança requisito e uma linha `decision:` fora do bloco que discorda dele.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: O teste de amarração lê as mensagens de todas as fontes de recusa de uma conclusão (e não só as quatro funções do preflight).
- R2: O brief declara as regras ainda não declaradas, em uma versão nova do template (a anterior é imutável).
- R3: Cada mensagem tem uma frase do brief que a declara ou uma isenção justificada.

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
