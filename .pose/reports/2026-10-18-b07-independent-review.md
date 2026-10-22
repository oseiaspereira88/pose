# Revisão independente B07 — PR #195

Decisão: **changes-requested**. Dez bloqueantes de merge (B01–B10) e quatro follow-ups de baixo risco (L01–L04). Nenhum selo, atestação, close, commit ou push foi realizado. Policy e pins de issuer não foram alterados.

Revisão de `81258bb7a6691b5daaa712c6cc0af397425df772`, branch `impl/pose-b07-git-snapshot-integrity`, spec `pose-git-and-dispatch-snapshot-integrity`, em 2026-10-18. Base local `origin/main`: `8aa729a76ab1c094690a0892263a86bf3bb7bb42`; merge-base do diff de PR: `1e0a313ba1d87cebd4a355eeb3b83c0bd080e5e8`. Inspecionados o diff `origin/main...HEAD`, os consumidores adjacentes e as correções B06 já em `origin/main`; não se interpreta a diferença two-dot como remoções introduzidas por este PR.

Identidade declarada do reviewer: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI; execução e vendor diferentes do implementador Claude/Anthropic. Esta declaração não constitui identidade autenticada. No pose-dist não se exige envelope para esta revisão; como há bloqueantes, não foi registrada atestação, assinada ou sem envelope.

## Rules applied during review

