# Revisão independente — B30r3 — PR #197

Data: 2026-10-18. **Decisão: changes requested. Cinco bloqueantes de merge e seis follow-ups aceitáveis.** Spec `pose-spec-facts-evidence-binding`, HEAD `0234a60cfda92cd8599622fc5ba439d3d4a77d1d`, branch `impl/pose-b30-spec-facts-binding`, repositório `oseiaspereira88/pose`. Este parecer avalia esse HEAD e preserva os pareceres anteriores.

## Autoridade e independência

Revisor `agent:independent-gpt-6.1-sol-review`, execução Codex/OpenAI; implementador Claude/Anthropic, conforme declaração do usuário e coautoria dos commits. Identidade e separação de vendors **declaradas**, sem autenticação criptográfica. Nenhum subagente utilizado. Aplicados skills `pose-review` e `pose-spec-closeout`, workflow `.pose/workflows/review.md`, AGENTS do escopo, POSE.md, template de spec e regras abaixo. Não houve selo, atestação, envelope, alteração de policy/pin de issuer, close, commit ou push.

`pose context` resolveu `xref:proj.pose-dist/spec:pose-spec-facts-evidence-binding`, mesma autoridade, in-progress, context revision `f81f2dd1dc1a8932d659a333a76aee69a69758319658a15e0ce8aa938057d470`. `gh pr view` confirmou PR/branch/HEAD; `git fetch origin main` atualizou a referência consultada. Examinados diff completo contra origin/main, R1–R8/V1–V6/D1, trailers, testes e mutações anteriores, documentação EN/pt-BR root/dist, contratos de geração/compactação e produtores/consumidores próximos. A CLI deste HEAD foi compilada em `/tmp/b30r3-pose`; probes independentes rodaram em cópia descartável de pose-mcp/mcp-enforce. **PASS de um probe significa defeito reproduzido, não aprovação.** Fontes e saídas estão na [evidência compacta](../results/b30r3-independent-review.json).

## Rules applied during review

- Change type: bugfix com mudança de contrato público de facts e documentação.
- Workflow: `.pose/workflows/review.md`.
- `.pose/rules/backend-go.md`: domínio, parsing, erros, identidade e consumidores Go.
- `.pose/rules/security.md`: integridade de bindings, entrada não validada, caminhos, render e dependências.
- `.pose/rules/documentation-style.md`: spec, manuais, changelog e parecer.
- `.pose/rules/delivery-evidence.md`: relevância, freshness, composição e atribuição de evidência.
- `.pose/rules/knowledge-governance.md`: consulta de ADRs e handoffs de generation, delivery integrity, compactação, assurance e B06; nenhum risco bloqueante aceito nem handoff de aprovação produzido.
- `.pose/rules/_base-recurrence.md`: classe e causa dos achados; recorrência mecânica verificada.
- Frontend, infraestrutura e outros stacks não se aplicam. Não existe tool MCP própria de facts; CLI texto/JSON consome a mesma estrutura. Assess integrate executado para conferir os limites compartilhados.

## Bloqueantes de merge

### F01 — high — Componente inválido ainda vira raiz e associa módulo alheio

**Local:** `pose-mcp/internal/pose/spec_facts.go:285–289, 439–441, 462–479`. **Requisitos:** R1, R2, R7.

A correção distingue módulo ausente/inválido em `factModuleKind`, mas os alvos perdem essa distinção. `normalizeFactPath("../outside")` e `normalizeFactPath("/outside")` devolvem `""`; o loop de components conserva esse valor porque o texto original não é vazio. `moduleCoversSpec` interpreta o alvo vazio como raiz universal. Na spec com único path atribuído `api/server.go`, basta um desses components inválidos e o digest scoped correto para um check de `web` entrar em `checks/associated` sem limite individual. O campo components é tag livre no parser; não existe rejeição anterior que torne o caso impossível.

**Varredura da classe:** conferidos `normalizeFactPath` em todos os call sites, módulo do resultado, components, paths normais e ambos os lados de rename, `moduleMatchesTarget`, `moduleCoverageIsInferred`, validação de artifact paths e parser de components. O lado módulo já rejeita vazio/absoluto/escape; o lado alvo não. Paths inválidos do índice também convergem para o mesmo alvo vazio. A convenção preexistente de alvo explicitamente `.`/`root` foi observada separadamente; o contraexemplo bloqueante não depende de contestá-la. IDs de componente que apenas não resolvem para paths ficam em L01.

