# Revisão independente — b28 / PR #196 / d7ebc300

Decisão: **changes requested**. **9 bloqueantes de merge (medium)** e **3 follow-ups aceitáveis (low)**. Este parecer responde pelo HEAD `d7ebc3001168a2121d1f8dbd493a659d4d8e1f5f` contra `origin/main` / merge-base `8aa729a76ab1c094690a0892263a86bf3bb7bb42`, na spec `pose-distributed-instructions-parity`.

O parecer local anterior, sobre `cc590314`, foi consultado como pista. Todos os casos B1–B8/F1–F3 foram reproduzidos com binário compilado deste HEAD; B9 é novo em `d7ebc300`. O parecer anterior e sua evidência foram preservados. A revisão cobriu os 28 arquivos do delta, fontes, mirrors e consumidores próximos. Três commits atribuídos têm `POSE-Spec: pose-distributed-instructions-parity`; a reconciliação nativa mostra 23 claims/28 observações, incluindo índices derivados.

Revisor declarado: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI; implementador Claude/Anthropic. A identidade e a separação de vendor são **declaradas**, sem autenticação criptográfica ou observação de independência cognitiva. A forma sem envelope das revisões anteriores (`rvb-6167567bf3fa6ac4` / `rva-5358c584e2a8bd4f`) foi conferida; seus julgamentos não foram reutilizados.

Nenhum selo, atestação, closeout, commit ou push foi realizado. Policy e issuer pins não foram alterados. O trabalho executável e as reconstruções ocorreram em cópias isoladas fixadas no HEAD; no clone original foram mantidos somente o novo parecer e a evidência compacta, além dos dois artefatos locais que já existiam.

Evidência: [b28-d7ebc300-independent-review.json](../results/b28-d7ebc300-independent-review.json). Contém resultados estruturados, reproduções sintéticas e caminhos/SHA-256 dos logs integrais temporários. Não contém credenciais.

## Rules applied during review

- Change type: bugfix misto com documentação e produtor de evidência.
- Skills: `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`; workflow `.pose/workflows/review.md`; manual `POSE.md` e template da spec consultados.
- `.pose/rules/backend-go.md`: inputs, erros, integração CLI/MCP e checks Go.
- `.pose/rules/security.md`: identidade de artefato, confinamento de paths e integridade da evidência.
- `.pose/rules/documentation-style.md`: comandos, textos EN/pt-BR e mirrors distribuídos.
- `.pose/rules/delivery-evidence.md`: separar sucesso mecânico, cobertura e comprovação funcional.
- `.pose/rules/knowledge-governance.md`: conhecimento anterior e riscos residuais; `.pose/rules/_base-recurrence.md`: causa e prevenção dos padrões repetidos.
- Conhecimento: `module-metadata-discovery-invalidates-review-provenance` e `review-assurance-requires-cryptographic-proof`.
- Frontend/Kubernetes como implementação não se aplicam: a PR não muda aplicações ou manifests dessas stacks. A descoberta das respectivas regras integra o contrato revisado.

## Plano efetivo e disposição dos tools

O contexto resolveu a autoridade local `proj.pose-dist`, spec em `in-progress`, no HEAD solicitado. A inconsistência dos índices B9 impede o binário deste HEAD de resolver o plano completo em um checkout sem caches. O plano parcial informa bloqueio; sua banda menor não prova baixo risco.

Após reconstruir os índices **somente na cópia isolada**, o plano completo tem 15 critérios, 12 entradas de tools, banda `critical` e separação `different-actor`. Digest: `sha256:4483bc4b7473f4cb3f1bc520ebbf74f7c3efd49ae41d8dcfdda836ee194ec207`. Bundle apenas preparado: `rvb-debf49d6885d705e`, digest `sha256:debf49d6885d705e374f4f0e81a5075bbb7fb3c117cb0be0a7935c39aed3727d`; patch `sha256:a0b6a256bf50a4fb3e2b1d27f0d00b9ff2b497f07d7721d7aeb080f5928e1b55`, tree `sha256:44494b88e4fdfd8e38d93b3f032a8512dc187e974ef0089339e9da7ac1efece4`. Não selado nem gravado como aprovação.

Required ativos executados: validate strict por módulo, artifact-check strict, skills-check strict e surface-check strict. A união por módulo cobre os dois módulos Go aplicáveis; não se alega execução literal de `validate --strict` global. Recommended executados: suggest review, assess design, assess discover --if-stale, assess integrate e assess tech-debt; nenhum recomendado omitido. Completion-only (`review-check`, `closeout-check`, auto-attest, attest, verify e close) diferidos porque há bloqueantes e o usuário proíbe selar/atestar nesse caso.

