# Revisão independente B38r11 (Claude) — PR #198

Data: 2026-10-19. Repositório: oseiaspereira88/pose.
Spec: `pose-release-manifest-tag-integrity`.
HEAD: `13195669215461a9d269cd7396480d0bbcf85a66` (correção em `11595cda`).
Base: `origin/main` (`8aa729a76ab1c094690a0892263a86bf3bb7bb42`, igual ao merge-base).

**Decisão: changes-requested. Um achado medium bloqueia o merge (R11-F01). Cinco follow-ups
low novos são aceitáveis, e os treze follow-ups low já registrados (L01–L13) continuam `[open]`.**
R10-F01, R10-F02 e R10-F03 estão corrigidos no HEAD e foram reexercitados. A correção de R10-F02,
porém, não chegou ao consumidor vizinho da mesma classe, o status. Esse é o R11-F01.
Não houve seal, bundle, atestação, review verify, close, commit nem push. Não alterei policy,
pin de issuer, código de produção, testes versionados, tags nem releases deste clone.

## Identidade e separação declaradas

`agent:independent-claude-opus-5-5-review`, Claude Opus 5.5 (Claude/Anthropic).
O implementador também é Claude/Anthropic: **NÃO há separação de vendor**. A separação é
apenas de execução e de contexto, porque o Codex está indisponível (limite de uso até
2026-10-24). A identidade não é autenticada: a separação é declarada, não verificada.
No pose-dist as atestações não são assinadas. Como há bloqueante, nenhuma atestação foi emitida.

## Rules applied during review

- Tipo de mudança: bugfix de integridade de dados, com contratos de CLI/JSON/MCP e de obligations.
- Skills: `pose-review` e `pose-spec-closeout`. Workflow: `.pose/workflows/review.md`.
- Regras: backend Go, `security.md` (integridade de dados e caminhos), `documentation-style.md`,
  `delivery-evidence.md` (alegar entrega exige gate) e `knowledge-governance.md`.
- Lidos: a spec, o diff contra `origin/main` (22 arquivos Go, +3330/−163), o commit `11595cda`
  e o parecer r10.

## Correções da r10 conferidas

| Achado | Correção no HEAD | Conferência |
|---|---|---|
| R10-F01 | `strings.Fields` no laço de tags do checker | Regressão `TestR10ALongTagNameDoesNotStopTheTagLoop` verde |
| R10-F02 | `governedReleaseWitness` filtra com `ValidateReleaseVersion` | Corrigido na testemunha; o status, vizinho da mesma classe, ficou de fora (R11-F01) |
| R10-F03 | `GitNotARepository` só responde "sem repositório" se nenhum ancestral tem `.git` (`Lstat`) | `TestR10ALostGitdirIsNotNoRepository` verde. O mesmo helper serve checker, status, `releaseInputs`, `ResolveReleaseTag` e archival |

## Bloqueante de merge

### R11-F01 — medium — Em projeto não adotado, o status lista toda tag SemVer como release "prepared" com gap

Local: `pose-mcp/internal/pose/release_lifecycle.go:464`–`:505` (laço de tags de `GetReleaseStatus`,
introduzido para R7-F01). Requirements R7 (checker/status/CLI/MCP compartilham a verificação) e R6.
Constraint da spec: "não adotar policy/capability automaticamente".

O laço que faz o status enxergar uma release cuja pasta saiu do worktree não depende de adoção.
Ele roda em todo projeto com tags `v*`, e o cutoff vem da policy (vazio no scaffold padrão). O checker
sai cedo quando `adopted_at == ""` (`cli/release_lifecycle.go:798`), mas o status não. Num projeto que
nunca adotou releases governadas, cada tag SemVer comum (`v1.0.0`, `v1.1.0`…) vira então
`{"state":"prepared","gaps":["tag … exists, its release directory is missing … no readable snapshot"]}`.

Essa projeção se propaga para os consumidores:

- `releaseQueueObligations` cria uma obligation `release-prepared` por tag, com a condição
  "the release commit is tagged `vX`", que é falsa, porque a tag existe. Elas aparecem em
  `pose state --attention` e na tool MCP de obligations.
- `pose index` grava cada tag em `.pose/indexes/releases.json`.
- `public-claims` informa `candidate_state: prepared` e um `prepared_source` apontando para um
  manifest que não existe.
- `pose release status` e a tool MCP `pose_release_status` respondem com essas releases.

