# POSE — frente 8: performance, processo residente e arquitetura compartilhada

Data: 2026-10-12. Revisão fonte:
`4872275cab34b9037cf06e05f27fb5ca93fa67b2`, após o merge da frente 7.
Motor executado: binário oficial 7.4.1; fontes do motor sem alterações em relação
à tag. Escopo: explicar o custo atual, confrontar a proposta de manter o POSE
residente e propor uma arquitetura comum para CLI/MCP. Esta análise não altera
motor, políticas, disposições, specs ou decisões arquiteturais aceitas.

## 1. Conclusão e decisões que a evidência permite

A direção de um motor residente é pertinente. O benefício esperado vem de
reutilizar catálogos, parsing, objetos Git imutáveis e resultados equivalentes
entre operações, com invalidação correta. Não basta manter o executável aberto:
o MCP já é um processo duradouro, mas parte das ferramentas executa outra CLI,
e as consultas diretas a specs ainda percorrem o catálogo completo por chamada.

O primeiro ganho deve vir da eliminação de trabalho repetido dentro da operação
e da extração de uma API de domínio comum. Um runtime opcional por worktree pode
então reutilizar esse domínio entre chamadas. Essa sequência entrega melhorias
à CLI independente e evita construir um daemon em torno dos mesmos algoritmos
custosos. A residência é uma etapa arquitetural própria, não apenas mais um
comando que junta operações.

O check completo na baseline levou 258 s na configuração padrão e 253 s com
`POSE_CHECK_WORKERS=8`, com saídas idênticas e 30 avisos existentes. Uma amostra
por configuração não demonstra efeito do pool. O custo agregado de CPU dos
filhos, aproximadamente 830 s e 803 s, mostra trabalho computacional e de
processos significativo; não identifica sozinho qual produtor o domina.
O limite de 30 s dos adaptadores MCP é incompatível com essas execuções completas.

Separar três problemas: custo para calcular, custo para transportar o resultado
e frescor da resposta. `recent` e paginação diminuem a resposta, mas atualmente
não evitam a leitura prévia de todas as specs. Nenhuma otimização de performance
justifica cortar o histórico global aos últimos 100 commits, omitir obrigações
ou transformar timeout em veredito negativo de domínio.

## 2. Método, corpus e limites

Usar o [snapshot de evidências](../results/2026-10-12-pose-front-08-performance.json)
para consultar amostras, comandos, saídas integrais dos checks, hashes do binário
e das fontes e URLs fixadas na revisão. Os scripts
[de consultas](../results/2026-10-12-pose-front-08-benchmarks.py) e
[do check](../results/2026-10-12-pose-front-08-check-benchmark.py) tornam o protocolo
repetível; são instrumentos de observação, não testes do motor.

O checkout destacado permaneceu na revisão fonte durante as medições. Continha
4.604 arquivos em `.pose`, 81.921.455 bytes, 380 specs Markdown planas (layouts em
diretórios são adicionais), 3.123.632 bytes nessas specs e índice de entrega
schema 2 de 6.633.259 bytes. A história alcançável por HEAD tinha 1.926 commits.
Não confundir esse número com o alcance `--all` usado pela atribuição.

O ambiente Linux expôs nove CPUs/afinidade nove, com quota cgroup de oito CPUs.
Executar as consultas depois dos checks, sem concorrência com o benchmark pesado.
Não remover caches do sistema operacional; as amostras incluem cache de páginas,
startup da CLI ou ida e volta stdio do MCP, parsing e serialização. A primeira
consulta de cada grupo usa o mesmo servidor inicializado; não representa disco
frio. Comparar medianas e intervalos de três amostras, sem alegação de p95.

As fixtures sintéticas usam Git local, identidade genérica e revisão desativada.
A escala de specs varia entre 10, 100 e 500; a de follow-ups semelhantes entre
10, 50 e 100. Elas isolam os caminhos estudados, sem reproduzir a complexidade
de 4.604 artefatos. A fixture de timeout usa um executável controlado que dorme
40 s, em vez de esperar uma falha real de gate. O protocolo distingue essa
observação de uma execução completa de `pose_check` sobre a baseline.

