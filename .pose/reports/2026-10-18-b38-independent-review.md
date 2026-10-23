# Revisão independente — b38 / PR #198

Decisão: **changes requested**. Dez achados bloqueiam o merge; um achado
editorial é follow-up aceitável. Não selar, atestar, fechar, commitar ou publicar
este estado. A spec permanece `in-progress`.

- Repositório: `oseiaspereira88/pose`.
- Spec: `pose-release-manifest-tag-integrity`.
- HEAD: `b58e1d8b836c6b53f3f633103093b8433a4a078b`.
- Base `origin/main`: `8aa729a76ab1c094690a0892263a86bf3bb7bb42`.
- Branch: `impl/pose-b38-release-manifest-integrity`.
- Revisor: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI; execução
  independente e vendor diferente do implementador Claude/Anthropic
  (`d2c6a2ac`, coautoria Claude Sonnet 5.5).
- Data: 2026-10-18. Evidência: [b38-independent-review.json](../results/b38-independent-review.json).

## Rules applied during review

- Change type: mixed — bugfix Go, contrato CLI/MCP/JSON e documentação.
- Workflow consulted: `.pose/workflows/review.md`.
- Skills: `.agents/skills/pose-review/SKILL.md` e
  `.agents/skills/pose-spec-closeout/SKILL.md`.
- `.pose/rules/backend-go.md`: domínio, consumidores, erros e subprocessos Git.
- `.pose/rules/security.md`: entrada JSON, paths, autoridade e scans.
- `.pose/rules/documentation-style.md`: spec, manuais, scaffold e parecer.
- `.pose/rules/delivery-evidence.md`: requisitos, atribuição e evidência atual.
- `.pose/rules/knowledge-governance.md`: consulta a decisões anteriores; nenhum
  risco foi aceito nem handoff de risco residual foi criado.
- `.pose/rules/_base-recurrence.md`: causa e prevenção das classes recorrentes.
- Frontend, Kubernetes e outros stacks: não aplicáveis ao delta.

Consultados ADRs de ledger imutável (2026-08-03), fonte autoritativa
(2026-10-02), archival pelo ledger (2026-09-11) e a nota sobre B39/B40/B41.
Replay temporal e autenticação de provider continuam no escopo próprio de B39;
este parecer não exige assinatura das atestações nem pin de issuer.

## Sujeito, plano e execução

Lidos a spec, todos os arquivos do diff `origin/main...HEAD`, testes, fixtures,
script de mutações e consumidores vizinhos. O diff tem 25 paths. Os cinco
commits atribuídos carregam `POSE-Spec: pose-release-manifest-tag-integrity`;
o merge não introduz implementação própria sem trailer. Atribuição foi
conferida com `artifact-check`, sem depender do parser Git de trailers.

Preparação read-only pelo binário compilado deste HEAD:

- Bundle **somente prepared**, nunca selado: `rvb-be678f7a075ba346`.
- Bundle digest: `sha256:be678f7a075ba3461382ae8e70ac85d1471739ed0fa314ed39970c1e284c7372`.
- Plan digest no payload do bundle: `sha256:70797e7086d99facb58f1b42d6ab439ed54d92c052b8716154a8fa2b1cdac7fa`.
- Patch digest: `sha256:a797c49496f1e56e292156599713a438d6a85324c6ca7108d4a4fc2963225071`.
- Tree digest: `sha256:48744dc122150987e3f7fb14aad230acdb340ad3c29903e9797cae9bd7ebfa12`.
- Componente: `pose-mcp`, Go, criticalidade high; falta owner nos metadados.
  Os paths de governança/documentação não mapeados foram revisados diretamente.
- Perfis: spec-closeout, backend-review, engineering-judgment e
  high-criticality-review. Quinze critérios; independência `different-actor`.
- Ferramentas ativas obrigatórias: validate global (inclui pose-mcp),
  artifact-check e surface-check executados. Resultados finais na evidência.
- Recomendadas executadas: suggest-review, assess-design, assess-discover
  `--if-stale --component pose-mcp`, assess-integrate e assess-tech-debt.
