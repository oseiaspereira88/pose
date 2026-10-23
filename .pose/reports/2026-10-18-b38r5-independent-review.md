# Revisão independente — b38r5 / PR #198

Decisão: **changes requested**. **Seis bloqueantes de merge e sete follow-ups
aceitáveis, low.** Spec `pose-release-manifest-tag-integrity`, HEAD
`63d0bdf5829a0f0acab756befcc01e31b25f7653`, repositório `oseiaspereira88/pose`.
Não selar, atestar, fechar, commitar ou enviar este estado.

Revisor: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI;
implementador Claude/Anthropic. Identidade e separação declaradas, vendors
diferentes; nenhuma alegação de autenticação criptográfica. Não foi criada
atestação, com ou sem envelope. Policy e issuer pins não foram alterados.

[Evidência compacta, fontes e saídas dos probes](../results/b38r5-independent-review.json).
Os probes foram adicionados apenas em cópia temporária dos fontes deste HEAD,
com o módulo irmão mcp-enforce; nenhum arquivo de produção/teste foi modificado
no clone. Não houve delegação.

## Rules applied during review

- Change type: mixed — bugfix Go, contrato de arquivo/CLI/MCP, testes e docs.
- Skills: `.agents/skills/pose-review/SKILL.md` e
  `.agents/skills/pose-spec-closeout/SKILL.md`.
- Workflow: `.pose/workflows/review.md`.
- [x] `.pose/rules/backend-go.md`: propagação de erros, domínio/consumidores e checks Go.
- [x] `.pose/rules/security.md`: integridade de inputs, paths e prova; scanners.
- [x] `.pose/rules/documentation-style.md`: spec, manuais e scaffold.
- [x] `.pose/rules/delivery-evidence.md`: distinguir execução verde de correção.
- [x] `.pose/rules/knowledge-governance.md`: consultar decisões e notas existentes.
- [x] `.pose/rules/_base-recurrence.md`: recorrência das classes e prevenção.
- Não aplicáveis: frontend, Kubernetes, Terraform, Workers; sem delta nesses domínios.

Lidos spec/template, workflow, regras, ADR-018, ADR de archival de 2026-09-11,
ADR/nota temporal de 2026-10-14, revisões anteriores, diff completo de produção
e testes contra origin/main, manuais/scaffold e índices/resultados derivados.
`pose context` resolveu a autoridade local sem ambiguidade. O diff é 46 arquivos;
JSON de índices/evidências foi tratado como material derivado, não como prova
autossuficiente da implementação.

## Plano e execução

Bundle apenas preparado, **não selado**: `rvb-85f5d554bbef11da`.
Digest: `sha256:85f5d554bbef11da8861b56f25502b860ee26563a98d5a1410d8e431195e2728`.
Plan: `sha256:28e7203f13dd69012afbe9cded907d03771d4aad33b7c420f3e949dc390a0fc1`.
Patch: `sha256:346bf010a95da42b804c002d69afdd99f9c88a1154369f87693f244eda91b2b5`.
Tree: `sha256:a8b2160735a1effe87f6bce2885c8f5001e6d55549f57d671d3d6d7f1e4ddd13`.

15 critérios, band critical, different-actor, componente pose-mcp com owner
ausente. Perfis spec-closeout, backend-review, engineering-judgment e
high-criticality-review. O change set atribui 13 commits; o range de provenance
inclui outros trabalhos, por isso não foi confundido com o diff atribuível.
Os trailers POSE-Spec foram conferidos no corpo das mensagens e por artifact-check.

Ferramentas requeridas ativas: validate dos módulos Go, artifact-check e
surface-check executados. Recomendadas: assess-tech-debt, assess-integrate e
suggest-review executadas. Assess-design dispensado após leitura direta de todo
o delta estrutural e suas bases R/D; assess-discover não foi atualizado porque
não há fechamento nem mudança de código nesta revisão, e foram consultados os
assessments existentes. Review-check e closeout-check são completion-only e
ficaram diferidos junto com seal/attest/verify/close pela instrução explícita
para o caso de bloqueantes. Recurrence-check de 14 dias, followups --all e lint
estrito executados. Nenhum follow-up foi disposed como spawned/covered/duplicate.