Não há Go disponível neste ambiente. Os testes existentes foram inspecionados,
mas a suíte Go e o race detector não foram executados. Não há perfil CPU/heap
por função; atribuições de dominância que exigem perfil permanecem hipóteses.
Uma tentativa separada de `strace` registra sua disponibilidade real no snapshot;
falha de permissão não vira contagem zero de leituras ou processos.

## 3. Resultados medidos

### 3.1 Check completo na mesma revisão

| Configuração | Wall (s) | CPU user+system dos filhos (s) | Máximo RSS de filho (KiB) | Resultado |
|---|---:|---:|---:|---|
| Default (pool pelo NumCPU) | 257,963 | 830,255 | 727.716 | Exit 0, 30 avisos |
| Workers 8 | 252,807 | 802,867 | 680.852 | Exit 0, mesmos 30 avisos |

O SHA-256 de stdout+stderr é igual nas duas execuções:
`efab64071c471dea1e0bccfee69e2c3ee1158e5c850c890623f243945fe68750`.
O máximo RSS é aproximadamente 711/665 MiB respectivamente. Não é a soma dos
filhos simultâneos, nem medida do heap do servidor residente. O check final do
PR tem corpus de documentação diferente e serve para validação, não para essa
comparação de performance.

### 3.2 Consultas da baseline, três amostras por caminho

Tempos em milissegundos, mediana e intervalo mínimo–máximo. Bytes são stdout
CLI ou resposta JSON-RPC completa, com diferenças de envelope/representação.

| Operação | CLI ms (intervalo) | MCP residente ms (intervalo) | Bytes CLI / MCP |
|---|---:|---:|---:|
| Listar specs | 21,30 (20,62–24,62) | 17,93 (14,40–21,83) | 204.926 / 407.276 |
| Recentes 5 | 16,63 (16,60–17,64) | 9,25 (7,57–10,20) | 2.576 / 5.359 |
| Attention/obligations | 697,99 (669,06–725,78) | 697,21 (675,23–738,28) | ~196.769 / ~364.930 |
| Follow-ups abertos | 9.024,00 (8.989,46–9.168,76) | 9.054,92 (8.965,96–9.261,89) | 45.845 / 106.430 |

O processo MCP duradouro não reduziu materialmente Attention ou follow-ups
nestas amostras. No segundo caso, o código confirma uma nova CLI por chamada.
Listar poucas specs já é rápido nessa baseline; a redução mais clara é de
resposta. Não supor que a mediana menor de MCP prova cache de catálogo: o caminho
continua listando disco por chamada.

Na última amostra CLI de Attention, o próprio relatório declarou closeout
0,680 s, start 0,362 s e followups 0,104 s. Esses produtores se sobrepõem em
concorrência; somar durações não estima wall. O produtor de follow-ups de
obligations e o comando de reconciliação com clustering não fazem exatamente o
mesmo trabalho: 0,104 s não contradiz os aproximadamente nove segundos do comando.

### 3.3 Escala sintética e distinção entre cálculo e resposta

| Specs | CLI lista ms | CLI recentes 5 ms | MCP lista ms | MCP página 5 ms | MCP GetSpec ms |
|---:|---:|---:|---:|---:|---:|
| 10 | 8,95 | 10,12 | 3,35 | 1,71 | 4,95 |
| 100 | 10,10 | 10,18 | 4,05 | 3,22 | 6,28 |
| 500 | 13,60 | 12,51 | 9,62 | 5,53 | 12,70 |

A página MCP de cinco com 100/500 specs usa 2.781 bytes, enquanto a lista passa
41.404 → 204.204 bytes. Mesmo a consulta de uma única spec cresce nessa escala.
O caminho no código explica a varredura; sem ptrace/perfil, não quantificar
percentual de latência devido à leitura contra serialização. As fixtures têm
corpos mínimos, portanto não extrapolar seus milissegundos para milhares de
bundles/grafos reais.

Os follow-ups semelhantes sintéticos produziram um cluster em cada escala;
medianas CLI foram 8,50 ms (10), 21,53 ms (50) e 50,70 ms (100). Isso confirma
crescimento de custo nessa fixture; a estrutura de pares no código fundamenta
O(F²) em quantidade de comparações. Não ajustar um expoente empírico ou atribuir
os nove segundos da baseline só ao clustering sem perfil.

