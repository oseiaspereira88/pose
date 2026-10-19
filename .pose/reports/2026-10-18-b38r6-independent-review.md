# Revisão independente — b38r6 / PR #198

Decisão: **changes requested**. **Seis bloqueantes de merge e oito follow-ups
aceitáveis, low.** Spec `pose-release-manifest-tag-integrity`, HEAD
`3a6962738d11e6469996c55eedb8ac6c1552e6b3`, base `origin/main`
`8aa729a76ab1c094690a0892263a86bf3bb7bb42`, repositório `oseiaspereira88/pose`.

Revisor `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI; implementador
Claude/Anthropic. Identidade e separação declaradas, vendors diferentes; sem
alegar autenticação criptográfica ou independência cognitiva observada. Pela
instrução do caso bloqueante, **não houve seal, attest (com ou sem envelope),
verify de aprovação, close, commit ou push**. Policy e issuer pins não mudaram.

[Evidência compacta, fontes reproduzíveis dos probes e resultados](../results/b38r6-independent-review.json).
Probes executados exclusivamente em cópia temporária dos fontes deste HEAD,
com o módulo irmão mcp-enforce. Nenhum fonte ou teste do clone foi modificado.
Não houve delegação. Todos os achados estão consolidados neste parecer.

## Rules applied during review

- Change type: mixed — correção Go, contratos de arquivo/CLI/MCP, testes e docs.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`.
- Workflow: `.pose/workflows/review.md`.
- [x] `.pose/rules/backend-go.md`: domínio/consumidores, erros e checks Go.
- [x] `.pose/rules/security.md`: integridade de inputs/paths/evidência e scanners.
- [x] `.pose/rules/documentation-style.md`: spec, manuais/scaffold e parecer.
- [x] `.pose/rules/delivery-evidence.md`: gates executados separados da correção semântica.
- [x] `.pose/rules/knowledge-governance.md`: consulta a decisões/notas existentes.
- [x] `.pose/rules/_base-recurrence.md`: classes recorrentes e prevenção.
- Não aplicáveis: frontend, Kubernetes, Terraform e Workers, sem delta.

Leitura: spec/template, workflow/regras, diff de produção e testes contra
origin/main, ADR-018, ADR de archival de 2026-09-11, proposta e decision-log
temporal de 2026-10-14, revisões anteriores e seus follow-ups. Índices/resultados
foram tratados como derivados, sem substituir a inspeção do runtime. Escopo B38;
replay de provider evidence (B39), exclusividade de seleção (B40) e recuperação
transacional/publicação (B41) não foram impostos retrospectivamente.

## Plano, identidade do objeto e execução

`pose context --task spec:pose-release-manifest-tag-integrity --json` resolveu
a autoridade local sem ambiguidade. Bundle apenas **preparado**, não selado:
`rvb-ff56d972cf39f338`.

- Bundle digest: `sha256:ff56d972cf39f338da360b7c9866cda3394e4876347073e2a7f2875e2f2cc0da`.
- Plan digest: `sha256:2701e435116e9327e2d11a3e86836f2535d058037ede660f9e724739da87e20f`.
- Patch digest: `sha256:2b9e160ad3b21e527e6f1dcbf601439585738602b77ff305914ff61469ca2b93`.
- Tree digest: `sha256:d366c46b33426e633b01450c9234e9a7bc1975c97ba11fa4616662d0cc78d1ed`.

Plano com 15 critérios, band critical, different-actor e componente pose-mcp;
metadata owner ausente é warning de cobertura do plano. Perfis spec-closeout,
backend-review, engineering-judgment e high-criticality-review. Foram conferidos
os trailers no corpo dos commits e as claims contra observed, sem usar o parser
Git de trailers como autoridade de atribuição. O range atribui 17 commits em
37 commits de provenance; trabalhos de outra spec não foram confundidos com
artefatos deste subject. O diff contra origin/main abrange 51 arquivos.

