# POSE — frente 9: consistência, concorrência e recuperação de falhas

Data: 2026-10-12. Revisão fonte:
`f78074bc1b104fb26a1afef89f2a3049d20c4c9d`.
Escopo: garantias efetivas de leitura/escrita, concorrência entre executores,
retomada e falhas, incluindo condições necessárias ao motor residente da
[frente 8](2026-10-12-pose-front-08-resident-performance.md).
Esta análise não implementa mudanças de motor ou novas políticas; os candidatos
precisam ser reconciliados na frente 10 antes de specs/ADRs/releases.

## 1. Conclusão

O POSE tem mecanismos sólidos e específicos: journal de ações com lock e revisão,
bundles publicados sem sobrescrita, start com baseline e retomada por digest,
transferências com fases e revalidação sob locks ordenados. Não existe, porém,
um contrato uniforme de consistência para todo `.pose`. Operações sobre o mesmo
arquivo podem usar locks diferentes; vários arquivos são publicados separadamente;
um digest de inputs não é uma transação de leitura nem um compare-and-swap no
momento de escrever.

Foram reproduzidos nove grupos de casos em fixtures descartáveis, com o binário
oficial 7.4.1. Os principais defeitos concretos são: supersessão parcialmente
gravada que um retry do mesmo ID não completa; journal que aceita conteúdo extra
na mesma linha; checkpoint corrompido apresentado como ausente; falha de Git
status apresentada como árvore limpa/coerente; publicação parcial de índices;
e state refresh que retorna erro depois de alterar o estado. Há também resultados
positivos: dois writers de ações não conseguiram registrar respostas na mesma
revisão, retries iguais não duplicaram eventos e o start recusou lock existente.

Antes de ampliar residência/concorrência, padronizar publicação, identidade dos
inputs, lock compartilhado entre writers e resultado parcial/inconclusivo. Não
resolver a questão com um mutex do daemon: CLIs independentes, editores, Git e
CI também podem escrever. Não exigir uma transação global para qualquer consulta;
escolher a garantia conforme o tipo de artefato e consequência da operação.

## 2. Método e grau de evidência

O [snapshot](../results/2026-10-12-pose-front-09-consistency.json) registra a revisão,
39 fontes com SHA-256/URLs imutáveis, comandos, respostas e fixtures. O
[harness](../results/2026-10-12-pose-front-09-probes.py) exercita o binário existente
em repositórios Git sintéticos sob `/tmp`, sem alterar o checkout analisado.
Usar identidade genérica, revisão desativada e capacidade de start explícita.
O lock de supersessão é mantido por um processo Python separado do processo POSE;
a disputa de respostas usa duas CLIs independentes iniciadas por uma barreira.

Distinguir resultados observados de inferências de código. A falha de publicação
é provocada por diretório no caminho esperado para um arquivo, não por queda de
energia. O estado intermediário de release é construído manualmente segundo a
ordem de escrita do código; não houve kill real durante `prepare`. O lock de
start também é criado deliberadamente na fixture; sua presença representa o
estado que um crash pode deixar, sem simular a morte do processo.

Não há Go disponível. Não executar ou alegar suíte Go, race detector, fault
injection interno dos testes ou prova de heap aliasing em runtime. A execução
foi Linux local; Windows, outros sistemas e filesystems de rede não foram
qualificados. O programa usa `fcntl` e não promete portabilidade. Os achados da
frente 8 são evidência anterior identificada, sem repetir seu benchmark pesado.

## 3. Classificar os dados antes de escolher consistência