**Ação:** conservar tipo/validade dos alvos e nunca converter alvo inválido/desconhecido em raiz; divulgar limite e manter candidato quando a cobertura só depender desse alvo. Cobrir components e paths inválidos, módulos irmãos/descendentes, aliases válidos e renames sem regressão do root explicitamente declarado.

### F02 — high — Um change set resolvido libera associação baseada em outro não resolvido

**Local:** `spec_facts.go:224–244, 291–313`. **Requisitos:** R1, R2, R7.

A guarda `resolvedSets > 0` só exige que algum change set tenha ResolvedHead e paths. Todos os paths, inclusive de sets não resolvidos, alimentam pertinência; o digest inclui todos os sets. Probe com um set de api resolvido e outro de web sem ResolvedHead promove o resultado de web a associado, com somente a ressalva de ancestry — nenhum aviso de delta não resolvido. A existência de observação em api não demonstra a observação usada para web.

Há ainda uma definição estrutural incompleta de resolução: head não vazio e paths não comprovam que base/diff foram resolvidos. A revisão não pede que cada leitura reconstrua Git: pede que dados incompletos não recebam o predicado forte por um contador global.

**Varredura da classe:** `ScopedDeliveryProvenanceDigest` inclui claims/sets sem validar resolução; `resolveGitChangeSet` produz Base/Head resolvidos, commits, paths e DiffDigest em conjunto; `loadRecordedChangeSets` aceita records JSON sem validar esse conjunto; o decoder schema 1/2 também não fornece a garantia. `validate` emite scoped provenance para toda spec encontrada nos sets. Comparados guards de artifact/surface e a distinção entre declarações e observações do ADR de delivery integrity. Não há validação anterior que impeça o contraexemplo misto.

**Ação:** verificar suficiência/resolução do fundamento efetivamente usado na associação, sem emprestar resolução de um set a outro; conservar como candidato/limitation o escopo inconclusivo. Cobrir zero sets, apenas não resolvido, misto, campos estruturais ausentes e controle inteiramente resolvido.

### F03 — high — Chave concatenada colide e certifica como fresh um arquivo que mudou de módulo

**Local:** `spec_facts.go:180–205, 303–307`; `pose-mcp/internal/pose/delivery_validation_run.go:17–42`. **Requisitos:** R1, R2, R3, R7.

`factResultKey` concatena strings com NUL e scope entries com `=`/`,` sem escape ou comprimento. O parser aceita esses caracteres nos campos e não valida a relação entre ID, módulo e nome. As tuplas `(ID="x", Module="api", Check="t\u0000unit")` e `(ID="x\u0000api", Module="t", Check="unit")`, com demais campos iguais, têm a mesma chave.

Probe com geração publicada e arquivo de resultados correspondente começa fresh com evidência de api. Reescrever somente id/module/name para a segunda tupla mantém `fresh` e continua mostrando o resultado antigo de api como associado, embora a fonte agora nomeie o módulo irmão t. O problema não é só ordenação: a mesma chave faz deduplicação e comparação de fonte, e pode apagar um resultado distinto conforme a ordem. Há também colisão de scopes entre `{"alpha":"a,b=c"}` e `{"alpha":"a","b":"c"}`; outcomes/severities fora do vocabulário também são aceitos pelo decoder e ampliam as colisões possíveis.

**Varredura da classe:** todas as utilizações de factResultKey, ParseDeliveryValidationRun e o antigo loader compartilhado, schema 1/2, compactValidationRunID, validationSetID, chaves de path/claim e campos emitidos por validate. CompactValidationRunID serializa o objeto JSON e distingue essas tuplas; o novo helper não preserva essa propriedade. Names/IDs de resultado não têm a restrição de NUL de nomes reais de arquivo Git. Chaves de paths de Git não tornam seguro aplicar concatenação a campos arbitrários de resultados.

