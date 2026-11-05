---
slug: pose-governance-read-view-generation
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-source-coverage-integrity, pose-git-and-dispatch-snapshot-integrity, pose-writer-coordination-and-atomic-publication, pose-index-generation-integrity, pose-obligation-projection, pose-spec-authority-transfer
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:governance-read-view-generation
---

# Spec: Fixar leitura coerente e invalidar toda dependência material

## 1. Intent

### Goal

Identificar a geração dos bytes efetivamente observados por uma operação e impedir reuso ou autorização com inputs alterados ou misturados.

### Business value

B34 / D03, T4-06, PERF-09 e CONS-10. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de consistência](../reports/2026-10-12-pose-front-09-consistency-recovery.md) e a [análise de performance](../reports/2026-10-12-pose-front-08-resident-performance.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Preservar `task_type: bugfix`, fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. A hipótese observável é que `CurrentObligationSnapshot` é lido antes dos producers, mas seu digest não vincula os bytes de journals/subjects e `ProjectObligations` não audita a closure depois deles; `dirty` não distingue edições sucessivas numa árvore já dirty. Corrigir isso no domínio existente e cobrir regressões antes de ampliar consumers.

O ADR B34 deve ser aceito antes de alterar o contrato. A leitura continua útil quando a closure não puder ser provada, mas deve expor `unknown`/`incomplete`; nenhum gate pode tratar metadado de read-only como autorização. Preservar registros históricos sem rehash, rewrite ou rótulo `current` retroativo. Aplicar as decisões já aceitas em B07/B14/B15; não adotar decisões Proposed de B31/B33/B35.

## 2. Requirements

- R1: Definir read view por operação, com identidade de projeto/worktree/root autorizado, versões de engine/parser/schema e closure de dependências relevantes. Registrar bytes/digests ou evidência equivalente dos inputs realmente lidos, presença/ausência e erros; HEAD, dirty boolean, GeneratedAt, tamanho ou mtime isolados não devem representar essa closure.
- R2: A closure deve incluir conforme o escopo specs/subjects, action requests/answers/journals, review bundles/provas/results, policy/authority/contratos, curadoria, matriz/tests/fontes usados na evidência, descriptor/conjunto de índices, roots federadas/refs Git pertinentes e arquivos untracked relevantes. Alteração material de qualquer input observado deve invalidar a identidade aplicável; documentar inputs excluídos com justificativa, sem hash universal fictício.
- R3: Capturar dados e resultados dos producers numa única read view, isolada das alterações concorrentes. Validar a closure ao fim da leitura e antes de certificar um gate; interleaving detectado deve provocar retry limitado ou incoherent/incomplete/unknown explícito. Não apenas recapturar digest final e rotular resultados antigos como geração nova; incluir criação/remoção/rename e alterações que voltam ao valor anterior quando o protocolo exige observar a transição.
- R4: Preservar a coverage por produtor/recorte de B06: item descartado por validação, leitura/parse falho, ausência de prova de closure ou fonte federada inacessível deve impedir Complete no recorte afetado. Coherent, complete e current devem ser dimensões distintas e não sinônimos de execução bem sucedida; falha alheia não impõe blocker global sem relação demonstrada.
- R5: Invalidar por conteúdo/identidade relevante mesmo com mesmo tamanho/mtime, árvore já dirty, edição untracked, replace atômico, rename, checkout/reset e refs/root/contexto alterados. Watcher deve ser hint, não prova: perda/overflow/restart/ausência de eventos exige reconciliação antes de servir dados como current.
- R6: Writes do motor devem usar B14/B15 para publicar ou invalidar a geração pertinente antes do receipt de sucesso; erro/crash não deve permitir geração anunciada sem commit correspondente. Read view deve fixar descriptor e verificar os bindings pertinentes sem exigir geração global única entre roots independentes; declarar vetor/limites de consistência federada e fonte indisponível.
- R7: Gate e writer que reutilizam avaliação devem revalidar closure e authority imediatamente antes do efeito, usando CAS/fencing de B14 quando aplicável. Mudança entre revalidação e commit deve conflitar ou impedir efeito; observação read-only não é autorização persistente. Explicitar limites contra writers externos que não participam do protocolo, sem promessa de snapshot transacional universal do filesystem.
- R8: Reuso entre operações só deve ocorrer com equivalência comprovada dos inputs/versões e escopo; invalidar verdict/prova derivados quando sujeito, answer, policy, tests/matriz/index ou refs relevantes mudam. Reusar parse/metadados não promove aprovação nem validade histórica a autorização atual; sem closure verificável, recomputar ou devolver unknown.
- R9: CLI/MCP/Attention/review/closeout devem consumir a mesma read view e divulgar geração, binding, freshness, coverage e limitations no mesmo escopo. Preservar legados com limites/versionamento explícitos; não retroinventar geração ou chamar snapshot histórico de current. Declarar budgets de bytes, trabalho, retries e ciclo/profundidade federados.
- R10: Qualificar mudanças de journal/subject/policy/tests/index/refs, perdas de leitura/watcher e interleavings com barreiras/fault injection em fixtures isoladas. Implementar o contrato no domínio existente, sem daemon/IPC de B35/B33, caches ilimitados ou truncar histórico para certificar gates; B36 consumirá a geração em consultas.

## 3. Technical Plan

### Artifacts

Paths abaixo declaram o escopo da implementação em andamento. A atribuição Git e os resultados executados devem ser reconciliados antes do fechamento; a declaração de um path não prova sua entrega. O start nativo está registrado em `3565be8a`.

- modified: .pose/specs/2026-10-13-pose-governance-read-view-generation.md
- created: .pose/specs/2026-10-13-pose-governance-read-view-generation.amendments.jsonl
- created: .pose/adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
- created: .pose/starts/pose-governance-read-view-generation.json
- created: .pose/changelogs/unreleased/pose-governance-read-view-generation.md
- created: .pose/knowledge/2026-11-04-handoff-pose-b34-implementation.md
- created: pose-mcp/internal/pose/governance_read_view.go
- created: pose-mcp/internal/pose/governance_read_view_test.go
- created: pose-mcp/internal/pose/governance_read_view_probe_test.go
- created: pose-mcp/internal/pose/governance_write_guard.go
- created: pose-mcp/internal/pose/governance_write_guard_test.go
- modified: pose-mcp/internal/pose/writer_publication.go
- modified: pose-mcp/internal/pose/operation_files.go
- modified: pose-mcp/internal/pose/index_generation.go
- modified: pose-mcp/internal/pose/spec_transfer_locks.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/obligation_projection_test.go
- modified: pose-mcp/internal/pose/obligation_sources.go
- modified: pose-mcp/internal/pose/review_operation_view.go
- modified: pose-mcp/internal/pose/review_operation_view_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/start.go
- modified: pose-mcp/internal/pose/start_test.go
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/operation_result.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/state_attention.go
- created: pose-mcp/internal/cli/governance_read_view_test.go
- modified: pose-mcp/internal/mcpserver/obligations_test.go
- created: pose-mcp/internal/mcpserver/governance_read_view_parity_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/mcpserver/server.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

- modified: pose-mcp/internal/cli/closeout_plan.go
- modified: pose-mcp/internal/cli/closeout_plan_test.go
- modified: pose-mcp/internal/cli/delegated_review_capability_test.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/index_test.go
- modified: pose-mcp/internal/cli/init_delegation.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/writer_publication_test.go
- modified: pose-mcp/internal/pose/action_recovery_preflight_test.go
- modified: pose-mcp/internal/pose/action_request_test.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/capability_catalog.go
- modified: pose-mcp/internal/pose/capability_catalog_test.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/pose/followup_candidates.go
- modified: pose-mcp/internal/pose/followups_parse.go
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- modified: pose-mcp/internal/pose/git_snapshot_integrity_test.go
- created: pose-mcp/internal/pose/governance_read_view_integration_test.go
- modified: pose-mcp/internal/pose/path_state.go
- modified: pose-mcp/internal/pose/principal_keys.go
- modified: pose-mcp/internal/pose/principal_keys_test.go
- modified: pose-mcp/internal/pose/project_file.go
- modified: pose-mcp/internal/pose/review_attribution.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/pose/roots.go
- modified: pose-mcp/internal/pose/spec_transfer.go
- modified: pose-mcp/internal/pose/spec_transfer_reconcile_test.go
- modified: pose-mcp/internal/pose/spec_transfer_test.go

- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/cli/stack_manifest.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_results.go
- modified: pose-mcp/internal/cli/validate_results_test.go
- created: pose-mcp/internal/mcpserver/governance_review_read_view_parity_test.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- created: pose-mcp/internal/pose/action_write_scope.go
- modified: pose-mcp/internal/pose/assessment_freshness.go
- modified: pose-mcp/internal/pose/closeout_terminal_verification_test.go
- modified: pose-mcp/internal/pose/delegated_verdicts.go
- modified: pose-mcp/internal/pose/falsifier_observation.go
- modified: pose-mcp/internal/pose/federated_acceptance_test.go
- created: pose-mcp/internal/pose/governance_read_view_git.go
- created: pose-mcp/internal/pose/governance_read_view_git_test.go
- created: pose-mcp/internal/pose/governance_read_view_index_test.go
- created: pose-mcp/internal/pose/governance_root_vector_files.go
- created: pose-mcp/internal/pose/governance_root_vector_files_test.go
- created: pose-mcp/internal/pose/governance_scoped_write.go
- created: pose-mcp/internal/pose/index_inputs.go
- created: pose-mcp/internal/pose/index_inputs_test.go
- modified: pose-mcp/internal/pose/main_test.go
- created: pose-mcp/internal/pose/module_discovery.go
- modified: pose-mcp/internal/pose/premise_validity.go
- modified: pose-mcp/internal/pose/readiness_phases_test.go
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/pose/release_manifest_integrity.go
- modified: pose-mcp/internal/pose/release_manifest_tag_integrity_test.go
- modified: pose-mcp/internal/pose/release_review_r2_test.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/pose/review_closeout_test.go
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- created: pose-mcp/internal/pose/review_write_scope.go

### Delivery targets

- capability:governance-read-view-generation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/obligation_projection.go

### Approach and execution gates

Knowledge consumido neste plano: `knowledge:pose-b06-source-coverage-closeout`, `knowledge:pose-b07-implementation` e `knowledge:adr-shared-worktree-runtime-review`.

A retomada da execução e seus limites estão em `knowledge:pose-b34-implementation`; esse handoff registra estado parcial, sem qualificar os requisitos.

Executar incrementalmente depois das fundações aceitas B06/B07/B14/B15. A primitiva nova estende `Files` e a leitura por operação existente; não troca `Files`, não copia o repositório e não mantém cache entre operações.

#### Incrementos

1. **Núcleo da view.** Criar `governanceReadView` ligado a uma operação e a `Store`: project/root autorizado, worktree, engine/parser/read-view/schema versions e bindings de cada root federada. Registrar somente inputs lidos, inclusive bytes/digests, ausência, erro, identidade de link e inventário de diretório. `ReadFile` e `Open` devem compartilhar os mesmos bytes capturados; `Open` não pode desviar da captura. `ReadDir`, `Stat`, `Lstat`, `Readlink` e `EvalSymlinks` também participam da observação e auditoria final. Reproduzir bytes imutáveis aos clones dos producers e atribuir cada leitura a um producer/recorte explícito; não inferir attribution de goroutine nem preencher todos os producers com um único rótulo global. Um caminho de leitura por `os` ou `LoadFoo(root)` precisa migrar para `Files` ou deixar aquele producer `unknown`/`incomplete`.
2. **Projection e superfícies.** Fazer `ProjectObligations` capturar uma vez, fixar um único `PinIndexes` por root e rodar todos os producers nesse mesmo store/view; não deixar subcalls pinarem gerações diferentes. Auditar a closure ao final. CLI `state attention` e MCP `pose_obligations` continuam consumindo a projeção comum e divulgam o mesmo `read_view` ID, binding vector, freshness, coverage e limitations para o mesmo input. Uma falha de closure degrada somente producers/recortes demonstravelmente afetados; `Complete` não fica true.
3. **Review e closeout.** Fazer `WithReviewOperationView` reutilizar a primitiva de observação e os limites existentes, sem alegar snapshot transacional. Integrar views e coverage a review/closeout. Antes de qualquer gate ou efeito de start/close/ação/seal/attestation que reutilize avaliação, revalidar closure e authority no ponto de escrita, sob os locks pertinentes e com CAS/fencing B14 quando aplicável. `ObligationSnapshotChanges` sozinho não é essa proteção e atualmente não tem callers de produção. Não usar resposta read-only como autorização persistente.
4. **Finalização e receipt.** Só devolver view `current` depois da auditoria final. Se ela mudar, descartar todos os resultados daquela tentativa e repetir o read-only completo uma vez; depois disso retornar resultados úteis com `coherent=false`, `freshness=unknown` e coverage afetada incompleta, nunca relabel de bytes antigos como geração nova. Escrita confirmada usa receipt B14/B15: conflito não produziu efeito; `publish-unsynced` descreve efeito comprometido com durabilidade não confirmada; geração de índices só é anunciada após o descriptor B15. Sem cache inter-operação, uma escrita torna views anteriores históricas; ela não cria uma geração global nova.

#### Contrato, budgets e limites

- Fazer o ID ser derivado dos bytes e bindings efetivamente observados, com vetor por root (project, root digest, worktree/ref, index generation quando houver), sem caminho absoluto na saída. Divulgar a view em `snapshot.read_view`, conforme o contrato dos testes de paridade. Vincular `Snapshot.Digest` à view nas projeções novas e revalidar a prova capturada em `ObligationSnapshotChanges`; o helper continua sem conceder autorização de escrita. Incluir versões conhecidas de engine/parser/schema. Se uma versão ou input necessário não puder ser vinculado, usar `unknown`; não substituir a closure por HEAD, `dirty`, mtime, tamanho, `GeneratedAt` ou watcher.
- Reutilizar inicialmente os limites já exercitados por `reviewOperationView`: 10.000 registros, 128 MiB de bytes de entrada e 8 MiB de metadados retidos. Limitar o read-only a uma repetição; federar até 16 roots autorizados e profundidade 8, com detecção de ciclo. Exceder qualquer limite produz `incomplete`/`unknown`, sem cortar histórico ou alegar `Complete`. Reavaliar esses números somente com fixture e medição versionadas.
- Revalidar inventários e bytes observados mesmo quando não há watcher ou após restart/overflow; watcher é apenas hint. Revalidar leitores Git/refs por suas primitives B07 já existentes. B15 fornece a geração de cada conjunto de índices; roots independentes produzem vetor, não geração global.
- Uma edição externa A→B→A, restaurada antes da auditoria e sem passar por um fencing/evento observável, não pode ser provada por hash/lstat. Essa leitura só prova equivalência do conteúdo observado no instante final, não ausência histórica de transição. Quando a semântica da operação depende da transição, exigir protocolo/lock/prova de writer; se ele não participa, o gate é `unknown` e não executa. Não prometer snapshot transacional universal do filesystem.
- Manter `ObligationReport.schema_version` em 1 e acrescentar disclosure aditiva `read_view` com `schema_version: 1`; manter `Obligation` e bundles históricos em seus schemas existentes. Report legado ou histórico sem binding permanece consultável com limitation, mas é `unknown` para um gate que exige prova atual. `coherent`, `complete`, `current`, producer coverage e limitação continuam dimensões distintas. Documentar contrato e limites em `POSE.md` e no catálogo MCP.
- Não implementar watcher, daemon/IPC, cache global, conservação B31, cache graph B33 ou runtime B35 nesta entrega.
- Na construção de índices, capturar os outputs e os bindings na mesma tentativa e auditar antes da publicação sob o guard. Distinguir diretório comprovadamente sem Git de estado Git ilegível: o cold start pode construir inputs exclusivamente de filesystem, mas qualquer dependência material de refs/histórico exige a prova Git correspondente. Isso não torna uma revisão Git ausente em autorização atual nem promove metadado desconhecido a `current`. Vincular também o material federado efetivamente usado, sem incluir roots alheios ao resultado.
- O binding semântico dos índices conserva a identidade do projeto e o material estrangeiro pertinente, mas não transforma HEAD local meramente informativo em dependência persistente de todos os índices. Preservar esses campos informativos nos outputs como observados durante o build; normalizar somente a projeção usada no digest, sem nova leitura e sem alterar manifests ou evidência históricos. Os consumers que precisam do estado Git atual continuam verificando esse input em runtime, conforme B15.

## 4. Tasks

- [x] Reproduzir dirty→dirty na base publicada e registrar o controle estável.
- [x] Aceitar o ADR, definir os testes por risco e aplicar o start nativo.
- [ ] Qualificar captura/replay, auditoria, retry e budgets nas leituras reais.
- [ ] Qualificar paridade de domínio, CLI e MCP preservando os controles positivos.
- [ ] Qualificar os writers reais com vetor exato de roots e sem perder autoridade.
- [ ] Construir outputs e bindings de índices na mesma view e auditar antes de publicar.
- [ ] Passar a suíte completa, race e os checks nativos registrados na matriz.
- [ ] Reconciliar artifacts/trace e obter revisão independente do bundle selado.
- [ ] Aplicar closeout, repetir os gates finais e publicar sem adoção no hospedeiro.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Vincular resultados à closure observada e revalidada, separando read view coerente de autorização atual.
- Rationale: Existência de dados ou sucesso de transporte não prova coerência, frescor ou autoridade; preservar bindings e incerteza impede autorizar com leitura obsoleta.

### Decision D2 — leitura por operação
- Basis: R1, R2, R3, R4, R5, R8, R9, R10
- Decision: Implementar captura read-only e bounded por operação sobre `Files`, servindo os mesmos bytes aos producers e validando ao fim a closure lida. Nenhum resultado recebe identidade `current` sem binding verificável; excesso de budget ou caminho não instrumentado permanece `unknown`/`incomplete`.
- Rationale: O `reviewOperationView` já prova que reuso de catálogo pode manter leitores vivos e auditar entradas observadas; estender essa seam conserva semântica dos producers, em vez de duplicar parser ou fazer hash universal.

### Decision D3 — contrato de saída
- Basis: R4, R7, R9
- Decision: Manter o report de obrigações em schema_version 1 e adicionar o objeto `read_view` com schema_version próprio 1; reportes legados continuam legíveis, mas ausência de binding não prova estado atual nem autoriza efeito. `Obligation` continua schema v1.
- Rationale: A disclosure é aditiva; CLI/MCP podem divulgar a mesma identidade sem quebrar consumidores v1, reescrever evidência histórica ou transformar read-only em gate.

### Decision D4 — fencing e geração federada
- Basis: R3, R6, R7, R8
- Decision: Serializar writers cooperativos por root com um guard explícito, adquirido antes de recomputar a avaliação e mantido até publicação/receipt. Usar a ordem root canônica → recurso, reaproveitar locking B14 e manter CAS do destino e descriptor B15. Propagar o lease por `Store`/`Files`, sem estado global por goroutine; migrar e testar cada publisher participante. Produzir vetor de roots e recusar gate quando closure/fencing necessário não está disponível. Não criar um coordenador global entre roots nem prometer contenção de writers externos.
- Rationale: B14 protege artefatos e lifecycle writers por recurso; B15 protege o conjunto derivado; nenhum dos contratos é uma transação universal de filesystem.

As decisões D2–D4 seguem o [ADR B34 aceito](../adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md). O aceite fixa o contrato de implementação; não qualifica os requisitos sem execução dos testes e revisão.

## 6. Validation

### Strategy

`pose-mcp` tem criticality alta. Usar fixtures sintéticas, worktrees descartáveis, barreiras e fault injection. Os cenários abaixo definem o plano de validação; o Execution log registra somente os comandos realmente executados e suas limitações. Testes focados parciais não qualificam a entrega completa. A validação inclui unit, integração/contrato e superfícies CLI/MCP; race é obrigatório por producers concorrentes e writers compartilhados. Não usar benchmark para inferir SLO nem mudança de gate.

| Cenário | Comando determinístico | Evidência esperada |
|---|---|---|
| obrigatório — V1: Capture/replay de `ReadFile`, `Open`, metadata, listing, ausência/erro, binding de root/project e parser; edição untracked, mesmo tamanho/mtime, rename, replace e link. Incluir equivalência entre ordens de acesso e inventário de módulos com os mesmos filtros do scanner. | `go -C pose-mcp test ./internal/pose -run 'GovernanceReadViewProbe\|GovernanceReadViewDirtyToDirtySnapshot\|GovernanceReadViewGit' -count=1` | Mesmos bytes servidos na operação; identidade muda com conteúdo/entry/link/root/version; não há bypass por `Open`; inputs desconhecidos não certificam current. |
| obrigatório — V2: Barreira entre producers; journal/answer/subject/policy/tests/index/ref muda durante captura; retry; fonte ilegível ou obrigação descartada. | `go -C pose-mcp test ./internal/pose -run 'GovernanceReadViewProjection\|GovernanceReadViewLocalProjection\|ObligationProjection' -count=1` | No máximo uma nova execução read-only; segunda falha resulta `Coherent=false`, `freshness=unknown` e coverage localizada incompleta; nenhum resultado misto ou truncado permanece `Complete`. |
| obrigatório — V3: Gate real de start, ação e closeout/review revalida closure/authority sob locks pertinentes; mudança antes do efeito; replay de proof/predecessor. | `go -C pose-mcp test ./internal/pose -run 'GovernanceWriteGuard\|GovernanceReadView(Start\|Action\|Review\|Closeout)\|ActionRequest\|Start\|ReviewCloseout' -count=1` | Com writers POSE concorrentes, conflito ou efeito impedido; nenhum efeito se baseia só em `ObligationSnapshotChanges`/metadata. Caso de writer externo sem fencing fica explicitamente fora da garantia e não recebe prova fabricada. |
| obrigatório — V4: Pin de índice e prova de inputs atuais; índice stale, sem binding, legado consumido e ausência não consumida; inventário de módulo novo, resultado de validação/smoke alterado e tags movidas; roots federadas, ciclos/profundidade/limites, crash/conflict e receipt B14/B15. | `go -C pose-mcp test ./internal/pose ./internal/cli -run 'GovernanceReadView\|IndexGeneration\|IndexFederatedMaterial\|IndexDiscovery\|IndexInput\|WriterPublication\|ReleaseStatusUsesCapturedTagObjectAfterRefMoves' -count=1` | Um descriptor B15 por root usada e vetor sem geração global; inputs derivados stale ou sem prova degradam os consumidores pertinentes; operação sem índice permanece útil. Inventário e evidência alterados invalidam a geração; receipt distingue conflito, commit e durabilidade incerta. |
| obrigatório — V5: CLI `state attention` e MCP `pose_obligations` sobre mesma fixture; campo `read_view` aditivo, saída legada sem binding e gate que exige geração atual. | `go -C pose-mcp test ./internal/cli ./internal/mcpserver -run 'GovernanceReadViewParity\|StateAttention\|Obligations' -count=1` | Views novas divulgam os mesmos ID/bindings/coverage; resposta legada continua legível mas é `unknown` para autorização; campos existentes e ambas as versões schema v1 permanecem. |
| obrigatório — V6: Concorrência entre producers e mutations; isolamento de clones/roots e ausência de attribution global. | `go -C pose-mcp test -race ./internal/pose ./internal/cli ./internal/mcpserver -run 'GovernanceReadView\|ReviewOperationView\|ActionRequest\|Start' -count=1` | Sem race, aliasing de bytes ou troca de producer; budgets e replays permanecem locais à operação. |
| obrigatório — V7: Validação registrada da matriz e gates integrados do módulo. | `pose validate --strict --module pose-mcp --report` e `pose check --strict` | Checks do módulo passam e matriz continua válida; report de validação aponta os testes de B34, sem declarar CI remoto. |
| opcional — V8: Uma restauração externa A→B→A sem evento/fencing, para qualificar o limite real de observabilidade. | `go -C pose-mcp test ./internal/pose -run 'GovernanceReadViewExternalABA' -count=1` | O teste documenta que hash/lstat não detectam transição que desaparece antes da auditoria; nenhuma alegação de snapshot transacional ou proteção de gate se infere desse teste. |

### Deterministic checks

Executar os cenários obrigatórios acima, a suíte Go do módulo e os checks da matriz atualizados; incluir controles positivos e negativos. `pose artifact-check --spec pose-governance-read-view-generation --strict`, `pose surface-check --spec pose-governance-read-view-generation --strict` e os gates de segurança pertinentes continuam obrigatórios no closeout. Se a matriz ganhar um check B34 permanente, validar seu schema com `pose check --strict`. Atualizar evidence/trace somente com comandos realmente executados; nenhum lint de documentação qualifica o target funcional.

### Execution log

- Em 2026-11-04, a sexta rodada focada congelada compilou os três pacotes e executou 247 testes/subtestes aprovados e 88 falhas. A seleção passou a executar também os testes CLI anteriormente bloqueados pela compilação, portanto não é comparável por contagem à rodada anterior. Persistiram falhas de contexto do default root sem ID explícito, cold start de índices, closeout/review e um novo print direto; logs: `/tmp/b34-focused-frozen-6.jsonl` e `/tmp/b34-focused-6-failures.txt`. Nenhum requisito é qualificado por essa rodada incompleta.
- Em 2026-11-04, a quinta rodada focada congelada selecionou os parents que falharam na suíte completa e os casos novos: 159 testes/subtestes aprovados e 46 falhas. Os testes de abertura/resolução MCP e de paridade passaram; CLI não compilou por helpers de teste colidentes, import `io/fs` ausente e campos de uma fixture inconsistentes. O domínio terminou em 45 s, ainda com falhas de identidade federada, retenção histórica e controles novos. Logs: `/tmp/b34-focused-frozen-5.jsonl` e `/tmp/b34-focused-5-failures.txt`. Esses números pertencem a uma seleção diferente da suíte completa e não qualificam o target.
- Em 2026-11-04, o handoff `knowledge:pose-b34-implementation` foi registrado; `pose knowledge-housekeeping list-expired` não listou vencidos e `pose knowledge-check --strict` passou com 21 artefatos, zero erros e zero referências inválidas. Essa validação cobre o registro operacional, não o target funcional.
- Em 2026-11-04 (UTC), a quarta rodada congelada executou `go -C pose-mcp test -json ./... -count=1 -timeout=10m` com temp/cache ajustados: 3.822 testes/subtestes aprovados, 146 falhas (incluindo subtestes e seus parents) e 12 skips. O pacote domínio terminou em 298 s e CLI em 213 s; MCP, CLI e domínio falharam. As falhas incluem contexto de projeto selecionado nas ações, abertura capturada de diretórios, guards aninhados e preservação do contrato histórico de closeout. Logs: `/tmp/b34-full-frozen-4.jsonl` e `/tmp/b34-full-4-failures.txt`. Esta rodada não qualifica entrega nem aprovação atual.
- Em 2026-11-04 (UTC), a terceira rodada focada congelada executou domínio, CLI e MCP com `-count=1 -timeout=5m`: 179 testes aprovados e 15 falhas. Os casos novos de bindings, consumo de índices e guards de ações federadas passaram; a integração projeção/review, os doubles de diretório e os controles positivos de closeout ainda falharam. Log: `/tmp/b34-focused-frozen-3.jsonl`. A rodada não qualifica o target composto.
- Em 2026-11-04 (UTC), a segunda rodada focada congelada registrou 161 testes aprovados, mas falhou em compatibilidade de review view, coverage de phase readiness e limite de impressão direta do CLI. O pacote MCP passou. Log: `/tmp/b34-focused-frozen-2.jsonl`. As correções posteriores e novos casos de índices ainda não estão qualificados por essa rodada.
- Em 2026-11-04 (UTC), a primeira suíte completa (`env GOTMPDIR=/var/tmp/pose-work-go TMPDIR=/var/tmp/pose-work-go go -C pose-mcp test ./...`) terminou com falhas de integração; log de execução em `/tmp/b34-increment-full.log`. As causas observadas incluem classificação do lock de root como mudança Git, primeira escrita com `.pose` ausente, classificação de contenção de índices, prova de ausência saindo da root e leitura federada ainda não composta. A rodada não qualifica a entrega. Os checks isolados de `./internal/scaffold` e `./internal/scaffold/distpolicy` passaram após a regeneração dos manuais; `TestCatalogMatchesGolden` passou após atualizar somente a descrição aditiva de `snapshot.read_view`.
- Na revisão de 2026-11-04 (UTC), `pose assess integrate` executou e registrou 70 contratos, 6 ativos e 64 gaps de consumidores não observados; isso é inventário estático, não prova de integração funcional. `pose assess tech-debt` encontrou 1 marcador coberto e nenhum descoberto no escopo lido, mas reportou scan **incompleto**: duas specs históricas inválidas não puderam ser lidas. Essa limitação permanece explícita e não equivale a check completo aprovado.
- Em 2026-11-04 (UTC), `env GOTMPDIR=/var/tmp/pose-work-go TMPDIR=/var/tmp/pose-work-go go -C pose-mcp test ./... -run '^$'` passou na compilação de todos os pacotes; não executou testes funcionais. Antes disso, os testes de transporte apresentaram ciclo de imports e um import não utilizado, ambos corrigidos preservando as regressões MCP existentes. Duas tentativas de build em `/tmp` falharam por quota de disco; essas falhas de infraestrutura não qualificam nem refutam o comportamento funcional.
- Em 2026-11-04 (UTC), `go -C pose-mcp test ./internal/pose -run '^TestGovernanceWriteGuard' -count=1` passou nos testes da primitiva de guard, antes da integração completa dos escritores. Esse resultado isolado não qualifica R6/R7.
- Na retomada de 2026-11-04 (UTC), o contexto nativo resolveu a autoridade `proj.pose-dist` e a spec `in-progress` no clone isolado, HEAD `dcffc886`. `go -C pose-mcp test ./internal/pose -run 'GovernanceReadView|GovernanceWriteGuard' -count=1` terminou em erro de compilação no núcleo parcial (colisão de campo/método `metadata`, assinatura de handle divergente e helpers de diretório ausentes); nenhum teste funcional executou nessa rodada. O assessment do componente foi atualizado. O check obrigatório `governance-read-view-integration` foi declarado na matriz para executar domínio/CLI/MCP com `-race`; sua declaração ainda não constitui resultado aprovado.
- Em 2026-11-03, a regressão `TestGovernanceReadViewDirtyToDirtySnapshot` foi executada em worktree descartável de `7079eb86`, com apenas o teste novo aplicado. Comando: `go -C pose-mcp test ./internal/pose -run '^TestGovernanceReadViewDirtyToDirtySnapshot$' -count=1`. O controle estável passou; a alteração de bytes com mesmo tamanho e mtime falhou com `same-size spec edit kept the operation identity` (exit 1). Isso reproduz o defeito antes da implementação; não é validação da correção. A primeira tentativa ficou bloqueada pelo cache Go somente leitura e não executou o teste.

### Recovery checkpoint — 2026-11-05

O clone temporário deixou de estar disponível. O trabalho foi reconstruído em
clone isolado persistente a partir de patches e leituras literais das sessões;
a proveniência e as lacunas estão em `recovery-notes/RECOVERY.md` e
`recovery-notes/DOC-RECOVERY.md`, arquivos locais de recuperação. A compilação de
todos os pacotes passou (`go -C pose-mcp test ./... -run '^$'`). A primeira
seleção de regressões atual teve 134 eventos de teste/subteste aprovados e seis
falhos na CLI; não qualifica a entrega. O helper de roteamento ausente foi
restaurado literalmente e ganhou novos controles de boundary e lease.

Os registros antigos de start e amendments não foram recuperados integralmente.
Os relatos de start e commits acima são históricos; não constituem recibos no
clone atual, cujo lifecycle veio da base draft. Um novo start nativo será
prospectivo, posterior à recuperação, sem declarar precedência sobre o código
recuperado. Nenhum resultado histórico foi promovido a evidência atual.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V2, V5 |
| R2 | V1–V4 |
| R3 | V2, V3, V8 |
| R4 | V2, V5 |
| R5 | V1, V2, V8 |
| R6 | V4 |
| R7 | V3, V4, V8 |
| R8 | V2, V3, V5 |
| R9 | V4, V5, V7 |
| R10 | V1–V8 |

## 7. Final Report

### Delivered scope

O código e o ADR aceito foram recuperados em clone persistente. Núcleo de captura, projeção, divulgação CLI/MCP, geração de índices e coordenação dos escritores estão em validação. A compilação atual passou; a primeira seleção atual ainda apresentou seis falhas CLI. Nenhum requisito R1–R10 está qualificado nesta edição. Correções, suíte completa, testes de concorrência, revisão final e gates de closeout permanecem pendentes. Os relatos de start anteriores são históricos; consulte o checkpoint de recuperação para o estado dos registros nativos.

### Residual risks

A closure é um contrato por operação, não um hash genérico do diretório. Um writer externo pode alterar e restaurar bytes entre leituras sem deixar evidência; só locks/fencing participados pelos writers qualificam transições requeridas. A view não é um snapshot transacional nem autorização persistente. O ADR aceita a divulgação aditiva com versão própria, e os gates devem tratar ausência de proof como `unknown`; a validação dessa composição ainda está em andamento.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
