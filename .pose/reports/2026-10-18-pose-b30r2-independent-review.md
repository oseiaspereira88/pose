# Revisão independente — B30r2 — PR #197

Data: 2026-10-18. **Decisão: changes requested. Seis bloqueantes de merge e três follow-ups aceitáveis.** Spec `pose-spec-facts-evidence-binding`, HEAD `a42ce4623bbbd5c9e256c10ee0c77697750bb4ce`, branch `impl/pose-b30-spec-facts-binding`, repositório `oseiaspereira88/pose`. Este parecer substitui a decisão anterior para o HEAD revisado; não reescreve a evidência da primeira revisão.

## Autoridade, independência e escopo

Revisor `agent:independent-gpt-6.1-sol-review`, execução Codex/OpenAI; implementador Claude/Anthropic, conforme declaração e trailers dos commits. Identidade e separação entre vendors **declaradas**, sem autenticação criptográfica. Nenhum subagente utilizado. Não foram selados bundle, emitida atestação, alteradas policy/issuer pins, encerrada spec, criados commits ou enviados commits ao branch.

Autoridade local resolvida por `context`: `proj.pose-dist`, spec in-progress, mesma autoridade, revisão `a42ce462`. PR e HEAD conferidos via `gh pr view`; `git fetch origin main` preservou base `8aa729a76ab1c094690a0892263a86bf3bb7bb42`. Inspecionados spec R1–R8/V1–V6/D1, diff de 22 arquivos contra origin/main, commits e seus trailers contíguos, implementação completa, testes, produtores/consumidores vizinhos, índices derivados, relatório e mutações anteriores. O change set atribui cinco commits e 22 paths; um merge sem trailer não foi confundido com implementação não atribuída.

Aplicados skills `pose-review` e `pose-spec-closeout`, workflow review, AGENTS pertinentes, POSE.md, template de spec, ADR de delivery integrity/compactação e knowledge de assurance/historical time. A CLI deste HEAD foi compilada em `/tmp/b30r2-pose`. Probes adicionais rodaram exclusivamente numa cópia descartável de pose-mcp e mcp-enforce. **PASS dos probes significa defeito reproduzido, não comportamento aprovado.** Fontes e saídas estão na [evidência compacta](../results/b30r2-independent-review.json).

## Rules applied during review

- Change type: bugfix com contrato público de facts e documentação.
- Workflow: `.pose/workflows/review.md`.
- `.pose/rules/backend-go.md`: contratos, erros, comparação de caminhos e consumidores Go.
- `.pose/rules/security.md`: integridade de bindings e fonte dos resultados; scan do diff e de dependências.
- `.pose/rules/documentation-style.md`: spec, changelog, manuais root/dist EN/pt-BR e parecer.
- `.pose/rules/delivery-evidence.md`: associação, freshness, composição, atribuição e limites de evidência.
- `.pose/rules/knowledge-governance.md`: consumo de decisões anteriores; não houve aceitação de risco bloqueante nem handoff de aprovação.
- Frontend, infraestrutura e outros stacks: não aplicáveis. Não existe tool MCP própria de facts; o handler CLI publica a mesma estrutura em texto/JSON. Integração executada para conferir os limites compartilhados.

## Bloqueantes de merge

### F01 — high — Comparação parcial permite resultados divergentes e até índice vazio `fresh`

**Local:** `pose-mcp/internal/pose/spec_facts.go:144–193`. **Requisitos:** R1, R3, R7, R8.

`resultsFileDisagrees` compara somente ID/outcome, generated_at e provenance_digest global. `git_head` é lido mas nunca comparado; scope_provenance, module, nome, severity e schema nem participam da comparação; evidence_class apenas filtra presença. Com geração publicada e inputs vinculados, os probes alteram isoladamente git_head, scope_provenance, module, nome, evidence_class ou schema_version=999: todos mantêm `fresh`. O arquivo já não descreve a execução/classe/escopo que os facts apresentam. Se o índice não tem resultados e surge no arquivo um check válido, `len(graph.ValidationResults)>0` impede detectar a adição: zero resultados continuam `fresh`.