**Ação:** comparar representação estrutural canônica/injetiva de todos os campos e maps; rejeitar ou divulgar entradas fora do contrato antes de associar. Não usar somente um filtro parcial de caracteres em um consumidor. Testar colisão de separadores em todos os campos livres, maps, ordem invertida, comparação de fonte e deduplicação, conservando multiplicidade de resultados diferentes.

### F04 — medium — Projeção descarta fonte e severidade que distinguem os resultados

**Local:** `spec_facts.go:41–66, 308–310`; `pose-mcp/internal/cli/specs_cmd.go:240–262`. **Requisitos:** R1, R4, R6, R7.

A identidade interna agora conserva Report e Severity, mas `SpecFactCheck` e seu binding descartam ambos. Dois resultados de arquivos de origem diferentes, ou um required e outro optional, tornam-se objetos de facts byte a byte iguais; a saída humana também é indistinguível. Manter duas entradas sem a informação que as distingue não conserva sua fonte nem permite auditar qual evidência está sendo copiada/reusada. `facts.source` aponta para o índice, não para cada fonte original. Probes reproduzem separadamente report e severity.

**Varredura da classe:** enumeração de todos os campos de DeliveryValidationResult, ParseDeliveryValidationRun, compactação/expansão, factResultKey, construção de SpecFactCheck, factLine e saídas CLI JSON/texto. A perda ocorre na nova projeção; produtor/índice já guardam os dados. ExpectedScopedDigest também não tem campo próprio no texto humano, mas divergências de digest são explicadas nos limites; isso não restitui Report/Severity perdidos em ambos os formatos.

**Ação:** preservar fonte original e severidade na projeção e no render, ou fornecer referência inequívoca que permita recuperá-las. Testar diferenças isoladas em cada campo e exigir que fontes/identidades distintas permaneçam observavelmente distintas. Não interpretar optional como required por omissão.

### F05 — high — Candidato ou path com newline injeta linha humana de evidência associada

**Local:** `specs_cmd.go:225–231, 240–270`; `pose-mcp/internal/cli/cliout/render.go:177–188`. **Requisitos:** R5, R6, R7; regra security.

`factLine` concatena dados sem escape; `Renderer.Field` escreve literalmente. Probe de resultado candidato sem scope, com newline no nome do check, produz uma linha separada exatamente `facts.check=api test (unit) pass [associated by scope-provenance]`. A marca CANDIDATE fica em outra linha. Um leitor/agente que copie a linha falsificada recebe evidência forte que o JSON corretamente não declarou. Um segundo probe usa path atribuído contendo newline e reproduz o mesmo efeito; nomes Git podem conter newline.

**Varredura da classe:** campos livres do resultado/binding/limites, Source/Freshness, facts.path normal e rename, novo factLine, observedPathText, Field/RecordField e serialização JSON. O render comum não escapa valores de Field; JSON mantém o dado dentro da string. Também há superfície para CR/controles de terminal e delimitadores do rótulo; não limitar a correção apenas ao nome do check usado no probe. A propriedade nova de linha copiável sempre identificada como candidata não é satisfeita.

**Ação:** garantir uma linha física por fato e codificação inequívoca dos dados, incluindo paths/renames e controles, preservando a paridade semântica com JSON. Cobrir LF, CR, delimitadores e nomes de arquivo válidos com newline. Não basta colocar CANDIDATE no sufixo de uma string multilinha.

## Follow-ups aceitáveis — variantes conservadoras de baixo risco

Todos têm ownership proposto `(owner:@core crit:low review:2026-12-01)`. Não foram adicionados à spec nesta rodada com bloqueantes; L01/L02 já estão [open].

