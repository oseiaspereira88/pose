# B13 — revisão independente de journal, checkpoint e lock

Data: 2026-10-16. Decisão: **changes-requested**. PR #187, branch
`impl/pose-b13-journal-checkpoint-lock-recovery`, HEAD revisado
`c282cabb86b0d902bec60e389784e633ea8296ed`; base `b8c33dad`.

Revisor autorizado pelo maintainer: `agent:independent-gpt-6.1-sol-review`,
execução Codex/OpenAI, de outro vendor que o implementador Claude/Anthropic,
sessão `296d1fcb-e989-40ae-9362-cf3514db8b26`, `agent:claude-opus-5-5`.
No julgamento de scope, registro que `agent:independent-` é a declaração
exigida pelo motor até `pose-delegated-review-run-provenance` substituí-la.
Identidade declarada; no pose-dist as atestações não são assinadas. Este parecer
não é uma atestação selada nem uma prova criptográfica de independência.

## Findings

A varredura continuou depois do primeiro defeito e cobriu a superfície completa
do diff. Os três findings médios impedem aprovação. Nenhum foi corrigido.

### B13-F1 — medium — aliases JSON sobrescrevem campos já validados

Local: `pose-mcp/internal/pose/journal_integrity.go:109`, com
`review_bundle.go:1849` e `closeout_plan.go:313`. Requisitos R1, R3, R5 e R9.

A checagem de duplicatas compara nomes exatos; o decoder de structs Go aceita
nomes sem distinguir maiúsculas. Um journal de abertura íntegro com
`"schema_version":999,"SCHEMA_VERSION":1` ou `"seq":999,"SEQ":1`
continua legível com exit 0 e revision 1. São dois campos que escrevem o mesmo
campo efetivo; a última interpretação vence sem diagnóstico.

O mesmo desvio alcança o checkpoint: `"completed":["context","seal"]`
sozinho causa `checkpoint-progress-mismatch`, sem passos executados. Acrescentar
`"COMPLETED":[]` faz esse checkpoint ler como valid e o resume entrar na etapa
`evidence`. Na fixture com trace corretamente dentro de Validation, essa etapa
falha por falta de matriz, mas já foi executada em vez de bloquear o checkpoint.
Não houve `--apply --digest` autorizando substituição. O primeiro probe, com o
trace original do driver, bloqueava no trace e foi refinado para separar causas.

Ação: exigir nomes exatos dos campos do schema ou recusar aliases que se ligam
ao mesmo campo antes do decode, inclusive em structs aninhadas. Cobrir journal,
checkpoint e consumidores com regressões adversariais. Evidência externa:
`extra-probes.json`, `checkpoint-alias.json`; hashes no JSON compacto.

### B13-F2 — medium — evento fora do schema é incorporado e ignorado na view

Local: `pose-mcp/internal/pose/journal_integrity.go:95`, com
`action_request.go:591`. Requisitos R1, R2 e R9.

Após uma abertura íntegra, acrescentar a linha terminada
`{"schema_version":1,"seq":2,"type":"not-an-event"}` resulta em show/list
com exit 0 e revision 2. O parser valida versão e sequência, incorpora o evento
e o fold, sem um caso para esse tipo, conserva silenciosamente o estado anterior.
Não há `ActionJournalError`, posição ou recusa por schema desconhecido. Um evento
que o motor não sabe aplicar fornece uma view aparentemente íntegra.

Essa permissividade já existia antes de B13, mas permanece no leitor que esta
spec qualifica como integral e fail closed; não é atribuída como regressão nova.
Ação: validar o discriminador contra os tipos suportados antes de aceitar a
linha, preservando diagnóstico e recusa de append; cobrir domínio, CLI e MCP.
Evidência: `extra-probes.json`.

### B13-F3 — medium — modo recover-lock vazio desaparece e permite apply

Local: `pose-mcp/internal/cli/start.go:55`, também linhas 68 e 83.
Requisito R9 e contrato de modos exclusivos herdado de B12.

Na fixture draft, obter o digest do preview e executar
`start spec:work --recover-lock "" --apply --digest <digest> --json` retorna
exit 0, registra o start e move a spec para in-progress. O argumento vazio passa
na validação; a contagem de modos usa `recoverLock != ""` e apaga a presença da
flag. `--recover-lock ""` sozinho retorna preview com exit 0.

Ação: registrar presença da flag separadamente do valor, recusar argumento vazio
e qualquer combinação com apply/status/cancel antes de chamar o domínio. Cobrir
as duas ordens das flags e provar ausência de efeitos. Evidência:
`start-empty.json`, contendo argumentos, retornos e postimagem sintéticos.

