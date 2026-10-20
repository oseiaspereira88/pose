# B07 — implementação para revisão

Data: 2026-10-20. Spec: `pose-git-and-dispatch-snapshot-integrity`.
Branch: `impl/pose-b07-git-snapshot-integrity`, reconciliada com PR #195.
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
adulteração do checkout produz failed; mudança da origem ou supersession produz stale, com transcript preservado.

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

## Reconciliation and current gates

The canonical PR195 history through 23643a2 is preserved by merge a4c58e9, including four prior independent review reports and their probes. The original native start is retained. Shared checkout documentation was preserved; reconciliation runs in /tmp/pose-b07-closeout-work on impl/pose-b07-git-snapshot-integrity.

The expired unused Actions waiver was retired separately in d957b4f under spec:pose-retire-unused-first-party-action-exception. Workflow pins and enforcement remain unchanged. The reconciled full suite and 11/11 production mutations passed; native strict validation subsequently passed 141/141 checks. Those results predate the three corrections requested in reconciled-review-r1 and will be regenerated for the final subject.

The first separate review requested changes for persisted dispatch Git loss, closeout output type/deletion exemptions and run ordering. These are corrected with actual-dispatch metadata-loss coverage, nine exact-name replacement scenarios and a nonvacuous history-order control. The final separate same-vendor review accepted the functional corrections on inspection with no remaining functional finding; the cross-vendor attempt hit its weekly limit. No human confirmation or cryptographic reviewer identity is claimed.

Default artifact attribution now reconciles 82 claims without errors after preserving the complete history. Final native artifact/surface/validation checks, sealed review and closeout remain required. Historical reports and mutation results are retained as historical evidence, not current delivery proof.

## Limits

Bounded observations do not provide adapter sandbox containment or detect every transient restored edit. Dispatch writes historical records and transcripts, not approval or attestation. Consumers revalidate persisted identity, transcript, supersession and semantic relevance. Outputs go to stdout or the external output placeholder; directory names grant no adapter write authority.

The R2 remaining persisted-materialization finding is corrected by reusing verifyBundleMaterialization at consumption. Intact-control regressions cover same-byte consumed-profile symlinks and hidden relevant-submodule edits. Review R3 records the final declared judgment and its mechanical evidence limits.