- Completion tools (`review-check`, `closeout-check`, verify, attest e close):
  adiados por bloqueantes, conforme instrução expressa do solicitante.

Os probes usam overlay Go em arquivos temporários e fixtures Git sintéticas.
Seus `PASS` significam que a execução diagnóstica terminou: os valores
`intact=true` / `valid=true` abaixo são a reprodução dos defeitos, não prova de
correção. Os fontes dos probes estão embutidos na evidência para reprodução.

## Achados — todos bloqueantes de merge

### F01 — high — O decoder não implementa o contrato JSON estrito (R1/R8)

Local: `pose-mcp/internal/pose/release_manifest_integrity.go:42`.
`decoder.More()` não verifica EOF: manifest válido seguido de `}` ou `]` retorna
zero issues. Chaves repetidas (`schema_version:2` seguido de `schema_version:1`)
e `VERSION` em vez de `version` também passam. `breaking` ausente ou null vira
false e é aceito. Isso permite entrada inválida/ambígua sob a alegação de schema
estrito. A verificação de tipos por struct não comprova presença dos campos.

Vizinhos: `LoadReleaseManifest` herda todas essas aceitações; snapshot, status,
índice e archival consomem esse loader. O decoder de `release record` também
faz apenas um Decode, sem EOF e sem detecção de duplicatas; o loader de events
é outro contrato, cujo replay temporal fica em B39. Controlados `specs` e
`categories` null com fragment não vazio: esses dois casos são recusados.

Ação: validar objeto, presença/tipo conforme schema, duplicatas e nomes exatos;
exigir EOF após o único documento. Cobrir os consumidores, não só o helper.

### F02 — high — Projeções são conferidas contra dados autodeclarados (R2/R3)

Locais: `release_manifest_integrity.go:89`, `:125` e `:276`.
O snapshot confere o hash dos blobs, mas não extrai deles os campos semânticos.
Alterar Category/Breaking/Body/Refs no manifest e recompor os campos redundantes
e o input digest deixa `Intact()==true`, mantendo os fragments e notas originais
intocados e seu notes digest correto. Dois specs distintos apontando para o
mesmo `alpha.md`, ou Body vazio, também passam. O hash pode permanecer igual
ao arquivo real enquanto o manifest descreve outra entrega.

Vizinhos: o `checkRelease` do worktree reconstrói a partir de
`LoadReleaseFragments` e recusa um path renomeado só no manifest (controle
`path_alias`). O snapshot e o checker global não fazem essa recomposição.
`LoadArchivedFragments` usa Spec/Path/Digest do manifest e pode atribuir o mesmo
arquivo a outro spec. O parser normal de fragments exige categoria válida,
body e spec; o novo verificador da tag não aplica esse contrato.

Ação: recompor os campos a partir dos blobs autoritativos do cut, incluindo
identidade, categoria, breaking, refs, body e path único; validar notas sob o
contrato histórico aplicável. Não usar a recomposição do hash de um objeto
adulterável como substituto da fonte.

### F03 — high — Policy digest não é vinculado à policy do cut (R3/R6)

Locais: `release_manifest_integrity.go:71`, `:125` e `:247`.
O snapshot nunca lê a policy da árvore da tag. Com policy schema 1 presente,
substituir `policy_digest` por outro SHA-256 e recalcular o input digest ainda
retorna intact, sem gap/limit. Ausência de policy também não é divulgada pelo
snapshot. Formato de digest e autoconsistência não provam policy autoritativa.

Vizinhos: `checkRelease` compara a policy atual (F08), enquanto o checker
global passa a confiar em `ReadReleaseSnapshot`; status e MCP expõem o resultado
fraco. Archival tampouco estabelece esse vínculo.

Ação: obter e validar a policy autoritativa na árvore/contrato do cut, comparar
o digest e declarar perda/unsupported explicitamente quando não verificável.
Não adotar policy nova nem reinterpretar policy histórica pela atual.