### B13-F4 — low — o teste do limite de linha sobrevive sem o limite

Local: `pose-mcp/internal/pose/recovery_integrity_test.go:51`. Requisito R1/V1.

Retirar apenas a guarda `len(text) > maxActionJournalLine` por Go overlay deixa
`TestAJournalIsReadWholeOrNotAtAll` passar. A fixture oversized contém somente
espaços; continua inválida como linha vazia, independentemente do limite. O teste
não distingue o mecanismo que afirma cobrir.

Ação: usar evento válido preenchido até o limite e um byte além, verificando a
causa específica. A implementação atual passou nesse probe independente: 1 MiB
aceito e 1 MiB + 1 recusado. É uma lacuna de regressão, não uma falha observada da
guarda em produção. Evidência: `mutations/journal-line-limit.log`, `boundaries.json`.

## Rules applied during review

- Change type: mixed, correção de persistência/concorrência com contratos aditivos.
- Workflow: `.pose/workflows/review.md`; skills `pose-review` e `pose-spec-closeout`.
- `.pose/rules/backend-go.md`: erros de I/O, parse, exclusão mútua, concorrência e race.
- `.pose/rules/security.md`: integridade de dados externos, binding dos campos e ausência de IDs crus de máquina no contrato publicado.
- `.pose/rules/documentation-style.md`: ADR, manuais, changelog e clareza dos limites.
- `.pose/rules/delivery-evidence.md`: validação executada e distinção entre implementação e entrega aprovada.
- `.pose/rules/knowledge-governance.md`: consulta às decisões anteriores; não houve aceitação de risco nem novo handoff.
- Frontend, infraestrutura e regras de outros backends: não aplicáveis ao diff.

Foram lidos AGENTS aplicáveis, partes de review/closeout do POSE.md, workflow,
spec, ADR, relatório front-09 (5.3–5.5 e CONS-02/07/12), decisões disponíveis em
knowledge e diff completo `origin/main..HEAD`. Não foram alterados código,
testes, Requirements, Technical Plan, policy, capabilities ou adoção.

## Evidência e cobertura

A CLI foi compilada da branch com o comando solicitado. Como `go -C pose-mcp`
resolve `-o ../pose-review` dentro do módulo, o executável resultante na raiz foi
movido para `../pose-review`, fora do repositório; todos os comandos nativos desta
revisão usaram esse binário. Nenhuma variável de seleção de projeto foi exportada.

Passaram `go vet ./...`, `go test ./... -count=1`, race nos três pacotes tocados e
`GOOS=windows`/`GOOS=darwin go build ./...`. A validação estrita global executou
135 checks, todos pass, sem fail/error/skip, sobre o HEAD revisado. Inclui
`quickstart-real-lifecycle-integration` e a integração B13. Os testes de quickstart
executam apply → waiting → attest → resume em fixtures, sem atestar esta spec.

Os probes P01–P17 reproduziram os resultados publicados, inclusive os controles
P06/P11/P12. A comparação exclui IDs, paths temporários, timestamps, hashes e
revisões Git sintéticas variáveis. P11/P12 do driver original bloqueiam no trace
antes de completar o closeout: sua equivalência não substitui a prova do fluxo
completo fornecida pelo quickstart.

Os nove journals reais de `.pose/actions/` foram lidos com sucesso. Os testes
conferem linha, offset, tamanho, SHA-256, valid_events, consumo integral, schema,
sequência, cauda, duplicatas, controles escapados e ausência de reparo/append sobre
corrupção. Foram encontrados dois leitores de produção, LoadActionRequest e
appendActionEvent, ambos usando o parser comum; F2 mostra a lacuna semântica desse
parser. Os limites exatos de linha foram qualificados com JSON válido.

Foram executadas 26 mutações de produção por `go test -overlay`, mantendo
intactos os arquivos da branch: 25 falharam por asserção, uma sobreviveu (F4),
nenhuma foi contada como detectada por erro de compilação. As 18 funções de teste
novas tiveram ao menos uma mutação dirigida detectada. São mutações adversas por
propriedade; não se afirma que controles positivos devam falhar na engine antiga.

Checkpoint: absent/valid/corrupt/unsupported foram cobertos pelos testes; o probe
com chmod 000 confirmou unavailable, apply/resume blocked, zero passos e arquivo
preservado. Escrita por temporário exclusivo/rename e erro de clear foram
inspecionados e exercitados; os testes B12 de publicação de resultados e receipts
parciais passaram. Progresso não refletido conflita no caso canônico, mas F1
contorna a leitura estrita. O plano continua sendo derivado do repositório.