### 3.4 Frescor, cursor e timeout controlados

Após ler uma spec no mesmo servidor, trocar `Retain behavior` por
`Modify behavior` preservando tamanho e mtime retornou o corpo atualizado.
A releitura atual tem benefício de frescor que o runtime novo precisa preservar;
um cache por stat não passaria esse caso.

Uma página inicial retornou `item-0000`, `item-0001`. Inserir `aaa` antes da
próxima página fez o cursor retornar `item-0001`, `item-0002`: houve duplicação
observada. O token em `pagination.go` guarda offset e versão, sem geração.
Isso não invalida a ordenação de uma chamada, mas revela o contrato ausente
entre chamadas quando o catálogo muda.

A fixture de processo lento recebeu resposta em 30,034 s: JSON-RPC sem erro,
`exit_code=-1`, `passed=false`, output vazio. O sucesso de transporte foi
registrado pelo harness como zero; o veredito de gate dentro do payload é -1.
Em `runCLIExit`, `*exec.ExitError` é convertido em verdict antes de verificar
o contexto expirado. Assim, um timeout controlado aparece como gate negativo,
não erro/inconclusivo de execução. Não foi necessário executar o check real de
253 s via MCP para demonstrar essa classificação. Remediar antes de promover
esse resultado a indicador confiável de entrega/release.


## 4. O que já foi otimizado e o que ainda se repete

### 4.1 Catálogo de specs e bundles

[`spec.go`](../../pose-mcp/internal/pose/spec.go) mostra `GetSpec` chamando
`ListSpecs`, procurando o slug linearmente e lendo novamente o corpo escolhido.
Sem `specSnapshot`, a listagem lê os arquivos completos para obter frontmatter.
Uma chamada dirigida a uma spec, portanto, não tem custo independente de N.

O snapshot usado por `ProjectObligations` elimina releituras do catálogo na mesma
operação e memoiza o scope dos bundles. É melhoria real, coberta por
`TestAttentionSnapshotListsWhatTheDiskLists` e
`TestAttentionSnapshotListsTheSameBundles`. Porém, cada `ListSpecs` ainda percorre
o snapshot e copia campos; `GetSpec` ainda faz busca linear e lê o corpo.
`checkDeliveryContracts` e `checkReviewCloseout`, em
[`check.go`](../../pose-mcp/internal/cli/check.go), constroem Store comum.
Repetir a busca por muitas specs nesses caminhos pode multiplicar a leitura do
catálogo. Isso é verificável no encadeamento do código, sem atribuir todo o tempo
de check a essa causa não perfilada.

Propor um catálogo por operação com mapa slug → entradas, índices de status e
componente, memo de corpo sob demanda e índice scope → bundles. Preservar
conflitos de identidade, layouts legados/split, limites de caminhos e recusas de
symlink. O mapa deve reter duplicidades para recusá-las; escolher a primeira
entrada não é uma otimização equivalente. Ampliar o reuso a produtores que
compartilham a mesma visão de entrada, sem memoizar falhas como ausência.

### 4.2 Grafo de entrega, bytes, parsing e cópias

[`delivery_integrity_cache.go`](../../pose-mcp/internal/pose/delivery_integrity_cache.go)
já memoiza o parsing por conteúdo exato. O arquivo continua sendo lido; comparar
bytes evita servir conteúdo antigo após alterações com mesmo mtime. Cada hit
retorna uma cópia defensiva. Essa cópia é necessária no contrato mutável atual,
mas pode continuar custando CPU/alocação e segurar o mutex durante a cópia.
Duas misses concorrentes podem decodificar o mesmo conteúdo; não há coordenação
por chave para produzir apenas uma vez. O cache guarda somente o último conteúdo,
então alternar projetos diferentes pode expulsar continuamente a entrada anterior.

