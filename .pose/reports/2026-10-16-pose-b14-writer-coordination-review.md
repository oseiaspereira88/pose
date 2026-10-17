# Parecer — revisão independente do B14 (`pose-writer-coordination-and-atomic-publication`)

- PR #188, branch `impl/pose-b14-writer-coordination`.
- Revisor: `agent:independent-claude-opus-5-5-review`, autorizado pelo maintainer.
- Implementador: `agent:claude-opus-5-5` (sessão `296d1fcb-e989-40ae-9362-cf3514db8b26`).
- **Rodada 1** (head `07e9eb5a`): alterações solicitadas. O texto da rodada 1
  segue abaixo sem mudanças.
- **Rodada 2** (head `00f743c3`): **aprovado**, com quatro findings baixos
  aceitos como risco. Bundle `rvb-b3ff5896947dc94f`, atestação
  `rva-7a84aea174407a89`, closeout nativo concluído. Veja "Second round".
- Em nenhuma rodada o revisor alterou código de produção, testes, policy,
  capabilities ou adoção.

## First round

## Condição de independência

Esta revisão é de ator diferente **apenas por execução**. Ela roda no mesmo
vendor e no mesmo modelo do implementador (Claude Opus 5.5, da Anthropic), numa
sessão separada e sem o contexto da sessão que implementou. A independência não
é de vendor nem cognitiva. O revisor de outro fornecedor (Codex) foi substituído
por decisão do maintainer, porque estava sem cota de uso. O prefixo
`agent:independent-` é uma declaração que o motor exige até a spec
`pose-delegated-review-run-provenance` substituí-lo. A identidade é declarada:
no pose-dist as atestações não são assinadas.

## O que foi verificado

As saídas brutas ficam fora do repositório. O sha256 de cada uma está em
`.pose/results/b14-independent-review.json`.

- `go vet ./...`: exit 0. `go test ./... -count=1`: exit 0 (10 pacotes).
  `go test -race` em `internal/pose` e `internal/cli`: exit 0.
  `GOOS=windows` e `GOOS=darwin go build ./...`: exit 0.
- Probes P01–P04 rodados com a CLI desta branch: resultado idêntico ao `after`
  registrado em `b14-front09-probes.json`. Só o sha256 do binário difere
  (build local).
- Mutações numa cópia `git archive` fora do repositório, 17 ao todo. Reprovam
  15: staging fixo; sem checagem de digest; imutável publicado por rename; retry
  imutável em conflito; transfer sem as travas de spec, ou só com a de origem;
  close sem trava; close não vinculado ao digest julgado; state com
  `ExpectAny`; histórico com falha simples; cauda torn colada; action policy
  com `ExpectAny`; baseline gravado no lugar; limpeza de todo `.tmp`; e
  `PublishFault` ignorado. Este último derruba três testes de start do B12/B13,
  o que confirma que eles continuam provando o que provavam antes de trocar o
  diretório plantado pelo hook. **Sobrevivem** a remoção de `tmp.Sync()` e a de
  `syncDir`, inclusive contra as suítes completas de pose, cli e mcpserver. As
  falhas nesse caso são idênticas às da cópia sem mutação: 16 testes de
  documentação que leem arquivos ausentes na cópia.
- Experimentos de comportamento em outra cópia (testes descartáveis, fora do
  repositório):
  - **E1**: o refresh disparado por hook publica o state e falha o histórico.
  - **E4**: a aprovação fica stale dentro da região protegida do close.
  - **E6**: start do destino de uma transferência parcial. O destino fica
    `blocked`, o start recusa com `spec-not-ready` e não há dupla autoridade.

## Findings

As severidades usam a escala do pedido. Há três findings médios, e por isso a
revisão não foi selada.

