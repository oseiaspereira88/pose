# Revisão independente B38r12 (Claude) — PR #198

Data: 2026-10-19. Repositório: oseiaspereira88/pose.
Spec: `pose-release-manifest-tag-integrity`.
HEAD: `75a0a9ed` (correção da r11 em `8282b126`). Base: `origin/main`.

**Decisão: changes-requested. Um achado medium bloqueia o merge (R12-F01), e há um follow-up low
novo (R12-L19). Os dezoito follow-ups low já registrados continuam `[open]`.**
R11-F01 está corrigido no caso que a r11 reproduziu (policy legível e não adotada). A correção,
porém, trata como "nunca adotado" uma policy de hoje que foi perdida ou que está ilegível. É
exatamente a classe "perda tratada como ausência", e o próprio comentário do código diz que isso
não deveria acontecer. Não houve seal, bundle, atestação, review verify, close, commit nem push.
Não alterei policy, pin de issuer, código de produção, testes versionados, tags nem releases.

## Identidade e separação declaradas

`agent:independent-claude-opus-5-5-review`, Claude Opus 5.5 (Claude/Anthropic).
O implementador também é Claude/Anthropic: **NÃO há separação de vendor**. A separação é só de
execução e de contexto, porque o Codex está indisponível (limite de uso até 2026-10-24). A
identidade não é autenticada: a separação é declarada, não verificada. No pose-dist as atestações
não são assinadas. Como há bloqueante, nenhuma atestação foi emitida.

## Rules applied during review

- Tipo de mudança: bugfix de integridade de dados, com contratos de CLI/JSON/MCP e de obligations.
- Skills: `pose-review` e `pose-spec-closeout`. Workflow: `.pose/workflows/review.md`.
- Regras: backend Go, `security.md` (integridade de dados), `documentation-style.md`,
  `delivery-evidence.md` (alegar entrega exige gate) e `knowledge-governance.md`.
- Lidos: a spec, o diff contra `origin/main`, o commit `8282b126` e os pareceres r10 e r11.

## Correção da r11 conferida

| Achado | Correção no HEAD | Conferência |
|---|---|---|
| R11-F01 | Laço de tags do status pula a tag sem manifest quando `!adopted` e a árvore da tag não tem policy adotada | Corrigido para policy legível e não adotada (`TestR11StatusDoesNotClaimTagsOfAProjectThatNeverAdopted` verde). Mas `adopted` vale `false` também quando a policy de hoje não pôde ser lida, e a testemunha de governança prescrita pela r11 não foi usada. Isso é o R12-F01. |

## Bloqueante de merge

### R12-F01 — medium — Policy de hoje perdida, ilegível ou apagada vira "projeto nunca adotado" no status, e `release open-next` aprova

Local: `pose-mcp/internal/pose/release_lifecycle.go:471`–`:492` (`GetReleaseStatus`).
Requirements R3 ("distinguir fonte ausente/ilegível de ausência legítima") e R7 (checker e status
compartilham a verificação).

```go
adopted := false
if policy, perr := LoadReleasePolicy(s.Root); perr == nil {
    adopted = policy.AdoptedAt != ""
}
...
if errors.Is(snapErr, ErrReleaseSnapshotMissing) && !adopted {
    // ... A policy that cannot be read is not "never adopted".
    if held, heldErr := TagHoldsAdoptedReleasePolicy(s.Root, tag); heldErr == nil && !held {
        continue
    }
}
```

Quando `LoadReleasePolicy` falha (JSON inválido, link cujo alvo se perdeu, schema errado, policy
apagada), `adopted` fica `false`. Uma tag governada sem manifest cuja árvore não traz policy
adotada some então do status. O comentário promete o contrário, mas só a policy da árvore da tag
recebe esse tratamento. A policy de hoje não recebe. A r11 prescreveu outro critério: "policy
perdida ou ilegível (conservador); ou policy ausente com uma testemunha de governança
(`governedReleaseWitness`)". O checker aplica esse critério (`cli/release_lifecycle.go:773`–`:796`:
policy perdida reprova, e policy ausente consulta a testemunha). O status não aplica.

Reprodução (sonda numa cópia isolada, `git archive HEAD` em `/tmp`, helper `adoptedTaggedRelease`).
A release `v1.0.0` está `verified` e sólida. Há também uma tag `v1.1.0` sem manifest, num commit cuja
árvore não tem policy. Com a policy íntegra, o próprio HEAD trata essa tag como governada: o checker
diz `governed tag v1.1.0 has no manifest in its tagged tree`, e o status a lista como `prepared` com gap.

| Policy de hoje | Status lista `v1.1.0` | `release open-next --version v1.2.0` | `pose check` (release) |
|---|---|---|---|
| Íntegra (controle) | sim | **recusa** (`latest release is not verified`) | errors=1 (tag governada sem manifest) |
| JSON corrompido | **não** | **aprova** (`Next-cycle plan: v1.2.0 after verified v1.0.0`) | errors=1 (`invalid release policy`) |
| Link com alvo perdido | **não** | **aprova** | errors=1 (link cujo alvo não é lido) |
| Apagada, com testemunha (`v1.0.0` no worktree) | **não** | **aprova** | errors=1 (`policy is missing, and the cut v1.0.0 … shows the project adopted`) |

