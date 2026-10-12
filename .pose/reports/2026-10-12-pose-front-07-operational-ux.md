# Frente 7 — eficiência operacional, combate à mecanização e UX

Data: 2026-10-12. Revisão fonte: `a0398f7faadeed14256efc03edab9d63041bcb58`, após o merge do PR #152. Código de `pose-mcp` idêntico ao tag v7.4.1.

[Índice](2026-10-11-pose-analysis-index.md) · [Evidências e fontes](../results/2026-10-12-pose-front-07-operational-ux.json) · [Harness dos probes](../results/2026-10-12-pose-front-07-ux-probes.py).

## 1. Parecer

O POSE já oferece mecanismos de proporcionalidade: specs minimal/standard/full, fatos derivados, assessment com binding e reuso, Attention, apresentação de requests, explicação de reuso de critérios e closeout recuperável. A evolução agora deve fazer esses mecanismos comporem uma jornada coerente. Não é necessário criar outra camada de checklists ou um comando por tipo de pendência. [E01–E20, E31, E32, E38]

A principal fricção está nas fronteiras: o plano pode anunciar conclusão que o verificador recusa; flags de automação são aceitos sem cumprir seu significado; filtros inválidos ampliam silenciosamente a consulta; os workflows distribuídos contradizem o reuso implementado. A UX do agente depende tanto desses contratos quanto a experiência da pessoa depende de mensagens acionáveis. A aparência do terminal não resolve divergência de estado ou próximos passos incompatíveis.

Recomendar primeiro **consistência de semântica, argumentos, fatos e instruções**; depois **orquestração proporcional de jornadas e validação de ergonomia com pessoas**. Preservar os gates que demonstram valor, automatizar sua preparação mecânica e limitar intervenção humana às exceções da policy. Um julgamento pendente precisa de reviewer competente, que pode ser outro agente quando o contrato permitir; não equivale automaticamente a uma decisão humana.

## 2. Método e limites

Inspecionar 50 fontes: CLI, renderer, Store, MCP, bootstrap, skills/workflows, template, tests, pilot, quickstart e métricas. Executar 15 probes em repositórios Git sintéticos com o binário oficial 7.4.1. P15 usa o transporte MCP real via stdio: initialize, tools/list e uma chamada read-only de closeout. Nenhum agente vendor, autenticação humana real ou serviço externo foi acionado.

A suíte Go não foi reexecutada porque não há executável Go disponível. O binário não foi recompilado, e não foi conduzido estudo de usabilidade humana. Fixtures pequenas comprovam comportamentos e controles; não medem latência de grandes projetos nem permitem afirmar ganho agregado de produtividade. Leitura de testes é evidência do contrato pretendido, não nova execução de toda a suíte.

Os resultados estruturados preservam exit codes, parsing do stdout e estado relevante, evitando duplicar JSON em strings. O harness conserva fixtures para inspeção. Não altera policy, disposition ou código do engine no repositório real. Recomendações e critérios de aceite são trabalho futuro, não novas specs automaticamente adjudicadas.

## 3. O que já reduz mecanização

| Capacidade atual | Valor comprovável | Limite da garantia |
|---|---|---|
| Spec mínima | Remove Tasks/Decisions opcionais e mantém fontes materiais | É redução de superfície do template, não autorização para omitir governança |
| `pose specs facts` | Deriva caminhos, commits e checks do index | Proveniência/freshness e associação dos checks precisam de disclosure mais preciso |
| `assess discover --if-stale` | Reutiliza estado vinculado a conteúdo, engine e matrix | Workflows ainda podem mandar rescan sempre; consolidado tem trabalho próprio |
| Context tipado | Declara projeto selecionado, autoridade e revision | Resolve identidade; não escolhe sozinho a próxima ação de engenharia |
| Attention | Mostra coverage, ator, efeitos por fase e residual | Não é um scheduler nem um certificado de autorização de write |
| Closeout recuperável | Replaneja; sela e registra passos mecânicos; stops têm checkpoint | Plano/status/verificação têm divergências e a superfície MCP é diferente |
| Reuso por critério | Mostra changed inputs e decisão de equivalência | Não decide adequação cognitiva ou confiança geral da origem |
| Renderer/JSON de gates | Canal de resultado, progresso separado, quiet e cores proporcionais | Há comandos fora do contrato comum e flags aceitos sem JSON |
| Corpus adversarial | Testa atalhos de satisfação meramente formal | Precisa percorrer as instruções e jornadas distribuídas, não só funções isoladas |

