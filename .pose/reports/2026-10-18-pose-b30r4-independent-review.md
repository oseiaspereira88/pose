# Revisão independente — B30r4 — PR #197

Data: 2026-10-18. **Decisão: changes requested. Cinco bloqueantes de merge e seis follow-ups aceitáveis.** Spec `pose-spec-facts-evidence-binding`, HEAD `ce83fbd6979ea75111aa8004fa150e2c8b7aa62e`, branch `impl/pose-b30-spec-facts-binding`, repositório `oseiaspereira88/pose`. Parecer independente sobre o diff completo contra `origin/main`, incluindo as correções da b30r3 e suas variantes nos consumidores próximos.

## Autoridade e independência

Revisor `agent:independent-gpt-6.1-sol-review`, execução Codex/OpenAI; implementador Claude/Anthropic, conforme declaração do usuário e coautoria dos commits. Identidade e separação de vendors **declaradas**, sem autenticação criptográfica. Nenhum subagente utilizado. Aplicados skills `pose-review` e `pose-spec-closeout`, workflow `.pose/workflows/review.md`, AGENTS do escopo, POSE.md, template de spec e regras abaixo. **Não houve selo, atestação, envelope, alteração de policy/pin de issuer, close, commit ou push.**

`pose context --task spec:pose-spec-facts-evidence-binding --json` resolveu `xref:proj.pose-dist/spec:pose-spec-facts-evidence-binding`, mesma autoridade, `in-progress`, context revision `7e0524fa492602ce370f8f53b72d0ec01f418ab7704bdf64e354eba3e09d1516`. `git fetch origin main` e `gh pr view 197 --repo oseiaspereira88/pose` confirmaram base, branch e HEAD. Examinados R1–R8/V1–V6/D1, todos os arquivos de produção/testes/documentação do diff, trailers, mutações já registradas e pareceres anteriores. A CLI deste HEAD foi compilada em `/tmp/b30r4-pose`. Probes independentes rodaram em cópia descartável de pose-mcp/mcp-enforce; fontes e saídas estão na [evidência compacta](../results/b30r4-independent-review.json). **PASS de um probe significa defeito reproduzido com controle, não aprovação.**

## Rules applied during review

- Change type: bugfix com mudança de contrato público de facts e documentação.
- Workflow: `.pose/workflows/review.md`.
- `.pose/rules/backend-go.md`: domínio, parsing, propagação de erros, identidade e consumidores Go.
- `.pose/rules/security.md`: integridade dos bindings, campos externos, caminhos, renderização e dependências.
- `.pose/rules/documentation-style.md`: spec, manuais EN/pt-BR root/dist, changelog e parecer.
- `.pose/rules/delivery-evidence.md`: associação, freshness, composição e atribuição da evidência.
- `.pose/rules/knowledge-governance.md`: consulta de handoffs/decision logs de bundles e assurance e dos ADRs de delivery integrity, gerações verificadas e review evidence versus delivery target. Nenhum risco bloqueante foi aceito para merge.
- `.pose/rules/_base-recurrence.md`: classificação por domínio/causa e prevenção das variantes recorrentes.
- Frontend, infraestrutura e outros stacks não se aplicam. Não há tool MCP própria de facts; texto e JSON de `pose specs facts` compartilham a projeção. Integração foi avaliada por tocar helpers usados por outros consumidores, sem alterar REST/Kafka/Protobuf/tools MCP.

## Bloqueantes de merge

### F01 — high — Caminho literal recebe semântica de alias ou perde espaços e associa módulo alheio

**Local:** `pose-mcp/internal/pose/spec_facts.go:449–467, 495–527`. **Requisitos:** R1, R2, R7.

`moduleCoversSpec` passa paths observados de Git ao mesmo `factModuleKind` usado para referências de módulo/componente. Esse helper remove espaços nas extremidades e trata `root` como alias da raiz. Um arquivo real chamado `root`, sem components declarados, torna qualquer módulo pertinente; um path literal ` api/server.go` passa a ser comparado como `api/server.go`, embora esteja em outro diretório. Resultado de `web` no primeiro caso ou de `api` no segundo, com scope digest correto, vira `checks/associated`. O arquivo `root` ainda elimina o limite de cobertura inferida. Ambos foram reproduzidos com **deltas reais de Git**, resolvidos por `resolveGitChangeSet`; não dependem de path impossível ou escape de diretório.