### F04 — high — Evidência de versão pode contradizer ou perder sua fonte (R1/R3)

Locais: `release_manifest_integrity.go:76` e `:310`;
`pose-mcp/internal/cli/release_lifecycle.go:360`.
Manifest/tag `v1.0.0` com value e arquivo VERSION ambos `v9.9.9` é intact:
compara-se arquivo com value, nunca com a versão da release. `{}` remove a
evidência e vira mero limit; kind inventado também. Fonte JSON inexistente vira
limit antes mesmo de tentar ler o blob. No cut ainda sem tag, alterar VERSION
para 9.9.9 mantém `release check --strict` com `valid:true`.

Vizinhos: `releaseInputs` compara a fonte com target na preparação, mas
check/status/checker não preservam essa garantia. O leitor público de fontes
recusa kinds desconhecidos e valida o arquivo; o snapshot evita esse leitor
quando kind não é text. O schema atual não retém a key JSON: isso justifica um
limite de recomposição, não tratar arquivo comprovadamente ausente como intact
sem perda nem aceitar kind arbitrário.

Ação: validar o contrato version_evidence e o vínculo value/version/policy;
verificar existência/leitura das fontes mesmo quando a recomposição é limitada;
distinguir legacy reconhecido, unsupported e evidência perdida. Conferir também
a fonte do candidato ainda sem tag.

### F05 — high — Status e consumidores ignoram integridade negativa (R4/R7)

Local: `pose-mcp/internal/pose/release_lifecycle.go:347`.
ReadReleaseSnapshot retorna nil error para manifest com Issues ou snapshot com
Gaps. GetReleaseStatus não consulta Intact nem propaga esses campos; notas
corrompidas na tag resultam em `state:tagged`, `gaps:[]`. O CLI textual imprime
somente `v1.0.0 tagged`, ocultando também limits. Eventos já projetados como
published/verified não são rebaixados nem impedidos por esse resultado.

Vizinhos: MCP delega diretamente a GetReleaseStatus; index publica a projeção;
publicclaims exige apenas os Gaps externos vazios; releaseQueueObligations
ignora inclusive Gaps externos ao escolher verified/progressed;
open-next verifica somente o state. `release record` também não chama o
verificador da tag: o probe registra `tagged` com exit 0 para tag sem manifest
no tree, que `release check` recusa imediatamente.

Ação: compartilhar o resultado de integridade/coverage entre leitura e
transições, propagar diagnóstico e impedir que consumidores usem estado com
fonte inválida/incompleta como evidência positiva. Preservar os fatos históricos
sem inventar validação; não basta mover os gaps para um campo ignorado.

### F06 — high — Perdas de leitura e inventário viram ausência limpa (R3/R7)

Locais: `release_manifest_integrity.go:290`, `release_lifecycle.go:333`;
CLI `release_lifecycle.go:398`, `:419`, `:768` e `:787`.
Um cut vazio cuja entrada de archive é um blob em vez de árvore fica intact:
o erro de ls-tree é descartado. `.pose/releases` como arquivo retorna status
vazio com nil error. Remover o diretório corrente também apaga a release da
projeção apesar de a tag ainda conter o snapshot. Pending malformado é ignorado
por check: `valid:true`. Erros de resolução Git são tratados como tag ausente.

Vizinhos: releaseInputs, LoadArchivedFragments e checker global descartam
erros de inventário; o global também ignora erro de `git tag --list`.
LoadReleaseFragments diferencia vários erros e GetReleaseStatus propaga erro do
pending, mas check não. Ausência legítima de archive vazio, diretório inexistente
em instalação nova e ENOTDIR/I/O/perda de fonte conhecida precisam de resultados
distintos; não exigir diretório Git vazio, que não é versionável.

Ação: manter ausência, leitura falha e perda de fonte conhecida separadas até os
consumidores; enumerar/considerar tags quando há snapshot histórico; devolver
coverage/unknown ou recusa em vez de clear/intact/valid. Cobrir todas as chamadas
vizinhas da classe e não somente a linha nova de ls-tree.