Lock: publicação por link com owner completo, fallback O_EXCL conservador,
classificação Linux por máquina/boot/namespace/PID/início, fallback não Linux
owner-unknown, ausência de heurística de idade e digests específicos da aplicação
foram inspecionados. Testes cobrem owner vivo, morto, PID reutilizado, host,
máquina e namespace diferentes, machine-id ausente, trava legada vazia, release
que preserva bytes alheios e retomada do start record. Vinte rodadas adicionais
concorreram duas recuperações com um apply; exatamente uma recuperação venceu,
o retry terminou e quatro applies concorrentes conservaram o registro started.
Não houve falha observada desse protocolo entre writers cooperantes.

CLI/MCP: renderers show/list, close plan/apply/resume, start apply/status/cancel/
recover-lock, erros estruturados action_journal_invalid e lock no start status
foram inspecionados e executados pelas suítes. F3 impede aprovar a exclusividade
do parser. Busca nos consumidores locais não encontrou dependência de produção
na trava vazia ou em leitura parcial agora recusada. Não foi inspecionado outro
clone nem o código externo de Harne8, respeitando o escopo de trabalho autorizado.

`assess tech-debt` encontrou zero markers e zero dívida descoberta. `assess
integrate` reportou o aviso existente de consumidor não observado de HTTP /mcp;
não foi estabelecido gap B13 adicional. `recurrence-check` não sinalizou recorrência.
Artifact-check atribuiu os dois commits B13 pelos trailers, apesar da preparação
inicial usar índice antigo. Nenhum finding de artifact-check foi atribuído a esta
spec. O surface-check estrito passou para a capability composta, sem findings.
Saídas integrais ficaram em `/tmp/b13-independent/`; somente fatos e hashes
SHA-256 constam em `.pose/results/b13-independent-review.json`.

## Julgamento D1, D2, ADR e compatibilidade

D1 é apropriada: checkpoint editável não comprova efeito e o repositório governa
a retomada. D2 é apropriada: identidade verificável e exclusão de recuperações
substituem idade/TTL. A distinção entre sentinel persistente e flock liberado
pelo kernel está preservada. A implementação ainda não satisfaz integralmente
essa direção devido a F1–F3.

A decisão de rename sem fsync para checkpoint não autoritativo é compatível com
R4 dentro do limite declarado de crash de processo; pode perder progresso e
replanejar, não promete resistência a power loss. Erros observáveis são expostos
como unconfirmed, e o efeito autoritativo deve ser observado no repositório.
Não se exige aqui um publisher geral ou recibo autoritativo novo.

Os limites de Linux/mesma máquina, writers cooperantes, ausência de qualificação
de power loss e exclusão de B10/B14/B15/B35 estão declarados honestamente. Os
campos opcionais e recusas mais estritas justificam classificação minor; nenhum
exit code de sucesso legado foi deliberadamente renumerado. F3 é um desvio da
recusa de uso inválido, não uma incompatibilidade autorizada.

R1 permanece incompleto por F1/F2; R2 funciona para erros reconhecidos, mas não
para F2; R3/R5 permanecem incompletos por F1; R4 foi qualificado dentro do ADR;
R6/R7/R8 passaram nos limites acima; R9 permanece incompleto por F1–F3. Essa
avaliação funcional não é substituída por lint nem por checks verdes.

## Estado nativo e próxima ação

Bundle somente preparado: `rvb-7ff8f2d1ead3f41f`; digest `sha256:7ff8f2d1ead3f41f5c5ea09cc8be3d2bcf48d4bc5e076ed6c3668f78146bc971`; plano `sha256:332464cfd3594e2eb900968742755fc2a299ec85c353454823915caffb196850`. O plano tem 15 critérios e 11 entradas de ferramenta. Ferramentas de conclusão ficaram deferred; assess-design e assess-discover recomendados não foram usados porque houve revisão direta do ADR/diff e não há entrega aprovada a recalcular. As execuções e limites constam no JSON compacto. Sem selo nem atestação. `review verify` retornou exit 1, ready-to-seal,
fresh=false, approved=false. `close` não foi executado devido aos findings.
`lint-spec --strict` passou para a spec ainda in-progress; `check --strict`
passou com 29 warnings do repositório. Não houve commit, push, PR novo ou merge.
Spec, tasks, programa e roadmaps permanecem sem declaração de delivery_complete.

Corrigir os três findings médios, fortalecer a regressão F4 e submeter o delta a
nova revisão independente. Não houve aceitação de risco residual, disposition
automática de follow-up ou staging de contribuição upstream.

