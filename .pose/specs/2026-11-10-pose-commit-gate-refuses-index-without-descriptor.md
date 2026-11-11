---
slug: pose-commit-gate-refuses-index-without-descriptor
status: done
created_at: 2026-11-10
completed_at: 2026-11-11
supersedes:
depends_on: pose-index-recovery-and-close-scope-consistency
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:commit-gate-refuses-index-without-descriptor
---

# Spec: O gate estrito recusa um conjunto de índices que o descritor não descreve

## 1. Intent

### Goal

`pose check` (o gate que o hook de commit roda) reporta, como erro no modo estrito e como aviso no tolerante, um conjunto de índices que o descritor da geração não descreve: a geração indisponível (arquivos ausentes ou diferentes do descritor) ou arquivos sob `.pose/indexes` que diferem da geração a que pertencem, e diz que o remédio é `pose index`.

### Business value

O harne8 ficou meses com 113 avisos porque o commit `e1593bde` regenerou `spec-graph.json` e `roadmaps.json` sem o `generation.json`, e nada no gate disse isso: cada review closeout dizia só "input closure is unknown". O gate agora nomeia a causa no primeiro commit que a cria, e o remédio funciona (spec `pose-index-recovery-and-close-scope-consistency`).

### Constraints

- Só a leitura estrutural do conjunto (a mesma de `pose index --verify` antes de recomputar entradas): o gate não recalcula as entradas dos índices, então um índice defasado em relação às fontes, o estado normal entre commits, não é reportado.
- Um repositório sem geração (legado ou nunca indexado) não é reportado.

### Non-goals

- Inspecionar o diff do commit: o gate lê o estado que o commit deixa.

## 2. Requirements

- R1: `pose check` no modo estrito reporta como erro "index set unavailable: <motivo>; run `pose index`" quando a geração publicada está indisponível, e "index files differ from generation <id>: <arquivos>; run `pose index`" quando arquivos do conjunto diferem da geração íntegra.
- R2: Um conjunto íntegro, um repositório sem geração e um conjunto reparado por `pose index` passam sem esse achado.
- R3: No modo tolerante o achado é um aviso.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/cli/check.go`.

### Artifacts

- modified: .pose/specs/2026-11-10-pose-commit-gate-refuses-index-without-descriptor.md
- created: .pose/starts/pose-commit-gate-refuses-index-without-descriptor.json
- created: .pose/changelogs/unreleased/pose-commit-gate-refuses-index-without-descriptor.md
- modified: pose-mcp/internal/cli/check.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md
- created: pose-mcp/internal/cli/check_index_generation_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- governance:commit-gate-refuses-index-without-descriptor module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`pose check` ganha o achado `index set unavailable` / `index files differ from generation`.

## 4. Tasks

- [x] O achado em `pose check`, em português e inglês
- [x] Um teste que falha sem ele (verificado): conjunto íntegro, indisponível, reparado, sem geração
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: rodar `pose index --verify` no gate
- Selected option: só a leitura estrutural do conjunto
- Rationale: o `--verify` recalcula as entradas (cerca de 20 s no harne8) e acusa defasagem, que é normal entre commits; o que o gate tem de impedir é o conjunto que nenhum leitor consegue ler
- Consequences: um índice defasado só é visto pelo `pose index --verify` e pelos closeouts
- Falsifier: um commit que deixa o conjunto indisponível e passa no gate estrito

## 6. Validation

### Strategy

O teste cria o estado que o commit do harne8 deixou (diretório da geração ausente e uma cópia legada diferente do descritor), vê o erro estrito com `pose index`, o aviso tolerante e a ausência do achado depois do reparo.

### Deterministic checks

`go test` do teste do rastreio e a matriz estrita em clone limpo.

### Requirement trace
- R1 [satisfied] test:TestCheckRefusesAnIndexSetThatItsDescriptorDoesNotDescribe check:check-index-generation-integration
- R2 [satisfied] test:TestCheckRefusesAnIndexSetThatItsDescriptorDoesNotDescribe
- R3 [satisfied] test:TestCheckRefusesAnIndexSetThatItsDescriptorDoesNotDescribe

### Known gaps
- O teste do modo tolerante confere que o achado é um aviso, não o código de saída do hook de cada instância; o motivo da indisponibilidade fica em inglês sob o locale pt-BR.

---

## 7. Final Report

### Delivered scope
`pose check` reporta o conjunto de índices que o descritor não descreve (R1 a R3).

### Files and modules changed
- pose-mcp/internal/cli/check.go e o teste; a matriz ganha `check-index-generation-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- Um índice defasado em relação às fontes continua sem aparecer no gate (por desenho).

### Follow-ups

- [open] Um `pose check` rodado enquanto o `pose index` publica pode ver uma divergência passageira e recomendar `pose index`: dar à mensagem o estado de publicação em andamento. (owner:@pose-maintainers crit:low review:2027-02-08)
- [open] Um hook de commit do harne8 que rode o `pose` com os roots explícitos, para que os avisos `unknown-project` das xref para `proj.pose-dist` deixem de aparecer. (owner:@pose-maintainers crit:low review:2027-02-08)