### F07 — high — CLI ainda aceita branch homônimo como tag (R4/R7)

Local: `pose-mcp/internal/cli/release_lifecycle.go:478`.
O helper antigo usa `git rev-list ... target^{}` sem `refs/tags/`. Criar somente
o branch `v1.0.0` faz `release check --strict --json` retornar exit 0,
`tagged:true`, `valid:true`. O helper novo corretamente não encontra a tag, e
releaseTagSnapshotGaps transforma isso em nada a verificar.

Vizinhos: cmdReleaseRecord usa o helper antigo para aceitar evidence.Commit;
GetReleaseStatus e checker usam o novo. Resultado: identidades Git diferentes
para a mesma release. É lacuna preexistente no consumidor, não introdução do
helper novo, mas impede o contrato compartilhado prometido nesta entrega.

Ação: usar o resolver explícito de refs/tags também em check/record e
distinguir tag ausente, objeto não commit e falha de leitura.

### F08 — medium — Policy futura invalida cut histórico correto (R6)

Local: `pose-mcp/internal/cli/release_lifecycle.go:364` e `:392`.
Após um cut correto e tag, mudar apenas AllowEmpty na policy corrente gera
`policy digest differs from prepared manifest` e `release input digest mismatch`.
O snapshot da tag continua intact. O teste de next-cycle exercita o helper do
domínio, não esse caminho completo do CLI; o teste CLI usa policy ausente e
descarta os gaps do worktree, logo não qualifica R6 nessa fronteira.

Vizinhos: checkRelease também passa a policy de changelog corrente ao parser de
archive; status não a compara com o cut, e checker usa snapshot. São critérios
temporais divergentes. Código de comparação preexistente, lacuna mantida na
integração desta correção.

Ação: verificar releases tagged sob os inputs históricos; aplicar policy atual
ao candidato pertinente sem mudar retrospectivamente o significado de um cut.

### F09 — high — Cut corrente e cut da tag não são vinculados (R5/R7)

Local: `pose-mcp/internal/cli/release_lifecycle.go:411`.
O CLI valida A na tag e B no worktree separadamente. Preparar A, taguear, depois
substituir fragment, notes e manifest por B autoconsistente sob `v1.0.0` resulta
em `valid:true`, `tagged:true`, zero gaps. Isso prova apenas duas release
structures válidas com nome igual, não o vínculo do snapshot que se reporta.

Vizinhos: GetReleaseStatus põe manifest corrente e snapshot histórico lado a
lado sem compará-los; record usa o manifest corrente e só confere o commit da
tag. Os consumidores podem associar metadados B à publicação A.

Ação: escolher a autoridade explícita e vincular o conteúdo selado entre as
representações, recusando divergência. Não exigir igualdade HEAD/commit da tag:
provider evidence e ciclo seguinte continuam podendo acrescentar commits.

### F10 — medium — Versão/anterior e ordenação semântica estão incompletas (R1/R5/R7)

Locais: `release_manifest_integrity.go:61`,
`pose-mcp/internal/pose/release_lifecycle.go:17`, `:366`;
CLI `release_lifecycle.go:167` e `:697`.
`previous_version:v9.9.9` em manifest v1.0.0 é intact após recompor o input.
ValidateReleaseVersion aceita `v1.0.0-..`. Os comparadores de domínio e CLI usam
texto após major/minor/patch: rc.10 vem antes de rc.2 e final vem antes de rc.1.
Os comparadores/regex são preexistentes; o novo validador os reutiliza e valida
previous somente por sintaxe e desigualdade, perpetuando a aceitação.

Vizinhos: ordenação alimenta status, publicclaims, supressão de obrigações por
newestVerified/newestProgressed, releaseInputs, open-next e legacy cutoff do
checker. `backfill` faz sort.Strings de versões (ao menos a apresentação é
lexical). sort de filenames em gitTreeNames e de specs em fragments não implica
precedência SemVer e não é defeito dessa classe. A versão de fonte text/json
usa contrato estável X.Y.Z e já recusa prerelease: registrar claramente a
compatibilidade desse contrato com as versões históricas aceitas pelo ledger.