- Binário `/tmp/b38r5-pose` compilado dos fontes de produção deste HEAD.
- `validate --strict --module pose-mcp`: **137/137 pass**, 309,0 s;
  inclui build, go test ./..., vet e os checks race registrados.
- `validate --strict --module mcp-enforce`: **3/3 pass**. Os dois módulos
  cobrem as 140 etapas Go aplicáveis; nenhum outro módulo afetado possui checks adicionais.
- `check --strict`: **FAILURE, 2 erros, 29 warnings**. Os dois erros são as
  tags históricas v1.1.0/v2.0.0, detalhadas em F06.
- `artifact-check --spec ... --strict`: 37 claims, 46 paths observados,
  640 warnings globais de orphan e zero finding error/critical. As quantidades
  distintas incluem artefatos derivados; não demonstram, por si, falta de trailer.
- `surface-check --spec ... --strict`: falhou com stale-evidence e
  unconsumed-capability; o resultado selável existente está ligado a provenance
  anterior. Não foi regenerado results_path para fingir closeout com bloqueantes.
- `lint-spec ... --strict`: sucesso; um warning de seção opcional Tasks.
- govulncheck v1.4.0: **No vulnerabilities found**.
- gitleaks v8.21.2, origin/main..HEAD: **13 commits, no leaks found**.
- `git diff --check origin/main...HEAD`: sem falhas.
- 75 manifests históricos carregam. Isso **não** prova a integridade dos
  snapshots: o check completo e o probe histórico detectam F06.
- Tech-debt foi executado e declarou scan INCOMPLETE por duas specs preexistentes
  inválidas: pose-instance-engine-version-tracking e
  pose-knowledge-durable-reference-type; zero markers não prova cobertura completa.
- Integrate reportou gaps preexistentes de consumidores HTTP /healthz e /mcp.
  Recurrence: 393 registros, zero flagged keys (limiar 3, janela 14 dias).

Os probes negativos falham deliberadamente ao exigir a recusa que falta;
isso reproduz os findings e não contradiz os 140 checks registrados verdes.
O probe exploratório de archival foi adjudicado separadamente abaixo.


## Bloqueantes de merge

### R5-F01 — high — Status/MCP aceitam candidato semanticamente inválido

Locais: `pose-mcp/internal/pose/release_lifecycle.go:402`, `:626`.
R2/R3/R7. Probes `TestB38R5PreparedSemanticParity` e
`TestB38R5PreparedStatusVsCheck`.

Sem tag, GetReleaseStatus chama apenas ReleaseFilesGaps. Esse helper confere
digests dos arquivos, mas não recompõe seus campos nem verifica policy ou fonte
de versão. Quatro casos retornam `State:prepared, Gaps:[], Sound:true`: notas
arbitrárias com NotesDigest atualizado; VERSION alterado para 9.0.0; body do
manifest adulterado com input digest/notas recompostos, preservando o fragment
original; release policy alterada para schema 999. O check preparado recusa os
quatro, respectivamente por renderização, versão, input digest e schema.

Vizinhos: MCP `pose_release_status` retorna diretamente GetReleaseStatus
(`internal/mcpserver/server.go:1491`); o índice consome o mesmo status.
O ramo tagged verifica a árvore e não reproduz essa lacuna. Publicclaims exige
Sound para publicação e open-next exige verified+Sound: não se alegou que um
candidato preparado isolado satisfaz esses gates. O problema é a falsa
qualificação de integridade do próprio candidato e a quebra explícita da paridade.

Ação: compartilhar a recomposição completa do candidato entre check e domínio,
incluindo notas renderizadas, fragments, policy e version_source.

### R5-F02 — high — Checker global não qualifica o worktree do cut

Locais: `pose-mcp/internal/cli/release_lifecycle.go:851`, `:887`.
R3/R7. Probe `TestB38R5CheckerWorktreeParity`.

Com policy adotada e um cut correto, alterar apenas as notas do worktree produz
`canonical notes digest mismatch` em release check. O subcheck de release do
checker acumula **zero erros**, tanto antes da tag como depois de criar uma tag
cuja árvore continua correta. Ele percorre fragments e snapshots tagged, mas
não aplica a verificação do candidato nem o binding da cópia do worktree.
A evidência prova a lacuna do subcheck responsável; não afirma que toda a
fixture mínima passa os demais gates estruturais do projeto.

