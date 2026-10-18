# Revisão independente b28r4 — PR #196

- Decisão: **changes-requested**. **Quatro bloqueantes de merge (medium)** e dez follow-ups aceitáveis (low), consolidados abaixo.
- Repositório: `oseiaspereira88/pose`; spec `pose-distributed-instructions-parity`; HEAD `6d38fefabb96ea74dca4166d71fb2bda3068a543`.
- Revisor declarado: `agent:independent-gpt-6.1-sol-review`, **Codex/OpenAI**, vendor diferente do implementador **Claude/Anthropic**. Execução independente; identidade e separação declaradas, sem autenticação criptográfica ou alegação de independência cognitiva verificável. Este parecer não é atestação.
- Evidência: [b28r4-independent-review.json](../results/b28r4-independent-review.json), com sondas sintéticas humano/JSON/Store, plano, checks e identificação do binário construído deste HEAD.
- Não selar, atestar ou fechar este sujeito antes da remediação. Esta execução não altera policy, não adiciona issuer pin, não commita e não envia alterações. A spec permanece `in-progress`.

## Rules applied during review

- Tipo **mixed**: bugfix, disponibilidade e diagnóstico de CLI/JSON/MCP, documentação distribuída e regressões.
- Skills `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`; workflow `.pose/workflows/review.md`; `AGENTS.md`, `POSE.md`, template e spec consultados.
- `.pose/rules/backend-go.md`: validação, erros, consumidores Go e checks do módulo.
- `.pose/rules/security.md`: integridade da trilha, entradas e limites de caminhos.
- `.pose/rules/documentation-style.md`: mensagens e fontes/mirrors EN/pt-BR.
- `.pose/rules/delivery-evidence.md`: evidência atual, artefatos atribuídos e distinção entre suíte verde e correção semântica.
- `.pose/rules/knowledge-governance.md` e `_base-recurrence.md`: memória, ownership e recorrência. Busca em `.pose/knowledge/`, nota `2026-09-12-note-cli-output-design-taxonomy` e os quatro pareceres anteriores consultados.
- Regras de runtime React/Kubernetes/infraestrutura não aplicáveis: o delta descobre seus nomes, sem implementar esses runtimes.

## Escopo, plano e ferramentas

Revisado o diff `origin/main...HEAD`: 41 arquivos, incluindo produção, testes, matriz, spec, instruções, mirrors, relatórios/resultados anteriores e índices. Também revisado o delta após `4deb172e`. Os trailers `POSE-Spec:` foram conferidos no corpo das mensagens. O range canônico contém trabalho de outra spec; sua extensão não atribui esses commits à B28.

Autoridade local `proj.pose-dist`, sem redirect, transferência ou ambiguidade. Context revision `2ae33ad9a0f103d3f28db276f07e8a29c5e2d63e906d6e1b8f001875c80cb9e4`.

- Plano `spec-closeout@1`, 15 critérios e 12 ferramentas; band `critical`, independência `different-actor`. Profiles: backend-review, engineering-judgment, high-criticality-review e structural-materiality, além do base.
- Plan digest: `sha256:7b5995e05b67a62890c6dabdf2024aefb4c25ea708d4b52556a6691d89c84796`.
- Bundle apenas **prepared**: `rvb-fa7b0f8e10067d1a`; digest `sha256:fa7b0f8e10067d1a6212cd0a3e7696d20f61c0849e3ad6df01c04ec383348725`.
- Patch `sha256:e71e8eaf39e28d756f2e0dfb01da1ebdf8e0c62c86a6751887c63d692538461f`; tree `sha256:d3cf882d92a8f1c98b3de8d463f41880575d281e5260975c092763dfbe7a8d6d`.
- Componente pose-mcp `declared+observed`; owner ausente e paths de governança não mapeados continuam avisos explícitos. Design assessment observou 25/25 arquivos, um check e duas alterações de instruções governadas.
- Required ativos: artifact-check, skills-check, validate e surface-check. Recommended executados: suggest-review, assess-design, assess-discover (`--if-stale`), assess-integrate e assess-tech-debt. Resultados finais abaixo.
- Completion-only `review-check` e `closeout-check` **deferred**: não há review aprovado. Seal, auto-attest, attest, verify de aprovação e close não executados, conforme o pedido com bloqueantes.

