---
slug: pose-retroactive-attribution-record
status: draft
created_at: 2026-11-11
completed_at:
supersedes:
depends_on: 
remediates:
priority: 2
components: pose-mcp
task_type: feature
---

# Spec: Registro de atribuição retroativa para histórico selado

## 1. Intent

### Goal

O motor aceita um registro assinado e versionado que reatribui caminhos de commits já publicados (compartilhados por vários trailers ou sem spec declarante) a uma spec, sem editar a seção Artifacts selada das specs fechadas.

### Business value

O `pose check --strict` do pose-dist falha com 19 erros "undeclared governed path" desde o fechamento do roadmap `pose-shared-runtime` (2026-11-10): o commit de congelamento `80b49143` cita seis specs e altera `resident_git_memo.go` e seu teste, que nenhuma spec fechada declara; `57e20957` (um trailer) alterou arquivos declarados por outras duas; `pose-project-agnostic-assessment-engines` tem dois casos próprios. Uma regra de motor para caminhos de commit compartilhado reduziu os erros de 19 para 14 mas não chega a verde (testado e descartado em 2026-11-11), e a seção Artifacts das specs fechadas é selada. O gap é de governança: o histórico publicado não pode ser reescrito, então a correção é um registro explícito, auditável e que não afrouxe o gate para commits futuros.

### Constraints

Rascunho escrito em 2026-11-11 a partir das pendências da sessão de fechamento do roadmap `pose-shared-runtime`; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: ADR primeiro: forma do registro (arquivo, assinatura, quem pode escrever), escopo (caminho + commit + spec declarante) e por que não uma regra implícita.
- R2: `pose index` e `pose artifact-check` leem o registro e deixam de reportar `undeclared` para o par (commit, caminho) reatribuído, com a razão e o autor visíveis no relatório.
- R3: O registro só cobre commits já existentes no momento da escrita; um commit novo com o mesmo problema continua reprovando.
- R4: Um registro que referencia commit ou spec inexistentes, ou um caminho que o commit não alterou, é recusado.
- R5: Os 19 erros do pose-dist são resolvidos por registros, e o `pose check --strict` do pose-dist passa.

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