- Tipo: mixed — bugfix Go, contrato de integridade, ADR, documentação e evidência.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`; workflow `.pose/workflows/review.md`.
- `.pose/rules/backend-go.md`: propagação de erros, subprocessos, comparação de conteúdo e concorrência.
- `.pose/rules/security.md`: fronteiras de paths, inputs, policy e evidência selada.
- `.pose/rules/documentation-style.md`: spec, ADR, changelog e paridade com os manuais/scaffold.
- `.pose/rules/delivery-evidence.md`: rastreio de requisitos, proveniência e gates de entrega.
- `.pose/rules/knowledge-governance.md`: consultada na procura de decisões/riscos prévios; nenhum risco bloqueante foi aceito nem handoff criado.
- React, Kubernetes e demais stacks: não aplicáveis ao diff. AGENTS dos manuais traduzidos e do scaffold também foram consultados.

## Plano efetivo e ferramentas

`pose context --task spec:pose-git-and-dispatch-snapshot-integrity --json` resolveu a autoridade local, status `in-progress`, sem ambiguidade. `pose suggest review --path pose-mcp` foi consultado. Não foi feita escrita cross-project.

Plano `sha256:c7befa40c9e262a3b585a41452445108ace7feb3ec42443dffa6dab4be52b555`: componente `pose-mcp`, criticalidade high, banda critical, independência `different-actor`, 15 critérios. Perfis spec-closeout, backend-review, engineering-judgment, high-criticality-review e structural-materiality. Owner ausente e paths documentais não mapeados são avisos, não evidência de correção.

Bundle apenas **preparado**, nunca selado: `rvb-40594ac04fa38d63`, digest `sha256:40594ac04fa38d6302b2127497cb6d6888b5d15a9a6578d7c0bc5fea99892040`. Digests do subject: patch `sha256:e7cf5944d31d3217b2d22da6728fe6cae9183adcae557fb52c076ebef24fb493`; tree `sha256:ec4b8e229ca7d0a5fc28c3fd4099b7c3537fae0335f01035cde3901af1a5c34d`.

Ferramentas ativas requeridas: artifact-check, validate do módulo e surface-check executados. A validação do único módulo alterado executou seus 136 checks registrados, incluindo defaults e todos os overrides. Ferramentas recomendadas suggest-review, assess-design, assess-discover `--if-stale`, assess-integrate e assess-tech-debt foram executadas para examinar contrato, estrutura e cobertura. Nenhuma recomendada foi omitida. Review-check e closeout-check, dependentes de review completo, foram diferidos; seal, auto-attest, attest, verify e close não foram executados diante dos bloqueantes, conforme instrução desta tarefa.

## Checks e evidência

Evidência compacta: [b07-independent-review.json](../results/b07-independent-review.json). Probes: [runner reproduzível](../results/b07-independent-probes.py) e [extrato](../results/b07-independent-probes.log). O runner copia `pose-mcp` e `mcp-enforce` para diretório temporário; não altera o clone. Seu exit 1 é esperado neste HEAD: as asserções descrevem garantias quebradas, não erro de compilação.

| Check | Resultado |
|---|---|
| `pose validate --strict --module pose-mcp` | PASS — 136/136, 352,0 s |
| `go test ./internal/pose ./internal/cli -count=1` | PASS |
| `go test -race ./internal/pose ./internal/cli` | PASS |
| `go vet ./...` | PASS |
| Builds Windows/amd64 e Darwin/arm64 `go build ./...` | PASS; não equivalem a testes nessas plataformas |
| `pose check --strict` | PASS, 29 avisos |
| `pose lint-spec pose-git-and-dispatch-snapshot-integrity --strict` | PASS; Tasks opcional ausente |
| `pose artifact-check --spec pose-git-and-dispatch-snapshot-integrity --strict --json` | PASS no escopo, 19 claims, ambos os commits atribuídos |
| `pose surface-check --spec pose-git-and-dispatch-snapshot-integrity --strict --json` | FAIL — evidência stale e integração corrente ausente, B10 |
| Driver `b07-mutations.py`, rerun integral independente | 20/20 killed, zero build-error e zero survived |
| `git diff --check origin/main...HEAD` | PASS |
| Inspeção de segredos + padrões de chave privada/PAT GitHub/AWS no diff | Zero ocorrências; não equivale a scan completo de gitleaks |

`gitleaks` e `govulncheck` não estão instalados; não houve mudança de manifests/dependências. Assessment de tech-debt retornou zero marcadores; integração retornou 64 integrações e 63 gaps heurísticos de consumidor não observado, que não foram promovidos automaticamente a defeitos. Discover/design executados; design observou 16/16 arquivos. `recurrence-check --tolerant --window-days 14`: 393 registros, zero chaves sinalizadas; sem gatilho de escalation. Os outputs derivados produzidos por assessments/index foram revertidos ao HEAD após a coleta; ficam somente os artefatos desta revisão em reports/results.

## Bloqueantes de merge

### B01 — High — A pertinência atual dos inputs não é revalidada

Local: `pose-mcp/internal/pose/review_dispatch.go:325–341`. R5, R6, R7.

`dispatchNoLongerGoverns` recarrega o mesmo bundle imutável, compara o digest da pasta de policy e procura o último bundle gravado. Não prepara novamente o escopo nem compara os inputs relevantes da origem. Alterar `pose-mcp/lib.go` ou substituir `R1: works` por `R1: changed` na spec de origem durante o adapter retorna exit 0 / `review_dispatch.status=completed`, sem novo bundle. O arquivo selado ainda existir não prova pertinência atual.

Reprodução: `TestB07IndependentSourceChangedDuringRun`. Controle negativo complementar: remover `.git` da origem invalida o worktree vinculado e é recusado; não foi reportado como bypass.

Ação: comparar a identidade semântica e os inputs pertinentes da origem antes da execução e antes do efeito/consumo; preservar commits documentais alheios. Vincular publicação/consumo à precondição examinada ou conservar somente histórico não governante se a janela não puder ser protegida. O ADR admite a janela após a leitura, mas isso não satisfaz a exigência explícita da spec de recusar o efeito nesse caso. Não requer policy nova nem isolamento completo do adapter.

### B02 — High — Commit selado e inputs do bundle são tratados como a mesma coisa

Local: `pose-mcp/internal/pose/review_dispatch.go:195–199,242–246,300`. R2, R6.

As duas comparações usam somente `plan.SealedCommit`. Um bundle pode selar requisitos atuais que não estão nesse commit de implementação. No probe, a spec selada contém `sealed-new-requirement`; o adapter lê a spec antiga do checkout (`R1: works`) e o run completa. Para isolar B02 do desempate B04, o probe conserva apenas o bundle corrente antes de planejar o dispatch.

Reprodução: `TestB07IndependentBundleInputsNotCommit`.

Ação: verificar/materializar os inputs realmente selados (seções semânticas, configurações e demais digests relevantes), ou recusar o dispatch quando o checkout não os representar. Comparar o checkout consigo mesmo e com seu commit não comprova o bundle.

### B03 — High — Regeneração ainda é autorizada por forma/prefixo, inclusive remoções

Local: `pose-mcp/internal/pose/git_snapshot_integrity.go:145–163,201–202`; consumidor vizinho `closeout_plan.go:75,244–250`. R4.

O dispatch aceita qualquer path em assessments/state e qualquer JSON direto em indexes, exceto um nome. Aceita, por exemplo, `.pose/indexes/not-a-produced-index.json`, `.pose/state/arbitrary.txt` e a remoção de `.pose/state/sealed.txt` rastreado. A autorização é consultada antes de ler existência/tipo/conteúdo, sem produtor ou inputs. No closeout, a whitelist antiga de prefixos continua aceitando alteração da própria `validation-matrix.json` e de resultado selado: `ungeneratedChanges()` devolve `[]`, nil.

Reprodução: `TestB07IndependentDerivedRules` e `TestB07IndependentCloseoutPrefix`.

Ação: usar o conjunto exato de outputs/produtores/inputs e ações permitidas nos dois consumidores; proteger evidência já selada mesmo em árvore derivada. Adição, substituição por link, edição e remoção não podem receber a mesma permissão por path.

### B04 — High — Desempate por hash pode conservar supersession e aprovação indevidas

Local novo: `review_dispatch.go:336–339`; leitores adjacentes `review_bundle.go:1716,1975–1980,2615–2620,2750–2754`, `review_closeout.go:1965–2008`.

Seals truncam o tempo para segundos; ListReviewBundles desempata pelo ID derivado do hash. Um novo bundle no mesmo segundo com ID menor é ignorado pelo dispatch do antigo: no probe `rvb-cfe687f0132cd13b` permanece governante após `rvb-c5eda8b4cc30f2b0`.

A busca da mesma classe nos consumidores encontrou consequência mais grave: aprovação seguida de `changes-requested` no mesmo segundo pode continuar `Approved=true`, pois ListReviewAttestations/Verify/retention escolhem o maior hash. Probe: aprovação `rva-f8759c798e9f05e4`, negativa posterior `rva-bd8433747da6ff33`, selecionada `approved`.

Reprodução: `TestB07IndependentSameSecondSupersession` e `TestB07IndependentNegativeSameSecond`.

Ação: decidir por identidade semântica atual e ordem demonstrável; quando a ordem for ambígua, recusar certificação. A normalização de instantes trazida pela B06 em origin/main resolve offsets, mas `reviewInstantLess` ainda desempata instantes iguais por ID: a mera integração de main não elimina este caso. A variante negativa é herdada, encontrada na varredura exigida; o novo consumidor B07 reproduz a mesma premissa insegura.

### B05 — High — PolicyDigest não prova que a policy foi lida

Local: `review_dispatch.go:209–210,333`; helper compartilhado `obligation_projection.go:711–739`; consumidor adjacente `closeout_plan.go:95`. R1, R5, R7.

`directoryDigest` ignora symlinks e arquivos que não consegue ler; erro de diretório vira a string `unavailable`. No dispatch, string vazia desliga a checagem e `unavailable == unavailable` passa. Um `review.json` consumido por symlink pode ter seu alvo alterado durante o adapter sem mudar o hash: o run completa. Não é ausência de policy nem prova de igualdade; é perda de observação. Closeout e snapshots usam o mesmo helper.

Reprodução CLI: `TestB07IndependentPolicySymlink`; invariantes do domínio: `TestB07IndependentUnknownPolicyDigest`.

Ação: snapshot tipado dos inputs de policy realmente consumidos, com recusa de leitura incompleta e regras explícitas de links/fronteira. Exigir baseline válida no domínio em vez de aceitar campo opcional/vazio ou sentinela como digest. Nenhuma mudança de policy/issuer é necessária.

### B06 — High — Gitlinks selados são ignorados, inclusive quando ausentes

Local: `git_snapshot_integrity.go:196–199`; consumidor/classificação de submódulos em `review_bundle.go:748–753,787–792`. R2, R3.

Todo objeto `kind == commit` é pulado antes de qualquer verificação da materialização. Um gitlink selado e inexistente no disco produz `diverged=[]`, nil. A classificação de bundles reconhece submódulos como subject; declará-los outro repositório não prova que o input pertinente exista no checkout de review. Um worktree recém-criado também não inicializa recursivamente submódulos.

Reprodução: `TestB07IndependentGitlink`.

Ação: verificar materialização e identidade dos gitlinks pertinentes, com recursão/conteúdo conforme o escopo, ou recusar como unknown quando esse input não puder ser verificado. Não certificar ausência como integridade.

### B07 — Medium — A perda de coerência desaparece no consumidor do snapshot

Local: `obligation_projection.go:644–649,668–699`. R1, R7.

A nova leitura corretamente marca `Coherent=false` após corrupção do index Git, mas `snapshotDigest` não inclui coerência e `ObligationSnapshotChanges` não a verifica. Um snapshot inicialmente coerente, seguido de index ilegível, mantém o mesmo digest e retorna `changes=[]`. A documentação dessa API apresenta o resultado vazio como revalidação do estado para um write gate. Não foi localizado consumidor de produção atual dessa função além dos testes; o defeito é no contrato de revalidação do domínio, não uma alegação de que a CLI já publicou uma aprovação nesse probe.

Reprodução: `TestB07IndependentSnapshotRevalidation`.

Ação: propagar unknown/incoerência na identidade e no resultado de revalidação, recusando elegibilidade em vez de representar perda de leitura como ausência de mudanças. Exercitar também o consumidor, não apenas o campo novo do produtor.

### B08 — High — Normalização de backslash concede permissão a outro path no POSIX

Local: `git_snapshot_integrity.go:145–148`. R3, R4.

Git `-z` entrega bytes exatos, mas IsDerivedRegeneration troca todo `\` por `/`. Em POSIX, o arquivo literal `.pose\state\source.go` é um nome no diretório raiz, distinto de `.pose/state/source.go`. Alterá-lo após o commit retorna `diverged=[]`, nil, porque ele ganha a exceção de state.

Reprodução: `TestB07IndependentBackslashAlias`.

Ação: manter a representação de paths do protocolo Git e aplicar somente conversões de plataforma comprovadas, sem transformar caracteres válidos em separadores. Comparar o mesmo path que foi classificado e aberto.

### B09 — High — Bundle superveniente ilegível é interpretado como inexistente

Local: `review_dispatch.go:336–341`; leitor adjacente `review_bundle.go:1948–1960`. R1, R5, R7.

Após selar um bundle novo, truncá-lo para `{` faz ListReviewBundles(scope) ignorá-lo antes da verificação de carga. O dispatch do anterior recebe `reason=""`, embora já tenha ocorrido supersession e a leitura atual seja incompleta. A proteção que verifica o bundle original não cobre perdas dos outros candidatos.

Reprodução: `TestB07IndependentLostSupersedingBundle`.

Ação: consumir cobertura/perdas, não somente a lista sobrevivente. Em origin/main a B06 já fornece `ReadReviewBundles`/`SourceCoverage`; o wrapper legado `ListReviewBundles` descarta a cobertura. O novo consumidor deve exigir leitura completa, inclusive para loss sem atribuição recuperável. A mera integração de main não corrige a chamada nova. A busca vizinha também examinou atestações, retenção, negativos e listagem de runs; não se tratou uma lista vazia como prova de ausência.

### B10 — Medium — O gate de entrega ainda consome evidência de outro estado

Local: `.pose/policy/delivery.json:5`, `.pose/results/delivery-validation.json:4–7`, spec Validation/Requirement trace.

O arquivo canônico consumido pelo gate é de 2026-10-17, HEAD `ff50b4eba9f1df07e0ca3f49b88162647b65c78d`, anterior à implementação B07. Surface-check retorna `stale-evidence` e `unconsumed-capability` por falta de integration corrente. Os 136 checks executados nesta revisão comprovam execução atual, mas não reescrevem automaticamente esse arquivo nem fazem esse gate passar.

Ação: após corrigir os defeitos, regenerar `results_path` pelo produtor registrado no HEAD revisado, indexar e confirmar surface-check verde antes de seal/attest/close. É um bloqueante mecânico de evidência/closeout, não um novo defeito funcional. A evidência compacta desta revisão não foi apresentada como substituta silenciosa do arquivo canônico.

## Follow-ups aceitáveis de baixo risco

São fail-closed, limitação de diagnóstico ou atualização de evidência auxiliar. Nenhum deles permite aprovar conteúdo indevido. Como esta execução termina com bloqueantes, não se alterou a spec nem se aceitou risco em atestação. As disposições propostas para a próxima revisão/closeout são:

- [open] L01: preservar causa tipada de erro ao ler input. `WorktreeDivergence` transforma EACCES/erro de leitura em divergência comum (`git_snapshot_integrity.go:205–224`); o run falha, mas perde a distinção unknown versus edição/ausência. `TestB07IndependentReadErrorDiagnostic` reproduz com modo 000. (owner:@core crit:low review:2026-12-01)
- [open] L02: exigir raiz de worktree ou normalizar explicitamente o root em `WorktreeDivergence`. Um subdiretório limpo é comparado com nomes de `ls-tree --full-tree` relativos à raiz e rejeitado. `TestB07IndependentSubdirectory` retorna `[a.txt dir/a.txt root.txt]`; o dispatch atual fornece a raiz correta, logo a variante observada só causa recusa indevida. (owner:@core crit:low review:2026-12-01)
- [open] L03: tratar metadados de executável no Windows de forma compatível com a plataforma. O teste `info.Mode().Perm()&0111` será falso para arquivos regulares no Go/Windows (`os/types_windows.go` do toolchain usa 0444/0666), recusando árvores limpas com modo Git 100755; há scripts assim no repositório. Inferência estática, sem execução Windows; o build cruzado verde não cobre isso. A falha é conservadora. (owner:@core crit:low review:2026-12-01)
- [open] L04: atualizar `b07-mutations.json` e tornar o driver auxiliar não-zero diante de survived/build-error. O arquivo versionado registra 16 killed/20 e quatro build-error, contrariando a descrição da spec; o rerun independente sobre este HEAD deu 20 killed/20. O driver atual apenas grava contagens e termina normalmente. A suíte registrada passa e o rerun confirma as mutações atuais; isso não é usado para dispensar os bloqueantes nem como gate suficiente. (owner:@core crit:low review:2026-12-01)

## Varredura por classe e limites

| Classe | Consumidores examinados antes de concluir | Resultado |
|---|---|---|
| Ordenação por texto/ID | bundle/atestação, CurrentReviewBundle, Verify, retenção de scopes, latestNegativeReview, attempts, ListReviewRuns, governance waits e versões em origin/main | B04; offsets tratados em B06 não resolvem desempate no mesmo instante. Runs só contam tentativas no fluxo inspecionado; nenhuma conclusão adicional de aprovação foi inferida da sua ordenação. |
| Ausência versus perda | ReadGitStatus/Head, metadata `.git`, bundle original e candidatos, helpers de policy, snapshots, leitores com SourceCoverage de B06 | B05/B07/B09. Diretório realmente exportado sem `.git` mantém a compatibilidade declarada no ADR; não é por si só um novo achado. |
| Campo sem validação | PolicyDigest vazio/unavailable, Coherent versus digest/revalidation, paths `tracked`, conteúdo do SealedCommit versus payload | B02/B03/B05/B07. Validar apenas presença de campo não prova leitura ou integridade. |
| Comparação de paths | status `-z`, os dois lados de rename, cache de status, archive committed, allowlists closeout/dispatch, gitlinks e fronteira do worktree | B03/B06/B08, L02/L03. Controle de link em diretório pai foi detectado como `[dir]`; não se reportou esse caso como bypass. |

A pergunta adversarial foi aplicada: o modo mais barato de cumprir a forma do gate é conservar o JSON do bundle e um checkout do commit, mudar os inputs da origem ou um path que a whitelist chama de derivado, e devolver exit 0. B01/B02/B03 demonstram que a comparação atual ainda aceita essa forma sem comprovar o valor pretendido. O corpus em `internal/cli/testdata/adversarial/README.md` cobre atalhos gerais, mas não declara esses casos B07; o check B07 registrado passa sem exercitá-los. Os probes anexos documentam os gaps, sem alterar corpus ou policy nesta revisão.

Compatibilidade: `stale` é um novo resultado público de run, e a CLI já retorna não-zero quando o status difere de completed. A mudança mais estrita está declarada como minor no ADR/changelog; não há migração de issuer ou adoção de capability. A documentação afirma garantias mais amplas que as comprovadas em B01–B09 e deve ser reconciliada com os respectivos fixes. A janela de publicação admitida pelo ADR não foi aceita como dispensa de R5.

Não se extrapolou para CAS geral, quota/ledger ou sandbox do adapter. Os problemas descritos podem ser corrigidos nos contratos de observação, materialização e pertinência desta spec. Uma revisão posterior deve repetir os probes após os fixes, executar a matriz e regenerar a evidência consumida antes de qualquer selo.

Contributor Mode: as oportunidades foram registradas neste parecer, com reproduções sintéticas. Nenhum draft foi criado em `.pose/contributions/` e nenhuma issue upstream foi enviada; essas ações dependem de decisão explícita do usuário.
