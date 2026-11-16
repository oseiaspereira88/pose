---
slug: pose-doctor-recommends-the-commit-graph
status: in-progress
created_at: 2026-11-16
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: feature
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:doctor-recommends-the-commit-graph
---

# Spec: pose-doctor-recommends-the-commit-graph

## 1. Intent

### Goal
Fazer o `pose doctor` recomendar, e o `pose doctor --fix` escrever, o commit-graph do Git com filtros de Bloom de caminhos alterados num projeto cujo histórico é grande o bastante para pesar, e fazer o CI do pose-dist escrevê-lo antes dos gates.

### Business value
A spec `pose-review-ledger-is-read-once-per-operation` mediu, na primeira chamada `pose_closeout_state` de um servidor MCP no harne8, 1.116 comandos `git log -1 --format=%H -- <path>` somando 81 s: cada um percorre o histórico do pose-dist (3.528 commits) até achar o último commit que tocou o arquivo. Com o commit-graph com caminhos alterados (`git commit-graph write --reachable --changed-paths`), cada consulta cai de 47 ms para 2 ms, sem mudar a saída do Git, e a primeira chamada cai de 136 s para 59 s. É manutenção do repositório, que o motor não faz sozinho numa leitura de governança; o `doctor` é o lugar onde o POSE já recomenda e aplica remediações seguras.

### Constraints
- A verificação só lê arquivos do Git; não roda nada que escreva.
- A correção só escreve sob o diretório de objetos do Git, é idempotente e reversível (apagar o arquivo devolve o estado anterior), como as demais do registro do `doctor --fix`.
- Uma instalação nova, com histórico pequeno, continua sem passos pendentes.

### Non-goals
- Escrever o commit-graph durante uma operação de governança.
- Manter o commit-graph atualizado a cada commit (o `git gc` e o `git maintenance` já o reescrevem preservando os filtros).

---

## 2. Requirements

### Functional
- R1: When the project root is a Git repository whose HEAD has at least 500 commits, and Git's own trace of a last-commit lookup by path shows no changed-path Bloom filter consulted, `pose doctor` shall report the check `git.commit-graph` at level `next`, with the command that writes it; the judgement follows whichever commit-graph Git loads (its own object directory or an alternate, a single file or a chain).
- R2: `pose doctor --fix --yes` shall remediate `git.commit-graph` by running `git commit-graph write --reachable --changed-paths --max-new-filters=-1` in the project root, so `commitGraph.maxNewFilters` cannot leave commits without filters, after which the check shall report `ok`.
- R3: A repository with fewer than 500 commits on HEAD, or one whose last-commit lookup Git answers from the filters, shall report `ok`; a project root that is not a Git repository, has no commit on HEAD, has a shallow history or has no file in HEAD's tree shall not report the check; Git warnings printed by a successful lookup, such as an unreadable commit-graph, shall report `warn`; no diagnostic command shall fetch from a remote; any other failure to read the state shall report `warn` with a bounded description of the failure; when Git configuration keeps it from reading commit-graphs or their changed-path filters, `pose doctor` shall report `git.commit-graph-reads` at `warn`, naming the setting, instead of recommending a rewrite.
- R4: The pose-dist CI jobs that check out the full history shall write the commit-graph with changed paths before their gates.

### Non-functional
- A verificação roda três comandos Git de leitura e uma busca do último commit de um arquivo com o trace2 ligado.

### Security
- A correção roda o Git com argumentos fixos no root do projeto; nenhuma entrada do usuário entra no comando.

### Compatibility
- Saída do `doctor` ganha um check novo; o formato do relatório não muda.

---

## 3. Technical Plan

### Affected areas
- `cli/doctor.go`: o check `git.commit-graph` e a entrada no registro de correções.
- `cli/doctor_commit_graph.go`: a contagem do histórico e a pergunta ao Git, pelo trace2, se ele usa os filtros.
- `.github/workflows/ci.yml`: o passo que escreve o commit-graph.