O schema compacto diminui disco e JSON armazenado. A expansão em
[`delivery_integrity_compact.go`](../../pose-mcp/internal/pose/delivery_integrity_compact.go)
reconstitui o grafo em memória; não garante redução equivalente de heap nem de
pares entrega/resultado. Os 78 MiB de arquivos não são orçamento de RAM de um
motor que manteria bytes, objetos parseados, índices, cópias e respostas ao mesmo
tempo. O RSS medido é máximo de um filho, não soma de todos os workers.

Propor vistas imutáveis compartilhadas e projeções pequenas, com API que proíba
mutação através das coleções devolvidas. Manter cópias nos limites públicos até
provar a migração. Avaliar representação compacta também para consulta, evitando
expandir todo o grafo para responder um filtro pequeno. Revisar aliasing de
campos aninhados na frente 9; a inspeção desta frente não reproduziu corrupção
por um consumidor atual. Os testes de cópia existentes não substituem essa prova.

### 4.3 Git, atribuição e trabalho imutável

[`git_spawn_cache.go`](../../pose-mcp/internal/pose/git_spawn_cache.go) já guarda
paths alterados por root+commit imutável. O limite é 20.000 entradas e limpa o
mapa inteiro; limita quantidade, não bytes ou latência de expulsão. O snapshot
de working tree é por operação, corretamente distinto de commit imutável.
Não promovê-lo a um cache residente de `git status` sem uma geração validada.

O batch reader em [`git_batch.go`](../../pose-mcp/internal/pose/git_batch.go) usa
um `git cat-file --batch` por coletor, com framing, limites e vida limitada.
O [ADR aceito](../adr/2026-10-01-scoped-git-batch-reader-for-structural-assessment.md)
proíbe pool global/cache persistente nesse desenho. Reutilizar o padrão scoped
onde houver equivalência; qualquer mudança de propriedade/lifetime exige novo
ADR, não reinterpretar residência como autorização para compartilhar o stream.

A atribuição em [`artifact_integrity.go`](../../pose-mcp/internal/cli/artifact_integrity.go)
lê commits com `POSE-Spec:` via `git log --all --reverse`, e coleta name-status e
patch binário por commit. Há espaço para lote e reuso por objeto imutável ainda
fora do coletor estrutural. Um índice incremental precisa reconciliar mudança
de refs, commits antes invisíveis e alteração de alcance. Cortar história faria
uma entrega antiga perder a atribuição: performance precisa preservar o conjunto
completo ou demonstrar uma closure de dependências equivalente.

### 4.4 Algoritmo de follow-ups

[`followups.go`](../../pose-mcp/internal/cli/followups.go) compara pares de entradas
de specs distintas. Cada par normaliza texto, monta tokens e calcula similaridade
sequencial recursiva. A quantidade de pares cresce quadraticamente com o número
de entradas; o custo do texto é adicional. Manter arquivos em RAM não remove
essas comparações nem o trabalho de normalização por par.

Pré-calcular formas normalizadas/tokens; reutilizar a comparação de textos
imutáveis sob orçamento. Investigar um gerador de candidatos que preserve a
semântica atual antes de excluir pares. Se a estratégia aproximar resultados,
declarar cobertura e separar observação aproximada de lista de obrigações.
Semelhança continua sendo candidato para julgamento, não equivalência de dívida
nem disposição automática, conforme a frente 5.

### 4.5 Paralelismo e caches que mudam de risco com residência

[`check_parallel.go`](../../pose-mcp/internal/cli/check_parallel.go) usa
`runtime.NumCPU()` e override positivo por `POSE_CHECK_WORKERS`, com saída ordenada.
O default nove diante de quota oito merece observabilidade e teste em várias
máquinas, mas a diferença desta amostra não sustenta trocar o default para oito.
Mais workers podem aumentar CPU, alocações e processos sem reduzir tempo total.
Medir trabalho por produtor antes de aumentar concorrência.

`designDeltaCache` guarda relatórios por digest e não tem expulsão;
`testCatalogCache` guarda por root uma vez por processo, sem invalidação.
Os consumidores atuais de `LoadTestCatalog` são caminhos CLI de lint/close; não
foi demonstrado um bug de teste stale na consulta MCP direta. Tornar esses
avaliadores duradouros mudaria sua premissa de vida útil: alterações em testes,
matriz e módulos precisam invalidar o catálogo. Não transportar caches de CLI
curta para um daemon sem revisar chave, limites, erros e concorrência.

