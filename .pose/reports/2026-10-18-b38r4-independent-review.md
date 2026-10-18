# Revisão independente — b38r4 / PR #198

Decisão: **changes requested**. **Oito bloqueantes de merge e cinco follow-ups
aceitáveis, low.** Não selar, atestar, fechar, commitar ou enviar este estado.
A spec permanece `in-progress`; os follow-ups abaixo são propostas para o
próximo closeout, não alterações feitas na spec nesta revisão.

- Repositório: `oseiaspereira88/pose`, PR #198.
- Spec: `pose-release-manifest-tag-integrity`.
- HEAD: `74ca559db942dbaeec218e9af3ecfd915215dbac`.
- `origin/main` e merge-base: `8aa729a76ab1c094690a0892263a86bf3bb7bb42`.
- Revisor: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI.
  Implementador: Claude/Anthropic. Identidade e separação declaradas;
  vendors diferentes; nenhuma alegação de autenticação criptográfica.
- Evidência compacta, saídas e fontes dos probes:
  [b38r4-independent-review.json](../results/b38r4-independent-review.json).
- Policy e issuer pins não foram alterados. Não há selo ou atestação,
  com ou sem envelope: a instrução do solicitante proíbe ambos com bloqueantes.

## Rules applied during review

- Change type: mixed — bugfix Go, contrato de manifest, consumidores, testes,
  matriz de validação, documentação e scaffold.
- Skills: `.agents/skills/pose-review/SKILL.md` e
  `.agents/skills/pose-spec-closeout/SKILL.md`.
- Workflow: `.pose/workflows/review.md`.
- Rules selected:
  - [x] `.pose/rules/security.md`: integridade, JSON, paths e fronteiras de prova.
  - [x] `.pose/rules/backend-go.md`: erros, contratos, build, testes e vet.
  - [x] `.pose/rules/documentation-style.md`: spec, manuais e scaffold.
  - [x] `.pose/rules/delivery-evidence.md`: verificar o runtime revisado, sem
    confundir suíte verde com entrega correta.
  - [x] `.pose/rules/knowledge-governance.md`: consultar decisões e riscos.
  - [x] `.pose/rules/_base-recurrence.md`: classificar recorrências e prevenção.
- Não aplicáveis: frontend, Kubernetes, Terraform, Workers; nenhum delta.

Foram lidos spec/template, workflow, regras, ADRs do ledger imutável e archival,
memória sobre evidência temporal, revisões anteriores, diff contra `origin/main`,
implementação, testes e consumidores vizinhos. Os índices e resultados gerados
foram inspecionados como evidência derivada, separada do código e das claims.
`pose context` resolveu a autoridade local sem ambiguidade. Não houve delegação.

## Plano e checks

Preparação **não selada**: `rvb-4ebc3bd1079f5ff3`.
Bundle digest: `sha256:4ebc3bd1079f5ff35a34827fa355b9fd3c0f04f1f61526e7727ca02baf1b0924`.
Plan digest: `sha256:882f78cf69f15e14a89033e57401485d12ca4978d97dfbb2bfb5a3e2655c7982`.
Patch digest: `sha256:90758460aaebaa72cca285eabe52ad574b1e8c586b35258f46ca05526618e77f`.
Tree digest: `sha256:a74ce841b595c869234da989c02003bda388b4b37b7d3ed6985bcadf7771a880`.
Perfis: spec-closeout, backend-review, engineering-judgment e
high-criticality-review; 15 critérios, componente `pose-mcp`, band critical,
independência different-actor. Owner do componente continua ausente.

**Distinção necessária de engine:** o `pose` instalado foi compilado de
`8aa729a7`, antes desta implementação. Seu validate global passou 140/140 etapas
e seu check estrito passou com 29 warnings. Isso não qualifica o novo runtime.
Foi compilado `/tmp/b38r4-pose` de cópia dos fontes de produção deste HEAD, sem
alterá-los; os probes só adicionam arquivos `_test.go` nessa cópia temporária.

- Binário deste HEAD: `validate --strict --module pose-mcp` passou **137/137**,
  incluindo build, `go test ./...`, vet e os checks race registrados.
- Binário deste HEAD: **`check --strict` falhou com 131 erros**: 42 de release
  e 89 de delivery/archival. Todos se relacionam à regressão F01.
- `release status --json` deste HEAD falhou ao ler `v0.16.0`, por F01.
- Artifact-check estrito da spec: sem erro atribuído à spec; 33 claims;
  640 warnings globais de orphan, separados dos achados desta revisão.