| Classe | Exemplos | Garantia necessária | Recuperação admissível |
|---|---|---|---|
| Fonte mutável de intenção | Spec, roadmap, política, seção curated | Não perder edição; autorizar sobre inputs identificados | Conflito explícito, nova leitura/plano |
| Evidência imutável | Bundle/attestation selado, manifest congelado | Publicar completo e não sobrescrever outra identidade | Reusar payload igual; recusar colisão |
| Journal autoritativo | Ações, eventos de release | Sequência/transição válidas, retry idempotente, erro de corrupção visível | Retomar intenção registrada sem inventar evento |
| Projeção reconstruível | Índices, seções derived, caches | Distinguir geração atual, stale e unavailable | Recalcular; não promover snapshot parcial a atual |
| Estado de operação | Start record, checkpoint, transfer receipts | Fase/payload verificáveis, efeito parcial navegável | Resume/reconcile por precondições, não só flag |
| Coordenação transitória | Lock, endpoint IPC, worker | Escopo/owner/lifetime conhecidos | Liberar após verificar executor; nunca por idade apenas |

O project-state mistura curated e derived. Reconstruir o arquivo inteiro sem
comparar sua versão pode apagar edição humana concorrente, mesmo que todo o
arquivo seja publicado por rename. Preservar essa distinção na API e no repair.
`stats replay` é um relatório contrafactual somente leitura, conforme
[`governance_replay.go`](../../pose-mcp/internal/pose/governance_replay.go);
não é executor de recovery de operações nem revoga atestações históricas.

## 4. Mapa das garantias efetivas

| Caminho | Mecanismo atual | O que cobre | Limite relevante |
|---|---|---|---|
| Append de ação | Lock de arquivo, reler journal, revisão, idempotency key, Sync | Writers cooperantes do mesmo journal | Checks de política feitos antes do lock; supersessão envolve dois journals |
| Bundle/attestation | CreateTemp exclusivo, Sync, Link sem overwrite | Leitor não vê payload parcial publicado; colisão não substitui | Concorrência de replay pode retornar already exists; durabilidade do diretório não demonstrada |
| Start | Lock por spec via O_EXCL, baseline, digests before/after | Start cooperante e retomada da mesma intenção | Lock abandonado exige intervenção; outro writer pode ignorar esse lock |
| Transferência | Locks de roots ordenados, validação de inventário/HEAD, arquivos/receipts por fase | Transferências cooperantes, retomada e autoridade staged | Não é transação distribuída; outros comandos não usam os mesmos locks |
| Close simples | Reavalia gates; contexto opcional; temp exclusivo + rename | Evita publicar arquivo lifecycle parcialmente escrito | Sem lock/CAS do conteúdo final contra editor ou outro comando |
| Close plan | Replaneja após etapas; checkpoint e resume | Não toma Completed como autorização cega para pular gate | Checkpoint direto, erro de persistência ignorado, corrupção como ausência |
| Index | WriteAtomic por arquivo | Troca individual de alguns arquivos | Temp fixo, ordem de map variável, conjunto sem commit de geração |
| State refresh | Renderiza estado, publica, appende history/log | Projeção e diagnóstico de falha de append | Erro pode vir depois do efeito; read-modify-write sem CAS |
| Release prepare | Moves, notes e manifest; rollback local | Retorna arquivos em alguns erros capturados | Sem journal de operação/lock; rollback best effort; crash não executa rollback |
| Release record | Valida histórico, deduplica, O_APPEND | Retry sequencial e transição daquele histórico lido | Sem lock envolvendo validação+append, sem Sync explícito |

Referências centrais: [`action_request.go`](../../pose-mcp/internal/pose/action_request.go),
[`review_bundle.go`](../../pose-mcp/internal/pose/review_bundle.go),
[`start.go`](../../pose-mcp/internal/pose/start.go),
[`spec_transfer.go`](../../pose-mcp/internal/pose/spec_transfer.go),
[`review_closeout.go`](../../pose-mcp/internal/cli/review_closeout.go),
[`maintenance.go`](../../pose-mcp/internal/cli/maintenance.go),
[`index.go`](../../pose-mcp/internal/cli/index.go),
[`state_refresh.go`](../../pose-mcp/internal/cli/state_refresh.go) e
[`release_lifecycle.go`](../../pose-mcp/internal/cli/release_lifecycle.go).

### 4.1 Atomicidade de publicação não é exclusão nem durabilidade completa