Vizinhos: release check e status tagged detectam esse mesmo drift;
record tagged/published/verified já exige o worktree do cut. O checker global
não deve anunciar integridade por um critério diferente.

Ação: usar a mesma verificação de candidato/cópia do cut no checker, preservando
leitura histórica da tag, commits posteriores legítimos e separação estado/integridade.

### R5-F03 — high — Perda de fonte no worktree ainda vira ausência

Locais: domínio `release_lifecycle.go:115`, `:162`, `:287`, `:375`;
CLI `release_lifecycle.go:853`, `:858`.
R3/R7. Probes `TestB38R5DanglingPoliciesAndFragments`,
`TestB38R5PendingLossPassesCheck`, `TestB38R5PolicyLossPassesCheck`,
`TestB38R5DanglingAncestor`, `TestB38R5DanglingEvents`,
`TestB38R5CheckerPendingLoss`, `TestB38R5CheckerPendingError`.

A guarda ENOENT nova ainda segue links antes de decidir ausência:

- `.pose/policy/release.json -> lost` usa a policy legacy, e release check fica
  válido se ela coincide com o digest. A entrada primária existe e está perdida.
- `unreleased -> lost` vira lista vazia; check preparado aceita e ambos os
  subchecks release/changelog deixam de reportar o inventário perdido.
- `.pose -> lost`, sem tags, devolve `Pending:[], Releases:[], nil`. O Lstat de
  `.pose/releases` não encontra a perda no ancestral.
- `events.jsonl -> lost-events` vira stream legitimamente vazio. Status tagged
  continua Sound sem divulgar a perda de evidência. Isto é distinção de fonte,
  não extensão ao replay temporal de B39.

Vizinhos: o checker ainda descarta o erro explícito de LoadReleaseFragments
para pending e o erro de ReadDir de changelogs. Um pending malformado também é
ignorado pelo subcheck release (o checkChangelogs possui diagnóstico próprio
para esse caso; não se afirmou bypass global por malformed). ReleaseFilesGaps
engole ENOENT de archive dangling quando o manifest é vazio. A raiz direta de
releases dangling foi corrigida na r5; a guarda não cobre seus ancestrais.
Git tree/blob da tag agora conserva falhas e exige arquivo regular: não foi
reaberto o defeito anterior de classificação dos objetos Git.

Ação: provar ausência examinando entradas/ancestrais pertinentes sem seguir
links quebrados, propagar erros de inventário e usar empty/default/fallback só
para ausência legítima. Preservar o controle de release vazia sem pasta no Git.

### R5-F04 — medium — Prepare escreve por manifest ilegível existente

Locais: `pose-mcp/internal/cli/release_lifecycle.go:239`, `:348`.
R3/R7. Probe `TestB38R5PrepareOverwritesDanglingManifest`.

Preparar release vazia permitida, remover sua pasta de archive vazia (Git não
versiona diretório vazio) e substituir manifest.json por link para lost.json.
Novo prepare --apply retorna sucesso e **cria lost.json através do link**,
reconstruindo um manifest onde havia um candidato ilegível. O caminho é todo
sintético e interno à fixture; não é demonstração ofensiva ou de escape.
A guarda de R2-F08 trata erro de leitura diferente de ENOENT, mas o link
quebrado passa como espaço inexistente para escrita.

Vizinhos: policy fallback, inventários e stream de eventos repetem a classificação
em F03. A pasta de archive existente barra alguns casos, mas não esse controle
legítimo de release vazia. Não se dependeu de falha de processo nem concorrência.

Ação: recusar uma entrada manifest existente e ilegível antes de qualquer efeito;
não criar seu alvo nem fabricar novo freeze. Acrescentar regressão da guarda,
sem ampliar para recuperação/publicação de B41.

### R5-F05 — medium — Changelog policy não validada no candidato

Locais: CLI `release_lifecycle.go:413`, `:422`, `:439`;
domínio `changelog_policy.go:46`. R1/R3/R7.
Probe `TestB38R5ChangelogContractParity`.

Acrescentar ao candidato correto uma changelog policy `{broken` ou
`{"schema_version":999}` mantém release check sem gaps. LoadChangelogPolicy
ignora erro de decode e usa defaults; tampouco valida schema. Commitar a mesma
policy e criar a tag faz ReadReleaseSnapshot recusá-la corretamente. Portanto
um candidato qualificado pode ficar inválido só ao congelar exatamente esses
mesmos inputs em uma tag.

