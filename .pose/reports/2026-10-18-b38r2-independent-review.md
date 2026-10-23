# Revisão independente — b38r2 / PR #198

Decisão: **changes requested**. **Oito bloqueantes de merge e um follow-up
aceitável, editorial e low.** Não selar, atestar, fechar, commitar ou enviar este
estado. A spec permanece `in-progress`. Este parecer é da segunda execução de
review; não sobrescreve o parecer anterior.

- Repositório: `oseiaspereira88/pose`, PR #198.
- Spec: `pose-release-manifest-tag-integrity`.
- HEAD revisado: `c37cf39de59da51ac124dd70d0d6f11657cfb5d4`.
- Base `origin/main` e merge-base local: `8aa729a76ab1c094690a0892263a86bf3bb7bb42`.
- Identidade declarada: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI.
  Implementador: Claude/Anthropic; execução e vendor diferentes, separação
  declarada, sem alegação de autenticação criptográfica.
- Evidência compacta e fontes dos probes:
  [b38r2-independent-review.json](../results/b38r2-independent-review.json).
- Policy e issuer pins não foram alterados. Não foi criada atestação, assinada
  ou sem envelope, pois a instrução do solicitante proíbe atestar com bloqueantes.

## Rules applied during review

- Change type: mixed — correção Go, contrato de manifest, consumidores,
  testes, matriz de validação, documentação e scaffold.
- Skills: `.agents/skills/pose-review/SKILL.md` e
  `.agents/skills/pose-spec-closeout/SKILL.md`.
- Workflow consulted: `.pose/workflows/review.md`.
- Rules selected:
  - [x] `.pose/rules/security.md`: integridade, entrada JSON, paths, autoridade,
    scans de segredos e dependências.
  - [x] `.pose/rules/backend-go.md`: propagação de erro, contratos, consumidores,
    build, testes e vet.
  - [x] `.pose/rules/documentation-style.md`: spec, notas, manuais e scaffold.
  - [x] `.pose/rules/delivery-evidence.md`: separar checks verdes de entrega e
    de julgamento semântico; não declarar closeout sem gate.
  - [x] `.pose/rules/knowledge-governance.md`: consulta às decisões e riscos;
    não há handoff de entrega aceita, pois o merge foi recusado.
  - [x] `.pose/rules/_base-recurrence.md`: classificar a recorrência por causa.
- Rules not applicable: frontend, Kubernetes, Terraform e Workers; não há delta
  nesses domínios. Não há mudança de concorrência de produção; os checks race
  já registrados na matriz continuam aplicáveis e são executados por validate.

Foram lidos a spec, seu template, o diff integral contra `origin/main`, a revisão
anterior, os testes e consumidores de manifest/snapshot/projeção, as políticas
vigentes e os ADRs de ledger imutável, fonte de versão, compatibilidade histórica
e archival por testemunho. `pose context` resolveu a autoridade local sem
ambiguidade. Foram consultados os AGENTS específicos dos manuais de scaffold.

## Plano efetivo e ferramentas

Plano: `spec-closeout@1`, overlays backend-review, engineering-judgment e
high-criticality-review; 15 critérios, componente `pose-mcp`, criticidade high,
independência `different-actor`. Falta owner nos metadados do componente;
paths documentais/governança não mapeados foram revisados diretamente.

Preparação **não selada**:

- Bundle: `rvb-682907e72c85787d`.
- Bundle digest: `sha256:682907e72c85787dabf54347af5a45c5c54c5fd5fe60b7450ffcffd0d9e46df1`.
- Plan digest: `sha256:4734ae96d14cde7d3ba803a337bf6a179256139a3dd40f617152967450858a5d`.
- Patch digest: `sha256:5fcb084f1281849d2203c1e9e167761ad416a7acf68f24b6ae539889141b26c4`.
- Tree digest: `sha256:0133e17be3c45d967df032f0da30a087a0d5830511dc88fd39fd6e1ebb490663`.