- **L01 — low — ID de componente sem resolução para path.** Um ID legítimo diferente de seu diretório pode deixar um resultado pertinente como candidato ou produzir explicação imprecisa. Aceitável somente a variante conservadora; conversão em raiz/associação alheia é F01. Resolver por metadata antes de explicar cobertura.
- **L02 — low — Unavailable não projetado quando o índice falta/corrompe.** GetDeliveryIntegrity retorna erro antes de construir Freshness; CLI responde falha facts-unavailable, sem resultado vazio de sucesso. Expor structured unavailable preservando a recusa. Conferidos GetDeliveryIntegrity, ReadIndex, handler e testes de índice danificado.
- **L03 — low — Explicação genérica incorreta de unknown.** A limitação sempre diz “not published as a set”, inclusive para geração publicada sem bindings ou histórico ilegível. Status/Reason permanecem conservadores. Usar a causa real e alinhar manuais/changelog; nunca elevar unknown a fresh.
- **L04 — low — Desempate parcial de ordenação.** Associated e candidates ordenam por Module/Check/ID/Outcome, sem classe/binding. Inverter dois resultados diferentes só nesses campos inverte o JSON. Todos permanecem presentes e nenhuma seleção por ordem é feita, portanto esta variante não aprova indevidamente. Usar desempate estrutural completo. Conferidos também sort por ID no loader, BuildDeliverySurface e compactação, sort completo de paths e semântica temporal de reviews: aqui não há escolha cronológica que justifique ordenar timestamps por texto. Colisões com perda/associação são F03.
- **L05 — low — Leitura de histórico sem limite de saída.** SpecTrailerCommits usa exec.Output sobre todo log --all que menciona POSE-Spec, enquanto a atribuição canônica limita a saída a 32 MiB. Repetidas consultas podem consumir memória/latência proporcional a todo histórico e corpos de outros slugs. Compartilhar leitura limitada e, ao exceder limite, manter unknown/erro. A variante aceita é indisponibilidade/performance, sem truncar silenciosamente e chamar fresh. Não foi criado histórico gigante para estressar memória.
- **L06 — low — Identidade da execução vazia desaparece.** Se o arquivo não produz nenhum resultado indexável, ParseDeliveryValidationRun retorna zero entries e descarta o header. Alterar head, generated_at e scope de um run vazio conserva fresh; o probe confirma isso. Nenhuma evidência associada aparece e a limitação de ausência de checks permanece, portanto esta variante isolada não permite aprovação indevida. Esclarecer que freshness compara a projeção indexada ou conservar identidade do run vazio; não transformar zero checks em entrega comprovada. Fonte ausente sem resultados retidos foi distinguida de perda de resultados retidos; adição/remoção de resultado e schema inválido já rebaixam freshness.

## Plano, contratos e julgamento

Bundle somente **prepared**, não selado: `rvb-43b7f9e881420deb`, digest `sha256:43b7f9e881420deb59e12488654469b179ab947b298439c26ed736d0b7d7cd7c`. Plano `sha256:b06d9b4f6e01080b02894d037858857f6174aeda5ab97e441c88e5fdb7f5cb56`; patch `sha256:5313ace5bf6ec77d6558b66112dbb499a1655e14aa9fa366e54c125967420db4`; tree `sha256:683323365c4ce71d383ca02d6e60c084b721fdcc016f8f5ad60400a1036f0cad`. 15 critérios, 11 ferramentas, componente pose-mcp (Go, high, declared+observed); owner ausente e governança não mapeada são warnings explícitos, não assurance adicional.

Os seis bloqueantes de b30r2 receberam correções e testes diretos; a nova varredura identifica variantes residuais, sem declarar que um teste verde prova a classe inteira. Renames agora conservam ambos os lados, o parser de resultados e a semântica dos trailers foram compartilhados, resultados diferentes nos campos usuais são preservados e o digest do vazio sem set resolvido não associa.

Scope e proporcionalidade da solução são adequados; separar association, freshness e julgamento é causalmente pertinente. Não há dependência de produção ou concorrência nova. Contudo F01–F05 impedem concluir correctness, requirements, input/errors, contracts, observability, security, compatibility, documentation e operability. Assumption-integrity e design-causality precisam tratar a premissa falsa de fundamento verificado. Integração e concurrency mecânicas não substituem esses julgamentos. Os testes com race registrados na matriz cobrem os limites compartilhados.

Contrato público: checks reduz sua semântica e candidates/freshness/binding são aditivos, classificados minor no changelog. Não foram alterados REST/Kafka/Protobuf/tools MCP ou policy. A garantia de binding verificável, preservação da fonte e linha copiável segura ainda não é sustentada.

Atalhos adversariais baratos: componente inválido que vira raiz; um set resolvido para liberar caminhos de um não resolvido; separadores nos campos para colidir com a projeção antiga; newline que imprime uma falsa facts.check. O corpus `internal/cli/testdata/adversarial/README.md` não contém caso B30 para essas variantes. Os probes anexos fornecem reproduções; negativos devem entrar na suíte antes da próxima revisão.

