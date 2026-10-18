# Revisão independente b28r3 — PR #196

- Decisão: **changes-requested**. **Quatro grupos de bloqueantes de merge**, com todas as variantes encontradas abaixo; sete follow-ups de baixo risco.
- Repositório: `oseiaspereira88/pose`; spec `pose-distributed-instructions-parity`; HEAD `4deb172ebcdc30d0fc0810c91997355adb1cdc51`.
- Revisão do diff completo `origin/main...HEAD`, da spec, dos pareceres anteriores e das correções da b28r3. Atribuição por `POSE-Spec:` conferida no corpo das mensagens, sem depender do parser de trailers do Git.
- Revisor: `agent:independent-gpt-6.1-sol-review`, **Codex/OpenAI**, execução independente, vendor diferente do implementador **Claude/Anthropic**. Identidade e separação são **declaradas**, não autenticadas. Este parecer não é atestação.
- Evidência compacta: [b28r3-independent-review.json](../results/b28r3-independent-review.json), incluindo entradas sintéticas, stdout/stderr, resultados dos consumidores, checks e identidade do binário construído deste HEAD.
- Não houve seal, attest, close, commit ou push; policy e pins de issuer não foram alterados. A spec permanece `in-progress`. A alteração que já existia em `.pose/indexes/generation.json` foi preservada; não é achado da PR. Escritas dos assessments foram isoladas ou restauradas ao estado inicial.

## Rules applied during review

- Tipo: **mixed** — bugfix de descoberta, contrato JSON aditivo, documentação distribuída e testes.
- Skills: `.agents/skills/pose-review/SKILL.md`, `.agents/skills/pose-spec-closeout/SKILL.md`; workflow `.pose/workflows/review.md`; manual `POSE.md`; template e spec lidos.
- `.pose/rules/backend-go.md`: entradas, erros, CLI/MCP e testes Go.
- `.pose/rules/security.md`: confinamento, identidade dos caminhos e integridade da trilha retornada.
- `.pose/rules/documentation-style.md`: instruções, mensagens e mirrors EN/pt-BR.
- `.pose/rules/delivery-evidence.md`: separar checks verdes de comprovação das bordas negativas.
- `.pose/rules/knowledge-governance.md` e `_base-recurrence.md`: memória consultada, ownership e recorrência. Consultados a nota `2026-09-12-note-cli-output-design-taxonomy`, a busca por suggest/B28 em `.pose/knowledge/` e os três pareceres anteriores.
- React, Kubernetes e infraestrutura: sem implementação de runtime nestes domínios; as regras específicas não são necessárias para revisar seus aliases/nomes de extensão.

## Plano e escopo

Autoridade local `proj.pose-dist`, sem redirect, ambiguidade ou transferência; context revision `06306ea22bf1c6ee37a39551604aceb1a92c5b87af4fd69f65e3367797454346`.

- Plano `spec-closeout@1`, 15 critérios, 12 ferramentas, band critical, independência `different-actor`; profiles backend-review, engineering-judgment, high-criticality-review e structural-materiality. Componente pose-mcp: declared+observed, owner ausente explicitamente diagnosticado; paths de governança não mapeados também diagnosticados.
- Plan digest: `sha256:301704aa1f2c4094b1391036cf5fa1f23432622870ebc4d23b43a85a9cd3bda2`.
- Bundle somente **prepared**: `rvb-884ecbefec853902`, digest `sha256:884ecbefec853902a0c06ff92f5583b9e2b88d37f57670d99db738148fe10658`.
- Patch: `sha256:0472dd4e05db542327a3e9fe6dd88cd4c98f71168ac938bee327c680abff650c`; tree: `sha256:9566582579a5950c44811286663d4b3355ad64f6abbb71d771dd199653bb691a`.
- A atribuição contém 11 commits; o range inclui outros trabalhos. Os 24 arquivos do sujeito canônico foram observados pelo assessment de design, com três deltas (um check e duas instruções governadas). O diff contra origin/main foi inspecionado adicionalmente; não se atribuem os commits intermediários de outra spec à b28.