O worktree de auditoria é separado e fixado neste HEAD. Escritas de assessments e resultados de validação ficaram nele. No clone solicitado, a modificação prévia de `.pose/indexes/generation.json` foi preservada; ela não é um achado da PR. Os exemplos `rvb-6167567bf3fa6ac4` / `rva-5358c584e2a8bd4f` confirmam o formato sem envelope; nenhum julgamento antigo foi reutilizado.

## Bloqueantes de merge — lista consolidada

### B1 — Workflow e skill ilegíveis são anunciados como disponíveis

- **Medium; bloqueante de merge**. R4/R6/R8. `pose-mcp/internal/cli/suggest.go:504–513`.
- Reprodução, como uid 1000: task aponta workflow e skill regulares com modo `000`. Ambos os arquivos dão `PermissionError`/errno 13 ao ler; `suggest feature --json` sai 0 com `references_status[].availability=available`. A saída humana não dá ressalva. `Store.Suggest`, usado pelo MCP, devolve a mesma disponibilidade falsa.
- Causa: `Stat` verifica existência/tipo, mas não a possibilidade de abrir/ler. O comentário promete leitura, porém a implementação não a testa.
- Ação: verificar leitura dos dois tipos de referência, preservar erro como `unreadable` e exercitar permissões em humano/JSON/Store, com controle legível. Não abrir FIFO/dispositivo; verificar tipo regular antes de ler.
- Vizinhos: `ruleFileState` verifica tipo e `Open` e não reproduz esse falso positivo; `GetSkill`/`GetWorkflow` tentam ler e recusam. `checkTaskMap.requireTaskRef` também usa apenas `Stat`, logo não é proteção para esta variante. Não estender a exigência de confinamento de go.mod a links legíveis de regras/skills, que já são suportados.

### B2 — A detecção de chaves repetidas não cobre o case folding do struct taskMap

- **Medium; bloqueante de merge**. R4/R6/R7/R8. `suggest.go:53–59`, `pose-mcp/internal/pose/json_duplicate_keys.go:6`.
- Entrada: `{"tasks":{"feature":{"rules":["security"]}},"Tasks":{"feature":{"rules":[]}}}`. A CLI humana e JSON saem 0, sem diagnóstico, e retornam a task sem security. `Store.Suggest` reproduz. Inverter os dois campos restaura security: a ordem do documento decide as obrigações.
- Variantes reproduzidas: `TASKS`, `TaSkS` e `taſks` (long s Unicode) também substituem o mesmo campo do struct sem serem recusadas pelo guard. A correção deve acompanhar o case folding real do decoder, não apenas lowercase ASCII.
- Causa: `encoding/json` casa `tasks`/`Tasks` no campo do struct sem diferenciar caixa; o helper exportado usa `rejectDuplicateJSONKeys(..., fold=false)`. As chaves são diferentes para o guard e equivalentes para o consumidor.
- Ação: validar a identidade dos campos com a mesma semântica usada na decodificação, ou selecionar estritamente a chave canônica antes de decodificar. Não aplicar case folding indiscriminado às chaves de mapas que são realmente sensíveis a caixa. Cobrir as duas ordens e manter os negativos de repetição exata.
- Vizinhos: `checkTaskMap` verifica `rawDocument["tasks"]`, mas depois decodifica em outro struct: a mesma entrada passa pela primeira condição e a task sobrescrita não exige security. O leitor de repo-map de suggest usa mapas de chaves exatas, portanto esta variante de caixa não é equivalente ali. `loadReviewRepoEntries` usa structs e deve entrar na auditoria de parsers de metadados: não possui a mesma validação estrita do novo leitor de suggest. Os leitores de headers de review já têm proteção específica de repetição com case folding e testes `TestCaseFoldedHeaderNamesAreRepeatedNames`; reutilizar o guard sem a opção correta não herda essa proteção.

### B3 — O confinamento verifica o diretório do módulo, mas não o alvo de go.mod

