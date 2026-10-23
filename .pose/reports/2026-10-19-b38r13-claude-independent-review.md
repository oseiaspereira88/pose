# Revisão independente B38r13 (Claude) — PR #198

Data: 2026-10-19. Repositório: oseiaspereira88/pose.
Spec: `pose-release-manifest-tag-integrity`.
HEAD revisado: `07e77f89` (correção da r12 em `9625c720`). Base: `origin/main`.

**Decisão: approved. Nenhum bloqueante de merge.** R12-F01 está corrigido. A varredura da
classe "perda tratada como ausência" achou um resíduo (R13-L20). Ele é low e entra como
follow-up aceitável: só esconde uma obligation advisory, o `pose check` continua vermelho e
`release open-next` continua recusando. Os dezenove follow-ups low já registrados continuam
`[open]`. Com isso a spec passa a ter vinte.

## Identidade e separação declaradas

`agent:independent-claude-opus-5-5-review`, Claude Opus 5.5 (Claude/Anthropic).
O implementador também é Claude/Anthropic: **NÃO há separação de vendor**. A separação é só de
execução e de contexto, porque o Codex está indisponível (limite de uso até 2026-10-24). A
identidade não é autenticada: a separação é declarada, não verificada. No pose-dist as
atestações não são assinadas, e esta atestação foi registrada sem envelope.

## Rules applied during review

- Tipo de mudança: bugfix de integridade de dados, com contratos de CLI/JSON/MCP e de obligations.
- Skills: `pose-review` e `pose-spec-closeout`. Workflow: `.pose/workflows/review.md`.
- Regras: backend Go, `security.md` (integridade de dados), `documentation-style.md`,
  `delivery-evidence.md` e `knowledge-governance.md`. `pose suggest review --path pose-mcp`
  confirmou o trail.
- Lidos: a spec, o diff contra `origin/main`, o commit `9625c720` e o parecer r12.

## Correção da r12 conferida

| Achado | Correção no HEAD | Conferência |
|---|---|---|
| R12-F01 | `ErrReleasePolicyMissing` separa a policy provadamente ausente da ilegível. No status, policy ilegível conta como adotada, e policy ausente com um corte no worktree também | Corrigido nas três variantes da r12 (JSON corrompido, link perdido e policy apagada com corte no worktree). `TestR12LostPolicyOfTodayDoesNotMakeTheProjectLookUnadopted` está verde, e os dois mutantes novos e os seis vizinhos (adoção e R11) foram reexecutados por mim e morreram por asserção. A mensagem de erro é a mesma de antes. A unificação com `governedReleaseWitness` pedida pela r12 ficou parcial: veja R13-L20. |

## Bloqueantes de merge

Nenhum.

## Follow-up aceitável — low (novo)

Registrado na spec como `[open]` com `(owner:@core crit:low review:2026-12-01)` e como finding
`b38-r13-l20` (accepted-risk) na atestação.

- **R13-L20 — com a policy de hoje provadamente ausente, o status não usa a testemunha completa
  do checker.** `GetReleaseStatus` (`pose-mcp/internal/pose/release_lifecycle.go:480`–`:486`) só
  aceita um corte no worktree como prova de adoção. `governedReleaseWitness`
  (`pose-mcp/internal/cli/release_lifecycle.go:906`) também aceita o manifest na árvore de
  qualquer tag e a policy adotada na árvore de outra tag. Para reproduzir (sonda numa cópia
  isolada): `v1.0.0` é uma tag com snapshot, e `v1.1.0` é uma tag sem manifest num commit de
  árvore vazia. A policy e `.pose/releases/` são apagados.

  | Policy de hoje | Status lista `v1.1.0` | Obligations | Checker | `open-next --version v1.2.0` |
  |---|---|---|---|---|
  | Adotada (controle) | sim | 2 | — | — |
  | Apagada, sem inventário no worktree | **não** | 1 | testemunha `the manifest in the tree of tag v1.0.0`: reprova | recusa (`latest release is not verified`) |

  Por que é follow-up e não bloqueante: nenhum gate aprova indevidamente. `open-next` exige uma
  release `verified`. Isso só existe com eventos no worktree, e eventos no worktree já tornam o
  projeto adotado pelo predicado novo. O `pose check` reprova pela testemunha. O efeito é a
  ausência de uma obligation advisory e de uma entrada em `releases.json`/`pose_release_status`.
  Ação: levar `governedReleaseWitness` para o domínio e usá-la no status quando a policy estiver
  provadamente ausente.

## Follow-ups já registrados

Os dezenove itens `[open]` da spec (incluindo R12-L19) continuam como estão, sem mudança de
disposição.

## Varredura por classe

