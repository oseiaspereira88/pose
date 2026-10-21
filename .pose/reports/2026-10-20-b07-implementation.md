# B07 — implementação para revisão

Data: 2026-10-20. Spec: `pose-git-and-dispatch-snapshot-integrity`.
Branch: `impl/pose-b07-git-snapshot-integrity`, reconciliada com PR #195.
Fechamento local nativo concluído em 2026-10-21. Adoção no Harne8 e publicação não fazem parte deste registro.

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

## Reconciliation history

The canonical PR195 history through 23643a2 is preserved by merge a4c58e9, including four prior independent review reports and their probes. The original native start is retained. Shared checkout documentation was preserved; reconciliation runs in /tmp/pose-b07-closeout-work on impl/pose-b07-git-snapshot-integrity.

The expired unused Actions waiver was retired separately in d957b4f under spec:pose-retire-unused-first-party-action-exception. Workflow pins and enforcement remain unchanged. The reconciled full suite and 11/11 production mutations passed; native strict validation subsequently passed 141/141 checks. Those results predate the three corrections requested in reconciled-review-r1 and will be regenerated for the final subject.

The first separate review requested changes for persisted dispatch Git loss, closeout output type/deletion exemptions and run ordering. These are corrected with actual-dispatch metadata-loss coverage, nine exact-name replacement scenarios and a nonvacuous history-order control. The final separate same-vendor review accepted the functional corrections on inspection with no remaining functional finding; the cross-vendor attempt hit its weekly limit. No human confirmation or cryptographic reviewer identity is claimed.

An earlier default artifact reconciliation covered 82 claims without errors after preserving the complete history. Final native artifact/surface/validation checks, sealed review and closeout remain required. Historical reports and mutation results are retained as historical evidence, not current delivery proof.

## Limits

Bounded observations do not provide adapter sandbox containment or detect every transient restored edit. Dispatch writes historical records and transcripts, not approval or attestation. Consumers revalidate persisted identity, transcript, supersession and semantic relevance. Outputs go to stdout or the external output placeholder; directory names grant no adapter write authority.

The R2 remaining persisted-materialization finding is corrected by reusing verifyBundleMaterialization at consumption. Intact-control regressions cover same-byte consumed-profile symlinks and hidden relevant-submodule edits. Review R3 records the final declared judgment and its mechanical evidence limits.

## Final native closeout — 2026-10-21

The final canonical matrix passed 141/141 on d6ce7bc, using isolated test temporaries in /var/tmp to avoid an intermittent ancestral .git in /tmp. Full race suite passed, 14/14 production mutations were killed, govulncheck found no vulnerabilities, and a five-family changed-diff secret pattern scan found no matches. This pattern scan is not represented as a full gitleaks audit.

B07 sealed rvb-0ae879ef12b5b804 and recorded rva-476a47d625b27e1a. The separate waiver spec sealed rvb-5419e7c7ab90aebe and recorded rva-d25bd22736b23e42. Both native closes succeeded. Reviewer execution 01a15b9e-18b0-7f51-8130-99398a5825b7 is separately declared, same vendor; no cryptographic identity or human confirmation is claimed. Scope artifact and surface gates passed, with zero B07 surface findings.

Global pose check reported five unrelated undeclared documentation paths under pose-canonical-positioning; those concurrent changes were preserved. Tech-debt still discloses incomplete coverage for two invalid legacy specs. Three historical development retries are retained, covered by existing feature/security validation workflows; they are not treated as proof of an uncovered process incident.

The final validation phase was also recorded natively under its original auto-validate/strict context, deriving pass from the actual 141/141 log. recurrence-check --strict then reported zero flagged keys and one resolved cluster; the three failed records remain immutable. Required history-check and knowledge-check were run by the implementation execution (13 knowledge records, zero errors); they were not performed by the read-only reviewer. Native artifact, surface, review and closeout gates are collected by this execution report; the R3 report is the source of the separately concluded judgments.

The initial attestations remain immutable history. Final attestations rva-85befe078e2707ca (B07) and rva-843cecb0445d824f (waiver retirement) refine collector-tool evidence references to this execution report; judgment references remain the same actual R3 execution and sealed test/structure evidence. No new human or reviewer execution is asserted.