## Second round

Revisão em 2026-10-16 sobre `8e8b13074e1484aef364806a9bed232fde947fca`,
incluindo a remediação `30bc0986`. A rodada 1 acima permanece histórica e
inalterada. F1–F4 foram resolvidos; a nova varredura integral não identificou
finding médio ou superior, nem regressão nas propriedades antes aprovadas.
A aprovação e o closeout dependem dos gates nativos registrados abaixo.

A revisão different-actor continua sendo a execução Codex/OpenAI autorizada
pelo maintainer, distinta do implementador Claude/Anthropic. No julgamento
`scope`, `agent:independent-` é a declaração exigida pelo motor até a spec
`pose-delegated-review-run-provenance` substituí-la. Identidade declarada;
as atestações do pose-dist não são assinadas.

### Correções e contraprovas

F1: `foldJSONKey` escolhe um representante das classes de `unicode.SimpleFold`
que o decoder Go considera equivalentes. O walker verifica duplicatas em todos
os objetos, inclusive estruturas aninhadas; o modo anterior dos demais leitores
estritos conserva sua política de controles. Aliases de schema/seq agora recusam
o journal; `completed` junto com `COMPLETED` bloqueia como checkpoint-corrupt,
sem passos. F2: o discriminador é conferido contra os eventos do contrato antes
de derivar a view. F3: presença de recover-lock é independente do valor; vazio
retorna usage/exit 2, sozinho e combinado com apply, sem mover a spec. F4: a linha
oversized agora é um evento válido, recusado exclusivamente por exceder 1 MiB.

Foram repetidas 26 mutações da rodada 1 e quatro reversões isoladas das correções
(folding no journal, folding no checkpoint, tipo desconhecido e valor vazio de
recovery): **31/31 falharam por asserção**, cobrindo as 18 funções de teste novas.
Uma quinta reversão isolou somente o folding da chave aninhada question e fez a regressão nested field in other case falhar. Nenhuma falha de compilação foi contada como detecção. Go overlays mudaram só a
produção usada pelo compilador; código e testes tracked permaneceram intactos.
O limite antes não detectado agora é detectado pelo próprio teste corrigido.

### Varredura repetida

P01–P17 e controles P06/P11/P12 coincidem com o after publicado, comparando
outcomes, causas e estados sem valores voláteis das fixtures. P11/P12 ainda são
controles parciais; o quickstart qualifica a jornada completa. Os nove journals
reais continuam legíveis. Os probes adicionais confirmam 1 MiB aceito e um byte
além recusado, unavailable por chmod 000 bloqueando apply/resume antes de passos,
checkpoint adulterado preservado e 20 rodadas de duas recuperações concorrentes
com aquisição/retry e quatro applies idempotentes sem perda de start record.

Foram relidos os produtores e consumidores do diff completo: parser e append de
journal, diagnóstico tipado e hash dos bytes, checkpoints e pós-imagens, ambos os
renderers CLI, MCP, publicação/release/recovery da trava e o decoder compartilhado.
Não há leitor de journal de produção alternativo ao parser comum. O flock
serializa recuperações; EEXIST impede aquisição antes da remoção, e a segunda
recuperação relê digest e liveness antes de remover. Nenhum identificador cru de
máquina/boot é publicado. A classificação conserva owner-unknown em contexto que
não permite prova local. O evento reservado refused não introduz transição de
satisfação; auditoria de recusas pertence explicitamente a B10/R8, sem ampliar B13.

D1, D2 e o ADR continuam justificados: estado do repositório comprova efeitos;
idade não prova abandono. Rename sem sync do checkpoint não autoritativo atende
R4 dentro do limite de crash de processo declarado; não há promessa de power
loss. Persistência/clear falhos aparecem em unconfirmed. Os limites Linux/mesma
máquina, writers cooperantes e exclusões B10/B14/B15/B35 são honestos. As recusas
mais estritas e campos opcionais justificam minor, com exit codes preservados.
Consumidores locais, scripts, skills, workflows, docs e CI foram pesquisados;
fontes externas do Harne8 não foram lidas devido ao escopo de clone único.

Passaram vet, suíte Go completa sem cache de resultados, race nos três pacotes
e builds Windows/Darwin. Tech-debt: zero markers e uncovered. Integração expõe
avisos de consumidores HTTP não observados já alheios ao delta; nenhum gap B13
foi estabelecido. Recurrence: zero flagged keys. As regras da primeira rodada
continuam aplicáveis. Saídas completas e overlays ficam em `/tmp/b13-round2/`;
o JSON no repositório preserva a rodada 1 e acrescenta fatos/hashes compactos.