Não propor novamente os mecanismos que já foram implementados. Os testes existentes protegem reuso, separação de streams, preview sem seal, recuperação sem duplicação e parada em judgment; os probes desta frente incluem controles positivos dessas ideias. [E18–E20, E26–E30, E43–E47]

## 4. Jornada recomendada e fontes de verdade

A experiência deve orientar a pessoa ou agente por um percurso proporcional ao trabalho, com poucas decisões materiais e muita preparação derivada.

| Momento | Pergunta | Fonte/ação atual a reutilizar | Decisão material |
|---|---|---|---|
| Entrada | Qual tarefa/projeto tem autoridade? | `context`, spec existente, seleção explícita | Resolver ambiguidade real; não inferir projeto externo por path |
| Dimensionamento | Quanto governo este trabalho precisa? | Materialidade, surface, profiles, policy adotada | Intenção, risco e escopo; não checklist baseado só no tamanho do diff |
| Preparação | O que preciso ler/validar? | suggest, instructions, assessment fresh, knowledge pertinente | Escolha arquitetural quando material |
| Execução | O que pode continuar? | Readiness por fase, targets, requests | Respeitar restrições sem paralisar escopos independentes não afetados |
| Validação | O que foi demonstrado? | Resultados estruturados com provenance/coverage | Adequação ao requisito, quando exige julgamento |
| Review/closeout | O que falta e quem conclui? | Plan, independent reviewer/delegation, verify, guarded transition | Conclusões e exceções, sem atribuir prova a um rótulo de identidade |
| Retomada | O que mudou desde a interrupção? | Context atual, checkpoint e replanejamento | Reavaliar somente inputs materiais alterados |
| Entrega | O que terminou e o que permanece? | Facts, residual, release contract | Compromisso e risco residual explícitos |

Um typo dentro de escopo já autorizado pode exigir apenas edição e check pertinente. Um bug material pode reutilizar a spec e validação dirigida. Uma mudança de contrato pede análise, spec/ADR e integração. Uma decisão externa pode abrir ActionRequest com efeito apenas em closeout. O plano de review já é component-aware; fazer essa proporcionalidade chegar às instruções e ao executor, sem substituir política pelo julgamento informal do agente.

Distinguir superfície mínima de spec obrigatória mínima: não criar uma nova spec para cada mudança trivial apenas porque existe template menor. Preservar AGENTS, que exige spec para mudança não trivial, e rotas que podem referenciar autoridade existente. Evidência gerada elimina transcrição de fatos; não gera intenção, justificativa de arquitetura, aceite de risco ou equivalência semântica.

## 5. Sessões, continuidade e superfícies

Há quatro conceitos distintos que podem ser confundidos como “sessão”: contexto de projeto/tarefa por consulta; protocolo/transporte MCP com cursor/SSE; seleção persistida de continuous closeout; e runs opcionais do Conductor. Não são um único runtime persistente compartilhado. Um connection id não é task authority, e um run de board não encerra uma spec. [E04, E21–E25]

A CLI tem start preview/apply/status, closeout plan/apply/resume e continuous-closeout. No catálogo real observado via stdio havia 61 tools; a leitura de closeout é nativa, mas o plano recuperável de closeout não aparece como tool. O único outro nome contendo close foi `conductor_run_close`, cuja semântica é run externo, não fechamento de spec. Não recomendar esse tool como substituto do gate de closeout. [E21, E22, P15]

O MCP compartilha a projeção hierárquica do Store. P15 mostrou `terminal: false` e o mesmo next action da CLI para review ausente. Isso funciona. A oportunidade é expor planejamento read-only/ação admissível de forma consistente e oferecer handoff explícito para operações CLI que o MCP não executa. Não exigir paridade irrestrita de writes: assinatura, presença, contexto e identidade de execução têm contratos próprios.