Ferramentas requeridas ativas executadas: validate global/módulo, artifact-check
e surface-check. Recomendadas executadas: suggest-review, assess-design,
assess-discover --if-stale, assess-integrate e assess-tech-debt. Discovery
refrescou o componente por mudança de conteúdo/matriz. Recurrence-check de 14
dias, lint estrito e followups --all também executados. Ferramentas completion-only
(review-check/closeout-check), seal/attest/verify/close ficaram diferidas pela
instrução explícita de não aprovar nem fechar com bloqueantes. Não se regenerou
o `results_path` de closeout para selar uma revisão que pede correções.

Binário de validação/check: `/tmp/b38r6-pose`, compilado dos fontes deste HEAD.
As consultas iniciais de contexto/plano/assessments usaram o pose instalado,
cuja build é de origin/main; artifact/surface foram repetidos com a build do HEAD.
As constatações de runtime abaixo usam exclusivamente os fontes/build do HEAD.

- `validate --strict` global: **140/140 pass**, 338,3 s; inclui os 3 checks de mcp-enforce.
- `validate --strict --module pose-mcp`: **137/137 pass**, 336,6 s; inclui build, go test ./..., vet e os checks race registrados.
- `check --strict`: **FAILURE, 2 erros, 29 warnings**, detalhados em R6-F06.
- `artifact-check --spec ... --strict`: 42 claims, 51 paths observados, 639 findings warning de orphan, nenhum error/critical.
- `surface-check --spec ... --strict`: **FAILURE**, stale-evidence e unconsumed-capability; a policy lê delivery-validation.json, não os resultados independentes desta revisão. Não se selou evidência anterior nem se afirmou prontidão de closeout.
- `lint-spec ... --strict`: sucesso; somente warning de seção opcional Tasks.
- `followups --all`: 314 itens, 111 open, 16 overdue, 42 unowned; coverage incompleta pelas duas specs preexistentes inválidas, sem disposições alteradas.
- `govulncheck ./...`: **No vulnerabilities found**.
- `gitleaks git --log-opts=origin/main..HEAD`: **17 commits, no leaks found**.
- `git diff --check origin/main...HEAD`: sem falhas.
- História real: **75 manifests carregados**; inventário independente de **81 tags: 74 snapshots íntegros, 7 sem snapshot, zero outros erros Git**. Ausência de snapshot foi contada explicitamente, sem transformar qualquer erro em legacy. O worktree de uma release tagged ainda pode divergir da tag (F06).
- Tech-debt: 1 TODO coberto, zero uncovered; scan **INCOMPLETE** por duas specs preexistentes inválidas (pose-instance-engine-version-tracking e pose-knowledge-durable-reference-type).
- Integrate: gaps observacionais de consumidores MCP/HTTP, incluindo /healthz e /mcp; não são prova de quebra introduzida por este delta.
- Discovery: refreshed por conteúdo/matriz; design: 28/28 arquivos observados e um delta de validation-check, sem novo contrato estrutural detectado.


Os probes negativos falham deliberadamente ao exigir uma recusa ou preservação
histórica que falta. As falhas reproduzem os achados; não são falhas dos checks
registrados. A evidência inclui os fontes dos probes e a saída integral.

## Bloqueantes de merge

### R6-F01 — high — Exceção Compatibility exclui conteúdo de fragments

Locais: `pose-mcp/internal/pose/release_candidate.go:71`, `:92`.
R2/R3/R7. Probes `TestB38R6CompatibilityCategory` e
`TestB38R6CompatibilityConsumerParity`.

Uma policy de changelog válida pode declarar a categoria customizada
`compatibility`. O renderer apresenta seus fragments sob `## Compatibility`.
`notesWithoutCompatibility` remove a primeira seção com esse título, sem
identificar se ela veio do registry ou dos fragments. Alterar somente o texto
dessas notas e atualizar NotesDigest mantém os blobs e o Body do manifest
originais, mas a divergência material desaparece da comparação.