**Varredura da classe:** módulo do resultado, components, paths normais e as duas pontas de rename, `componentDeclares`, `normalizeFactPath`, `moduleMatchesTarget`, `moduleCoverageIsInferred`, `ValidateArtifactPath`, `parseGitNameStatus` e compactação/expansão. Renames com origem `root` ou destino ` api/server.go` repetem o problema. Os helpers vizinhos de cobertura também têm convenção de alias `root` para referências declaradas, mas isso não transforma um arquivo Git literal em referência de componente; a validação de artifact paths conserva espaços e permite um arquivo chamado `root`. A correção anterior para alvo inválido não corrige essa confusão de tipos. A variante conservadora de ID de componente desconhecido permanece L01.

**Ação:** distinguir referência de módulo/componente de path Git literal; conservar os bytes significativos dos paths e aplicar aliases somente ao tipo que os admite. Cobrir espaços nas duas extremidades, arquivo/diretório chamado root, módulos irmãos e renames, preservando o root explicitamente declarado.

### F02 — high — Predicado de resolução ainda promove change sets estruturalmente incompletos

**Local:** `spec_facts.go:255–261, 347`; `pose-mcp/internal/cli/artifact_integrity.go:326–349`; decoder em `delivery_integrity_cache.go:61–85`. **Requisitos:** R1, R2, R7.

Exigir que todos os sets tenham head, algum path e algum elemento em commits fecha o contraexemplo misto da b30r3, mas não comprova que o set usado foi resolvido. Um set sem `ResolvedBase` e sem `DiffDigest`, ou com `Commits: [""]`, ainda é contado como resolvido. Um vetor com um path vazio também satisfaz o contador, emprestando cobertura de components; um rename sem destino também fornece `len(Paths)>0` e cobre a origem. Os probes obtiveram `associated` com digest coincidente e sem limitação sobre o fundamento incompleto. A própria fixture positiva usa um set sem base/diff, por isso a suíte atual não estabelece essa fronteira.

**Varredura da classe:** todos os campos de ChangeSet/ObservedPath, produtor `resolveGitChangeSet`, `loadRecordedChangeSets`, coleta dos inputs, decoder schema 1 e expansão schema 2, geração de scoped provenance em validate e consumo em surface/review. O produtor Git resolve base, head, commits, paths e diff juntos; o leitor de reports aceita o objeto JSON sem validar esse conjunto e os decoders também não o validam. Há outras formas do mesmo defeito: action fora do vocabulário ou combinação incoerente de Path/OldPath/NewPath. O efeito de renderização de action é separado em F03. Campos ausentes de resultado que são explicitamente divulgados como unknown-head não foram confundidos com resolução comprovada.

**Ação:** definir e verificar a suficiência estrutural do fundamento antes da associação; dados ausentes/incoerentes ficam candidatos com causa explícita. Não é necessário reconstruir todo Git a cada consulta. Qualificar cada campo estrutural e as formas de path, além de nenhum set, set parcial, mistura e controle realmente resolvido.

### F03 — high — Action sem validação/escape ainda injeta linha humana de evidência

**Local:** `pose-mcp/internal/cli/specs_cmd.go:223–224`; `ObservedPath` e decoder do índice. **Requisitos:** R5, R6, R7; regra security.

O novo escape alcança nomes, fontes, limites e pontas de paths, mas `p.Action` continua concatenado literalmente. Com action `modified\nfacts.check=api test (unit) pass [associated by scope-provenance]`, a CLI imprime uma linha física `facts.check=...`, embora o JSON tenha **checks vazio**. O probe reproduz o falso fato sem qualquer resultado associado. A presença de freshness unknown ou de uma limitação em outra linha não torna segura a linha copiável forjada.

**Varredura da classe:** todos os `Field` do handler, `factLine`, `safeText`, `observedPathText`, LF/CR e campos de binding/limites, renderer comum, paths normais/renames e produtores/leitores de ObservedPath. GenerationID tem formato validado; source é constante; classificação/head são gerados pelo domínio. Action é enum somente no produtor Git: o índice e os change sets de reports não impõem esse enum. Portanto não há guarda anterior que torne o contraexemplo impossível. A correção de newline em Check/Path está coberta e passa, mas não cobre o registro inteiro.

