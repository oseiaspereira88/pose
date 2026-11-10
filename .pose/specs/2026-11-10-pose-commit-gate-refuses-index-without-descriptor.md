---
slug: pose-commit-gate-refuses-index-without-descriptor
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: 
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
---

# Spec: Gate de commit recusa um índice alterado sem o descritor da geração

## 1. Intent

### Goal

O gate de commit, no modo estrito, recusa um commit que muda um arquivo do conjunto de índices (`.pose/indexes/*.json` da geração) sem mudar o `generation.json` que o descreve, e diz que o remédio é `pose index`.

### Business value

O harne8 ficou com 113 avisos persistentes porque o commit `e1593bde` regenerou `spec-graph.json` e `roadmaps.json` sem o descritor; só um `pose index` reparador (spec `pose-index-recovery-and-close-scope-consistency`) o desfez. Nada impediu o commit.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Um commit que altera um arquivo do conjunto de índices sem alterar o `generation.json` é recusado pelo gate estrito, nomeando os arquivos e `pose index`.
- R2: Um commit que altera o descritor e todos os arquivos que ele nomeia passa; um commit que não toca índices passa.
- R3: O modo tolerante avisa em vez de recusar.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.