Ferramentas required ativas executadas: artifact-check, skills-check, validate e surface-check. Recomendadas executadas: suggest-review, assess-design, assess-discover com `--if-stale`, assess-integrate e assess-tech-debt. `review-check` e `closeout-check` permanecem **deferred**: sua precondição review-complete não foi alcançada, e o pedido proíbe selar/atestar com bloqueantes.

## Bloqueantes de merge — lista consolidada

### B1 — Repo-map null continua sendo interpretado como mapa sem correspondência

- **Medium, bloqueante de merge**; R3/R6/R7/R8. Local: `pose-mcp/internal/cli/suggest.go:264` e `:275`.
- Reprodução: repo-map com o conteúdo literal `null`, task válida e `--path svc/main.py` retorna exit 0, `domain_source=repo-map-no-match`, `domain_status=no domain inferred`, stderr vazio. A CLI e `Store.Suggest` devolvem a trilha sem regras do domínio, indistinguível de uma resposta legítima sem match. `{}` foi usado como controle de objeto vazio, não como defeito.
- Causa: Unmarshal de null em `map[string]any` não dá erro e deixa o mapa nil; o loop de coleções ausentes conclui ausência. As validações de coleção/item novas só são alcançadas se houver uma chave.
- Ação: exigir objeto não-null na raiz, distinguindo perda da configuração de objeto vazio suportado; cobrir a raiz além das coleções/items em humano, JSON e consumidor.
- Vizinhos procurados: `ReadIndex` valida bytes/geração, não o schema legado; `loadReviewRepoEntries` também aceita null no struct e segue para fallbacks. `checkTaskMap` já exige tasks objeto, mas aceita task null; a correção de suggest para task null está presente. O handler MCP delega a `Store.Suggest`; a propagação de B1 foi exercitada nesse Store.

### B2 — Definições repetidas ainda podem remover regras ou escolher domínio pela ordem

- **Medium, bloqueante de merge**; R3/R4/R6/R8. Locais: `suggest.go:51`, `:264` e `:312`.
- Variante JSON: `{"tasks":{"feature":{"rules":["security"],"rules":[]}}}` retorna exit 0 e `rules_status=[]`. No repo-map, repetir `domain` como backend-go e depois unknown remove a inferência sem diagnóstico. A segunda ocorrência substitui a primeira antes das validações. `Store.Suggest` reproduz ambos os sucessos.
- Variante entre módulos: entries `svc` (Go) e `./svc` (backend/Python) têm o mesmo caminho normalizado. Inverter sua ordem muda `domain_effective` de backend-go para backend e elimina a regra Go. Não se trata de ordenação temporal: o desempate `len(p) <= best` conserva a primeira configuração conflitante sem divulgar a ambiguidade. A mesma classe atravessa apps/services/packages.
- Ação: recusar chaves repetidas dos documentos consumidos e definições conflitantes para o mesmo caminho canônico, ou fornecer um contrato explícito de conflito que não oculte obrigações. Uma repetição equivalente pode ser deduplicada; uma divergência não deve virar inferência inequívoca. Acrescentar negativos de repetição em todos os níveis relevantes e de permutação das coleções.
- Vizinhos procurados: `checkTaskMap`, `loadReviewRepoEntries` e `loadModuleMetadata` usam Unmarshal permissivo; o loader de review também deduplica o path conservando o primeiro. O produtor `scanModules` mantém um módulo por diretório, mas isso não valida um índice legado consumido. Os leitores de bundles/atestação e journal já têm rejeição de campos JSON repetidos; `source_coverage_round5_test.go` cobre a perda de um campo de tentativa repetido. Não é necessário novo mecanismo de policy.

### B3 — Caminhos são validados, normalizados e confinados com semânticas diferentes