Avisos de owner ausente em pose-mcp e paths documentais não mapeados foram mantidos; nenhuma policy foi relaxada para fazer o plano passar.

## Checks e seus limites

- `validate --strict --module pose-mcp`: **137/137 pass**, zero falhas/erros/skips, incluindo build, suíte Go completa, vet, integração, races registrados e o novo check da b28.
- `validate --strict --module mcp-enforce`: **3/3 pass**. União: **140/140**.
- `check --strict` na cópia limpa do HEAD da b28: **exit 1, 54 erros de closeout**; o mesmo binário em `origin/main` retorna **exit 0, 29 warnings**. Após reconstruir somente os índices na cópia isolada, o mesmo check passa com 29 warnings, como main: os 54 erros eram consequência de B9. O HEAD commitado continua inconsistente.
- `surface-check ... --strict` no original: **exit 1** por evidência antiga/stale. Com índices reconstruídos e o run fresco no results_path, **exit 0, 137 resultados, zero findings** na cópia isolada. O original não recebeu os resultados novos.
- `skills-check --strict`: 22 skills, zero erros/warnings; `lint-spec ... --strict`: sucesso, aviso da seção opcional Tasks.
- `artifact-check ... --strict`: exit 0, 23 claims/28 observações e 644 diagnósticos globais; o próprio comando registra que **não conseguiu atualizar o delivery index**, por B9. Exit 0 não demonstra que essa escrita ocorreu.
- `index --verify` no original: **exit 1**, geração divergente. A comparação de hashes foi feita diretamente sobre blobs Git em todos os sete arquivos do descritor e cinco revisões; não depende do estado temporário da cópia.
- `assess design`: inicialmente inconclusivo por B9; após reconstrução isolada, 20/20 entradas observadas e três deltas (check novo e duas instruções governadas). O rebuild não corrige os artefatos commitados da PR.
- `assess integrate`: 64 contratos/63 consumidores não observados estaticamente. São limites de observação, não 63 regressões demonstradas. Store.Suggest, handler MCP e GetRule foram conferidos diretamente.
- `assess tech-debt`: um TODO coberto, zero uncovered; **scan INCOMPLETE**, com duas specs históricas inválidas. `origin/main` reproduz a mesma cobertura incompleta; não se alega ausência global de dívida.
- `recurrence-check --tolerant --window-days 14`: 393 registros, zero flags. Não atingiu threshold de escalação.
- Install **e update reais**, EN e pt-BR: sucesso; 36 arquivos Markdown por instância sem discovery incondicional. Hashes de todos os arquivos de policy permanecem iguais antes/depois do update.
- Mutações em TMPDIR normal: 12/12 registradas como killed. O controle adversarial B8 demonstra que o executor também aceita erro de shell como killed; o número isolado não prova a causa.
- Scan offline de linhas adicionadas por padrões de chaves privadas/tokens/secrets: zero candidatos. Sem novas dependências. govulncheck, gitleaks e golangci-lint não estavam disponíveis; não se alega execução deles. `git diff --check` passou.

## Achados classificados — lista consolidada

B1/B6 incluem predicados preexistentes que R3/R6/R7 explicitamente prometem resolver; não se atribui sua introdução a esta PR. Os demais itens distinguem nova chamada, novo produtor ou artefato novo.

### B1 — Módulo raiz perde regras de domínio

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:188`; requisitos: R3/R6/R8.
- Reprodução/evidência: Após install, update e index reais em EN e pt-BR, repo-map declara language=go com path vazio. suggest feature --path main.go sai 0 e omite backend-go. A condição p != vazio exclui o módulo raiz; path ponto e prefixos não normalizados também não casam.
- Ação: Normalizar ambos os caminhos; reconhecer raiz como fallback de menor especificidade; preservar fronteiras e módulo mais específico.
- Mesma classe nos consumidores vizinhos/origem: Predicado preexistente, não resolvido pelo escopo que promete inferência correta. Vizinhos: produtor scanModules converte ponto para vazio; review_plan também exclui raiz como componente e documenta a limitação. Controle svc-other não casa com svc; svc/web vence svc.

### B2 — Kubernetes existente no catálogo é anunciado sem extensão conhecida

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:250`; requisitos: R4/R6/R8.
- Reprodução/evidência: Sem kubernetes.md, --domain kubernetes produz unavailable, sem next_step; humano afirma no extension known. extensions/pose-rule-kubernetes/extension.json existe e AGENTS documenta seu install.
- Ação: Reconciliar catálogo completo de arquivos/extensões, incluindo Kubernetes, sem instalação automática.
- Mesma classe nos consumidores vizinhos/origem: Novo consumidor inverte mapa do doctor incompleto. Conferidos todos os 12 pares existentes, manifests de extensions e resolução/install; nenhum segundo arquivo duplicado no mapa.