## 5. Processo residente: arquitetura proposta e alternativas

### 5.1 Comparar alternativas pelo trabalho que reutilizam

| Alternativa | Reuso possível | Custo/limite | Papel recomendado |
|---|---|---|---|
| CLI standalone melhorada | Dentro de uma operação | Novo processo por comando | Base obrigatória e CI |
| MCP atual | Caches de caminhos diretos | Adaptadores CLI abrem filhos; specs releem disco | Host existente a aproveitar |
| CLI composta com várias ações | Alguns inputs dentro de uma execução | Termina ao final; não ajuda o próximo comando | UX da frente 7, complementar |
| Motor comum com runtime opcional | Entre CLI, MCP e operações | Invalidação, IPC, memória e lifecycle | Direção arquitetural desta frente |
| Espelhar toda `.pose` em RAM | Bytes e parsing se retidos | Carrega história pouco usada e mantém repetição algorítmica | Evitar como desenho inicial |

Extrair uma interface de execução de domínio tipada: query/evaluation recebe
root autorizado, contexto, modo e inputs; devolve resultado, cobertura, geração
e estado de execução. CLI e MCP adaptam flags/JSON/renderização e compartilham
regras. Evitar duplicar implementação para escapar do subprocesso: a intenção
atual de adaptador sobre fonte única deve permanecer, com a fonte no domínio.

A ordem de dependências é: catálogo/visão por operação → API comum → runtime
opcional → cache entre gerações com invalidação comprovada. É possível reduzir
custo da CLI antes de entregar a última etapa. Não afirmar que o MCP atual já
implementa essa arquitetura só por permanecer aberto.

### 5.2 Identidade, transporte e compatibilidade

Associar o runtime a usuário + worktree canônica + identidade Git + versão do
motor/schema. `project_id` é estável entre checkouts; não identifica a árvore
mutável atendida. Dois worktrees do mesmo projeto devem ter gerações/catálogos
diferentes. Federação exige resolver a raiz autorizada de cada artefato, sem
cachear conteúdo sob um ID genérico compartilhado.

Aproveitar o host MCP quando adequado; sua conexão stdio pertence ao cliente
que o iniciou. Uma CLI avulsa não pode tomar esse stream. Avaliar endpoint IPC
local próprio, por exemplo socket Unix ou named pipe conforme plataforma,
com handshake de root, engine e capacidades. Recusar endpoint incompatível,
proteger permissões locais e preservar a seleção de projeto. Essas são opções
para um ADR futuro, não infraestrutura instalada por esta análise.

Manter fallback standalone para CI, export de fontes, uma única operação e
falhas do serviço. Não iniciar daemon obrigatoriamente a cada comando. Expor
attach/start/stop/status claros, shutdown drenado, timeout de inatividade e
restart após atualização do motor. O workaround atual para self-update resolve
o caminho executável dos filhos; não prova compatibilidade entre um motor antigo
residente e CLI nova. O handshake deve detectar essa combinação.

### 5.3 Catálogo em memória e geração

Carregar metadados e rotas de artefatos, com corpos sob demanda. Orçar caches em
bytes por worktree e globalmente, manter métricas de entradas/hits/misses/evicções
e coordenar misses da mesma chave. Priorizar dados ativos; relatórios históricos
não precisam ser parseados no startup para uma consulta sobre uma spec.

Usar eventos de filesystem como indícios, não prova única de frescor. Reconciliar
após perda de eventos, rename, replace atômico, checkout/reset e mudanças em
índices Git. Mesmo tamanho/mtime não garante conteúdo igual. Writes efetuados
pelo motor devem invalidar ou publicar a próxima geração antes da confirmação.
Chaves devem incluir versão de parser, conteúdo e dependências relevantes, como
políticas, matriz, roots federadas, refs e entradas Git.

Uma consulta pode usar snapshot coerente; um gate e uma escrita precisam declarar
qual geração avaliaram e validar as condições antes de concluir. Cache de verdict
exige provar equivalência dos inputs, não confiar em `dirty=false`. A frente 9
é responsável pelo protocolo de concorrência, fencing, crash/recovery e interleavings.
Esta frente exige essa dependência e não antecipa que um watcher a resolveu.

