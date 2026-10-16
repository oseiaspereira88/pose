# B12 — review independente de resultados operacionais

Data: 2026-10-16. Decisão: **changes-requested**. PR #186, branch
`impl/pose-b12-operation-results`, HEAD revisado
`e17e44ea72c47b6ef8761086aa17045026675ad3`.

Revisor autorizado: `agent:independent-gpt-6.1-sol-review`, execução Codex/OpenAI.
Implementador: `agent:claude-opus-5-5`, sessão Claude/Anthropic
`296d1fcb-e989-40ae-9362-cf3514db8b26`. São execuções de vendors diferentes.

Julgamento de scope/assumption-integrity: o prefixo `agent:independent-` é uma
declaração exigida pelo motor até a spec `pose-delegated-review-run-provenance`
substituí-lo. A identidade é declarada; no pose-dist as atestações não são
assinadas. Esse prefixo não prova independência cognitiva nem identidade
criptográfica. Nenhuma atestação foi criada nesta revisão.

## Escopo, método e regras aplicadas

Mudança mista de bugfix e contrato público. Foram consultados AGENTS.md, as
seções de review e closeout de POSE.md, workflow review, skills pose-review e
pose-spec-closeout, template, spec B12, ADR de resultados, relatórios das frentes
7/8/9, diff origin/main..HEAD e knowledge pertinente a bundles e assurance.

Regras aplicadas: security para argumentos, subprocessos e exposição de saída;
backend-go para erros, contexto e concorrência; documentation-style para
contratos distribuídos; delivery-evidence para não promover testes verdes a
entrega; knowledge-governance para os registros consultados. Não houve mudança
de policy, capability ou adoção. Não foram criados contribuição upstream ou
handoff de risco aceito: os defeitos abaixo continuam bloqueantes.

A CLI foi compilada desta branch com
`go -C pose-mcp build -o ../pose-review ./cmd/pose`. O arquivo resultante fica
na raiz do clone; por isso as invocações a partir da raiz usam `./pose-review`,
e a partir de pose-mcp usam `../pose-review`. Não foi usado o pose do PATH para
os comandos nativos de revisão. Não foram exportados POSE_PROJECT_ROOT,
POSE_PROJECT_ROOTS ou POSE_DEFAULT_PROJECT_ID. O harness F07 configura seu
próprio subprocesso, conforme seu código preexistente; os probes MCP adicionais
usaram HARNE8_PROJECTS_DIR e project_id explícito em fixtures sintéticas.

Evidência estruturada: [b12-independent-review.json](../results/b12-independent-review.json).
Fixtures e mutações foram descartáveis; arquivos de produção foram restaurados
byte a byte após cada mutação. Nenhum teste ou código de produção foi alterado
como resultado da revisão. A alteração preexistente em
`.pose/indexes/delivery-integrity.json` foi preservada.

## Findings bloqueantes

### F1 — medium: confirmação de evidência que não foi gravada

Local: `pose-mcp/internal/cli/closeout_plan.go:241`, confirmação na linha 245.
Requisitos: R1, R6 e R9; V6; D1 e contrato de efeitos do ADR.

O executor aceita exit 1 de cmdValidate como execução que produziu resultados,
registra evidence como done e confirma results-written antes de replanejar.
Mas validate também retorna 1 quando não consegue publicar o arquivo JSON
(`validate.go:798`), além de usá-lo para reprovação legítima de checks.

Reprodução: em fixture in-progress com critério unit, módulo Go e check local
`true`, configurar results_path como `.pose/results/blocked.json` e criar um
diretório nesse caminho. Executar `close spec:alpha --apply --reviewer
agent:reviewer --json`. O resultado termina em index failed, mas contém:

```json
{"steps_run":[{"id":"evidence","result":"done"},{"id":"index","result":"failed"}],"effects":[{"step":"evidence","effect":"results-written","ref":".pose/results/blocked.json"}],"unconfirmed":[{"step":"index","reason":"the step failed after it may have written; inspect it before retrying"}]}
```

O caminho continua sendo um diretório; nenhum documento foi publicado. Além da
confirmação falsa, a causa original da falha de publicação desaparece quando o
buffer é limpo e o erro posterior é atribuído ao index. O produtor deve separar
veredito negativo com resultado publicado de falha de publicação, conservar sua
causa e confirmar o efeito somente após observação do resultado. Acrescentar
regressão com falha real de escrita: o teste novo de efeitos apenas omite o
reviewer depois de um seal bem-sucedido e não cobre esta situação.

### F2 — medium: rotas JSON suportadas não retornam documento na falha

Locais: `pose-mcp/internal/cli/start.go:82`, 107, 125 e 137;
`pose-mcp/internal/cli/closeout_plan.go:72`;
`pose-mcp/internal/cli/specs_cmd.go:85`.
Requisitos: R2, R7 e R9; V1 e V8.

Uma spec inexistente produz exit 1, stdout vazio e somente erro humano em stderr
em `start spec:missing --json`, `start spec:missing --status --json`,
`start spec:missing --apply --digest bad --json` e
`close spec:missing --plan --json`. Falha de leitura do diretório de specs
também produz exit 1 e stdout vazio em `specs --json`; isso foi reproduzido
substituindo apenas na fixture o diretório por um arquivo.

São falhas de operação em rotas que suportam JSON, não recusas de flags
incompatíveis. Não há um documento JSON consumível como exige R2. Em contraste,
close apply já devolve o erro plan-unavailable em JSON. A preservação dos
documentos de sucesso por D2 é apropriada, mas não satisfaz o contrato de falha:
definir e renderizar a falha estruturada conservando os documentos legados de
sucesso. Os ramos são em parte preexistentes; o finding é uma lacuna de entrega
do requisito solicitado, não uma alegação de que todos surgiram neste diff.

### F3 — medium: pose_list_specs ainda amplia inputs que a CLI recusa

Local: `pose-mcp/internal/mcpserver/server.go:1001`, validações em 1008 e 1015.
Requisitos: R3, R7 e R9; V2 e V7; paridade declarada pelo ADR/changelog.

Na fixture com duas specs, chamadas reais JSON-RPC de pose_list_specs com
`{"recent":0}`, `{"since":""}` ou `{"sttaus":"done"}` devolvem
isError=false e total=2. A CLI recusa respectivamente `--recent 0`,
`--since ''` e `--sttaus done` com exit 2. O MCP perde a distinção entre número
omitido e zero explícito, ignora since vazio e usa json.Unmarshal permissivo
para nomes desconhecidos. `limit:0` também é tratado como não paginado, apesar
de a nova mensagem/ADR descreverem valor positivo.

Separar ausência de valor explícito, validar os mesmos domínios de entrada e
recusar argumentos desconhecidos sem listar dados. Preservar a chamada sem
filtro e a paginação omitida. Os testes novos só cobrem recent/limit negativos,
datas inválidas e um controle positivo, deixando esses caminhos descobertos.

### F4 — medium: specs facts contorna a validação de argumentos

Local: `pose-mcp/internal/cli/specs_cmd.go:15`, parser da subrota na linha 178.
Requisitos: R2, R3 e R7; V2/V3.

Com índice válido, `specs facts alpha --bogus --json` retorna exit 0 e texto
facts.*, enquanto `specs facts alpha --json --bogus` retorna exit 0 e JSON.
O dispatch para facts ocorre antes do parser estrito; a subrota só olha o
segundo argumento e ignora o restante. Assim uma flag desconhecida é aceita e
a posição de --json decide se o canal é máquina ou texto.

É outra lacuna preexistente na superfície de specs solicitada para a revisão.
Recusar flags desconhecidas/incompatíveis antes de derivar fatos e interpretar
--json sem depender da posição. A correção de listagem não protege essa rota.

## Execuções e sensibilidade dos testes

Passaram `go vet ./...`, `go test ./... -count=1` e `go test -race
./internal/cli ./internal/pose ./internal/mcpserver ./internal/usage`, em
pose-mcp. O quickstart executável `tests/quickstart/first-governed-loop.sh
--binary <CLI desta branch>` também passou todos os cinco capítulos, incluindo
as linhas closeout_plan.attest e closeout_plan.result consumidas pela página.

O harness F07 executou os 15 probes. Os exits e single_json registrados no
bloco after de b12-front07-probes.json coincidiram. P03 retornou
already-satisfied em apply/resume; P06/P07 recusaram os casos inválidos com 2;
P08 retornou JSON. Os onze controles mantiveram as propriedades registradas.

Cada teste novo de regressão foi exercitado sem sua mudança correspondente:
reversão temporária a origin/main de pose/cli.go, cli/closeout_plan.go,
cli/start.go, cli/specs_cmd.go, cli/review_closeout.go, mcpserver/server.go e
pose/spec.go; mutações dirigidas do construtor e das classificações de waiting
e duração desconhecida. Todos esses testes reprovaram. O controle
TestAGateThatExitsNonZeroInTimeIsStillAVerdict passou também no código antigo,
como deve acontecer com um controle de compatibilidade. Não se apresenta esse
controle como teste que distingue a implementação nova.

Como a reversão integral do executor faz o teste parar na primeira decodificação
JSON, foi feita uma segunda mutação isolada trocando apenas o outcome de no-op
por completed. TestCloseApplyJSONSeparatesCompletionFromAlreadySatisfied
reprovou especificamente na asserção de already-satisfied. A evidência JSON
lista os testes, resultados e mutações; as execuções verdes da suíte completa
foram anteriores às mutações e a suíte dirigida foi repetida após restaurá-las.

No transporte HTTP JSON-RPC real, um executável sintético que imprime uma linha
e dorme 35 s retornou em 32,018 s: isError=true,
error_code=execution_inconclusive, cause=deadline-exceeded, output parcial e
cleanup explícitos, sem campo passed. pose_usage observou errors=1, unknown=1,
fail=0, completed=0 e duration_samples=1. Cancelamento e prazo curto foram
exercitados pelos testes de domínio, incluindo a detecção de pipe mantido por
filho. Não foi alegado encerramento de toda a árvore de processos.

assess tech-debt encontrou zero marcadores e zero itens descobertos;
assess integrate avaliou 64 contratos, com 1 ativo e 63 consumer unobserved,
incluindo três endpoints HTTP, sem comprovar ausência de consumidor externo.
O conteúdo desses assessments permaneceu igual ao anterior, salvo baseline e
timestamp; são diagnósticos preexistentes, não 63 novos defeitos deste diff. recurrence-check
--tolerant --window-days 14 encontrou zero chaves acima do limiar.

## Julgamento de requisitos e decisões

R1/R6 têm a violação concreta F1. R2 tem F2/F4. R3/R7 têm F3/F4 e a lacuna de
falha JSON. R4 passou em waiting, pending IDs, comando de continuação e no-op;
o no-op não é promovido a verificação atual. R5 passou nas rotas subprocessadas
CLI/MCP: contexto antes de ExitError, inconclusivo tipado, saída parcial e limite
de cleanup. R8 passou em waiting, duration_unknown e duration_samples; o ADR
declara que leitores antigos rejeitam linhas waiting. R9 não está integralmente
atendido, pois os controles originais e a suíte verde não cobrem os falsificadores
adicionais de publicação/argumentos/erro demonstrados aqui. V6 em particular não
foi demonstrado pelo cenário novo de reviewer ausente.

D1 é a separação correta entre execução e veredito, porém sua implementação
não cumpre o contrato de efeito confirmado em F1. D2 preserva adequadamente
GateResult, CloseoutPlan e os documentos de start; a classificação minor das
recusas mais estritas e do no-op humano é deliberada. Não foi observado consumidor
documentado que dependa das combinações agora recusadas; o quickstart real passou.
Essa preservação de formato não justifica stdout vazio nas falhas previstas em R2.

O ADR declara honestamente a ausência de propagação cooperativa de contexto nas
chamadas de domínio em processo do CLI. A conclusão de R5 aqui é limitada aos
subprocessos pertinentes e ao MCP; terminar o processo CLI por sinal não é
evidência de cleanup/rollback interno. As rotas MCP de close plan/apply/resume
não existem, conforme o catálogo observado em P15; o plano técnico declara a
fronteira e não promete implementá-las. Não se atribuiu paridade de writes a
tools inexistentes. Nenhum desses limites corrige os quatro findings concretos.

Não foi alterado gate de governança que pudesse ser satisfeito apenas em forma;
o risco central é um recibo operacional afirmar um efeito inexistente. O probe
de publicação é um falsificador funcional que a suíte atual não contém.

## Estado nativo e encerramento da revisão

Bundle apenas preparado: `rvb-1440070659dc8146`, digest
`sha256:1440070659dc81466c7968264001053a879f8b18479c31eb9be5acbd2d9cedfd`.
Nenhum selo ou atestação. review verify terminou com exit 1,
state=ready-to-seal, fresh=false e approved=false. lint-spec --strict passou,
mas informa trace.entries=0 e trace.missing=9 para a spec ainda in-progress;
isso não demonstra satisfação funcional nem habilita seu fechamento.