### Artifacts
- created: pose-mcp/internal/cli/doctor_commit_graph.go
- created: pose-mcp/internal/cli/doctor_commit_graph_test.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: .github/workflows/ci.yml
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:doctor-recommends-the-commit-graph module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção pelo `pose doctor` e pelo `pose doctor --fix`. Check permanente `doctor-recommends-the-commit-graph-integration`, evidenceClass `integration`, required: `go test ./internal/cli -run 'TestDoctorRecommendsTheCommitGraph|TestDoctorCommitGraph' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma no `.pose/`; a correção escreve o commit-graph sob `.git/objects/info/`.

### Technical risks
- O evento `bloom` do trace2 é saída de diagnóstico do Git; uma versão que o renomeie faria o check recomendar a correção sempre. A correção é inofensiva, e o teste de integração com o Git real reprova se o evento sumir.

---

## 4. Tasks

### Implementation
- [x] Check e correção `git.commit-graph` no `doctor` (R1, R2, R3).
- [x] Passo do CI (R4).

### Validation
- [x] Testes com repositórios reais acima e abaixo do limite, com e sem filtros, com prova de reprovação.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-16
- Context: o follow-up F1 da spec `pose-review-ledger-is-read-once-per-operation`; o maintainer pediu para escrever o commit-graph nos checkouts dele e seguir com o F1.
- Options considered: (a) o motor escreve o commit-graph numa leitura de governança; (b) o `doctor` recomenda e o `doctor --fix` escreve; (c) só documentar.
- Decision: (b), com o limite de 500 commits.
- Basis: R1, R2, R3
- Rationale: (a) faria uma leitura escrever no repositório, o que a read view trata como mudança de entrada; (c) não chega a quem não lê o manual; (b) usa o caminho de remediação segura que o POSE já tem. Abaixo de cerca de 500 commits a busca de histórico custa poucos milissegundos.
- Consequences: um projeto grande vê um passo pendente até rodar a correção.

---

## 6. Validation

### Strategy
Testes do `doctor` com repositórios Git reais; medição já registrada na spec de origem.

### Deterministic checks

#### Test
- Command: `go test ./internal/cli -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-16
- Environment: Linux, Go 1.26, Git 2.56.
- Notes: o teste monta uma instalação real, cria histórico acima de um limite reduzido e confere o check sem commit-graph (`next`, corrigível), depois da correção (`ok`), com um commit-graph sem filtros (`next`) e com um commit-graph dividido em camadas com filtros (`ok`); abaixo do limite, `ok`. Reprova quando a leitura ignora os chunks BIDX/BDAT, quando ignora a cadeia de camadas e quando a correção não está registrada. O guarda `TestEngineStartsGitOnlyThroughTheRawReader` reprovou a primeira versão, que chamava o Git direto; ela passou a usar `RawHistoryGit`. No checkout do pose-dist do maintainer, com o commit-graph escrito, o check dá `ok`. O `pose doctor` no harne8 passa de 900 s com o binário instalado antes desta spec, e por isso não foi medido lá.

### Results summary
- Successes: o `doctor` recomenda e escreve o commit-graph; o CI o escreve antes dos gates.
- Failures: nenhuma.
- Warnings: o `pose doctor` no harne8 leva mais de 900 s, já antes desta spec (follow-up F1).

### Requirement trace
- R1 [satisfied] test:TestDoctorRecommendsTheCommitGraph test:TestDoctorCommitGraphProbesALiteralPath check:doctor-recommends-the-commit-graph-integration
- R2 [satisfied] test:TestDoctorRecommendsTheCommitGraph test:TestDoctorCommitGraphFixIgnoresMaxNewFilters
- R3 [satisfied] test:TestDoctorRecommendsTheCommitGraph test:TestDoctorCommitGraphReportsAnUnreadableHistory test:TestDoctorCommitGraphTellsUnbornFromBroken test:TestDoctorCommitGraphNamesDisabledReads test:TestDoctorCommitGraphSkipsAShallowClone test:TestDoctorCommitGraphSkipsAnEmptyTree test:TestDoctorCommitGraphFollowsConfigurationPrecedence test:TestDoctorCommitGraphNeverFetches test:TestDoctorCommitGraphReportsAnUnreadableGraph
- R4 [satisfied] O passo `Write the Git commit-graph with changed-path filters` nos dois jobs de `.github/workflows/ci.yml` que fazem checkout do histórico inteiro.

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