### 5.4 Contrato de execução e tamanho de resposta

Distinguir avaliação concluída com falha de domínio, erro de execução, cancelamento
e deadline. Propagar contexto também pelo domínio e pelos filhos. Dar budgets
por operação; aumentar arbitrariamente os 30 s não corrige a classificação.
Para gates longos, avaliar jobs consultáveis com ID/generation/result persistível
se a experiência exigir; isso é desenho futuro, não promessa de scheduler ou
sessão retomável já existente. Curto caminho de query não deve disputar todo o
pool com um check pesado.

Retornar resumos e detalhe sob demanda, preservando compatibilidade MCP, inclusive
`content` e `structuredContent`. Medir bytes da resposta completa: o transporte
atual inclui texto e estrutura, podendo duplicar dados. Paginar sobre um catálogo
indexado evita reconstituir tudo. Vincular cursor à geração/filtro/ordem ou recusar
explicitamente cursor obsoleto; um offset estável dentro de uma chamada não é
snapshot estável entre chamadas. O cursor de eventos HTTP/Redis é outro contrato,
sem representar catálogo, plano de execução ou sessão de trabalho.

## 6. Reconciliar os ganhos históricos

As specs de performance registram melhorias implementadas, mas suas medições
pertencem a outros corpus/ambientes. Não extrapolar o tempo atual por subtração
nem declarar regressão do motor apenas comparando esses números.

| Registro fonte | Ganho observado naquela baseline | Leitura para esta frente |
|---|---|---|
| [Construir grafo uma vez](../specs/2026-09-20-check-builds-the-delivery-graph-once.md) | Aproximadamente 635 → 39 s, 117 reconstruções removidas | Confirmar construção compartilhada atual; não refazer trabalho por consumidor |
| [Parse por conteúdo](../specs/2026-09-20-parse-the-delivery-index-once-per-content.md) | 39,3 → 16,6 s; 528 parses de JSON evitados | Parsing pode dominar leitura; reavaliar custo das cópias |
| [Índice compacto](../specs/2026-10-02-pose-delivery-integrity-index-compaction.md) | Disco 10,94 → 4,63 MB; tempo 46,85 → 48,01 s | Compactação foi ganho de tamanho, não aceleração demonstrada |
| [Menos Git spawns](../specs/2026-10-09-pose-check-spawns-fewer-git-processes.md) | 29.500 → 8.911 processos; 125 → 101 s | Há caminhos Git residuais; batch scoped não os cobre todos |
| [Attention por operação](../specs/2026-10-09-pose-attention-within-a-second.md) | POSE 1,39 → 0,91 s, outra instância 2,27 → 1,33 s | Reusar snapshot além dessa projeção; medir o corpus atual |

Comentários que chamam o custo remanescente de “intrínseco” ou paralelismo de
“única alavanca” descrevem a investigação daquele momento. GetSpec, cópias de
catálogo/grafo, clustering e subprocessos ainda oferecem outras alavancas no
código atual. Preservar as melhorias já entregues e corrigir essa generalização
na futura documentação técnica com evidência de perfil.

## 7. Candidatos de trabalho e critérios de aceite

Não são novas issues adjudicadas ou compromissos de release. A frente 10 deve
reconciliar cada candidato com specs existentes e ordenar a implementação.

