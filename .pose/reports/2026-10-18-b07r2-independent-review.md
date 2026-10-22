# Revisão independente B07r2 — PR #195

**Decisão: changes-requested. 13 bloqueantes de merge; 3 follow-ups aceitáveis de baixo risco.**

Revisão de 2026-10-18 sobre `51f771a049a8c4042e0ed2444842f342668a26bc`, branch `impl/pose-b07-git-snapshot-integrity`, diff contra `origin/main` e spec `pose-git-and-dispatch-snapshot-integrity`. A API do GitHub confirmou esse HEAD no [PR #195](https://github.com/oseiaspereira88/pose/pull/195). A classificação abaixo reúne os achados desta rodada, inclusive as variantes encontradas nos consumidores vizinhos; não declara ausência de outros defeitos possíveis.

Revisor: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI; execução independente, vendor diferente do implementador Claude/Anthropic, conforme autoria do commit `371109cf`. Identidade e separação **declaradas**, sem alegação de autenticação criptográfica. Nenhum bundle foi selado e nenhuma atestação, envelope, commit, push ou close foi produzido. Policy e issuer não foram alterados. A spec permanece `in-progress`; os arquivos desta revisão ficam sem commit em reports/results.

## Rules applied during review

- Tipo: misto, correção Go e documentação de contrato.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`; workflow `.pose/workflows/review.md`; manual `POSE.md`.
- `.pose/rules/backend-go.md`: propagação de falhas, estado, fronteiras de consumidores e concorrência.
- `.pose/rules/security.md`: materialização, identidade do worktree, links e evidência governada; inspeção de segredos no diff.
- `.pose/rules/documentation-style.md`: spec, ADR, changelog, manuais e scaffold.
- `.pose/rules/delivery-evidence.md`: proveniência, requisitos e gates de entrega.
- `.pose/rules/knowledge-governance.md`: procura de decisões e riscos anteriores; nenhum risco foi aceito para merge nesta rodada, portanto nenhum handoff de aceitação foi criado.
- AGENTS raiz e específicos dos manuais traduzidos/scaffold consultados. Demais stacks não se aplicam ao diff. Nenhuma delegação a outro agente foi executada.

## Plano, ferramentas e checks

`context` resolveu a autoridade local `proj.pose-dist`, sem ambiguidade. Plano efetivo: `sha256:b94c1f031318c494b293c1c5751630587f1f250d641d49d5c2b4354e707693c3`, componente `pose-mcp`, banda critical, independência different-actor, 15 critérios. Bundle apenas preparado: `rvb-b00e3a38ce1068d8`, digest `sha256:b00e3a38ce1068d807190605f58356cf71b97c6ffb1a4266e0d0e93ed938f9b8`; patch `sha256:7cbcedf28e79aa32e518df2d730c35bf1a780b1d85a8e12f63dad1eb27918324`, tree `sha256:979693d9984f37253e562b527599e405abc87946575d11e1dbc84ac22e19ab95`. É a preparação inicial, não um selo nem uma aprovação; assessments posteriores podem acrescentar evidência derivada.

Ferramentas ativas requeridas executadas: artifact-check, surface-check e validação estrita do único módulo alterado. Ferramentas recomendadas executadas: suggest-review, assess-design, assess-discover `--if-stale`, assess-integrate e assess-tech-debt. Review-check e closeout-check, de conclusão, foram diferidos; seal, auto-attest, attest, review verify, close e assessment de encerramento não foram executados diante dos bloqueantes. Não houve transição de follow-up.

O `pose` instalado tinha revision `8aa729a7` de main. Após identificar isso, a CLI revisada foi compilada com `go build -o /tmp/pose-b07r2 ./cmd/pose` em `pose-mcp`; context, bundle/plan, artifact/surface/check/lint, validação e assessments foram repetidos com esse binário. Os resultados abaixo usam o HEAD revisado. Probes importam o código atual em cópias temporárias dos módulos.

| Check | Resultado |
|---|---|
| `/tmp/pose-b07r2 validate --strict --module pose-mcp` | FAIL: 137 checks, 136 pass, 1 fail, 232,5 s; B13 |
| Default `go test ./...` e `go build ./...` da matriz | PASS |
| `go test -race ./internal/pose ./internal/cli` | PASS, sem cache indicado |
| `go vet ./...` | PASS |
| Builds Windows/amd64 e Darwin/arm64 `go build ./...` | PASS; não são execução nessas plataformas |
| `/tmp/pose-b07r2 check --strict` | PASS, 29 avisos |
| `/tmp/pose-b07r2 lint-spec pose-git-and-dispatch-snapshot-integrity --strict` | PASS, aviso de seção Tasks opcional |
| artifact-check do escopo, strict | PASS; 29 claims, zero findings do escopo |
| surface-check do escopo, strict | FAIL: stale-evidence e unconsumed-capability; B12 |
| Driver `b07-mutations.py`, execução independente | 32/32 killed; nenhum survived/build-error |
| Probes independentes desta rodada | 13 asserções de regressão falham e 2 controles passam; não há build-error |
| `git diff --check origin/main...HEAD` | PASS |
| Inspeção do diff e padrões de chave privada/PAT/AWS | Zero ocorrências dos padrões; não equivale a scan completo de gitleaks |

Gitleaks, govulncheck e golangci-lint não estão instalados. Não há mudança de dependências. Tech-debt encontrou 1 marcador, coberto pela spec `pose-residual-material-coverage-and-identity`, e declarou leitura incompleta de duas specs antigas; essa limitação não foi convertida em ausência de dívida. Integração e seus gaps heurísticos foram examinados sem tomar ausência de referência de consumidor como defeito comprovado. Design observou 22/22 arquivos do subject; discover reaproveitou a observação após atualização inicial. Recurrence: 393 registros, zero chaves sinalizadas; sem gatilho de escalation. Avisos de owner, mapeamento e backlog anteriores não são prova de integridade nem novos achados deste diff.

Evidência compacta: [b07r2-independent-review.json](../results/b07r2-independent-review.json). Reprodução: `python3 -I .pose/results/b07r2-independent-probes.py`; [runner](../results/b07r2-independent-probes.py) e [extrato](../results/b07r2-independent-probes.log). O exit 1 é esperado neste HEAD: as asserções exigem as garantias violadas. Fixtures alteram somente repositórios sintéticos, nunca a policy deste clone.

## Bloqueantes de merge

### B01 — High — O checkout não é comparado ao subject de implementação selado

`review_dispatch.go:258–271,358–381`; produtor vizinho `review_bundle.go:800–805,1225–1260`. R2/R6.

`sealedInputsThatDiffer` compara seções/configurações/regras, mas omite `Payload.Subject`. Um arquivo de implementação editado com assume-unchanged pode ser selado com seu digest atual enquanto `Subject.Head` ainda aponta ao commit antigo. O checkout passa na comparação com seu próprio commit; a origem ainda prepara o mesmo bundle. O adapter lê `package posemcp`, embora o bundle sele `sealed-new-subject`, e o run termina completed.

Probe: `TestB07R2SealedSubjectNotCheckout`. Ação: confrontar também cada entrada material de implementação com o subject selado, incluindo identidade/tipo, ou recusar a divergência antes de executar. Corrigir somente a seção Requirements, como na B02 da primeira revisão, não cobre a mesma classe na implementação.

### B02 — High — Os blockers da preparação são descartados na revalidação

`review_dispatch.go:268–271,413–420`; `review_bundle.go:477,800–802,564–577`. R1/R5/R7.

`PrepareReviewBundle` pode retornar sucesso com `Blockers`; eles não fazem parte do digest do payload. O dispatch compara apenas o digest. Corromper `.git/index` da origem durante o adapter torna seu status unknown, mas mantém o payload e produz completed, exit 0. A nova proteção do status no produtor é perdida no consumidor.

Probe: `TestB07R2UnreadableSourceStatus`. Ação: exigir preparação sem blockers e leitura conhecida, tanto do checkout quanto da origem, nas duas fronteiras do run; digest igual não substitui elegibilidade.

### B03 — High — Gitlink materializado não prova conteúdo nem fronteira

`git_snapshot_integrity.go:246–252,315–321`; consumidores `review_dispatch.go:248–256,331` e classificação `review_bundle.go:745–752,803–808`. R2/R3.

`gitlinkMaterialized` só consulta `.git` e HEAD. Com o mesmo pin, um arquivo do submódulo editado e escondido por assume-unchanged retorna `diverged=[]`. Substituir o diretório do submódulo por symlink a um repositório externo no mesmo pin também retorna vazio. A correção cobre ausência, mas não as outras materializações pertinentes.

Probe: `TestB07R2DirtyGitlink`, com controle íntegro antes das duas mutações. Ação: verificar tipo/fronteira e conteúdo relevante do submódulo, recursivamente conforme o escopo, ou recusar unknown; o pin sozinho não comprova o que foi lido.

### B04 — High — O ambiente pode redirecionar a verificação a outro worktree

`git_snapshot_integrity.go:95–108,213–225`; vizinhos `git_spawn_cache.go:67`, `obligation_projection.go:705–713` e `federated_acceptance.go:803–810`. R3/R6.

Os subprocessos herdam `GIT_DIR`/`GIT_WORK_TREE`. `WorktreeDivergenceFor` ainda troca a raiz solicitada pela resposta de show-toplevel sem comprovar a vinculação. Com a raiz solicitada modificada e uma segunda materialização íntegra apontada pelo ambiente, divergência e status retornam vazios, sem erro.

Probe: `TestB07R2OtherWorktreeEnvironment`. Ação: fixar explicitamente a identidade do repositório/worktree e tratar variáveis Git de redirecionamento nos leitores; normalizar um subdiretório legítimo não autoriza substituir a raiz por outra materialização.

### B05 — High — Empate entre bundles ainda pode conservar aprovação sobre negativa

`review_bundle.go:2028–2029`; `review_closeout.go:2151–2198`; vizinhos CurrentReviewBundle/Verify, latestNegativeReview, governance replay/waits e attempts. R5/R7.

`attestationLess` resolveu o empate entre atestações de um bundle. A retenção de scope fechado continua percorrendo bundles por sealed_at/ID. No mesmo instante, o bundle aprovado `rvb-cfe687f0132cd13b` ordena depois do bundle negativo `rvb-865c99ddd5285d6e`. Após mudança rotineira da matriz, Verify retorna `Approved=true`, state closed, conservando a aprovação antiga apesar da negativa posterior.

Probe: `TestB07R2NegativeAcrossBundles`. Ação: aplicar ordem demonstrável ou recusa conservadora também entre bundles antes da retenção; hash não estabelece precedência. O probe vizinho `TestB07R2LegacyNegativeSameInstant` seleciona rvw-z, mas não aprova: a exigência de supersedes barra essa variante, portanto ela não foi reportada como bypass adicional.

### B06 — High — O digest da policy tem serialização ambígua

`obligation_projection.go:821–833`; consumidores `review_dispatch.go:199,399`, `closeout_plan.go:91` e snapshots. R1/R5/R7.

O hash concatena nome, NUL, bytes arbitrários, NUL. Um arquivo `a` contendo `x\0z.json\0{}` gera exatamente os mesmos bytes de hash que dois arquivos: `a=x` e `z.json={}`. Não é colisão de SHA-256: é ambiguidade da entrada serializada. O probe CLI altera o conjunto de arquivos da policy durante o adapter usando essa equivalência, e o run completa.

Probes: `TestB07R2PolicyDigestFraming` e `TestB07R2PolicyChangeWithEqualDigest`. Ação: usar representação canônica com comprimentos/encoding inequívoco, ou digests individuais em estrutura tipada. O contrato de projeto vizinho usa JSON estruturado; o helper de diretório não oferece a mesma propriedade.

### B07 — Medium — Closeout admite troca por link e checkpoint arbitrário como regeneração

`closeout_plan.go:231–250,261–275`; contraparte `git_snapshot_integrity.go:254–267,302–305`. R3/R4.

O closeout reaproveita a classificação de nomes, mas não as verificações de tipo do dispatch. Trocar `.pose/state/project-state.md` ou `.pose/results/delivery-validation.json` por symlink deixa `ungeneratedChanges=[]`. O prefixo `.pose/closeout-plans/` aceita também um checkpoint arbitrário rastreado, inclusive substituído por link, sem relação com o output do escopo.

Probe: `TestB07R2CloseoutType`. Ação: validar tipo/fronteira/ação e os outputs exatos do produtor e do escopo. O probe demonstra contexto indevidamente limpo; não afirma que já tenha publicado atestação ou escrito no destino do link. Os gates posteriores não justificam chamar essa alteração não autorizada de regeneração.

### B08 — High — A revalidação semântica da origem omite modo executável

`review_dispatch.go:413–418`; `review_bundle.go:1225–1260`; contraste `git_snapshot_integrity.go:282–285`. R2/R5/R6.

O digest do subject lê bytes e status, sem modo. Durante o adapter, configurar core.filemode=false na origem e mudar o executável de input pertinente mantém o mesmo bundle e produz completed. A comparação de modo adicionada ao worktree descartável não protege a pertinência do input na origem.

Probe: `TestB07R2SourceMode`. Ação: vincular os metadados materiais pertinentes à identidade comparada, sem depender da configuração de status. Preservar o tratamento Windows já introduzido para recusa indevida de modos inexistentes nessa plataforma.

### B09 — High — A janela entre última leitura e publicação continua sem proteção

`review_dispatch.go:227–232,348–351`; ADR Consequences e spec Technical Plan. R5/R7.

Após `dispatchNoLongerGoverns`, `finish` chama o relógio e publica o registro sem vincular o efeito às precondições lidas. Um callback de relógio determinístico altera o input nessa fronteira e o run ainda é publicado completed. O probe modela uma edição concorrente no ponto exato; não requer sleeps ou processos escapando do adapter.

Probe: `TestB07R2PublishWindow`. Ação: definir o ponto protegido de consumo/publicação e recusar o efeito governante sem essa garantia, como a spec exige. Não se exige CAS geral nem sandbox de adapter. O ADR declara a janela aberta, mas isso não dispensa a condição expressa da spec; nenhum run foi convertido em atestação neste probe.

### B10 — Medium — Revalidação de snapshot ainda retorna vazio para estado inelegível ou mudado

`obligation_projection.go:740,744–773`. R1/R7.

A perda true→false de Coherent passou a ser detectada, mas uma falha persistente já presente no snapshot retorna `changes=[]`. Além disso, a transição WorktreeDirty=false→true não é comparada: editar uma spec governada depois de snapshot limpo também retorna vazio. O digest muda nesse segundo caso, mas o consumidor não o consulta. O comentário apresenta vazio como revalidação para write gate.

Probes: `TestB07R2PersistentUnknown` e `TestB07R2SnapshotCleanToDirty`. Ação: exigir coerência nas duas leituras e comparar todos os campos vinculantes relevantes, inclusive a transição clean/dirty. Não foi encontrado write gate de produção que chame essa função além dos testes; o achado é no contrato exposto do domínio, sem alegar uma aprovação CLI produzida por ele.

### B11 — Medium — O plano de closeout não bloqueia HEAD desconhecido independentemente do status

`closeout_plan.go:91,117–132`; produtor `federated_acceptance.go:803–810`. R1/R7.

`gitHeadAtRoot` converte falha de HEAD em string vazia; PlanCloseout guarda o campo sem validá-lo. Um wrapper sintético que falha somente em rev-parse --verify HEAD, mantendo status legível e limpo, produz context=done e etapas evidence/index/seal pendentes, sem erro. A recusa por status ilegível não cobre falha independente de HEAD.

Probe: `TestB07R2CloseoutUnknownHead`, com policy sintética válida. Ação: distinguir export sem Git, quando suportado, de repositório com HEAD ilegível; bloquear a etapa de contexto deste último com causa preservada. Não se alega que os gates posteriores já tenham aprovado esse plano.

### B12 — Medium — Evidência canônica de entrega permanece stale

`.pose/policy/delivery.json:5`, `.pose/results/delivery-validation.json` e Requirement trace da spec.

O surface-check deste HEAD ainda retorna stale-evidence e unconsumed-capability por falta de integration corrente. Rodar a matriz sem regenerar o results_path não atualiza o arquivo consumido pelo gate; o JSON desta revisão não o substitui.

Ação: após os fixes e validação verde, regenerar o results_path com o produtor registrado, indexar e comprovar surface-check estrito antes de selar. Bloqueante mecânico de entrega/closeout, não outro bypass funcional.

### B13 — Medium — Um check obrigatório da matriz não passou

`install-and-upgrade-journeys-integration`, `TestJourneysInstallAndUpgradeFromThePublishedRelease`.

As duas execuções da matriz (motor instalado e binário deste HEAD) falharam ao listar releases publicados: curl HTTP 403. A execução autoritativa desta revisão terminou 136/137, exit 1. Os demais 136 checks passaram, inclusive o de integração B07.

Ação: restabelecer acesso ao recurso requerido e obter validação estrita integral verde, sem enfraquecer a matriz. É limitação externa comprovada, não diagnóstico de regressão de código B07; continua bloqueante mecânico conforme o workflow.

## Follow-ups aceitáveis de baixo risco

Não foram inseridos na spec porque há bloqueantes. Disposições propostas para uma futura revisão/closeout:

- [open] L01: aplicar a mesma seleção de gitlinks pertinentes antes e depois do adapter. O pré-check usa WorktreeDivergenceFor com filtro; o pós-check usa WorktreeDivergence, que examina todos. Gitlink ausente e alheio ao subject passa antes e é recusado depois. `TestB07R2IrrelevantGitlink` registra pre=[] e post=[module]. A variante só produz recusa indevida. (owner:@core crit:low review:2026-12-01)
- [open] L02: tornar `TestB07IndependentLostSupersedingBundle` específico para perda de cobertura. O DispatchPlan omite PolicyDigest; a guarda de baseline o recusa antes de ReadReviewBundles, então o teste não prova o ramo que seu nome/trace atribuem. Fornecer baseline válida e controle íntegro. A guarda de cobertura existe e a limitação de teste não foi tomada como prova de bypass; B02 foi reproduzido separadamente. (owner:@core crit:low review:2026-12-01)
- [open] L03: preservar causa tipada também nos erros de Lstat, Readlink e verificação de gitlink. Só ReadFile regular foi corrigido para GitUnknownError; os demais viram divergência/false. Para essas falhas de leitura a variante é conservadora, com diagnóstico menos específico. Não abrange a aceitação indevida de conteúdo/link de B03. (owner:@core crit:low review:2026-12-01)

## Varredura por classe, contratos e encerramento

| Classe | Consumidores examinados antes de concluir | Resultado |
|---|---|---|
| Ordenação texto/ID | bundles, atestações, retenção, Verify, latestNegativeReview, replay, waits, outcomes, attempts e findings/obligations | B05 entre bundles; empate de atestações do mesmo bundle corrigido; variante legacy ensaiada recusada por supersedes. Ordenação de runs serve ao histórico/contagem, não foi promovida a aprovação. |
| Ausência versus perda | status/HEAD/tree, metadata .git, coverage de bundles, directoryDigest, snapshots e plano de closeout | B02/B03/B10/B11; sentinelas ausente/indisponível agora distintas, mas consumidores ainda perdem precondições. |
| Campo sem validação | Blockers, Subject, PolicyDigest, Coherent, WorktreeDirty, Inputs.Revision e modo/tipo dos inputs | B01/B02/B06/B08/B10/B11; validar presença ou digest não comprova elegibilidade. |
| Paths/materialização | porcelain -z e rename dos dois lados, cache, allowlists de dispatch/closeout, subdiretórios, gitlinks, links e identidade Git herdada do ambiente | B03/B04/B07 e L01; backslash POSIX e subdiretório corrigidos, mas identidade/fronteira ainda têm variantes abertas. |

Pergunta adversarial aplicada: a forma mais barata de cumprir o gate é conservar o commit e o payload aparentemente estáveis enquanto se muda o subject selado, perde-se o status da origem ou verifica-se outra materialização. B01–B04 mostram execução/observação aceita sem a garantia pretendida. O corpus `pose-mcp/internal/cli/testdata/adversarial/README.md` tem casos gerais, mas não declara essas variantes B07; os probes anexos registram o gap sem alterar corpus ou policy. A matriz e as 32 mutações verdes não contradizem esses contraexemplos.

Compatibilidade: `stale` é estado público novo de run; a CLI o converte em não-zero. ADR/changelog classificam a recusa mais estrita como minor. Não se encontrou mudança de dependência, issuer ou assinatura nesta rodada. As afirmações amplas de integridade no changelog/manuais devem ser reconciliadas com os fixes; não foram contadas novamente como bloqueantes documentais independentes dos respectivos defeitos.

Os artefatos derivados gerados por assessments/index foram restaurados ao HEAD após coletar a evidência. Permanecem apenas o parecer e seus resultados sem commit. Não houve aceitação de risco de aprovação indevida, alteração de lifecycle ou contribuição upstream. Contributor Mode: oportunidades e reproduções sintéticas estão neste parecer; nenhum draft em `.pose/contributions/` foi criado, pois essa decisão cabe ao usuário. Para nova rodada, corrigir B01–B11, obter os gates B12/B13 verdes e repetir a varredura antes de qualquer selo.
