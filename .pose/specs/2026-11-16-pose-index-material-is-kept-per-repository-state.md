---
slug: pose-index-material-is-kept-per-repository-state
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-16
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates: spec:pose-check-shares-the-federated-material@defect-fix
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:index-material-is-kept-per-repository-state
---

# Spec: pose-index-material-is-kept-per-repository-state

## 1. Intent

### Goal
Fazer um processo do motor calcular o material federado de entrada do índice uma vez para cada estado dos repositórios autorizados, e não em cada operação de revisão.

### Business value
Toda operação de revisão confere se o índice que lê está em dia, e o material federado é uma das entradas dessa conferência. No harne8, com o pose-dist como dependência e o pin em dia, esse cálculo levou cerca de 90 s de um `closeout-check` de 140 s (perfil de CPU: `BuildIndexFederatedMaterialIn`, por `checkIndexInputs`, 71% do tempo). Um servidor MCP de vida longa pagava isso em toda chamada. Os clientes do `conductor` esperam de 20 a 120 s, e o `TestReviewConfirmationRealEngine` passou de 31 minutos e reprovou numa chamada `pose_review_bundle`. A spec `pose-check-shares-the-federated-material` já compartilha o material dentro de um `check`; esta estende o compartilhamento entre operações do mesmo processo.

### Constraints
- O material guardado vale só enquanto os repositórios estão no mesmo estado; qualquer commit, arquivo novo ou edição recalcula.
- Ler o estado não pode mudar nada que a read view observa.

### Non-goals
- Reduzir o custo da primeira construção do material, que fica para uma spec de paralelismo.

---

## 2. Requirements

### Functional
- R1: When an operation carries an explicit resolver, the engine shall keep, per process, the digest of the index's federated material for one state of the operation's root and of every root the resolver authorizes, and shall not build the material again for that state.
- R2: The state shall cover the engine version, the project, and for each root its HEAD, its full status (tracked, untracked and submodule entries) and the bytes of every path that status lists; any commit, new or removed file, or edit of a listed file, including a second edit of a dirty file, shall build the material again.
- R3: A store without an explicit resolver, or a state with more than 4,096 listed paths, shall build the material every time; a build that fails shall not be kept.
- R4: Reading the state shall not write to the repository: Git runs without optional locks, so the read view observes no change.

### Non-functional
- No harne8, a segunda chamada `pose_closeout_state` de um mesmo servidor MCP cai de cerca de 136 s para menos de 30 s, com o mesmo veredito.

### Security
- Nenhuma mudança de autorização: a chave cobre exatamente as raízes que o resolvedor autoriza.

### Compatibility
- Um processo de uma operação só calcula o material uma vez, como antes.

---

## 3. Technical Plan

### Affected areas
- `pose/index_material_kept.go`: o cache e a chave de estado.
- `pose/index_generation.go`: `IndexInputBindingForStoreIn` passa pelo cache para `federated-material`.

### Artifacts
- created: pose-mcp/internal/pose/index_material_kept.go
- created: pose-mcp/internal/pose/index_material_kept_test.go
- modified: pose-mcp/internal/pose/index_generation.go
- modified: .pose/adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:index-material-is-kept-per-repository-state module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção pelo servidor MCP e por toda operação de revisão que confere o índice. Check permanente `index-material-is-kept-per-repository-state-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestIndexMaterialIsBuiltOncePerRepositoryState|TestIndexMaterialFollowsEveryRepositoryState|TestIndexMaterialWithoutAResolverIsNotKept' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Um arquivo ignorado pelo Git que o material leia não entra na chave. O material lê specs, roadmaps, policies e registros de revisão, que são versionados.

---

## 4. Tasks

### Implementation
- [x] Cache por estado dos repositórios (R1, R2, R3).
- [x] Leitura do estado sem locks opcionais (R4).
- [x] Emenda D13 do ADR das leituras de governança.

### Validation
- [x] Testes do cache e de cada mudança de estado, com prova de reprovação; medição no harne8.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-16
- Context: o maintainer escolheu corrigir na origem o custo das chamadas MCP, em vez de subir os tetos dos clientes do conductor.
- Options considered: (a) memo do material com chave nas leituras que ele faz; (b) chave no estado de cada repositório autorizado (HEAD, status, bytes dos caminhos listados).
- Decision: (b), registrada como D13 do ADR.
- Basis: R1, R2, R3, R4
- Rationale: (a) precisaria saber de antemão tudo o que o material lê; (b) cobre qualquer leitura de um arquivo versionado ou não rastreado, e custa dois processos Git por raiz.
- Consequences: o material só é recalculado quando um repositório muda.

---

## 6. Validation

### Strategy
Testes com dois repositórios git reais e um resolvedor que autoriza os dois; medição de duas chamadas MCP seguidas no harne8.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-16
- Environment: Linux, Go 1.26; clone limpo do harne8 em cc7780b1 com o pose-dist em 7e46b1e.
- Notes: duas chamadas `pose_closeout_state` de `spec:harne8-abm-governed-execution` no mesmo servidor MCP: sem o cache, 138 s e 136 s; com ele, 138 s e 11 s, terminal nas quatro. A primeira versão lia o estado com `git status` comum, que regravava o índice do Git: a read view acusou ".git changed" e o fechamento deixou de ser terminal; com `GIT_OPTIONAL_LOCKS=0`, o veredito voltou. Sem o hash dos bytes, um arquivo sujo editado de novo reaproveitava o material, e o teste reprova.

### Results summary
- Successes: a segunda chamada de um servidor MCP fica 12 vezes mais rápida, com o mesmo veredito.
- Failures: nenhuma.
- Warnings: a primeira chamada ainda leva cerca de 138 s.

### Requirement trace
- R1 [satisfied] test:TestIndexMaterialIsBuiltOncePerRepositoryState check:index-material-is-kept-per-repository-state-integration
- R2 [satisfied] test:TestIndexMaterialFollowsEveryRepositoryState
- R3 [satisfied] test:TestIndexMaterialWithoutAResolverIsNotKept
- R4 [satisfied] O fechamento de harne8-abm-governed-execution continua terminal com o cache (log de execução).

### Known gaps
- R4 não tem teste automático próprio; foi provado pela medição no harne8.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

- [open] F1: reduzir o custo da primeira construção do material federado (cerca de 90 s no harne8), por exemplo verificando em paralelo as dependências de outro projeto; com ela, a primeira chamada de um servidor MCP caberia nos tetos dos clientes do conductor. (owner:@pose-maintainers crit:high review:2026-11-23)