`writeAtomic` usa `.<basename>.tmp` fixo. Dois writers podem abrir/truncar o mesmo
temporário, um publicar os bytes do outro e o segundo falhar no rename. Essa é
uma sequência possível pelo código; não foi produzida uma corrida desse helper
nativo com controle de interleaving. O probe de index demonstra um problema
diferente e determinístico: parte do conjunto já foi publicada antes da falha.

Trocar por temporário exclusivo remove a colisão de staging. Não evita lost
update: dois read-modify-write de uma spec ou de uma seção curated ainda podem
publicar versões diferentes sobre o mesmo original. É necessário validar versão
sob coordenação compartilhada, ou retornar conflito/rebase. O rename final é um
ponto de publicação por arquivo, não um lock de negócio.

Bundles têm desenho melhor para imutabilidade: Write+Sync no temporário e Link
sem sobrescrita. Start/transfer têm Sync em algumas escritas, mas não foi
identificado Sync do diretório pai nesses publishers. Não equiparar “Sync de
arquivo” a prova de persistência dos nomes após power loss. O contrato deve
separar crash de processo, restart da máquina e corrupção do storage, com
qualificação por filesystem; esta análise não mede esses últimos dois cenários.

O_APPEND com um Write por linha reduz o risco de disputa pelo offset. Não torna
atômico o par “ler histórico, verificar transição/idempotência, appendar”. O
release record faz esse par sem o lock de ação. Duas operações podem verificar
o mesmo histórico e ambas registrar. A possibilidade é fundamentada no código,
sem alegar que o probe de ações reproduziu a mesma coisa em releases.

### 4.2 Transferência tem recovery real, mas escopo cooperante

O motor registra plan/receipts e fases `planned`, `prepared`, `source-retired`,
`activated`. Mantém destino staged/blocked antes de ativar autoridade e permite
resume. Após adquirir locks dos roots, valida inventário, revisões e preimagens;
isso é mais forte do que comparar um plano apenas no preview. Os testes
`TestSpecTransferInterruptedOperationsResumeWithoutDuplicateAuthority`,
`TestSpecTransferResumeBlocksChangedReferenceInventory` e
`TestSpecTransferConcurrentApplySerializesWriters` expressam essas propriedades.
Foram inspecionados, não executados aqui.

Os locks impedem outras transferências que aderem ao protocolo. Não impedem uma
CLI close/start que usa outro namespace, um editor ou checkout. O CAS em
`writeTransferFile` lê/verifica preimagem antes de escrever temp/rename; um
writer não cooperante ainda pode alterar o path entre verificação e publicação.
Projetar o coordenador comum sem remover as fases, a autorização de todos os
projetos ou os bloqueios de inventário incompleto. Recusar escritas concorrentes
é preferível a ativar duas autoridades; a disponibilidade parcial deve ser
reportada por projeto/fase, não mascarada como atomicidade distribuída.

Proteções de paths/symlinks recusam layouts estáticos inseguros em vários
caminhos. Checks separados de path e abertura não provam resistência à troca de
path durante I/O. Não foi executado exploit de symlink concorrente; revisar
essa janela quando uniformizar a camada de escrita, mantendo a autorização
por root e a identidade do worktree da frente 8.

## 5. Nove grupos de probes e implicações

### 5.1 Respostas concorrentes e idempotência: comportamento correto observado

Duas CLIs resolveram uma ação contra revision 1 com chaves distintas. Uma
teve exit 0, a outra exit 1; o journal ficou em revision 2. Repetir a escrita
vencedora com a mesma chave e revision antiga retornou exit 0 sem novo evento.
Usar a mesma chave com answer diferente retornou `action-idempotency-conflict`.
Esse resultado corresponde à releitura sob lock e à checagem de idempotência
antes da revisão em `appendActionEvent`.

Não generalizar a prova a todo writer. A ação bloqueia o arquivo de journal e
revalida sua sequência; a política/assinatura/contexto são verificados antes de
entrar nessa região. Decidir explicitamente qual versão de política rege a
aceitação e verificar o digest correspondente no ponto de commit. A derivação
já invalida satisfação quando o subject muda; preservá-la e acrescentar um
contrato claro de linearização, em vez de tratar autorização como booleano eterno.