Continuous closeout persiste um escopo e permite status/completar quando terminal. Não automatiza toda a execução em background; não introduzir daemon ou scheduler por necessidade de UX. O processo residente e seus custos pertencem à frente 8. Recuperação de arquivos/checkpoints e concorrência pertencem à frente 9.

O closeout de spec já possui uma trilha recuperável de nove passos; o checkpoint não é a fonte de verdade de entrega. Replanejar a partir do repositório é uma escolha acertada. Explicar o que foi feito, o que está ainda válido, o que precisa ser repetido e o que aguarda outro ator. Preservar checkpoints como histórico de execução útil sem convertê-los em atestação de suficiência. [E01, E02]

## 6. Resultados reproduzidos

Execute `python3 .pose/results/2026-10-12-pose-front-07-ux-probes.py --binary /caminho/pose --output /tmp/front07.json`. O source checkout fornece apenas o template atual; `--source` permite indicá-lo explicitamente.

| Probe | Preparação | Resultado | Interpretação |
|---|---|---|---|
| P01 | Draft, preview de closeout | JSON com bloqueio, spec byte a byte igual, zero bundles | Controle de preview; exit 0 significa consulta válida, não ready |
| P02 | Spec done criada depois da adoção, sem review requerido | Plan terminal true, passos verify/judgment done; closeout-check exit 1, terminal false | Lifecycle bruto virou atalho de conclusão no plano |
| P03 | Apply/resume com `--json` sobre a spec done | Exit 0 e texto `closed`, sem documento JSON | Flag aceito, contrato de saída descumprido; não demonstra write aprovado pelo gate |
| P04 | `context` resolve autoridade; plan recebe `--expect-context` | Context JSON válido; plan recusa o flag com exit 2 | Rotas do mesmo comando não compõem a mesma interface |
| P05 | Facts de spec sem components, index com check de outro módulo e digest distinto | Check estrangeiro aparece em checks; só limitation de change set ausente | Falta distinguir resultado associado de candidato não vinculado |
| P06 | Unknown flag, since inválido, recent inválido | Exit 0, JSON com todos os registros | Consulta amplia silenciosamente em vez de revelar input inválido |
| P07 | Start apply/status juntos; close apply/plan em ordens diferentes | Start retorna status draft sem aplicar; close depende do último modo | Argumentos ambíguos tratados por prioridade/ordem |
| P08 | Continuous status recebe `--json` | Exit 0, `continuous.active=false` em texto | Flag ignorado; se não suportado, deve ser recusado explicitamente |
| P09 | Assessment duas vezes, mesma árvore/engine/matrix | Segunda operação reused, state de componente preservado | Controle positivo de reuso; não mede ausência total de trabalho |
| P10 | Criar minimal e full com template atual | Minimal 5.493 bytes/5 headings; full 6.966 bytes/7 headings | 1.473 bytes a menos; não prova redução equivalente de esforço humano |
| P11 | Suggest aponta backend-go inexistente no fixture | rules_effective inclui a regra, exit 0, sem availability | Trilha declarada não demonstra regra disponível/aplicável |
| P12 | Review policy ilegível para Attention | Consulta exit 0, complete false, attention incomplete true | Controle de disclosure: sucesso da consulta não é coverage completa |
| P13 | Context e start preview | Um JSON válido por chamada, digests e autoridade explícitos | Controle positivo de automação |
| P14 | Project-state ausente | Pass, initialized false e disclosure additive | Não força inicialização como cerimônia de toda sessão |
| P15 | Mesmo estado via CLI e servidor MCP stdio real | Mesmos terminal false e next action; catálogo sem native close plan | Paridade da leitura funciona; handoff de planejamento ainda tem gap |