Reprodução: Body autoritativo `Required content.`, notas `False content.`.
Candidato retorna gaps vazios e um limit; snapshot tagged retorna Intact=true;
release check retorna gaps vazios; status retorna Sound=true; release notes
retorna código 0 e imprime o texto falso. Não é apenas diferença histórica do
registry. O próprio conteúdo de entrega deixou de ser recomposto.

Vizinhos: candidato e snapshot chamam o mesmo helper; check, notes, record,
status/MCP e checker herdam essa aceitação. Publicclaims/open-next exigem Sound,
mas essa guarda não corrige um snapshot erroneamente intacto. Não se afirmou
que este probe, sem provider events, basta para declarar publicação.

Ação: distinguir a seção gerada pelo registry das seções de fragments e nunca
excluir uma categoria autoritativa da recomposição. Preservar os controles
históricos legítimos v1.1.0/v2.0.0 sem reescrever suas notas/tags.

### R6-F02 — high — Ancestral não-tree na tag vira ausência de policy

Locais: `pose-mcp/internal/pose/release_manifest_integrity.go:321`, `:330`, `:388`.
R3/R4/R7. Probes `TestB38R6GitPolicyAncestor` e `TestB38R6PolicyFileAncestor`.

`gitTreeEntry` consulta só o path completo. Se `.pose/policy` é um symlink ou
um arquivo comum no tree, `ls-tree .pose/policy/release.json` devolve vazio.
O helper converte isso em ausência, sem verificar que um ancestral não é tree.
`releasePolicyAt` então usa a policy legacy, enquanto changelogPolicyAt usa
defaults. Com a policy legacy copiada do cut, o snapshot responde
**Intact=true, issues=[], gaps=[]** nos dois casos.

Vizinhos: o worktree agora usa ProvenAbsent e detecta essa perda/tipo; `gitBlobAt`
também recusa corretamente link, diretório ou submódulo no path final. Falta a
mesma guarda nos ancestrais do snapshot. gitTreeNames e a leitura da fonte de
versão usam o mesmo helper, portanto também perdem precisão do diagnóstico;
para policies, a perda permite fallback/defaults e falsa integridade.

Ação: percorrer os ancestrais no tree, exigindo tree a cada nível; separar
entrada legitimamente inexistente de blob/link/gitlink que impede a leitura.
Não seguir o link usando o worktree nem aplicar policy atual à tag.

### R6-F03 — high — Inventário perdido ainda some do checker/plan

Locais: CLI `pose-mcp/internal/cli/release_lifecycle.go:119`, `:831`.
R3/R7. Probes `TestB38R6CheckerLostReleases` e `TestB38R6PlanLostInventory`.

A nova passagem do checker só executa quando ReadDir(releases) tem sucesso e
não trata o else. Substituir `.pose/releases` por arquivo ou link perdido,
com policy adotada e sem tags, deixa o subcheck de release com **zero erros**.
GetReleaseStatus, no mesmo estado, devolve erro corretamente.

No consumidor vizinho releaseInputs, ENOENT do mesmo diretório continua sendo
aceito sem ProvenAbsent. Um link perdido responde previous="", err=nil,
anunciando ausência de releases anteriores quando o inventário foi perdido.
O probe exerce o planejamento, sem executar publicação nem escrita por links.

Vizinhos: status/MCP, loaders de fragments/events/policy e ReleaseFilesGaps
já têm guardas de ausência provada para roots absolutos. Check por versão
recusa manifest ausente. A detecção global e a descoberta de previous ainda
usam critérios distintos. O probe do checker é do subcheck responsável;
não se alegou que uma fixture mínima passa todos os demais gates do projeto.