Há ainda perda de informação ao reduzir os checks a map ID→outcome, e erro de policy é tratado como comparação bem-sucedida (`return ""`). Ausência de fonte sem resultados pode ser legítima; perda/erro/contrato não suportado ou uma fonte agora não vazia precisam ser distinguidos dessa ausência.

**Vizinhos examinados antes do achado:** produtor `cli/validate.go`, `validationRunResult`, loader `cli/surface_check.go:55–85`, `LoadDeliveryPolicy`, decoder de schema 1/2, compactValidationRun e expansão, `deliveryEvidenceCurrent`. O loader exige schema válido e transporta head, escopo, módulo, nome, classe, severidade e fonte; a compactação distingue runs pelos campos completos. A nova segunda leitura compara menos do que o produtor/loader consome. Não é necessário reconstruir o índice em cada consulta nem mudar policy.

**Ação:** comparar a projeção completa e validada que o loader realmente indexa, preservando multiplicidade/identidade da execução e distinguindo ausência de falha. Novo resultado num índice vazio é divergência. Campo não verificável deve baixar freshness para unknown/stale, não certificar fresh. Cobrir todas as variantes enumeradas, incluindo duplicatas e policy ilegível/inválida.

### F02 — high — Digest do escopo vazio é promovido a associação

**Local:** `spec_facts.go:278–301`; `delivery_surface.go:567–589`. **Requisitos:** R1, R2, R7.

`ScopedDeliveryProvenanceDigest` sempre produz um SHA-256, inclusive quando não há claim nem change set da spec. Com apenas `components: api`, um resultado de api portando o hash desse vazio entra em `checks/associated`, sem limite individual e com `unknown-head`. O probe remove todos os change sets e claims e obtém essa associação. O aviso global de ausência de paths não rebaixa o resultado: componente declarado e digest reproduzível de nada continuam apresentados como evidência associada. O mesmo teste de igualdade não estabelece resolução de um change set que esteja incompleto.

**Vizinhos:** produtor de validate só acrescenta scopes que encontra em change sets; ScopedDeliveryProvenanceDigest distingue conteúdo por hash, mas não certifica que esse conteúdo existe ou é resolvido. Artifact reconciliation e surface mantêm observação de Git separada da declaração; o graph decoder aceita o registro sintético, portanto não há validação anterior que assegure essa premissa ao leitor de facts.

**Ação:** exigir fundamento observável suficiente para o binding de delta antes de associar; escopo inexistente/não resolvido permanece candidato com limite explícito. Não inferir validade semântica de components nem da igualdade de hashes do vazio. Testar vazio, claims sem observação e change set não resolvido, além do controle real resolvido.

### F03 — high — Módulo ausente vira raiz do repositório

**Local:** `spec_facts.go:298–301/393–438`. **Requisitos:** R1, R2, R7.

`normalizeFactPath("")` vira raiz e `moduleCoversSpec` aceita raiz como cobertura universal. Um resultado sem módulo, com scoped digest correto de api, é associado a api sem limite de cobertura. O probe reproduz `module:""`, `associated`, `limits:[]`. Um campo omitido não demonstra execução registrada na raiz. Normalização também trata `/` como raiz relativa: falta distinguir path inválido/absoluto de referência explícita e válida à raiz.

**Vizinhos:** `moduleMatchesTarget` e `moduleCoverageIsInferred` também dão tratamento universal a vazio/root; portanto há pressuposto compartilhado, não proteção no consumidor seguinte. O loader transporta o módulo sem exigir presença; validate produz diretório explícito (`.` para raiz). `ValidateArtifactPath`/confinamento em produtores vizinhos distingue paths inválidos de paths relativos. Foram conferidos igualdade, ancestor, descendant, prefixo, `./api`, `api/../web`, raiz explícita e ausência.

**Ação:** separar módulo desconhecido/inválido de raiz declarada antes de limpar paths; manter desconhecido como candidato com limite. Preservar o controle positivo de raiz explícita e os testes de direção da cobertura. Auditar os helpers compartilhados sem ampliar este delta para uma nova policy.

### F04 — high — A deduplicação ainda perde resultados e escolhe classificação pela ordem

**Local:** `spec_facts.go:288–297`. **Requisitos:** R1, R2, R4, R7.