Vizinhos: prepare, status pending, preparedReleaseGaps e checker usam o loader
permissivo; changelogPolicyAt valida decode/schema; LoadReleasePolicy valida
seu próprio schema. O checker de changelog valida sintaxe, mas não schema.
Nenhuma policy do clone foi alterada para reproduzir.

Ação: disponibilizar loader validante para qualificação/preparação e compartilhar
o contrato com changelogPolicyAt. Defaults cabem na ausência legítima; schema
futuro deve ser unsupported. Não aplicar policy atual a cuts históricos.

### R5-F06 — high — Renderer atual invalida tags históricas legítimas

Locais: `pose-mcp/internal/pose/release_manifest_integrity.go:527`;
`release_lifecycle.go:230`. R3/R6/R8.
Probe `TestB38R5HistoricalSnapshotRendering`; `check --strict` deste HEAD.

As tags reais e imutáveis v1.1.0 e v2.0.0 têm manifest legível, digests das
notas corretos e blobs originais, mas ReadReleaseSnapshot as declara não intactas.
A única diferença de renderização é uma seção **Compatibility** que o
RenderReleaseNotes atual acrescenta consultando ContractsIntroducedIn, do
registry compilado atual. Essa seção não existia nas notas congeladas por
esses cortes. O checker termina com dois erros, ambos por essa comparação.
A evidência compacta contém os diffs precisos de notas originais versus renderer atual.

Vizinhos: snapshot chama RenderReleaseNotes usando policy do cut, mas o registry
e a implementação do renderer permanecem atuais. Status, checker, release check
e record herdam a falsa recusa do snapshot. Prepared usa o renderer atual, o que
é apropriado ao novo corte; aplicar esse mesmo contrato sem versão aos cortes
antigos viola R6. O teste novo de 75 manifests históricos exercita apenas o
loader e não detecta essa regressão de snapshots (ver L07).

Ação: vincular a recomposição ao contrato de renderização do cut e tratar
legacy/unsupported explicitamente, preservando a prova dos digests. Não
reescrever notas/tags históricas, aumentar cutoff para esconder a história,
nem remover indiscriminadamente a recomposição que detecta adulteração real.

## Follow-ups aceitáveis — low

Propostos para o próximo closeout; **não foram escritos na spec**, pois há
bloqueantes. Nenhum permite aprovar conteúdo materialmente divergente.

- R5-L01: recusar `version_evidence.kind:null`, hoje equivalente a kind vazio
  do engine; source exato/value continuam obrigatórios. Probe LowNullKind.
  (owner:@core crit:low review:2026-12-01)
- R5-L02: harmonizar filtros auxiliares do archive (README/.gitkeep/outros).
  Probe LowReadme: loader ignora README, snapshot recusa. É recusa excessiva.
  (owner:@core crit:low review:2026-12-01)
- R5-L03: mostrar Snapshot.Limits no status humano e qualificar diagnóstico
  do candidato em publicclaims. Probe LowHumanLimits. JSON mantém limites e
  publicação exige Sound. (owner:@core crit:low review:2026-12-01)
- R5-L04: ampliar pattern do integration check para TestF*/TestR2*/TestR4*;
  go test ./... já os executa. (owner:@core crit:low review:2026-12-01)
- R5-L05: reconciliar texto de authoring da spec que afirma que código/testes
  não existem. Manter o item editorial já aberto, sem duplicação.
  (owner:@core crit:low review:2026-12-01)
- R5-L06: explicitar/harmonizar symlinks de worktree com bytes idênticos:
  FilesGaps/archival seguem o link, LoadReleaseFragments recusa. Probe
  WorktreeSymlinks preserva digests; não demonstra conteúdo divergente aceito.
  A árvore da tag já exige arquivo regular.
  (owner:@core crit:low review:2026-12-01)
- R5-L07: fazer smoke histórico falhar ou declarar skip quando ReadDir falha
  ou não há fixtures; hoje pode passar com zero releases. Nesta revisão as
  **75 releases** foram verificadas explicitamente pelo probe AllHistoricalLoads.
  (owner:@core crit:low review:2026-12-01)

