---
slug: pose-review-cleanup-removes-go-build-leftovers
status: in-progress
created_at: 2026-11-14
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:review-cleanup-removes-go-build-leftovers
---

# Spec: pose-review-cleanup-removes-go-build-leftovers

## 1. Intent

### Goal
Fazer a limpeza de uma execução de revisão delegada remover os diretórios de build do Go que um `go test` interrompido do revisor deixou no diretório de execução, para que uma revisão aprovada não seja recusada por limpeza incompleta.

### Business value
Em 2026-11-14 duas revisões seguidas da spec `pose-adopt-request-shows-what-it-sets` foram aprovadas pelo Codex e recusadas pelo motor ("the run's cleanup is not confirmed complete"). O diretório de execução retido tinha 175 MB, acima do orçamento de 128 MiB do inventário de limpeza: 151 MB eram um `go-build*` com o binário de teste da CLI que o revisor compilou. A remoção desses diretórios existia só no wrapper do perfil de revisão do Claude; para qualquer outro adaptador, uma revisão que compila os testes da CLI esgota o orçamento e a run deixa de governar.

### Constraints
- O inventário limitado continua valendo para todo o resto do diretório de execução.
- Um link com nome de `go-build*` nunca é seguido.

### Non-goals
- Não elevar o orçamento do inventário do diretório de execução.

---

## 2. Requirements

### Functional
- R1: On Unix platforms, when a review run is cleaned up, every real directory named `go-build<digits>` at the top of its runtime directory shall be removed under its own bounded inventory before the runtime directory's inventory, for every adapter. On other platforms the go-build directory is left to the runtime inventory, unchanged from before this spec (follow-up F3).
- R2: An entry named like one that is a link or not a directory shall be left to the runtime inventory, and its target shall never be touched.
- R3: A go-build directory that exceeds its own bounded inventory, or that changes identity during cleanup, shall leave the cleanup incomplete, as any other unconfirmed cleanup.

### Non-functional
- Orçamento por diretório de build: 200.000 entradas e 8 GiB.

### Security
- A remoção usa a raiz do diretório de execução já capturada e confere a identidade do diretório aberto.

### Compatibility
- Nenhum formato muda.

---

## 3. Technical Plan

### Affected areas
- `pose/review_adapter_containment.go`: `removeReviewGoBuildLeftovers` antes do inventário do diretório de execução.

