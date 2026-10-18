# Revisão independente B07r3 — PR #195

**Decisão: changes-requested. 10 bloqueantes de merge (9 funcionais/de contrato e 1 gate de entrega); 5 follow-ups aceitáveis de baixo risco.**

Revisão de 2026-10-18 sobre `75a20ba10b170e28ce494c8e7f649b9152a70566`, branch `impl/pose-b07-git-snapshot-integrity`, spec `pose-git-and-dispatch-snapshot-integrity`, diff completo contra `origin/main` (`8aa729a76ab1c094690a0892263a86bf3bb7bb42`). A API GitHub confirmou esse HEAD e base no [PR #195](https://github.com/oseiaspereira88/pose/pull/195), aberto. Foram lidos os patches, testes, ADR, spec, manuais, scaffold, resultados e pareceres B07/B07r2, além dos consumidores adjacentes das classes indicadas pelo solicitante.

Revisor: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI; revisão independente do implementador Claude/Anthropic, vendor diferente. Identidade e separação **declaradas**, sem autenticação criptográfica ou alegação de independência cognitiva verificável. Nenhuma atestação foi produzida porque há bloqueantes. Não houve selo, envelope, close, commit, push, alteração de policy ou pin de issuer. A spec continua `in-progress`. Não houve delegação a outro agente.

## Rules applied during review

- Tipo: misto, bugfix Go e documentação de contrato.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`; workflow `.pose/workflows/review.md`; manual `POSE.md` e AGENTS raiz/específicos dos manuais traduzidos/scaffold.
- `.pose/rules/backend-go.md`: falhas de leitura, validação, consumidores e concorrência.
- `.pose/rules/security.md`: limites do worktree, identidade, integridade dos registros e inspeção de segredos.
- `.pose/rules/documentation-style.md`: spec, ADR, changelog, manuais e scaffold.
- `.pose/rules/delivery-evidence.md`: proveniência, composição e gates de entrega.
- `.pose/rules/knowledge-governance.md`: busca em knowledge e critérios para eventual aceitação de risco; nenhum risco de aprovação indevida foi aceito, nem handoff de aceitação criado.
- Outros stacks não se aplicam. As regras Go constam do plano mesmo não aparecendo no resultado simplificado de `suggest`.

## Plano e ferramentas

`context` resolveu a autoridade local `proj.pose-dist`, sem ambiguidade. CLI revisada compilada deste HEAD: `/tmp/pose-b07r3`; os checks autoritativos abaixo usam esse binário. O `pose` instalado foi usado somente para a leitura inicial de contexto/plano/suggest.

Bundle **preparado**, não selado: `rvb-627e097bd3f5e166`; digest `sha256:627e097bd3f5e1665bee653174f9c8eddff67013467cd267880bb2ce790bc370`. Plano: `sha256:4eca26fc3e66c59cde82304daf7e4aceffb06538658464bb28a5f3b73e528417`. Subject patch `sha256:b3f35c6ce8dc05ab266747d163cb36afccaced491d58b3e8b7f35cb32079cc97`, tree `sha256:0beab96b6b75a51d4ed13779b6778ee4decbf984f5a706712c9f93b57ba2b7b1`. Componente: `pose-mcp`, criticality high, independência `different-actor`, 15 critérios. A preparação inicial não representa selo nem aprovação e antecede os assessments e resultados desta revisão.

Ferramentas ativas requeridas executadas: artifact-check, surface-check e validate estrito do módulo alterado. A matriz fornece as duas ocorrências de validate do plano, incluindo cobertura do componente. Recomendadas executadas: suggest-review, assess-design, assess-discover com `--if-stale`, assess-integrate e assess-tech-debt. Review-check e closeout-check são completion-only e ficaram diferidos; seal, auto-attest, attest, review verify, close e discover de encerramento não foram executados, conforme a instrução expressa para bloqueantes.

| Check | Resultado |
|---|---|
| `validate --strict --module pose-mcp --json-out .pose/results/b07r3-validation.json` | PASS: 137/137, 350,3 s |
| `go test ./...` e `go build ./...` | PASS na matriz |
| `go test -race ./internal/pose ./internal/cli` | PASS; execução adicional, sem achados do detector |
| `go vet ./...` | PASS |
| `GOOS=windows GOARCH=amd64 go build ./...` | PASS; compilação, sem execução Windows |
| `GOOS=darwin GOARCH=arm64 go build ./...` | PASS; compilação, sem execução Darwin |
| `check --strict` | PASS, 29 avisos |
| `lint-spec pose-git-and-dispatch-snapshot-integrity --strict` | PASS, aviso da seção Tasks opcional |
| `artifact-check --spec ... --strict` | PASS, exit 0; 37 claims/42 paths observados no range, zero erros do escopo; 638 avisos de orphan do repositório |
| `surface-check --spec ... --strict` | FAIL, exit 1: stale-evidence e unconsumed-capability; B10 |
| Driver `b07-mutations.py` | PASS: 48/48 killed, sem survived/error |
| Probes novos, cópias temporárias | 10 asserções falham, 5 controles/observações passam; sem build/setup error |
| `git diff --check origin/main` | PASS |
| Inspeção de segredos no diff e padrões de private key/PAT/AWS | Zero ocorrências nos padrões examinados |

Gitleaks, govulncheck e golangci-lint não estão instalados; não houve mudança de dependências. A inspeção por padrões não equivale a esses scanners. Tech-debt encontrou um TODO coberto, zero dívida descoberta sem cobertura, mas declarou perda de leitura de duas specs antigas; não foi tratada como ausência de dívida. Integração gerou diagnósticos heurísticos, sem tomar consumidor não observado como defeito comprovado. Design observou 26/26 entradas do subject. Recurrence: 393 registros, zero chaves sinalizadas, sem gatilho de escalation. Follow-ups globais foram consultados sem transições de disposição.

Os testes herdados B07/B07r2 estão cobertos pela suíte completa. Em particular, o antigo bloqueante de acesso aos releases (B13 da r2) **não se reproduziu**: `install-and-upgrade-journeys-integration` passou. As 48 mutações mortas não cobrem os contraexemplos adicionais abaixo.

Evidência compacta: [b07r3-independent-review.json](../results/b07r3-independent-review.json). Evidência da matriz: [b07r3-validation.json](../results/b07r3-validation.json). Reprodução dos probes: `python3 -I .pose/results/b07r3-independent-probes.py`; [runner](../results/b07r3-independent-probes.py), [log](../results/b07r3-independent-probes.log), [mutações](../results/b07r3-mutations.json). Exit 1 do runner é esperado neste HEAD: as asserções exigem as garantias violadas. As fixtures alteram apenas repositórios sintéticos nas cópias temporárias.

## Bloqueantes de merge

### B01 — High — Perda da atestação de bundle empatado restaura aprovação

`review_bundle.go:2053–2059`, consumidor `review_closeout.go:2167–2198`. R5/R7.

A nova ordenação coloca uma negativa por último apenas quando consegue ler suas atestações. Erro ou coverage incompleta deixa o bundle na categoria não negativa. A retenção pode então retornar a aprovação do irmão de hash maior antes de visitar o bundle com perda. Probe `TestB07R3TiedBundleLostAttestation`: dois bundles do mesmo instante, negativa de hash menor; controle íntegro recusa, mas tornar `decision` dessa negativa um número faz `VerifyReviewBundle` retornar `Approved=true`, state `closed`. A leitura da negativa retorna erro, ignorado no desempate. Não é uma simples inversão cosmética de lista.

Ação: propagar perda/unknown de qualquer membro pertinente do empate até a retenção/verify e impedir retenção antes de resolver a cobertura do grupo. Não substituir erro por `negative=false`. Examinar também latestNegativeReview e leitores globais de atestações.

### B02 — High — Falha de ls-tree da origem é ignorada

`review_dispatch.go:480–483`, `dispatchNoLongerGoverns:540`. R1/R5/R7.

`sourceModesDiffer` trata tanto erro Git quanto entrada inexistente como motivo para continuar. Um wrapper sintético que falha somente em `git -C <origem> ls-tree` mantém as demais leituras funcionando. O adapter altera o executável da origem com `core.filemode=false`; CLI termina `completed`, exit 0. Probe `TestB07R3SourceModeUnknown`. A falha pode ocorrer já na leitura anterior ao adapter e não impede sua execução.

Ação: retornar estado tipado de leitura, separar ausência comprovada de falha, e recusar a execução/vigência quando não se puder confrontar metadados materiais. Comparação de bytes não responde pelo modo que deixou de ser lido.

### B03 — High — Troca de tipo na origem passa com bytes iguais

`review_dispatch.go:485–487`; produtor `review_bundle.go:1226–1255`, validador `delivery_integrity.go:123–158`. R2/R3/R5.

A origem é preparada por digest de bytes e status; a comparação extra de modo ignora tudo que não é arquivo regular. Com `assume-unchanged`, substituir `pose-mcp/lib.go` por symlink interno a uma cópia de mesmos bytes deixa a preparação igual e o run `completed`. Probe `TestB07R3SourceType`, CLI exit 0. O checkout descartável continua íntegro; o defeito está na pertinência da materialização da origem.

Ação: vincular tipo e alvo de link pertinentes ao que o bundle/commit selou também na origem. Não pular silenciosamente Lstat não regular. Comparar separadamente mudanças legítimas de conteúdo, modo e tipo.

### B04 — High — Symlink em diretório ancestral escapa do worktree

`git_snapshot_integrity.go:254,277,291–304`; consumidor `review_dispatch.go:350`. R2/R3.

`Lstat` examina somente a folha; ancestrais são seguidos. O controle com symlink ancestral visível é recusado porque `ls-files --others` lista o link. A variante esconde o nome do diretório em `.git/info/exclude`, move `pose-mcp` para fora e o substitui por symlink: as folhas externas têm os bytes esperados e a lista de extras ignora o link. CLI retorna `completed`, exit 0. Probes `TestB07R3AncestorSymlink` e `TestB07R3DispatchAncestorLink` são controles conservadores; `TestB07R3DispatchIgnoredAncestorLink` demonstra o bypass.

Ação: verificar cada componente ancestral e a fronteira real dos inputs rastreados, independentemente das regras de ignore. A proteção não pode depender de o link aparecer como untracked.

### B05 — Medium — Perda do registro stale ressuscita o run supersedido

`review_dispatch.go:145–154,396–403`. R5/R7.

O novo mecanismo de correção da janela de publicação depende do registro `Supersedes`. `ListReviewRuns` descarta JSON ilegível, symlinks e outros tipos sem registrar perda. Probe `TestB07R3LostSupersedingRun`: controle completed governa, controle com sucessor stale recusa; após corromper somente o JSON desse sucessor, `ReviewRunStillGoverns` volta a retornar string vazia. O histórico não prova que nenhuma supersession existe.

Ação: retornar cobertura de leitura e recusar vigência quando um registro perdido puder superseder o run. Aplicar a distinção ausência/perda já usada em bundles/atestações. Não foi encontrado consumidor de produção que transforme esse helper em atestação: o defeito demonstrado é no novo contrato de vigência do domínio, não uma atestação CLI produzida pelo probe.

### B06 — Medium — Vigência de run não confere identidade persistida

`review_dispatch.go:153,405–410`. R5/R6/R7.

`ReviewRunStillGoverns` carrega o bundle por ID e passa esse bundle atual ao comparador, sem confrontar `run.BundleDigest`. A leitura dos runs só exige JSON e RunID não vazio, sem verificar identidade do registro. Probe `TestB07R3PersistedRunIdentity`: dispatch válido, controle de vigência íntegro, alteração do bundle_digest no JSON persistido para `sha256:wrong`, releitura e vigência ainda vazia. `TestB07R3RunFields` também demonstra commit inválido passando quando a tentativa de ler seu modo falha (B02).

Ação: validar binding de bundle, scope, sealed commit e integridade do run antes de afirmar vigência. Reusar o bundle atual não comprova o que o registro diz ter revisado. Não se pede assinatura nem pin de issuer; é verificação de campos e digests locais. Mesma limitação de consumo de B05: sem alegar atestação governante criada.

### B07 — Medium — Checkpoint JSON alheio continua autorizado como regeneração

`closeout_plan.go:280–282`. R4.

A correção de tipo da r2 recusa symlink, mas a autorização ainda aceita qualquer arquivo regular `.json` diretamente em `.pose/closeout-plans/`, inclusive rastreado e de outro escopo. Probe `TestB07R3CheckpointScope`: modificar `not-this-scope.json` retorna `ungeneratedChanges=[]`. O teste anterior só cobria troca por link/nomes sem o formato aceito; não provava ownership do output.

Ação: permitir somente o checkpoint exato do escopo/produtor e a ação autorizada. Propagar o escopo para a classificação em vez de deduzir autorização da extensão. O resultado demonstrado é contexto indevidamente limpo, sem alegar close efetivado.

### B08 — Medium — Recuperar leitura não torna válida a baseline unknown

`obligation_projection.go:748–754`. R1/R7.

A revalidação testa apenas `now.Coherent`, embora o relatório da spec declare coerência nas duas leituras. Probe `TestB07R3SnapshotRecoversUnknown`: corromper index, tirar snapshot com `Coherent=false`, restaurar os bytes originais do index; a nova leitura é coerente e `ObligationSnapshotChanges` devolve vazio. Os campos restantes são iguais e a mudança de coerência não é comparada.

Ação: exigir baseline e leitura atual coerentes; invalidar/reobter baseline desconhecida. Não foi encontrado write gate de produção chamando esse helper além dos testes; é violação do contrato de domínio que se anuncia como revalidação para write gate.

### B09 — Medium — TrimSpace troca a raiz solicitada por outro repositório

`git_snapshot_integrity.go:216–218`. R3/R6.

A normalização de subdiretório usa `strings.TrimSpace` na resposta de show-toplevel, removendo bytes válidos de um nome. Probe `TestB07R3RootWhitespace`: repositório `repo ` modificado e clone íntegro `repo` com o mesmo commit; status da raiz solicitada comprova sujeira, mas `WorktreeDivergence` retorna vazio sem erro, verificando o clone. É uma segunda materialização indevidamente usada como prova.

Ação: remover somente o terminador do protocolo e validar a identidade/fronteira da raiz devolvida. Vizinhos `gitHeadAtRoot`, committedFederatedInputs, federatedGitlinkMatches, artifact_ref, spec_transfer e test_catalog também fazem TrimSpace; os que comparam `sameProjectRoot` tendem a recusar ou perder a identificação, em vez de substituir silenciosamente a raiz. A reprodução aqui é no helper de conteúdo; os caminhos temporários gerados pelo dispatch não têm esse espaço final.

### B10 — Medium — Gate canônico de entrega continua sem evidência corrente

`.pose/policy/delivery.json:5`, `.pose/results/delivery-validation.json`. Gate mecânico, sem novo bypass funcional.

`surface-check --strict` termina exit 1 com stale-evidence e unconsumed-capability. O results_path da policy continua apontando ao resultado antigo. A matriz desta revisão passou e está em `b07r3-validation.json`, mas esse nome não substitui automaticamente a fonte canônica consumida pelo gate.

Ação: após os fixes, regenerar o results_path canônico pelo produtor registrado, indexar e obter surface-check estrito verde antes de selo/close. O resultado novo comprova execução; não comprova publicação dessa evidência no gate.

## Follow-ups aceitáveis de baixo risco

Não foram acrescentados à spec nesta rodada porque há bloqueantes; as disposições abaixo ficam propostas para o próximo closeout. Nenhuma variante abaixo autoriza aprovação indevida.

- [open] L01: preservar causa tipada nos erros de Lstat, Readlink e verificação de gitlink em WorktreeDivergence; atualmente geram divergência/false e recusam conservadoramente. Já existe item equivalente aberto na spec, não duplicá-lo. Não inclui o bypass de tipo/ancestral de B03/B04. (owner:@core crit:low review:2026-12-01)
- [open] L02: ordenar StartedAt dos runs por instante, com desempate explícito, em `review_dispatch.go:161`; comparação textual pode inverter histórico com offsets ou precisões diferentes. Os consumidores atuais contam prior_runs/projetam histórico e a checagem de Supersedes percorre todos: esta variante de ordenação não decide aprovação. (owner:@core crit:low review:2026-12-01)
- [open] L03: harmonizar desempate de bundles nas métricas de `governance_replay.go:130` e `governance_waits.go:202`; esses consumidores ainda selecionam/reordenam por instante e hash, desfazendo a regra negative-last. O efeito observado por inspeção é escolha de subject para métricas/contagem de rework, sem gate de aprovação; não confundir com B01. (owner:@core crit:low review:2026-12-01)
- [open] L04: respeitar o results_path configurado ao reconhecer a saída mutável do closeout; `closeoutGenerated` só excepciona o nome delivery-validation.json. `TestB07R3CustomResultsPath` confirma recusa conservadora de arquivo rastreado custom.json produzido na localização declarada pela policy. (owner:@core crit:low review:2026-12-01)
- [open] L05: ler gitlinks com `ls-files --stage -z` no produtor vizinho `reviewBundleGitlinks`; sem -z um nome não ASCII entra no mapa com aspas/escapes, diferente do path real. `TestB07R3QuotedGitlink` mostra a chave quoted; para o path `pose-mcp/module é`, a classificação implementation tenta ler diretório como arquivo e bloqueia, em vez de aprovar indevidamente. Este follow-up cobre essa variante conservadora. (owner:@core crit:low review:2026-12-01)

## Varredura por classe e critérios

| Classe | Consumidores examinados | Conclusão |
|---|---|---|
| Ordenação texto/ID | ReadReviewBundles, atestações, retenção, latestNegativeReview, attempts, replay, waits, outcomes e runs | B01: a nova ordenação ainda perde unknown; L02/L03 limitados a histórico/métricas. O empate íntegro de negativa é recusado no controle. |
| Ausência versus perda | status/HEAD/tree, .git, directoryDigest, coverage de bundles/atestações, runs/supersession, snapshots e closeout | B01/B02/B05/B08; ausência real não justifica ignorar leitura perdida. |
| Campo sem validação | Blockers, Subject, Coherent, WorktreeDirty, PolicyDigest, BundleDigest, SealedCommit, Supersedes e RunID | B02/B03/B05/B06/B08; digest igual e status completed não provam todos os bindings. |
| Paths e materialização | porcelain -z/rename, cache, gitlinks, comparação de conteúdo, allowlists dispatch/closeout, roots/subdiretórios, symlinks e ambiente Git | B03/B04/B07/B09; L04/L05 são recusas indevidas. Probe com GIT_CONFIG_COUNT/core.worktree não redirecionou a leitura neste ambiente e não é achado. |

A pergunta adversarial do workflow foi aplicada: cumprir a forma sem garantir o valor custa conservar bytes/digest/status aparentemente iguais e perder metadados, cobertura histórica ou fronteira de materialização. O corpus `pose-mcp/internal/cli/testdata/adversarial/README.md` cobre atalhos gerais, mas não enumera essas variantes B07; o runner anexo fornece contraexemplos executados sem alterar corpus ou policy.

Os 15 critérios foram avaliados: requirements/correctness/backend-input-errors/security falham nos contraexemplos; backend-contracts/backend-integration-impact falham nas novas garantias de vigência/revalidação; documentation/operability têm afirmações mais fortes que os casos reproduzidos. Scope e compatibility têm suporte na spec/ADR/changelog (recusa mais estrita classificada minor, sem dependências novas); não compensam as falhas. Backend-concurrency tem race verde, sem prova de fechamento transacional da janela. Assumption-integrity, design-causality e solution-proportionality: a direção Git tipado + comparação por conteúdo responde a D1/R1–R6 e o escopo evita CAS/sandbox gerais, mas assumir ausência por falha, folha por caminho íntegro e registro atual por binding comprovado contradiz a causa e requer os fixes. Não foram fabricadas respostas passed nem mappings de atestação. A observação estrutural material é o ADR; a adição isolada de check na matriz é não material pelo contrato do detector.

R1–R7 conservam contraexemplos; R8 tem controles e mutações úteis, porém ainda não cobre as variantes anexas. Manuais/changelog e a narrativa de entrega devem ser reconciliados após os fixes, sem contar cada afirmação ampla como um bloqueante documental duplicado.

## Encerramento

Não há autorização de merge nesta revisão. Corrigir B01–B09, publicar a evidência corrente de B10 e repetir os gates/probes antes de selar. As análises de baixo risco são propostas de backlog, sem alterar a spec ou descartar obrigações existentes. Nenhuma contribuição foi staged/submitted: Contributor Mode exige decisão do usuário, e os achados permanecem apenas como parecer e reproduções sintéticas autorizados neste trabalho.

Assessments/índices derivados alterados pelas ferramentas foram restaurados ao HEAD, exceto `generation.json`, que já estava modificado na entrada e foi deixado sem descarte. Os artefatos próprios da revisão ficam em reports/results sem commit. Nenhum arquivo de implementação foi editado.