### 5.2 Supersessão parcial: retry não conclui o segundo efeito

Abrir a substituta escreve seu journal primeiro e appenda `superseded` no
journal antigo depois. Mantendo um flock no antigo, o comando retornou erro
“opened but ... was not marked superseded”. Liberar o lock e repetir a abertura
com exatamente o mesmo novo ID retornou sucesso, mas a antiga permaneceu open.
A listagem continha as duas abertas.

A causa está no retorno antecipado de `OpenActionRequest`: se já existe nova
request com mesmo digest, devolve existing antes do append ao journal anterior.
O harness verifica igualdade do novo ID. Como a CLI gera identidade incluindo
requested_at em segundos, não inferir idempotência por texto igual em outra
hora; se o relógio cruza o segundo entre os comandos, a fixture é repetida de
forma limitada. Esse detalhe também indica que um retry do cliente precisa
reter a identidade/intenção da operação, não apenas reenviar flags com now novo.

Corrigir retomando o efeito pendente com a mesma chave `supersede:<id>`, com
validação do predecessor e locks ordenados dos journals. Registrar a intenção
de supersessão/fase, ou derivar seu estado transacional, sem ocultar uma
substituta parcial. Não apagar nenhuma request nem presumir que uma aprovação
anterior satisfaz automaticamente a nova pergunta.

### 5.3 Journal incompleto é recusado; JSON extra na linha é aceito

Acrescentar uma próxima linha truncada produziu exit 1 e `line 2: unexpected EOF`.
Isso é fail closed e precisa continuar assim: não ignorar silenciosamente um
possível answer/waiver incompleto para apresentar estado aparentemente válido.

Acrescentar um segundo objeto JSON após a linha de abertura válida retornou exit
0 e revision 1. `parseActionJournal` decodifica um objeto, mas não exige EOF
naquela linha. O dado extra não foi interpretado nem recusado. Exigir consumo
integral, limites e política de chaves duplicadas compatível com os demais
artefatos estritos. Uma rotina de repair deve preservar bytes/posição/hash e
pedir adjudicação do tail, sem completar resposta, remover evento válido ou
assumir que toda corrupção é só interrupção no último byte.

### 5.4 Start com lock preexistente e retry: proteção e custo operacional

Uma fixture com `.pose/starts/.work.lock` recusou apply com `start-in-progress`,
nomeando o arquivo e exigindo confirmar que nenhum start está ativo. Remover
essa trava sintética e executar o mesmo plano registrou a baseline e mudou a
spec; repetir o digest retornou sucesso sem uma segunda baseline.

O comportamento é conservador e intencional. O arquivo O_EXCL pode sobreviver a
um crash, enquanto o flock de journal/transfer é liberado pelo kernel ao fechar
os descritores/processo. Não remover locks reais por TTL ou mtime. Um recovery
melhor deve informar worktree, operação, fase e executor/host/boot, distinguir
PID reutilizado e provar que o owner não está ativo antes de limpar a trava.
A remoção no script é exclusiva da fixture descartável, não recomendação de
repair automático em projeto real.

### 5.5 Checkpoint corrompido perde diagnóstico, não prova de gate

Um checkpoint JSON sintético válido apareceu em `close --plan` como interrupted.
Após truncá-lo, o comando retornou exit 0 e exatamente o mesmo stdout do caso
sem checkpoint. `ReadCloseoutCheckpoint` retorna false tanto para ausência
quanto para erro de leitura/parse, e não valida schema/scope/plan digest ali.
`cmdClosePlan` também ignora erro de `WriteCloseoutCheckpoint`.