- **Medium, bloqueante de merge**; R3/R6/R7/R8. Locais: `suggest.go:38`, `:303`, `:323` e `:340`.
- Variante de identidade: módulos distintos `svc` (Go) e ` svc` (backend/Python), ambos nomes válidos no filesystem, colidem por TrimSpace. `--path ' svc/main.py'` retorna backend-go e a extensão Go; CLI humana, JSON e Store reproduzem o domínio errado. O produtor `relativePath` conserva o espaço, portanto pode produzir o índice que o consumidor interpreta incorretamente.
- Variante de confinamento lexical: `path=' ../svc'` passa na validação do índice, mas é transformado em `../svc` depois; retorna no-match sem divulgar path inválido sob a semântica usada para comparar. O mesmo desalinhamento existe no argumento CLI: `--path ' ../external/k8s/deploy.yaml'` é aceito antes de trim e depois infere k8s. O Store recusa esse argumento com `..`, de forma conservadora, enquanto aceita e reproduz a colisão de nomes com espaço.
- Variante de link: módulo `svc` é symlink para um diretório **sintético** fora do projeto contendo go.mod; repo-map declara backend/Python. Suggest faz Stat no go.mod externo e publica backend-go. Nenhum conteúdo privado foi lido: somente fixtures temporárias e a presença do arquivo sintético foram usadas.
- Ação: usar a mesma representação para validar e comparar, preservando nomes literais suportados ou recusando-os explicitamente; validar após a canonicalização adotada e não seguir módulo fora da raiz ao consultar go.mod. Manter raiz legítima, `./`, fronteira de prefixo e especificidade.
- Vizinhos procurados: `confinedRelativePath` limpa sem TrimSpace; `relativePath` preserva nomes; `validateReviewPath` resolve links e recusa escape real; Store.Suggest usa IsAbs/Contains("..") e recusa mais caminhos que a CLI. `ruleFileState`/GetRule foram inspecionados separadamente: links de regras legíveis são suportados, links quebrados agora são unreadable; isso não constitui autorização de módulo externo.

### B4 — Validação parcial da task deixa metadados forjarem a trilha e referências impossíveis

- **Medium, bloqueante de merge**; R4/R6/R8. Locais: `suggest.go:78`, `:107`, `:153` e `:169`.
- Variante de campos derivados: uma task com `rules:["security"]`, `name:"other"`, `domain_effective:"backend-go"` e `rules_effective:[]`, usando repo-map `{}`, sai 0. JSON diz `name=other`, domínio backend-go e regras efetivas vazias, ao mesmo tempo que `rules_status` diz security instalada e `domain_status` diz no domain inferred. Humano mantém feature e security. `Store.Suggest` entrega o JSON contraditório ao consumidor MCP.
- Variante de referências: `workflow:17, skill:false` é anunciado com sucesso; paths de workflow inexistente e skill inexistente também são devolvidos sem estado de indisponibilidade ou diagnóstico. As sondas humano/JSON retornam exit 0. A validação nova de rules não cobre os outros campos que compõem o roteiro.
- Ação: construir a resposta com campos declarados permitidos e campos derivados autoritativos, sem deixar configuração sobrescrevê-los; validar tipos e divulgar ausência/ilegibilidade de referências declaradas antes de sugeri-las como roteiro disponível. Não exigir campos opcionais nem instalar nada automaticamente. Cobrir humano/JSON/Store com controles válidos e inválidos.
- Vizinhos procurados: `checkTaskMap` rejeita tipos numéricos/booleanos de workflow/skill e verifica existência de referências não vazias, mas suggest não o chama; o check não conhece campos derivados e não valida rules_by_domain. `requireTaskRef` confina paths lexicalmente. GetRule valida o identificador antes de ler, mas não corrige a cópia indiscriminada de campos da task; MCP e Store não filtram a resposta.

## Follow-ups aceitáveis — baixo risco, sem caminho para aprovação indevida

Os itens abaixo não dispensam B1–B4. A spec já conserva F1–F5 como `[open]`; nenhuma disposition ou ownership da spec foi alterada com bloqueantes. Ownership para os itens residuais: `(owner:@core crit:low review:2026-12-01)`.