check --strict passou com exit 0 e 29 avisos; o log integral está na evidência
estruturada. A suíte dirigida após restauração também passou nos quatro pacotes.
validate --strict com regeneração de delivery-validation, index, surface-check,
auto-attest, seal e close da spec ficaram fora do caminho de aprovação: a
instrução direta manda não selar nem corrigir quando há finding medium ou maior.
As demais ferramentas do plano não recebem dispositions de aprovação fictícias.

Spec, tasks, trace, Final Report, matriz de implementação e roadmaps não foram
marcados como entregues. Não houve commit, push ou merge. Os findings devem ser
corrigidos pelo implementador e submetidos a nova revisão antes do closeout.

## Second round

Data: 2026-10-16. Decisão: **changes-requested**. HEAD desta rodada:
`8cf3d3be79d927aea0abbdfcef5da7d289a478ad`; remediação examinada:
`e4ecdd74`. A primeira rodada acima foi preservada integralmente. A evidência
estruturada mantém seus campos originais e acrescenta `second_round`.

### Independência, escopo e método

Revisor: `agent:independent-gpt-6.1-sol-review`, autorizado pelo maintainer,
execução Codex/OpenAI distinta da execução Claude/Anthropic do implementador
`agent:claude-opus-5-5`, sessão `296d1fcb-e989-40ae-9362-cf3514db8b26`.
Julgamento de scope/assumption-integrity: o prefixo `agent:independent-` é uma
declaração exigida pelo motor até `pose-delegated-review-run-provenance`
substituí-lo; identidade declarada, atestações não assinadas no pose-dist.
O prefixo não constitui prova criptográfica de identidade ou independência.

Reaplicados AGENTS, workflow review, skills pose-review/pose-spec-closeout,
manual, template e regras security, backend-go, documentation-style,
delivery-evidence e knowledge-governance. Consultados novamente spec, ADR,
frentes 7/8/9, knowledge de bundles e diff completo contra origin/main.
Frontend e infraestrutura não se aplicam a este delta. `context` resolveu a
mesma autoridade local, sem transferência ou ambiguidade. `suggest review
--path pose-mcp` ainda seleciona security/documentation-style pelo domínio
`go`; backend-go foi aplicado explicitamente. Nenhuma extensão foi instalada.

A CLI foi compilada com o comando solicitado. O destino `../pose-review` de
`go -C pose-mcp build` é a raiz deste clone: a partir dela os comandos usam
`./pose-review`, e a partir do módulo usam `../pose-review`. Nenhum comando de
revisão usa o pose do PATH. Nenhuma das três variáveis de seleção de projeto
proibidas foi exportada pelo revisor. Os testes/harness existentes mantêm sua
própria configuração de fixtures; o probe MCP adicional usa
HARNE8_PROJECTS_DIR e project_id explícito, sem root/default override.

A varredura incluiu todo o executor de close, seus helpers e destinos de
stdout, rotas de start/specs/facts/continuous, parser e catálogo MCP,
subprocessos, journal/agregação, documentação distribuída e consumidores em
scripts, testes, workflows, skills, documentação e CI. As mutações temporárias
foram restauradas byte a byte. Não houve alteração final em produção ou testes.

### Disposição de F1–F4

| Finding original | Resultado desta rodada |
|---|---|
| F1 | Parcialmente corrigido: o diretório é recusado, mas um JSON anterior recente ainda gera confirmação falsa. Continua aberto abaixo. |
| F2 | Corrigido nos ramos originalmente enumerados: start preview/status/apply, close plan, specs e facts agora retornam um documento failed com causa. Uma rota adicional anunciada por help permanece descoberta: F6. |
| F3 | Corrigido para zero explícito, since vazio, campos desconhecidos e sort arbitrário não vazio; continuam os casos explícitos null descritos abaixo. |
| F4 | Corrigido: um slug e --json em ambas as posições; flags desconhecidas e argumentos extras recusados antes de consultar. |

O limite de 100000 dias fecha o overflow observado anteriormente. Os probes
reais MCP recusam 100001d e a entrada enorme da primeira rodada, e aceitam
100000d e data ISO. As regressões adicionadas em e4ecdd74 reprovam contra os
arquivos de produção anteriores a esse commit, conforme a matriz de mutações.

### Findings bloqueantes consolidados

#### F1 — medium, continua aberto: mtime recente não prova publicação desta execução

Local: `pose-mcp/internal/cli/closeout_plan.go:118`, confirmação na linha 295.
Requisitos: R1, R6, R9; V6; D1 e contrato de efeitos do ADR.

`observePublishedResults` aceita qualquer arquivo regular com JSON válido e
mtime dentro de um segundo antes do início do validate, sem observar se esta
execução conseguiu escrevê-lo. Na fixture da regressão, substituir o diretório
por um arquivo JSON preexistente somente leitura (0444), escrito imediatamente
antes de close. Validate retorna 1 com permission denied, os bytes permanecem
iguais, mas close registra evidence done e confirma results-written. O erro
posterior passa a ser do index, apagando a causa de publicação.

Repetido com um documento estruturado legítimo produzido por um validate
anterior bem-sucedido: o arquivo permaneceu byte a byte igual, mas effects
continha results-written e index-regenerated; a falha final foi
`evidence-insufficient`, relativa a change set, não `results-not-published`.
Um arquivo com mtime antigo e o diretório original são corretamente recusados;
a diferença observada depende apenas do timestamp do arquivo anterior.

Esperado: confirmar publicação por um fato vinculado à execução atual e
conservar sua falha de escrita; um arquivo anterior, ainda que válido e recente,
não autoriza confirmar o efeito. A regressão atual prova apenas o diretório,
não esse caminho de falha real de escrita com resultado anterior presente.

#### F3 — medium, continua aberto: null explícito ainda vira ausência de filtro no MCP

Local: `pose-mcp/internal/mcpserver/server.go:1000`, validações em 1012/1020/1026.
Requisitos: R3, R7, R9; V2/V7.

No transporte JSON-RPC real, `pose_list_specs` com `recent:null`, `limit:null`
ou `since:null` retorna isError=false e total=2, igual à chamada sem filtros.
Os ponteiros usados para distinguir omissão de zero também colapsam null em
nil. O catálogo anuncia integer/string, sem null, e não há validação de schema
antes do dispatch que recuse esses valores. A consulta aceita um valor inválido
explicitamente fornecido e amplia o conjunto.

Esperado: distinguir ausência de valor explícito inválido e recusá-lo antes de
listar; manter a chamada `{}` válida. O mesmo probe mostra `sort:null` e
`sort:""` aceitos, contrariando a recusa declarada de sort fornecido diferente
de date. O impacto de ampliação está comprovado pelos filtros null acima.

#### F5 — medium, novo: falha de start omite baseline já publicada e nega retomada

Local: `pose-mcp/internal/cli/start.go:130`, descarte do record na linha 133;
`pose-mcp/internal/cli/closeout_plan.go:132` inicializa a falha genérica.
Requisitos: R1, R2, R6; V6.

Em uma spec draft com atomic_start_version=1, obter o preview e tornar somente
`.pose/specs/` não gravável (0555), mantendo `.pose/starts/` gravável. Executar
start apply com seu digest e --json. ApplyStart publica `work.json` com
phase=baseline-recorded e baseline identificada, depois falha ao criar o temp
da transição da spec. O adapter descarta o StartRecord retornado junto com o
erro e emite:

```json
{"operation":"start.apply","outcome":"failed","steps_run":[],"effects":[],"resumable":false,"error":{"cause":"apply-refused"}}
```

Não há fase, efeito confirmado, unconfirmed ou próximo comando, embora a
baseline esteja persistida. Após restaurar a permissão, repetir exatamente o
mesmo digest completa o start com phase=started: a operação era retomável.

Esperado: conservar a fase e os efeitos observados do resultado parcial de
ApplyStart e indicar reconcile/resume seguro. Não alterar o documento legado de
sucesso; D2 não exige perder o recibo parcial na falha. Não presumir que toda
recusa de ApplyStart aconteceu antes de escrever.

#### F6 — medium, novo: a rota simples close anuncia --json e descarta a flag

Local: `pose-mcp/internal/cli/review_closeout.go:1576`;
contrato anunciado em `pose-mcp/internal/cli/help_catalog.go:469`.
Requisitos: R2, R3, R7; V1/V8.

O help alterado neste PR ainda anuncia
`pose close <scope|xref> [--expect-context <digest>] [--json]`. Sem
plan/apply/resume, cmdClose usa parseScopeCheckArgs, aceita --json e descarta o
booleano. Na fixture F07 context-plan-flags, `close spec:alpha --json` retorna
exit 1, stdout vazio e erro humano de elegibilidade em stderr. Não é recusa de
flag: a rota executa sua leitura e avaliação. O destino de sucesso também é
cmdCloseLocal, que imprime `Closed: ...` em texto.

Esperado: cumprir o JSON anunciado nessa rota em sucesso/falha, ou recusar a
combinação explicitamente antes da operação e reconciliar o help. É uma lacuna
preexistente na rota, mantida no contrato editado por esta entrega, não uma
regressão atribuída ao novo renderer de apply/resume.

#### F7 — medium, novo: catálogo MCP anuncia sort slug que o parser passou a recusar

Local: `pose-mcp/internal/mcpserver/server.go:1032` e `server.go:2361`.
Requisitos: R3, R7, R9; V2/V8; compatibilidade D2.

`tools/list` anuncia para pose_list_specs `sort: "date" ... or "slug"
(alphabetical)`. Uma chamada real com `sort:"slug"`, válida segundo esse
contrato e anteriormente atendida pela ordem padrão de ListSpecs, agora retorna
isError=true e `sort must be "date" when given`. O parser foi restringido, mas
o contrato entregue ao cliente continua recomendando uma invocação recusada.

Esperado: reconciliar a ordem documentada e a validação. Se a retirada de slug
é deliberada, explicitar a compatibilidade e atualizar o catálogo realmente
consumido; se deve ser preservada, aceitar e testar o modo. A classificação
minor do ADR não elimina essa contradição entre descoberta e execução.

### Evidência independente e sensibilidade dos testes

Passaram novamente go vet ./..., go test ./... -count=1 e go test -race nos
quatro pacotes tocados, dentro de pose-mcp. Passou o quickstart executável,
incluindo as linhas closeout_plan.attest/result consumidas pela documentação.
A suíte dirigida dos quatro pacotes passou novamente após todas as restaurações.

O harness F07 executou os 15 probes. P03/P06/P07/P08 mantêm os exits e a
propriedade single_json do bloco after versionado. Os onze controles foram
reexecutados e inspecionados: preview não escreve, done não significa review
verificado, context e start preview preservam documentos, policy ilegível mantém
coverage incomplete, state ausente mantém disclosure, avaliação repetida reusa
e MCP conserva a projeção read-only e a ausência de close plan/apply/resume.
O arquivo b12-front07-probes.json só guarda os quatro grupos alterados; não se
alega comparação byte a byte de controles ausentes nele. P05 e P11 continuam
controles de comportamentos cuja correção pertence a outras unidades.

As reversões seletivas a origin/main fizeram reprovar todos os testes novos
que distinguem esta implementação. O controle de gate nonzero concluído
passou também no código antigo. A primeira reversão de close não compilou
porque start/specs agora chamam writeOperationFailure definido nesse arquivo;
foi repetida conservando somente esse helper, e todos os testes de close
selecionados reprovaram por comportamento. A reversão integral de usage também
não compilou pela nova ExecutionOutcome; a mutação dirigida da classificação
waiting fez sua regressão reprovar. Essas falhas de compilação estão preservadas
na evidência e não são apresentadas como demonstração de sensibilidade.

A reversão anterior a e4ecdd74, conservando o helper quando necessário, fez
reprovar especificamente as regressões de publicação e JSON failures; start e
specs separadamente fizeram reprovar suas falhas JSON, facts reprovou por flags
e MCP reprovou pelos novos casos de input. Mutações isoladas de already-satisfied,
listas do construtor e agregação de duration_unknown fizeram suas regressões
reprovar. O teste de JSON query com timeout foi repetido isoladamente contra o
adapter antigo: a primeira seleção combinada excedeu o timeout da suíte após
os casos de deadline/cancelamento e o controle nonzero. Não se confunde esse
aborto com execução do caso faltante; a repetição isolada também reprovou.

No MCP stdio real, o gate sintético que imprime e dorme retornou em 32,031 s
como execution_inconclusive, cause deadline-exceeded, output parcial marcado
incompleto, cleanup explícito e nenhum passed. pose_usage registrou errors=1,
unknown=1, failed=0, fail=0, completed=0 e duration_samples=1. Deadline e
cancelamento também passaram no domínio. Não se alega encerramento da árvore
inteira de processos. Continuous status foi exercitado com seleção inválida e
scope inexistente: um documento failed em ambos. Os JSONs legados de plano,
start e GateResult foram preservados nos caminhos verdes inspecionados.