A identidade inclui ID/module/check/outcome/head/report, mas omite scope_provenance, provenance_digest, generated_at e evidence_class. Duas entradas iguais nesses seis campos e distintas no scoped digest colapsam; se a primeira tem binding válido, resta só `associated`; invertendo a ordem, resta só `candidate`. Os probes reproduzem as duas classificações e a perda de uma execução quando só generated_at, evidence_class ou provenance_digest diferem. Reexecuções do mesmo check no mesmo head naturalmente podem compartilhar ID e diferir no instante/binding. Não basta a correção anterior que preserva pass/fail com IDs diferentes.

**Vizinhos:** compactValidationRunID usa serialização de todos os campos da execução; compactação/expansão preserva a diferença entre runs. BuildDeliverySurface conserva entradas e ordena IDs; não certifica unicidade antes deste consumidor. Foram examinados também os maps por ID de F01, ordenação por module/check/ID/outcome aqui, ordenação por ID no loader/grafo e leitura de reviews por `reviewInstantLess`. Não há neste delta seleção cronológica de review a corrigir; o defeito é deduplicação por identidade incompleta, não exigir datas em ordem lexical.

**Ação:** remover somente duplicatas realmente equivalentes ou rejeitar explicitamente colisões inconsistentes; nunca escolher uma entrada silenciosamente. Preservar source/binding/classe de cada resultado e uma ordem determinística com desempate completo, sem interpretar ordem textual de timestamps como temporal. Testar cada campo que participa da evidência, nos dois sentidos de entrada.

### F05 — high — Freshness usa semântica de trailer/histórico diferente da atribuição

**Local:** `spec_facts.go:198–208/251–257`. **Requisitos:** R3, R7, R8.

A busca exige exatamente `POSE-Spec: alpha`, sem whitespace adicional e só no histórico de HEAD. A atribuição canônica (`allCommitsWithSpecTrailers`) usa `--all`, lê o corpo e aplica TrimSpace à linha e ao slug. Probes adicionam um commit com `POSE-Spec:  alpha` e, separadamente, um commit com trailer canônico em outra ref: ambos ficam fora do índice, mas facts permanece `fresh`. Ambos são entradas reconhecidas pelo produtor canônico. Não se trata de formato de trailer Git nem de pin/assinatura.

Além disso, falha de leitura do Git só rebaixa freshness se uma segunda consulta conseguir obter HEAD. Sem repositório/HEAD legível, as fixtures publicadas continuam fresh embora o eixo histórico não tenha sido verificado; falta divulgar esse limite em vez de usar ausência de Git para certificar o índice.

**Vizinhos:** `cli/artifact_integrity.go:147–173` (`--all`, TrimSpace, saída limitada), resolved change sets, `gitHeadAtRoot`, IndexInputNames e StaleInputs. Os bindings da geração não incluem histórico Git; o novo consumidor precisa observar o mesmo universo da atribuição ou explicitar unknown. A solução não requer rerun de check em commit documental alheio: só commits realmente atribuíveis devem alterar esse eixo.

**Ação:** compartilhar semântica de extração e universo de refs com a atribuição canônica; erro/indisponibilidade de histórico deve produzir unknown/limitation. Qualificar espaços, tabs, múltiplos trailers, outra ref e falha de Git, sem substituir o parser por `git %(trailers)`.

### F06 — high — Renames colapsam, perdem caminhos humanos e distorcem pertinência de checks

**Local:** `spec_facts.go:239–243/271/424–425`; `cli/specs_cmd.go:226–227`. **Requisitos:** R1, R6, R7.

A chave de path é action+Path, mas um ObservedPath de rename usa OldPath/NewPath e deixa Path vazio. Dois renames viram um só. Com renames em api e web, o probe retém só api e rebaixa o check **fail** de web a candidato, embora o grafo original atribua a alteração a web. A ordenação por Path vazio não resolve a colisão. Num rename único api→web, a cobertura considera somente NewPath e exclui o check de api, omitindo o lado removido da mudança. O texto humano imprime `facts.path=renamed: `, enquanto JSON ainda carrega OldPath/NewPath do rename sobrevivente.