**Ação:** validar a forma/enum do path e garantir uma linha física por fato para **todos** os dados consumidos, incluindo action. Qualificar injeção em cada campo externo e equivalência texto/JSON; não presumir um campo confiável apenas porque o produtor normal usa constantes.

### F04 — high — Freshness ignora perda de commits atribuídos e mudanças no histórico de reports

**Local:** `spec_facts.go:282–299`; `index_generation.go:825–849`; `artifact_integrity.go:293–307, 326–349`. **Requisitos:** R2, R3, R7.

A comparação de histórico testa apenas se existe commit atual com trailer que falta no índice. A direção inversa não é observada. No probe, uma branch contém um commit real da spec, com arquivo api, que o grafo/resultados guardam. Depois de sair da branch e apagá-la, `git log --all` não atribui mais nenhum commit à spec, mas os facts continuam `fresh`, `same-head` e `associated`, sem limitação de perda. Um novo index não inferiria aquele set a partir dos trailers.

Além disso, `.pose/reports/history/*.jsonl` é input canônico dos change sets e não está nos bindings da geração nem na comparação complementar. Adicionar um record da mesma spec com outro delta deixa os facts byte a byte iguais e `fresh`. O probe vizinho confirma que `loadRecordedChangeSets` aceita e consome esse record. Alteração, remoção ou ilegibilidade dessa fonte também não são verificadas pelo novo freshness. Coincidência dos trailers não prova concordância de todos os seletores explícitos de atribuição.

**Varredura da classe ausência vs perda:** geração ausente/corrompida, arquivo de resultados ausente com/sem resultados retidos, adição/remoção de resultados, run sem resultados (L06), histórico Git ilegível, commits novos e removidos, atribuição por trailers e por reports explícitos, decoder e consumidores de surface/review. Arquivo de resultados perdido com entradas retidas já rebaixa freshness; ausência inicial sem entradas não é essa perda. A simetria falta no eixo de atribuição. Não se pede rerun por commit documental alheio: a mudança relevante é na fonte que atribui o delta desta spec.

**Ação:** comparar o conjunto pertinente de fontes/seletores e seu resultado com o índice, distinguindo ausência inicial, perda, erro e retenção histórica explícita. Se a equivalência não puder ser estabelecida, retornar stale/unknown com a causa. Incluir reports explícitos e alterações nas duas direções sem invalidar reuso material legítimo.

### F05 — high — Grafo e freshness podem pertencer a gerações diferentes

**Local:** `spec_facts.go:94–96, 241–246`; `index_generation.go:340, 390–404, 646–650`. **Requisitos:** R1, R3, R7.

`GetDeliveryIntegrity` lê o grafo por `ReadIndex`, mas `specFactsFreshness` abre outra view por `ReadIndexGeneration`. Não há vínculo entre as duas observações. Uma publicação entre as leituras permite rotular os paths/resultados de A como `fresh` da geração B. Com Store pinado isso é reproduzível sem corrida: pinar A, publicar B, ler facts pelo Store pinado. O resultado conserva api e um check associado de A, informa o ID de B e `fresh`; a leitura atual de B demove o mesmo check a candidato, pois o delta de B é outro.

**Varredura da classe:** `ReadIndex`, `ReadIndexGeneration`, `PinIndexes`, preparação de review plan/bundle e closeout. Esses consumidores vizinhos já fixam a geração na entrada; `ReadIndex` respeita `indexPin`, a nova função de freshness não. O probe usa esse contrato existente para controlar a interleaving que também pode ocorrer na CLI sem pin. O controle com Store atual demonstra a consequência semântica, não somente um ID cosmético diferente.

**Ação:** derivar grafo, identidade e diagnóstico da mesma geração fixada, ou recusar/repetir quando mudarem. Reutilizar o contrato de pin existente é suficiente; não exige ampliar a entrega para a read view B34 nem reconstruir índices. Cobrir publicação entre observações, Store já pinado e controle sem publicação.

## Follow-ups aceitáveis — variantes conservadoras de baixo risco