### Julgamento R1–R9, V1–V8, D1/D2 e limites

R1/R6 e V6 continuam bloqueados por F1 e F5. R2 e V1 têm a lacuna adicional F6,
embora as rotas originais de F2 tenham sido corrigidas. R3/R7 e V2/V7/V8
continuam bloqueados por F3, F6 e F7. R4/V4 passaram para waiting, pending IDs,
próximo comando e already-satisfied sem promoção a verificação atual.
R5/V5 passaram na fronteira subprocessada: contexto antes de ExitError,
inconclusivo sem verdict e limites de cleanup explícitos. R8 passou para
waiting e intervalos desconhecidos; duration_samples separa ausência de amostra
de tempo medido zero. O ADR declara o efeito em leitores antigos que rejeitam
waiting; não houve migração fictícia de journal nem inferência de esforço humano.
R9 permanece incompleto: os probes originais e os testes verdes não cobrem os
falsificadores adicionais acima. V3 passou para status JSON e recusa de flags
não suportadas de continuous; os controles de compatibilidade executáveis passam,
mas não demonstram a compatibilidade do catálogo contraditório de F7.

D1 continua apropriada como decisão, com implementação incompleta de efeitos.
D2 preserva corretamente contratos de sucesso e separa GateExecutionError de
GateResult; não autoriza ignorar --json anunciado nem descartar fase parcial.
O ADR declara honestamente o limite de contexto nas chamadas em processo do CLI;
a conclusão de R5 fica restrita aos subprocessos pertinentes e ao MCP, conforme
a fronteira técnica explícita. Não há promessa implementada de rollback ou
cleanup interno por sinal. As rotas MCP de close não existem e não foram
inventadas ou contadas como paridade de writes. Essa delimitação é aceitável;
os cinco findings concretos impedem aprovar esta entrega.

Não se alterou um gate de governança para aceitar satisfação meramente formal.
O falsificador mais barato do recibo de efeito é um JSON recente não gravável;
o teste atual não o impede. Não foi gerada atestação mecânica como substituto
de julgamento. Nenhum risco bloqueante foi aceito ou remetido automaticamente
a outra spec, e nenhuma contribuição upstream foi criada.

### Ferramentas nativas, estado de entrega e encerramento

Bundle apenas preparado: `rvb-db58701699784009`, digest
`sha256:db587016997840098131ac1070810a4bc457cd2fcfccddc6b702ffcbfa2a0f5a`.
Não foi selado. Nenhuma atestação foi criada. `review verify` retornou exit 1,
state=ready-to-seal, fresh=false, approved=false. `lint-spec --strict` passou,
mas ainda informa trace.entries=0 e trace.missing=9 para a spec in-progress;
isso não comprova requisitos. `check --strict` passou com 29 avisos.

`artifact-check --spec ... --strict` passou com claims=30 e observed=30,
atribuindo quatro commits até 8cf3d3be; os 608 avisos globais de orphan não são
608 defeitos novos de B12. Não foi usada a leitura de trailers do parser Git
como substituto da atribuição nativa. `surface-check --spec ... --strict`
retornou exit 1: stale-evidence e unconsumed-capability, por ausência de
integration com scoped provenance atual. O índice de entrega já estava alterado
antes desta rodada; este resultado descreve a evidência disponível, não um novo
defeito de produção. A validação do implementador 131/131 não foi promovida a
composição atual aprovada sem regenerar os resultados/index necessários.

`assess design` passou e observou ADR/governance-contract e o novo
validation-check. `assess tech-debt` encontrou zero marcadores descobertos;
`assess integrate` avaliou 64 contratos, 1 ativo e 63 consumer unobserved,
sem demonstrar ausência de consumidores externos. `recurrence-check --tolerant
--window-days 14` encontrou zero chaves acima do limiar. As saídas dessas
execuções estão na evidência. Os quatro arquivos gerados de assessments/state
foram restaurados, sem incluir assessment atualizado em commit.

Disposições operacionais do plano preparado, sem atestação: suggest-review,
assess-design, assess-integrate, assess-tech-debt e artifact-check foram
executados; surface-check executado e reprovado. Assess-discover foi dispensado
nesta revisão sem implementação/fechamento: a estrutura do componente não mudou
e discovery não responde aos falsificadores de comportamento. Validate global e
validate para pose-mcp não receberam disposição passed: a sequência estrita de
regeneração de delivery-validation pertence ao ramo condicional de aprovação,
interrompido pelos findings. Review-check e closeout-check continuam deferred,
pois exigem review-complete. Não foram executados index, seal, auto-attest,
attest, close ou assess discover --update-state para contornar essa situação.

A suíte dirigida foi repetida com os arquivos restaurados e passou nos quatro
pacotes. A diff final de produção e testes em relação a HEAD está vazia. A
alteração preexistente em delivery-integrity foi preservada. Spec, Requirements,
Technical Plan, tasks, trace, Final Report, matriz B12 e roadmaps não foram
marcados como entregues. Não houve commit, push, merge, alteração de policy,
capability ou adoção. O closeout continua pendente das correções e de nova
revisão independente com evidência atual.

## Third round

Data: 2026-10-16. Decisão: **changes-requested**. HEAD examinado:
`d65ac036d85387bf790a2db37ba82e2513c09c96`; correção examinada:
`7bfd098ede64d3d380005b26190c6f6e4b530433`. As duas rodadas anteriores
permanecem intactas. Evidência desta rodada: campo `third_round` de
`.pose/results/b12-independent-review.json`.

### Independência e regras aplicadas

Revisor autorizado pelo maintainer: `agent:independent-gpt-6.1-sol-review`.
Julgamento de scope/assumption-integrity: `agent:independent-` é uma declaração
exigida pelo motor até `pose-delegated-review-run-provenance` substituí-la.
Esta execução Codex/OpenAI é de outro vendor que o implementador
Claude/Anthropic, `agent:claude-opus-5-5`, sessão
`296d1fcb-e989-40ae-9362-cf3514db8b26`. A identidade é declarada; no
pose-dist as atestações não são assinadas. O prefixo não prova identidade nem
independência criptograficamente.

Mudança mista de bugfix, contrato e documentação. Reaplicados AGENTS, workflow
review, skills pose-review e pose-spec-closeout, manual, template e regras
security, backend-go, documentation-style, delivery-evidence e
knowledge-governance. Segurança rege integridade de inputs e efeitos; backend-go
rege contexto e erros; documentação e delivery-evidence regem contratos e claims;
knowledge-governance rege a leitura de decisões e riscos prévios. Frontend e
infraestrutura não são domínios deste delta. Consultados spec, ADR, relatórios
fonte das frentes 7/8/9, knowledge de bundles, histórico e diff contra origin/main.
`context` resolveu autoridade local, sem ambiguidade ou transferência.

A CLI foi compilada com `go -C pose-mcp build -o ../pose-review ./cmd/pose`.
O destino desse comando é `./pose-review` visto da raiz do clone, usado nas
operações nativas. Nenhum comando usa o pose do PATH; nenhuma das três variáveis
proibidas de seleção de projeto foi exportada pelo revisor. As configurações
internas de fixtures dos testes existentes não foram alteradas. Os probes MCP
usam HARNE8_PROJECTS_DIR e project_id explícito.

### Disposição das correções anteriores

| Finding | Verificação nesta rodada |
|---|---|
| F1 | O caso original de JSON recente 0444 agora é recusado como results-not-published, sem confirmar efeito. A varredura encontrou outro produtor da mesma linha de saída que mantém a confirmação falsa: F1 abaixo. |
| F2 / F4 | Permanecem corrigidos: falhas operacionais nas rotas JSON produzem um documento; facts valida slug, flags e posição de --json. |
| F3 | Corrigido nos casos anteriores: null explícito nos filtros, sort vazio, campos desconhecidos, zeros e since inválido são recusados. O MCP real também recusou components:null, cursor:null e status:null. |
| F5 | O cenário 0555 anterior agora revela baseline-recorded, deixa transition unconfirmed e é retomável depois de reparar a permissão, conforme a regressão executada. A fase posterior à transição contém uma promessa de retomada falsa: F8 abaixo. |
| F6 | Corrigido: close simples com --json é recusado com exit 2 antes da leitura; help e usage estão reconciliados. |
| F7 | Corrigido: sort:slug é aceito e ordena alpha antes de beta; a regressão executada confirma a ordem e o probe MCP real confirma a aceitação. |

### Findings bloqueantes consolidados

#### F1 — medium, permanece aberto: saída de check filho pode confirmar publicação que falhou

Local: `pose-mcp/internal/cli/closeout_plan.go:115`, confirmação de evidence no
mesmo executor; produtor da colisão em `pose-mcp/internal/cli/cliout/steps.go:143`
e `pose-mcp/internal/cli/validate.go:748`. Requisitos R1, R6, R9; V6; D1 e ADR.

O scanner usa TrimSpace em todas as linhas do buffer conjunto de validate.
Esse buffer também contém as últimas linhas de um check que falhou, indentadas
por Step.Detail. Portanto a premissa de que a linha só pode ser impressa após
a escrita bem-sucedida do validate não vale para o buffer que é inspecionado.

Reprodução sintética: na fixture publicationFailureFixture, manter um JSON
anterior em .pose/results/blocked.json com modo 0444. Configurar o check da
fixture como `sh` com argumentos `-c`, `printf "json: %s\n" "$1"; exit 1`,
`probe`, e o caminho absoluto desse arquivo. O check apenas imprime uma linha;
não escreve o arquivo. Validate inclui a linha indentada no detalhe da falha,
então sua própria escrita retorna permission denied. Close retorna um JSON,
mas declara evidence done e confirma results-written, atribuindo a falha ao
index em vez de results-not-published.

Repetido com resultado estruturado legítimo gerado por validate anteriormente:
os hashes anterior e posterior são iguais
`954c79b5936263d6314220228cfb768bc4fee35c9f9cc00fc3e78c950a6c7b92`;
mesmo assim effects contém results-written e index-regenerated. Não se exige
comprometer um gate para produzir o problema: logs de checks são dados de saída,
não recibos da escrita do comando pai. A evidência guarda ambos os cenários,
a matrix sintética e os resultados observados.

Esperado: distinguir o fato de publicação do próprio validate dos dados
reproduzidos de seus filhos. Recusar este cenário com results-not-published,
preservar evidence unconfirmed e não alegar regravação do documento anterior.
A nova regressão 0444 passa, mas não cobre essa colisão de proveniência textual.

#### F8 — medium, novo: start promete retomada que falha depois da transição

Local: `pose-mcp/internal/cli/start.go:200`, ramo de retomada em `start.go:119`;
fronteira no domínio `pose-mcp/internal/pose/start.go:200` e gravação final na
linha 277. Requisitos R1, R6, R9; V6; promessa de resume do ADR.

Em uma fixture draft com atomic_start_version=1, obter o digest do preview e
criar o diretório `.pose/starts/work.json.tmp`. Executar start apply com esse
digest e --json. A baseline é publicada e a spec muda para in-progress; a
publicação final do registro falha porque seu temporário é diretório.
O recibo confirma baseline-recorded, marca transition unconfirmed e afirma
resumable=true, instruindo repetir exatamente o mesmo --apply --digest.

Remover somente o diretório temporário e executar o comando recomendado.
A chamada retorna exit 1, cause=plan-digest-mismatch, mantém resumable=true e
repete a mesma instrução que acabou de falhar. A spec já mudou, então PreviewStart
produz outro digest; o atalho do adapter aceita somente phase=started, enquanto
o registro ficou em baseline-recorded. ApplyStart rejeita o digest antes de
reconciliar a transição já publicada. A evidência contém uma reprodução desde
o primeiro apply e outra partindo do baseline parcial de F5; ambas confirmam
este ciclo sem alteração concorrente ou commit intermediário.

A limitação de recuperação do domínio já existia; o defeito relevante desta
mudança é o novo recibo classificá-la como retomável e anunciar um comando que
não a retoma. Esperado: uma retomada comprovadamente suportada nessa fase, ou
um resultado que exponha a necessidade de reconcile sem prometer esse retry.
Não basta repetir o resultado verde do cenário anterior à transição.

### Validação independente e sensibilidade

Passaram `go vet ./...`, `go test ./... -count=1` e `go test -race` nos quatro
pacotes tocados, no módulo pose-mcp. Depois de todas as restaurações, a suíte dos
quatro pacotes passou novamente. Passou o quickstart com `--binary` apontando
para a CLI desta branch: julgamento exit 3, closeout_plan.attest e
closeout_plan.result continuam consumíveis pela página e pelo script.
Uma primeira invocação do harness sem a opção --binary foi descartada como
prova da CLI selecionada; a execução válida é a repetição explícita.

Foram executadas 16 mutações seletivas, sem falhas de compilação. Arquivos de
produção foram restaurados byte a byte em finally; testes não foram alterados.
Contra origin/main, reprovam os testes novos de close, parsers, continuous,
MCP e subprocessos, incluindo deadline, cancelamento e JSON query. O controle
de gate concluído não zero continua passando no adapter anterior, como deve.
Na reversão de close foi conservado apenas writeOperationFailure para manter
compiláveis os consumidores novos que dependem desse helper.