- **F1, low** — Mostrar domínio/alias/source explícitos na saída humana sem `--path`. O JSON explica a origem e as regras/estados coincidem no caso válido. Já open.
- **F2, low** — Normalizar `Go` antes do ValidateName no adaptador MCP; hoje o Store recusa conservadoramente o alias aceito pela CLI. Nenhuma trilha é aprovada nessa recusa. Já open.
- **F3, low** — Remover “no apply exists in 6.x” de `internal/pose/legacy_inventory.go:49`. A recusa de apply é correta; o texto residual não habilita nenhuma operação. Já open.
- **F4, low** — Aplicar o executor sem shell e a classificação estruturada também a b14/b15. Não usar esses executores como evidência nova desta revisão. Já open.
- **F5, low** — Doctor ainda recomenda instalar extensão quando `ruleExtensionInstalled` colapsa unreadable em false. O diagnóstico é advisory, não instala nem aprova nada; suggest já preserva unreadable. Já open.
- **F6, low** — Linha vazia adicional no EOF do parecer d7ebc300, linha 173, faz `git diff --check origin/main...HEAD` sair 2. Higiene documental, sem efeito no julgamento dos gates.
- **F7, low** — Atualizar o changelog da b28 para também citar o estado `unreadable`; sua enumeração narrativa lista installed/extension-needed/unavailable. O runtime já devolve a falha explicitamente e não recomenda instalação nesse estado. A lacuna é descritiva.

## Varredura por classe e limite do atalho adversarial

- **Ordenação por texto:** sorts de tipos de task e de apresentação não governam cronologia. Conferidos o desempate de módulos (B2), inversão do catálogo regra/extensão (13 arquivos distintos) e consumidores de review. Não foi encontrado novo defeito de ordenação temporal neste delta.
- **Ausência versus perda:** conferidos raiz/tasks/listas/domain maps, raiz/coleções/item do repo-map, ReadIndex, ruleFileState/doctor/GetRule, inventários fonte/instalado e inventário migrate. Os quatro bloqueantes anteriores têm correção e regressão; B1 e as variantes B2/B3 ainda perdem informação silenciosamente. As supressões históricas de InventoryLegacy não são apresentadas como resolvidas pela b28.
- **Campos sem validação:** cobertos nomes de regra, task nula, listas nulas, mapas, tipos de path/domain/language, campos repetidos, workflow/skill e chaves derivadas. Foram seguidos os leitores e validadores vizinhos antes de classificar B1/B2/B4.
- **Comparação de caminhos:** raiz, `./`, prefixo com fronteira, especificidade, empates, espaços, travessia antes/depois de normalização e links externos/quebrados foram comparados com produtor, CLI, Store, loader de review e leitura de regras.
- **Inventários e distribuição:** leitura fonte e instalada agora propaga erros; guards de AGENTS e as regressões novas foram inspecionados. Install/update reais EN/pt-BR passaram sem adoção. Não se registra novamente o B4 anterior como aberto.
- **Integridade de índice:** `index --verify` no checkout isolado do HEAD passa, geração `gen-4550e4483c5c54f3`; a divergência do descritor pré-existente no clone original é separada da PR.

Atalho adversarial mais barato remanescente: duplicar rules para [] ou copiar rules_effective/identidade falsos para a task; o comando retorna uma trilha válida na forma que perdeu obrigações ou contradiz a leitura real. A suíte verde e as 34 mutações não contêm esses negativos. O corpus `pose-mcp/internal/cli/testdata/adversarial/README.md` cobre separação declarada e produtores indisponíveis em outros gates, mas não fecha estas bordas de suggest. Regressões dos leitores devem impedir esses atalhos antes do merge.

## Checks e critérios