Ação: propagar falhas de ReadDir nos dois consumidores e só usar vazio depois
de provar ausência do mesmo path/ancestrais. Preservar o caso legítimo de
repositório novo ou release vazia sem diretório de fragments.

### R6-F04 — medium — Paths relativos são procurados na raiz errada

Locais: `pose-mcp/internal/pose/path_state.go:17`, `:20`;
`pose-mcp/internal/pose/roots.go:137`.
R3/R7. Probe `TestB38R6RelativeAbsence`.

ProvenAbsent inicia current em `/`, mesmo quando recebe `.pose/releases`.
Depois de Chdir para uma fixture com `.pose -> lost`, ele examina `/.pose`
e responde **true,nil**. Store{Root:"."}.GetReleaseStatus retorna
**Pending:[], Releases:[], nil** em vez de diagnosticar a fonte perdida.

Vizinhos: policy/legacy, changelog policy, fragments, events e release directory
usam esse helper. StoreFor conserva DefaultRoot/Explicit sem absolutizá-los,
portanto a configuração Go/MCP de root relativo pode alcançar o defeito; a CLI
normal resolve projectRoot para absoluto e não reproduz essa variante.
Na fila de obrigações, o status vazio também se propaga como fila vazia.
Não é uma alegação de bypass do executável CLI com seu root normal.

Ação: absolutizar com erro propagado, ou conservar a base do cwd na travessia;
qualificar os consumidores com roots relativos e absolutos, sem quebrar links
válidos nem converter perdas em inventário vazio.

### R6-F05 — medium — Checker revalida categoria histórica sob policy atual

Local: `pose-mcp/internal/cli/release_lifecycle.go:811`.
R6/R7/R8. Probe `TestB38R6HistoricalCategoryPolicy`.

Após taguear um cut correto com fragment `fixed`, mudar legitimamente a policy
atual para aceitar somente `added` faz appendReleasePolicyChecks falhar com
`invalid category: fixed (want added)`. A tag não mudou. Release check responde
gaps=[] e status responde Sound=true usando a policy do cut.

A nova passagem canônica não elimina a passagem anterior que carrega todos os
archives com LoadChangelogPolicy(checker.root). A regressão permanece no checker
global, embora o snapshot e os qualificadores específicos estejam corretos.

Vizinhos: snapshot lê changelogPolicyAt; candidatos/pending usam corretamente
policy atual; checkChangelogs valida categorias apenas dos pending, não repete
a revalidação histórica. Archival witness tem contrato limitado de spec/path/bytes,
sem exigir categorias atuais. Não se propôs ampliar esse witness.

Ação: descobrir identidade/inventário histórico sob o contrato do cut, mantendo
a policy atual para pending/candidatos. Preservar detecção de duplicação entre
locais sem aplicar a categorias antigas uma decisão tomada depois da tag.

### R6-F06 — medium — O gate estrutural obrigatório falha na baseline real

Local: `pose-mcp/internal/cli/release_lifecycle.go:836`; dados de
`.pose/releases/v0.16.4/` e `.pose/releases/v6.3.0/`.
R6/R8 e gate de delivery-evidence. Evidência: execução integral de
`/tmp/b38r6-pose check --strict` e release check estrito para as duas versões.

O checker obrigatório termina FAILURE com 2 erros e 29 warnings. A nova
passagem por checkRelease torna visíveis duas condições reais do clone:

- `v0.16.4`: tag e snapshot íntegros, mas worktree com notas que divergem do
  digest e fragment renomeado de `pose-upgrade-compatibility-lab.md` para
  `pose-scaffold-backup-preservation.md`, sem o manifest da tag acompanhar a
  troca. Release check tagged também recusa o worktree; o diagnóstico é
  consistente com o binding byte a byte de R5, não uma falsa recusa da tag.
- `v6.3.0`: manifest congelado e sem eventos nem tag local; o verificador de
  candidato compara sua versão `v6.3.0` com o engine atual `v7.4.1` e recusa.
  A ausência de tag foi conferida, sem inferir publicação externa, abandono
  autorizado ou corrupção do snapshot de uma tag inexistente.