Contra a produção anterior a 7bfd098e, reprovam especificamente o teste de JSON
recente não regravável, o de start parcial 0555, o de close simples --json e os
de null/sort e slug no MCP. O controle positivo TestResultsThisRunPublishedAreConfirmed
passa também antes dessa última correção, porque a publicação bem-sucedida já
era confirmada; não se apresenta esse controle como regressão sensível à troca
de mtime por recibo. Mutações isoladas de already-satisfied, construtor, waiting
e agregação de duration_unknown também fazem suas regressões reprovar.

O harness F07 executou os 15 grupos. P03/P06/P07/P08 coincidem com after da
evidência versionada em exit_code e single_json. Os onze controles foram
inspecionados: preview sem write, done separado de gate verificado, recusa de
context flag não suportado, coverage incompleta explícita, reuso em avaliação
repetida, documentos legados de context/start e ausência de rotas MCP de close.
Não há comparação byte a byte com controles ausentes de b12-front07-probes.json;
P05/P11 continuam observações de outras unidades. A busca de consumidores em
scripts, skills, workflows, docs, testes e CI não encontrou outro consumidor
quebrado pelas recusas alteradas. JSON legacy de start/plan e GateResult é
preservado nos caminhos verdes; inputs inválidos continuam exit 2, sem JSON
falso de sucesso.

O probe MCP real via stdio parou o subprocesso lento em aproximadamente 32,013 s:
execution_inconclusive, deadline-exceeded, output parcial marcado incompleto,
cleanup explícito e ausência de passed. A telemetria registrou errors=1,
unknown=1, failed=0, fail=0, completed=0, duration_samples=1. Os testes de domínio
cobrem também cancelamento. Waiting não vira failed e duration_unknown não entra
na média; duration_samples distingue ausência de amostra de duração medida zero.
O ADR declara corretamente a rejeição de waiting por leitores antigos.

### Julgamento dos requisitos e decisões

R1/R6/V6 e a qualificação completa R9 continuam bloqueados por F1/F8.
R2/V1 passou nos caminhos válidos de sucesso, espera e falha exercitados:
stdout tem um documento e não recebe progresso do laço de close. Os caminhos
de publicação parcial também mantêm um documento, mas os fatos nele são os
findings acima. R3/R7 e V2/V7/V8 passaram nos parsers, aliases, documentos
legados e catálogo examinados. R4/V4 passou para pending, próximo comando,
waiting e already-satisfied sem revalidar gates retrospectivamente.
R5/V5 passou na fronteira subprocessada e MCP; R8/V7 passou no journal.
V3 passou para status JSON e recusa explícita de flags não suportadas.

D1 é apropriada, mas a prova de efeitos continua incompleta. D2 evita quebrar
contratos JSON estabelecidos e classifica corretamente as recusas como minor;
isso não autoriza recibo falso de publicação ou promessa falsa de retomada.
O ADR declara com honestidade o limite de contexto no domínio CLI em processo.
O aceite de R5 é limitado aos subprocessos pertinentes e MCP, conforme a
fronteira técnica; sinal não prova cleanup nem rollback. As rotas MCP de close
não existem e não foram contadas como paridade de writes. Não houve alteração
de policy, capability ou adoção, nem automação de julgamento.

Não foi introduzido novo gate de governança. O falsificador mais barato da
integridade do recibo é uma linha de check filho igual à confirmação esperada;
a suíte verde não impede essa satisfação apenas textual. Nenhum risco medium
foi aceito ou transferido para outra spec, e nenhuma contribuição upstream foi
criada sem consentimento.

### Ferramentas nativas e estado final

Bundle preparado, sem selo: `rvb-33630722b3148f48`; digest
`sha256:33630722b3148f48026202d04e576bac064186fc0e95c2bf8a7c7968ccd637e2`. O plano efetivo tem 15 critérios e 11 entradas de
ferramenta. Nenhuma atestação foi criada. `review verify` retornou exit 1,
state=ready-to-seal, fresh=false, approved=false. `close` não foi
executado. `lint-spec --strict` passou, mas trace.entries=0 e trace.missing=9
continuam explícitos na spec in-progress; lint não satisfaz R1–R9.
`check --strict` passou com 29 avisos.

`artifact-check --strict` passou com claims=30 e observed=30, seis commits
atribuídos até d65ac036; os 608 avisos globais de orphan não são findings novos
de B12. `surface-check --strict` retornou exit 1 com stale-evidence e
unconsumed-capability: falta integration ligada ao scoped provenance atual.
A evidência 131/131 do implementador não foi apresentada como regeneração atual
independente do delivery index. A preparação inicial encontrou índice sem change
set atribuído; artifact-check recompôs sua projeção e o plano final acima foi
lido com o subject observado. Isso não foi usado para contornar gate de aprovação.

Assess design passou; tech-debt encontrou zero marcadores descobertos;
integrate observou 64 contratos, um ativo e 63 sem consumidor observado;
recurrence-check encontrou zero chaves acima do limiar. As saídas integrais
estão na evidência. Suggest-review foi executado; backend-go foi aplicado
explicitamente além das regras selecionadas pelo domínio go. Assess-discover foi
dispensado nesta rodada sem implementação/closeout. Validate global e por módulo
não recebem passed: a validação estrita de entrega, index, seal, auto-attest,
attest, close e assess discover --update-state pertencem ao ramo de aprovação,
interrompido pelos findings. Review-check e closeout-check permanecem deferred
pela precondição review-complete. Surface-check foi executado e sua falha foi
registrada, sem disposição fictícia.

As projeções geradas de assessments/state e delivery-integrity foram restauradas
ao estado inicial deste turno. Produção e testes não têm diff em relação a HEAD.
Somente este parecer e a evidência das três rodadas são alterações de revisão.
Spec, tasks, trace, Final Report, matriz B12 e roadmaps permanecem sem marca de
entrega. Não houve commit, push, merge, alteração de policy/capability/adoção
ou submissão upstream. A entrega exige remediar F1/F8 e nova revisão independente.

## Fourth round

Data: 2026-10-16. Decisão: **changes-requested**. HEAD examinado:
`ed9cd2c056fdffc427b97675718d5f79b668846e`; correção examinada: `411fe645`.
As rodadas 1–3 permanecem intactas. Evidência desta rodada: `fourth_round` em
`.pose/results/b12-independent-review.json`.

### Independência, escopo e regras

Revisor autorizado: `agent:independent-gpt-6.1-sol-review`, execução
Codex/OpenAI, outro vendor em relação ao implementador Claude/Anthropic
`agent:claude-opus-5-5`, sessão `296d1fcb-e989-40ae-9362-cf3514db8b26`.
Julgamento de scope/assumption-integrity: o prefixo `agent:independent-` é uma
declaração exigida pelo motor até a spec `pose-delegated-review-run-provenance`
substituí-lo. Identidade declarada; no pose-dist as atestações não são assinadas.
O prefixo não constitui prova criptográfica de identidade ou independência.

Reaplicados AGENTS, manual nas partes de review/closeout, workflow review,
skills pose-review e pose-spec-closeout, template, spec, ADR, relatórios fonte
das frentes 7/8/9, knowledge de bundles/assurance e diff origin/main..HEAD.
Mudança mista de bugfix e contrato público. Regras: security (integridade,
autorização e exposição), backend-go (erros, contexto e concorrência),
documentation-style (contratos distribuídos), delivery-evidence (claims versus
prova) e knowledge-governance (decisões consultadas). Frontend e infraestrutura
não se aplicam. Context resolveu a autoridade local, sem ambiguidade.

A CLI foi recompilada desta branch com
`go -C pose-mcp build -o ../pose-review ./cmd/pose`; da raiz do clone o destino
é `./pose-review`, usado em todos os comandos nativos. Não foram exportadas as
três variáveis de seleção de projeto proibidas. Os harnesses preexistentes
configuram somente seus subprocessos de fixtures. Não houve nova delegação,
edição de produção/testes, alteração de policy/capability/adoção no clone real,
contribuição upstream ou aceite de risco bloqueante.

### Correções anteriores e sensibilidade das regressões

F1 foi corrigido nos cenários das rodadas anteriores: diretório no destino,
arquivo 0444 recente/antigo, inclusive check filho que imprime `json: <path>`.
O snapshot anterior é comparado com o JSON regular observado após validate,
com generated_at não anterior ao início da execução. O recibo textual deixou
de confirmar publicação. A regressão `TestACheckThatPrintsTheWriteReceiptDoesNotConfirmIt`
passa em HEAD e reprova revertendo somente closeout_plan.go para antes de
411fe645. O controle `TestResultsThisRunPublishedAreConfirmed` continua passando;
não se atribui sensibilidade a uma correção a um controle que também passa antes.

F8 foi corrigido para a falha original após a transição: manter
`.pose/starts/work.json.tmp` como diretório, aplicar, reparar somente esse
obstáculo e repetir o mesmo digest agora termina em phase=started. A repetição
posterior é um no-op. A regressão
`TestAStartThatFailsAfterMovingTheSpecResumesWithTheAdvertisedCommand` passa em
HEAD e reprova retirando a correção do adapter; também reprova mantendo a
assinatura de ResumeStart mas desativando sua recuperação no domínio. Um probe
separado mudou o requisito após a baseline: compare-and-swap-conflict agora tem
resumable=false. A nova retomada tem os defeitos adjacentes F9/F10 abaixo.

F2/F3/F4/F5/F6/F7 permanecem corrigidos nos casos anteriores: falha operacional
JSON, filtros null/zero/vazio e desconhecidos no MCP, facts estrito, start com
baseline parcial, recusa de close simples com --json e sort:slug anunciado.

### Findings bloqueantes consolidados

#### F9 — high: retomada ignora uma restrição vigente e inicia a spec

Locais: `pose-mcp/internal/cli/start.go:122` e
`pose-mcp/internal/pose/start.go:263`, publicação invocada na linha 280.
Requisitos afetados: R6, R7 e R9; V6/V8; preservação dos contratos governados
existentes, em particular pose-governed-effect-enforcement.

Cenário determinístico, sem corrida: em fixture draft com atomic_start_version=1
e agency_readiness_version=1, obter preview e digest. Impedir temporariamente a
escrita na pasta de specs (0555) e executar apply: a baseline é gravada e a
transição falha. Restaurar a permissão e abrir uma ActionRequest de aprovação
com efeito start:block. O preview atual mostra ready=false e
waiting_on.code=action-request-pending. Repetir apply com o digest original.

HEAD retorna exit 0, phase=started e publica status: in-progress apesar da
request ainda aberta e pendente. O adapter encaminha qualquer divergência de
digest com phase=baseline-recorded para ResumeStart, inclusive divergência
causada pela nova restrição, e o domínio usa PreviewStart somente para obter o
path: não verifica a restrição/readiness atual antes de finishStart publicar
uma transição ainda pendente. Não é um problema hipotético de CAS genérico ou
de sincronização entre processos: a restrição já existe antes do retry.

O mesmo probe com somente o adapter anterior a 411fe645, selecionado por overlay
Go sem editar o clone, retorna exit 1, plan-digest-mismatch e mantém draft.
Isso isola uma regressão da correção de F8. A request não foi respondida,
cancelada ou dispensada em nenhum dos dois casos. Evidência:
`fourth_round.governed_start` e `fourth_round.governed_start_prior_adapter`.

Esperado: revalidar as restrições e precondições pertinentes antes de publicar
uma transição pendente na retomada, preservando a recuperação de uma transição
que já ocorreu. Uma baseline persistida não pode virar autorização permanente.
Acrescentar regressão com request aberta entre a falha pré-transição e o retry.
O caso adversarial `stale-preview-authorizes-apply` já está catalogado; o teste
existente cobre a primeira aplicação, mas não este caminho de retomada.

#### F10 — medium: falha na retomada oculta o recibo na saída humana

Local: `pose-mcp/internal/cli/start.go:131` (resultado construído na linha 125).
Requisitos afetados: R2 e R6; V1/V6; D1 e a promessa de humano/máquina sobre o
mesmo resultado.

Depois de uma falha com work.json.tmp como diretório que já moveu a spec,
manter o obstáculo e repetir o apply original sem --json. ResumeStart volta a
falhar ao publicar o registro final. startApplyFailure constrói baseline-recorded,
transition unconfirmed, resumable=true e next_action, mas o ramo humano descarta
esses dados: stdout vazio e só `Error: pose start: open ...work.json.tmp: is a
directory` em stderr. Com --json, a mesma tentativa expõe todos esses fatos.

