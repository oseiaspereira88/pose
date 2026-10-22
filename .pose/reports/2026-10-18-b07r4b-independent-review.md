# Revisão independente B07r4b — PR #195

Decisão: **changes-requested — bloqueia merge e closeout**. Dez achados bloqueantes (B01–B10) e seis follow-ups aceitáveis, limitados às variantes conservadoras ou informativas descritas abaixo. Nenhum bloqueante foi aceito como risco.

Revisão de 2026-10-18 do HEAD `9a4f17b46b2ec2d42e9679c277f6ac72c4a7f803`, branch `impl/pose-b07-git-snapshot-integrity`, spec `pose-git-and-dispatch-snapshot-integrity`. A consulta `gh pr view 195 --repo oseiaspereira88/pose` confirmou HEAD, branch, base main e PR aberto: [PR #195](https://github.com/oseiaspereira88/pose/pull/195). Base local `origin/main`: `8aa729a76ab1c094690a0892263a86bf3bb7bb42`. Foram examinados o diff completo, a spec R1–R8, ADR, testes, correções das rodadas anteriores, manuais/scaffold, matriz e consumidores adjacentes das quatro classes solicitadas.

Identidade declarada do revisor: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI; implementação atribuída a Claude/Anthropic pelo solicitante, portanto vendor diferente. Esta é declaração de identidade e separação, sem assinatura ou identidade criptograficamente verificada. Há bloqueantes: **nenhum bundle selado, atestação, close, commit ou push nesta rodada**. Policy e issuer não foram alterados. O fluxo sem envelope solicitado só seria executado na ausência de bloqueantes.

## Rules applied during review

- Change type: bugfix com mudança de contrato de integridade; classificação minor explicitada no ADR.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`.
- Workflow: `.pose/workflows/review.md`; contratos AGENTS da raiz e das cópias localizadas/scaffold.
- `.pose/rules/backend-go.md`: erros, validação de registros, subprocessos e fronteiras de leitura.
- `.pose/rules/security.md`: integridade, paths, segredo e dependências; sem ampliar para segurança ofensiva ou alterar policy.
- `.pose/rules/delivery-evidence.md`: distinguir checks verdes de requisitos realmente satisfeitos e de entrega corrente.
- `.pose/rules/documentation-style.md`: spec, ADR, parecer e cópias documentais.
- `.pose/rules/knowledge-governance.md` e `_base-recurrence.md`: consulta de memória e classificação de recorrência. A busca em `.pose/knowledge/` não encontrou decisão operacional B07 adicional. Não há aceitação de risco pós-merge; o parecer é o handoff de correção, sem novo artefato de knowledge.
- Regras de frontend/infra não se aplicam: não há mudança nessas superfícies. As instruções de Go da extensão instalada são cumulativas; `suggest review --path pose-mcp` também resolveu security/documentation-style.

## Escopo, plano e execução

`context` resolveu autoridade local `proj.pose-dist`, spec in-progress, sem ambiguidade ou transferência. O plano foi preparado sem `--seal`. Digests, contexto, critérios e disposição de tools estão na [evidência compacta](../results/b07r4b-independent-review.json). O plano tem 15 critérios e 11 entradas de tools; componente pose-mcp high-criticality, com aviso preexistente de owner ausente. O change set atribui nove commits; o range contém trabalho de B06 e commits de reindexação. A diferença claims/observed é registrada separadamente da revisão semântica; não foi inferida ausência de trailer pelo parser de trailers do Git.

CLI autoritativa compilada do HEAD: `/tmp/pose-b07r4b` (`go version -m` confirma a revisão). O `pose` instalado é de `8aa729a7` e foi usado inicialmente para contexto/plano/suggest e diagnósticos; verificações autoritativas foram executadas também com a CLI compilada. Os testes Go sempre usam o código do clone. As evidências separam as duas execuções.

Tools requeridas ativas: artifact-check, surface-check e validação estrita. Recomendadas executadas: suggest-review, assess-design (28/28 arquivos observados), discover (assessment reutilizada por freshness), integrate e tech-debt. `review-check` e `closeout-check` continuam deferred por exigirem review-complete; verify/close/atestar/selar não foram executados diante dos bloqueantes. Nenhum critério de julgamento foi considerado aprovado apenas por testes verdes. Correctness, requirements, input-errors, concurrency/materialização, scope e evidence têm contraexemplos abaixo; os demais critérios não constituem decisão positiva isolada.

## Checks e evidência

A tabela final de comandos, exits e resumos está em [b07r4b-independent-review.json](../results/b07r4b-independent-review.json). Logs dos probes: [b07r4b-independent-probes.log](../results/b07r4b-independent-probes.log); runner reproduzível: [b07r4b-independent-probes.py](../results/b07r4b-independent-probes.py). Execute `python3 -I .pose/results/b07r4b-independent-probes.py` na raiz; exit 1 é esperado neste HEAD, porque as asserções exigem garantias que o código viola. As cópias do código e os repositórios sintéticos são temporários; nenhum teste adicional foi instalado no código de produção.

Checks concluídos antes da consolidação: suíte focada B07 com `-race` PASS; builds Windows/amd64 e Darwin/arm64 PASS; govulncheck v1.4.0 PASS sem vulnerabilidades encontradas; gitleaks v8.21.2 no diff Git PASS (nove commits, saída redigida); `git diff origin/main...HEAD --check` PASS; lint-spec estrito PASS com aviso de Tasks opcional. A matriz da CLI deste HEAD passou em **137/137 checks, 355,3 s**, incluindo build, go test ./..., vet e integrações registradas; [resultado completo](../results/b07r4b-current-validation.json). O driver de mutações corrente passou em **46/46 killed**, sem survived/error; [resultado](../results/b07r4b-mutations.json). Esses controles não cobrem os contraexemplos novos desta revisão. O surface-check estrito falha por evidência stale e ausência de integração corrente (B10).

`pose check --strict` da CLI deste HEAD passou com 29 avisos preexistentes. `artifact-check --strict` passou: 45 claims, 50 observed, nove commits atribuídos e 638 avisos globais de orphan, sem erro de reconciliação. `surface-check --strict` falhou (B10).

Tech-debt encontra um TODO coberto, mas declara leitura **INCOMPLETE** por duas specs antigas inválidas (`pose-instance-engine-version-tracking` e `pose-knowledge-durable-reference-type`); não se afirma ausência de dívida nas fontes perdidas. Integrate reporta gaps heurísticos preexistentes de consumidores; não foram convertidos em defeitos funcionais B07 sem reprodução. O check de recorrência não sinalizou keys acima do threshold; as repetições efetivamente observadas são classificadas mais abaixo.

## Bloqueantes de merge

### B01 — High — A identidade do worktree pode ser trocada por outro checkout limpo

Local: `pose-mcp/internal/pose/git_snapshot_integrity.go:216` e uso pós-adapter em `review_dispatch.go:369`. R2/R3/R6.

O comparador aceita o top-level devolvido pelo Git e substitui a raiz pedida, sem provar que esse Git corresponde à materialização que foi entregue ao adapter. `TestB07R4BCheckoutIdentity` cria um clone limpo do mesmo commit, altera `pose-mcp/lib.go` no checkout de review e troca seu gitfile para o Git do clone, com `core.worktree` apontando ao clone. A leitura final compara o clone; HEAD continua igual. CLI responde **exit 0, completed**. `TestB07R4BOtherWorktreeConfig` reproduz diretamente: `core.worktree` aponta a outro checkout e o root alterado recebe `divergence=[]`, nil. Não é a variável GIT_DIR que a correção anterior já remove.

Ação: vincular e revalidar a identidade canônica de diretório, Git e worktree da materialização entregue ao adapter; recusar redirecionamento ou troca, inclusive com HEAD igual. A comparação deve continuar lendo o diretório originalmente entregue.

### B02 — High — Falha de leitura do top-level vira fallback capaz de certificar limpo

Local: `git_snapshot_integrity.go:216`. R1/R3/R6/R7.

Erro de `rev-parse --show-toplevel` é ignorado. Para chamada em subdiretório, a árvore completa do commit passa a ser comparada relativamente ao subdiretório. `TestB07R4BUnknownTopLevel`: o controle com raiz conhecida detecta o arquivo alterado na raiz real; falhar somente nessa leitura faz o mesmo comparador ler uma cópia íntegra sob subdiretório ignorado e retornar **[]/nil**. Não há erro tipado. A leitura de object-format também tem fallback silencioso; não se atribui a ela outro bypass sem prova.

Ação: propagar unknown se a raiz necessária não puder ser determinada e validar o vínculo com a raiz solicitada. Não reaproveitar diretório relativo como baseline conhecida.

### B03 — High — HEAD desconhecido da origem não impede completed

Local: `review_dispatch.go:543` (`dispatchNoLongerGoverns`); contraste com `closeout_plan.go:91` e HEAD do checkout pós-run. R1/R6/R7.

A pertinência repara digests/status da origem, mas não exige leitura conhecida do seu HEAD. `TestB07R4BSourceHeadUnknown` usa wrapper com newlines reais e falha 97 somente em `-C <origem> rev-parse --verify HEAD`, depois de o adapter começar. A injeção é confirmada por `ReadGitHead(origem)` ao final. O CLI ainda retorna **exit 0, completed**. O commit do grafo de entrega não prova que o HEAD atual foi lido. O controle positivo de commit documental independente deve continuar passando; não se pede igualdade indiscriminada de HEAD.

Ação: exigir identidade Git legível da origem antes/depois e no consumo, distinguindo avanço documental legítimo de falha de leitura. Closeout e snapshot já têm recusas que devem ser preservadas.

### B04 — High — A verificação de ancestrais não cobre origem e gitlinks

Locais: `review_dispatch.go:519`; `git_snapshot_integrity.go:247` antes de `linkAbove` em 262. R2/R3.

A correção de ancestral vale para blobs rastreados do checkout, mas os outros dois caminhos a contornam. `TestB07R4BSourceAncestor` marca o subject assume-unchanged, move o diretório da origem para fora e instala symlink ancestral; bytes e folha são iguais. CLI: **exit 0, completed**. `TestB07R4DerivedGitlinkAncestor` move o pai de um gitlink materializado para fora, instala symlink e ignora o nome do pai; a verificação devolve **[]/nil**, pois o ramo `kind == commit` executa antes de `linkAbove`.

Ação: aplicar a mesma verificação de fronteira e ancestrais em todas as materializações pertinentes, antes de seguir gitlinks ou ler folhas. O controle de ancestral extra não rastreado visível é recusado corretamente; não foi contado como outro defeito. Closeout e leitores de artefatos também foram examinados; a recusa visível de symlink não demonstra a segurança dos dois ramos reproduzidos.

### B05 — High — A ausência selada de um subject removido não é revalidada

Locais: `review_dispatch.go:465,505`; `review_bundle.go:790`; `git_snapshot_integrity.go:298`. R2/R5.

O subject removido não tem digest, é pulado na comparação dos inputs e na materialização da origem; só sobra o status. `TestB07R4BSourceRemoved` sela um subject com ação removed e reintroduz esse path na origem, ignorando o arquivo em `.git/info/exclude`. O CLI retorna **exit 0, completed**. A ausência é parte do subject revisado. O ramo de rename compara o novo path, mas tampouco verifica a ausência do antigo; esta variante foi localizada por inspeção da mesma classe, sem alegar uma segunda execução CLI.

Ação: revalidar explicitamente ausência dos removidos e do lado antigo de renames, independentemente de ignore/status. Manter conteúdo/tipo do lado novo na mesma comparação.

### B06 — High — A autorização de regeneração continua por formato/prefixo, sem produtor e inputs

Locais: `git_snapshot_integrity.go:170`; consumidor `closeout_plan.go:270`. R4.

`TestB07R4ArbitraryRegeneration` escreve conteúdo arbitrário, sem execução de produtor, em assessments/arbitrary.md, state/components/arbitrary.json, results/arbitrary.txt, indexes/generations/not-a-generation/arbitrary.json, review-bundles/arbitrary.txt, review-runs/arbitrary.txt e reports/history/arbitrary.txt. Para todos, **WorktreeDivergence=[]/nil e ungeneratedChanges=[]/nil**. Restringir extensão/profundidade não é vincular outputs à operação autorizada; quatro ramos continuam aceitando qualquer novo path sob prefixos. A fixture corrente de `TestDerivedRegenerationIsAListNotAPrefix` não cobre esses negativos.

Ação: autorizar o conjunto de outputs pelo produtor realmente admitido, com inputs/baseline e ações permitidas; verificar nomes de geração/arquivos e formatos dos novos registros. Compartilhar o contrato entre dispatch e closeout, preservando a proteção já existente da matriz, policy e resultados rastreados.

### B07 — High — O histórico de runs aceita registros incompletos e sem integridade de identidade

Locais: `review_dispatch.go:143,171,411,597`. R5/R7.

`ReadReviewRuns` só exige JSON e RunID não vazio. `TestB07R4PartialSupersedingRun`: o controle com sucessor stale impede vigência; reduzir somente esse sucessor a `{run_id: <id>}` resulta em **lost=[]** e restaura o completed antigo. `TestB07R4BRunFields` confirma schema futuro, RunID incompatível com filename, Output alterado, instante inválido, exit code não zero e Supersedes alterado aceitos sem perda. `TestB07R4BActualRunIdentity` usa um dispatch CLI real: alterar seu Output persistido mantém `ReviewRunStillGoverns==""`.

O consumidor também não exige encontrar o registro corrente nem confronta a cópia passada com os bytes persistidos: `TestB07R4BMissingRun` apaga o registro e a cópia em memória continua governando; a variante status em `RunFields` muda o registro para failed, mas a cópia original continua aceita. `TestB07R4BRunRootLink` mostra leitura de storage movido para symlink externo, diferente da fronteira exigida pelo escritor e pelos bundles/atestações vizinhos.

Ação: validar schema, campos obrigatórios, instantes, estados, relações e identidade derivada; tratar truncamento semântico como perda; exigir registro persistido íntegro e equivalente no ponto de consumo; aplicar a fronteira de storage também na leitura. Não envolve assinatura, envelope ou pin de issuer. Não se alega uma atestação CLI aprovada: o defeito reproduzido é na certificação de vigência do run e na sustentação da correção da janela de publicação.

### B08 — Medium — Transcript perdido ou adulterado não invalida a vigência do run

Locais: `review_dispatch.go:411,597`. R5/R7.

`TranscriptDigest` é escrito, mas nenhum leitor o confronta com o arquivo. `TestB07R4RunTranscript` grava um completed com transcript, confirma controle íntegro, apaga o transcript e recebe novamente **ReviewRunStillGoverns==""**. Os subcasos transcript_digest/transcript_path de `RunFields` também continuam aceitos após corrupção. O histórico deixa de demonstrar o que declara conservar. A integridade do JSON (B07) não basta para comprovar o arquivo separado.

Ação: validar path, tipo, existência e digest do transcript ao ler/consumir um run pertinente; perda deve permanecer identificada, sem certificar fulfillment vigente.

### B09 — Medium — O planejador descarta a cobertura perdida dos runs e executa além do limite conhecido

Locais: `review_dispatch.go:135,241,274`. R7.

O novo `ReadReviewRuns` expõe loss, mas o wrapper `ListReviewRuns` o descarta; `PlanReviewDispatch` usa o wrapper para PriorRuns. `TestB07R4BLostBudget`: adapter limitado a um run, primeira execução completa; corromper o JSON dessa execução e chamar novamente produz **prior_runs=0, exit 0, completed**. A leitura desconhecida virou ausência e permitiu outra execução. É a propagação da cobertura existente, não pedido para implementar o ledger/quota geral de outro lote.

Ação: usar leitura com cobertura no planejamento e recusar execução quando as perdas puderem afetar o limite/precondições. Conservar histórico sem converter falha persistente em baseline vazia.

### B10 — Medium — A evidência canônica de entrega ainda está stale

Locais: `.pose/policy/delivery.json:5`, results_path canônico e trace R1–R8. R8 e delivery-evidence.

O surface-check estrito da CLI deste HEAD retorna `stale-evidence` e `unconsumed-capability`: falta integration corrente para capability:git-and-dispatch-snapshot-integrity. O resultado independente com outro nome não substitui automaticamente o results_path canônico. Este é bloqueante mecânico de entrega/closeout, separado dos defeitos funcionais.

Ação: depois de corrigir B01–B09 e passar validação, regenerar a evidência no results_path declarado, indexar e comprovar surface-check estrito antes de selar. Esta revisão negativa não substituiu o arquivo canônico para aparentar entrega.

## Follow-ups aceitáveis (propostos; não escritos na spec nesta rodada)

As variantes abaixo não tornam aprovação indevida possível. O item L01 já está aberto na spec; os demais devem permanecer abertos com ownership no closeout futuro. Não houve disposition de spawned/covered/duplicate.

- [open] L01: preservar a causa tipada de Lstat, Readlink e verificação de gitlink; os erros hoje recusados como divergência perdem diagnóstico. Limitado aos ramos que já recusam, sem incluir B01–B04. (owner:@core crit:low review:2026-12-01)
- [open] L02: ordenar StartedAt dos runs por instante e desempate explícito. TestB07R4BRunOrdering demonstra que .1Z fica antes de Z no mesmo segundo. A checagem de Supersedes percorre todos os runs; essa variante só distorce histórico/projeção. (owner:@core crit:low review:2026-12-01)
- [open] L03: harmonizar desempate de bundles em governance_replay.go:130 e governance_waits.go:202, que reordenam por instante/hash e desfazem negative-last para métricas e rework. Não são gates de aprovação. (owner:@core crit:low review:2026-12-01)
- [open] L04: respeitar results_path configurado na exceção mutável de closeout; closeoutGenerated ainda fixa delivery-validation.json. TestB07R3CustomResultsPath confirma recusa conservadora do output rastreado customizado. (owner:@core crit:low review:2026-12-01)
- [open] L05: ler gitlinks com ls-files --stage -z em reviewBundleGitlinks; nomes não ASCII recebem quoting e classificação errada. TestB07R3QuotedGitlink registra a chave quoted; no path sob componente, ler diretório como blob bloqueia conservadoramente. (owner:@core crit:low review:2026-12-01)
- [open] L06: distinguir path literal com metacaracteres de uma expressão glob na validação de artefatos; a gramática atual recusa nomes Git válidos contendo colchetes. A variante examinada recusa o bundle antes do dispatch, sem aprovação indevida. (owner:@core crit:low review:2026-12-01)

## Varredura das classes e contratos

| Classe | Consumidores vizinhos examinados | Resultado |
|---|---|---|
| Ordenação por texto/ID | bundles, atestações, retenção, latestNegativeReview, attempts, obligations, replay, waits, outcomes, runs | Controles negativos de empate agora recusam; L02/L03 são limitados a histórico/métricas. Não atribuir a eles os bypasses funcionais. |
| Ausência versus perda | HEAD/status/tree/top-level, metadata Git, digest de policy, snapshots, closeout, coverage de bundles/atestações, runs e transcripts | B02/B03/B05/B07/B08/B09; baseline ilegível não é ausência. Controles de snapshot unknown persistente e unknown recuperado são exercitados pela matriz/regressões. |
| Campo sem validação | schema/ID/filename do run, estado/exit code, Supersedes, Output, timestamps, transcript, Coherent/dirty, PolicyDigest, blockers e bindings bundle/scope/commit | B07/B08/B09. As verificações de bundle_digest/scope/sealed_commit introduzidas na r4 não demonstram os demais bindings. |
| Comparação de caminhos | source/checkout, subdiretórios/raízes, configuração/env Git, tipos/modos/links/ancestrais, gitlinks, removed/rename, derivados e storage | B01/B02/B04/B05/B06/B07; controles visíveis recusam. L04/L05/L06 ficam restritos a falsas recusas. |

Pergunta adversarial do workflow: a maneira barata de cumprir a forma sem produzir integridade é manter bytes/HEAD/digest aparentes e trocar a materialização consultada, perder a supersession semanticamente, ou nomear conteúdo arbitrário como output derivado. Os controles atuais não impedem os contraexemplos acima. O corpus `pose-mcp/internal/cli/testdata/adversarial/README.md` foi consultado; não enumera estas novas variantes. O runner anexo é a evidência adversarial executada, sem modificar corpus/policy.

Contratos CLI/MCP e arquivos: a recusa mais estrita é explícita no ADR e não muda schemas públicos; contudo, o novo contrato de vigência de runs ainda não preserva todas as perdas e bindings. O domínio de dispatch é exposto pelo CLI; não foi encontrado consumidor MCP direto nem conversão run-to-attestation em produção. Isso limita a alegação de impacto: os `completed` e a vigência incorretos foram reproduzidos, sem alegar uma aprovação assinada/inexistente. Builds cruzados não substituem testes runtime Windows/Darwin.

## Recorrência e entrega ao implementador

Domínio implementação/contrato: prova parcial aceita como integridade total em leitores irmãos (folha/ancestral, bytes/ausência, digest/registro, registro/transcript). Domínio validação/processo: fixtures e mutantes cobrem correções anteriores, mas deixam variantes adjacentes; evidência canônica stale reaparece nas revisões. Prevenção: corrigir as primitivas compartilhadas e acrescentar negativos/controles positivos nos consumidores reais antes de acrescentar mais exceções de path. O recurrence-check nativo não sinalizou threshold; não se fabricou uma escalada formal nem contribuição upstream. Qualquer contribution draft continua dependendo do consentimento explícito do usuário.

Não há condição segura de merge sob os requisitos atuais enquanto B01–B10 persistirem. Corrigir o conjunto, reexecutar matriz/probes, reconciliar a evidência canônica e só então realizar revisão/closeout sem envelope. Os arquivos gerados de assessment/state já estavam modificados no início; as tools atualizaram essas projeções e índices derivados. Não foi alterado código de produção, spec, policy ou lifecycle, e nada foi commitado.
