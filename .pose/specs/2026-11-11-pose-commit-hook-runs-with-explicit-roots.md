---
slug: pose-commit-hook-runs-with-explicit-roots
status: abandoned
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

## 5. Decisions

### Decision D1
- Date: 2026-11-17
- Context: o rascunho foi escrito em 2026-11-11, antes da spec `pose-hooks-delegate-to-the-instance-runner` (done em 2026-11-15). Ela deixa a instância declarar em `.pose/policy/hooks.json` o executor dos seus hooks, e faz o `pre-commit` e o `post-merge` do motor delegarem a ele; o harne8 declara `scripts/pose-hook.sh`, que chama o motor pelo wrapper com os dois roots explícitos. O motor não instala `commit-msg`: os hooks gerenciados são `pre-commit` e `post-merge`, os dois que a delegação cobre.
- Decision: abandonar o rascunho, coberto por `pose-hooks-delegate-to-the-instance-runner`.
- Rationale: R2 e R3 estão entregues pela delegação: os hooks do harne8 rodam o motor com os roots que a instância declara, e em 2026-11-17 cada pre-commit do harne8 terminou com `Result: SUCCESS (tolerant mode)` e só os dois avisos de assessment de capabilities, sem `unknown-project` nem `federated-material changed`. R1 e R4 só eram necessários se a instância passasse a tratar roots como autorizados por descoberta; a delegação mantém os roots explícitos no executor da instância e não concede autoridade nova, então não há decisão de ADR a tomar.
- Consequences: nada a implementar; o maintainer escolheu este encaminhamento em 2026-11-17.

## 6. Validation

Sem validação própria: a cobertura é verificada pelos testes de `pose-hooks-delegate-to-the-instance-runner` (`TestAHookDelegatesToTheDeclaredRunner`, `TestAnInvalidHookRunnerIsRefused`, `TestHookStatusNamesTheRunner`) e pelo pre-commit do harne8 em 2026-11-17.

---

## 7. Final Report

### Delivered scope
Nada a entregar: coberto por `pose-hooks-delegate-to-the-instance-runner` (D1).

### Residual risks

Nenhum.

### Follow-ups

Nenhum.