## Checks e disposição de ferramentas

| Ferramenta / check | Resultado e disposição |
|---|---|
| validate global --strict | exit 0; **140/140 pass**, 252s; zero fail, optional_failed, errored ou skipped; [resultado estruturado](../results/b30r3-independent-global-validation.json) |
| validate / pose-mcp | coberto pelo mesmo run: **137/137 pass**, incluindo build ./..., test ./..., vet ./..., integração B30 e suites registradas com race; sem repetição desnecessária |
| check --strict | exit 0; 29 warnings preexistentes de governança, nenhum erro |
| lint-spec --strict | exit 0; oito requisitos/trace e dois follow-ups open; somente Tasks opcional ausente |
| artifact-check --spec --strict | exit 0; 25 claims, sete commits atribuídos, 31 paths observados; 644 warnings globais de orphan, nenhum erro; trailers conferidos no corpo das mensagens |
| surface-check inicial | exit 1; 139 resultados antigos da fonte configurada, stale-evidence e unconsumed-capability |
| surface-check --results b30r3-independent-global-validation.json --strict | exit 0; **140 resultados, zero findings**; override somente da consulta, sem mudar policy |
| assess-design / discover if-stale | executados; design observa 17/17 entradas do subject; discover atualiza componente por código/matriz alterados; evidência capturada e byproducts de assessment restaurados |
| assess integrate / tech-debt | executados; integração registra seus limites; debt tem um marcador coberto e cobertura incompleta por duas specs históricas inválidas, sem apresentar isso como ausência de dívida |
| suggest review --path pose-mcp | executado; seleção e rationale de regras acima |
| recurrence-check --tolerant --window-days 14 | exit 0; 393 registros, zero chaves sinalizadas; sem gatilho mecânico de escalation |
| govulncheck | exit 0; nenhuma vulnerabilidade encontrada; sem dependência de produção nova |
| scan de segredos do diff | quatro padrões, zero hits; gitleaks/golangci-lint não instalados; vet executado pela matriz |
| probes independentes | exit 0; **11 probes** em cópia descartável, reproduções e controles documentados na evidência compacta |

Ferramentas required ativas (artifact-check, surface-check e validate global/módulo) executadas; recommended design/discover/integrate/tech-debt/suggest usadas por pertinência à integridade de evidência e limites compartilhados. Completion-only closeout-check/review-check permanecem deferred por bloqueantes. Nenhuma ferramenta mecânica é tomada como resposta automática aos critérios de julgamento.

A consulta inicial de validate com JSON em /tmp foi recusada antes de executar checks; o run válido usou .pose/results/. O surface-check atual comprova composição mecânica, não resolve F01–F05. A recorrência por domínio é de contrato/implementação e lacuna de validação de variantes: prevenção proposta é um corpus negativo para alvos, resolução, identidades e render, sem alterar policy. Não foi criada contribuição separada.

Assessment byproducts foram restaurados após captura. `generation.json` já estava modificado na entrada; o artifact-check nativo republicou delivery-integrity e descriptor. Esses dois derivados permanecem modificados e sem commit, para inspeção junto ao parecer/evidências; não se afirma preservação byte a byte do descriptor inicial, que não foi snapshotado antes dos comandos nativos. HEAD e todos os arquivos de implementação/spec/policy permanecem intactos.


## Estado deixado e próximo passo

Completion tools (seal, auto-attest, attest, verify, review-check, closeout-check e close) foram **deferred por bloqueantes**, conforme instrução expressa. Não houve selo/atestação de rejeição. Nenhum commit/push, mudança de spec, policy ou issuer; nenhum feedback staged/submetido upstream sem consentimento. Este parecer é o registro local dos achados, não uma contribuição separada.

Corrigir F01–F05 e qualificar as variantes descritas, mantendo follow-ups aceitáveis abertos com o ownership solicitado quando houver aprovação. Gerar evidência atual antes de selar; somente sem bloqueantes, atestar sem envelope com identidade e separação declaradas, verificar, fechar e enviar o closeout ao branch. Não há declaração de entrega concluída neste parecer.