Uma perda da policy muda, portanto, o veredito de `release open-next`, o comando que diz que o
ciclo pode avançar, de recusa para aprovação. Também some a obligation da tag em
`releaseQueueObligations` (`pose state --attention`, tool MCP), a entrada em
`.pose/indexes/releases.json` e a release em `pose_release_status`. No último caso da tabela, checker e
status discordam sobre o mesmo projeto: o checker diz que ele adotou, e o status diz que não. É a
mesma régua de R10-F03 (uma perda distinguível que deixa um gate verde) e de R11-F01 (o critério de
adoção difere entre checker e status), e a regressão foi introduzida pela própria correção de
R11-F01. Por isso não entra como follow-up. É medium, e não high, porque exige uma tag governada
sem snapshot cuja árvore não traz a adoção, e porque o `pose check` continua vermelho.

Ação mínima: um predicado único de adoção no domínio, usado pelo checker, pelo status e pela
testemunha:

- policy legível: `adopted_at != ""`;
- falha de leitura que não é ausência provada: adotado (conservador);
- ausência provada (`ProvenAbsent` nos dois caminhos): a testemunha de governança. Para isso,
  mover `governedReleaseWitness` para o domínio ou reaproveitá-la, e manter o critério por tag
  (`TagHoldsAdoptedReleasePolicy`) só como complemento.

Regressões: as três variantes da tabela (JSON corrompido, link perdido e policy apagada com corte no
worktree), em que a tag continua listada e `open-next` recusa. Manter os controles da r11: policy não
adotada ou ausente, sem testemunha, não lista tags.

## Follow-up aceitável — low (novo)

Proposta: `[open]` com `(owner:@core crit:low review:2026-12-01)`. Não foi gravado na spec, porque o
parecer é bloqueante.

- **R12-L19 — o status de um projeto não adotado ainda custa vários processos Git por tag.**
  `ReadReleaseSnapshot` e `TagHoldsAdoptedReleasePolicy` rodam para cada tag antes do descarte. Com
  300 tags e a policy do scaffold não adotada, `pose release status --json` leva cerca de 2,9 s e
  responde `releases: []`. O mesmo custo atinge obligations, índice e public-claims. É só desempenho:
  nenhum estado sobe e nenhum gate aprova por isso. Ação: decidir a adoção (o predicado de F01)
  antes de ler snapshots, e ler a policy da árvore só quando a policy de hoje não decide.

## Follow-ups já registrados

Os dezoito itens `[open]` da spec (incluindo L14–L18 da r11) continuam como estão, sem alteração de
disposição.

## Varredura por classe

| Classe | Consumidores inspecionados | Resultado |
|---|---|---|
| Perda tratada como ausência | `LoadReleasePolicy` em todos os consumidores (status, `ReleaseCandidateGaps`, `ReleaseCandidateSuperseded`, checker, `release prepare`/`plan`, `pendingSourceUsed`); `ProvenAbsent`; `releasePolicyAt`/`TagHoldsAdoptedReleasePolicy`; `governedReleaseWitness`; `LoadReleaseEvents`; inventário em status, checker e testemunha; listagem de tags; `GitNotARepository`; `ResolveReleaseTag`; `releaseTagState`/`releaseTagSnapshotGaps`; consumidores do status (CLI `status`/`open-next`, MCP, obligations, índice, public-claims) | **R12-F01** no status. `ReleaseCandidateGaps` e `ReleaseCandidateSuperseded` tratam a falha de forma conservadora (gap, ou "não superseded", e então o corte é julgado). O cutoff vazio sob policy ilegível é conservador. `pendingSourceUsed` (`obligation_projection.go:112`) é anterior ao PR e só decide a aplicabilidade de produtores pendentes, e o produtor `release` já está registrado. Sem novidade. |
| Ausência confundida com presença | Laço de tags do status para policy legível não adotada | Corrigido (R11-F01). |
| Ordenação por texto | `CompareReleasePrecedence`, cutoff, ordenação do status (`Releases[0]` em `open-next`), `previous` | Nada novo. A ordenação é SemVer. |
| Campo sem validação | `releasePolicyAt` (decodificação não estrita, já em L13), `version_evidence`, `previous_version` | Já coberto por L01, L13 e L17. |
| Comparação de caminhos | `ProvenAbsent` nos dois caminhos de policy, caminhos de blob na árvore da tag | Nada novo. |

## Checks executados (HEAD 75a0a9ed)

- `go vet ./...` (pose-mcp): ok.
- `go test ./... -count=1` (pose-mcp): todos os pacotes ok.
- `go test -race -count=1 ./internal/cli ./internal/pose -run 'Release|R[0-9]|F[0-9]|Archiv|Snapshot|Tag'`: ok.
- `pose check` (binário compilado do HEAD, strict): SUCCESS com 30 warnings preexistentes.
- `pose artifact-check --spec pose-release-manifest-tag-integrity`: 47 claims, 1 change set.
- Marcadores de dívida (`TODO`, `FIXME`, `panic(`, `stub`) nas linhas Go adicionadas: 0.
- Sondas em cópia isolada: `.pose/results/b38r12-claude-independent-review.json`.

Os testes verdes não cobrem R12-F01. A regressão da r11 só remove a policy num repositório cujas
tags carregam a policy adotada na própria árvore, e nenhum teste usa uma policy ilegível.

## Estado deixado

A spec continua in-progress, com `completed_at` vazio. Os arquivos não rastreados criados por esta
revisão são este parecer e `b38r12-claude-independent-review.json`. O `artifact-check` regravou
`.pose/indexes/delivery-integrity.json`, e restaurei esse arquivo ao conteúdo do HEAD. A modificação
preexistente em `.pose/indexes/generation.json` foi mantida como estava.
Contributor Mode: não criei rascunho de contribuição nem issue.
