---
slug: pose-delegate-picker-fish-and-glyph-follow-ups
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-delegate-picker-and-claude-runs
remediates:
priority: 3
components: pose-mcp
task_type: bugfix
---

# Spec: Follow-ups do seletor de runs: fish, glifos e a matriz

## 1. Intent

### Goal

O seletor de `pose delegate watch` escapa `\\` num caminho do pose sob fish, mede largura em células e tem o teste novo na matriz.

### Business value

A revisão do seletor deixou quatro follow-ups baixos: o caminho do pose com barra invertida é alterado pelo fish dentro de aspas simples; o teste do fish é pulado em silêncio onde ele não existe; os glifos da lista têm largura ambígua em terminais CJK e a largura é contada em pontos de código; o padrão `-run` do `delegate-picker-integration` não inclui `TestThePickerRowsAlignEveryStateAndTheArgumentsHideTheID`.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: O comando do preview resiste a um caminho com `\\` sob um shell de login fish (testado onde há fish, com falha explícita em CI).
- R2: A largura da coluna de estado conta células e não pontos de código.
- R3: O padrão `-run` da matriz cobre os testes do seletor.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.