A atribuição POSE observa sete commits, com 26 claims e 32 paths observados;
seu range de proveniência inclui outros trabalhos anteriores à base do PR.
A comparação do PR e o subject atribuído foram inspecionados separadamente;
não se inferiu falta de trailer pelo parser de trailers do Git.

- Obrigatórias ativas: `validate --strict` global e de `pose-mcp`,
  `artifact-check --spec … --strict`, `surface-check --spec … --strict`.
- Recomendadas: suggest-review, assess-design, assess-discover `--if-stale
  --component pose-mcp`, assess-integrate e assess-tech-debt executadas.
- Adicionais: `check --strict`, lint estrito da spec, recurrence-check,
  followups `--all`, scanner de segredos no diff, govulncheck e probes negativos.
- Completion-only: review-check, closeout-check, seal, auto-attest, attest,
  review verify e close **adiados por bloqueantes**, conforme ordem expressa.
  Não se produziu uma atestação negativa para contornar essa ordem.

## Achados bloqueantes de merge

Todos os resultados abaixo foram reproduzidos em cópia temporária dos módulos.
Os `PASS` dos probes significam que a execução diagnóstica observou o defeito;
não representam asserção de que o comportamento de produção esteja correto.
Cada fonte e sua saída estão embutidas na evidência compacta para reprodução.

### R2-F01 — medium — O contrato estrito termina no objeto raiz

Local: `pose-mcp/internal/pose/release_manifest_integrity.go:82`.
Requisitos: R1/R8. Probe: `TestB38R2NestedSchema`.

Em `fragments[0]`, remover `breaking`, usar `breaking:null`, trocar `spec` por
`SPEC` ou acrescentar `refs:null` resulta em `issues=[]`. O decoder de struct
aceita aliases por caixa e transforma ausência/null em valores zero; a inspeção
prévia de presença/nomes/null contempla somente o objeto raiz. O input digest
permanece válido porque a estrutura decodificada é a mesma.

Vizinhos: LoadReleaseManifest, snapshot, status, índice e archival herdam a
aceitação. version_evidence tem allowlist explícita, mas kind vazio/null ainda
aciona o problema de F04. O decoder de evidence em `release record` também usa
um único Decode, sem EOF/duplicatas; isso é uma lacuna adjacente preexistente do
contrato de eventos, cujo replay é B39, não uma alegação de que B38 resolveu esse
contrato. Os controles de EOF, duplicatas e nomes exatos **na raiz** agora passam.

Ação: validar o contrato de cada objeto aninhado, incluindo presença, tipos,
null e nomes exatos, preservando `refs` opcional; testar os loaders consumidores.

### R2-F02 — high — O testemunho de archival ainda aceita outra spec

Local: `pose-mcp/internal/pose/release_archive.go:62`.
Requisitos: R2/R3/R8. Probe: `TestB38R2ArchiveWitness`.

Com blob `alpha.md` identificando `spec: alpha`, trocar a spec do manifest para
`beta` e recompor suas projeções/input digest produz um registro
`ArchivedFragment{Spec:beta,Intact:true}`. `archivalLedger.resolve` aceita esse
registro para beta com os paths rastreados. O digest prova os bytes de alpha,
mas não a autoria semântica atribuída a beta. O manifest do worktree tampouco é
vinculado ao manifest da tag nesse consumidor.

Vizinhos: o novo snapshot recusa a identidade divergente, mas o ledger não o
consulta nem extrai a identidade do blob. Artifact-check e delivery graph usam
esse testemunho para resolver claims; testes de review archival dependem dele.
O loader estrito comum comprova coerência interna, não identidade autoritativa.

Ação: compartilhar a verificação semântica do testemunho e seu vínculo com o
cut, preservando archival preparado legítimo; cobrir resolução de claims com
spec adulterada e arquivo cujo hash permanece correto.

### R2-F03 — high — Status e checker divergem sobre perda do cut corrente