Probes adicionais recusaram aliases Unicode (`ſeq`) e JSON escapados (`\u0053EQ`),
assim como 12 ordens de modos conflitantes de start, com exit 2 e sem efeitos.
Uma fixture com o diretório de checkpoint sem permissão de escrita confirmou
`results-written` e `index-regenerated` como efeitos e registrou separadamente
as falhas de progresso em `unconfirmed`; não inventou gravação de checkpoint.

### Aprovação e closeout nativo

Decisão da segunda rodada: **approved**, sem findings residuais. A rodada 1
acima continua sendo o parecer rejeitado daquele HEAD; seu prefixo original foi
conferido por SHA-256 e permaneceu inalterado.

Validação estrita autenticada: global **135/135** e módulo **132/132** em
`8e8b13074e1484aef364806a9bed232fde947fca`, sem fail/error/skip. As primeiras
execuções sem token tiveram somente o HTTP 403 na listagem de releases do GitHub
(134/135 e 131/132); os retries completos com GH_TOKEN restrito a cada comando
passaram. Uma tentativa inicial de saída JSON absoluta foi recusada por estar
fora do projeto e foi substituída pela execução do módulo sem json-out. Nenhuma
recusa foi convertida em aprovação por mudança de policy ou de produtor.

O primeiro selo foi recusado enquanto index ainda executava e não havia
atribuição publicada. Após a conclusão de index, o selo foi produzido normalmente:
`rvb-bbed7a49533bf1e3`, digest
`sha256:bbed7a49533bf1e34f8df34d4a9a5253f981c83599dc2fe837dde3b8536d7838`,
plano `sha256:332464cfd3594e2eb900968742755fc2a299ec85c353454823915caffb196850`.
Foram lidos os 15 critérios e as 11 entradas de ferramentas desse plano;
auto-attest preview mostrou oito julgamentos pendentes, respondidos manualmente.

Artifact-check strict: **34 claims / 34 caminhos observados**, quatro commits
atribuídos por POSE-Spec e nenhum finding desta spec. Os 615 avisos globais de
proveniência não foram apresentados como defeitos B13. Surface-check strict:
capability composta `journal-checkpoint-and-lock-recovery`, entrada de produção
`pose-mcp/internal/pose/closeout_plan.go`, evidência integration atual e zero
findings. Sua saída integral fica fora do repositório. Assess-design observou
29/29 arquivos; o delta material `SD-68d92dc5` do ADR foi mapeado a D1 e sua
relação com D2 foi julgada. Discover, integrate, tech-debt e suggest foram
executados, e validate foi disposto separadamente no componente vazio e em
pose-mcp. Review-check e closeout-check ficaram deferred na atestação até haver
aprovação; foram executados em seguida. O closeout-check pré-transição apontou
somente lifecycle-not-done e dirigiu a transição guardada.

Atestação aplicada após preview: `rva-dfa2006f7d36184d`. O gate de admissão
`review verify` confirmou ready-to-close, **fresh=true, approved=true**. O
Requirement trace foi preenchido antes da transição porque o gate nativo o
exige: cada R aponta para seus testes/probes efetivos, sem substituir prova por
lint. `close spec:pose-journal-checkpoint-and-lock-recovery` retornou exit 0;
status done e completed_at foram escritos pelo motor. Lint strict pós-close:
exit 0. Tasks, Final Report, estado delivery_complete de B13 e os dois roadmaps
foram reconciliados, preservando Requirements, Technical Plan e Decisions.

Os gates finais e seus hashes constam na evidência compacta. As verificações de
admissão do assunto de implementação e de retenção terminal após a reconciliação
são registradas separadamente. Discover --if-stale --update-state foi executado;
sua saída foi restaurada ao snapshot anterior e não entra no commit. Não houve
alteração de produção/testes, policy, capabilities ou adoção, novo PR ou merge.
O programa e os roadmaps conservam o lifecycle anterior; só B13 foi encerrado.

Após a reconciliação e index final, `review verify` retornou **closed,
fresh=true, approved=true**, com o mesmo bundle e a mesma atestação;
`review-check`, `closeout-check` e `lint-spec --strict` finais retornaram exit 0.
A verificação terminal conserva a prova selada da implementação no HEAD 8e8b1307;
não alega uma nova execução da suíte no commit que armazena o closeout.

`check --strict` final: **exit 0, 29 warnings do repositório**, sem blocker B13.
Commit/push são a publicação deste closeout no PR existente, sem merge; o hash
efetivamente publicado é informado na entrega da revisão.