Esse achado não prova que Completed pula checks: o planner recompõe o estado
do repositório e replaneja depois das etapas. A retomada não depende cegamente
da lista de passos do checkpoint, o que é positivo. O problema é a invisibilidade
do estado incompleto e a promessa de recovery sem diagnóstico confiável.
Distinguir absent, valid, corrupt, unavailable e incompatible. Separar erro do
gate do erro de checkpoint e persistir progresso de forma segura. Um checkpoint
antigo deve levar a reconciliação, não ser descartado nem usado como autorização.

### 5.6 Git status indisponível aparece como clean/coherent/complete

Modificar uma spec tracked produziu `worktree_dirty=true` na consulta normal.
Um shim local fez somente `git status` retornar 97, delegando o resto ao Git real.
A consulta retornou exit 0, `worktree_dirty=false`, `coherent=true`, `complete=true`
e perdeu a limitação de árvore dirty. A revisão HEAD permaneceu válida.

Em `CurrentObligationSnapshot`, falha de rev-parse já degrada coerência;
a de status não é representada. Em `takeWorkingTreeSnapshot`, erro também pode
se tornar snapshot `ok=false` que `change` interpreta como sem mudança. Separar
unknown/unavailable de clean, propagar cobertura e impedir que erro de Git
autorize gate como se nada tivesse mudado. Não “corrigir” indicando dirty sempre:
isso ainda inventa um fato e impede distinguir execução inconclusiva de conflito.

### 5.7 Release intermediária: detecção existe, retomada automática não

O dry-run normal anunciou um fragmento. Construir a fase intermediária com o
fragmento já em archive, diretório release criado e manifest ausente fez retry
`prepare --apply` falhar com `empty release is forbidden by policy`. O fragmento
continuou arquivado e nenhum manifest foi criado. Se outros pending fragments
existirem, outro caminho pode recusar archive já existente; não contar apenas
com a primeira mensagem para diagnosticar a fase.

`release check` detectou manifest ausente, mas retornou exit 0 no modo padrão;
com `--strict`, exit 1. Isso é a semântica explícita desse comando, não bug de
classificação como o timeout da frente 8. CI precisa consumir `valid/gaps` ou
usar strict. Um estado parcial detectável ainda não é uma operação resumível.
Propor journal de prepare com inventário/digests, staged publication e reconcile.
Não usar backfill para fabricar testemunho de um freeze incompleto.

### 5.8 Índices parcialmente publicados após erro

Plantar um diretório em `.services.json.tmp` fez `index --json` retornar exit 1.
Nesta amostra, cinco dos sete arquivos já tinham nova geração: repo-map,
spec-graph, roadmaps, delivery-integrity e releases. Services e packages mantiveram
a marca antiga. A ordem pode variar, pois outputs é um map Go. Remover o obstáculo
sintético e repetir completou a publicação; não houve rollback do primeiro run.

Os arquivos antigos na fixture eram sentinelas, não índices de domínio reais;
a prova é sobre fronteira de publicação, não veracidade do grafo sentinel.
Sem descriptor de geração, consumidores podem combinar arquivos de runs distintos.
Publicar um conjunto em diretório de geração e trocar um único descriptor, ou
estabelecer manifest/digests comuns com rejeição de mistura. Manter compatibilidade
dos paths atuais na migração e evitar que fallback trate missing index como zero.

### 5.9 Refresh com falha de history: efeito ocorreu antes do erro

Inicializar state, alterar o status da spec e substituir `history.jsonl` por um
diretório fez `state refresh` retornar exit 1. O hash de project-state mudou.
O código publica estado antes de append history; a mensagem de erro isolada
não diz que o estado já foi alterado. Esse resultado não prova corrupção de
conteúdo, mas prova que failed não equivale a nenhum efeito.

Expor receipt com committed effects, phase, erro secundário e caminho de resume.
Preservar curated com CAS e distinguir status de publicação de status de audit
log. Um retry só é seguro se não repetir decisões/eventos e não sobrescrever uma
edição posterior. Não devolver sucesso genérico para esconder o erro de history.

## 6. Leituras, snapshots e caches: condições para a residência