Local: `pose-mcp/internal/pose/release_lifecycle.go:387`.
Requisitos: R3/R5/R7. Probes: `TestB38R2StatusWorktree`, `TestB38R2StatusVsCheck`.

Após tag íntegra, apagar as notas ou `alpha.md` no worktree mantém status
`tagged`, `Sound=true`, `gaps=[]`. Para as mesmas notas ausentes, release check
retorna `canonical notes missing`. Status só compara o manifest corrente com o
da tag, não a completude dos arquivos que esse manifest declara.

Vizinhos: MCP delega ao mesmo GetReleaseStatus; índice publica sua projeção.
Publicclaims, obrigações e open-next passaram a usar Sound, mas recebem esse
resultado incompleto. A perda da pasta de manifest, em contraste, já gera gap
e Sound=false. O CLI record faz a verificação mais completa do worktree.

Ação: compartilhar os resultados de vínculo e coverage, mantendo a integridade
histórica da tag separada da perda/divergência do worktree, e eliminar a decisão
positiva inconsistente nos consumidores de Sound.

### R2-F04 — high — Evidência de versão ainda permite fonte falsa e limite evitável

Locais: CLI `release_lifecycle.go:425` e `:435`;
`release_manifest_integrity.go:432` e `:464`.
Requisitos: R1/R3/R7. Probes: `TestB38R2PreparedSource`,
`TestB38R2VersionPolicy`, `TestB38R2HumanLimits`.

Três variantes da mesma falta de vínculo:

- Candidato sem tag, policy text apontando VERSION correto: manifest com
  `source:missing.txt`, value correto e input digest recomposto retorna zero
  gaps. A recomposição usa a evidência do próprio manifest; a leitura real só
  compara value e ignora source/kind.
- Na tag, policy JSON contém `key:version` e o blob diz `9.9.9`, enquanto
  release/value dizem `v1.0.0`: o snapshot fica Intact com mero limit. A key está
  disponível na policy histórica que acabou de ser lida; não é falta de input.
- Policy sem version_source, source arbitrário VERSION e kind vazio: até um
  blob `9.9.9` é aceito como suposta fonte interna do engine, com Intact e limit,
  sem exigir `EngineVersionFile`.

Vizinhos: prepare/releaseVersionEvidence conhecem a fonte exata e
ReadReleaseVersionSource já conhece JSON/key. Snapshot só compara source/kind
quando a policy tem VersionSource. Intact/Sound ignoram Limits, checker global
não os divulga, e CLI humano imprime `valid prepared snapshot; tagged=true` ou
apenas `tagged`, escondendo a coverage que existe no JSON. A limitação legítima
de uma fonte realmente não recomputável continua distinta dessas contradições.

Ação: conferir todos os campos da evidência contra a fonte autoritativa;
recomputar JSON com a policy histórica quando possível; reconhecer fallback
interno pelo path exato e divulgar coverage em todos os canais/consumidores.

### R2-F05 — high — Falhas Git e perda de snapshot ainda viram ausência limpa

Locais: `release_lifecycle.go:403` e `:410`;
`release_manifest_integrity.go:240`; CLI `release_lifecycle.go:489`.
Requisitos: R3/R4/R7. Probes: `TestB38R2Inventories`,
`TestB38R2GitResolutionFailure`.

Falha injetada somente em `git tag --list` retorna `Releases=[]` e nil error no
status. Uma tag existente sem manifest no tree, com pasta corrente ausente,
também desaparece: o segundo loop descarta o erro. Falha injetada em cat-file
para uma tag existente vira ErrReleaseTagMissing; releaseTagState então responde
`tagged=false,gaps=[]`. A distinção pretendida no comentário não é alcançável
para essas falhas do resolver.

