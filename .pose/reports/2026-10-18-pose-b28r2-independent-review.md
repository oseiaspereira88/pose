# Revisão independente b28r2 — PR #196

- Decisão: **changes-requested**. Quatro bloqueantes de merge, relacionados abaixo, e seis follow-ups aceitáveis.
- Repositório: `oseiaspereira88/pose`; spec `pose-distributed-instructions-parity`.
- HEAD inspecionado: `f22be495c10a5aaf1780bf42da967acde937f0ca`; diff completo `origin/main...HEAD`, incluindo correções após d7ebc300, testes, documentos, mirrors, resultados e índices.
- Revisor: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI, execução independente, vendor diferente do implementador Claude/Anthropic. Identidade e separação **declaradas**, sem assinatura/envelope. Este parecer não é uma atestação.
- Evidência compacta: [b28r2-independent-review.json](../results/b28r2-independent-review.json).
- Não houve seal, attest, close, commit, push, mudança de policy ou pin de issuer. A spec continua in-progress. O bundle foi somente preparado/inspecionado.

## Rules applied during review

- Tipo: mixed, correção de descoberta e instruções distribuídas, com adições de contrato JSON e testes.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`; workflow `.pose/workflows/review.md`; manual `POSE.md`; spec e template de spec.
- `.pose/rules/backend-go.md`: erros, disponibilidade, contratos CLI/MCP e testes Go.
- `.pose/rules/security.md`: identificadores, caminhos, confidencialidade e fronteiras de autoridade.
- `.pose/rules/documentation-style.md`: fontes EN/pt-BR, mensagens e mirrors.
- `.pose/rules/delivery-evidence.md`: distinguir testes verdes de comprovação semântica e cobertura completa.
- `.pose/rules/knowledge-governance.md`: consulta a conhecimento anterior; nenhum risco foi formalmente aceito para merge, portanto nenhum handoff de aceitação foi criado neste ramo do pedido.
- Consultados o parecer d7ebc300 e o anterior, a nota `2026-09-12-note-cli-output-design-taxonomy` e os resultados da busca por suggest/B28 nos handoffs/notas. React, Kubernetes e infraestrutura não sofreram implementação neste delta; suas regras de runtime não se aplicam.

## Plano e identidade do sujeito

Contexto resolve a autoridade local `proj.pose-dist`, sem redirect, ambiguidade ou transferência. Plano efetivo `spec-closeout@1` com backend-review, engineering-judgment, high-criticality-review e structural-materiality: 15 critérios e 12 ferramentas; band critical, different-actor. Proveniência de componente pose-mcp é declared+observed; owner ausente e paths de governança não mapeados continuam warnings explícitos.

- Plan digest: `sha256:916f1610eaabb566f58c1c7b2ed9001d03784d563de0f7a2851cb35946bd3aae`.
- Bundle **prepared**, não selado: `rvb-9e506118538fb200`, digest `sha256:9e506118538fb2009e461b9414f03260cb2998c326290abcb5427d6652f07535`.
- Patch digest: `sha256:b5e9db60c5271b15ca9e6af493909eff46c0f529b60b00a5271e9fb8e52ca796`.
- Tree digest: `sha256:47ff3a5bbf9681b434d4d7c0e50411ba4d5eb2bbea401c29c1cc6ac79ccdfa57`.

Ferramentas required ativas: artifact-check, skills-check, validate e surface-check executadas; ferramentas de conclusão review-check/closeout-check ficam deliberadamente deferred, porque o usuário proíbe atestar com bloqueantes. Recomendadas suggest-review, assess-design, assess-discover (--if-stale), assess-integrate e assess-tech-debt executadas. Avaliação dos critérios e limites de cada produtor aparecem abaixo.

## Bloqueantes de merge — lista consolidada

### B1 — Task e listas null ainda apagam a configuração com sucesso

- Severidade: **medium**; classificação: **bloqueante de merge**; R4/R6/R8; residual do B4 anterior.
- Local: `pose-mcp/internal/cli/suggest.go:73`, `:176`, `:199` e `:205`.
- Reprodução real: `{"tasks":{"feature":null}}` é aceito, exit 0, rules_status vazio e workflow/skill ausentes. Com `rules:null` e `rules_by_domain.backend-go:null`, o domínio é anunciado com sucesso e rules_effective:null, sem diagnóstico da configuração. A sonda com lista null usou repo-map válido e stderr vazio.
- Causa: presença da chave da task é confundida com objeto válido; nil é aceito por strictStringSlice, e rules_by_domain:null é tratado como ausência. A validação nova cobre strings/números/arrays errados, mas conserva estes caminhos silenciosos. Uma lista explicitamente vazia pode ser válida; uma task null não fornece trilha e um campo configurado null precisa de tratamento explícito, sem perda silenciosa.
- Ação: validar o objeto da task e distinguir campo ausente, null e lista vazia segundo contrato explícito; recusar configuração inválida antes de publicar a trilha. Cobrir base, mapa por domínio e valor de domínio, em humano/JSON/MCP.
- Consumidores vizinhos: checkTaskMap também aceita task null via struct zero e não valida rules_by_domain; Store.Suggest e handler MCP delegam e propagam sucesso. GetRule valida nome, portanto não remedia uma regra que já desapareceu da lista. Não se pede nova policy.

### B2 — Repo-map com shape inválido vira ausência ou módulo raiz

- Severidade: **medium**; classificação: **bloqueante de merge**; R3/R6/R7/R8; residual do B6 anterior e borda nova do fallback raiz.
- Local: `pose-mcp/internal/cli/suggest.go:251–278`, `normalizeModulePath` em `:292`.
- Reprodução real: `services:"lost"` retorna exit 0, domain_source=repo-map-no-match, sem erro. Item com path numérico e domain=backend-go aplica regras Go a `svc/main.py` como se fosse módulo raiz; domain numérico/language booleano vira no-match. Elemento numérico no array também é descartado sem diagnóstico.
- Causa: Unmarshal só valida sintaxe; assertions de coleção/item/path/domain/language ignoram erros. Path inválido é convertido a string vazia; a correção de raiz agora permite que esse valor case com qualquer caminho. NormalizeModulePath também converte `/` a raiz relativa, embora seja caminho absoluto.
- Ação: validar o documento e seus itens/campos antes da seleção; preservar o caminho raiz válido produzido pelo index e recusar/diagnosticar paths absolutos, externos e tipos inválidos. Não converter perda em no-match; manter controles de fronteira e de módulo mais específico.
- Consumidores vizinhos: loadReviewRepoEntries usa decode tipado e valida caminhos, recusando os tipos errados em vez de sugerir trilha parcial; ReadIndex garante integridade da geração, mas não o schema do documento legado. Conferidos apps/services/packages, campos de identidade, produtor scanModules, Store.Suggest e normalização/confinamento do argumento CLI. Esta sonda não leu conteúdo fora do projeto.

### B3 — Links quebrados de regra ou raiz ainda são confundidos com ausência

- Severidade: **medium**; classificação: **bloqueante de merge**; R4/R6/R7; residual do B3 anterior.
- Local: `pose-mcp/internal/cli/rule_extension_resolver.go:136–145`.
- Reprodução real: backend-go.md como symlink para alvo inexistente é extension-needed, sem detail. `.pose/rules` como symlink para diretório perdido faz security virar unavailable e backend-go virar extension-needed; sai 0 e não divulga a perda da raiz.
- Causa: Stat segue o link e ENOENT tanto no arquivo quanto na raiz é interpretado como inexistência real. A entrada/configuração existe, mas o destino perdeu-se. A checagem adicional de parent repete a mesma ambiguidade.
- Ação: distinguir ausência real de entrada quebrada com Lstat/inspeção dos pais apropriada; divulgar indisponibilidade/perda e não recomendar instalação como remédio de acesso/configuração quebrados. Preservar suporte a links legíveis.
- Consumidores vizinhos: suggestRuleStatuses mantém a distinção de erro quando o helper a fornece; GetRule falha ao ler o mesmo link. ruleExtensionInstalled/doctor ainda colapsam qualquer estado diferente de readable em “não instalado” (F5). A busca cobriu regra, raiz e parent, não apenas o arquivo final. InventoryLegacy tem supressões históricas de erros de policy/supplements/transfers fora da correção textual R5; não se alega que a b28 as resolveu.

### B4 — Inventário instalado ainda aprova perda de cobertura; AGENTS perdido faz skip

- Severidade: **medium**; classificação: **bloqueante de merge**; R1/R2/R8; residual do B7 anterior.
- Local: `pose-mcp/internal/cli/distributed_instructions_parity_test.go:44–45` e `:225–243`.
- Reprodução real em cópia isolada: após freshInstall, inserir workflow `bad.md` contendo `pose assess discover` sob diretório 0000 faz o teste TestAFreshInstallReceivesInstructionsThatAskForDiscoveryOnlyWhenStale **PASS**. O controle idêntico com 0755 **FAIL** na instrução. O código do inventário permaneceu idêntico; a sonda só preparou o arquivo/permissão antes dele.
- Segundo controle da mesma classe: remover AGENTS.md da cópia faz os testes de instruções distribuídas e de mirrors saírem 0 com **SKIP**, embora a raiz esperada tenha sido perdida.
- Causa: o novo instructionMarkdown propaga erros na varredura das fontes, mas o teste de instalação mantém WalkDir/ReadFile ignorados; limiar de dez arquivos mascara perdas parciais. A checagem inicial de AGENTS continua convertendo erro/ausência em skip.
- Ação: aplicar inventário que falha em perdas também ao produto instalado e exigir raízes esperadas no checkout de distribuição; distinguir explicitamente um contexto legítimo sem fontes de um checkout incompleto. Cobrir arquivo ilegível, diretório ilegível e raiz perdida.
- Consumidores vizinhos: conferidos ambos os walkers, todos os ReadFile e guards de raiz; o teste dos quatro mirrors já lê com erro fatal. TestInventoryNeverSkipsWhatItCannotRead cobre apenas o helper das fontes, não o walker da instalação. As jornadas reais EN/pt-BR passaram, mas não impedem este falso positivo.

## Follow-ups aceitáveis — sem caminho para aprovação indevida

São reservas de baixo risco, não autorização de merge com B1–B4. Os quatro primeiros já estão [open] na spec. Com bloqueantes, não alterei spec nem criei aceitação/handoff. Ownership proposto para todos: `(owner:@core crit:low review:2026-12-01)`.

- **F1, low** — Saída humana não mostra domínio/alias/source explícitos quando não há --path (`suggest.go:139`). JSON divulga esses campos; regras e disponibilidade coincidem. Mostrar a mesma explicação no texto. Já [open].
- **F2, low** — Store.Suggest valida domínio antes de normalizar, recusando `Go` aceito pela CLI (`internal/pose/cli.go:34`). A recusa é conservadora, sem trilha parcialmente aprovada. Normalizar alias antes da validação, mantendo confinamento. Já [open].
- **F3, low** — InventoryLegacy ainda informa “no apply exists in 6.x” (`internal/pose/legacy_inventory.go:49`), também exibido no humano. Help e recusa real de --apply dizem o fato atual; nada passa a executar apply. Remover referência de versão. Já [open].
- **F4, low** — b14-mutations.py/b15-mutations.py conservam shell interpolado/heurística de substrings; a correção do executor b28 não se propagou a eles. Não usar esses produtores como prova nova; migrá-los para argv/cwd e eventos de teste. Já [open], sem atribuir resolução dessas unidades à b28.
- **F5, low** — Doctor oferece instalação para regra ilegível/diretório, apesar do estado unreadable do helper (`rule_extension_resolver.go:122`; `doctor.go:564`). Sonda com backend-go.md diretório resulta em rules.stack-extension-available=next e comando install; suggest divulga unreadable. Propagar estado/detalhe à recomendação. É diagnóstico advisory, sem instalar nada ou emitir aprovação, por isso aceita follow-up.
- **F6, low** — `git diff --check origin/main...HEAD` sai 2 por linha vazia adicional no EOF do parecer d7ebc300, linha 173. Corrigir higiene desse artefato quando editado; não afeta o julgamento nem a execução dos gates.

## Varredura por classe antes da classificação

- **Ordenação por texto:** sort de tipos em suggest e de details/unknown keys em migrate serve à apresentação; não decide cronologia/autoridade. Seleção de módulo usa maior especificidade e fronteira; empates conservam primeira entrada, sem nova ordenação temporal. Catálogo invertido conferido contra todos os manifests de extensões de regra: 13 filenames distintos, incluindo Kubernetes. Nenhum novo achado de ordenação temporal neste delta.
- **Ausência versus perda:** B1/B2/B3/B4 e F5; conferidos task-map, repo-map/ReadIndex, rule state/doctor/GetRule, walkers fonte/instalação e inventário migrate. Ausência simples continua caso válido; perda/configuração inválida não pode anunciar completude.
- **Campo sem validação:** B1/B2; cobertos task, rules, rules_by_domain, lista por domínio, arrays apps/services/packages, item, path/domain/language e consumidores checkTaskMap/loadReviewRepoEntries. Identificadores de regra agora usam ValidateName, resolvendo a travessia anterior; Store.GetRule usa a mesma identidade.
- **Comparação de caminhos:** raiz válida, `./`, limpeza, prefixo com fronteira e módulo mais específico têm testes verdes. A ampliação de match a raiz exige validar o path antes da normalização (B2). Links quebrados exigem preservar o estado da entrada (B3). O adaptador MCP recusa paths conservadoramente; nenhuma leitura de dados privados foi usada nas sondas.
- **Integridade de índices:** os sete blobs de HEAD batem com hash e tamanho do descritor commitado; index --verify em checkout limpo passa. O descritor já modificado no início deste turno foi preservado; sua divergência no working tree não é imputada à PR.

## Checks e disposições

Evidência final dos checks: ver JSON vinculado. O executável em PATH mostrou semântica anterior de suggest; os checks conclusivos e sondas foram executados com binário construído deste HEAD em `/tmp/b28r2-review/pose`. Nenhum binário instalado foi substituído. Assessments que escrevem estado foram preservados como evidência e suas escritas nesta instância foram desfeitas; discover/artifact-check foram executados em checkout Git isolado.

- `pose validate --strict`: **exit 0, 140/140 pass**, sem skipped/errored/failed; duração 283,3s. Inclui `go test ./...`, `go build ./...`, `go vet ./...`, integração CLI/MCP, jornadas, segurança de contratos e checks com race registrados na matriz.
- Testes focados de suggest/instruções/mirrors/install/migrate: **pass**, execução independente com `-count=1`.
- `python3 -I .pose/results/b28-mutations.py`: **23/23 killed**, exit 0. Controles do classificador: erro de shell e build viram error; skip vira survived. As mutações não substituem as sondas negativas adicionais B1–B4.
- `pose check --strict`: **exit 0**, 29 warnings (governança histórica/adoption/fragmentos/completude). `skills-check --strict`: 22 skills, zero erros/warnings. `lint-spec ... --strict`: exit 0, oito requirements/trace válidos e quatro follow-ups open; aviso de seção Tasks opcional.
- `artifact-check --spec ... --strict` em checkout Git isolado: **exit 0**, 30 claims, 36 paths observados, sete commits atribuídos pelo trailer; 641 findings do grafo global, nenhum da spec em revisão. O range contém outros trabalhos; somente commits/paths atribuídos definem a revisão.
- `surface-check --spec ... --results .pose/results/b28r2-validation.tmp.json --strict`: **exit 0**, zero findings, capability em pose-mcp com entrypoint suggest.go e 140 resultados atuais. O arquivo foi passado explicitamente, sem modificar delivery.json/results_path; a evidência compacta retém proveniência, nomes e resultados. O resultado bruto temporário foi retirado do worktree após a consulta e preservado em `/tmp/b28r2-review/validation.json`.
- `index --verify` no HEAD limpo: **exit 0**, geração gen-333c4389547a4b43. Comparação direta dos sete blobs confirma hash/tamanho. No worktree original: exit 1, exclusivamente porque o descritor já estava modificado antes da revisão; preservado e separado da conclusão sobre a PR.
- `assess design`: 24/24 arquivos observados, três deltas (metadado de entrega e duas instruções governadas). `assess discover --if-stale --component pose-mcp`: exit 0 em checkout isolado.
- `assess integrate`: 64 contratos, 63 consumidores não observados estaticamente; isso não comprova 63 defeitos. Leitores e adaptador foram inspecionados diretamente. `assess tech-debt`: um TODO coberto, zero uncovered; **complete=false**, com duas specs históricas inválidas divulgadas. Não se declara ausência global de dívida.
- `recurrence-check --tolerant --window-days 14`: 393 registros, zero flags; sem escalação de recorrência.
- Install **e update reais**, EN e pt-BR: exit 0, 36 arquivos Markdown por instância, zero discovery incondicional, hashes das policies preservados. Sem instalação de extensão ou adoção incidental.
- Scan offline das 7.264 linhas adicionadas por padrões de secrets: zero candidatos; sem novas dependências. govulncheck, gitleaks e golangci-lint indisponíveis; não se alega execução desses scanners. `git diff --check origin/main...HEAD`: exit 2 por F6; os artefatos desta revisão não adicionam esse erro.

## Critérios, contratos e decisão

- assumption-integrity/design-causality: D1 é proporcional ao reaproveitar freshness, aliases e catálogo; não demonstra disponibilidade completa enquanto null, schema inválido e inventário parcial forem tratados como ausência/sucesso (B1–B4).
- correctness/requirements/backend-input-errors/backend-observability/operability: requerem remediar B1–B4; um check verde sobre o happy path não demonstra esses casos negativos.
- backend-contracts/backend-integration-impact/compatibility: aliases/status são aditivos; recusa de apply mantém contrato. Store.Suggest/MCP propagam configuração silenciosamente perdida e mantêm recusa conservadora de caixa (B1/B2/F2). Não houve nova dependência ou contrato REST/evento.
- documentation: fontes e mirrors alterados conferem, jornadas reais passam; garantia de inventário completo ainda excede o produtor B4. F1/F3 são explicações residuais.
- backend-concurrency: sem nova concorrência de produção; checks com race são os exigidos pela matriz, não justificativa para ampliar escopo.
- security: ValidateName resolve travessia de identificador; schema/path de índice e links continuam limites de integridade relevantes (B2/B3). Sem achado de segredo nas adições examinadas.
- scope: atribuição por POSE-Spec e artifact-check conferidas; índices commitados íntegros. Não interpretar range multisspec como atribuição de todos os commits ao mesmo slug.
- solution-proportionality: corrigir localmente leitores e reutilizar inventário estrito; não mudar policy, assinatura, issuer nem criar novo scheduler.

Atalho adversarial mais barato: perder uma configuração de regras como null/schema inválido ou esconder uma instrução sob diretório ilegível. As sondas comprovam trilha vazia/sucesso ou teste verde sem cobertura. O corpus `pose-mcp/internal/cli/testdata/adversarial/README.md` não impede esses casos específicos por si só; precisam de regressões nos leitores e inventários envolvidos. As 23 mutações atuais não demonstram as bordas que não exercitam.

Contributor Mode: oportunidades adjacentes foram registradas neste parecer; nenhuma contribuição foi staged/submetida, porque o usuário decide esse passo. Recurrence-check não atingiu threshold. Não há necessidade de mudar policy nem pin de issuer para a remediação.

Corrigir B1–B4 e repetir a revisão no novo HEAD. O caminho autorizado deste resultado termina em parecer/evidência sem commit; não há aprovação ou atestação reutilizável.