`ProjectObligations` captura o snapshot antes de rodar produtores concorrentes.
O catálogo por operação reduz releituras, mas corpos, políticas, bundles e
outros inputs podem ser lidos em momentos diferentes. Não há barreira de
filesystem que congele todo `.pose`. `coherent=true` não prova que a projeção
veio de uma única geração, mesmo sem erro de Git.

`ObligationSnapshotChanges` compara identidade, HEAD, política, contrato,
contexto e índice, mas não constitui hash integral de todos os inputs lidos.
Seu próprio comentário admite alterações dentro de uma árvore já dirty.
`directoryDigest` pula erro de leitura de arquivo e cobre somente arquivos
regulares diretos. Não promover esses digests a fence universal para qualquer
gate/resposta. Definir dependências por operação, guardar os bytes efetivamente
lidos e revalidar a closure antes de autorizar escrita. Mudança de resultado,
evidência, ação, curadoria ou arquivo untracked pode importar sem novo HEAD.

Na frente 8, a leitura real de spec após alteração de mesmo tamanho/mtime
retornou conteúdo novo; o cursor ordinal duplicou item após inserção entre
páginas. Um watcher deve sugerir invalidação, não representar prova de frescor.
Perda de eventos, replace atômico, branch/reset e alterações externas exigem
reconciliação. O cursor deve nomear geração, ordem e filtros, ou recusar página
obsoleta. Um snapshot para navegação pode ser mantido por prazo/bytes limitados;
um gate não pode usar indefinite stale snapshot como autorização atual.

O cache de grafo copia slices externos, mas `DeliveryIntegrityNode.Attributes`
é map e a cópia de Nodes é rasa. O comentário afirma cópia de todo campo gravável;
`TestDeliveryGraphCopyCoversEveryField` não demonstra isolamento de todo objeto
aninhado. Há potencial de aliasing se um consumidor mutar Attributes; não foi
reproduzida uma mutação de consumidor atual nem uma data race nativa aqui.
Fazer teste de mutação profunda entre leituras e revisão dos campos aninhados,
ou adotar vistas imutáveis efetivas. Mutex do cache protege acesso ao cache,
não uso de objetos mutáveis devolvidos.

O MCP stdio processa sua sequência de mensagens, mas não serializa CLIs externas.
HTTP admite chamadas independentes; transport/auth/contexto não são locks de
artifacts. Root registry mutex, cache mutex e cursor store mutex não garantem
consistência de writes do projeto. Não confundir event cursor de transporte com
checkpoint de operação ou generation de catálogo.

## 7. Contrato recomendado de consistência e recuperação

### 7.1 Fixar a semântica de execução

| Estado de execução | Significado | Conduta de cliente |
|---|---|---|
| Completed/pass ou completed/fail | Gate terminou sobre inputs identificados | Interpretar verdict e cobertura |
| Conflict | Preconditions, revisão ou owner mudaram | Reabrir contexto/plano; não repetir decisão automaticamente |
| Incomplete/unavailable | Produtor, storage ou Git não respondeu | Repetir leitura quando causa resolvida; não inferir ausência |
| Cancelled/deadline | Avaliação não concluiu | Preservar output parcial marcado; não reprovar domínio |
| Partially committed | Alguns efeitos ocorreram; fase final não | Consultar receipt e retomar a mesma intenção |
| Corrupt/incompatible | Estado persistido não pode ser interpretado | Quarentena/diagnóstico/reconciliação explícita |

A frente 8 já reproduziu timeout de 30 s retornando `passed=false, exit_code=-1`
sem erro MCP. Incorporar a distinção acima à API comum, mantendo compatibilidade
com clients atuais. Gate negativo de domínio é resultado legítimo; execução
inconclusiva não o é. Incluir operation ID, input digest/generation, effects,
retryability e remediation específica. O exit code pode continuar resumindo
falha operacional, mas o payload precisa dizer se algo já foi publicado.

### 7.2 Padronizar o write protocol sem tornar tudo uma transação global