### B3 — Stat não prova disponibilidade e confunde ausência com falha de acesso

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:258`; requisitos: R3/R4/R6.
- Reprodução/evidência: Diretório backend-go.md e arquivo 0000 são installed; GetRule não os lê. Diretório .pose/rules sem permissão transforma regra existente em extension-needed. Diretório svc/go.mod também converte backend Python em Go.
- Ação: Separar inexistência de erro de I/O; verificar arquivo regular e legibilidade; não oferecer instalação como remediação de perda de acesso. Aplicar a mesma disciplina ao sinal go.mod.
- Mesma classe nos consumidores vizinhos/origem: Helper compartilhado ruleExtensionInstalled já usava apenas Stat no doctor; nova chamada o amplia a todas as regras. Vizinhos GetRule/ListRules recusam diretórios; requireTaskRef também apenas Stat; novo sinal go.mod repete o problema.

### B4 — Configuração de regras malformada é descartada com sucesso

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:86`; requisitos: R4/R6/R8.
- Reprodução/evidência: rules_by_domain.backend-go igual a string ou lista [17] resulta em domínio conhecido, exit 0, apenas regras base e nenhum diagnóstico; rules com elemento numérico é filtrado e task null também sai 0.
- Ação: Validar shape e tipos antes de montar a trilha; recusar configuração inválida com diagnóstico explícito, sem converter falha em conjunto vazio.
- Mesma classe nos consumidores vizinhos/origem: Coerção stringSlice preexistente herdada pelos novos statuses. Vizinhos checkTaskMap tipa rules base, mas não declara/valida rules_by_domain; Store.Suggest e MCP retransmitem o resultado.

### B5 — Identidade de regra sem validação anuncia caminho recusado pelo consumidor

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:254`; requisitos: R4/R6/R7.
- Reprodução/evidência: Com rules=[../../outside] e outside.md na raiz, sai 0 com availability installed e path .pose/rules/../../outside.md; GetRule recusa o mesmo identificador por ValidateName.
- Ação: Validar identificadores de regras antes de resolver paths; preservar confinamento e a mesma identidade aceita por GetRule. Não tratar arquivo arbitrário como regra instalada.
- Mesma classe nos consumidores vizinhos/origem: Nova classificação installed reutiliza string bruta. Vizinhos Store.GetRule/GetWorkflow/GetSkill validam nome; checkTaskMap verifica confinamento no projeto para regras base, mas não identidade de regra nem rules_by_domain. Não houve leitura/exfiltração de conteúdo pela sonda.

### B6 — Índice perdido ou inválido vira ausência silenciosa de domínio

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:173`; requisitos: R3/R6/R7.
- Reprodução/evidência: Mesmo path e task: mapa ausente, JSON malformado, geração inválida e mapa válido sem match retornam JSON idêntico, exit 0 e stderr vazio, sem domain_source/status. Um mapa inválido que continha módulo Go perde suas regras como se não houvesse módulo.
- Ação: Preservar source/estado e distinguir no-match de índice indisponível; propagar erro ou diagnóstico inconclusivo acionável antes de oferecer trilha parcial.
- Mesma classe nos consumidores vizinhos/origem: Fallback preexistente não resolve o contrato de diagnóstico da b28. Vizinhos ReadIndex preserva erro tipado e task-map inválido já falha; MCP herda o silêncio. InventoryLegacy também suprime alguns erros de policy/supplements: fora do delta semântico R5 e sem atribuir resolução a esta PR.

