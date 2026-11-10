---
slug: pose-worktree-runtime-resident-cache
status: in-progress
created_at: 2026-11-09
completed_at:
supersedes:
depends_on: pose-optional-worktree-runtime, pose-governance-audit-skips-pure-git-reads
remediates:
priority: 2
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:worktree-runtime-resident-cache
---

# Spec: O runtime hospeda os checks e guarda as leituras puras do Git

## 1. Intent

### Goal

O runtime opcional da fatia 1 passa a rodar, para quem pede, os checks de leitura do projeto (`review-check`, `closeout-check`, `artifact-check`, `surface-check`, `roadmap-check`), com resultado tipado e fallback standalone, e a guardar entre um comando e o seguinte a saída das leituras do Git que são funções puras de ids de objeto ou do commit do HEAD.

### Business value

Fatia 2 da B35 (emenda do ADR de 2026-11-09), reescrita pela medição que as duas specs de engine anteriores fizeram: o cache de grafos da B33 não ajudava (o custo estava em recalcular a vinculação das entradas e em rodar o Git duas vezes), e depois delas o `review-check` deste repositório leva cerca de 25 s, dos quais cerca de 14 s são os 242 `git log -1 -- <path>`. Um processo residente que guarda essas leituras, chaveadas pelo commit, responde o segundo `review-check` em cerca de 10 s, com a mesma saída.

### Constraints

- Opt-in: nada inicia o runtime sem pedido, e `pose runtime run` roda standalone sempre que o runtime não existe, é incompatível ou recusa.
- O chamador não ganha autoridade: o runtime recusa, antes de começar, um comando que não hospeda e um chamador cujo digest de ambiente difere do dele.
- Um comando que pode ter rodado (prazo vencido, resposta perdida) nunca é repetido standalone sem pedido.

### Non-goals

- Admissão, QoS e medição de carga: fatia 3 (`pose-worktree-runtime-qos-and-measurement`); o runtime roda um comando hospedado por vez.
- Rotear os comandos para o runtime por padrão: `run` é explícito.
- Hospedar escritas: os cinco checks são leituras, ainda que possam atualizar arquivos derivados, como fazem standalone.

## 2. Requirements

- R5: O reuso residente guarda só o que é função pura de ids de objeto ou do commit do HEAD nomeado quando a leitura rodou (`governancePureGitRead`), chaveado por raiz, commit, estado raso e comando exato, com budget em bytes e a menos recentemente usada primeiro; fica desligado salvo num processo que o liga (o runtime); um commit novo é um miss; um worktree vinculado não é guardado; os bytes de um hit são admitidos contra o budget da operação como os de uma leitura nova e a auditoria final não aponta mudança.
- R7: Um comando hospedado devolve resultado tipado (`outcome` `completed`, `refused`, `deadline` ou `error`, código de saída, saída, época, tempo e acertos do memo); `refused` não iniciou o comando e leva ao fallback standalone; `deadline` e uma resposta perdida são `unknown` para o chamador, que não repete o comando e sai com 1 dizendo isso.
- R10: O runtime só hospeda os cinco checks e só roda um chamador cujo digest de ambiente (variáveis `POSE_`, `HARNE8_`, `GIT_`, `CLAUDE`, `CODEX`, o locale, `HOME`, `NO_COLOR`, o usuário) é o dele; a saída de um comando hospedado é a mesma que ele dá standalone.

## 3. Technical Plan

### Artifacts

- created: .pose/starts/pose-worktree-runtime-resident-cache.json
- created: .pose/changelogs/unreleased/pose-worktree-runtime-resident-cache.md
- modified: .pose/specs/2026-11-09-pose-worktree-runtime-resident-cache.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/governance_read_view_git.go
- created: pose-mcp/internal/pose/resident_git_memo.go
- created: pose-mcp/internal/pose/resident_git_memo_test.go
- modified: pose-mcp/internal/worktreeruntime/runtime.go
- modified: pose-mcp/internal/worktreeruntime/runtime_test.go
- created: pose-mcp/internal/cli/runtime_host.go
- created: pose-mcp/internal/cli/runtime_host_test.go
- modified: pose-mcp/internal/cli/runtime_cmd.go
- modified: pose-mcp/internal/cli/cli.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:worktree-runtime-resident-cache module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/cli.go

### API/contract changes

`pose runtime run [--envelope] [--timeout <d>] <command> [args...]` e o documento `pose.runtime.run/1`; o protocolo do runtime ganha a operação `run` e os campos de resultado tipado.