- `pose validate --strict --module pose-mcp`: **exit 0, 137/137 pass**, sem failed/errored/skipped; 264,8 s. Inclui build, go test ./..., vet, integrações CLI/MCP, jornadas, contratos e checks com race registrados na matriz. A execução sem filtro é coberta pela partição dos módulos: `mcp-enforce` passou **3/3**, totalizando **140/140 checks** entre os dois módulos.
- `python3 -I .pose/results/b28-mutations.py`: **exit 0, 34/34 killed**, em cópias isoladas; não substitui as sondas negativas B1–B4.
- `surface-check --spec ... --results <resultado atual> --strict`: **exit 0**, um target capability, 137 resultados, zero findings. Resultado passado explicitamente; delivery.json/results_path não foi alterado nem usado para selar evidência antiga.
- `artifact-check --spec ... --strict`, no checkout isolado do HEAD: **exit 0**, 32 claims, 38 paths observados, 11 commits atribuídos, 641 findings globais históricos; nenhum bloqueante da spec apontado pelo gate. Contagem global não é contagem de defeitos desta PR.
- `skills-check --strict`: **exit 0**, 22 skills, zero erros/warnings. `lint-spec --strict`: **exit 0**, oito R-IDs/trace válidos e cinco follow-ups open; aviso de Tasks opcional. `check --strict`: **exit 0**, 29 warnings históricos/adoption/completude.
- `index --verify` no HEAD isolado: **exit 0**, geração gen-4550e4483c5c54f3. O descritor inicialmente modificado no clone original foi preservado.
- `assess design`: **exit 0**, 24/24 arquivos observados, três deltas. `assess discover --if-stale --component pose-mcp`: **exit 0**, um componente atualizado por mudança de conteúdo/matriz, executado isoladamente para preservar a instância.
- `assess integrate`: **exit 0**, 64 contratos, 63 consumidores não observados estaticamente; consumidores de suggest foram seguidos diretamente e Store.Suggest foi exercitado. `assess tech-debt`: **exit 0**, um TODO coberto, zero uncovered observados; **cobertura incompleta** por duas specs históricas inválidas, não prova global de ausência de dívida.
- `recurrence-check --tolerant --window-days 14`: **exit 0**, 393 registros, zero flags. Sem escalação automática. `followups --all --json`: executado para consultar backlog/duplicatas, sem alterar dispositions.
- Install e update reais em Git fixtures EN e pt-BR: **exit 0** em todas as operações, 36 Markdown por instância, nenhuma discovery incondicional e hashes de policy preservados.
- Scan offline por padrões de segredo sobre 10.147 linhas adicionadas: zero candidatos; sem dependências alteradas. govulncheck, gitleaks e golangci-lint indisponíveis; não se alega executá-los. `git diff --check origin/main...HEAD`: exit 2 por F6.


- assumption-integrity/design-causality: D1 reutiliza mecanismos existentes de forma proporcional; não resolve as perdas/ambiguidades B1–B4. Os deltas de workflow/regra têm base em R1/R2, e a adição do check em R8; não se atribui validade causal só à presença dos links.
- correctness/requirements/backend-input-errors/backend-observability/operability: checks verdes não anulam reproduções negativas com sucesso indevido; estes critérios permanecem insatisfeitos.
- backend-contracts/backend-integration-impact/compatibility: JSON e adaptador foram confrontados, inclusive execução de Store.Suggest; B4 permite divergência de conteúdo entre humano e máquina. Não há dependência nova nem nova concorrência de produção; os checks com race são os já registrados na matriz.
- documentation/scope: fontes/mirrors/install/update coerentes para freshness e apply; F1/F3/F7 são lacunas descritivas. Atribuição, bundle prepared e assessment registram os limites do range multisspec.
- security: sem candidato de segredo nas adições; B3 é limite de integridade/confinamento de metadados, sem demonstração de exfiltração. Identidade do revisor e vendor são declarações, sem pretensão de prova assinada.
- solution-proportionality: corrigir leitores, construção da resposta e canonicalização, com negativos. Não mudar policy, issuer, assinatura, scheduler ou contratos done para obter um selo.

A revisão só pode seguir para seal/atestação sem envelope, verify e close depois de remediar todos os grupos bloqueantes e repetir os checks pertinentes. Os itens aceitáveis ficam abertos com ownership no eventual closeout. Não foi criado handoff de risco aceito para merge, pois a decisão é changes-requested.

Contributor Mode: oportunidades adjacentes foram registradas no parecer. Nenhum draft foi staged e nenhuma contribuição upstream foi enviada; esses atos dependem da decisão explícita do usuário. Sem threshold de recorrência atingido.