A chave e o render de Path já existiam antes da B30; nesta revisão o impacto relevante é também novo: `moduleCoversSpec` passa a usar a lista reduzida para classificar pertinência da evidência. Não se está exigindo refatoração alheia ao delta. A variante textual foi reproduzida no handler CLI em cópia descartável.

**Vizinhos:** produtor de Git em `cli/artifact_integrity.go:231/259`, helpers `observedKey`, `observedPaths`, `firstObservedPath`, `firstObservedDeliveryPath`, claimKeyNoSpec, matchingDeliveryRoot e compactação de change sets. `observedKey` inclui ambos os paths; `observedPaths` conserva as duas extremidades. O helper de surface que escolhe o destino serve a composição de entrega, não é representação completa dos artefatos afetados para todos os checks da spec.

**Ação:** usar identidade completa de ObservedPath para deduplicação e ordenação, mostrar old→new em texto e considerar os dois lados pertinentes de um rename na associação dos resultados. Cobrir múltiplos renames, entre módulos, controle sem components e paridade de texto/JSON; não promover fonte omitida a ausência de delta.

## Follow-ups aceitáveis — apenas variantes conservadoras de baixo risco

- **L01 — low:** permanece a distinção entre ID de componente e path. A comparação de aliases de path (`./api`) foi corrigida; a variante residual em que um ID não resolve para o diretório real só torna a explicação/associação conservadora (candidato), sem promover resultado alheio. Resolver IDs pela metadata antes de explicar cobertura. Variantes de promoção indevida pertencem a F03, não são aceitas aqui. `(owner:@core crit:low review:2026-12-01)`
- **L02 — low:** índice ausente/corrompido ainda retorna erro antes de projetar Freshness unavailable. O handler devolve falha `facts-unavailable`, sem sucesso vazio nem aprovação. Expor o estado estruturado conservando a recusa, ou documentar precisamente o contrato. Conferidos GetDeliveryIntegrity, ReadIndex, handler e teste de índice danificado. `(owner:@core crit:low review:2026-12-01)`
- **L03 — low:** a limitação de status unknown diz sempre “not published as a set”, inclusive para geração publicada sem bindings ou histórico desconhecido. O status e Reason estruturados continuam conservadores. Renderizar a causa real; não mudar unknown para fresh. Conferidos specFactsFreshness, degradeFreshness e texto/JSON. `(owner:@core crit:low review:2026-12-01)`

L01/L02 já estão [open] na spec. Não se adicionaram disposições/ownership nem handoff de aceitação durante uma revisão com bloqueantes. Numa futura aprovação L03 pode ficar [open] com o ownership acima.

## Plano, contratos e julgamento

Bundle somente **prepared**, não selado: `rvb-bb48a43ed030af28`, digest `sha256:bb48a43ed030af28aaeae9d1da476145bbd6114bc0d45bff02e5cdd0d78d7ee5`. Plano `sha256:0b4f710caf36eed39137a7f7c7ab0479249362c8786cabbc26e40f930206c9b0`; patch `sha256:b84baed7188b8dbb66fb3943ecfd02f72aa4ddc0cce3c2d200a4c2c6538ded4e`; tree `sha256:5c0cdfaa9d1b6a1178cd0402f338fd2af0436aeb852afa50c51a78dc656e69a7`. 15 critérios, 11 ferramentas; componente pose-mcp, Go, high criticality, declared+observed. Warnings de owner ausente e governança sem componente mapeado não representam prova de assurance adicional.

Critérios: scope e proporcionalidade da solução são adequados; não há judgment/approval automático, dependência de produção nova ou concorrência nova. A estratégia de separar buckets e expor freshness é causalmente adequada, mas a premissa de binding demonstrado é falsificada por F01–F06. Correctness, requirements, input/errors, contracts, observability, compatibility/documentation, operability e security não permitem aprovar a implementação atual. Assumption-integrity/design-causality devem explicitar os limites encontrados. Concurrency não exige race adicional por este delta; a matriz executa seus testes registrados com race. Integração mecânica não substitui esses julgamentos.