- Surface-check estrito, usando explicitamente os resultados novos desta
  revisão: passou, sem findings. A tentativa com o arquivo default antigo
  diagnosticou evidência stale; não foi usada para afirmar entrega.
- Lint estrito da spec passou; warning de seção Tasks opcional ausente.
- Govulncheck do módulo: nenhuma vulnerabilidade. Gitleaks sobre os 11 commits
  de `origin/main..HEAD`: nenhum segredo. `git diff --check`: passou.
- Recomendadas executadas: suggest, assess-design, discover `--if-stale`,
  integrate e tech-debt. Não se inferiu cobertura completa de contratos a partir
  de heurística: integrate mantém integrações não observadas; tech-debt divulga
  coverage incompleta por follow-up inválido preexistente em outra spec.
- Recurrence-check tolerante de 14 dias e followups `--all` executados.
- Completion-only — selo, auto-attest, attest, verify, review-check,
  closeout-check e close — **adiados por bloqueantes**, conforme autorização.
  Discover com `--update-state` de entrega também fica para o closeout válido.

## Bloqueantes de merge

Os probes são diagnósticos: `PASS` significa que a observação foi reproduzida,
não que o comportamento esteja correto. Fontes e saídas estão na evidência JSON.

### R4-F01 — high — O leitor recusa corpos multiline válidos e quebra a história

Local: `pose-mcp/internal/pose/release_manifest_integrity.go:55`.
Requisitos: R1/R6/R8. Probes: `TestB38R4MultilineFragmentIsRefused`,
`TestB38R4HistoricalManifests`; check/status reais deste HEAD.

`rejectDuplicateJSONKeysAndControls` recusa também caracteres de controle
**decodificados** dentro de strings, incluindo `\n` escapado permitido em JSON.
Um fragment válido com duas linhas é aceito por LoadReleaseFragments e congelado
por NewReleaseManifest, mas o próprio loader recusa esse manifest. O inventário
real tem 42 manifests recusados e 33 aceitos; todos os 42 têm esse diagnóstico.
O check estrito deste HEAD produz 131 erros e o status global deixa de responder.

Vizinhos: LoadReleaseManifest, ReadReleaseSnapshot, status/MCP, index,
LoadArchivedFragments, artifact/delivery graph e review subject herdam a recusa.
Os 89 erros de archival não são 89 defeitos independentes. O helper continua
apropriado a contratos de review com strings restritas; isso não autoriza aplicá-lo
indiscriminadamente ao texto livre de changelog.

Ação: preservar detecção de duplicatas, aliases e documento único, mas aplicar
restrições de caracteres por campo; aceitar conteúdo multiline legítimo.
Cobrir produtor→loader e história real, sem editar manifests/tags antigos.

### R4-F02 — high — Blob de symlink é reinterpretado como arquivo de versão

Local: `pose-mcp/internal/pose/release_manifest_integrity.go:549` e `:569`.
Requisitos: R3/R4/R8. Probe: `TestB38R4VersionSourceSymlink`.

Com VERSION sendo symlink para `1.0.0`, cujo arquivo contém `9.9.9`, o leitor
normal retorna `v9.9.9`. Na tag, cat-file entrega os bytes do link (`1.0.0`);
o verificador grava esses bytes como arquivo regular temporário e conclui
`Intact=true`, `Sound=true`, sem gaps ou limits, para release `v1.0.0`.

Vizinhos: ReadReleaseVersionSource confina/resolve symlinks e verifica arquivo
regular; o snapshot perde o mode do tree. Notas/fragments/policies também são
lidos como blobs sem comprovação de mode. LoadReleaseFragments recusa symlink,
enquanto TaggedReleaseWorktreeGaps usa ReadFile. Status, checker e record recebem
a falsa prova da fonte; o caso externo não precisa ser usado para reproduzir.

Ação: ler mode/tipo e respeitar o contrato da fonte no tree da tag; recusar
symlinks não suportados ou resolvê-los dentro desse tree com confinamento.
Não transformar o conteúdo de um link em conteúdo de arquivo regular.

### R4-F03 — medium — Notas preparadas não são recompostas; leitor ignora gaps

Locais: CLI `release_lifecycle.go:430` e `:578`.
Requisitos: R2/R3/R7. Probe: `TestB38R4PreparedNotesProjection`.