Reprodução (binários compilados do HEAD e do `origin/main`; `pose install` num repositório novo,
policy do scaffold com `adopted_at: ""`, um commit e tags `v1.0.0`, `v1.1.0`, `v2.0.0`):

| Medida | HEAD | origin/main |
|---|---|---|
| Releases em `pose release status --json` | **3** (todas `prepared` com gap) | 0 |
| Obligations `release-*` em `pose state --attention` | **3** | 0 |
| Entradas em `.pose/indexes/releases.json` | **3** | 0 |
| `pose check` | SUCCESS | SUCCESS |
| Com 303 tags: obligations / tempo do status | **303 / 1650 ms** | 0 / 6 ms |

Sonda Go (cópia isolada): com policy não adotada e tag `v1.0.0`, o checker dá `errors=0`, e o status
dá `[v1.0.0:prepared]`. No controle adotado, o checker dá `errors=1`, e o status dá o mesmo.
Checker e status discordam, portanto, sobre o mesmo projeto não adotado.

Isso não aprova nada indevidamente: os efeitos das obligations são advisory e nenhum gate fica verde
por causa disso. Mesmo assim, é uma regressão funcional do HEAD sobre contratos públicos (status
JSON, MCP, obligations, índice). Atinge o caso mais comum, um projeto com POSE instalado e tags de
versão que nunca adotou releases governadas, e impõe a ele a governança de release. É a mesma classe
e a mesma régua de R10-F02, que foi classificado como bloqueante. Por isso não entra como follow-up.
O custo linear em tags (cerca de 8 processos Git por tag em todo status, obligations e índice) agrava
o efeito em repositórios com muitas tags.

Ação mínima: aplicar no laço de tags do status o mesmo critério de adoção do checker, de preferência
por um predicado único do domínio usado pelos dois e pela testemunha. Esse critério é: policy adotada;
ou policy perdida ou ilegível (conservador); ou policy ausente com uma testemunha de governança
(`governedReleaseWitness`). Policy presente e não adotada não lista tags que não estão no inventário.
Regressões necessárias: policy do scaffold não adotada com tags SemVer (status vazio e zero obligations),
controle adotado (a tag continua julgada e listada) e policy perdida (a tag continua listada).

## Follow-ups aceitáveis — low (novos)

Proposta: `[open]` com `(owner:@core crit:low review:2026-12-01)`. Não foram gravados na spec,
porque o parecer é bloqueante.

- **R11-L14 — ref de tag corrompida resolve como "tag inexistente".** Uma ref solta com lixo gera
  só o aviso `ignoring broken ref` do Git, e uma linha malformada em `packed-refs` some em silêncio.
  `rev-parse --verify --quiet` sai com 1, e `ResolveReleaseTag` devolve `ErrReleaseTagMissing`. Com
  `adoptedTaggedRelease`, o status fica `prepared` sem gaps e o checker dá `errors=0` (também com o
  inventário removido). É low porque exige corromper o armazenamento de refs do Git, e o estado só
  desce (sem tag): um journal que registra `tagged` continua reprovando. Ação: tratar o aviso
  `ignoring broken ref` no stderr como falha de leitura.
- **R11-L15 — `GitNotARepository` ignora `GIT_DIR`.** Com `GIT_DIR` apontando para um gitdir perdido e
  nenhum `.git` nos ancestrais, a resposta é "sem repositório" (com o inventário removido, `errors=0`).
  A perda está no ambiente do processo, não no projeto, e exige duas perdas.
- **R11-L16 — a testemunha reprova `pose check` por uma tag SemVer que aponta para árvore ou blob**
  (por exemplo `v2.6.11-tree`, como no git.git e no kernel). Isso dá `ErrReleaseTagNotCommit`, que vira
  "could not be told". Falha fechada, com forma de tag rara, e só num projeto sem o arquivo de policy, que já
  reprova pela referência quebrada do scaffold. Harmonizar com R11-F01 e L12.
- **R11-L17 — `previous_version` é validado, mas não é derivado de novo nem divulgado como limite.** Só uma
  edição deliberada que também recalcula `release_input_digest` escapa, e nenhum estado sobe por isso.
- **R11-L18 (anterior ao PR) — `release check` descarta `ForeignFragmentClaims` em silêncio** quando
  o archive ou as specs não são lidos (`cli/release_lifecycle.go:462`–`:466`). A perda do archive já é
  gap de `checkRelease`, e `pose check` mantém a verificação estrutural de claims.

