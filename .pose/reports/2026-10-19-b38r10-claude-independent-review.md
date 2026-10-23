# Revisão independente B38r10 (Claude) — PR #198

Data: 2026-10-19. Repositório: oseiaspereira88/pose.
Spec: `pose-release-manifest-tag-integrity`.
HEAD: `59be226d1e377c35887439e0d2159dc78cd23f2b`.
Base: `origin/main` (`8aa729a76ab1c094690a0892263a86bf3bb7bb42`, igual ao merge-base).

**Decisão: changes-requested. Três achados medium bloqueiam o merge; quatro follow-ups
low novos são aceitáveis; os nove follow-ups low já registrados continuam `[open]`.**
As duas falhas high da r9 (R9-F01 e R9-F02) estão corrigidas no HEAD e foram reexercitadas.
Não houve seal, bundle, atestação, review verify, close, commit nem push. Não alterei
policy, pin de issuer, código de produção, testes versionados, tags nem releases deste clone.

## Identidade e separação declaradas

`agent:independent-claude-opus-5-5-review`, Claude Opus 5.5 (Claude/Anthropic).
O implementador também é Claude/Anthropic: **NÃO há separação de vendor**. A separação é
apenas de execução e de contexto, porque o Codex está indisponível (limite de uso até
2026-10-24). A identidade não é autenticada; a separação é declarada, não verificada.
No pose-dist as atestações não são assinadas. Como há bloqueantes, nenhuma atestação foi emitida.

O worktree já continha um rascunho r10 não rastreado e incompleto
(`2026-10-19-b38r10-independent-review.md` e `b38r10-head-validation.json`), com identidade
declarada Codex e marcadores `CHECKS_PENDING`/`WORKSPACE_PENDING`. Não o alterei nem o
tomei como prova: o achado R10-F01 dele foi reproduzido de forma independente abaixo,
e os demais achados deste parecer são próprios.

## Rules applied during review

- Tipo de mudança: bugfix de integridade de dados, com contratos de CLI/JSON/MCP.
- Skills: `pose-review` e `pose-spec-closeout`. Workflow: `.pose/workflows/review.md`.
- Regras: backend Go (erros, entradas e integração dos consumidores), `security.md`
  (integridade de dados, caminhos), `documentation-style.md`, `delivery-evidence.md`
  (alegação de entrega exige gate) e `knowledge-governance.md`.
- Lidos: spec, diff contra `origin/main` (22 arquivos Go, +3257/−159), commit de correção
  `36de909a` e os pareceres r2, r4–r9.

## Bloqueantes de merge

### R10-F01 — medium — Erro de scan da listagem de tags apaga tags do checker

Local: `pose-mcp/internal/cli/release_lifecycle.go:880`–`:900` (`appendReleasePolicyChecks`).
Requirements R3, R4, R7.

O checker percorre a saída de `git tag --list v*` com `bufio.Scanner` (linha máxima de
64 KiB) e não consulta `scanner.Err()`. Uma ref de tag com nome de 65.536 bytes ou mais
(possível em `packed-refs`) encerra o loop com `ErrTooLong`; essa tag e todas as que vêm
depois na ordem da listagem deixam de ser julgadas. É perda tratada como fim da lista.

Reprodução independente (fixture temporária com Git real e policy adotada com
`legacy_cutoff: v0.1.0`): uma tag legacy `v0.0.1-aaaa…` em `packed-refs`, seguida da tag
governada `v1.0.0`, sem manifest em lugar nenhum.

| Sufixo da tag legacy | Linhas listadas | `checker.errors` |
|---|---|---|
| 20 bytes | 2 | 1 (`governed tag v1.0.0 has no manifest in its tagged tree`) |
| 65.000 bytes | 2 | 1 |
| 65.536 bytes | 2 | **0, saída vazia** |
| 70.000 bytes | 2 | **0, saída vazia** |

O padrão é anterior ao PR (`origin/main` linhas 760–762), mas este delta promete que o
checker compartilha a recusa de tags sem snapshot (R7), e a variante viola essa promessa
com aprovação indevida. Os vizinhos (`governedReleaseWitness`, status, `releaseInputs`,
backfill) usam `strings.Fields` e não têm o defeito.

Ação mínima: usar `strings.Fields` como os vizinhos, ou transformar `scanner.Err()` em erro
do checker. Só aumentar o buffer move a fronteira. Adicionar regressão com tag longa legacy
antes de uma governada, e um controle abaixo do limite.

### R10-F02 — medium — Regressão: tag `v*` fora de SemVer reprova `pose check` em projeto sem policy