- **Medium; bloqueante de merge**. R3/R6/R7/R8. `suggest.go:350`.
- Reprodução: repo-map declara `svc` como backend/Python; `svc` é um diretório real dentro da raiz, porém `svc/go.mod` é symlink para um arquivo regular sintético fora dela. `--path svc/app.py` retorna `backend-go`, source repo-map, e recomenda a extensão Go. Humano/JSON/Store reproduzem.
- Causa: `Stat` segue o link do arquivo, enquanto `insideProject` resolve somente `svc`. O teste novo cobre o diretório `svc` externo, mas não o arquivo `go.mod` externo.
- Ação: resolver e confinar o arquivo efetivamente consultado, preservando controles de go.mod interno, ausência, diretório no lugar do arquivo e módulo raiz. A mera presença fora da raiz não deve alterar o domínio local.
- Vizinhos: comparados argumento CLI, normalização do índice, `insideProject`, `confinedRelativePath`, `validateReviewPath`, `hasPackageDependency` e leitores de regras/referências. `validateReviewPath` resolve o path que recebe; aqui o path recebido pelo teste de confinamento é incompleto. Leitura de regra/skill via symlink tem contrato próprio e não foi arbitrariamente proibida.

### B4 — Os demais campos operacionais da task ainda aceitam tipos impossíveis

- **Medium; bloqueante de merge**. R6/R8. `suggest.go:110–114` e `:178`; validadores `suggestTaskRules`/`suggestTaskReferences`.
- Entrada: `{"tasks":{"feature":{"rules":["security"],"requires_spec":[],"requires_adr":{},"validation":false,"knowledge_consume":17}}}`. Humano publica `Spec: []`, `ADR: map[]`, `Validation: false`, `consume=17`, exit 0; JSON e Store devolvem os mesmos campos inválidos sem diagnóstico.
- A varredura adicional reproduziu null, lista, objeto e número em `requires_spec`, `requires_adr`, `validation`, `knowledge_consume`, `knowledge_produce` e `description`: os 24 casos saem 0 nos três consumidores. Nem todos os metadados têm o mesmo peso, mas a perda de instrução de validação e das obrigações operacionais não pode ser aceita como reserva editorial.
- Causa: o novo limite valida rules, workflow/skill e campos derivados, mas copia/renderiza os demais campos sem validar sua forma. Uma task sem instrução de validação válida continua parecendo uma trilha canônica bem-sucedida.
- Ação: validar os campos operacionais declarados segundo seus tipos suportados, preservando bool/string nos campos que já admitem ambos e os campos opcionais ausentes. `validation`/`description` textuais não devem aceitar objetos/listas/números/booleanos; não exigir que a prosa suportada seja um comando executável. Cobrir também null declarado e controles válidos do task-map distribuído.
- Vizinhos: `checkTaskMap` tipa apenas workflow/skill/rules e não fiscaliza esses metadados; o handler MCP delega ao Store e não filtra a resposta. `strictStringSlice` resolve listas de regras, não o restante da task. Portanto outro check não corrige a saída publicada.

## Follow-ups aceitáveis — baixo risco

Nenhum item abaixo dispensa B1–B4. Não são aceitos como risco os casos que omitem obrigações ou afirmam disponibilidade falsa. Ownership sugerida para todos: `(owner:@core crit:low review:2026-12-01)`. Como há bloqueantes, a spec não foi modificada para triá-los nesta execução.

- **F1** — Expor domínio/alias/source na saída humana sem `--path`. JSON explica; regras e estados válidos coincidem. Já `[open]`.
- **F2** — Normalizar `Go` antes de ValidateName no adaptador MCP. Hoje a recusa é conservadora, sem trilha aprovada. Já `[open]`.
- **F3** — Remover “no apply exists in 6.x” de `internal/pose/legacy_inventory.go:49`. Dry-run continua informativo e apply é recusado corretamente. Já `[open]`.
- **F4** — Remover shell interpolado e heurística das mutações b14/b15. Não foram usados como evidência desta revisão. Já `[open]`.
- **F5** — Doctor deve distinguir regra ilegível de ausente, em vez de recomendar instalar extensão quando `ruleExtensionInstalled` retorna false. Advisory; não instala nem aprova. Já `[open]`.
- **F6** — Remover a linha vazia extra ao EOF do relatório d7ebc300, linha 173; `git diff --check origin/main...HEAD` a aponta. Higiene sem efeito no veredito dos gates.
- **F7** — Acrescentar `unreadable` à enumeração de estados no changelog B28. O runtime já anuncia a falha de regra e não a trata como instalada.
- **F8** — Workflow/skill com link quebrado, inclusive diretório pai perdido, deve dizer perda/`unreadable`, e não ausência/`unavailable`. A sonda dos links de arquivo reproduziu nos dois campos. A classificação atual já impede afirmar disponibilidade; o refinamento é diagnóstico. `ruleFileState` já distingue esse caso para regras.
- **F9** — Ampliar o filtro de `distributed-instructions-parity-integration` para incluir as regressões novas de root module, configuração malformada/nula, chaves repetidas, referências e inventário ilegível. O filtro atual não seleciona esses testes, embora a execução obrigatória de `go test ./...` os cubra: não se dispensa esse check completo nem se usa o filtro estreito como prova dos casos ausentes.
- **F10** — Distinguir falha de leitura de task-map de arquivo ausente em `suggest.go:48–51`. Hoje ambos dizem `task-map missing`; ambos também saem 2, sem retornar trilha válida. Refinar diagnóstico sem enfraquecer a recusa.