Vizinhos: CLI resolveReleaseTag e releaseTagSnapshotGaps também descartam o erro;
status, MCP, obrigações e publicclaims herdam inventário incompleto. Checker
global e releaseInputs melhoraram a falha de listagem quando o repositório é
reconhecido, mas gitUnavailableAt ainda trata toda falha de rev-parse como
não-repositório. LoadArchivedFragments continua sem canal de erro/coverage;
seu testemunho positivo está em F02. Controle: pending malformado é recusado por
release check; diretório de release ENOTDIR já é recusado pelo status.

Ação: distinguir ausência comprovada, objeto não commit e erro de leitura;
reter diagnóstico/coverage para cada tag governada que não pôde fornecer
snapshot. Respeitar explicitamente o cutoff legacy, sem inventar releases.

### R2-F06 — medium — Os inputs históricos não recebem a validação do seu contrato

Locais: `release_manifest_integrity.go:306` e `:390`.
Requisitos: R2/R3/R6. Probes: `TestB38R2PolicyValidation`,
`TestB38R2InvalidCategory`.

Policy schema 1 adotada sem provider/repository é recusada por LoadReleasePolicy,
mas ReadReleaseSnapshot aceita seu digest e retorna Intact. Um fragment com
`category:invented` é recusado pelo loader normal sob a policy default do próprio
cut, mas aceito pela tag quando o manifest repete a categoria. O leitor histórico
compara a categoria, sem validar seu domínio, e não lê a policy de changelog.

Vizinhos: LoadReleasePolicy valida adoção e VersionSource.Validate; o helper
histórico verifica somente schema. LoadReleaseFragments valida categorias sob
ChangelogPolicy; a recomposição do candidato o usa, o snapshot não. Status,
checker global e record confiam em Intact. Concordância entre dois campos
inválidos não estabelece um cut válido.

Ação: reutilizar validação pura dos contratos do cut e obter sua policy de
changelog; declarar limites legacy/unsupported, sem aplicar policy futura.

### R2-F07 — medium — O digest das notas não comprova sua projeção canônica

Local: `release_manifest_integrity.go:353`.
Requisitos: R2/R3. Probe: `TestB38R2Notes`.

Substituir as notas por `# An unrelated release` e mudar somente notes_digest
mantém `Intact=true,gaps=[]`. Fragments e release_input_digest permanecem
intocados. O snapshot verifica a igualdade de um hash autodeclarado com o blob,
mas não recompõe as notas derivadas dos fragments congelados.

Vizinhos: NewReleaseManifest e prepare usam RenderReleaseNotes para produzir a
projeção; tanto check preparado quanto snapshot só comparam o digest. `release
notes` retorna o arquivo congelado; status, checker e record aceitam o resultado.
Não se pede comparar com fragment ou policy atuais.

Ação: conferir a projeção sob o renderizador/contrato e policy de changelog do
cut; declarar coverage limitada quando a recomposição histórica for realmente
unsupported, sem apresentá-la como verificação completa.

### R2-F08 — medium — Prepare sobrescreve manifest que o novo loader recusa

Local: CLI `release_lifecycle.go:239`.
Requisitos: R1/R6/R8. Probe: `TestB38R2PrepareInvalidExisting`.

Release vazia permitida, archive ausente (Git não versiona diretório vazio),
manifest existente sem breaking: `release prepare --apply` retorna 0 e o
sobrescreve. O decoder antigo aceitava o campo ausente como false; o novo loader
agora retorna erro, e o consumidor ainda trata todo erro como ausência. O teste
observa `overwrote=true`, com mensagem de sucesso de congelamento.

Vizinhos: check e status recusam o erro de schema; archival descarta a entrada.
Prepare é o consumidor mutável dessa classe e prossegue com mkdir/escritas e
rollback que remove releaseDir. O caso sem archive representa, por exemplo, um
clone de cut vazio; não exige apagar uma pasta de fragment não vazio.

Ação: permitir criação só após ENOENT comprovado; manifest existente inválido ou
ilegível deve recusar sem sobrescrita/remoção. Cobrir clone vazio e rollback.

## Follow-up aceitável

