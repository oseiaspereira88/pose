---
slug: pose-commit-hook-runs-with-explicit-roots
status: draft
created_at: 2026-11-11
completed_at:
supersedes:
depends_on: 
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
---

# Spec: Os hooks de commit rodam o pose com os roots da instância

## 1. Intent

### Goal

Os hooks que o `pose hooks install` escreve (pre-commit e commit-msg) resolvem o projeto e os roots autorizados da instância sem depender do ambiente do chamador, e a instância pode declarar o binário a usar.

### Business value

Em cada commit do harne8 o gate imprime "[!] warning review closeout … federated-material changed since generation" e avisos `unknown-project`: o hook roda `pose` sem `POSE_PROJECT_ROOTS`, e a geração dos índices foi feita pelo wrapper `scripts/pose-cli-local.sh` com os dois roots explícitos. O ruído esconde avisos reais. Resolver tratando o submódulo da instância como root autorizado de forma implícita é decisão de autoridade (concede escrita), por isso a spec começa por ADR.

### Constraints

Rascunho escrito em 2026-11-11 a partir das pendências da sessão de fechamento do roadmap `pose-shared-runtime`; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: ADR primeiro: como a instância declara os roots e o binário dos hooks sem conceder escrita a outro projeto por descoberta (a regra de `POSE_PROJECT_ROOTS` explícito continua valendo).
- R2: O hook pre-commit lê essa declaração da instância (arquivo versionado) e exporta os roots antes de chamar o motor; sem declaração, mantém o comportamento atual.
- R3: Num commit do harne8 o gate não imprime `unknown-project` nem `federated-material changed` quando nada mudou.
- R4: Um teste mostra que o hook não amplia a autoridade: root não declarado continua sem escrita.

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