## Varredura por classe e atalho adversarial

- **Ordenação por texto/ordem de entrada:** sorts de apresentação de task/rules não decidem cronologia. Conferidos empate de módulos e inversão regra/extensão; catálogo tem destinos distintos. O desempate de módulos conflitantes foi corrigido. B2 registra o novo caso de ordem de JSON, sem confundi-lo com ordenação temporal. Conferidos consumidores de bundles/atestação e seus testes de comparação por instante.
- **Ausência versus perda:** raiz/tasks/listas/domain maps, repo-map/coleções/items, índice verificado, task-map ilegível, regras, workflows, skills e inventários fonte/instalado. Raiz null e repetição exata foram corrigidas; B1 e F8/F10 mostram as bordas remanescentes de leitura e diagnóstico.
- **Campo sem validação:** nomes, listas, null, campos derivados, referências, restantes metadados da task e compatibilidade do decoder com chaves repetidas. B2/B4 incluem os validadores vizinhos, não só o produtor da saída.
- **Comparação de caminhos:** raiz, `./`, espaços literais, fronteira de prefixo, especificidade e empates, travessia, links de diretório e links de arquivo. A correção para espaços e para diretório externo está presente; B3 cobre o próximo segmento do mesmo caminho.
- **Distribuição:** fontes/mirrors e inventário de Markdown revisados; jornadas reais de install/update EN/pt-BR executadas. Não tratar teste textual como prova da inferência de domínio ou disponibilidade de arquivo.

Atalho mais barato ainda possível: manter `tasks` com security e escrever `Tasks` depois com regras vazias; a detecção nova de repetição passa e humano/JSON/MCP recebem uma trilha sem security. Outro atalho é afirmar leitura com `Stat` de arquivo fechado por permissão. O corpus `pose-mcp/internal/cli/testdata/adversarial/README.md` protege declaração versus autenticação de independência e perdas em outros gates, mas não possui essas bordas de suggest. A suíte e mutações existentes não substituem esses negativos.

## Checks e limites

