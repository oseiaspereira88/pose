---
slug: pose-runtime-resource-limits-per-command
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-worktree-runtime-qos-and-measurement
remediates:
priority: 4
components: pose-mcp
task_type: feature
---

# Spec: Limites de CPU, memória e E/S por comando hospedado

## 1. Intent

### Goal

O runtime limita a CPU, a memória e a E/S que um comando hospedado pode consumir, onde a plataforma oferece (cgroups no Linux), e declara o que é suportado, configurado e efetivo.

### Business value

O runtime limita o que admite (faixas, filas) e o que guarda (o memo), mas não o que cada comando consome; o ADR declarou cota e afinidade de CPU como não entregues.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Um limite por faixa de CPU e de memória, aplicado onde há cgroup v2, e reportado como efetivo ou não no status.
- R2: A afinidade de CPU (conjunto de CPUs de cada faixa) e um limite de E/S (`io.max` no cgroup v2) são configuráveis e reportados como efetivos ou não, como a CPU e a memória.
- R4: A medição inclui a CPU, a memória e a E/S sob o limite e um comando que o excede.
- R3: ADR: o limite é do processo do runtime ou de um filho por comando.

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