Os 15 cenários não representam 15 defeitos independentes. P02–P03 têm o mesmo ramo curto para specs `done`, além do problema distinto de saída. P07 cobre dois parsers com a mesma classe de ambiguidade. P08 é falta de rejeição de um flag não anunciado para essa subrota, não promessa de JSON já documentada. P05 não demonstra que um gate aceitou o check como evidência válida. P02 não afirma que `closeout-check` foi contornado: esse gate recusou corretamente a fixture.

A saída `closed` de P03 deve revelar no-op de lifecycle separado de verificação atual. É possível preservar idempotência sem marcar critérios como validados apenas porque `status: done` existe. Também não aplicar policies novas retrospectivamente a scopes legacy: usar o verificador que já conhece o contrato aplicável. [E01–E04]

## 7. Fricções nas instruções e na descoberta

### 7.1 Reuso entregue, mas procedimento ainda manda repetir

As skills de feature e spec-closeout, e AGENTS, usam `assess discover --if-stale`. Os workflows de feature em inglês e pt-BR, incluindo ambas as cópias distribuídas, ainda mandam executar discovery incondicional no início e no fechamento. O teste `TestNoWorkflowAlwaysRunsDiscovery` inspeciona AGENTS/skills, mas não esses workflows. R4 de `pose-adaptive-assessment-freshness` exige a mesma regra em todos. É gap real de integração documental, não falta de engine de reuso. [E18, E19, E30–E36, E44, E50]

A workflow lista 20 passos; a skill de feature lista 12 e a de closeout 14, além da trilha preferida. Esses números não são o total de comandos necessários: muitos passos são raciocínio, condicionais e alternativas. Somá-los produziria uma métrica falsa. O problema é o leitor precisar determinar sozinho quais partes são obrigatórias, alternativas ou já satisfeitas pelo plano.

Manter um caminho recomendado e procedimentos manuais de diagnóstico vinculados, sem duplicar a sequência inteira em fontes que divergem. Tornar obrigatoriedade/condição explícita e alinhar root, locale e scaffold. O teste deve validar a jornada distribuída inteira e os comandos sugeridos, não apenas a presença do flag em seis arquivos selecionados.

### 7.2 Trilha declarada versus aplicabilidade

Na baseline, `suggest feature --path pose-mcp --json` inferiu domain `go` e retornou apenas security/documentation-style. O task-map usa `backend-go`; a inferência converte `backend`, mas não o domain `go` declarado no repo-map atual. Na chamada explícita backend-go, a trilha inclui backend-go mesmo sem a regra root instalada. O mecanismo de extensões já existe; a UX deve resolver disponibilidade e sugerir aquisição governada, sem apontar um arquivo ausente como regra efetiva. [E10, E37]

Não codificar cegamente que todo domain vira Go: preservar taxonomia, aliases e source. A decisão sobre instalar uma extensão é diferente de descobrir uma regra aplicável. Exibir uma recomendação não instala pacotes nem implica autorização para essa alteração.

### 7.3 Governança sem pedidos artificiais

ActionRequest é pertinente quando a resposta material muda scope, risco, autoridade ou operação e nenhuma autorização existente a cobre. As skills já dizem isso. Não converter cada escolha reversível, toda sugestão lexical ou pending criterion em pergunta à pessoa. As frentes 2–4 definem assurance, delegação e resolução; esta frente trata encaminhamento e clareza da próxima ação, sem criar backlog paralelo.

A skill de closeout exige confirmação para spawned/covered/duplicate; o root exige consentimento para contribution staging. Já existe precedência e a autorização do usuário persiste. Recomendar instruções que distingam decisão nova de decisão já autorizada e sugiram propostas em lote. Não retirar critérios de autoridade nem repetir a mesma confirmação em toda retomada.

O comando preencher templates de attestation ajuda a evitar tentativa e erro, mas não deve induzir aprovação pronta sem leitura. Evidência mecânica deve ser derivada e judgment deve conservar conclusão material e suas limitações. A integração autônoma de revisão delegada continua pertencendo à frente 3. “Never answers judgment” no executor atual não significa “uma pessoa sempre precisa responder”; significa que a conclusão deve vir do ator competente pelo contrato. [E02, E31, E32]