| Classe | Consumidores inspecionados | Resultado |
|---|---|---|
| Perda tratada como ausência | Todos os chamadores de `LoadReleasePolicy` (status, `ReleaseCandidateGaps`, `ReleaseCandidateSuperseded`, checker, `release plan`/`prepare`, `pendingSourceUsed`); `ProvenAbsent` em `loadReleaseFragments`, `LoadReleaseEvents`, inventário do status, checker, testemunha e archive do candidato; `TagHoldsAdoptedReleasePolicy`; `ReadReleaseSnapshot`; consumidores do status (CLI `status`/`open-next`, MCP, obligations, índice, public-claims) | **R13-L20** (low). Nos demais, a falha de leitura vira erro, gap ou "adotado". `pendingSourceUsed` é anterior ao PR, como a r12 já registrou. |
| Ausência confundida com presença | Policy legível não adotada (R11) | Mantida: os controles da r11 passam. Policy ilegível agora lista as tags, o que é conservador. |
| Ordenação por texto | `CompareReleasePrecedence`, cutoff, `Releases[0]` em `open-next`, obligations | Nada novo. |
| Campo sem validação | `releasePolicyAt` (L13), `version_evidence` (L01), `previous_version` (L17) | Já coberto. |
| Comparação de caminhos | `ProvenAbsent` nos dois caminhos de policy | Nada novo. |

## Pergunta de gate (passo 8 do pose-review)

O jeito mais barato de satisfazer a forma do gate sem produzir seu valor seria apagar ou
corromper a policy para que o status esqueça tags governadas. Depois da r12, só o caso R13-L20
ainda esconde algo, e mesmo nele o checker reprova e `open-next` recusa. O corpus adversarial
não tem um caso para R13-L20. Essa lacuna fica registrada no próprio follow-up.

## Checks executados (HEAD 07e77f89)

- `go vet ./...` e `go test ./... -count=1` (pose-mcp): ok.
- `go test -race -count=1 ./internal/cli ./internal/pose -run 'Release|R[0-9]|F[0-9]|Archiv|Snapshot|Tag'`: ok.
- `pose validate --strict --json-out .pose/results/delivery-validation.json`: 140/140.
  `pose validate --strict --module pose-mcp`: 137/137.
- `pose check` (strict, binário do HEAD): SUCCESS com 30 warnings preexistentes.
- `pose artifact-check --spec … --strict`: exit 0, 47 claims, 78 observed. Os warnings de órfãos são do repositório.
- `pose surface-check --spec … --strict`: 1 target, 140 resultados, 0 findings.
- `pose assess tech-debt`: 1 marcador, coberto, e 0 não cobertos. O scan sai INCOMPLETE só por
  duas specs de 2026-08-15 que ele não lê, sem relação com este PR. `pose assess integrate` e
  `pose assess design` (observed) também rodaram, e `pose recurrence-check --tolerant --window-days 14`
  deu 0 chaves.
- Mutações reexecutadas (`b38-mutations.py --only`): 8/8 mortas por asserção.
- Sondas: `.pose/results/b38r13-claude-independent-review.json`.

## Review e closeout nativos

- Plano `sha256:3a4c917e89236ce003abf862f3791cb86b7a29ee533c02a5968032e97bb28a1c`
  (different-actor, 15 critérios, 11 ferramentas).
- Bundle `rvb-7a5a4b8cac65cf0b`
  (`sha256:7a5a4b8cac65cf0b6e4dd71728bca5345023cd761ead30a1cef3b9433367a512`), selado com 137
  evidências.
- `auto-attest` sem `--apply`: 8 critérios de julgamento pendentes. Todos receberam conclusão
  explícita.
- Atestação `rva-fc93233a05b7e708`: approved, sem envelope, com o finding `b38-r13-l20`. Os papéis
  prepared, concluded e applied são todos do revisor, sem confirmação humana.
- `review verify`: fresh e approved. `review-check`: ok. `pose close`: Closed. `closeout-check`:
  terminal, sem próxima ação. `lint-spec --strict`: ok.
- As ferramentas de conclusão (`review-check` e `closeout-check`) ficaram `deferred` na atestação
  imutável e foram executadas depois.

## Estado deixado

A spec está `done`, com `completed_at: 2026-10-19`. A Final Report registra a revisão, a falta
de separação de vendor e o follow-up novo. O plano marca B38 como `delivery_complete`, e o
roadmap `pose-release-sufficiency-and-recovery` registra o closeout. `assess discover --if-stale --update-state`
rodou. `technical-debt.md`/`.json` foram mantidos no conteúdo do HEAD porque a versão regenerada
grava um link `file://` com o caminho absoluto local do revisor. Isso é uma fricção do motor e
candidata a contribuição no Contributor Mode. Nenhum rascunho foi criado sem consentimento.
Não alterei policy, pin de issuer, código de produção, testes, tags nem releases. O merge fica
com o maintainer.