### Artifacts
- modified: pose-mcp/internal/pose/review_adapter_containment.go
- modified: pose-mcp/internal/pose/review_adapter_containment_regression_test.go
- created: pose-mcp/internal/pose/review_go_build_cleanup_unix.go
- created: pose-mcp/internal/pose/review_go_build_cleanup_other.go
- created: pose-mcp/internal/pose/review_go_build_cleanup_unix_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:review-cleanup-removes-go-build-leftovers module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto por `pose delegate review`, cuja limpeza de execução chama esta remoção. Check permanente `review-cleanup-removes-go-build-leftovers-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestAdapterCleanupRemovesGoBuildLeftovers|TestAdapterContainmentCleanup' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Nenhum novo: o diretório removido é do Go, criado sob o TMPDIR que o motor dá à execução.

---

## 4. Tasks

### Implementation
- [x] Remoção limitada dos `go-build*` de primeiro nível (R1, R2, R3).

### Validation
- [x] Teste com um `go-build` de 200 MiB e 4.200 arquivos e um link com o mesmo padrão de nome; teste sincronizado de troca e sumiço do diretório depois de aberto.

---

## 6. Validation

### Strategy
Teste de regressão sobre o fixture de limpeza de contenção existente; testes de contenção existentes.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-14
- Environment: Linux, Go 1.26.
- Notes: diretório retido da run `rrn-331ef7311adb902b`: 1.730 entradas, 175 MB, com `go-build4066122796` de 151 MB. Sem a mudança, o teste reprova com a mesma mensagem da recusa ("runtime cleanup inventory incomplete; resources retained"); com ela, passa.

### Results summary
- Successes: limpeza completa com o leftover; link não seguido.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestAdapterCleanupRemovesGoBuildLeftovers test:TestReviewGoBuildEnumerationStopsAtTheBudget test:TestReviewGoBuildEnumerationDescriptorIsCloseOnExec test:TestGoBuildLeftoversDoNotCountAgainstTheRuntimeInventory check:review-cleanup-removes-go-build-leftovers-integration
- R2 [satisfied] test:TestAdapterCleanupRemovesGoBuildLeftovers
- R3 [satisfied] test:TestAdapterCleanupRefusesAReplacedGoBuildDirectory test:TestAdapterCleanupRefusesAGoBuildReplacedBeforeRemoval test:TestAdapterCleanupDoesNotBlockOnAFIFOReplacement test:TestAdapterCleanupDoesNotBlockOnAFIFOBeforeTheFirstOpen test:TestAdapterCleanupRefusesANestedGoBuildDirectoryReplacedBeforeRemoval check:release-cross-build-windows check:release-cross-build-darwin check:release-test-compile-windows check:release-test-compile-darwin

### Known gaps
- Nenhuma.

---

## 7. Final Report

### Delivered scope
A limpeza de uma execução de revisão remove, com inventário limitado próprio, os diretórios de build do Go deixados no diretório de execução, para qualquer adaptador, sem seguir links.

### Files and modules changed
- `pose/review_adapter_containment.go`, teste de regressão, matriz.

### Validation executed
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Result: verde.

### Residual risks
- Nenhum.

### Follow-ups

- [open] F1: abrir a raiz da execução e a worktree na limpeza de contenção de forma não bloqueante e exigindo diretório; `os.Root.OpenRoot` bloqueia se o nome for trocado por um FIFO entre a conferência e a abertura. Os diretórios `go-build` desta spec já não têm essa janela. (owner:@pose-maintainers crit:medium review:2026-12-13)
- [open] F2: os testes de `internal/cli` não compilam para Windows (`syscall.Kill` em `delegate_mcp_test.go`, e o `go vet` para no primeiro erro de cada pacote); estender `release-test-compile-windows` ao módulo inteiro quando compilarem. (owner:@pose-maintainers crit:low review:2026-12-13)
- [open] F3: remover os diretórios `go-build` da limpeza também no Windows, com abertura relativa a handle e sem seguir reparse points; hoje ficam para o inventário da execução, como antes desta spec, e uma execução de revisão no Windows com um binário de teste grande continua com a limpeza incompleta. (owner:@pose-maintainers crit:low review:2026-12-13)

### Review

Rodada 1 (run rrn-11320f4eeb5bcde3, attestation rva-823ad7b9b9b0738b, changes-requested): 1 bloqueador, tratado. F1 a identidade do diretório de build só era conferida ao abri-lo: renomeado para fora e trocado por um vazio durante a limpeza, o substituto seria removido e a limpeza confirmada, e o sumiço era aceito em silêncio. Depois de esvaziá-lo, o nome tem de ligar ainda o mesmo diretório, ou a limpeza fica incompleta; um teste sincronizado cobre a troca e o sumiço. A ferramenta `validate` saiu reprovada só porque a reexecução no sandbox do revisor estourou 600 s; a evidência selada passou.

Rodada 2 (run rrn-78b11adc9ff36dd0, attestation rva-b65df389f4e7a242, changes-requested): o F1 ficou parcial e veio 1 bloqueador, tratado. N1 entre a última conferência do nome e a remoção, o diretório ainda podia ser trocado e a remoção do substituto confirmaria a limpeza. O diretório agora fica aberto durante a remoção, e só a confirmação, pelo descritor, de que ele ficou sem nenhum link (o original removido) confirma a limpeza; uma plataforma sem contagem de links não prova e recusa. Um teste sincronizado troca o diretório entre a conferência e a remoção.

Rodada 3 (run rrn-1a96cee8e24eb008): o motor recusou a conclusão porque o rascunho deu aos achados novos ids com prefixo de rodada (`R3-N1`); os dois achados foram tratados mesmo assim. N1 (alto) `syscall.Stat_t` num arquivo sem build tag não compila no Windows, alvo do release: a contagem de links passou para arquivos por plataforma (`unix` e o resto, que recusa), e a matriz ganhou os checks `release-cross-build-windows` e `release-cross-build-darwin`, cuja ausência deixou a rodada 2 passar a validação completa sem compilar para Windows. N2 a reabertura pelo nome (`root.Open`) bloqueava se o diretório fosse trocado por um FIFO: o diretório verificado fica aberto, sem reabertura, até a confirmação da remoção por ele; um teste sincronizado troca por um FIFO e exige o fim sem bloqueio. A janela entre a primeira conferência e a abertura é a da limpeza inteira (follow-up F1).

Rodada 4 (run rrn-67a184e6a36ce92e, attestation rva-fed978aec59d28de, changes-requested): os achados das rodadas anteriores estão corrigidos, e vieram 2 bloqueadores, tratados. N3 o teste do FIFO usava `syscall.Mkfifo` num arquivo de teste compilado também no Windows: os testes de FIFO foram para `review_go_build_cleanup_unix_test.go`, e a matriz ganhou `release-test-compile-windows` e `release-test-compile-darwin` (`go vet`, que compila os testes). N4 a primeira abertura (`OpenRoot`) ainda bloqueava num FIFO posto entre a listagem e a abertura: no Unix a remoção do `go-build` passou a ser toda feita por descritores (`openat` com `O_DIRECTORY|O_NOFOLLOW|O_NONBLOCK` relativo ao descritor da raiz, `unlinkat` relativo ao descritor que contém a entrada, com inventário limitado), e o que não abre como diretório fica para o inventário da execução. Nas outras plataformas o `go-build` fica para esse inventário, que mantém a limpeza incompleta se exceder os limites. Um teste sincronizado troca por um FIFO antes da primeira abertura e exige o fim sem bloqueio.

Rodada 5 (run rrn-dfbb00c5cb708475, attestation rva-4ef3d3f9821cfeb3, changes-requested): N3 e N4 corrigidos, e vieram 4 bloqueadores, tratados. N5 a implementação fora do Unix não faz nada, e o R1 prometia a remoção em toda plataforma: o R1 foi restringido ao Unix, e o Windows ficou como follow-up F3, sem afirmar entrega que não há. N6 `Readdirnames(-1)` lia todas as entradas antes de consultar o limite: a leitura passou a ser em lotes de 256, cobrados do limite ao serem lidos, e o topo do diretório da execução também é lido em lotes, até o limite de 4.096 do inventário. N7 um diretório trocado ou sumido entre a listagem e a abertura era entregue ao inventário: a identidade lida antes da abertura liga o nome, e a troca deixa a limpeza incompleta; o teste agora exige esse erro. N8 `unix.Dup` tira o close-on-exec: a duplicação passou a ser `F_DUPFD_CLOEXEC`, atômica. Cada correção tem teste que reprova sem ela.

Rodada 6 (run rrn-18325d957ae046e0, attestation rva-2f60a62a1aaa0648, changes-requested): N6 a N8 corrigidos e N5 substituído pela emenda do R1; vieram 2 bloqueadores, tratados. N9 os subdiretórios eram removidos sem a confirmação feita no topo: toda remoção de diretório, em qualquer nível, passa por `unlinkReviewGoBuildBound`, que mantém o descritor aberto, confere o nome antes e exige, por ele, o original sem links depois. N10 o topo contava os `go-build` contra o limite de 4.096 do inventário de que eles saem antes: cada `go-build` é removido à medida que é lido, sem acumular nomes, e só as outras entradas contam. Um teste para cada um reprova sem a correção.