A pessoa perde a informação de publicação parcial e de como retomar justamente
na rota nova de recuperação. O ramo normal de falha do apply, mais abaixo no
mesmo arquivo, imprime effects e next_action; a duplicação introduzida pela
correção não o faz. Evidência: `fourth_round.additional_probes.start_partial`
com `human_retry_while_record_blocked` e `json_retry_while_record_blocked`.
Esperado: renderizar o resultado de falha também nesse ramo humano, preservando
fase/efeitos conhecidos e orientação de recuperação; cobrir a repetição que
continua falhando, além da repetição após o reparo.

### Varredura completa, checks e decisões

A varredura continuou após os findings: parsers e despacho, todos os ramos
novos de stdout JSON, laço de close e buffers, confirmação e incerteza por
fase, documentos legados, adapters de subprocesso/MCP, journal/agregação,
catálogo/help, distribuição EN/pt-BR, scripts, skills, workflows, docs, testes
e CI. Não foi encontrado outro defeito de severidade medium ou maior nesta
superfície. O close captura a saída dos comandos internos em buffers; os
caminhos de sucesso/espera/falha JSON exercitados produzem um documento, sem
progresso no stdout. F10 é perda de fatos no renderer humano, não contaminação
JSON. No-op continua separado de verificação atual.

Passaram `go vet ./...`, `go test ./... -count=1` e `go test -race` nos quatro
pacotes tocados, em pose-mcp. A suíte dos quatro pacotes passou novamente após
restaurar todas as mutações. Passou o quickstart com --binary explícito desta
branch: exit 3 e closeout_plan.attest/result continuam consumíveis pela página
e pelo script. A busca de consumidores não encontrou outro uso afetado pelas
recusas alteradas. Os documentos verdes de start, --plan --json e GateResult,
alias --component e --recent sem número continuam preservados.

Foram executadas 19 mutações seletivas: todas retornaram 1 por falhas de testes,
sem falha de compilação. Cobrem cada teste de regressão novo, inclusive
cancelamento/deadline, query interrompida, parsers, já satisfeito, construtor,
MCP e waiting/duration_unknown. O controle de gate concluído não zero passa
na produção anterior, como deve. A retirada de close conserva somente o helper
writeOperationFailure para os outros arquivos compilarem; a mutação do domínio
ResumeStart conserva sua assinatura. Os arquivos foram restaurados byte a byte.

O harness F07 executou os 15 grupos: P03/P06/P07/P08 coincidem com after da
evidência versionada em exit_code e single_json. Os onze controles foram lidos
separadamente, sem inventar comparação com controles ausentes daquele arquivo:
preview não escreve, done não prova gate, contexto não suportado é recusado,
coverage incompleta é explícita, discovery reutiliza inputs equivalentes,
start/context mantêm os documentos e não existe rota MCP de close apply.

O MCP real via stdio retornou execution_inconclusive/deadline-exceeded em
aproximadamente 32,031 s, sem passed, com output parcial incompleto e limite de
cleanup. O journal registrou errors=1, unknown=1, failed=0, fail=0, completed=0
e duration_samples=1. Cancelamento também está coberto no domínio. Waiting não
vira failed; duração desconhecida não entra na média; duration_samples distingue
uma amostra de zero de ausência de medição. O ADR declara que leitores antigos
recusam waiting; não promete compatibilidade retroativa desses eventos locais.

R1/R4/R5/R8 e V3/V4/V5/V7 passaram nas fronteiras qualificadas; R2/V1 é bloqueado
por F10, R6/V6 e R7/V8 por F9 e F10, e R9 permanece incompleto. R3/V2 passou nos
parsers e aliases examinados. A verificação da superfície não foi promovida a
entrega só porque a suíte ficou verde. D1 é uma boa separação de fatos, mas o
renderer de retomada ainda a perde. D2 preserva os documentos públicos e
classifica as recusas como minor de forma apropriada; não autoriza ignorar uma
ActionRequest. O ADR declara honestamente o limite de contexto dentro do CLI
em processo e de cleanup de descendentes; R5 foi qualificado para a fronteira
subprocessada/MCP pertinente. Não se atribui paridade de writes a rotas MCP de
close inexistentes. B13/B14/B35 não são usados para adiar esses dois defeitos.

### Ferramentas nativas e estado final

Bundle somente preparado: `rvb-05bbd70e9b5bfdb3`, digest
`sha256:05bbd70e9b5bfdb33fd0e2dad13252d600f7415c21adad28c49f2766c2865b16`. Plano lido com 15 critérios e 11 entradas de
ferramenta; nenhum selo ou atestação criado. Review verify: exit 1,
state=ready-to-seal, fresh=false, approved=false. Close não executado.
Lint-spec --strict: exit 0, mas trace.entries=0 e trace.missing=9 para a spec
in-progress; esse lint não satisfaz requisitos. Check --strict: exit 0, 29 avisos.

Artifact-check --strict: exit 0, claims=31, observed=31, oito commits atribuídos
até ed9cd2c0. Os 607 avisos globais de orphan não são 607 findings novos de B12.
Surface-check --strict: exit 1, stale-evidence e unconsumed-capability, por
integration sem scoped provenance atual. A evidência 131/131 do implementador
foi consultada, sem promovê-la a validação independente atual de composição.
A preparação antes de artifact-check reportou ausência de change set imutável;
a preparação acima usa a projeção nativa recomposta, sem contornar aprovação.

Assess design, tech-debt, integrate e recurrence-check foram executados:
zero marcadores descobertos; 64 contratos, um ativo e 63 consumer unobserved;
zero chaves acima do limiar de recorrência. Consumer unobserved é limitação de
observação, não prova de inexistência de consumidores externos. Suggest-review
executado para pose-mcp. Assess-discover foi dispensado como recomendado nesta
revisão sem implementação/closeout; não responde aos falsificadores funcionais.
Validate global e por módulo não receberam passed: a regeneração estrita de
delivery-validation, index, seal, auto-attest, attest e close é condicional à
aprovação, interrompida por F9/F10. Review-check e closeout-check seguem deferred
pela precondição review-complete. Não foi executado discover --update-state.
As disposições por entrada/componente e as saídas estão na evidência.

As projeções de assessments/state e delivery-integrity produzidas pela revisão
foram restauradas ao estado inicial. Produção e testes não têm diff. Somente o
parecer e a evidência das quatro rodadas permanecem não commitados. Requirements,
Technical Plan, tasks, trace, Final Report, matriz B12 e roadmaps não foram
alterados ou marcados como entregues. Nenhum risco bloqueante foi aceito, nenhum
handoff de risco aceito ou contribuição upstream foi criado. Não houve commit,
push ou merge. O closeout exige remediar F9/F10 e nova revisão independente.

## Fifth round

Data: 2026-10-16. Decisão: **changes-requested**. HEAD revisado:
`62331ed11fa00b36beed769c257d633aa5c4db1d`; correção examinada:
`c462fc551e217d72454a18ec9435017e74b5ba26`. As rodadas 1–4 acima permanecem
intactas; a evidência acrescenta `fifth_round` sem substituir seus registros.

### Independência, escopo e regras aplicadas

Revisor autorizado: `agent:independent-gpt-6.1-sol-review`, execução
Codex/OpenAI, outro vendor em relação ao implementador Claude/Anthropic,
`agent:claude-opus-5-5`, sessão `296d1fcb-e989-40ae-9362-cf3514db8b26`.
Julgamento scope/assumption-integrity: o prefixo `agent:independent-` é uma
declaração exigida pelo motor até a spec `pose-delegated-review-run-provenance`
substituí-lo. Identidade declarada; no pose-dist as atestações não são
assinadas. O prefixo não comprova identidade criptográfica ou independência
cognitiva.

Reaplicados AGENTS.md, partes de review/closeout de POSE.md, workflow review,
skills pose-review e pose-spec-closeout, template e regras security,
documentation-style, delivery-evidence e knowledge-governance. Aplicados os
critérios de backend Go do plano; nenhuma extensão ou policy foi instalada.
Frontend e infraestrutura não se aplicam. Relidos spec, ADR, frentes 7/8/9,
knowledge pertinente e diff integral origin/main..HEAD. Context resolveu a
autoridade local. CLI compilada da branch pelo comando solicitado; a partir
da raiz o executável produzido é `./pose-review`, e a partir de pose-mcp é
`../pose-review`. Não foi usado pose do PATH nem exportada qualquer das três
variáveis de seleção de projeto proibidas. Fixtures e harnesses existentes
mantêm suas próprias configurações de subprocessos.

### Disposição de F9/F10 e testes de sensibilidade

F9 **corrigido**: `ResumeStart` só conclui o digest after já publicado.
Quando a spec permanece no before, a recusa transition-not-published cai no
plano atual; o probe 0555 + ActionRequest start:block manteve a spec draft,
com exit 1. `TestARetriedStartDoesNotOutliveARequestThatNowRestrictsIt` passa
no HEAD e falha restaurando somente `internal/pose/start.go` de c462fc55^.

F10 **corrigido**: a retomada que volta a falhar em work.json.tmp imprime
baseline-recorded, unconfirmed.transition e next_action para pessoas, assim
como o recibo JSON. `TestAResumeThatFailsAgainKeepsItsReceiptForPeople` passa
no HEAD e falha restaurando somente `internal/cli/start.go` de c462fc55^.

Reexecutadas também as contraprovas anteriores: parsers de close/start/specs,
facts, continuous status, MCP, limites de since, publicação de resultados,
recibo textual de check filho, retomada após spec movida, construtor,
already-satisfied, waiting e duração desconhecida. As 22 mutações finais
reprovam por asserção funcional, sem erro de compilação. O teste de verdict
não zero concluído é um controle: passa também na produção antiga. O teste de
ordem slug falha contra o adapter que recusava esse valor anunciado. Uma
mutação de recovery que já não encontrava a expressão original foi detectada
e substituída por uma que remove efetivamente a chamada atual a finishStart;
só a mutação efetiva entra na matriz final. Produção foi restaurada byte a byte
em finally; nenhum teste foi editado. A suíte dirigida voltou a passar depois
das restaurações.

### Finding bloqueante consolidado

#### F11 — medium: baseline obsoleta ainda é anunciada como retomável por um comando que não pode concluí-la

Locais: `pose-mcp/internal/pose/start.go:294`;
`pose-mcp/internal/cli/start.go:136`, `:150`, `:229` e `:234`.
Requisitos afetados: R1, R6, R7 e R9; casos V6/V8; D1 e contrato de recovery
do ADR. Este é o único finding medium ou maior aberto nesta rodada.

Cenário independente, sem editar código ou policy do clone:

1. Em fixture Git com atomic start adotado, calcular preview de uma spec draft.
2. Tornar .pose/specs 0555 e aplicar seu digest: baseline-recorded é publicado,
   a transição falha. Restaurar 0755.
3. Criar um commit vazio, representando avanço de HEAD sem alterar a spec.
4. Repetir exatamente o comando anunciado, com o digest original, duas vezes.

Ambas as chamadas retornam exit 1 e um documento JSON válido, porém dizem:

```json
{"outcome":"failed","resumable":true,"error":{"cause":"plan-digest-mismatch","message":"start: plan-digest-mismatch"},"next_action":"fix the cause, then repeat `pose start spec:work --apply --digest <original>`; the recorded baseline is reused and an already moved spec is reconciled"}
```

A spec continua draft. O digest atual incorpora a nova Revision e nunca
coincide com o original nesse HEAD. Pedir um novo preview e aplicar o digest
novo também falha: conflict, spec-already-started-with-another-plan, sem
next_action. A recuperação suportada desta fixture é cancel, novo preview e
novo apply: foi executada e concluiu started. A saída não orienta essa
reconciliação e apresenta como retomável um retry permanentemente incompatível.

Há ainda uma regressão direta do conflito CAS examinado na quarta rodada:
em vez do commit vazio, editar R1 depois da baseline. O gate de ResumeStart
agora colapsa before e um terceiro digest em transition-not-published; o
adapter então perde compare-and-swap-conflict e gera o mesmo recibo
resumable=true/plan-digest-mismatch acima. O mesmo probe, com os dois arquivos
de start de c462fc55^ via overlay de compilação, retorna
compare-and-swap-conflict, resumable=false e orientação de reconciliação.
Portanto não é apenas uma dúvida editorial sobre a mensagem. O novo fluxo
regrediu a classificação e a promessa de retry do conflito já observado.

Esperado: manter F9 fechado, distinguir baseline sem transição de conflito
real e divulgar quando o digest persistido já não pode autorizar continuação.
Nesses casos, orientar status/reconciliação e, quando aplicável, cancel +
preview atual, sem afirmar que repetir o digest obsoleto conclui. Não retomar
uma transição não publicada usando autorização antiga. Acrescentar regressões
para avanço de HEAD sem alteração da spec e para spec num terceiro digest,
verificando outcome/cause/resumable/next_action nos dois renderers.

Evidência: `fifth_round.stale_start`, `cas_conflict` e
`cas_conflict_prior_production` no JSON. As fixtures e drivers completos estão
incluídos para reprodução; não houve correção de produção nesta revisão.

### Varredura completa, validação e decisões