Local: `pose-mcp/internal/cli/release_lifecycle.go:935`–`:952` (`governedReleaseWitness`,
introduzida em `36de909a`), consumida em `:790`–`:796`. Constraint da spec ("não adotar
policy/capability automaticamente") e R6.

`git tag --list v*` também lista tags que não são versões de release (`v1.0`, `v2`,
`vendor-drop`). Para essas tags, `ReadReleaseSnapshot` → `ResolveReleaseTag` devolve
`"…" is not a release tag`, que não é `ErrReleaseSnapshotMissing`, então cai no `default`
e vira erro. Num projeto **sem** policy de release, isso vira:

```
✖ error release: the release policy is missing and whether the project adopted governed
releases could not be told: tag v1.0 could not be read: pose: "v1.0" is not a release tag
```

Reprodução: repositório simples, sem `.pose/policy/release.json` nem inventário, com uma tag.

| Tag | `checker.errors` |
|---|---|
| `v1.0` | **1** |
| `vendor-drop` | **1** |
| `v1.0.0` (controle, sem manifest nem policy no tree) | 0 |

`appendReleasePolicyChecks` roda sempre em `pose check` (`check.go:148`), e o modo strict
é o padrão. Na prática, todo projeto que nunca adotou releases governadas e tem uma tag
`v*` fora de SemVer passa a falhar no gate principal (inclusive no gate pós-install/update).
Isso impõe a governança de release a quem não a adotou. Não aprova nada indevidamente, mas
é uma regressão funcional deste HEAD sobre um contrato público, e não pode entrar como follow-up.

O mesmo tratamento de tag inválida como "governada" já existia no loop do checker em projetos
adotados (`versionLess(cutoff, "v1.0")` é verdadeiro porque um texto inválido ordena depois
de toda versão). Isso é conservador e anterior ao PR, mas diverge do status, que ignora essas
tags (`ValidateReleaseVersion(tag) != nil → continue`).

Ação mínima: filtrar com `posemodel.ValidateReleaseVersion(tag) == nil` antes de julgar, em
`governedReleaseWitness` e no loop do checker, como fazem status e `releaseInputs`. Adicionar
regressão com `v1.0`/`vendor-*` sem policy (quieto) e com policy adotada.

### R10-F03 — medium — Gitdir perdido é tratado como "não é repositório" (ausência)

Local: `pose-mcp/internal/cli/release_lifecycle.go:427`–`:434` (`gitUnavailableAt`),
`pose-mcp/internal/pose/release_lifecycle.go:630`–`:634` (`gitNotARepository`), consumidas em
`release_lifecycle.go` (cli) `:845`, `:926`, `:147`; status `pose/release_lifecycle.go:468`;
`ResolveReleaseTag` `release_manifest_integrity.go:288`; archival `release_archive.go:76`.
Requirements R3, R4, R7 (a mesma classe de R9-F01, variante 3).

Os dois helpers classificam como "sem repositório" qualquer falha cujo stderr contenha
`not a git repository`. Um `.git` que existe mas cujo gitdir sumiu (gitfile de worktree
apontando para um diretório podado ou movido, ou um `.git` corrompido) gera exatamente
`fatal: not a git repository: <gitdir>`. A perda do repositório vira então ausência de tags.

Reprodução: `adoptedTaggedRelease` (tag `v1.0.0` íntegra), inventário `.pose/releases` removido,
e `.git` trocado por um gitfile `gitdir: <dir inexistente>`.

| Variante | Com o `.git` original | Com o gitdir perdido |
|---|---|---|
| Policy adotada presente | errors=1 (`governed tag v1.0.0 is not a sound cut…`) | **errors=0, saída vazia** |
| Policy também removida | errors=1 (`…the manifest in the tree of tag v1.0.0 shows…`) | **errors=0, saída vazia** |

Pelo mesmo caminho, o status responde "nenhuma release" em vez de erro, e
`LoadArchivedFragments` trata a tag como inexistente (`boundToTag` continua verdadeiro).
Com o inventário presente, o journal (`tagged`/`published` sem tag) ainda produz um gap. Por isso
a variante exige duas perdas simultâneas, o que justifica medium e não high. Ainda assim, o
subgate aprova em silêncio uma perda que dá para distinguir. Os helpers são novos neste PR.

Ação mínima: só tratar "not a git repository" como ausência quando a ausência de `.git` for
provada (`ProvenAbsent` na raiz e nos ancestrais, ou o equivalente a `GIT_DIR`). Um `.git`
existente que o Git não consegue abrir é perda e tem de virar erro. Centralizar num único
helper usado pelo checker, status, `releaseInputs`, `ResolveReleaseTag` e archival.
Regressão: gitfile com gitdir perdido nas duas variantes acima, e controle de diretório sem `.git`.

## Follow-ups aceitáveis — low (novos)

Proposta: `[open]` com `(owner:@core crit:low review:2026-12-01)`. Não foram gravados
na spec, porque o parecer é bloqueante.

- **R10-L10 — producer ignora entradas de inventário que não são diretório:** `releaseInputs`
  (`cli/release_lifecycle.go:131`) pula com `entry.IsDir()` um `vX.Y.Z` que é link perdido ou
  arquivo ao calcular `previous`, de modo que o plan/prepare de uma versão menor é aceito. O checker
  e o status recusam essa entrada (`LoadReleaseManifest` falha / erro "not a directory"), então
  nada é aprovado. Variante do produtor, conservadora no gate.
- **R10-L11 — checker e `checkRelease` carregam os fragments pendentes com o loader leniente**
  (`LoadChangelogPolicy`, `cli/release_lifecycle.go:808` e `:371`): uma policy de changelog
  perdida vira o default. Os fragments só servem para detectar duplicidade, e um erro de carga vira
  erro/gap. Mesma classe do L05 (plan/prepare); corrigir junto.
- **R10-L12 — divergência de filtro de tags entre consumidores:** em projetos adotados, o checker julga
  tags `v*` fora de SemVer como governadas, e o status as ignora. É conservador e anterior ao PR.
  A correção de R10-F02 deve harmonizar os dois; se não fizer, isto fica como follow-up.
- **R10-L13 — `TagHoldsAdoptedReleasePolicy` deriva adoção só de `adopted_at != ""`** no tree
  da tag, sem validar a policy pelo contrato estrito. Um erro de leitura vira erro (não há
  aprovação); é só um ponto de validação de campo a alinhar com `LoadReleasePolicy`.

## Follow-ups já registrados (L01–L09)

Continuam `[open]` na spec com ownership. Nenhuma disposição foi alterada. Não os reexecutei
um a um; nesta rodada inspecionei os vizinhos de L05 e L09, e nada neles mudou a classificação low.

## Varredura por classe

| Classe | Consumidores inspecionados | Resultado |
|---|---|---|
| Perda tratada como ausência | `ProvenAbsent`; inventário em checker/witness/status/`releaseInputs`; listagem de tags (checker, witness, status, inputs, backfill); `ResolveReleaseTag`; `gitUnavailableAt`/`gitNotARepository`; `releaseTagSnapshotGaps`; `taggedReleaseWorktreeGaps`; `neverHadAnEvent`; `ReleaseCandidateSuperseded`; archival; publicclaims; obligations | F01 (scan parcial) e F03 (gitdir perdido) são bloqueantes; L10/L11 são variantes conservadoras. R9-F01 está corrigido: as falhas de inventário, de Git e de objeto viram erro. |
| Ordenação por texto | `CompareReleasePrecedence`, cutoff, `previous`, status, obligations, backfill | A precedência é SemVer; um texto inválido ordena depois das versões, o que faz o checker julgar tags inválidas (L12/F02). Nenhuma aprovação por ordem textual. |
| Campo sem validação | manifest, version evidence, policy no tree da tag | L01 e L13; R9-F02 corrigido (falha de I/O agora é gap). |
| Comparação de caminhos | `VersionSource.Path` limpo contra `source`, fragments sem separador, `ProvenAbsent` por ancestral | Nada novo. |
| Enumeração parcial | Scanner contra `strings.Fields` | Só o Scanner do checker (F01). |

## Checks executados (HEAD 59be226d)

- `go vet ./...` (pose-mcp): ok.
- `go test ./... -count=1` (pose-mcp): todos os pacotes ok.
- `go test -race ./internal/cli ./internal/pose -run 'Release|R[0-9]|F[0-9]|Archiv|Snapshot|Tag'`: ok.
- `pose validate --strict --module pose-mcp`: 137/137 pass, SUCCESS
  (`.pose/results/b38r10-claude-module-validation.json`, matrix `sha256:1fd3f2be…`).
- `pose check` (strict): SUCCESS com 30 warnings preexistentes.
- `pose artifact-check --spec pose-release-manifest-tag-integrity`: 47 claims, 1 change set.
- Probes independentes (cópia isolada em `/tmp`, sem tocar a árvore versionada):
  `.pose/results/b38r10-claude-independent-review.json`.

Os testes verdes não cobrem os três bloqueantes: nenhuma regressão registrada exercita tag longa,
tag fora de SemVer sem policy, nem gitdir perdido.

## Estado deixado

A spec continua in-progress, com `completed_at` vazio e nove follow-ups open. Arquivos não
rastreados criados por esta revisão: este parecer, `b38r10-claude-independent-review.json` e
`b38r10-claude-module-validation.json`. As modificações preexistentes em `.pose/assessments`,
`.pose/state` e `.pose/indexes` e o rascunho r10 anterior não foram tocados.
Contributor Mode: nenhum draft de contribuição nem issue foi criado; isso fica a critério do usuário.