## 8. Dados para UX e economia operacional

Facts devem conservar fonte, revision, freshness, scope e limites. O generator atual não inventa rationale, mas pode associar checks não vinculados quando components estão vazios; também lê o index sem revalidar sua freshness. Separar `associated checks` de `unscoped indexed results`. Isso evita copiar o output como prova do que a spec executou. Não exigir rebuild de index em toda leitura: expor input binding e degradar honestamente. [E08, E09]

Nos passos do plano, distinguir operation completed, already satisfied, waiting reviewer, waiting external e failed. Exit 3 para judgment é estado esperado que deve conservar pending IDs, comando válido e possibilidade de continuação; stderr/exit não substituem um documento de resultado consumível por agente. A telemetria atual classifica códigos não zero de non-gates como failed; `close` não está na lista de gates examinada, e não há signal específico nesse executor. Assim, analisar exit 3 como falha genérica distorce fricção. Esse efeito de classificação foi confirmado por leitura, não reproduzido com um journal de usage ativado. [E02, E49]

A UX de Attention já limita residual human-readable a dez itens e aponta a consulta completa; JSON conserva o conjunto. Não ocultar coverage para deixar a tela mais limpa. O agrupamento `gates` pode conter pendências consultivas, e o renderer mostra `restricts execution/advisory` mesmo sem bloqueio. Reconciliar vocabulário, ordenação e ação sugerida; não alterar efeitos do domínio por apresentação. A frente 4 mantém ownership da semântica das obligations. [E11, E12]

O preset mínimo reduziu cerca de 21% dos bytes deste template, mas não existe evidência nova de que reduziu 21% do tempo de trabalho. Continuam necessários intenção, requirements, trace e decisões materiais. Uma view progressiva deve mostrar conteúdo pelo momento e materialidade, conservar rationale existente e priorizar o que falta sobre boilerplate. Evitar formatos paralelos que invalidam o lint do documento canônico.

## 9. Pilotos, métricas e evidência de excelência

O relatório de pilot é honesto: rehearsal automatizado, sem uma pessoa responder. O JSON histórico registra stop/go pending; um append no relatório menciona decisão/adoption posterior no Harne8. Não tratar esses registros de snapshots diferentes como estado operacional único nem reabrir uma decisão já respondida. A consulta atual de requests é a fonte para estado, e o relatório deve apresentar a cronologia e os limites de cada evidência. [E39, E40]

O quickstart possui um harness que verifica recusas esperadas, start, resposta assinada, trace e o comando de attest emitido pelo closeout. É melhor evidência que screenshot estático ou guia sem execução. Mas preencher a conclusão de review por script não mede capacidade de uma pessoa julgar, nem prova ergonomia de um agente sem o script. A medição em ambiente limpo declara excluir setup de imagem/pacotes e leitura/desenvolvimento humano; preservar esse disclosure. [E41, E42]

Nesta baseline, Attention retornou coverage completa com duas dependencies, uma evidence advisory, um judgment e 108 residual-debt. Há um item for_actor, três no agrupamento gates e restrições de start/execution/closeout. Isso demonstra projeção, não prioridade adequada, custo cognitivo baixo ou satisfação do operador. Não usar redução do número de residuals como produtividade. [E11, E12]

Medir separadamente: tempo de comando; tempo de operador; calls necessárias por tarefa; trocas CLI/MCP; recusas úteis e artificiais; perguntas materiais e redundantes; retrabalho por invalidation; quantidade de texto/boilerplate; taxa de retomada sem reconstrução manual; resultado entregue com coverage. Não juntar tudo em um score de eficiência/qualidade.

Pilotar ao menos: mudança trivial autorizada; bug com regressão; alteração de contrato em dois módulos; judgment por agente independente; exceção humana de risco; operação externa; retomada depois de source mudar; task cross-project. Um piloto humano deve registrar o que a pessoa conseguiu resolver sem conhecimento interno. Um piloto de agente deve registrar as tentativas e loops, sem atribuir seu esforço à pessoa simulada. Fixar corpus/revision/engine para comparar desempenho; a frente 8 fará as medições de escala.

