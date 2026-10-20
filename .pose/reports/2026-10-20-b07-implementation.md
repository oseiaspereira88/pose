# B07 — implementação para revisão

Data: 2026-10-20. Spec: `pose-git-and-dispatch-snapshot-integrity`.
Branch: `impl/pose-b07-snapshot-integrity`, sobre `8aa729a7`.
Estado: in-progress; sem aprovação, closeout, publicação ou adoção no Harne8.

## Comportamento

`PlanCloseout` exige HEAD e status conhecidos, interpreta porcelain NUL com os
dois paths de rename e compara o tree commitado com a materialização real.
Edições escondidas por flags do índice, troca de tipo, links nos pais e mudanças
em submódulos não certificam clean. Mudança de HEAD durante a leitura é conflito
observável, com retry explícito. Outputs do closeout têm nomes/produtores
delimitados: índices conhecidos, cópias da geração atual conferidas contra o
descritor, resultado configurado na delivery policy, checkpoint da spec e novos
artefatos de review com identidade própria. Não há permissão genérica para as
árvores de indexes/results/state; evidência de review já commitada é imutável.

Dispatch vincula preview ao worktree selecionado e verifica bundle semântico,
brief, adapter e inputs materializados antes de executar. O worktree descartável
precisa materializar os inputs selados, não somente o mesmo HEAD. Após o processo,
compara disco inteiro, inclusive arquivos ignorados; reconfere HEAD, materialização,
adapter e freshness da origem. Um commit documental independente é permitido;
outra origem, adulteração ou supersession produz failed com transcript preservado.

Adapters não carregam autorização autenticada de produtor/output. Portanto
stdout e `{output}` são seus destinos; não podem regenerar estado dentro do
worktree por alegar um prefixo derivado. O plano de execução é leitura opt-in
no MCP: `pose_closeout_state(include_plan=true)` usa `Store.PlanCloseout` e
propaga erro sem devolver um estado de sucesso. O formato legado sem a opção
permanece disponível; ADR e golden MCP foram atualizados.

## Evidência

- `go test ./internal/pose ./internal/cli ./internal/mcpserver -run 'TestGitSnapshot|TestReviewDispatch|TestCatalogMatchesGolden|TestClosePlan' -count=1`: passou.
- `go test -race ./internal/pose ./internal/cli ./internal/mcpserver`: passou.
- Regressões finais com `-race -run 'TestGitSnapshot|TestReviewDispatch|TestClosePlan' -count=1`: passaram nos três pacotes.
- `go test ./internal/pose -run TestGitSnapshot -count=1`: passou após acrescentar os controles de submódulo e de geração autorizada/adulterada.
- `go test ./internal/cli -run TestQuickstartScriptRunsEveryChapter -count=1`: passou após a correção da comparação SHA-256 prefixada.
- `go vet ./...`: passou; também é check da matriz nativa.
- `pose check --strict`: passou com 29 avisos já existentes, antes da reconciliação final de artefatos.
- `pose assess discover --if-stale --component pose-mcp` e `pose assess integrate`: executados; saídas persistidas. O segundo conserva os gaps observados, sem afirmar integração externa nova.
- `pose knowledge-check --strict`: passou; nenhum handoff expirado.
- [Resultado da matriz nativa](../results/b07-validation.json) e [relatório nativo](2026-10-20-standard-b07-snapshot-integrity.md): fonte de verdade para checks, comandos, outputs e outcome global.

A rodada integral executou 137 checks: 136 passaram e um falhou, exclusivamente
pela exceção vencida descrita abaixo. A conferência final de paths de submódulo,
payload do bundle e limites de input é coberta pela repetição das regressões B07;
a revisão deve reexecutar a matriz no sujeito que concluir.

As rodadas iniciais registraram falhas do sandbox (cache somente leitura e
bind HTTP recusado); foram repetidas com o acesso necessário. A primeira
matriz após a extensão MCP também detectou golden desatualizado, corrigido
somente com a propriedade e descrição previstas no ADR. Esses resultados não
são promovidos a passes retrospectivos; a history do relatório os conserva.

## Gate pendente

O commit de implementação é `703b8318063f78fcf1d461410bc06d0ed456e7ab`.
A reconciliação explícita `pose artifact-check --spec
pose-git-and-dispatch-snapshot-integrity --from 8aa729a7 --to 703b831 --strict`
passou: 31 claims e 31 paths observados, com avisos globais de orphans. O comando
também atualizou nativamente delivery-integrity e seu descritor de geração.

O seletor padrão por trailers continua abrangendo a história anterior da spec:
base `1e0a313`, três commits atribuídos e 46 paths observados. Ele reporta como
undeclared `review_bundle.go`, `obligation_projection.go` e `git_spawn_cache.go`,
que esta implementação não editou. Não foram acrescentadas declarações falsas
para fazer esse intervalo passar. Reconciliar essa fronteira histórica ao preparar
o sujeito canônico de review, preservando a distinção de atribuição registrada
no AGENTS.md. Não tratar o passe do intervalo explícito como passe do padrão.

`surface-check --spec pose-git-and-dispatch-snapshot-integrity --strict` ainda
recusa o fechamento: a evidência antecede o commit e tem outro scoped provenance
digest, além da fronteira de atribuição acima. Reexecutar os produtores registrados
sobre o sujeito final após reconciliar a autoridade do change set. Os resultados
anteriores continuam sendo execução real de testes, sem promover freshness.

O teste `TestWorkflowSecurityContract` recusa a exceção
`first-party-actions-tag-pinning` em `.github/security-exceptions.json`, expirada
em 2026-10-19. O arquivo e o teste não foram alterados por B07. Não houve
renovação de data ou dispensa de segurança: revisar a exceção ou corrigir o
pinning em escopo próprio, então reexecutar a matriz integral.

Revisão e admissão nativas ainda são devidas. Não marcar a B-unit como
delivery_complete, não fechar a spec e não atualizar o gitlink consumidor com
base apenas nestes testes. Conferir attribution e surface-check no sujeito
commitado e na evidência fresca antes de concluir a review.

## Limites

As leituras são limitadas a 100000 entradas e 256 MiB; orçamento excedido é
unavailable, nunca cobertura parcial certificada. Há custo de verificação de
conteúdo além de `git status`; B32 conserva otimizações e benchmarks.

Antes/depois não detecta toda escrita transitória, não cria sandbox e não
coordena escritores Git externos. A única publicação do dispatch é história;
ele não aplica attestation, approval ou fulfillment. Futuras operações governadas
devem verificar os inputs/autorização atuais. B16/B18/B19 continuam responsáveis
por ledger, containment e consumo governado, respectivamente.

Retomada: [handoff B07](../knowledge/2026-10-20-handoff-pose-b07-implementation.md).