Esses bytes já estão em origin/main: não são uma corrupção escrita pela PR.
O delta relevante é o novo gate obrigatório verificar também suas cópias de
worktree/candidatos. O resultado verde dos 140 checks registrados e do smoke
de tags não satisfaz o requisito separado de `pose check --strict` verde.
É um bloqueante de validação/integração da entrega, separado dos cinco defeitos
de runtime acima; não se atribuiu a B38 uma implementação de replay B39.

Ação: reconciliar as duas falhas antes do merge, com restauração/disposição
fundamentada dos dados e contrato explícito para o candidato antigo sem tag.
Não reescrever o manifest/tag históricos, inventar eventos, elevar cutoff,
enfraquecer policy nem eliminar a verificação de worktree para obter verde.

## Follow-ups aceitáveis — low

Propostos para backlog, com ownership para eventual closeout. Não foram
escritos na spec nesta revisão bloqueada. Nenhum dos casos isolados abaixo
faz um qualificador aprovar conteúdo materialmente divergente; F01–F04 cobrem
as variantes que efetivamente escondem divergências ou perdas.

- **R6-L01:** recusar `version_evidence.kind:null`, equivalente a kind vazio do engine; source/value continuam vinculados. Probe LowVariants/null_kind. (owner:@core crit:low review:2026-12-01)
- **R6-L02:** harmonizar filtros auxiliares (README/.gitkeep/subdiretórios/outros) entre loader, FilesGaps e snapshot. README é ignorado no candidato e recusado na tag: recusa excessiva, sem aceitar conteúdo falso. Probe LowVariants/README_archive. (owner:@core crit:low review:2026-12-01)
- **R6-L03:** mostrar Snapshot.Limits no status humano e preservar os limits devolvidos pelo candidato no status/check e qualificar o diagnóstico do candidato em publicclaims (seu CandidateState ainda é o estado bruto, sem expor gaps). JSON tagged mantém limites; gaps materiais continuam bloqueantes. Probe LowHumanLimits e inspeção dos call sites. (owner:@core crit:low review:2026-12-01)
- **R6-L04:** ampliar pattern do integration check para TestF*/TestR2*/TestR4*. A suíte Go completa já executa essas regressões. (owner:@core crit:low review:2026-12-01)
- **R6-L05:** reconciliar authoring da spec que ainda afirma que implementação/testes não existem. Manter o item editorial já aberto sem duplicar. (owner:@core crit:low review:2026-12-01)
- **R6-L06:** explicitar symlinks de worktree com bytes idênticos: FilesGaps/archival seguem o link, loader recusa; a tag já exige arquivo regular. Probe LowByteIdenticalLinks conserva o digest e não demonstra aceitação de bytes divergentes. (owner:@core crit:low review:2026-12-01)
- **R6-L07:** tornar smokes históricos explícitos sobre falha de leitura/zero fixtures. O novo smoke de tags continua em qualquer snapErr, sem provar que é legacy; o loader também ignora erro de ReadDir. Nesta revisão foram lidos 75 manifests e qualificado separadamente o inventário real de 81 tags (74 snapshots íntegros e 7 ausentes, sem outros erros); checker de produção permanece gate separado. (owner:@core crit:low review:2026-12-01)
- **R6-L08:** usar loader estrito no plan/prepare. `{broken` e schema 999 ainda permitem prepare --apply com sucesso e criação de candidato que check/status/snapshot recusam imediatamente. O efeito residual é um cut local inutilizável e retrabalho, não falsa aprovação: os qualificadores estritos corrigidos recusam ambos. Probe PrepareChangelogPolicy. (owner:@core crit:low review:2026-12-01)

## Varredura por classe e critérios