## 10. Backlog proposto e aceite

As prioridades são propostas; conservar contratos históricos e reaproveitar specs existentes. Não abrir automaticamente um novo artifact para cada linha da tabela.

| ID | Prioridade | Entrega | Critérios de aceite |
|---|---|---|---|
| F07-01 | P1 | Closeout semântico consistente | P02 distingue done/no-op de verified/terminal; plano e closeout compartilham verificador aplicável; legacy não sofre enforcement retrospectivo; mensagem não certifica passos não executados |
| F07-02 | P1 | Contrato de automação no executor | P03 emite um JSON válido em success/wait/failure quando solicitado; inclui step/result/pending/next action; human output e machine output derivam do mesmo resultado; P08 implementa ou recusa flag explicitamente |
| F07-03 | P1 | Parsers sem ampliação silenciosa | P06 rejeita flags/datas/números inválidos; P07 rejeita modos incompatíveis; aliases documentados seguem funcionando; invalid args não produzem um conjunto falsamente completo |
| F07-04 | P2 | Contexto e handoff de superfície | P04 explica compatibilidade de plan/context ou usa route comum; MCP oferece planejamento read-only ou handoff tipado explícito; P15 preserva paridade da leitura; writes mantêm assurance/authority |
| F07-05 | P1 | Fatos com associação e freshness | P05 mostra check não vinculado como candidato/limite, não prova da spec; inclui binding e status do index; conteúdo semântico não é gerado; scoped checks adequados continuam reutilizáveis |
| F07-06 | P1 | Instruções e regras executáveis | Workflows EN/pt-BR/scaffold usam mesma freshness; teste cobre arquivos distribuídos; suggest normaliza aliases e revela regra ausente/extension-needed; não instala automaticamente |
| F07-07 | P2 | Percursos proporcionais únicos | Rotas para trivial/bug/contract/decision são condicionais; spec existente é reutilizada; handoff é criado quando necessário; manual closeout aparece como diagnóstico; review delegado segue frente 3 |
| F07-08 | P2 | Priorização e requests sem cerimônia | Próxima ação revela fase, target, responsible actor e autorização existente; não pergunta novamente após replay/retomada sem mudança material; propostas de triagem podem ser revisadas em lote |
| F07-09 | P2 | Resultados operacionais e métricas | Waiting exit 3 não é failed genérico; distinguish query success de readiness/coverage; contar trabalho gerado/reusado e causa de rework; não transformar ausência de evento em tempo zero |
| F07-10 | P2 | Piloto de UX e contrato adversarial | Executar os oito percursos acima com pessoa/agente separados; medir corpus fixo; critérios de stop/go e rollback explícitos; regressões incluem instruções e output de erro, não só casos felizes |

F07-01–02 continuam recoverable closeout e machine-channel; F07-03 completa parsers dos comandos usados no cotidiano. F07-05 continua progressive spec facts, sem assumir adequação de evidência pela sua presença. F07-06 fecha integração pendente de adaptive assessment e rule delivery. F07-07–10 consolidam o roadmap `governance-efficiency`, corpus e pilot existentes. As frentes de confiança, delegação e obligations mantêm as correções semânticas já propostas.

## 11. Composição de entregas e alternativas

**Entrega A — contratos coerentes no caminho recorrente.** F07-01–06. Corrigir green/no-op enganoso, JSON ignorado, filtros silenciosos e instruções contraditórias. Acrescentar testes de integração de parsers e de comandos sugeridos, incluindo erros, retomada e stdout/stderr. Não redesenhar todos os comandos nem acrescentar outro gate obrigatório ao adotante.

**Entrega B — percurso proporcional e eficácia observada.** F07-07–10, depois dos fundamentos de confiança/delegação das frentes 2–4. Reusar a trilha do motor e preencher uma representação comum de próximo passo, com o menor conjunto necessário de ações e limites explícitos. Pilotar tarefas pequenas e materiais; não criar scheduler no core. Versões definitivas ficam para a frente 10.

