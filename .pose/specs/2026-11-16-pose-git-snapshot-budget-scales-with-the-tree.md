---
slug: pose-git-snapshot-budget-scales-with-the-tree
status: in-progress
created_at: 2026-11-16
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:git-snapshot-budget-scales-with-the-tree
---

# Spec: pose-git-snapshot-budget-scales-with-the-tree

## 1. Intent

### Goal
Fazer o orçamento de bytes das capturas de Git que comparam o disco com a árvore commitada crescer com o tamanho dessa árvore, em vez de ficar num teto fixo de 256 MiB.

### Business value
A árvore commitada do pose-dist passou de 256 MiB (256,8 MiB; 88 MiB de review runs e 102 MiB de review bundles). Desde então, todo `pose close` falha em `gitMaterializedChangesForIn` com "snapshot exceeds 256 MiB", e toda revisão delegada falha na limpeza do worktree descartável, que tira um `dispatchDiskSnapshot` do mesmo tamanho. Duas specs aprovadas ou em revisão (`pose-review-ledger-is-read-once-per-operation`, `pose-doctor-recommends-the-commit-graph`) ficaram sem poder fechar. As duas capturas precisam ler a árvore inteira para responder; um teto abaixo dela só pode reprovar.

### Constraints
- Um repositório pequeno não pode ganhar um teto menor que o de hoje.
- O disco que cresceu além da árvore commitada continua reprovando de forma fechada.
- Um teto absoluto continua existindo.

### Non-goals
- Reduzir o tamanho do repositório (retenção de registros de revisão antigos).
- Mudar o limite de 100.000 entradas.

---

## 2. Requirements

### Functional
- R1: `gitMaterializedChangesForIn` and `dispatchDiskSnapshot` shall bound the bytes they read by the committed tree's blob sizes plus a 256 MiB margin, so a clean tree of any size up to the ceiling is compared and snapshotted whole.
- R2: When the bytes read exceed that budget, both shall fail closed with an error that names the budget.
- R3: The budget shall never be less than the 256 MiB margin and never more than a 4 GiB ceiling; when the committed tree cannot be listed, the disk snapshot shall use the margin alone.

### Non-functional
- A listagem da árvore com tamanhos sai do mesmo `ls-tree` que a comparação já fazia; a captura de disco faz um `ls-tree -l` a mais.

### Security
- O teto absoluto de 4 GiB limita o trabalho de uma captura; o disco que cresceu além da árvore mais a margem continua reprovando.

### Compatibility
- Para um repositório de até 256 MiB commitados o teto só cresce; nada que passava passa a reprovar.

---

## 3. Technical Plan

### Affected areas
- `pose/git_snapshot_integrity.go`: o orçamento das duas capturas.

### Artifacts
- created: pose-mcp/internal/pose/git_snapshot_budget_test.go
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:git-snapshot-budget-scales-with-the-tree module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção pelo `pose close` (passo do plano de fechamento que procura mudanças escondidas) e pela limpeza de toda revisão delegada. Check permanente `git-snapshot-budget-scales-with-the-tree-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestGitSnapshotBudget' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Um repositório acima de 3,75 GiB commitados volta a reprovar no teto; isso fica para uma spec de retenção.

---

## 4. Tasks

### Implementation
- [x] Orçamento escalonado nas duas capturas (R1, R2, R3).

### Validation
- [x] Testes com margem reduzida, árvore maior que ela e disco crescido, com prova de reprovação.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-16
- Context: o maintainer escolheu, entre escalonar com a árvore, subir a constante para 1 GiB e retenção dos registros de revisão, escalonar com a árvore.
- Options considered: (a) orçamento proporcional à árvore commitada, com margem e teto; (b) constante fixa de 1 GiB; (c) retenção de registros antigos.
- Decision: (a), margem de 256 MiB e teto de 4 GiB.
- Basis: R1, R2, R3
- Rationale: (b) volta a quebrar quando a árvore passar de 1 GiB; (c) muda dado governado e é mais lenta para destravar; (a) segue o trabalho que a captura precisa fazer e mantém a reprovação para o disco que cresceu.
- Consequences: o custo de uma captura cresce com o repositório, como já crescia o trabalho necessário.

---

## 6. Validation

### Strategy
Testes com um repositório real de quatro arquivos de 4 KiB e a margem reduzida a 1 KiB, de modo que a árvore commitada passa da margem, como a do pose-dist passou de 256 MiB.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-16
- Environment: Linux, Go 1.26, Git 2.56.
- Notes: com o teto antigo (só a margem), o teste da árvore inteira reprova com "snapshot exceeds its budget of 1024 bytes"; com o orçamento escalonado passa, e um arquivo que cresce 60 KiB além da árvore reprova nas duas capturas. Os fechamentos de `pose-review-ledger-is-read-once-per-operation` e `pose-doctor-recommends-the-commit-graph` reprovaram com "snapshot exceeds 256 MiB" sobre a árvore de 256,8 MiB do pose-dist.

### Results summary
- Successes: as duas capturas cobrem uma árvore maior que a margem e continuam reprovando o disco crescido.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestGitSnapshotBudgetCoversTheCommittedTree check:git-snapshot-budget-scales-with-the-tree-integration
- R2 [satisfied] test:TestGitSnapshotBudgetStillBoundsGrowth
- R3 [satisfied] test:TestGitSnapshotBudgetIsBounded

### Known gaps
- O caso de a listagem da árvore falhar na captura de disco usa só a margem, sem teste próprio.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

- [open] F1: decidir a retenção dos registros de revisão, que somam 190 MiB dos 256,8 MiB da árvore do pose-dist e crescem a cada rodada. (owner:@pose-maintainers crit:medium review:2026-11-30)