### R2-F09 — low — Texto de authoring superado na spec

Constraints e validação ainda dizem que esta interação só escreve a spec e que
os testes não foram criados/executados, enquanto Final Report descreve o código
entregue. É o follow-up editorial já aberto. Não muda autoridade nem decisão de
gate, não permite aprovar indevidamente e pode permanecer aberto:

- [open] Reconciliar texto de authoring com implementação e validação executadas,
  mantendo o closeout pendente explícito. (owner:@core crit:low review:2026-12-01)

A spec não foi editada nesta execução, conforme a instrução para o caminho com
bloqueantes. Não foram criadas specs sucessoras nem dispostas duplicatas.

## Varredura das classes e limites de escopo

- Ordenação por texto: conferidos CompareReleasePrecedence, status, previous,
  releaseInputs, open-next, legacy cutoff, backfill, publicclaims e obrigações.
  Comparação numérica de tamanho arbitrário e identificadores prerelease está
  compartilhada; não há novo achado de ordenação. Sorts de paths/specs não foram
  confundidos com ordenação de versões.
- Ausência versus perda: conferidos blobs, árvores, resolver, inventário de
  tags/releases, archive, pending, status, checker, archival e prepare. F03,
  F05 e F08 documentam as variantes restantes; ausência legítima de diretório
  Git vazio não foi classificada como corrupção.
- Campo sem validação: raiz, fragments, version_evidence, policy, changelog,
  notas, eventos e testemunho de archival. F01/F02/F04/F06/F07; eventos temporais
  e tentativa/publicação continuam sendo contratos de B39/B41, sem declarar
  essa entrega como sua correção.
- Comparação de caminhos: path limpo e confinamento, alias/caixa de campos,
  fonte VERSION versus source declarado, fragments repetidos, vínculo manifest
  corrente/tag, namespace refs/tags e consumers de record/archival. F02/F03/F04
  e F08; branch homônimo não é mais tratado como tag no controle existente.

Atalho mais barato para satisfazer o gate formalmente: recompor campos públicos
e o input digest do manifest e deixar um consumidor que só observa esse objeto
validar a si mesmo. F02/F04/F07 mostram atalhos que ainda passam; F05 mostra o
atalho alternativo de transformar fonte ilegível em inventário vazio. O corpus
`pose-mcp/internal/cli/testdata/adversarial/README.md` tem controles adjacentes
`local-metadata-read-as-published` e `unavailable-producer-reads-as-empty`, mas
não tem caso B38 que exercite estes caminhos. A correção deve incluir regressão
na composição real e registrar sua delegação no corpus, não apenas testar
coerência interna do objeto.

Recorrência: domínio contratos/backend, causa implementação e validação parcial
nos consumidores. Vários achados são variantes das classes da primeira revisão.
O recurrence-check de 14 dias leu 393 registros, threshold 3, flagged_keys 0;
isso não invalida a recorrência observada manualmente. Prevenção: reusar loaders
semânticos/resultado de coverage e testar cada consumidor antes de declarar a
classe encerrada. Nenhum feedback upstream foi staged/submetido sem consentimento.

## Critérios do plano e compatibilidade

- scope: sem refactor alheio detectado; o objetivo continua incompleto.
- requirements / correctness: não satisfeitos semanticamente por F01–F08,
  apesar dos testes registrados verdes.
- security / backend-input-errors: F01/F04/F05/F08; scans de segurança verdes
  não substituem validação de integridade.
- compatibility / backend-contracts: recusa mais estrita foi classificada minor
  no fragment, Snapshot é aditivo; F06/F08 impedem aceitar compatibilidade como
  plenamente demonstrada.
- backend-integration-impact: F02/F03/F04/F05 nos consumidores reais, inclusive
  MCP delegando a GetReleaseStatus.
- backend-observability / operability: F04/F05 ocultam limits e erros.
- documentation: F09 é low; afirmações de completude dos manuais dependem da
  correção dos bloqueantes, não bastam como prova.