| Alternativa | Benefício | Decisão/limite |
|---|---|---|
| Somente melhorar mensagens/cores | Baixo custo e melhor leitura | Insuficiente para inconsistência de resultado; renderer existente deve ser aproveitado |
| Um wizard obrigatório para tudo | Descoberta centralizada | Aumenta cerimônia e esconde autoria; evitar |
| Um front door opcional que projeta o próximo passo | Reduz busca, conserva CLI/MCP | Recomendado como composição read-only dos contratos, depois de corrigir sua semântica |
| Duplicar plano/executor no MCP | Paridade aparente | Evitar; expor domínio comum e manter writes proporcionais à identidade do transporte |
| Reduzir checks indiscriminadamente | Menos comandos | Perde cobertura; reutilizar evidência adequada e validação por materialidade |
| Exigir intervenção humana em todo judgment | Fácil regra de apresentação | Contraria autonomia por policy; usar reviewer competente e requests nas exceções |

Uma nova façade não deve acrescentar fonte de verdade nem renomear estado incompatível. Deve referenciar a spec/obligation/plan atual, expor alternativas admissíveis e o menor passo necessário, e permitir abrir detalhes. A ação recomendada pode ser emitir evidência, corrigir source, consultar reviewer, responder request ou continuar trabalho autorizado. Quando o engine não sabe, revelar unknown; não inventar um plano autônomo pelo fato de não ter encontrado blocker.

## 12. Fronteiras e validação

Frente 2: identidade, assurance e evidência até consumidores. Frente 3: autonomia e fulfillment do review delegado. Frente 4: requests, efeitos e satisfação. Frente 5: classificação/encaminhamento do residual. Frente 6: suficiência do cut. Aqui ficam contratos de jornada, descoberta e ergonomia, com referências cruzadas em vez de backlog duplicado.

Frente 8: custo algorítmico, benchmarks, runtime residente e reuso por operação. Frente 9: concorrência, crash, checkpoints e integridade de writes. Frente 10: deduplicação e releases. O executor de closeout chama validate tolerante sem module filter nessa rota; isso sugere trabalho potencialmente fora do escopo, mas não foi medido/reproduzido nesta frente. Não reduzir a validação antes de confrontar o plano de coverage; investigar com a frente 8.

O harness final foi executado em 15 cenários, incluindo MCP stdio real; seus 14 primeiros cenários também foram executados antes dessa extensão. JSON, links locais, 50 hashes de fontes e sintaxe Python foram conferidos. `pose check --strict` passou na baseline e após os documentos: `Result: SUCCESS — (strict mode) with 30 warning(s).` Os mesmos 30 avisos preexistentes foram mantidos; comando, resultado e hash do log constam no JSON. A suíte Go e estudos humanos não foram executados. O diff entrega somente relatório, evidências, harness e índice.

## 13. Fontes vinculadas à baseline