## Varredura por classe e adjudicação

| Classe | Consumidores inspecionados | Conclusão |
|---|---|---|
| Ordenação por texto | SemVer, previous, releaseInputs, status, cutoff, backfill, publicclaims, fila/open-next | Comparador semântico compartilhado; controles 9/10, rc.2/rc.10/final passaram. Sorts de paths/specs/categorias são apresentação, não precedência. |
| Ausência versus perda | Git trees/blobs, primary/legacy policies, releases/pending/archive, eventos, prepare, check/status/MCP e archival | F03/F04; ausência legítima de pasta vazia preservada como controle. |
| Campo sem validação | raiz/nested manifest, projections/digests/notes, versão/source/kind, release/changelog policy, Sound e consumidores | F01/F02/F05/F06; kind null residual equivalente em L01. Replay/validação semântica de provider events permanece B39. |
| Comparação de caminhos | clean/relative, namespace refs/tags, versão/source real, modes de Git, worktree/tag, archival e subject de review | Binding da tag corrigido; variantes dangling F03/F04; worktree byte-identical L06. |

Probe exploratório ArchiveInvalidSnapshot não virou finding: archival é
**testemunho limitado de spec/path/bytes**, conforme ADR de 2026-09-11. Um fragment
correto continua testemunhando sua mudança de local se notas da release estiverem
erradas; Intact ali não certifica toda a release. Não impor retroativamente uma
nova dependência de integridade das notas ao contrato do witness.

A forma barata de satisfazer o gate sem seu valor é ajustar digests redundantes
sem recompor fontes, ou sumir com inventários para obter vazio. Os probes F01,
F03 e F05 provam os caminhos restantes. O corpus adversarial tem controles
vizinhos de produtor indisponível e metadata local versus publicação, mas ainda
não uma linha dedicada às variantes B38 acima. Testes verdes não suprem essa
qualificação negativa.

## Critérios, recorrência e limites

Requirements/correctness/backend-input-errors/integration-impact não passam por
F01–F06; segurança de integridade e operabilidade falham especialmente por
F03/F04. Backend-contracts e compatibility têm classificação minor documentada,
75 manifests históricos legíveis e sem mudança de policy, mas a paridade pública
e a validade dos snapshots históricos ainda falham por F06. Escopo permanece B38; não se exigiu replay temporal B39, exclusividade
B40 nem recuperação B41. D1 continua causalmente adequado; a implementação
mantém verificadores parciais paralelos que invalidam sua suposição de contrato
compartilhado. O menor ajuste suficiente é reunir/reutilizar essas verificações
e conservar erros/coverage, sem framework novo, novo schema ou pins.

Documentação requer L05 e a correção do runtime antes de sustentar sua claim de
paridade. Não há novo mecanismo concorrente na mudança; a matriz inclui os
checks race já registrados. Nenhuma conclusão sobre independência cognitiva
ou autenticidade histórica foi inferida de identidade textual.

Recorrência: domínio Go/contrato e validação; F01/F02/F05 são recomposição parcial
entre consumidores, F03/F04 repetem perda interpretada como ausência; F06 é contrato atual aplicado
retrospectivamente à história. A prevenção
é uma verificação canônica compartilhada com regressões de cada consumidor e
matriz de ausência/ilegibilidade/tipo/ancestral. Recurrence-check não sinalizou
chave acima do limiar; isso não apaga a recorrência observada nas revisões.
Nenhum feedback upstream foi staged/submetido: Contributor Mode exige decisão
explícita do usuário, e a autorização desta tarefa é produzir o parecer local.

Não há aceitação de risco nem autorização de merge: os lows são propostas de
backlog, portanto não foi criado handoff de risco aceito. Spec continua
in-progress, sem completed_at. Não houve close, commit ou push.

## Estado do workspace

O descriptor `.pose/indexes/generation.json` já estava modificado no início.
Artifact/surface-check atualizaram pelos writers normais o índice derivado
`delivery-integrity.json` e seu descriptor; ficaram sem commit. Os outputs
transitórios de assess-integrate/tech-debt foram recolhidos na evidência e
restaurados ao estado inicial. Somente parecer/evidência e esses índices derivados
ficam como delta da revisão; não houve edição de spec, policy ou fonte.