| ID | Candidato | Prova necessária para aceitar |
|---|---|---|
| PERF-01 | Catálogo por operação em checks, mapa de slug e índice de scopes | Mesmo verdict/ordem/identidade em layouts e conflitos; leituras de catálogo limitadas à visão |
| PERF-02 | Perfil CPU/heap/subprocessos por produtor | Fixture e corpus fixos; tempo, CPU, RSS, leituras/parses/cópias, número Git e bytes |
| PERF-03 | Normalização única e clustering incremental | Mesmos candidatos exatos, ou aproximação explicitamente contratada; escala de entradas/texto |
| PERF-04 | Vistas de grafo imutáveis/compactas | Mesmo grafo/filtragem; mutação de consumidor não altera outro; alocações e RAM menores medidas |
| PERF-05 | Lote/reuso de atribuição Git | Histórico completo e refs alteradas mantêm claims/observed; limites/framing/falhas preservados |
| PERF-06 | API de domínio comum CLI/MCP | Paridade de resultados/erros e autorização; wrappers deixam de spawnar onde extraídos |
| PERF-07 | Deadline/cancelamento como execução inconclusiva | Probe não resulta em gate reprovado; erro tipado, contexto propagado e output parcial identificado |
| PERF-08 | Runtime opcional por worktree | Attach compatível, CI fallback, atualização/restart, permissões e isolamento demonstrados |
| PERF-09 | Geração e invalidação completas | Alteração mesmo mtime, rename, troca de branch, tests/matriz e perda de watcher não servem stale como atual |
| PERF-10 | Limites de cache e misses coordenadas | Alternância de projetos/gerações tem RAM limitada; apenas uma produção por chave concorrente |
| PERF-11 | Cursor vinculado à geração e consultas indexadas | Mutação entre páginas provoca restart explícito ou snapshot; limit evita leitura total quando possível |
| PERF-12 | Budget de concorrência e observabilidade | Quota CPU e jobs simultâneos não causam regressão de query; output determinístico e métricas por categoria |

Propor três entregas revisáveis: (1) medição e algoritmos por operação, incluindo
timeout; (2) domínio comum, vistas e budgets; (3) runtime opcional, condicionado
às garantias de consistência da frente 9. Preservar rollback standalone em todas.
Não misturar prova de reuso, adoção do daemon e eficácia em corpus real.

Qualificar cada entrega com corpus congelado em pelo menos três escalas,
execução inicial e repetida, múltiplos worktrees e cold process declarado.
Fixar SLOs somente após perfil: medir percentis com amostra suficiente, além de
máximo de memória, CPU total, hit/miss, startup e cancelamento. Um objetivo de
query subsegundo pode orientar experiência, mas não é SLA já comprovado para
qualquer `.pose`. Para gates, comparar cobertura/verdict antes de comparar tempo.
Uma aceleração que perde evidência não atende ao critério.

## 8. Encaminhamento e validação

Reproduzir as observações em um checkout destacado na revisão fonte, com o
binário 7.4.1 identificado pelo hash do snapshot. Os comandos abaixo escrevem
somente saídas temporárias; a fixture de consultas é criada em `/tmp` e não
modifica a baseline. Rodar cada medição isoladamente.

```sh
python3 .pose/results/2026-10-12-pose-front-08-check-benchmark.py \
  --root /caminho/baseline --binary /caminho/pose-7.4.1 \
  --workers default --output /tmp/check-default.json
python3 .pose/results/2026-10-12-pose-front-08-check-benchmark.py \
  --root /caminho/baseline --binary /caminho/pose-7.4.1 \
  --workers 8 --output /tmp/check-workers8.json
python3 .pose/results/2026-10-12-pose-front-08-benchmarks.py \
  --root /caminho/baseline --binary /caminho/pose-7.4.1 \
  --repeats 3 --output /tmp/queries.json
```


A frente 7 continua responsável pelo custo humano de comandos e saída útil;
esta frente responde por trabalho computacional e reuso. A frente 9 detalhará
consistência, write races, invalidação, falhas de Git, aliasing e recuperação.
As frentes 2 e 6 preservam autoridade das evidências e suficiência de release;
o cache/runtime não pode transformar resultado antigo em autorização atual.
A frente 10 consolidará backlog e composição das releases.

Validação concluída: `POSE_USAGE_SUPPRESS_CLI=1 POSE_CHECK_WORKERS=8 pose check
--strict` terminou com exit 0, SUCCESS e os mesmos 30 avisos da baseline. O
snapshot retém a saída integral. Os dois harnesses passaram na análise de
sintaxe Python; os links locais do relatório/índice e os 29 hashes de fontes
foram conferidos. Os probes foram executados no binário 7.4.1. Não executar
suíte Go/race detector sem Go disponível; esse limite permanece explícito.
Não houve execução de agentes externos nem chamadas de rede nos benchmarks.