- `pose validate --strict --module pose-mcp --json-out .pose/results/b28r4-validation.json`: **exit 0, 137/137 pass**, zero failed/errored/skipped, 287,3 s. Inclui build, `go test ./...`, vet, integrações CLI/MCP, jornadas e os checks com race registrados na matriz.
- `pose validate --strict --module mcp-enforce`: **exit 0, 3/3 pass**. A união por módulo cobre **140/140 checks**; não se alega ter executado literalmente a invocação global sem filtro.
- `python3 -I .pose/results/b28-mutations.py --output /tmp/b28r4-mutations.json`: **exit 0, 43/43 killed**, em cópias isoladas. Ainda assim, as 38 sondas adicionais (cada uma em humano, JSON e Store) encontram B1–B4. Não houve contagem de erro de compilação como morte nesta execução.
- `surface-check --spec pose-distributed-instructions-parity --results .pose/results/b28r4-validation.json --strict`: **exit 0**, um target, 137 resultados atuais, zero findings; provenance digest `sha256:c57935ed81013338eba98973ee8a094c7a771405cbe4c9afce634b733ed3de75`. Usado o arquivo recém-gerado explícito, sem selar evidência antiga do `results_path`.
- `artifact-check --spec ... --strict`: **exit 0**; 13 commits atribuídos, 35 claims, 41 observed. Os 641 findings exibidos são warnings globais de proveniência/orphans; não se convertem automaticamente em defeitos novos da B28 nem se confundem com a atribuição do slug revisado.
- `skills-check --strict`: **exit 0**, 22 skills, zero erros/warnings. `lint-spec ... --strict`: **exit 0**, 8 requisitos com trace, zero falhas; Tasks opcional ausente é aviso.
- `check --strict`: **exit 0**, 29 warnings históricos explícitos, sem os esconder como resultado sem ressalvas. `index --verify`: **exit 0**, geração `gen-3f98362207cc186a` current no HEAD isolado.
- `assess design`, `discover --if-stale --component pose-mcp`, `integrate` e `tech-debt`: **exit 0**. Discovery refreshed por conteúdo/matriz; design 25/25 arquivos; integrate observou 64 contratos e 63 gaps de consumidor não observado, que não provam desconexão funcional. Tech-debt encontrou 1 marcador e nenhum descoberto sem cobertura, mas declara **scan INCOMPLETE** devido a dois specs históricos com follow-up vazio: `pose-instance-engine-version-tracking` e `pose-knowledge-durable-reference-type`. Não se afirma cobertura global limpa.
- `recurrence-check --tolerant --window-days 14`: **exit 0**, 393 records, zero flagged/flapping keys, zero resolved clusters, threshold 3. `followups --all --json`: **exit 0**, também divulga as duas fontes históricas inválidas; nenhum item foi automaticamente descartado ou deduplicado.
- Install seguido de `update --no-self`, em repositórios Git sintéticos EN e pt-BR: **4 operações exit 0**, 36 Markdown verificados por operação, zero discovery incondicional. Policies geradas pelo install ficam byte a byte iguais após update; artifacts/delivery continuam disabled e adoption-decisions vazio. Isto verifica os defaults de instalação e a ausência de adoção adicional, não a inexistência de arquivos de policy na fixture.
- Scan local dos padrões de private key, token GitHub e chave AWS no diff adicionado: zero matches; revisão manual sem segredo observado. Gitleaks, govulncheck e golangci-lint não estão instalados; não se alega executá-los. Nenhuma dependência mudou. Build/vet e skills security scan passaram; os limites do scanner local permanecem explícitos.
- `git diff --check origin/main...HEAD`: **exit 2**, apenas F6. Não é tratado como check verde.

## Critérios, compatibilidade e próximo passo

- `backend-concurrency`: sem concorrência nova; verificações Go/matriz proporcionais ao delta.
- `backend-contracts` e `compatibility`: campos de disponibilidade aditivos; recusas de configuração corrompida não exigem quebrar callers válidos. Entretanto, B2/B3/B4 impedem afirmar preservação semântica completa da trilha.
- `backend-input-errors`, `backend-observability`, `backend-integration-impact`, `correctness`, `requirements`, `operability` e `security`: não satisfeitos integralmente por B1–B4; Store/MCP preservam as falhas do produtor.
- `documentation`: mirrors/freshness verificados; F3/F6/F7 são reservas editoriais.
- `scope`: intenção de D1 e R1–R8 preservada, sem adoção de policy; as correções adicionais devem permanecer no mesmo limite de leitores e caminhos.
- `assumption-integrity`, `design-causality` e `solution-proportionality`: a intenção de reutilizar taxonomia/guard é adequada, mas a premissa de mesma identidade dos campos e leitura/confinamento efetivos falha nas sondas. A correção proporcional é alinhar decoder/validador, verificar leitura real e confinar o arquivo consultado. Não há justificativa para novo mecanismo de assinatura ou adoção de policy.

Recorrência: domínio implementação/contrato/validação, causa validação parcial entre leitores vizinhos e regressões que cobrem só a forma anterior do defeito. `recurrence-check` não sinalizou keys/clusters acima do threshold; não se abriu escalonamento formal. A prevenção é acrescentar as variantes do mesmo consumidor e seus controles às regressões. Nenhuma contribuição upstream foi staged ou enviada sem consentimento do usuário.

Remediar B1–B4, executar novamente os negativos e checks e realizar nova revisão independente do HEAD resultante. Só então preparar evidência nova no `results_path`, indexar, selar, atestar **sem envelope**, verificar e fechar. Os follow-ups seguros devem permanecer `[open]` na spec com a ownership indicada. Não foi criado handoff de aceitação de risco: não há aprovação condicionada ou risco bloqueante aceito.