Revisados todos os branches do laço de close e suas chamadas de validate,
index, seal, attestation e transition, helpers de renderização, checkpoint e
replanejamento; start preview/status/apply/retry/cancel; specs/facts;
continuous status; parsing, subprocessos, dispatch/catálogo MCP, journal e
agregação. A busca por stdout direto não encontrou emissão de progresso fora
dos buffers no laço. Os cenários executados de success, waiting e failure nas
rotas JSON suportadas continuam com um documento, e combinações recusadas
saem 2. O resultado de F11 é JSON válido com semântica de recuperação errada.

Passaram go vet ./..., go test ./... -count=1 e go test -race nos quatro
pacotes tocados, em pose-mcp. Passaram quickstart executável e suíte dirigida
após restauração. Os 15 probes F07 foram reexecutados: P03/P06/P07/P08
coincidem em exit/single_json com b12-front07-probes.json. Esse arquivo lista
os outros 11 controles por ID; suas propriedades foram comparadas com a
execução completa preservada da quarta rodada. P12 preserva query válida e
coverage incompleta; só os hashes sintéticos da fixture na prosa de limitation
foram normalizados. Nenhum controle foi promovido a prova de completeness.

Publicação em diretório ou arquivo anterior read-only não confirma
results-written; a falha conserva unconfirmed. Publicação observável e a
retomada de uma transição já publicada passam. Os testes cobrem close com
waiting/pending/next action, efeitos anteriores a falha e no-op distinto de
verificação. Não foi observado outro efeito confirmado inexistente nos
cenários executados. F11 continua uma falha de causa e recovery.

MCP real por stdio recusou zeros explícitos, since vazio/inválido, null,
unknown argument e sort inválido, preservando omitidos e sort slug; o teste de
integração HTTP também passou. O gate sintético interrompido retornou
execution_inconclusive, deadline-exceeded, partial_output_complete=false,
saída parcial e limite de cleanup, sem passed. Seu journal registrou errors=1,
unknown=1, fail=0, completed=0 e duration_samples=1. Cancelamento e prazo curto
passaram no domínio. Não se afirma cleanup de toda a árvore de processos.
Waiting, duration_unknown e duration_samples passaram; o ADR declara a rejeição
de linhas waiting por leitores antigos.

A busca de consumidores em scripts, skills, workflows, docs, testes e CI não
encontrou dependência documentada das combinações agora recusadas. O
quickstart preservou closeout_plan.attest e closeout_plan.result=closed no
fechamento efetivo. CloseoutPlan de --plan --json, os documentos legados de
start e GateResult continuam preservados. Exit codes 0/1/2/3 foram exercitados.

R2/R3/R4/R5/R8 têm evidência funcional nos cenários enumerados; R1/R6/R7/R9
continuam com a lacuna F11. Lint não satisfaz esses requisitos. D1 continua a
separação estrutural apropriada, mas o recibo não deve prometer recovery que o
domínio recusa. D2 evita quebra desnecessária dos documentos existentes; a
classificação minor é deliberada. O ADR declara honestamente a fronteira de
contexto em processo do CLI e a ausência de rotas MCP de close; R5 foi
qualificado para os subprocessos pertinentes e MCP, sem inferir rollback pelo
fim do processo. Não houve mudança de gate de governança: o atalho em forma
que esta revisão falsifica é declarar resumable apenas porque há baseline.
Nenhum julgamento foi automatizado a partir de testes verdes.

### Ferramentas nativas e encerramento

Bundle somente preparado: `rvb-51d0dd40ee76ab34`, digest
`sha256:51d0dd40ee76ab34ffd9ba551663b8e7f6c85a9adc746bddf94a7b678a7dcaf8`; nenhum selo ou atestação.
Review verify: exit 1, state=ready-to-seal, fresh=false,
approved=false. Close não executado. Lint-spec --strict: exit 0, spec ainda
in-progress, trace.entries=0/trace.missing=9; isso não é evidência funcional.
Check --strict: exit 0. As saídas completas estão em fifth_round.native.

Artifact-check --strict passou (31 claims e 31 observed). A preparação inicial
sem projeção atual recusou change set; artifact-check recompôs a projeção
nativa e o bundle acima pôde ser preparado, sem selo ou contorno. Surface-check
--strict terminou 1, com stale-evidence e unconsumed-capability: falta evidência
integration de scoped provenance atual. Isso não é promovido a um finding de
produção nem a passed; a validação estrita de entrega e sua regeneração são
condicionais à aprovação, interrompida por F11. Não foi usado o 131/131 do
implementador como substituto de verificação independente atual de composição.

Assess design, integrate, tech-debt, suggest review e recurrence-check foram
executados. Tech-debt: zero marcadores descobertos. Integrate: 64 contratos,
um ativo e 63 consumer unobserved; isso não prova ausência de consumidor
externo. Recurrence: zero chaves acima do limiar. Assess-discover recomendado
foi dispensado nesta revisão sem implementação/closeout; review-check e
closeout-check permanecem deferred pela precondição review-complete. Validate
global e validate de pose-mcp não receberam passed fictício. As disposições
por ferramenta/componente estão no JSON. Não houve discover --update-state.

Após todas as mutações, as ferramentas nativas foram repetidas com produção
e testes restaurados; só essa repetição limpa fundamenta os resultados acima.
As projeções temporárias de assessments/state/delivery-integrity foram
restauradas aos bytes iniciais. Requirements, Technical Plan, tasks, trace,
Final Report, matriz B12 e roadmaps não foram modificados nem marcados como
entregues. Não houve contribuição upstream ou handoff de risco aceito; F11
não foi aceito como risco residual. Parecer e evidência preservam todas as
rodadas. Não houve commit, push ou merge. O closeout depende de remediar F11
e repetir a revisão independente.

## Sixth round

Data: 2026-10-16. Decisão: **changes-requested**. HEAD independente revisado:
`fc5da36c878abcd9e2910c2b8208ee2fa2996923`; remediação F11 em `103eab34`.
As cinco rodadas anteriores permanecem preservadas acima e no JSON. Esta rodada
não aceita findings como risco residual nem autoriza fechamento.

### Identidade e regras aplicadas

Revisor autorizado: `agent:independent-gpt-6.1-sol-review`, Codex/OpenAI;
implementador: `agent:claude-opus-5-5`, Claude/Anthropic, sessão
`296d1fcb-e989-40ae-9362-cf3514db8b26`. Julgamento de scope/assumption-integrity:
o prefixo `agent:independent-` é a declaração exigida pelo motor até a spec
`pose-delegated-review-run-provenance` substituí-lo. Esta execução é de outro
vendor; identidade declarada, sem atestação assinada no pose-dist. Não se atribui
prova criptográfica ao prefixo.

Aplicadas novamente as regras security, backend-go, documentation-style,
delivery-evidence e knowledge-governance, workflow review e skills pose-review
e pose-spec-closeout. Consultados AGENTS, manual, spec, ADR, relatórios das
frentes 7/8/9, knowledge e diff completo origin/main..HEAD (31 arquivos).
Não há novo dependency manifest nem alteração de autorização/policy/adoção.
A revisão de exposição de segredos no diff não identificou segredo; a matriz
não registra scanner de secrets/vulnerabilities separado para este módulo.
Não se declara scanner inexistente como executado.

Recompilada a CLI com o comando solicitado. Como `go -C pose-mcp ... -o
../pose-review` produz o arquivo na raiz do clone, ele foi movido para
`../pose-review` a partir da raiz; todos os comandos nativos desta rodada
usaram esse binário, sem usar pose do PATH. Nenhuma das três variáveis de
seleção de projeto proibidas foi exportada. Os testes preexistentes e o harness
mantêm suas próprias fixtures e variáveis internas.

### F9, F10 e F11 — fechados

F11: ambos os cenários de `TestAStaleStartBaselineIsNotAdvertisedAsResumable`
passam com a implementação atual e reprovam com o adapter de `103eab34^`.
Colapsar a classificação stale/conflict do domínio em retry também reprova os
dois. Probes independentes de HEAD avançado e terceiro digest da spec retornam
`conflict`, causas `stale-start-baseline`/`compare-and-swap-conflict`,
`resumable=false`, orientação de status/cancel e novo preview; repetir o digest
antigo não publica a transição. Cancel + novo preview volta a permitir o start.

F9 continua fechado: reintroduzir apenas a recuperação de uma transição ainda
não publicada reprova `TestARetriedStartDoesNotOutliveARequestThatNowRestrictsIt`;
a produção atual recusa a autorização antiga. F10 continua fechado: o adapter
anterior à correção reprova `TestAResumeThatFailsAgainKeepsItsReceiptForPeople`;
a produção atual conserva recibo e próximo comando em humano e JSON.
A recuperação de uma transição publicada ainda funciona depois de reparar a
causa. Essas conclusões não dispensam o finding adicional abaixo.

### F12 — medium: start omite a transição publicada do recibo tipado

Local: `pose-mcp/internal/cli/start.go:230`–235, em `startApplyFailure`;
a construção do recibo também deixa `steps_run` vazio em todos os ramos.
Requisitos afetados: R1, R2, R6 e R9; V6, D1 e o contrato de efeitos/execução do ADR.

Cenário reproduzido desde um start novo, sem concorrência: criar uma spec draft,
obter o digest e criar o diretório `.pose/starts/work.json.tmp` antes do apply.
A baseline é publicada pelo writer exclusivo; a spec passa a `in-progress`;
somente a atualização final do registro falha porque o temporário é diretório.
`StartRecovery` lê a postimagem e confirma `StartRecoveryPublished`. O SHA-256
dos bytes da spec corresponde a `spec_digest_after` do registro persistido.
Entretanto, o resultado contém apenas:

```json
{"outcome":"failed","step":"record","steps_run":[],"effects":[{"step":"baseline","effect":"baseline-recorded","ref":".pose/starts/work.json"}],"unconfirmed":[{"step":"record","reason":"the spec moved but the start record does not say started yet"}],"resumable":true}
```

A mudança da spec é um efeito confirmado e ocorrido nesta chamada, mas não
aparece em `effects`; baseline e transição executadas também não aparecem em
`steps_run`. Um consumidor genérico do resultado versionado perde o efeito de
lifecycle e a história da execução. A frase livre em `unconfirmed.reason`
conta que a spec mudou, mas não substitui um efeito confirmado tipado; os hashes
esperados em `data` tampouco são uma observação tipada da publicação. No humano,
a transição igualmente aparece apenas na prosa de um passo não confirmado.
Não há pedido de rollback, publisher geral ou mudança do JSON legacy de sucesso.

Esperado: refletir a transição observada como efeito confirmado, com referência
à spec, manter somente a finalização do registro como não confirmada e registrar
os passos realmente executados pela chamada. Uma retomada deve distinguir seus
passos dos efeitos herdados. Cobrir o primeiro apply que publica a transição e
falha no registro, além do retry, nos dois renderers.

Evidência: `sixth_round.published_transition_receipt` contém before/after,
postimage_matches_record=true, stdout/stderr dos dois renderers, retry e retomada
após reparo. `sixth_round.receipt_contract_probe` contém o teste independente
`TestIndependentPublishedStartHasCompleteTypedReceipt`, executado por overlay
externo sem alterar os testes do clone: exit 1 por falta de efeito de spec e
história vazia. A suíte entregue continua verde; ela não exige esses fatos no
recibo. Não foi aplicada correção de produção.

### Varredura completa e evidência funcional

Reexaminados o laço inteiro de close, evidence/index/seal/mechanical/judgment/
verify/transition, checkpoint, replanejamento e renderers; todos os modos de
start, specs/facts, continuous status, subprocessos, dispatch/catálogo MCP,
journal e seus agregadores. Nenhum outro finding médio ou maior foi identificado
nos caminhos e cenários examinados. Não se afirma prova de ausência universal.

Executadas 24 mutações temporárias de produção, todas com falha de teste real,
sem erro de compilação ou timeout do harness. Incluem reversão por arquivo para
origin/main e revisões anteriores, parsers, JSON, publicação, already-satisfied,
construtor, timeout/cancelamento, MCP, waiting/duração e recuperação F9–F11.
Os controles positivos de verdict concluído preservam pass na produção antiga,
como devem: não se exige que um controle de compatibilidade reprove. A suíte
atual completa e a dirigida depois da restauração passam. Fontes das mutações,
resultados por teste e outputs integrais estão no JSON.

Passaram `go vet ./...`, `go test ./... -count=1` e `go test -race` nos quatro
pacotes tocados, em pose-mcp. Passou o quickstart real com o binário compilado,
incluindo `closeout_plan.attest` e `closeout_plan.result=closed`. Exit codes
0/1/2/3 foram exercitados. Os 15 probes F07 foram repetidos: P03/P06/P07/P08
coincidem com b12-front07-probes.json; os 11 controles coincidem com a execução
completa preservada da quinta rodada, normalizados apenas paths temporários,
digests sintéticos, generated_at e tempos de produtores. P12 continua consulta
válida com coverage incompleta; não se promove transporte a completeness.