- **E01** — [pose-mcp/internal/pose/closeout_plan.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/closeout_plan.go)
- **E02** — [pose-mcp/internal/cli/closeout_plan.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/closeout_plan.go)
- **E03** — [pose-mcp/internal/pose/review_closeout.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/review_closeout.go)
- **E04** — [pose-mcp/internal/cli/review_closeout.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/review_closeout.go)
- **E05** — [pose-mcp/internal/cli/start.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/start.go)
- **E06** — [pose-mcp/internal/pose/start.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/start.go)
- **E07** — [pose-mcp/internal/cli/specs_cmd.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/specs_cmd.go)
- **E08** — [pose-mcp/internal/pose/spec_facts.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/spec_facts.go)
- **E09** — [pose-mcp/internal/pose/delivery_integrity.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/delivery_integrity.go)
- **E10** — [pose-mcp/internal/cli/suggest.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/suggest.go)
- **E11** — [pose-mcp/internal/cli/state_attention.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/state_attention.go)
- **E12** — [pose-mcp/internal/pose/attention.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/attention.go)
- **E13** — [pose-mcp/internal/cli/output.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/output.go)
- **E14** — [pose-mcp/internal/cli/cliout/record.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/cliout/record.go)
- **E15** — [pose-mcp/internal/cli/cliout/profile.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/cliout/profile.go)
- **E16** — [pose-mcp/internal/cli/help_catalog.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/help_catalog.go)
- **E17** — [pose-mcp/internal/cli/state.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/state.go)
- **E18** — [pose-mcp/internal/pose/assessment_freshness.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/assessment_freshness.go)
- **E19** — [pose-mcp/internal/cli/assess.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/assess.go)
- **E20** — [pose-mcp/internal/pose/reuse_equivalence.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/reuse_equivalence.go)
- **E21** — [pose-mcp/internal/mcpserver/server.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/mcpserver/server.go)
- **E22** — [pose-mcp/internal/mcpserver/catalog.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/mcpserver/catalog.go)
- **E23** — [pose-mcp/internal/bootstrap/bootstrap.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/bootstrap/bootstrap.go)
- **E24** — [pose-mcp/internal/pose/agent_project_context.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/agent_project_context.go)
- **E25** — [pose-mcp/internal/cli/project_context.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/project_context.go)
- **E26** — [pose-mcp/internal/cli/closeout_plan_test.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/closeout_plan_test.go)
- **E27** — [pose-mcp/internal/cli/output_contract_test.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/output_contract_test.go)
- **E28** — [pose-mcp/internal/cli/spec_surface_test.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/spec_surface_test.go)
- **E29** — [pose-mcp/internal/cli/adversarial_corpus_test.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/adversarial_corpus_test.go)
- **E30** — [pose-mcp/internal/cli/assessment_freshness_workflow_test.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/assessment_freshness_workflow_test.go)
- **E31** — [.agents/skills/pose-feature/SKILL.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.agents/skills/pose-feature/SKILL.md)
- **E32** — [.agents/skills/pose-spec-closeout/SKILL.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.agents/skills/pose-spec-closeout/SKILL.md)
- **E33** — [.pose/workflows/feature.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/workflows/feature.md)
- **E34** — [locales/pt-BR/.pose/workflows/feature.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/locales/pt-BR/.pose/workflows/feature.md)
- **E35** — [pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md)
- **E36** — [pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/feature.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/feature.md)
- **E37** — [.pose/indexes/task-map.json](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/indexes/task-map.json)
- **E38** — [.pose/roadmaps/governance-efficiency.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/roadmaps/governance-efficiency.md)
- **E39** — [.pose/reports/pose-agency-readiness-pilot.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/reports/pose-agency-readiness-pilot.md)
- **E40** — [.pose/results/pose-agency-readiness-pilot.json](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/results/pose-agency-readiness-pilot.json)
- **E41** — [tests/quickstart/first-governed-loop.sh](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/tests/quickstart/first-governed-loop.sh)
- **E42** — [tests/quickstart/measure-clean-environment.sh](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/tests/quickstart/measure-clean-environment.sh)
- **E43** — [.pose/specs/2026-10-04-pose-progressive-spec-surface.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/specs/2026-10-04-pose-progressive-spec-surface.md)
- **E44** — [.pose/specs/2026-10-04-pose-adaptive-assessment-freshness.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/specs/2026-10-04-pose-adaptive-assessment-freshness.md)
- **E45** — [.pose/specs/2026-10-04-pose-recoverable-closeout-plan.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/specs/2026-10-04-pose-recoverable-closeout-plan.md)
- **E46** — [.pose/specs/2026-10-04-pose-agency-readiness-pilot.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/specs/2026-10-04-pose-agency-readiness-pilot.md)
- **E47** — [.pose/specs/2026-10-04-pose-mechanization-adversarial-corpus.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/.pose/specs/2026-10-04-pose-mechanization-adversarial-corpus.md)
- **E48** — [pose-mcp/internal/pose/governance_waits.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/pose/governance_waits.go)
- **E49** — [pose-mcp/internal/cli/usage.go](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/pose-mcp/internal/cli/usage.go)
- **E50** — [AGENTS.md](https://github.com/oseiaspereira88/pose/blob/a0398f7faadeed14256efc03edab9d63041bcb58/AGENTS.md)