Todos mantêm ownership proposto `(owner:@core crit:low review:2026-12-01)`. L01/L02 já estão `[open]` na spec; não foram escritos novos follow-ups nela durante esta rodada com bloqueantes. São seis itens residuais, sem aceitar sua variante insegura.

- **L01 — low — ID de componente sem resolução para path.** Um ID legítimo diferente do diretório conserva evidência pertinente como candidata ou dá explicação imprecisa. Aceitável somente a recusa conservadora. Comparar o path errado ou transformar alias/path literal em cobertura alheia é F01.
- **L02 — low — Unavailable não projetado quando o índice falta/corrompe.** A leitura falha antes de construir Freshness; CLI retorna erro facts-unavailable, sem sucesso vazio. Expor structured unavailable conservando a recusa.
- **L03 — low — Mensagem genérica de unknown incorreta.** A limitação diz “not published as a set” também para geração publicada sem bindings ou histórico ilegível. Status/Reason continuam conservadores. Usar a causa observada e alinhar os quatro manuais/changelog.
- **L04 — low — Ordenação com desempate parcial.** Module/Check/ID/Outcome não desempata classe, fonte, severidade ou binding. Inverter dois resultados que diferem só nesses campos muda a ordem do JSON, mas preserva ambos e não escolhe um resultado pelo outro; probe confirma. Comparados também loader, BuildDeliverySurface, compactação e ordenação de bundles/atestados: os últimos usam instantes (`reviewInstantLess`), não texto de timestamp. Tornar o desempate estrutural completo; não confundir ordenação de apresentação com seleção temporal ou deduplicação.
- **L05 — low — Consulta de histórico sem limite de saída.** `SpecTrailerCommits` usa `exec.Output` para todo log --all; a atribuição vizinha limita a saída a 32 MiB. Risco local de memória/latência; não houve teste de estresse gigante. Compartilhar leitura limitada e conservar erro/unknown ao exceder, sem truncar silenciosamente e chamar fresh.
- **L06 — low — Identidade de run vazio é descartada.** ParseDeliveryValidationRun projeta checks com classe e perde o header quando não há entradas. Alterar head/time/scope de um run vazio mantém a projeção vazia. Não produz associated nem aprovação; a ausência de evidência continua explícita. Conservar identidade do run ou esclarecer o alcance da freshness da projeção, sem promover zero checks a entrega comprovada.

## Plano, contratos e julgamento

Bundle somente **prepared**, não selado: `rvb-3f6c2107c1c1b3c0`, digest `sha256:3f6c2107c1c1b3c029286f661e0d84d6e29d6be4df844b5d8abbb1ac2cc50900`. Plano `sha256:b06d9b4f6e01080b02894d037858857f6174aeda5ab97e441c88e5fdb7f5cb56`; patch `sha256:d8904ccd893a96ec104bd4378d5d81efeceb017a1c4a30991f04ef4b3b7dd211`; tree `sha256:b3bc068bf027f34b0e31d0d6a1b3ba2a1da4c64ae89685754e41df8adcb20758`. 15 critérios, 11 ferramentas; componente pose-mcp Go/high, origem declared+observed. Metadata sem owner e governança não mapeada são warnings explícitos.

A range de atribuição contém nove commits com o trailer desta spec e 20 commits de outros trabalhos; o bundle distingue paths atribuídos de range contaminada. Os nove trailers foram conferidos no corpo das mensagens, sem usar o parser de trailers do Git como prova negativa. A spec declara 27 claims, observados 34 paths; artifact-check não registra erro nem finding desta spec. Não há mudança de dependência de produção ou concorrência nova. Checks/candidates/freshness/binding são mudança pública classificada minor; fonte/severidade agora são preservadas. Nenhuma policy, adoção ou issuer foi alterado.

As correções diretas de b30r3 têm testes verdes: alvos inválidos não viram raiz, sets mistos não emprestam resolução, a identidade de resultados usa JSON, fonte/severidade estão no render e nomes/path com LF são escapados. Isso não elimina F01–F05. Scope e proporcionalidade da direção são adequados; correctness, requirements, input/errors, contracts, observability, security, compatibility/documentation e operability não podem receber passed enquanto essas variantes persistirem. Assumption-integrity/design-causality precisam reconhecer que digest coerente não comprova fundamento suficiente nem freshness da observação certa. Critérios mecânicos de integração/concurrency não suprem esse julgamento.