As rotas JSON de sucesso/espera/falha exercitadas emitem um documento. A busca
por stdout direto no laço e seus helpers não encontrou progresso fora dos
buffers. Falhas de publicação em diretório/read-only não confirmam results-written;
a publicação observável confirma. No-op não alega nova verificação. O finding
F12 é de conteúdo do recibo, não de sintaxe JSON nem de recuperação indevida.

MCP real via stdio preservou omitidos e sort slug e recusou unknown args, null,
zero explícito e since/sort inválidos; integração HTTP e callToolCtx passaram
nos testes Go. O timeout controlado retornou execution_inconclusive sem passed,
com causa deadline-exceeded, partial_output_complete=false e limite de cleanup.
O journal registrou errors=1, unknown=1, fail=0, completed=0 e duration_samples=1.
Cancelamento passou no domínio. Não se afirma cleanup de toda a árvore de filhos.
Waiting, duration_unknown e duration_samples passaram; a rejeição de waiting por
leitores antigos está declarada no ADR.

A busca de consumidores em scripts, skills, workflows, docs, testes e CI não
identificou dependência documentada das invocações recusadas. Plan JSON, documentos
legacy de start e GateResult mantêm formato; aliases documentados passaram.
D1 continua adequado, mas seu recibo não está completo em F12. D2 é proporcional,
preserva contratos existentes e classifica as recusas como minor. O ADR declara
honestamente contexto em processo do CLI fora do escopo e a ausência de rotas
MCP de close: a paridade foi julgada nas rotas existentes, sem inventar um writer
MCP. R3/R4/R5/R7/R8 têm a evidência funcional descrita; R1/R2/R6/R9 permanecem
incompletos em F12. Lint não satisfaz requisito funcional. Não foi adicionado gate
de governança; o atalho formal aqui falsificado é emitir um envelope válido sem
a lista completa dos efeitos observados.

### Ferramentas nativas e encerramento

Bundle apenas preparado: `rvb-518ba8c9a1c2350d`, digest
`sha256:518ba8c9a1c2350d60a0aafb0553560239901c2fdeb73e9eae097cf15eacf3b6`.
Nenhum selo ou atestação. Review verify: exit 1, ready-to-seal, fresh=false,
approved=false. Close não executado. Lint-spec --strict passou, ainda com spec
in-progress e trace.entries=0/trace.missing=9; não é evidência de entrega.
Check --strict: exit 0. Os outputs e as dispositions reais estão em sixth_round.native.

Artifact-check --strict passou com 31 claims/31 observed. O plano inicial sem
projeção atual recusou change set; artifact-check atualizou a projeção nativa e
a preparação passou. Surface-check --strict foi realmente executado e falhou:
stale-evidence e unconsumed-capability, sem integration de provenance atual.
A validação estrita global/de módulo e a regeneração de delivery-validation
continuam condicionais à aprovação, interrompidas por F12. O 131/131 publicado
pelo implementador não substitui essa etapa independente nem resolve o finding.

Assess design, discover --if-stale --component pose-mcp, integrate, tech-debt,
suggest review e recurrence-check foram executados. Tech-debt encontrou zero
marcadores; integrate lista 64 contratos, um ativo e 63 consumers unobserved;
recurrence não tem chave acima do limiar. Os outputs reais fundamentam as
dispositions no JSON; review-check/closeout-check ficam deferred por
review-complete não satisfeito, e validate não recebe passed fictício.

Após as mutações, produção e testes foram conferidos contra HEAD; as projeções
temporárias de assessment/state/delivery-integrity foram restauradas aos bytes
iniciais. Não houve discover --update-state. Parecer e evidência preservam as
rodadas anteriores; Requirements, Technical Plan, tasks, trace, Final Report,
matriz B12 e roadmaps não foram alterados nem marcados entregues. Não houve
contribuição upstream, aceitação de risco, commit, push, merge ou PR novo.

## Seventh round

Revisão de `5efb9ed5c1115b0d0062bc6c84d9e6c733146cf0`, remediação F12
em `df8072d5`, realizada em 2026-10-16. Decisão: **changes requested**.
F9/F10/F11 continuam fechados; F12 fechado nos cenários reclamados. Resta
um finding bloqueante novo, F13, abaixo. A varredura continuou após reproduzi-lo.

### Identidade, escopo e regras aplicadas

Revisor autorizado: `agent:independent-gpt-6.1-sol-review`, execução
Codex/OpenAI, de outro vendor em relação ao implementador Claude/Anthropic,
`agent:claude-opus-5-5`, sessão `296d1fcb-e989-40ae-9362-cf3514db8b26`.
Julgamento de scope/assumption-integrity: o prefixo `agent:independent-` é
uma declaração exigida pelo motor até a spec
`pose-delegated-review-run-provenance` substituí-lo. Identidade declarada;
no pose-dist as atestações não são assinadas. O prefixo não prova identidade
criptográfica nem independência cognitiva.

Tipo mixed: correção de comportamento com contrato CLI/MCP e ADR.
Consultados AGENTS raiz e dos paths documentais, POSE.md (review, trace e
closeout), `.pose/workflows/review.md`, skills pose-review e
pose-spec-closeout, spec, ADR, relatórios de origem das frentes 7/8/9,
diff integral `origin/main..HEAD` e conhecimento relacionado. Regras:
backend-go (erros/contexto e testes), security (inputs, subprocessos e ausência
de segredos no delta), documentation-style (contratos e quatro manuais),
delivery-evidence (sem promover suíte verde a entrega), knowledge-governance
e _base-recurrence (proveniência e classificação da recorrência). Não há
mudança de frontend, Kubernetes, adoção ou policy. Nenhum gate de governança
novo foi criado: o falsificador relevante é um envelope sintaticamente válido
que perde fatos de publicação parcial.

A CLI foi compilada nesta branch. O comando solicitado com `go -C pose-mcp
build -o ../pose-review ./cmd/pose` gera `pose-review` dentro do clone; esse
build foi copiado para `../pose-review`, caminho dos comandos solicitados.
Os primeiros probes externos que ainda usavam o binário da sexta rodada foram
descartados e repetidos com o build atual, incluindo a reprodução F13.
O binário transitório dentro do clone foi removido. Não foram exportadas as
três variáveis de seleção de projeto proibidas. Os testes Go usaram o source
atual, independentemente dessa correção do caminho do binário.

### F9, F10, F11 e F12 — fechados

F9: a retomada de uma baseline sem transição não ignora restrição governada
nova. A regressão atual passa; mutação que recupera o estado before diretamente
volta a iniciá-la e reprova. F10: falha repetida da retomada preserva o recibo
humano; o adapter anterior à correção reprova. F11: head movido e spec alterada
produzem conflito não retomável com orientação de reconciliação; tanto o adapter
anterior quanto a classificação colapsada no domínio reprovam os dois casos.
Os probes externos de stale baseline e CAS foram repetidos com a CLI atual.

F12: `TestAStartReceiptNamesThePublishedTransitionAndOnlyItsOwnSteps` e
`TestAStartWhoseTransitionFailsClaimsNoSpecEffect` passam; os dois reprovam
com somente cli/start.go anterior a df8072d5, mantendo a assinatura atual de
ResumeStart para permitir compilar a reversão. Com work.json.tmp como diretório,
o primeiro apply confirma baseline-recorded e spec-in-progress com o path da
spec e lista baseline=done,transition=done,record=failed. Retry JSON e humano
listam apenas record=failed, conservando os dois efeitos herdados. Com specs
0555, baseline=done,transition=failed e nenhum efeito da spec. O probe externo
confirma esses fatos e o retry após reparar a causa conclui o registro.
Não reabro F12: o defeito novo acontece antes de uma baseline válida existir.

### F13 — medium: falha durante a primeira escrita da baseline desaparece do recibo parcial

Local principal: `pose-mcp/internal/cli/start.go:242` (erros de leitura da
recuperação ignorados), `:273`–`:281` (fallback sem fase, incerteza ou ação)
e `:293`–`:297` (só reconhece baseline concluída). Origem do efeito:
`pose-mcp/internal/pose/start.go:242`–`:243` e `:527`–`:536`: O_CREATE|O_EXCL
cria o registro final e Write pode gravar parte dos bytes antes de retornar erro;
ApplyStart retorna StartRecord vazio e nenhum hook foi alcançado.

Reprodução independente, sem alterar produção/testes ou policy do clone:
fixture sintética com atomic_start_version:1, uma spec draft com R1 e commit
inicial; obter preview/digest normalmente. Executar somente o subprocesso
`start spec:work --apply --digest <digest> --json` com
`resource.setrlimit(resource.RLIMIT_FSIZE, (512, 512))`. O limite reproduz
uma escrita curta/EFBIG de forma determinística, sem precisar esgotar o disco.
O processo retorna exit 1 e um JSON válido:

```json
{"operation":"start.apply","outcome":"failed","steps_run":[],"effects":[],"resumable":false,"error":{"cause":"apply-failed","message":"write .../.pose/starts/work.json: file too large"}}
```

O arquivo final `.pose/starts/work.json` existe com **512 bytes**, JSON
truncado. A spec permanece draft. Sem o limite, repetir o comando falha com
`invalid-start-record`; `--status`/`--cancel` tampouco devem ser apresentados
como reparação automática de um registro inválido. Não há `step: baseline`,
`baseline=failed`, `unconfirmed` para a escrita parcial nem `next_action` de
inspeção/reconciliação. O renderer humano recebe a mesma ausência de fatos.

Requisitos afetados: **R1, R2, R6, R9 / V6**. A ausência de efeito confirmado
é correta — bytes truncados não são uma baseline válida —, mas não elimina
a obrigação de informar o passo que falhou depois de poder escrever. A mensagem
EFBIG por si só não distingue ausência de escrita de um registro final parcial.
Este finding não exige o publisher CAS geral de B14 nem atomicidade nova: exige
preservar o fato/fase da tentativa de baseline no resultado tipado e classificar
sua escrita não confirmada, com orientação segura para inspeção e reconciliação.
Adicionar regressão de escrita inicial curta e conferir JSON e humano, retry e
preservação dos bytes parciais; não apagar automaticamente o material só para
permitir retry. Driver completo, stdout/stderr, bytes e retry estão em
`seventh_round.partial_baseline` e `partial_baseline_driver` da evidência.

### Varredura completa e validação independente

Reexaminados laço completo de close, evidence/index/seal/mechanical/judgment/
verify/transition, checkpoint, replanejamento e renderers; todos os modos de
start e suas recusas/recuperações; specs/facts e aliases; continuous status;
subprocessos de gates e queries, dispatch/catálogo/transporte MCP; journal e
agregadores; consumidores em scripts, skills, workflows, docs, testes e CI.
Nenhum outro defeito médio ou maior foi identificado nos caminhos examinados.
Isso não é uma prova universal de ausência de defeitos.

Passaram go vet ./..., go test ./... -count=1 e go test -race nos pacotes
internal/cli, internal/pose, internal/mcpserver e internal/usage, em pose-mcp.
A suíte dirigida foi repetida e passou depois da restauração das mutações.
Passou o quickstart executável inteiro com o caminho absoluto da CLI atual,
incluindo closeout_plan.attest e closeout_plan.result=closed.

Executadas **25 mutações temporárias** de produção: todas obtiveram falha
real de teste, sem erro de build nem timeout do harness. Cobrem cada regressão
nova, inclusive timeout/cancelamento, query, parsers, already-satisfied isolado,
construtor, MCP, waiting/duração, publicação e F9–F12. O controle de gate que
termina a tempo continua passando na produção antiga; o controle de sort slug
passa na main e reprova na revisão que passou a recusá-lo. Não se exige que
controles de compatibilidade reprovem no código anterior. Os outputs por teste,
fontes e adaptações mecânicas de assinatura estão no JSON.

Os 15 probes F07 foram repetidos: P03/P06/P07/P08 coincidem com o after de
b12-front07-probes.json; os 11 controles coincidem com a sexta rodada preservada,
normalizados paths temporários, digests/revisões sintéticos, generated_at e
tempos dos produtores. P12 mantém coverage incompleta, sem promover consulta a
complete. As rotas JSON de sucesso/espera/falha exercitadas emitem um documento;
argumentos incompatíveis saem 2 antes da operação. A busca do laço de close não
encontrou escrita de progresso no stdout fora dos buffers. A falha F13 é de
conteúdo do recibo, não de sintaxe JSON. Evidência não publicada não vira
results-written; no-op não vira verificação nova. Exit codes 0/1/2/3 exercitados.

MCP real stdio preserva omitidos e sort slug e recusa unknown args, null, zeros,
since/sort inválidos. Prazo controlado retorna execution_inconclusive sem passed,
com deadline-exceeded, partial_output_complete=false e limite de cleanup.
Journal: errors=1, unknown=1, fail=0, completed=0, duration_samples=1.
HTTP JSON-RPC e cancelamento também passam nos testes Go. Waiting,
duration_unknown e duration_samples passam; a rejeição de waiting por leitores
antigos está declarada no ADR. Não se afirma cleanup de toda a árvore de filhos.