Ação: definir e aplicar a mesma precedência de versão em todos os consumidores,
validar a relação previous/current e recusar formas fora do contrato. Se um
tipo de versão não é suportado, expor esse limite sem usá-lo para escolher uma
release verificada que dispense obrigações de outra.

## Varredura das classes nos consumidores vizinhos

A reexecução conferiu as quatro classes antes de consolidar os achados:

- Ordenação por texto: `compareReleaseVersions` e `versionLess` chegam a status,
  publicclaims, obrigações, seleção do predecessor, open-next e cutoff. O sort
  de paths/specs e a apresentação do backfill foram distinguidos da precedência.
- Ausência versus perda: `gitBlobAt`, `gitTreeNames`, resolução/listagem de tags,
  inventários de releases/archive/pending, loaders de policy e eventos foram
  conferidos até checker, CLI, MCP e obrigações (F03–F08).
- Campo sem validação: decoder, loader, projeções, fragments, version evidence,
  policy digest, previous/current e identidade do cut foram seguidos pelos mesmos
  consumidores, incluindo `release record` e archival (F01–F05/F09/F10).
- Comparação de caminhos: o validador novo, `LoadReleaseFragments`,
  `LoadArchivedFragments`, `ReleaseVersionSource.Validate` e
  `ConfinedProjectPath` foram confrontados. O leitor público confina paths e
  symlinks; isso não substitui recompor a identidade dos blobs na tag nem
  detectar dois fragments com o mesmo path. O controle `path_alias` é recusado
  pelo CLI atual; o snapshot aceita o path duplicado (F02). Não foi alegado
  escape de filesystem sem reprodução.

## Cobertura, critérios e prevenção

O ataque mais barato ao gate é editar os dados repetidos do manifest e
recalcular o hash público, sem alterar os inputs autoritativos. F02/F03/F04 e
F09 mostram que isso ainda satisfaz a forma do gate. O corpus adversarial
canônico não tem P04/P06 registrado; os testes específicos de B38 cobrem
adulteração simples, mas não esses casos de consistência falsificada nem os
consumidores negativos. Os 23/23 mutantes retidos são evidência histórica do
implementador, não substituem os probes desta revisão nem foram reapresentados
como nova execução independente.

Critérios correctness, requirements, backend-contracts, backend-input-errors,
backend-integration-impact, backend-observability, compatibility, operability,
documentation, scope, security, assumption-integrity e design-causality não
podem ser aprovados diante dos casos acima. Não há nova concorrência no delta;
os checks race registrados continuam aplicáveis pela matriz. A solução é pequena
e reutiliza o domínio, mas sua suficiência ainda depende de fechar as fronteiras
de autoridade. Nenhuma conclusão foi gravada como atestação.

Recurrence-check: janela 14 dias, 393 registros, threshold 3, zero flagged keys e
zero flapping keys. Sem gatilho formal para escalation. Causas observadas:
contrato/implementação (autoridade, JSON, SemVer) e validação (casos negativos
restritos ao helper). Prevenção: regressões por classe atravessando checker,
CLI, MCP, index, archival, publicclaims e obrigações. Assess tech-debt encontrou
um marcador coberto, mas complete=false por dois follow-ups históricos sem
descrição; não foi declarado como clean. Integração é inventário heurístico
(64 integrações, 63 gaps), não prova de execução de providers.

## Follow-up aceitável

### F11 — low — A spec retém instruções de authoring já superadas

Locais: `.pose/specs/2026-10-13-pose-release-manifest-tag-integrity.md:29`,
`:51` e `:119`. Constraints diz que esta interação só escreve a spec; Artifacts
diz que somente a spec é criada neste lote; Validation diz que os testes ainda
não foram criados ou executados. Essas frases descrevem a etapa anterior e
contradizem os paths/testes e a trace atuais. O status continua in-progress e a
revisão pendente está explícita: corrigir essas frases não muda critério de
aprovação, autoridade ou comportamento. Por isso é follow-up editorial low,
sem usar a classificação para aceitar qualquer defeito funcional.