- [open] F1: medir e reduzir o `pose doctor` no harne8, que passa de 900 s em Go, sem processo Git filho, já com o binário anterior a esta spec. (owner:@pose-maintainers crit:medium review:2026-11-23)

### Review

Rodada 1 (run rrn-237e7d9be5f0c4f5, changes-requested): 4 bloqueadores, corrigidos. F2 a leitura ignorava diretórios de objetos alternativos; F3 preferia a cadeia de camadas ao arquivo único, ao contrário do Git; F4 aceitava cabeçalhos de versão não suportada e arquivos truncados. Os três vinham de reimplementar a leitura do commit-graph; o `doctor` deixou de ler os arquivos e passou a perguntar ao próprio Git: roda `git log -1 -- <path>` com `GIT_TRACE2_EVENT` e procura o evento `bloom`/`statistics` com algum filtro consultado, de modo que vale o grafo que o Git carrega, com as regras dele. O teste cobre um grafo só no alternate (`clone --shared`) e o arquivo único sem filtros ao lado de uma cadeia com filtros (`next`, porque o Git carrega o arquivo único). F5 qualquer erro de leitura suprimia o check; agora só a falta de repositório ou de commit em HEAD o suprime, e as demais falhas viram `warn` com a descrição limitada (`TestDoctorCommitGraphReportsAnUnreadableHistory`, com um commit pai apagado).

Rodada 2 (run rrn-c7c3feb56ea14783, recusada porque os ids dos achados novos traziam o prefixo de rodada `R2-`; o rascunho declarou F2 a F4 corrigidos e F5 parcial). Os achados, corrigidos com teste que reprova sem a correção: F6 qualquer falha do primeiro `rev-parse` era tratada como "não se aplica"; agora só um diretório que não é repositório (`rev-parse --git-dir` com a mensagem do Git, lida no locale C) e um HEAD que não resolve (saída 1 de `rev-parse --verify --quiet HEAD`) suprimem o check, e um HEAD cujo commit não pode ser lido, ou um repositório que o Git não abre, é `warn` (`TestDoctorCommitGraphTellsUnbornFromBroken`). F7 o caminho sondado era lido como pathspec, e um nome como `*` mudava a busca; agora a busca roda com `GIT_LITERAL_PATHSPECS=1` (`TestDoctorCommitGraphProbesALiteralPath`). F8 com `core.commitGraph=false`, `commitGraph.readChangedPaths=false` ou `commitGraph.changedPathsVersion=0`, reescrever o grafo não mudava a resposta; agora o `doctor` reporta `git.commit-graph-reads` em `warn`, sem correção automática, nomeando a configuração, e não recomenda reescrever o grafo (`TestDoctorCommitGraphNamesDisabledReads`).

Rodada 5 (run rrn-6d639d1e28201081, changes-requested; as rodadas 3 e 4 falharam na limpeza do worktree pelo teto de 256 MiB, mas a 3 registrou F9 a F14). Seis bloqueadores, F15 a F20, que reabrem F9 a F14, corrigidos com teste que reprova sem a correção. F15 um clone raso, em que o Git não usa nem escreve commit-graph, recebia uma correção que não resolve; agora não se aplica (`TestDoctorCommitGraphSkipsAShallowClone`). F16 a correção herdava `commitGraph.maxNewFilters`; agora passa `--max-new-filters=-1` (`TestDoctorCommitGraphFixIgnoresMaxNewFilters`). F17 uma árvore de HEAD vazia recomendava reescrever para sempre; agora não se aplica (`TestDoctorCommitGraphSkipsAnEmptyTree`). F18 `readChangedPaths` era lido antes de `changedPathsVersion`, que o Git trata como prioritário; a ordem segue o Git (`TestDoctorCommitGraphFollowsConfigurationPrecedence`). F19 os comandos de diagnóstico podiam buscar objetos num clone parcial; agora rodam com `GIT_NO_LAZY_FETCH=1`, e a falta de objetos vira `warn` sem contato com o remoto (`TestDoctorCommitGraphNeverFetches`, que conta os objetos antes e depois). F20 o stderr de um comando bem-sucedido era descartado, e um commit-graph ilegível, que o Git anuncia e contorna, virava recomendação; agora é `warn` com a mensagem do Git (`TestDoctorCommitGraphReportsAnUnreadableGraph`).