## 4. Tasks

- [x] Memo residente das leituras puras do Git, ligado só pelo runtime
- [x] Protocolo: operação `run`, resultado tipado, prazo e resposta grande
- [x] Handler hospedado: admissão, digest de ambiente, um por vez
- [x] `pose runtime run` com fallback standalone e sem repetir o que pode ter rodado
- [x] Medir: o segundo `review-check` no runtime contra standalone, com a saída comparada
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R5, R7, R10
- Minimal option: hospedar os comandos sem guardar nada (só plumbing)
- Selected option: hospedar os cinco checks e guardar as leituras puras do Git por commit
- Rationale: a medição mostrou que o custo restante é o Git; o que é função pura de um commit pode ser guardado sem prova de frescor por mtime, que o ADR rejeita, porque o commit é a chave
- Consequences: o runtime usa até 256 MiB; um clone raso aprofundado durante a vida do runtime muda um `log -1` sem mudar o HEAD (a chave inclui o estado raso lido do arquivo, não o histórico)
- Falsifier: uma saída hospedada diferente da standalone, ou um hit que devolve um commit que o Git não devolveria

## 6. Validation

### Strategy

Fixtures sintéticas: repositórios Git temporários, o runtime em processo, um semáforo de teste no lugar do comando para o prazo; a medição no `pose-dist` com o binário real.

### Deterministic checks

`go test -race` dos testes do rastreio; a matriz estrita em clone limpo.

### Requirement trace
- R5 [satisfied] test:TestTheResidentGitMemoIsOffByDefaultBoundedAndKeyedByTheCommit test:TestASecondOperationIsAnsweredFromTheMemoAndANewCommitMissesIt check:resident-cache-integration
- R7 [satisfied] test:TestAReplyThatArrivesInPartIsAnUnknownOutcome test:TestACommandRunningPastItsDeadlineStillCountsAsInFlight test:TestACommandThatMayHaveRunIsNotRepeatedStandalone test:TestAHostedCommandAnswersThroughTheRuntimeAsItDoesStandalone
- R10 [satisfied] test:TestTheRuntimeRefusesWhatItDoesNotHostAndAnotherEnvironment test:TestAHostedCommandAnswersThroughTheRuntimeAsItDoesStandalone test:TestTheEnvironmentDigestFollowsWhatDecidesAnAnswer

### Known gaps
- A medição (cerca de 25 s standalone, cerca de 10 s o segundo `review-check` no runtime, saída idêntica) foi feita à mão com o binário real, não num teste.
- A saída hospedada é a de um não terminal: um chamador num terminal recebe sem cor.
- O runtime não limita a carga: um comando por vez, sem admissão (fatia 3).

---

## 7. Final Report

### Delivered scope
O runtime hospeda `review-check`, `closeout-check`, `artifact-check`, `surface-check` e `roadmap-check` por `pose runtime run`, com resultado tipado, digest de ambiente, fallback standalone e sem repetir o que pode ter rodado (R7, R10), e guarda entre comandos as leituras puras do Git por commit (R5); o segundo `review-check` deste repositório levou cerca de 10 s contra cerca de 25 s standalone, com a mesma saída.

### Files and modules changed
- pose-mcp/internal/pose (memo residente e gancho na captura), worktreeruntime (protocolo), cli (hospedagem e `run`), ajuda e os quatro POSE.md; a matriz ganha `resident-cache-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- O digest de ambiente é uma lista: uma variável que decida uma resposta e não esteja nela faz o runtime responder com o ambiente dele.
- Um comando hospedado que atualiza arquivos derivados o faz com a identidade do processo do runtime, a mesma do usuário.

### Follow-ups

- [open] Test the admission of a memo hit's bytes against an operation's input budget when the budget is nearly spent. (owner:@pose-maintainers crit:low review:2027-02-08)
- [open] Put the user and global Git configuration in the memo key, as the repository's own is. (owner:@pose-maintainers crit:low review:2027-02-08)
- [open] Admissão, orçamento e proteção das consultas curtas contra checks pesados, com medições fixadas (fatia 3, `pose-worktree-runtime-qos-and-measurement`). (owner:@pose-maintainers crit:medium review:2027-02-08)
- [open] Rotear os checks para o runtime quando ele existe, sem `run` explícito, e renderizar para terminal. (owner:@pose-maintainers crit:low review:2027-02-08)