- backend-concurrency: sem nova concorrência; suites de concorrência/race
  existentes executadas pela matriz, sem necessidade de novo mecanismo.
- assumption-integrity: confiança na coerência do objeto, num manifest corrente
  isolado e na suposta ausência de tags é refutada pelos probes.
- design-causality: leitura do tree e SemVer corrigem as causas correspondentes,
  mas bypasses nos consumidores deixam a garantia incompleta.
- solution-proportionality: reuso da validação histórica e de resultados com
  coverage é a alternativa suficiente; não é necessário adotar nova policy,
  pin de issuer nem camada de assinatura para corrigir estes problemas.

## Checks executados

Ver os resultados estruturados vinculados na evidência compacta. A validação
POSE e os probes são de naturezas diferentes: suite verde não aprova os casos
negativos que a suite não afirma.

- `pose validate --strict --module pose-mcp`: **137/137 pass**, 352,6 s,
  incluindo build, go test ./..., vet, integração B38 e checks race registrados.
  [Resultado](../results/b38r2-independent-validation.json).
- `pose validate --strict --module mcp-enforce`: **3/3 pass**.
  [Resultado](../results/b38r2-mcp-enforce-validation.json).
- `pose validate --strict` global: **140/140 pass**, 232,9 s.
  [Resultado](../results/b38r2-global-validation.json).
- `pose check --strict`: sucesso, 29 warnings históricos.
- `pose artifact-check --spec pose-release-manifest-tag-integrity --strict`:
  sem errors; 641 warnings de proveniência/orphans históricos. Claims e observed
  são separados e constam na evidência; não são prova de suficiência funcional.
- `pose lint-spec … --strict`: sucesso; seção Tasks opcional ausente.
- `pose surface-check --spec … --strict`: **falha**, stale-evidence e
  unconsumed-capability. O arquivo canônico `.pose/results/delivery-validation.json`
  ainda contém outra proveniência. Os resultados desta revisão foram gravados em
  arquivos diagnósticos próprios; não se atualizou o results_path para preparar
  selo/closeout de uma entrega recusada. Esse estado é um gate pendente de
  closeout, não um nono defeito funcional atribuído ao PR.
- `pose assess design`: 21/21 arquivos observados, sem truncamento.
- `pose assess discover --if-stale --component pose-mcp`: refreshed.
- `pose assess integrate`: executado, dois gaps HTTP/MCP sem consumidor observado;
  diagnóstico estático não substitui a revisão direta da delegação MCP.
- `pose assess tech-debt`: um marcador coberto, zero uncovered; **INCOMPLETE**
  por duas specs históricas inválidas, identificadas na evidência.
- `pose recurrence-check --tolerant --window-days 14`: zero flagged.
- `pose followups --all`: consultado, sem disposições automáticas.
- `govulncheck@v1.4.0 ./...` em pose-mcp: nenhuma vulnerabilidade encontrada.
- `gitleaks@v8.21.2 git --log-opts=origin/main..HEAD` com configuração do repo:
  sete commits, nenhum segredo encontrado.
- `git diff --check origin/main...HEAD`: sucesso.
- Reexecução de `.pose/results/b38-mutations.py`: **43/43 mutantes mortos**,
  exit 0. [Resultado](../results/b38r2-mutations.json). São os mutantes existentes;
  não cobrem as oito variantes reproduzidas nesta revisão.
- `go test ./internal/pose ./internal/cli -run TestB38R2 -count=1 -v` em cópia
  temporária: probes concluídos, com as reproduções descritas acima.

Não foi feita operação remota, alteração de tag nem publicação de provider.
O HEAD permanece o revisado; os artefatos desta revisão ficam sem commit.

Os assessments, estados e índices derivados emitidos pelos checks também ficam
sem commit. A diferença prévia de `.pose/indexes/generation.json` não foi
descartada. Nenhum desses derivados altera o lifecycle da spec ou constitui selo.