1. **F1 — média — `pose-mcp/internal/cli/state_hooks.go:124-135` (R6, R7).**
   `stateRefreshConsumer` não distingue o novo `stateRefreshPartialError`.
   Fora do modo estrito, um refresh disparado por `lint-spec` no closeout, por
   `amend`, por `assess snapshot` ou por evidência publica o state, falha o
   append do histórico, chama `markRefreshPending` sobre o state recém-publicado
   e devolve `nil`. Isso foi reproduzido no E1: `consumer-err=<nil>`, state
   alterado, `refresh_pending` marcado e log `partial`. O erro secundário vira
   sucesso genérico, e o state passa a afirmar uma pendência que não existe.
   A correção do B14 só cobre `state init|refresh`.
2. **F2 — média — `pose-mcp/internal/cli/assess.go:664-758` e
   `pose-mcp/internal/cli/state_hooks.go:156-176` (R2, R7).** Há outros writers
   de `project-state.md` fora do publisher. `updateProjectStateArchitecture`
   (via `assess discover --update-state`) e `markRefreshPending` leem o state,
   alteram e gravam no lugar com `os.WriteFile`, sem precondição. Uma edição
   curada feita entre a leitura e a escrita se perde, e um leitor pode ver o
   arquivo truncado. Isso contradiz a consequência que o ADR declara ("mutable
   records are published over what was read or refused") justamente no artefato
   que a spec diz proteger, e não aparece entre os limites.
3. **F3 — média — `pose-mcp/internal/cli/review_closeout.go:1665-1689` e
   `1761-1779` (R4, V4).** O close vincula a publicação ao digest da spec, mas a
   autorização (review aprovado) é julgada antes dos gates e não é revalidada
   depois de tomar a trava. No E4, uma mudança no artefato entregue feita
   enquanto os gates rodavam deixou o review não aprovado dentro da região
   protegida ("current semantic inputs are not represented by a sealed
   bundle"), e mesmo assim o close saiu com 0 e publicou `status: done`. R4 pede
   que a autorização seja revalidada dentro da região protegida, e V4 que
   autorização stale não governe a publicação. O limite que o ADR declara
   cobre só a janela entre a verificação e o rename dos bytes da spec.
4. **F4 — baixa — writers mutáveis ou autoritativos fora do inventário (R2, R3,
   R5).**
   - `artifact-backfill --apply --confirm-spec-edits`
     (`artifact_integrity.go:606-616`) reescreve specs via `writeAtomic`
     (`derived`, `ExpectAny`, sem sync e sem trava de ciclo de vida). Ele pode
     reverter uma transição de close publicada entre a sua leitura e a sua
     escrita, e publica uma spec como `derived`, contrariando "authoritative
     (specs…)".
   - A atualização do start record (`start.go:554`) e as policies de adoção
     (`capability_catalog.go:119`) usam `ExpectAny` em registros mutáveis e
     autoritativos, contra a regra do próprio publisher ("ExpectAny… only for
     derived artifacts").
   - O seed da validation matrix (`stack_seed.go`, `wizard.go`), o merge de
     manuais (`managed_docs.go:315`), `project.json` (`project_file.go:72`),
     o capability assessment (`capability_triggers.go:112,419`,
     `assess.go:276`) e o lock de extensões ficam fora.

   Meu julgamento: o writer de spec pertence a este escopo, porque a spec é o
   recurso do ciclo de vida. Os demais podem ir para outro lote, mas precisam
   ser declarados como limite ou follow-up. Hoje não são.
5. **F5 — baixa — `pose-mcp/internal/pose/writer_publication.go:150-156` (R6).**
   Uma falha de `syncDir` depois do rename volta como erro simples. O close
   imprime `pose close: <erro>` com exit 1 sobre uma spec já `done`, e o start e
   as policies também relatam falha sem efeito. Falta uma falha tipada de
   "publicado, durabilidade não confirmada".
6. **F6 — baixa — `writer_publication.go:104-106,154-156` (R5, R8).** O sync de
   arquivo e o de diretório da classe `authoritative` não são qualificados por
   nenhum teste: as mutações M05 e M06 sobrevivem. R5 pede efeitos observáveis.
   Bastaria um hook de teste no sync, como `PublishFault`.
7. **F7 — baixa — `state_refresh.go:255` (R5, R7).** O state é `derived` (sem
   sync), mas contém seções curadas que o motor não recomputa, o que contradiz
   a definição do próprio ADR ("a projection the engine can compute again").
   Uma queda de energia pode perder prosa curada ainda não commitada. A perda de
   energia está declarada como não qualificada e o state é rastreado no Git,
   daí a severidade baixa. Mesmo assim, a classe precisa de justificativa
   explícita ou deve ser `authoritative`.
8. **F8 — baixa — `start.go:546` (R8).** O baseline passou de `O_EXCL` para
   `ExpectAbsent` por hard link, sem fallback, ao contrário da trava do B13
   (`publishStartLock`). Em filesystems sem hard links, o start, que antes
   funcionava, passa a falhar sempre. O ADR declara o limite, mas a
   classificação de compatibilidade só lista recusas novas.
9. **F9 — baixa — `start_lock.go:172` e `cli/start.go:398-400` (R3).** Uma trava
   mantida por close ou por transfer aparece como `start-in-progress`, e a
   próxima ação diz "confirm no start is running". O próprio teste
   `TestACloseWaitsForNoOtherLifecycleWriter` exige `start-in-progress` para um
   dono `transfer.apply`. A fase e a autoridade divulgadas não são fiéis.
10. **F10 — baixa — `writer_publication.go:89` (R1).** Um staging órfão deixado
    por crash (`.<nome>.*.tmp`, com nome aleatório) não está no `.gitignore` e
    nunca é varrido. Sob `.pose/specs` ou `.pose/starts`, ele bloqueia o passo
    de contexto do closeout ("uncommitted changes outside generated paths") até
    ser removido à mão.
11. **F11 — baixa — spec, linha 126 (R8).** A task marcada afirma "12 mutações
    reprovando", mas nenhum registro de mutação existe no repositório. A minha
    execução independente está registrada na evidência desta revisão.

## Pontos verificados sem finding

- O staging é exclusivo, e a limpeza remove só o próprio arquivo.
  `ExpectDigest` não produz lost update entre writers que passam pelo publisher.
  `ExpectAbsent` não sobrescreve, e o retry com os mesmos bytes converge.
- A trava de ciclo de vida é única por arquivo, então aliases de root convergem
  para o mesmo lock. O transfer a toma depois das travas de root, em ordem
  canônica. Nenhuma trava espera, então não há ciclo de espera nem deadlock.
- O close chamado pelo plano (`close --apply|--resume`) toma a trava pela
  primeira vez em `cmdCloseLocal`, porque o plano não a detém, e por isso não
  há auto-recusa. Uma trava deixada por um start interrompido recusa o close com
  o estado do dono, e a recuperação vem de `start --recover-lock` ou da remoção
  manual se o dono for desconhecido.
- Roadmaps não tomam trava de spec, mas publicam só sobre o digest julgado.
- O state refresh faz CAS sobre o state lido, devolve resultado parcial pela
  CLI, o retry não duplica entrada e o append não se cola numa cauda torn.
- Windows sem sync de diretório: limite aceitável, já que a perda de energia
  não é qualificada. Mas "rename journaled" não equivale a durabilidade, e o
  texto do ADR deveria dizer isso. No darwin, `File.Sync` usa `F_FULLFSYNC`
  com fallback; ali só a compilação cruzada foi verificada.
- Não há rota MCP que escreva ciclo de vida, então a declaração procede. Nenhum
  consumidor operacional (scripts, skills, workflows, CI) depende de
  temporários fixos. As únicas referências são relatos históricos de reprodução
  do B12.

## D1 e o ADR

Os limites declarados são honestos: sem atomicidade entre arquivos ou roots,
janela entre verificação e rename para writers não cooperantes, perda de energia
não qualificada, hard links exigidos para imutáveis e MCP sem rota de escrita.
Nenhum deles contradiz um requisito por si.

As consequências afirmadas, porém, vão além do entregue. "Mutable records are
published over what was read or refused" não vale para o state (F2) nem para as
specs reescritas por `artifact-backfill` (F4). E o vínculo do close cobre os
bytes da spec, mas não a autorização (F3).

## Próximo passo

Corrigir F1–F3, e declarar ou corrigir F4–F11. Depois, nova revisão por ator
diferente antes do closeout.

## Second round

Revisão do head `00f743c3`, que traz a resposta `7999f50d` e a validação
estrita publicada em `00f743c3`. Mesma condição de independência da rodada 1:
ator diferente **apenas por execução**, mesmo vendor e mesmo modelo do
implementador (Claude Opus 5.5, da Anthropic), numa sessão separada e sem o
contexto da sessão implementadora. Não é revisão de outro vendor. O revisor da
Codex foi substituído por decisão do maintainer, por falta de cota de uso. O
prefixo `agent:independent-` é a declaração que o motor exige até a spec
`pose-delegated-review-run-provenance` substituí-lo. A identidade é declarada e
a atestação não é assinada.

### O que foi verificado

As saídas brutas ficam fora do repositório. O sha256 de cada uma está em
`.pose/results/b14-independent-review.json`. A CLI foi compilada desta branch
com `go -C pose-mcp build -o ../pose-review ./cmd/pose`, o que a coloca na raiz
do clone (`./pose-review`). Todos os comandos usaram esse binário.

- `go vet ./...`: exit 0. `go test ./... -count=1`: exit 0 (11 pacotes com
  testes). `go test -race` em `internal/pose` e `internal/cli`: exit 0.
  `GOOS=windows` e `GOOS=darwin go build ./...`: exit 0.
- Probes P01–P04: idênticos ao `after` de `b14-front09-probes.json`. Só o
  sha256 do binário difere.
- Validação estrita global: 136/136 em `00f743c3`. A primeira tentativa
  reprovou só na jornada de install, com 403 anônimo da API do GitHub. A
  repetição com `GH_TOKEN` exportado apenas naquele comando passou, e o token
  não aparece em nenhuma saída. Módulo `pose-mcp` estrito: 133/133.
- `artifact-check --strict`: 37 claims e 37 observados em 4 commits. Os 617
  avisos são órfãos do repositório inteiro, anteriores a esta spec.
  `surface-check --strict`: 1 alvo, 136 resultados, 0 findings.
  `assess design`: 30/30 arquivos, dois deltas (ADR e validation matrix),
  ambos mapeados a D1. `assess tech-debt`: 0 marcadores.
- O driver versionado `b14-mutations.py` rodou numa cópia `git archive` fora do
  repositório, porque ele edita os arquivos no lugar: 25/25 mortas, o mesmo que
  `b14-mutations.json`. Rodei mais 10 mutações próprias e comparei com a linha
  de base da cópia. Nessa cópia, um único teste reprova mesmo sem mutação
  (`TestEveryGateOnTheMachineChannelPrintsOneDocument`, ambiental: a cópia não
  tem `.git`). Morrem X1 (parcial deduplicado), X3 (backfill sem trava), X5
  (retry exclusivo em conflito) e X10 (aviso `history-not-appended` trocado).
  Sobrevivem X2 (state de volta a `derived`), X4 (closeout volta a contar
  staging órfão), X6 (revalidação ignora action requests), X7 (fallback do
  marcador sem comparar bytes), X8 (próxima ação volta a citar só start) e X9
  (criação exclusiva sem sync).
- Experimentos com testes descartáveis, numa terceira cópia:
  - **E7**: o sync de diretório falha depois do rename do state. O state fica
    publicado, o histórico não recebe entrada, o refresh log registra `failed`,
    o hook marca `refresh_pending` num state atual e devolve "could not mark
    refresh_pending", embora tenha marcado. Pela CLI, `state refresh` sai com
    exit 1 e uma mensagem que diz que o arquivo foi publicado.
  - **E8**: o mesmo no close. A spec fica `done`, mas o receipt do
    `close --apply` vem como `failed` com causa `transition-refused`. O
    `close --resume` converge (`already-satisfied`), mas ainda carrega o erro
    anterior.

### Resposta a cada finding da rodada 1

- **F1 — resolvido.** Com resultado parcial, o hook não marca
  `refresh_pending`, o resultado `partial` deixa de ser deduplicado e
  `pose state` mostra `history-not-appended`. As mutações "hook parcial marca
  pendente", "validate não avisa parcial", X1 e X10 morrem. O hook em modo
  best-effort ainda devolve sucesso ao chamador, como já fazia com qualquer
  falha. Agora, porém, o efeito parcial fica registrado e visível.
- **F2 — resolvido.** `markRefreshPending` e `updateProjectStateArchitecture`
  publicam com `ExpectDigest` sobre o state lido, e as mutações dos dois
  morrem. O fallback no lugar só roda quando o diretório recusa o staging e
  confere os bytes antes, mas não tem teste (X7, em r2-f3).
- **F3 — resolvido.** `cliCloseoutState` é julgado de novo depois da trava, e o
  hook `closeGatesJudged` reproduz o E4 da rodada 1. A mutação "close sem
  revalidar autorização" morre. A parte de action requests não tem teste (X6).
- **F4 — resolvido no escopo e declarado fora dele.** O backfill toma a trava
  e faz CAS (morrem a mutação versionada e X3). As policies de adoção usam CAS.
  O update do start record mantém `ExpectAny` sob a trava, como justifica o
  ADR, e conferi que todo writer do start record roda sob ela. A lista de
  limites, porém, tem imprecisões (r2-f2).
- **F5 — parcial.** `*PublishDurabilityError` existe e é testado, e a mensagem
  diz que o arquivo foi publicado. Mas só `markRefreshPending` distingue o
  tipo: o state refresh e o close ainda classificam o efeito como falha (E7 e
  E8, em r2-f1).
- **F6 — resolvido.** `PublishSyncHook` observa os dois syncs, e as mutações
  de sync de arquivo e de diretório agora morrem.
- **F7 — resolvido no código.** O state é `authoritative`. Nenhum teste fixa a
  classe (X2), e um parágrafo do ADR ainda diz `derived` (r2-f2).
- **F8 — resolvido.** Sem hard link, o imutável é criado com `O_EXCL`, o retry
  converge e outros bytes dão conflito. Mutações "imutável sem fallback" e X5
  morrem. O alcance do fallback está em r2-f4.
- **F9 — resolvido.** Um dono que não é start recebe
  `spec-lifecycle-in-progress`, e a próxima ação cita a operação. O código é
  testado; o texto da próxima ação não (X8).
- **F10 — resolvido.** O padrão `.pose/**/.*.[0-9]*.tmp` está no `.gitignore`,
  e `ungeneratedChanges` ignora o staging órfão. O predicado é testado; a
  integração no closeout não (X4). Fora de `.pose/`, um staging órfão ainda
  aparece no `git status`, mas não bloqueia o closeout.
- **F11 — resolvido.** Driver e resultado versionados e reproduzidos (25/25).

### Nada do que a rodada 1 aprovou regrediu

Staging exclusivo, limpeza só do próprio arquivo, `ExpectDigest` e
`ExpectAbsent`, a trava única por spec tomada depois das travas de root e sem
espera, o close pelo plano (`close --apply|--resume`) sem auto-recusa, a trava
deixada por um start interrompido, roadmaps só sobre o digest julgado, o CAS do
state, o retry sem duplicar entrada e o append sem colar em cauda torn: todos
seguem cobertos pelas mesmas regressões, que passam, e pelas mutações
versionadas, que morrem. `PublishFault` não mudou, então os testes de start do
B12/B13 continuam provando o que provavam (M17 da rodada 1). Os efeitos de
adoção (selo do ledger legado) não escrevem `review.json` nem `dor.json`, então
o CAS das policies não recusa a própria adoção.

### Findings da segunda rodada

Nenhum é de severidade média ou maior. Os quatro foram registrados na atestação
como `accepted-risk`, com dono `maintainer`, revisão até 2026-11-16 e ação de
follow-up.

1. **r2-f1 — baixa — `pose-mcp/internal/cli/state_refresh.go:256`,
   `review_closeout.go:1793-1799`, `state_hooks.go:187` (R6, R7).** Nenhum
   consumidor autoritativo usa `ErrPublishUnsynced`. Com o state agora
   `authoritative`, uma falha de sync de diretório depois do rename faz o
   refresh registrar `failed`, pular o histórico e marcar `refresh_pending`
   num state atual (E7). No close, o receipt vem como `transition-refused`
   sobre uma spec já `done` (E8). As mensagens dizem "published" e o resume
   converge, e o gatilho é um erro raro de filesystem; daí a severidade baixa.
2. **r2-f2 — baixa — ADR, "Writers moved to it" e "Consequences" (R2, R5,
   D1).** A lista de limites não descreve fielmente os writers não migrados.
   `project.json` (`project_file.go:72`) e o capability assessment
   (`capability_triggers.go:112,419`, `assess.go:276`) não passam por
   `writeAtomic`: usam `os.WriteFile` no lugar, com escrita torn possível e,
   no assessment, read-modify-write sem precondição. `stack_seed.go:262-332`
   e `354-408` reescrevem `.pose/policy/review.json`, que o ADR classifica
   como autoritativo, com `writeAtomic` (`derived`, `ExpectAny`). As edições
   de status de contribuições (`contribute.go:543`) não aparecem. E o
   parágrafo dos writers ainda chama o state de `derived`. Todos rodam em
   install, update ou assessment, por um operador, o que sustenta a
   severidade baixa e o adiamento para B34 e B35.
3. **r2-f3 — baixa — testes (R8).** Seis comportamentos da segunda rodada não
   são fixados por nenhuma regressão: X2, X4, X6, X7, X8 e X9 sobrevivem.
   Verifiquei cada um por leitura e, quando cabia, por experimento.
4. **r2-f4 — baixa — `pose-mcp/internal/pose/writer_publication.go:152-158`
   e `publishExclusive` (R1, R2).** Qualquer erro de link que não seja
   `EEXIST` leva à escrita no lugar, inclusive erros transitórios ou de
   espaço. E uma escrita que falha não remove o registro que ela mesma cortou,
   então todo retry depois recusa com `start-record-exists`. A trava do B13
   (`publishStartLock`) remove nesse caso. O ADR declara que o fallback pode
   cortar a escrita, e o cenário é raro.

### D1 e o ADR

Os limites declarados continuam honestos e não contradizem nenhum requisito.
Ficam de fora a atomicidade entre arquivos ou roots, a janela entre verificação
e rename para writers não cooperantes, a perda de energia, o Windows sem sync de
diretório (aceitável, já que a perda de energia não é qualificada) e o MCP sem
rota de escrita de ciclo de vida. As consequências que a rodada 1 apontou como
além do entregue (F2, F3, F4) agora valem no código. Resta corrigir a precisão
do texto do ADR (r2-f2).

### Closeout nativo

- `review bundle --seal`: `rvb-b3ff5896947dc94f` (15 critérios, 11
  ferramentas, banda critical, independência different-actor).
- `review attest --decision approved --apply`: `rva-7a84aea174407a89`, com
  todos os critérios julgados, `design-causality` mapeado para os dois deltas,
  as ferramentas executadas, `review-check` e `closeout-check` adiados até a
  aprovação, e os findings r2-f1 a r2-f4.
- `review verify`: `ready-to-close`, fresh, approved. `review-check`: exit 0.
  `close`: `Closed: spec:pose-writer-coordination-and-atomic-publication`
  (`completed_at: 2026-10-17`, a data UTC do motor). `closeout-check`:
  terminal. `lint-spec --strict`: 1/1. `check --strict`: SUCCESS, com avisos
  anteriores a esta spec.
- A saída de `assess discover` foi restaurada e não entra no commit.

## Decisão

**Aprovado** na rodada 2, com quatro riscos baixos aceitos e follow-ups para o
maintainer. Merge fica com o maintainer.