### B7 — Inventário de instruções aprova varredura incompleta

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/distributed_instructions_parity_test.go:33`; requisitos: R1/R2/R8.
- Reprodução/evidência: Em cópia exata do HEAD, arquivo com pose assess discover sem if-stale sob subdiretório 0000 faz TestDistributedInstructionsAskForDiscoveryOnlyWhenStale passar; chmod 0700 faz o mesmo teste falhar na instrução.
- Ação: Falhar em erros de WalkDir/ReadFile e em raízes esperadas ausentes; não substituir cobertura por contagem mínima global. Manter exclusões explícitas quando necessárias.
- Mesma classe nos consumidores vizinhos/origem: Código novo. Vizinhos teste de install ignora erros tanto de walk quanto de leitura; AGENTS ausente causa Skip, e os limiares 20/10 não detectam perda parcial de inventário. Teste de igualdade dos quatro mirrors usa leitura fatal corretamente.

### B8 — Executor de mutações registra falha de shell como mutação morta

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `.pose/results/b28-mutations.py:48`; requisitos: R8.
- Reprodução/evidência: TMPDIR com espaços gera cd: too many arguments, exit 2 e nenhum go test executado; o script registra killed=1,total=1. Captura observacional de subprocess.run preservou execução real e comprovou o erro.
- Ação: Executar argv com cwd, sem shell interpolado; distinguir infraestrutura/build de falha da asserção esperada, preferindo go test -json e eventos do teste-alvo; não contar erro desconhecido como killed.
- Mesma classe nos consumidores vizinhos/origem: Código novo. Vizinhos b14-mutations.py e b15-mutations.py repetem shell=True e a mesma heurística de duas substrings. Executor nativo validate usa program+args+cwd e falha em exit não-zero. As 12 mutações em TMPDIR normal foram reexecutadas e morreram, mas esse resultado não cobre este controle adversarial.

### B9 — Geração de índices commitada não corresponde ao delivery-integrity entregue

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `.pose/indexes/generation.json:31`; requisitos: R7; integridade dos artefatos/checks do closeout.
- Reprodução/evidência: No HEAD d7ebc300, index --verify sai 1: o descritor gen-169e0f9ccb87666c declara sha256:06cbfa8a09c2d217412d2578fda41be4e235b17587408b93493e7a9a8222aec7 para delivery-integrity.json, cujo conteúdo commitado tem sha256:07487a7af61f7a4af06bae81d998660ddd73b3add76ecd3ecb1fa5334c053f4e. Artifact-check também diagnostica index-generation-unavailable e não atualiza o índice.
- Ação: Reconstruir e publicar o conjunto de índices de forma coerente, preservando o descritor junto dos arquivos; exigir index --verify em checkout limpo antes do closeout.
- Mesma classe nos consumidores vizinhos/origem: Comparados todos os sete arquivos do descritor em origin/main, a358ec23, cc590314, merge 6c25d6a5 e d7ebc300. Main, a358ec23 e cc590314 são coerentes. O merge deixou releases.json divergente; d7 corrigiu esse arquivo mas introduziu a divergência em delivery-integrity.json. Vizinhos ReadIndex e ReplaceIndexOrLegacy recusam o conjunto inconsistente; o problema está nos artefatos entregues, não é pedido para relaxar o leitor.

### F1 — Saída humana não divulga alias/origem explícitos

- Severidade: **low**; classe: **follow-up aceitável**.
- Local: `pose-mcp/internal/cli/suggest.go:125`; requisitos: R3/R6.
- Reprodução/evidência: --domain Go tem domain_input/domain_source no JSON, mas sem --path a saída humana não mostra domínio/alias; mostra as mesmas regras e disponibilidade.
- Ação: Mostrar domínio efetivo, input e source também em texto.
- Mesma classe nos consumidores vizinhos/origem: Sem relaxar regras ou aprovação: diferença de explicação; ownership owner:@core crit:low review:2026-12-01.

### F2 — MCP recusa alias de caixa aceito pela CLI

- Severidade: **low**; classe: **follow-up aceitável**.
- Local: `pose-mcp/internal/pose/cli.go:34`; requisitos: R3/R6.
- Reprodução/evidência: CLI --domain Go normaliza; Store.Suggest/pose_suggest rejeita Go por ValidateName antes de delegar. Chamada real do adaptador reproduziu o erro.
- Ação: Normalizar o alias aceito antes de validar, mantendo bloqueio de travessia; testar ambas as superfícies.
- Mesma classe nos consumidores vizinhos/origem: Recusa explícita e conservadora, sem produzir resultado indevidamente aprovado; aliases minúsculos funcionam. ownership owner:@core crit:low review:2026-12-01.

### F3 — Limitação do dry-run ainda fala em 6.x

- Severidade: **low**; classe: **follow-up aceitável**.
- Local: `pose-mcp/internal/pose/legacy_inventory.go:49`; requisitos: R5.
- Reprodução/evidência: migrate v7 --dry-run --json em fixture retorna no apply exists in 6.x, enquanto help e recusa de --apply foram atualizados e motor é 7.4.1-dev.
- Ação: Alinhar limitations e comentários correntes ao fato sem versão: este motor não oferece apply.
- Mesma classe nos consumidores vizinhos/origem: O comando continua dry-run, writes vazio e apply recusado; texto não concede capacidade nem aprovação. ownership owner:@core crit:low review:2026-12-01.

## Varredura por classe e julgamento dos critérios

- **Comparação de caminhos:** B1/B5; conferidos scanModules, mapReviewPath, confinedRelativePath, Store.Suggest e GetRule/GetWorkflow/GetSkill. A raiz produzida por index é vazia, e o consumidor a descarta. Os controles de fronteira `service`/`service-other` e de módulo mais específico `service/web` são cobertos pelo teste executado `TestSuggestDomainUsesMostSpecificModuleAndSafeFallback`. A normalização precisa ocorrer nos dois lados sem perder essas fronteiras.
- **Ausência versus perda:** B3/B6/B7/B9; conferidos doctor, ReadIndex/ReplaceIndexOrLegacy, inventário do install e inventário migrate. O novo sinal go.mod repete o Stat fraco. InventoryLegacy também suprime alguns erros de policy/supplements/transfers: dívida adjacente fora da correção textual R5/B43; esta revisão não atribui sua resolução à b28.
- **Campo sem validação:** B4/B5; checkTaskMap tipa rules base, mas omite rules_by_domain; stringSlice filtra elementos inválidos; GetRule valida identidade e rejeita o identificador anunciado por suggest. O adaptador MCP herda a resposta CLI, mas aplica ValidateName ao domínio antes da normalização (F2).
- **Ordenação por texto:** tipos de tarefa e detalhes de migrate são ordenados só para apresentação; seleção de módulo compara comprimento com fronteira; inversão do catálogo tem 12 filenames distintos. Foram comparados todos os 13 manifests de extensão de regra: somente Kubernetes falta no mapa. Não foi encontrado novo julgamento temporal/semântico indevido por ordenação textual nesse delta.
- **Integridade do conjunto:** B9 compara os sete arquivos, não só o arquivo em erro. Main, a358ec23 e cc590314 são coerentes; o merge intermediário divergiu em releases.json; d7ebc300 diverge somente em delivery-integrity.json.

`assumption-integrity`, `correctness`, `requirements`, `backend-input-errors`, `backend-observability`, `operability` e `compatibility` não estão demonstrados: B1–B9 cobrem trilha parcial, falsos installed e falsos positivos de evidência. `backend-contracts`/`backend-integration-impact` exigem corrigir a divergência com GetRule, catálogo e MCP. `documentation` tem mirrors coerentes, mas a garantia narrada ainda excede os casos reais. `security` tem B5, sem segredo ou mudança de authority observados. `backend-concurrency` não ganhou concorrência de produção; os races aplicáveis passaram. `scope` está reconciliado pelo mecanismo nativo, com a falha derivada B9 explicitada.

D1 reutiliza corretamente freshness e catálogo, mas `design-causality` não sustenta disponibilidade real só com Stat nem morte por asserção só com exit diferente de zero. `solution-proportionality` pede correções locais e regressões negativas; não exige nova policy, issuer ou mecanismo de governança.

O atalho mais barato para satisfazer o produtor em forma sem entregar valor é ocultar parte do inventário (B7) ou fazer o shell falhar antes do teste (B8). As sondagens reproduzem ambos; o corpus `.pose` de adversarial (`pose-mcp/internal/cli/testdata/adversarial/README.md`) não tem controles que impeçam esses atalhos específicos. Incorporar os controles antes de usar o produtor como comprovação suficiente de R8.

## Disposição operacional

Os follow-ups F1/F2/F3 são diferenças de explicação, recusa conservadora ou texto residual; não abrem caminho para aprovação indevida. Se mantidos após remediar B1–B9, a spec deve recebê-los `[open]` com `(owner:@core crit:low review:2026-12-01)`. Como há bloqueantes, não houve edição da spec nem handoff que formalize aceitação de risco; a autorização deste ramo do pedido é parecer/evidência.

Não foi staged contribuição upstream: o contrato de Contributor Mode exige consentimento do usuário para draft/issue. A recorrência adjacente B8 foi registrada sem criar contribuição. Nenhuma mudança de policy ou pin é necessária para remediar os achados.

Corrigir B1–B9, repetir sondagens e checks em novo HEAD fixado e verificar os índices em checkout limpo. Só uma revisão posterior sem bloqueantes poderá selar, atestar sem envelope, verificar e fechar. Esta execução não produziu aprovação reutilizável.

