---
slug: pose-governance-audit-skips-pure-git-reads
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-index-input-binding-reads-each-transfer-once
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:audit-skips-pure-git-reads
---

# Spec: A auditoria final não reexecuta leituras do Git que não podem mudar

## 1. Intent

### Goal

A operação de leitura governada reexecuta, ao terminar, cada comando do Git que capturou, para provar que a saída não mudou durante a operação. Para os comandos cuja saída é função pura de ids de objeto, ou do commit que o HEAD nomeava quando rodaram, a reexecução não prova nada que o commit já não prove: ela é pulada enquanto esse commit se mantém, e roda como antes quando o HEAD se moveu.

### Business value

Continuação da medição de 2026-11-10 (spec `pose-index-input-binding-reads-each-transfer-once`): depois de ler os planos de transferência uma vez, o `review-check` deste repositório ainda executava cerca de 3100 comandos do Git, e cada um dos 242 `git log -1 -- <path>` (cerca de 57 ms cada) rodava duas vezes, uma na captura e outra na auditoria final. Pular a segunda leva o `review-check` de cerca de 52 s para cerca de 25 s, com a saída idêntica, para toda chamada, com ou sem runtime.

### Constraints

- O veredito da auditoria é o mesmo de antes em todo caso: o que ela pula não poderia ter mudado, e o que poderia mudar continua sendo reexecutado.
- Só as formas exatas listadas são puras. Nome de ref, `--all`, formato que nomeia refs ou qualquer flag que a classificação não conheça mantêm a reexecução.
- Os comandos que o `rev-parse --verify HEAD` observa continuam reexecutados: é por eles que o movimento do HEAD é visto.

### Non-goals

- Memoização entre chamadas num processo residente (a fatia 2 da B35).
- Mudar a captura, o orçamento de bytes ou a identidade da observação.

## 2. Requirements

- R1: Só `show <id40>:<path>`, `cat-file blob <id40>`, `ls-tree [-z|-r|-t|-l|--name-only|--full-tree] <id40> [-- <path>...]` (puros por id de objeto) e `log -1 --format=%H -- <path>...` (pura pelo commit do HEAD) são classificados como puros; qualquer outra forma não é.
- R2: Uma observação pura por id de objeto, bem-sucedida e sem stderr, não é reexecutada pela auditoria final.
- R3: Uma observação pura pelo HEAD é lida com o HEAD antes e depois do comando; só se os dois forem o mesmo ela é marcada pura com esse commit, e a auditoria a reexecuta se, e só se, o HEAD de agora for outro; nesse caso a mudança continua sendo encontrada.
- R4: A saída de `review-check` num repositório real é idêntica antes e depois, e o tempo diminui de modo medido.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/pose`: a captura e a auditoria da visão de leitura governada.

### Artifacts

- created: .pose/specs/2026-11-10-pose-governance-audit-skips-pure-git-reads.md
- created: .pose/starts/pose-governance-audit-skips-pure-git-reads.json
- created: .pose/changelogs/unreleased/pose-governance-audit-skips-pure-git-reads.md
- created: pose-mcp/internal/pose/governance_git_pure.go
- modified: pose-mcp/internal/pose/governance_read_view_git.go
- created: pose-mcp/internal/pose/governance_git_pure_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- governance:audit-skips-pure-git-reads module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

Nenhuma: a saída e o veredito da auditoria não mudam.

## 4. Tasks

- [x] Medir: os comandos do Git de um `review-check` rodam duas vezes (captura e auditoria)
- [x] Classificar as formas puras e marcar a observação
- [x] A auditoria pula a pura enquanto o commit se mantém e reexecuta quando o HEAD se moveu
- [x] Testes: classificação, contagem das reexecuções e detecção da mudança com o HEAD movido
- [x] Comparar a saída antes e depois e medir o tempo
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3
- Minimal option: manter a auditoria reexecutando tudo
- Selected option: pular a reexecução do que é função pura de ids de objeto, e do que é função pura do commit do HEAD enquanto o HEAD de agora é o mesmo de quando rodou
- Rationale: um objeto do Git não muda sob o seu id, e a saída de um `log -1 -- path` a partir de um commit é função do grafo abaixo dele; a reexecução só poderia divergir se o HEAD tivesse se movido, e isso é verificado por uma leitura do HEAD por raiz
- Consequences: o HEAD é lido duas vezes por leitura de histórico na captura (cerca de 4 ms contra 57 ms economizados) e uma vez por raiz na auditoria; um repositório que ganha histórico por `git fetch --deepen` durante a operação não é visto por essas leituras
- Falsifier: uma auditoria que deixa de apontar uma mudança que a reexecução apontaria, ou uma saída de `review-check` que difere

## 6. Validation

### Strategy

Um teste classifica os formatos aceitos e recusados; outro conta as reexecuções com um repositório real (nada muda: as puras não são reexecutadas e as demais são; o HEAD se move: a de histórico é reexecutada e a mudança é apontada). O teste de contagem falha quando a classificação é desligada (verificado).

### Deterministic checks

`go test` dos testes do rastreio e a matriz estrita em clone limpo; a comparação byte a byte da saída e o tempo foram medidos no `pose-dist`.

### Requirement trace
- R1 [satisfied] test:TestOnlyReadsThatCannotDependOnMutableStateAreClassifiedPure check:audit-pure-git-integration
- R2 [satisfied] test:TestTheFinalAuditSkipsPureReadsWhileTheirCommitStandsAndOnlyThen
- R3 [satisfied] test:TestTheFinalAuditSkipsPureReadsWhileTheirCommitStandsAndOnlyThen
- R4 [satisfied] test:TestTheFinalAuditSkipsPureReadsWhileTheirCommitStandsAndOnlyThen

### Known gaps
- R4 é uma medição feita à mão (52,3 s para 25,0 s no `review-check`, com o binário anterior e o novo no mesmo repositório, e saída sem diferença), não um teste de regressão; a carga da máquina entra nesses números.
- Um `git fetch --deepen` durante a operação muda o resultado de um `log -1 -- path` sem mover o HEAD; a auditoria não o vê para essas leituras.

---

## 7. Final Report

### Delivered scope
As formas puras do Git são classificadas e marcadas na captura, e a auditoria final as pula enquanto o commit que as determinou se mantém (R1-R3); o `review-check` deste repositório passou de cerca de 52 s para cerca de 25 s com a saída idêntica (R4).

### Files and modules changed
- pose-mcp/internal/pose: governance_git_pure.go e governance_read_view_git.go, mais os testes; a matriz ganha `audit-pure-git-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- O aprofundamento de um clone raso durante a operação (ver Known gaps).
- A classificação é uma lista de formas exatas: uma forma nova de leitura do Git mantém a reexecução até ser classificada.

### Follow-ups

- [open] Memoizar, no runtime residente da B35, as leituras puras do Git por commit entre chamadas. (owner:@pose-maintainers crit:low review:2027-02-08)
- [open] Dar teste de regressão à saída idêntica e ao tempo do `review-check`. (owner:@pose-maintainers crit:low review:2027-02-08)