D1 é a decisão adequada, mas F13 deixa uma execução com escrita parcial sem
sua fase e incerteza. D2 preserva CloseoutPlan, os documentos de start e
GateResult; recusas e semântica humana são deliberadamente minor. Não foi
identificado consumidor documentado que dependa das invocações recusadas.
O limite de contexto em processo do CLI está declarado honestamente; R5 foi
avaliado nos subprocessos pertinentes, sem alegar rollback pelo sinal. A ausência
de rotas MCP de close está declarada no plano; paridade julgada nas rotas que
existem. Esses limites não excluem a obrigação de receipt parcial de start.
R3/R4/R5/R7/R8 têm as evidências descritas. R1/R2/R6/R9 permanecem incompletos
em F13; lint não é seu produtor funcional.

### Ferramentas nativas e encerramento

Bundle somente preparado após restauração: `rvb-d5ee3bfe9cba2c19`, digest
`sha256:d5ee3bfe9cba2c192c32d1240cfaed812385802c7fefb389f37c084d971b60fa`; nenhum selo ou atestação.
Review verify: exit 1, state=ready-to-seal, fresh=false,
approved=false. Close não executado. Lint-spec --strict:
exit 0, spec in-progress, trace.entries=0/trace.missing=9. Check --strict: exit 0,
29 avisos. Artifact-check --strict: exit 0, 31 claims/31 observed. Surface-check
--strict foi executado e falhou com stale-evidence/unconsumed-capability;
a validação estrita global/de módulo e a regeneração de delivery-validation
ficam no caminho condicional de aprovação, interrompido por F13. O 131/131 do
implementador não substitui aprovação funcional independente.

Assess design, discover --if-stale --component pose-mcp, integrate, tech-debt,
suggest review e recurrence-check foram executados; bundle, discover, design,
integrate, tech-debt e review verify foram obtidos novamente após restaurar
produção. Tech-debt: zero marcadores; integrate: 64 contratos, um ativo e
63 consumers unobserved (diagnóstico preexistente, não 63 defeitos novos).
Recurrence: zero chaves acima do limiar. Review-check/closeout-check ficam
deferred por review-complete não satisfeito; validate não recebe passed fictício.
As dispositions reais e os outputs estão em seventh_round.native. Os arquivos
operacionais de assessment/state/delivery-integrity foram restaurados aos bytes
iniciais; discover --update-state não foi executado, pois não houve fechamento.

Classificação da recorrência: backend/contrato/implementação e lacuna de
validação em erros por fase de publicação. A prevenção proposta é estender
V6 à escrita inicial curta, preservando os falsificadores anteriores. Não foi
criado/staged feedback ou contribuição upstream; nenhuma adjudicação foi
inferida. Não houve aceitação de risco, alteração de Requirements/Technical
Plan, tasks, trace, Final Report, matriz B12 ou roadmaps. Não houve selo,
atestação, close, commit, push, merge ou PR novo. Parecer e evidência preservam
integralmente as seis rodadas anteriores. F13 precisa de remediação e nova
revisão independente antes do closeout.

## Eighth round

Data: 2026-10-16. Head revisado: `1615155905283a5ae7ff41b0a7e32ae5198ac6c7`;
remediação F13: `f59019483e658208aeaa87871cf4843cd4912896`. PR #186,
branch `impl/pose-b12-operation-results`.

**Decisão: aprovada.** F13 fechado; F1–F12 continuam fechados nos cenários
reexaminados. A varredura completa não identificou outro defeito médio ou maior.
Essa conclusão é limitada às rotas e cenários examinados, não uma prova universal.

Revisor: `agent:independent-gpt-6.1-sol-review`, execução Codex/OpenAI diferente
do implementador Claude/Anthropic `agent:claude-opus-5-5`, sessão
`296d1fcb-e989-40ae-9362-cf3514db8b26`. O prefixo `agent:independent-` é a
declaração exigida pelo motor até `pose-delegated-review-run-provenance`
substituí-lo. Identidade e separação declaradas; atestações não assinadas no
pose-dist. A autorização do maintainer permite a operação e não é apresentada
como adoção humana destas conclusões ou prova criptográfica de independência.

### Rules applied during review

Tipo: mixed, correção de bugs com contrato estrutural versionado. Consultados
AGENTS.md, POSE.md (review, closeout e assurance), workflow review, skills
pose-review e pose-spec-closeout, spec B12, ADR de resultados e relatórios de
origem das frentes 7, 8 e 9. Regras aplicadas: backend-go para contexto,
concorrência e persistência; security para limites de paths, argumentos e
observabilidade; documentation-style para CLI/locale/scaffold e consumidores;
delivery-evidence para proveniência e composição; knowledge-governance na
consulta de decisões e riscos anteriores. Frontend, infraestrutura e regras de
outras linguagens não se aplicam ao delta. Nenhuma policy, capability ou adoção
foi alterada. Requirements e Technical Plan foram preservados byte a byte.

### F13 e contraprovas

O primeiro apply sob RLIMIT_FSIZE=512 deixa exatamente os bytes parciais do
registro e a spec draft. Retorna um documento JSON com `step=baseline`,
`steps_run=[baseline=failed]`, nenhuma baseline confirmada, `unconfirmed` do
passo e `resumable=false`. O retry JSON não atribui passos a esta nova chamada;
o retry humano dá a mesma orientação de inspecionar `.pose/starts/work.json`
e removê-lo somente quando nenhum start estiver em curso, antes de novo preview.
A regressão `TestAStartBaselineWriteCutShortIsReportedForInspection` passa no
head e reprova por assertion com somente start.go de produção anterior a
f5901948, restaurado em seguida. Nenhum teste foi editado.

Executadas 26 mutações temporárias de produção. Todas geraram falhas de teste,
sem build failure ou timeout do harness. Abrangem os testes novos do construtor,
timeout/cancelamento e query, parsers e aliases, already-satisfied isolado,
publicação de evidência, MCP, journal e remediações anteriores. Controles de
compatibilidade não precisam reprovar na main; sort slug reprova no adapter
que passou a recusá-lo. As adaptações de assinatura nas versões antigas foram
somente mecânicas para compilar o hook atual. Fontes, comandos e outputs por
assertion estão em `eighth_round.mutations` e `mutation_driver` da evidência.
Produção restaurada e diff de pose-mcp vazio ao concluir a revisão.

F9: uma baseline não autoriza iniciar sob restrição governada posterior.
F10: a retomada que falha conserva o recibo humano. F11: HEAD avançado ou spec
alterada retorna conflito, sem prometer retry do digest que já não aplica.
F12: baseline e transição publicadas são efeitos confirmados; os passos de
uma retomada não se confundem com passos da chamada anterior. As contraprovas
respectivas reprovaram; os testes atuais e os probes de falha/retry passaram.

### Varredura e contratos

Reexaminados todos os ramos de close apply/resume/plan: evidence/index,
seal/mechanical/judgment/verify/transition, checkpoint e replanejamento;
start preview/apply/status/cancel, publicação inicial, transição, registro e
recuperação; specs/facts; continuous status; subprocessos, dispatch, catálogo,
transporte e telemetria MCP; journal e agregadores; consumidores em scripts,
skills, workflows, docs, testes e CI. Não há escrita de progresso no stdout
no laço de close: chamadas subordinadas usam buffers. As rotas JSON examinadas
emitem um documento em sucesso, espera e erro operacional. Argumentos recusados
saem 2 antes da operação; JSON não suportado é recusado.

Os 15 probes F07 foram repetidos. P03/P06/P07/P08 coincidem com o after de
b12-front07-probes.json; os 11 controles coincidem com a rodada 7 preservada,
normalizados paths, digests, timestamps e tempos dos produtores sintéticos.
P12 conserva coverage incompleta. O quickstart executável completo passou,
incluindo os consumidores de closeout_plan.attest e closeout_plan.result.
Exit codes 0/1/2/3, CloseoutPlan JSON, documentos de start e aliases foram
exercitados; nenhum consumidor documentado exige a invocação agora recusada.

Resultados não publicados não viram results-written. Falhas depois de escrita
potencial ficam unconfirmed; efeitos são observados nos artefatos, nunca inferidos
do texto de um check. No-op não é verificação nova. O MCP stdio real recusa
unknown/null/zeros/since/sort inválidos e preserva omitidos/sort slug. Timeout
controlado retorna execution_inconclusive sem passed, partial_output_complete=false
e o limite de cleanup. Telemetria: errors=1, unknown=1, fail=0, completed=0,
duration_samples=1. HTTP JSON-RPC e cancelamento passam nos testes Go. Waiting,
duration_unknown e duration_samples passaram; o ADR declara que leitores antigos
recusam waiting. Não há duração humana ou rollback inferidos.

D1 trata a causa material: execução e efeitos não são verdict nem coverage.
D2 é proporcional: preserva documentos já contratados e introduz o resultado
onde faltava, com recusas mais estritas e semântica pública classificadas como
minor. Uma troca geral de envelopes quebraria consumidores sem acrescentar fato.
O ADR declara honestamente o limite de contexto dentro do processo CLI e o
possível filho sobrevivente; R5 foi qualificado nos subprocessos pertinentes.
As rotas MCP de close não existem e sua ausência está explicitada, sem alegar
paridade de writes. B13/B14/B35 continuam fora deste delta. Esses limites não
retiram a obrigação de informar os efeitos de start/close aqui examinados.

Não foi acrescentado um novo gate de aprovação: um JSON válido ou um check
executado não aprova um domínio. Os testes de waiting, no-op, inconclusive e
publicação falsificam esses atalhos de satisfação formal. Os gates governados
continuam recusando ausência de julgamento ou restrição de start.

### Gates nativos e closeout

Bundle `rvb-6bd4b19097039e81`, digest `sha256:6bd4b19097039e8131e7bcdb77b4c554cda17eae85692b9be9340f7379cbe7b0`;
plan digest `sha256:c8fca18cf7dd91f54d4c145496719a178daa35e527ed62124308aed81b78a230`, different-actor, critical.
Atestação `rva-87212ea94b920fde`, aprovada por preview e apply nativos,
15 critérios respondidos, 11 dispositions e o delta `SD-cb1a381d` mapeado a D1.
Global strict: **134/134**; módulo explicitamente executado: **131/131**,
ambos no head 16151559, sem falhas/erros/skips. O GH_TOKEN foi vinculado somente
aos comandos; nenhum valor de credencial foi registrado. Surface-check strict
passou para capability:cli-mcp-operation-results; artifact-check strict passou,
31 claims/31 observed. Os warnings globais de orphans não são findings novos
neste escopo. Todos os tools ativos do plano foram executados; assess design,
discover, integrate, tech-debt e suggest tiveram execução real, não disposição
copiada do template. Review-check passou e closeout-check pós-close é terminal.

Review verify final: exit 0,
fresh=true, approved=true, state=closed.
O resultado pós-edições usa a retenção nativa de review concluído: o bundle
atesta a implementação imutável no head original, não uma reexecução dos checks
sobre cada texto operacional posterior. As atualizações de trace/tasks/report e
roadmaps foram revistas por este ator e preservam R1–R9/D1/D2. Não houve bypass,
reescrita de bundle ou mudança de policy. Close: exit 0,
status done e completed_at=2026-10-16. Lint-spec strict: exit 0,
9/9 trace, sem falhas. Check strict inicial: exit 0;
check strict final: exit 0. O roadmap do programa conserva status active e o roadmap de recovery conserva
status draft; suas consultas tolerantes não são conclusão de programa. As evidências por ferramenta estão no JSON, com resumos delimitados e
hashes para as saídas que continham o grafo inteiro do repositório.

Go vet, suíte completa sem cache de resultados e race nos quatro pacotes tocados
passaram. Tech-debt não encontrou marcadores e recurrence-check não encontrou
chaves acima do limiar. Integrate mantém o diagnóstico de consumers não observados,
sem convertê-lo em novos defeitos nem prova automática de compatibilidade.
A primeira preparação usou índice antigo e não encontrou atribuição; artifact-check
e index recompuseram a proveniência antes da selagem. Assess design foi repetido
sobre o assunto efetivo. Nenhum bloqueio do domínio foi contornado.

Parecer e evidência preservam integralmente as sete rodadas anteriores. O trace
aponta produtores reais de R1–R9, e não lint. Tasks, Final Report, matriz B12 e
os dois roadmaps foram reconciliados; o programa permanece ativo e depende do
merge do maintainer, sem escolha automática da próxima implementação.
A descoberta final foi executada e seus arquivos gerados não entraram no commit.
Não houve correção de produção/testes, feedback staged, contribuição upstream,
aceitação nova de risco, merge ou novo PR. Os riscos residuais são os limites
explícitos do ADR e as unidades já existentes do programa; nenhum finding aberto
foi dispensado por conveniência.