Trocar notas por `# Unrelated notes` e ajustar somente NotesDigest deixa
`prepared_gaps=[]`. O rebuild compara ReleaseInputDigest, que não inclui as
notas, e não compara o NotesDigest/rendering recomposto. Depois de taggear esse
mesmo cut, o snapshot detecta a projeção errada, mas `release notes` ainda retorna
0 e imprime as notas recusadas: descarta os gaps retornados por checkRelease.

Vizinhos: o snapshot agora recompõe corretamente o rendering; prepare e
NewReleaseManifest também o produzem. O checker preparado e o leitor/alias de
notas continuam com as duas variantes da lacuna anterior R2-F07.

Ação: compartilhar a recomposição canônica sob os inputs do cut e fazer o leitor
respeitar o resultado de integridade, com limites explícitos quando aplicáveis.

### R4-F04 — medium — Status preparado continua sólido após perder arquivos

Local: `pose-mcp/internal/pose/release_lifecycle.go:384`.
Requisitos: R3/R7/R8. Probes: `TestB38R4PreparedStatus`,
`TestB38R4PreparedLostArchiveNotes`.

Sem tag, remover notas, fragment ou VERSION mantém `state=prepared`,
`Sound=true`, `gaps=[]`. Para notas/fragment ausentes, checkRelease do mesmo cut
retorna os gaps correspondentes. A composição de integridade no status só ocorre
no ramo com tag; ProjectRelease sozinho verifica eventos, não o cut preparado.

Vizinhos: MCP e índice delegam ao status; publicclaims divulga estado candidato
a partir dele. O ramo tagged passou a compartilhar TaggedReleaseWorktreeGaps e
detecta a perda corretamente. A correção de R2-F03 ficou restrita àquele ramo.

Ação: verificar também o cut preparado no domínio compartilhado, preservando a
distinção entre candidato legítimo sem tag e candidato cujo snapshot está perdido.

### R4-F05 — high — Consumidores ignoram a falta de solidez do estado declarado

Locais: `pose-mcp/internal/pose/obligation_sources.go:135`;
CLI `release_lifecycle.go:368` e `:523`.
Requisitos: R4/R7. Probes: `TestB38R4UnsoundVerifiedObligation`,
`TestB38R4LostTagVsRecordedState`.

Release com eventos até verified, mas notas apagadas: status informa
`verified`, `Sound=false`, gap; releaseQueueObligations retorna `[]` porque o
segundo loop pula verified/yanked incondicionalmente. O primeiro loop ganhou
Sound, mas a decisão de omitir a própria release não ganhou a guarda.

Outra variante: após `release record ... tagged` legítimo, apagar a tag deixa
status `tagged,Sound=false`; o check estrito retorna 0,
`valid=true,tagged=false,gaps=[]`. Trata a falta da tag reclamada como candidato
inicial sem tag, sem reconciliar a projeção dos eventos com o snapshot exigido.

Vizinhos: publicclaims e open-next exigem Sound para promover publicação/avanço;
o consumidor de obrigações ainda perde a dívida. A correção não requer implementar
o replay temporal de B39: basta não transformar uma contradição conhecida em
validade/ausência limpa. `yanked` legítimo deve manter sua semântica de revogação.

Ação: propagar integridade/coverage nos consumidores e reconciliar tag requerida
pelo estado; não omitir release não sólida como se já estivesse resolvida.

### R4-F06 — high — Archival aceita outro cut sob uma tag existente

Local: `pose-mcp/internal/pose/release_archive.go:62` e `:87`.
Requisitos: R3/R5/R7. Probe: `TestB38R4ArchiveDifferentCut`.

Tag congela alpha. Em commit posterior, substituir fragment, identidade e manifest
por beta, recompondo hashes/projeções, deixa o testemunho beta `Intact=true` e
ledger.resolve aceita sua claim. Status do mesmo repositório informa
`Sound=false`: o manifest corrente não é o da tag. O probe commita os arquivos;
os dois paths exigidos estão rastreados, não se trata de arquivo solto.

Vizinhos: artifact-check, surface-check, delivery graph e resolução de fragment
do review subject usam LoadArchivedFragments; não consultam o binding que status,
check e record já verificam. Conferir a spec escrita no blob corrigiu R2-F02,
mas não o vínculo do testemunho com o cut já taggeado.

Ação: preservar archival preparado legítimo e, quando houver tag, vincular o
testemunho ao cut correspondente. Um manifest posterior não deve fabricar outro
archival para uma release cujo conteúdo selado é diferente.