Contrato público: `checks` muda de semântica e `candidates`/freshness/binding são aditivos; o changelog classifica como minor. Não foram alterados REST, Kafka, Protobuf ou tools MCP, policy, issuer ou contratos selados. A inconsistência de observação em renames e o possível excesso de associação/freshness impedem validar a compatibilidade semântica anunciada.

Atalho adversarial barato: informar scoped digest do vazio, omitir módulo, fornecer índice/arquivo de resultados divergentes nos campos não comparados ou ordenar primeiro a duplicata com binding válido. A igualdade de strings ainda basta para algumas associações sem fundamento. O corpus `internal/cli/testdata/adversarial/README.md` não contém caso B30 que impeça esses atalhos. A suíte/mutações cobre a forma corrigida dos achados anteriores, mas não os contraexemplos independentes desta rodada. Registrar os negativos antes de solicitar nova revisão.

## Checks e disposição de ferramentas

| Ferramenta / check | Resultado e disposição |
|---|---|
| validate global --strict | exit 0; **140/140 pass**, zero fail/optional_failed/errored/skipped; [resultado estruturado](../results/b30r2-independent-global-validation.json) |
| validate / pose-mcp | coberto pelo mesmo run global: **137/137 pass**, incluindo go build ./..., go test ./..., go vet ./..., integração B30 e suites registradas com race |
| check --strict | exit 0; 29 warnings de governança existente, nenhum erro |
| lint-spec --strict | exit 0; oito requisitos/trace, dois follow-ups open; apenas Tasks opcional ausente |
| artifact-check --spec --strict | exit 0; 17 claims, cinco commits atribuídos, 22 paths observados; zero finding da spec e zero erro; warnings globais de orphan não atribuídos ao delta |
| surface-check inicial | exit 1; stale-evidence/unconsumed-capability porque a fonte configurada carregava evidência anterior |
| surface-check --results b30r2-independent-global-validation.json --strict | exit 0; 140 resultados atuais, **zero findings**; apenas override da consulta, sem alteração de policy |
| assess-design / discover if-stale / integrate / tech-debt | executados; design observa 11/11 arquivos do subject; byproducts restaurados |
| suggest-review / pose-mcp | executado; regras e rationale explicitados acima |
| recurrence-check --tolerant --window-days 14 | pass, zero recorrências sinalizadas |
| govulncheck | exit 0; nenhuma vulnerabilidade encontrada; nenhuma dependência de produção alterada |
| scan do diff | quatro padrões de segredos, zero hits; gitleaks/golangci-lint não instalados; vet executado pela matriz |
| probes independentes | exit 0 na cópia descartável, reproduzindo os defeitos de F01–F06; fontes completas na evidência compacta |

Não foi repetida validação após passar: o run global cobre integralmente o módulo. O segundo surface-check usa esse run para verificar composição mecânica, sem declarar que os contraexemplos semânticos foram resolvidos. O comando initial de validate com saída em /tmp foi recusado antes de executar checks (saída deve ficar no projeto); o run válido acima usou .pose/results/. Nenhuma limitação ambiental impediu checks nesta rodada.

Ferramentas de completion (`review-check`, `closeout-check`, auto-attest, verify e close) ficam **deferred por bloqueantes**, por instrução expressa do usuário. Sem selo/atestação de reprovação. A geração.json já estava modificada ao início e foi preservada; byproducts próprios dos assessments e índices foram restaurados. Restam apenas relatório/evidências desta rodada além da modificação preexistente.

Recurrence-check: pass, 393 registros, zero clusters sinalizados; não há gatilho de escalation. Tech-debt declara cobertura parcial por dois follow-ups históricos sem descrição e um marcador coberto, sem atribuir dívida preexistente ao delta. Discover/design e integração foram executados e suas saídas preservadas de forma compacta. Não houve staging nem envio de feedback upstream; os achados ficam neste parecer, sem contribuição separada não autorizada.

## Próxima revisão

Corrigir F01–F06, acrescentar negativos de toda a classe e rerodar validação pertinente. O relatório não declara entrega concluída. Após correções, gerar evidência nova antes de selar; só então atestar sem envelope com identidade/separação declaradas, verificar e fechar conforme autorizado pelo usuário.
