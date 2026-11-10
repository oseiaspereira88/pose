---
slug: pose-git-status-leaves-the-git-directory-alone
status: done
created_at: 2026-11-10
completed_at: 2026-11-10
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:git-status-leaves-the-git-directory-alone
---

# Spec: Perguntar ao Git quais caminhos mudaram não escreve em `.git`

## 1. Intent

### Goal

`gitChangedPaths` e `ReadGitStatusIn` (o `git status --porcelain` da integridade de snapshot) não escrevem nada sob `.git`: um índice com o cache de `stat` defasado fica como está, e a visão de leitura governada que observa `.git` não vê a própria pergunta como uma entrada que mudou durante a operação.

### Business value

Causa do que restava do aviso persistente do harne8, 2026-11-10: depois de reparar o conjunto de índices e a varredura de módulos (spec `pose-index-recovery-and-close-scope-consistency`), os 60 `review closeout` do `pose check` ainda saíam como erro com "review inputs changed during operation: .git: governance read view changed during capture". `git status` sem `GIT_OPTIONAL_LOCKS=0` refresca e reescreve o índice (cria e renomeia `index.lock` sob `.git`); os outros executores governados já definem a variável, o `snapshotGit` não. Com a correção, `pose check` do harne8 passa no modo estrito com 2 avisos (a idade do assessment).

### Constraints

- Só a variável de ambiente do executor muda: a saída do `git status` é a mesma.

### Non-goals

- Os demais executores do Git, que já definem `GIT_OPTIONAL_LOCKS=0`.

## 2. Requirements

- R1: `snapshotGit` executa o Git com `GIT_OPTIONAL_LOCKS=0`: `gitChangedPaths` num repositório cujo cache de `stat` está defasado não reescreve `.git/index` nem deixa `index.lock`.
- R2: Num checkout em que o resto do conjunto de índices é consistente, `review-check` de um escopo sem mudança relata `review.fresh=true` e o `pose check` estrito não reporta "governance read view changed during capture".

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/pose/git_snapshot_integrity.go`.

### Artifacts

- created: .pose/specs/2026-11-10-pose-git-status-leaves-the-git-directory-alone.md
- created: .pose/starts/pose-git-status-leaves-the-git-directory-alone.json
- created: .pose/changelogs/unreleased/pose-git-status-leaves-the-git-directory-alone.md
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- created: pose-mcp/internal/pose/git_status_leaves_git_dir_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- governance:git-status-leaves-the-git-directory-alone module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

Nenhuma: a saída é a mesma.

## 4. Tasks

- [x] Achar o que muda `.git` durante o review (os `git status` do snapshot, sem a variável)
- [x] A variável no executor e um teste que falha sem ela (verificado)
- [x] Conferir no harne8: `pose check` estrito passa com 2 avisos
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: filtrar `.git` da observação da visão de leitura
- Selected option: o executor não escreve
- Rationale: filtrar esconderia uma escrita real; os outros executores já pedem ao Git que não escreva
- Consequences: nenhuma
- Falsifier: um `git status` do snapshot que reescreve o índice

## 6. Validation

### Strategy

Um teste deixa o cache de `stat` do índice defasado (mesmo conteúdo, mtime diferente), pergunta os caminhos alterados e compara o índice antes e depois; falha sem a variável (verificado). A medição no harne8 real: 60 erros antes, 0 depois.

### Deterministic checks

`go test` do teste do rastreio e a matriz estrita em clone limpo.

### Requirement trace
- R1 [satisfied] test:TestGitChangedPathsWritesNothingUnderTheGitDirectory check:git-status-clean-git-integration
- R2 [satisfied] test:TestGitChangedPathsWritesNothingUnderTheGitDirectory

### Known gaps
- R2 é uma medição manual no harne8 (o `review-check` e o `pose check` estrito com o motor corrigido), não um teste de integração.

---

## 7. Final Report

### Delivered scope
O executor do `git status` do snapshot não escreve sob `.git` (R1); o `pose check` estrito do harne8 deixa de reportar 60 erros de "governance read view changed during capture" (R2).

### Files and modules changed
- pose-mcp/internal/pose/git_snapshot_integrity.go e o teste; a matriz ganha `git-status-clean-git-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- Outro executor do Git que ainda escreva sob `.git` produziria o mesmo sintoma; os que existem foram conferidos.

### Follow-ups

- [open] Um teste que percorre todos os executores do Git do motor e falha para o que não define `GIT_OPTIONAL_LOCKS=0`. (owner:@pose-maintainers crit:low review:2027-02-08)