| Classe | Consumidores conferidos antes da consolidação | Resultado |
|---|---|---|
| Ordenação por texto | SemVer, previous, releaseInputs, status, cutoff, backfill, publicclaims, fila/open-next | Comparador semântico compartilhado; sorts restantes de paths/specs/categorias são apresentação. Nenhum novo achado de precedência. |
| Ausência versus perda | Git tree/blob e ancestrais, policies primária/legacy/changelog, pending/archive/releases, events, prepare/plan/check/status/MCP e archival | F02/F03/F04; guardas de worktree absoluto melhoraram, mas tree, dois consumidores e roots relativos divergem. |
| Campo sem validação | Manifest raiz/nested/projeções/digests, notes/registry/categorias, versão/source/kind, policies e Sound | F01; L01/L03/L08 são variantes que não aprovam payload divergente isoladamente. Replay de events permanece B39. |
| Comparação de caminhos | Clean/relative, refs/tags, modos finais/ancestrais de Git, source, worktree/tag e archival | F02/F04; L02/L06 são diferenças de filtro/tipo que preservam bytes ou causam recusa. |
| Contrato histórico | Policy da tag, policy atual, renderer/registry, checker, status/check e smoke real | F05; as tags reais que antes falhavam por seção de registry agora passam, mas existe a colisão material F01. |

Forma barata de satisfazer o gate sem seu valor: colocar conteúdo de entrega
na categoria customizada compatibility e atualizar só o digest das notas;
ou obter vazio/fallback ao perder fontes. Os probes F01–F04 demonstram os
caminhos e o que falta para barrá-los. O corpus adversarial tem casos vizinhos
de produtor indisponível e metadata local/publicação, mas não um caso dedicado
à colisão de seção/ancestral/roots relativos acima; isso fica registrado como
lacuna de qualificação, sem dispensar as correções de runtime.

Requirements/correctness/backend-input-errors/integration-impact não podem
passar por F01–F05; o gate obrigatório também falha em F06. R2/R3/R4/R6/R7/R8 têm limitações materiais descritas acima.
Compatibility/backend-contracts exigem corrigir a falsa recusa histórica e a
paridade dos consumidores; o endurecimento já foi classificado para minor nas
revisões anteriores, sem exigir novo schema ou policy. Operability e integridade
também falham quando perda vira vazio. D1 continua adequado causalmente:
recomposição e leitura do tree são o desenho mínimo suficiente, desde que
exceções não apaguem conteúdo e todos os consumidores conservem erros/contrato.
Não há mudança concorrente nova; foram executados os checks race registrados.

Recorrência observada: domínio Go/contrato/validação, causas de recomposição
parcial (F01), perda interpretada como ausência e consumidores paralelos
(F02–F04), contrato atual aplicado ao histórico (F05). Prevenção: helpers
canônicos com matriz de nível final/ancestrais, tipo/erro, root absoluto/relativo,
policy atual/do cut e seções registry/categoria. Recurrence-check: 393 registros,
zero chaves acima do limiar 3 em 14 dias; isso não elimina a recorrência
observada nas revisões. Não foi staged/submetido feedback upstream; Contributor
Mode reserva essa decisão ao usuário. Não há aceitação de risco nem autorização
de merge, portanto não foi criado handoff de risco aceito ou feito closeout.

## Estado deixado no workspace

Spec permanece in-progress, sem completed_at. Os oito lows são propostas de
backlog; nenhum foi disposed como spawned/covered/duplicate. Produção, testes,
policy e pins não foram editados. O descriptor generation.json já estava dirty
no início. Artifact/surface-check refrescaram os índices derivados delivery-integrity
/e generation; ficaram sem commit. Outputs transitórios dos assessments foram
recolhidos na evidência e restaurados ao conteúdo inicial. Os dois JSONs brutos
de validate são evidências adicionais sem commit; o parecer e a evidência
compacta acima concentram a decisão e a reprodução.
