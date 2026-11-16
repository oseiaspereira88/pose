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
- R1: When the project root is a Git repository whose HEAD has at least 500 commits, and its commit-graph is missing or any layer of it lacks changed-path Bloom filters, `pose doctor` shall report the check `git.commit-graph` at level `next`, with the command that writes it.
- R2: `pose doctor --fix --yes` shall remediate `git.commit-graph` by running `git commit-graph write --reachable --changed-paths` in the project root, after which the check shall report `ok`.
- R3: A repository with fewer than 500 commits on HEAD, or a commit-graph whose every layer carries the filters, shall report `ok`; a project root that is not a Git repository shall not report the check.
- R4: The pose-dist CI jobs that check out the full history shall write the commit-graph with changed paths before their gates.

### Non-functional
- A verificação lê só o cabeçalho e a tabela de chunks de cada camada do commit-graph.

### Security
- A correção roda o Git com argumentos fixos no root do projeto; nenhuma entrada do usuário entra no comando.

### Compatibility
- Saída do `doctor` ganha um check novo; o formato do relatório não muda.

---

## 3. Technical Plan

### Affected areas
- `cli/doctor.go`: o check `git.commit-graph` e a entrada no registro de correções.
- `cli/doctor_commit_graph.go`: a leitura do commit-graph e de suas camadas.
- `.github/workflows/ci.yml`: o passo que escreve o commit-graph.

### Artifacts
- created: pose-mcp/internal/cli/doctor_commit_graph.go
- created: pose-mcp/internal/cli/doctor_commit_graph_test.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: .github/workflows/ci.yml
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:doctor-recommends-the-commit-graph module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção pelo `pose doctor` e pelo `pose doctor --fix`. Check permanente `doctor-recommends-the-commit-graph-integration`, evidenceClass `integration`, required: `go test ./internal/cli -run 'TestDoctorRecommendsTheCommitGraph' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma no `.pose/`; a correção escreve o commit-graph sob `.git/objects/info/`.

### Technical risks
- Um formato futuro do commit-graph com outra assinatura é lido como sem filtros e recomendado de novo; a correção do Git o reescreve no formato atual.

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
- R1 [satisfied] test:TestDoctorRecommendsTheCommitGraph check:doctor-recommends-the-commit-graph-integration
- R2 [satisfied] test:TestDoctorRecommendsTheCommitGraph
- R3 [satisfied] test:TestDoctorRecommendsTheCommitGraph
- R4 [satisfied] O passo `Write the Git commit-graph with changed-path filters` nos dois jobs de `.github/workflows/ci.yml` que fazem checkout do histórico inteiro.

### Known gaps
- R3 no caso de um root que não é repositório Git é coberto pelo código (a leitura do estado falha e o check não é emitido), sem teste próprio.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

- [open] F1: medir e reduzir o `pose doctor` no harne8, que passa de 900 s em Go, sem processo Git filho, já com o binário anterior a esta spec. (owner:@pose-maintainers crit:medium review:2026-11-23)