Atalhos adversariais baratos: um path literal que vira alias, set incompleto com contadores não vazios, newline no enum não validado, perda de fonte fora dos bindings e troca de geração entre observações. O corpus `pose-mcp/internal/cli/testdata/adversarial/README.md` ainda não tem caso B30 dessas variantes. Os dez probes independentes anexos são evidência reproduzível para a correção e os controles; nenhum deles foi incorporado à implementação nesta revisão. As mutações anteriores foram inspecionadas, sem rerun integral: não acrescentariam resposta aos contraexemplos novos e a matriz completa já foi executada.

## Checks e disposição de ferramentas

- **validate global --strict: exit 0, 140/140 pass, 210,7s**, sem failed/optional_failed/errored/skipped. Inclui pose-mcp 137/137 e mcp-enforce 3/3, build/test/vet de todo módulo, integração B30 e suites com race registradas. [Resultado estruturado](../results/b30r4-independent-global-validation.json).
- **check --strict: exit 0**, 29 warnings históricos, nenhum erro. **lint-spec --strict: exit 0**, oito requisitos/trace, dois follow-ups open e apenas Tasks opcional ausente.
- **artifact-check --spec --strict: exit 0**, nove commits atribuídos, 27 claims, 34 paths; 644 warnings globais de orphan, nenhum erro/finding da spec.
- **surface-check inicial: exit 1**, 139 resultados antigos da fonte configurada, stale-evidence e unconsumed-capability. **surface-check --results .pose/results/b30r4-independent-global-validation.json --strict: exit 0**, 140 resultados e zero findings. Override só na consulta; nenhuma mudança de policy. A fonte configurada não foi reescrita, pois não haverá selo/closeout com bloqueantes.
- **assess design/discover/integrate/tech-debt: exit 0**. Design observa 17/17 entradas; discover refresca pelo código/matriz deste HEAD. Integração informa 64 integrações, um contrato ativo e 63 gaps de inventário, não 63 defeitos novos desta PR. Debt tem um marcador coberto e `complete:false` por duas specs históricas com follow-up vazio; a cobertura incompleta não foi apresentada como dívida zero.
- **suggest review --path pose-mcp, recurrence-check --tolerant --window-days 14 e followups --all: exit 0**. Recurrence leu 393 registros, zero clusters sinalizados; sem gatilho mecânico de escalation. A recorrência semântica é de contrato/implementação e de qualificação incompleta de classes, com prevenção proposta nos negativos e no corpus, sem mudar policy.
- **govulncheck: exit 0**, nenhuma vulnerabilidade encontrada. Scan direcionado do diff com quatro padrões de segredo: zero hits; gitleaks/golangci-lint indisponíveis, vet coberto pela matriz. Esse scan não equivale a uma varredura completa por gitleaks.
- **probes independentes: exit 0, dez probes**, em cópia descartável; todos os contraexemplos esperados reproduzidos, incluindo controle atual versus Store pinado e deltas reais Git. O código do HEAD permaneceu intacto.

Ferramentas required ativas (artifact-check, surface-check e validate global/módulo) executadas; recommended design/discover/integrate/tech-debt/suggest utilizadas por pertinência aos limites compartilhados. Completion-only closeout-check/review-check e demais passos de selo/atestação/verificação/close permanecem **deferred por bloqueantes**, conforme instrução expressa. Não foi criado handoff de aprovação/risco aceito nem contribuição separada staged/submetida upstream.

## Estado deixado e próximo passo

Parecer e evidências desta rodada ficam locais, sem commit. A modificação preexistente de `.pose/indexes/generation.json` foi snapshotada antes dos comandos; os derivados de assessments/state/index alterados pelos checks foram restaurados ao snapshot. Spec, implementação, policy, branch e HEAD não foram alterados. Corrigir **F01–F05**, qualificar a classe inteira e preservar os seis follow-ups conservadores; somente após revisão sem bloqueantes, atualizar ownership `[open]`, gerar evidência na fonte configurada, selar, atestar **sem envelope** com identidade/separação declaradas, verificar, fechar e enviar o closeout. Não há declaração de entrega concluída.