### R4-F07 — high — Perda de fonte ainda vira ausência/default/fallback

Locais: `release_manifest_integrity.go:319`, `:334`, `:355`, `:515`;
`release_lifecycle.go:111` e `:365`.
Requisitos: R3/R4/R7. Probes: `TestB38R4RealLostGitObjects`,
`TestB38R4GitObjectLossIsAbsence`, `TestB38R4DanglingReleaseRoot`,
`TestB38R4PolicyFallbackOnLoss`.

Reprodução com Git real: apagar o objeto tree de `.pose/policy` de uma fixture
taggeada, mantendo uma policy legacy equivalente, resulta em
`Intact=true,gaps=[]`. O stderr de resolução de path vira os.ErrNotExist; o
leitor troca a policy perdida pelo fallback e o changelog perdido por defaults.
Apagar apenas o blob de changelog, em contraste, é corretamente diagnosticado.
O shim adicional confirma a classe para o erro textual `Not a valid object name`.

Vizinhos: gitTreeNames repete a classificação por texto; ausência legítima de
diretório Git vazio deve continuar aceita, mas não uma árvore referenciada que
não pôde ser lida. No worktree, `.pose/releases -> missing` sem tags transforma
um inventário perdido em `Releases=[],nil`; entradas de versão symlink são
puladas por IsDir. LoadReleaseFragments tem o mesmo ENOENT para raiz dangling.
LoadReleasePolicy cai no legacy após qualquer erro do primário: diretório com
child no lugar do JSON passa no check preparado por fallback, enquanto o snapshot
diagnostica esse mesmo tipo errado. Archival não dispõe de canal de coverage.

Ação: demonstrar ausência consultando árvore/entrada autoritativa; preservar
falhas de objeto, listagem, symlink e tipo como perda. Usar fallback/default só
após ausência legítima comprovada, sem normalizar fonte ilegível.

### R4-F08 — high — Policy sem validação de cutoff pode apagar tags do inventário

Locais: `release_lifecycle.go:136` e `:424`;
`release_manifest_integrity.go:519`; CLI `release_lifecycle.go:876`.
Requisitos: R3/R6/R7. Probes: `TestB38R4InvalidCutoffHidesTags`,
`TestB38R4InvalidCutoffSkipsChecker`, `TestB38R4ChangelogFuturePolicy`.

`legacy_cutoff:"not-a-version"` passa ValidateReleasePolicy. A nova comparação
total ordena texto inválido depois de toda versão válida; o cutoff então exclui
todas as tags. Uma tag governada sem snapshot desaparece do status e o subcheck
de release tem zero erros. Remover o cutoff na mesma fixture restaura a release
com gap e o erro do checker. Não é necessário mudar policy deste clone para
reproduzir: foram usados somente dados sintéticos em diretório temporário.

Variante do contrato histórico: changelog policy com `schema_version:999` é lida
como schema 1, sem limit, e produz `Intact=true`. O novo leitor não verifica o
schema que acaba de decodificar. Isso não demonstra suporte a contrato futuro.

Vizinhos: checker global, inventário do status e consumidores da fila usam o
cutoff; ValidateReleaseVersion já protege manifest.version e previous_version,
mas não o cutoff. LoadChangelogPolicy também ignora schema: o leitor histórico
deve declarar unsupported, em vez de herdar uma falsa comprovação completa.

Ação: validar cutoff antes de compará-lo e validar/sinalizar explicitamente o
contrato histórico da policy de changelog. Não aplicar uma policy futura
retrospectivamente nem permitir que valor inválido suprima fontes governadas.

## Follow-ups aceitáveis — low

Nenhum item abaixo permite aprovar um cut materialmente incorreto. Devem ficar
`[open]` na spec quando o caminho de closeout voltar a estar autorizado.

- R4-L01: `version_evidence.kind:null` decodifica como vazio e passa no caso do
  engine. Probe `TestB38R4NullEvidenceKind`. O source exato do engine e o value
  continuam exigidos; essa variante não contorna a prova de fonte. Recusar null
  preserva a promessa de shape estrito, sem exigir mudança no fallback legítimo.
  (owner:@core crit:low review:2026-12-01)
- R4-L02: filtros de archive divergem para README/.gitkeep e outros auxiliares.
  Probe `TestB38R4ReadmeInArchive`: loader ignora README, snapshot recusa como
  fragment não congelado; TaggedReleaseWorktreeGaps também filtra diferente.
  É recusa excessiva, sem falsa aprovação; harmonizar/documentar o conjunto.
  (owner:@core crit:low review:2026-12-01)