## Follow-ups já registrados (L01–L13)

Continuam `[open]` na spec com ownership, e nenhuma disposição foi alterada. Reconfirmei L12 nesta
rodada: com policy adotada, a tag `v1.0` reprova o checker e o status a ignora. É uma falha conservadora,
anterior ao PR.

## Varredura por classe

| Classe | Consumidores inspecionados | Resultado |
|---|---|---|
| Perda tratada como ausência | `ProvenAbsent`; `LoadReleasePolicy`; `loadReleaseFragments`; `LoadReleaseEvents`; `ReleaseFilesGaps`; inventário em checker, testemunha, status e `releaseInputs`; listagem de tags (checker, testemunha, status, inputs, backfill); `GitNotARepository`; `ResolveReleaseTag`; `gitTreeEntry` e ancestrais; `releasePolicyAt`; `changelogPolicyAt`; `releaseTagState`/`releaseTagSnapshotGaps`; `neverHadAnEvent`; `ReleaseCandidateSuperseded`; `LoadArchivedFragments` (`boundToTag`); `cmdReleaseRecord`/`prepare`/`check`; publicclaims; obligations; índice | R10-F03 está corrigido. As variantes restantes são L14 (ref corrompida) e L15 (`GIT_DIR`), que só descem estado. `LoadArchivedFragments` sem inventário só retira testemunhos, o que é conservador. Uma tag que não resolve por falha do Git num corte superseded é pega pelo laço de tags, porque a ref aparece na listagem. |
| Ausência confundida com presença (vizinha) | Laço de tags do status contra o gate de adoção do checker | **R11-F01** |
| Ordenação por texto | `CompareReleasePrecedence`, cutoff (status e checker usam a mesma comparação), `previous`, ordenação do status, open-next, backfill, ledger de archival | O ledger recusa duas archivals da mesma spec, então não depende de ordem. Os sorts restantes são de apresentação. Nada novo. |
| Campo sem validação | Manifest raiz e aninhado, `version_evidence`, `previous_version`, policy na árvore da tag | L01, L13 e L17 |
| Comparação de caminhos | `VersionSource.Path` limpo contra `source`, paths de fragment sem separador, `ProvenAbsent` com link no ancestral, `GitNotARepository` lógico contra físico | O caso lógico contra físico falha fechado. Nada novo. |

## Checks executados (HEAD 13195669)

- `go vet ./...` (pose-mcp): ok.
- `go test ./... -count=1` (pose-mcp): todos os pacotes ok.
- `go test -race -count=1 ./internal/cli ./internal/pose -run 'Release|R[0-9]|F[0-9]|Archiv|Snapshot|Tag'`: ok.
- `pose validate --strict --module pose-mcp`: 137/137 pass, SUCCESS
  (`.pose/results/b38r11-claude-module-validation.json`).
- `pose check` (strict): SUCCESS com 30 warnings preexistentes.
- `pose artifact-check --spec pose-release-manifest-tag-integrity`: 47 claims, 1 change set.
- `pose release status` deste repositório: 75 releases. Só `v6.3.0` tem gap (corte superseded, já conhecido).
- Marcadores de dívida (`TODO`, `FIXME`, `panic(`, `stub`) nas linhas adicionadas do diff Go: 0.
- Sondas independentes em cópia isolada (`git archive HEAD` em `/tmp`) e binários do HEAD e do
  `origin/main`. Os resultados estão em `.pose/results/b38r11-claude-independent-review.json`.

Os testes verdes não cobrem R11-F01: nenhuma regressão exercita o status, as obligations ou o índice
de um projeto com policy não adotada e tags SemVer.

## Estado deixado

A spec continua in-progress, com `completed_at` vazio e treze follow-ups open. Os arquivos não rastreados
criados por esta revisão são este parecer, `b38r11-claude-independent-review.json` e
`b38r11-claude-module-validation.json`. O `pose artifact-check` regravou
`.pose/indexes/delivery-integrity.json` como efeito colateral, e restaurei esse arquivo ao conteúdo do HEAD.
A modificação preexistente em `.pose/indexes/generation.json` não foi tocada.
Contributor Mode: não criei nenhum rascunho de contribuição nem issue. Fica a critério do usuário se a fricção
de `artifact-check` regravar um índice versionado merece um rascunho.