Adquirir coordenação por worktree/recurso em ordem canônica. Compartilhar o
namespace entre start, close, transfer, policy e index quando seus efeitos
conflitam. Reler inputs/preimagens e autorização relevante dentro dessa região;
validar digests/expected revisions; preparar arquivos exclusivos; publicar;
registrar fase/receipt; devolver o resultado correspondente ao que ocorreu.

Imutáveis devem permanecer create-if-absent com comparação de payload no retry.
Mutáveis precisam de CAS/versão e conflito explícito. Para múltiplos arquivos,
usar staging e descriptor de commit de geração, ou saga com receipts verificáveis
quando houver raízes/fases independentes. Sync de arquivo/diretório e garantias
de filesystem precisam fazer parte do contrato da classe autoritativa; não
impor esse custo a caches descartáveis sem justificativa.

Um editor não cooperante pode ignorar advisory locks. Revalidar preimagens e
autoridade, detectar alterações externas, declarar o limite da coordenação e
fornecer reconciliação. Não prometer serialização perfeita de Git/editor só por
usar flock. Não permitir que uma invalidação pendente seja perdida na confirmação
de escrita do motor residente; ligar publicação e avanço de geração ao receipt.

### 7.3 Recuperar pela evidência, com UX que evita tentativas cegas

Expor `status` de operação com fase, roots, preimagens/postimagens, erros e efeitos
confirmados. `resume` deve manter ID, validar dados atuais e executar somente
efeitos pendentes comprovados. `reconcile` compara intenção e estado, propondo
repair reviewable para dados divergentes. Um rollback só pode remover arquivos
que ainda correspondem ao payload produzido pela operação; não apagar diretórios
indiscriminadamente após falha de compensação.

No release rollback atual, erros de Rename de retorno são ignorados e depois
os diretórios são removidos. Se uma compensação falhar, isso pode apagar a cópia
arquivada restante; é hipótese fundada nesse ramo, sem falha de rollback injetada
nesta execução. Preservar material e registrar rollback incomplete. Em close
plan, persistir falha de checkpoint separadamente sem ocultar falha de gate.
Em supersessão, mostrar predecessor e substituta até reconciliar o segundo append.

Não limpar journals corrompidos, locks antigos ou diretórios parciais só para
fazer check passar. Repair deve ser seletivo, verificável e autorizado segundo a
natureza do efeito. Não adicionar aprovações/waivers ou manifest retrospectivo
como compensação automática. Preservar a autoridade das frentes 2, 4 e 6.

## 8. Candidatos, dependências e aceite

Candidatos, sem adjudicação de issues ou compromisso de release. Reutilizar specs
de start, transfer, review bundle, closeout recuperável e release lifecycle antes
de criar novos itens. A frente 10 consolidará prioridade final.

| ID | Candidato | Evidência atual | Critério de aceite |
|---|---|---|---|
| CONS-01 | Retomar supersessão no retry da mesma intenção | Probe + retorno antecipado | Lock/falha entre journals deixa fase explícita; retry mesmo ID converge uma vez |
| CONS-02 | Parser integral de journal e recovery de tail | Extra JSON aceito, truncado recusado | Exigir EOF por linha e seq/schema; diagnóstico preserva bytes sem inventar resposta |
| CONS-03 | Unknown de Git e cobertura coerente | Probe clean/coherent/complete com status 97 | Status/HEAD/inputs falhos não autorizam clean ou complete; erro tipado |
| CONS-04 | Publisher comum com temporário exclusivo | Helper fixo e tipos distintos de escrita | Writers não compartilham staging; mutáveis não perdem updates; imutáveis não sobrescrevem |
| CONS-05 | Geração de conjunto de índices | Cinco novos/dois antigos no probe | Leitor vê geração completa ou estado indisponível; retry/crash não mistura versões |
| CONS-06 | Coordenador/CAS por recurso entre comandos | Locks start/transfer/ação distintos | Start vs close vs transfer vs editor/CI tem conflito/fase explicável; nenhuma dupla autoridade |
| CONS-07 | Checkpoint robusto e receipt parcial | Parse vira absent; write error ignorado | Corrupção/erro de persistência visíveis; resume revalida inputs e progresso |
| CONS-08 | Prepare/record de release recuperáveis | Estado construído não retoma; append sem lock | Crash/concorrência por fase preservam inventário; retry evento não duplica nem viola transição |
| CONS-09 | State/history e curadoria com CAS | Arquivo mudou apesar de exit 1 | Partial commit explicitado; history reparável; curated concorrente preservada |
| CONS-10 | Read view/generation e dependências completas | Snapshot anterior a leituras concorrentes | Alteração de qualquer input relevante é detectada; gates não aceitam mistura de gerações |
| CONS-11 | Isolamento profundo de caches | Attributes map copiado rasamente | Mutation test e race detector; memória limitada por worktree; sem contaminação cruzada |
| CONS-12 | Lifecycle/recovery de locks e runtime | Start lock exige operador; ausência de contrato universal | Owner verificável; daemon restart/fallback mantém geração e intenção sem quebrar lock vivo |
| CONS-13 | Resultado de execução comum, deadlines/cancelamento | Probe da frente 8, efeito parcial desta | Typed outcome distingue verdict, timeout, conflict e committed effects em CLI/MCP |