- R4-L03: status humano não mostra Snapshot.Limits; probe
  `TestB38R4HumanStatusLimits`. O check humano agora mostra o limite, e JSON
  mantém a distinção; melhorar a divulgação no status e distinguir estado
  declarado do candidato não sólido no diagnóstico de publicclaims. O gate de
  publicação de publicclaims já exige Sound. (owner:@core crit:low review:2026-12-01)
- R4-L04: o pattern do check integration novo cobre o lote inicial, mas não
  seleciona `TestF*`/`TestR2*`. Os testes são executados por `go test ./...` e
  portanto não estão ausentes da validação; ampliar a seleção melhora a
  rastreabilidade do check citado pela trace. (owner:@core crit:low review:2026-12-01)
- R4-L05: reconciliar o texto de authoring que ainda diz que testes/código não
  existem. É o follow-up editorial já aberto; não duplicá-lo na spec nem mudar
  sua disposição automaticamente. (owner:@core crit:low review:2026-12-01)

## Varredura das classes, critérios e limites

| Classe | Consumidores conferidos | Resultado |
|---|---|---|
| Ordenação por texto | precedência, previous, status, releaseInputs, open-next, cutoff, backfill, publicclaims, obrigações | Sem novo erro para versões válidas; cutoff inválido em F08. Sorts de paths/specs não confundidos com SemVer. |
| Ausência versus perda | blobs/trees, resolver, policies, pending/archive/releases, prepare, status, checker, notes, archival | F04/F05/F07; ausência legítima de diretório vazio preservada. |
| Campo sem validação | raiz/nested JSON, version_evidence, fragments, projeções, notas, policies, estados/eventos e testemunho | F01/F02/F03/F05/F08; null kind equivalente em L01. |
| Comparação de caminhos | paths limpos, fonte declarada/real, modes/symlinks, manifest worktree/tag, namespace refs/tags, archival e review subject | F02/F06/F07; alias ./VERSION normalizado pelo produtor e path não limpo recusado pelo loader nos controles. |

Atalho adversarial mais barato: fornecer campos/digests coerentes a si mesmos,
usar um consumidor que não confere a mesma fonte, ou converter fonte perdida em
ausência. F02/F03/F06/F07/F08 demonstram aceitação indevida. O corpus
`pose-mcp/internal/cli/testdata/adversarial/README.md` tem os casos adjacentes
`local-metadata-read-as-published` e `unavailable-producer-reads-as-empty`, mas
não delega casos B38 para essas variantes. Acrescentar controles na composição
real ao corrigir; apenas testes de objeto autocoerente não bastam.

Critérios: requirements/correctness/backend-input-errors e integration-impact
falham semanticamente pelos bloqueantes; compatibility/backend-contracts falham
também pela regressão histórica F01; security falha em fonte/path e perda de prova;
operability/observability falham em validade/coverage divergentes. Scope e
proporcionalidade são adequados, mas não compensam as lacunas. O desenho de ler
o tree da tag é causalmente apropriado; suas implementações paralelas divergem.
Não há nova concorrência de produção; os checks race registrados passaram.
Documentação descreve intenção maior que o comportamento verificado.

Compatibilidade: stricter/minor está declarado no fragment, e Snapshot é aditivo;
recusar corpos multiline que o produtor suporta não é migração deliberada nem
limite legacy documentado. Preservar contratos de federação/composição existentes.
Replay temporal de eventos, seleção/exclusividade de escopo e recuperação de
tentativas continuam B39/B40/B41; não foram declarados entregues por B38.

Recorrência: backend/contratos, causas implementação e validação parcial por
consumidor; F03–F07 são variantes das classes anteriores. O recurrence-check
formal não sinalizou clusters, sem invalidar a recorrência observada manualmente.
Prevenção: compartilhar resultado de integridade/coverage, testar produtor→leitor,
history e cada consumidor antes de declarar a classe encerrada. Nenhum feedback
upstream foi staged/submetido sem consentimento.

Não há risco residual aceito para merge: não se criou handoff de entrega aceita,
não se dispôs follow-up como spawned/covered/duplicate e não se alterou a spec.
Ferramentas assess/artifact-check regeneraram artefatos de assessment/state/índice;
estes efeitos e a alteração prévia em generation.json estão separados da evidência
de código. Nenhum arquivo de implementação, policy ou tag foi alterado neste clone.