- [open] F11: reconciliar o texto de authoring com a implementação e validação efetivamente executadas, mantendo separado o closeout pendente (owner:@core crit:low review:2026-12-01)

O item fica registrado no parecer; não foi inserido na spec porque esta revisão
tem bloqueantes e a instrução limita a persistência a reports/results. Na futura
trilha sem bloqueantes, transportar esse item para Follow-ups da spec se ainda
estiver pendente.

## Disposição e limites

Falta de owner de componente, warnings de
proveniência históricos e limitações das avaliações são contexto do gate, não
desculpa para aceitar os dez defeitos bloqueantes. Nenhum deles foi transformado em
`[open] crit:low`, pois os caminhos descritos podem aprovar indevidamente ou
contradizem requisito explícito de entrega. Não foi alterada a spec.

Contratos públicos: recusa mais estrita classificada minor no changelog, e
campos snapshot/snapshot_limits adicionados; a documentação promete autoridade
da tag e paridade ainda não demonstradas. Os achados não pedem adotar policy,
pin de issuer, assinatura, backfill de prova nem mutação de tags reais.

Artefatos gerados incidentalmente por assessments/checks foram restaurados ao
HEAD após a execução. Permanecem somente parecer e evidências não commitadas.
Não houve staging de contribuição upstream, commit, push, selo ou atestação.

## Checks finais

- `pose validate --strict`: **140/140 pass**, zero failed/errored/skipped;
  inclui build, go test ./..., vet e os produtores de integração/race da matriz.
  O produtor específico de B38 passou. Execução de 315,4 s, código do HEAD;
  runner instalado da base origin/main, cujo motor de validate não muda no PR.
- Binário compilado deste HEAD: `check --strict` passou com 29 warnings;
  `artifact-check --strict` passou com 643 warnings de órfãos, 20 claims e
  25 observed; `surface-check --strict --results` passou com um target,
  140 resultados e zero findings. A tentativa de surface-check anterior ao fim
  da validação recusou falta de integração; a repetição após o resultado passou.
- Lint estrito da spec passou; Tasks é seção opcional ausente.
- Govulncheck v1.4.0: nenhuma vulnerabilidade encontrada.
- Gitleaks v8.21.2, redacted, diff origin/main..HEAD: cinco commits atribuídos,
  nenhum segredo encontrado.
- `git diff --check origin/main...HEAD`: passou.
- Probes independentes de domínio e CLI: execução completa, com aceitações
  negativas documentadas em F01–F10. Não alteram fontes do repositório.

Resultados e limitações finais estão no JSON de evidência e no arquivo de
validação independente a ele vinculado. Revisão não aprovada mesmo quando os
checks registrados passam: os probes expõem lacunas de cobertura.

## Reexecução de confirmação no mesmo HEAD

O parecer e as evidências já estavam não commitados na entrada desta execução.
Foram conferidos contra o código e o diff do mesmo HEAD, e os probes embutidos
foram materializados novamente em overlay temporário: reproduziram F01–F10,
incluindo os controles negativos recusados. A preparação read-only reconfirmou
o mesmo bundle e plan digest, sem selo.

O binário recompilado do HEAD executou novamente `validate --strict`: 140/140
pass, nenhum failed/errored/skipped. `check --strict` passou com 29 avisos;
artifact-check e lint-spec passaram; govulncheck e gitleaks passaram. Também
foram repetidos suggest-review, assess-design, discover if-stale, integrate,
tech-debt e recurrence-check. A evidência JSON distingue esta reexecução dos
resultados anteriores e contém a nova saída dos probes e o hash da validação.

A decisão permanece changes requested; nenhuma atestação foi criada.

`surface-check --strict` repetido com a nova validação passou: um target, 140
resultados, zero findings. Provenance digest:
`sha256:ab8b495a2fde8822003f264b132c6573210ec7523e0ff5d8771a0a959c6b8908`.