Atacar primeiro defeitos determinísticos de CONS-01/02/03/07 e o resultado de
execução, em escopos pequenos. Consolidar publisher, geração de índices e
coordenação depois, sem abandonar standalone. Condicionar cache residente e
jobs concorrentes às provas de geração/CAS/isolamento. A arquitetura comum pode
ser extraída incrementalmente; não é necessário interromper melhorias locais
até que todo o runtime opcional esteja pronto.

Qualificar com matriz de interleavings e falhas por fase: duas escritas iguais,
diferentes e com mesma chave; cancelamento antes/depois de commit; lock perdido;
falha Write/Sync/Link/Rename/history; disco cheio; tail truncado; checkpoint
incompatível; kill antes/depois de cada receipt; restart; branch/reset; rename e
mesmo tamanho/mtime; dois worktrees de mesmo project_id; roots federadas
indisponíveis; ambiente Windows e filesystem suportado. Verificar invariantes,
não só exit code: ausência de lost update, payload imutável intacto, ordem de
journal, no máximo uma autoridade, curadoria preservada e erro incompleto visível.
Separar testes de crash de processo de qualificação de power loss.

## 9. Encaminhamento e validação

Esta frente responde por consistência do motor e recuperação; a frente 8 por
performance/arquitetura compartilhada, a frente 7 pela UX das ações e a frente 2
pela autoridade das evidências. A frente 10 transforma candidatos reconciliados
em backlog e releases, incluindo o impacto de parcialidade sobre suficiência
investigado na frente 6.

Reproduzir os nove grupos em Linux com o binário 7.4.1 do hash registrado:

```sh
python3 .pose/results/2026-10-12-pose-front-09-probes.py \
  --binary /caminho/pose-7.4.1 --output /tmp/consistency-probes.json
```

O programa só escreve fixtures e a saída solicitada. Os IDs/commits/timestamps
sintéticos mudam entre execuções. A ordem/quantidade de índices publicados antes
do obstáculo pode variar; comparar a propriedade de publicação por arquivo,
não fixar cinco como comportamento universal. Não aplicar a remoção da trava
sintética ou o dano de arquivos em repositório real.

Validação concluída: `POSE_USAGE_SUPPRESS_CLI=1 POSE_CHECK_WORKERS=8 pose check
--strict` terminou com exit 0, SUCCESS e 30 avisos já existentes. A saída integral
está no snapshot. O harness foi executado e seus nove grupos/assertions passaram;
a sintaxe Python, os 39 hashes de fontes e os links locais do relatório/índice
foram verificados. `git diff --check` não encontrou erros de whitespace.

Não executar suíte Go/race detector sem Go disponível. A validação não comprova
segurança de todas as interleavings, durabilidade em power loss ou portabilidade;
esses requisitos permanecem explicitados nos critérios de aceite.
