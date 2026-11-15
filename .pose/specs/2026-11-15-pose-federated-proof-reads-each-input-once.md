---
slug: pose-federated-proof-reads-each-input-once
status: in-progress
created_at: 2026-11-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:federated-proof-reads-each-input-once
---

# Spec: pose-federated-proof-reads-each-input-once

## 1. Intent

### Goal
Fazer a prova federada de um projeto que depende do pose-dist ler cada entrada imutável uma vez por processo: o material de um change set num commit, o snapshot de uma release num commit, o commit a que o objeto de uma tag chega e o escopo que o conteúdo de um bundle declara.

### Business value
Ao fixar o pose-dist em `bf2ff02` com o pin de federação certo, o `pose index` do harne8 passou a levar 23 minutos (1.383 s) e o `check --tolerant` do pre-commit chegou a 11 GB em um minuto, com 16 workers. Enquanto o pin estava velho (`485d59a`), a prova federada parava cedo na divergência de revisão e o custo ficava escondido. Medido pelo caminho que a ferramenta toma (contagem de processos git e perfil de CPU): a preparação de cada bundle de spec recalculava o material de todos os change sets do grafo com um `ls-tree` por set (20 mil processos em três minutos); cada leitura das releases arquivadas relia cada snapshot blob a blob e resolvia cada tag duas vezes, com três processos por tag; e cada consulta por escopo relia e reparseava o JSON de todos os bundles (63% da CPU).

### Constraints
- Nenhum resultado muda: os índices do harne8 gerados com e sem a mudança são idênticos byte a byte.
- Só entra no cache o que é função de conteúdo endereçado: um commit, um objeto de tag, os bytes de um bundle. Uma ref (HEAD, tag) é lida a cada vez.

### Non-goals
- O recálculo do material federado por escopo dentro do `check` (follow-up F1).

---

## 2. Requirements

### Functional
- R1: The current material of a change set shall be computed once per process for the same inputs at the same commit, with HEAD resolved once per graph; a HEAD that cannot be resolved shall be read per set and not kept.
- R2: The snapshot of a release tag at a commit shall be read from git once per process; each reader shall receive its own copy, and a failing git read shall not be kept.
- R3: The commit a tag object peels to, and whether it is annotated, shall be kept per object, while the tag's ref is read every time; the archived releases shall list the tags' refs in one git process and read each snapshot at the commit already resolved, answering as `ResolveReleaseTag` answers each tag alone.
- R4: What a review bundle's bytes declare about their scope shall be parsed once per content digest, each file keeping its own artifact path in a loss.
- R5: The kept entries of every cache shall be bounded for a long-running process.
- R6: A key missed by several callers together shall be computed once; a result that records a failing read (an unreadable material, a snapshot with gaps, a failed peel) shall not be kept, and a shallow checkout shall read its material uncached.

### Non-functional
- O `pose index` do harne8 com o pose-dist em `bf2ff02` cai de 1.383 s para no máximo 300 s, com índices idênticos.

### Security
- Nenhuma mudança: a chave de cada cache é o conteúdo ou o objeto, e uma ref nunca é cacheada.

### Compatibility
- Nenhum resultado muda.

---

## 3. Technical Plan

### Affected areas
- `pose/material_binding.go` e `cli/material_engine.go`: HEAD resolvido por grafo e digest guardado por (raiz, commit, entradas).
- `pose/release_manifest_integrity.go` e `pose/release_archive.go`: snapshot por commit, resolução de tag por objeto e listagem única das refs.
- `pose/review_operation_view.go`: parse do cabeçalho por digest do conteúdo.

### Artifacts
- modified: pose-mcp/internal/pose/material_binding.go
- modified: pose-mcp/internal/cli/material_engine.go
- modified: pose-mcp/internal/pose/release_manifest_integrity.go
- modified: pose-mcp/internal/pose/release_archive.go
- modified: pose-mcp/internal/pose/review_operation_view.go
- created: pose-mcp/internal/pose/current_material_cache_test.go
- created: pose-mcp/internal/pose/release_snapshot_cache_test.go
- created: pose-mcp/internal/pose/review_bundle_header_cache_test.go
- created: pose-mcp/internal/pose/kept_cache.go
- created: pose-mcp/internal/pose/kept_cache_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:federated-proof-reads-each-input-once module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por `pose index`, `pose check` e por toda verificação de bundle federado. Check permanente `federated-proof-reads-each-input-once-integration`, evidenceClass `integration`, required: `go test ./internal/pose -run 'TestCurrentMaterialIsReadOncePerCommit|TestCurrentMaterialFollowsAMovedHead|TestCurrentMaterialWithoutAHeadIsNotKept|TestCurrentMaterialKeptDigestsAreBounded|TestEvidenceIsJudgedAgainstTheCurrentMaterial|TestAReleaseSnapshotIsReadOncePerCommit|TestAKeptReleaseSnapshotIsCopiedToEachReader|TestAMovedReleaseTagIsReadAtItsNewCommit|TestAFailedReleaseSnapshotReadIsNotKept|TestAReleaseTagPeelIsKeptPerObject|TestAListedReleaseTagResolvesAsAloneOne|TestABundleHeaderIsParsedOncePerContent|TestAKeptValueIsComputedOnceForConcurrentMisses|TestADeclinedKeptValueIsComputedAgain|TestAnUnreadableMaterialIsAskedAgain|TestASnapshotWithGapsIsReadAgain|TestAFailedReleaseTagPeelIsAskedAgain|TestARecoveredReadIsComputedOnceAndKept|TestABoundCrossingKeepsAPendingKey|TestAConcurrentBurstKeepsNoMoreThanTheBound' -count=1`.

### API/contract changes
- Nenhuma.

### Data/storage changes
- Nenhuma.

### Technical risks
- Um cache com chave errada daria um resultado velho; cada chave é um conteúdo endereçado, e os testes provam a tag movida, o HEAD que andou e o conteúdo mudado.

---

## 4. Tasks

### Implementation
- [x] Material por commit (R1), snapshot por commit (R2), tag por objeto e listagem única (R3), cabeçalho por conteúdo (R4), limites (R5).

### Validation
- [x] Testes de cada cache, com prova de reprovação; índices do harne8 idênticos antes e depois; tempo medido.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-15
- Context: o primeiro cache (snapshots) não mudou o tempo sozinho, nem o segundo (material) sozinho; cada um só deslocou o gargalo para o outro, e os dumps de goroutines, por amostrarem instantes, apontaram fases que não eram as dominantes.
- Options considered: (a) parar no primeiro cache sem efeito; (b) medir cada passo pelo caminho da ferramenta (processos git e perfil de CPU) e manter só os caches que, juntos, mudam o tempo.
- Decision: (b).
- Rationale: medidos juntos, material e snapshots levaram o `index` de 1.383 s a 959 s; tags por objeto e listagem única, a 546 s; o parse por conteúdo, a 249 s.
- Consequences: cada cache tem a chave no conteúdo endereçado, e nenhuma ref entra em cache.

---

## 6. Validation

### Strategy
Teste de cada cache com prova de reprovação; os índices do harne8 gerados por cada variante comparados byte a byte com os do motor sem mudança; tempo do `index` medido em cada passo.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-15
- Environment: Linux, Go 1.26, 16 núcleos; harne8 com o pose-dist em bf2ff02 e as raízes que `scripts/pose-cli-local.sh` injeta.
- Notes: `pose index` do harne8: 1.383 s sem mudança; 1.496 s só com o cache de snapshots; 1.530 s só com o de material; 959 s com os dois; 570 s com tags por objeto; 546 s com a listagem única; 249 s com o parse por conteúdo. Os três índices (`roadmaps.json`, `spec-graph.json`, `delivery-integrity.json`) idênticos em todas as variantes. Processos git em três minutos: 45 mil (20 mil `ls-tree HEAD`) sem mudança, 14 mil com os três primeiros caches. O `check --tolerant` com 4 workers termina (37 min, 2,9 GB); com 16 passa de 11 GB: o recálculo do material federado por escopo é o follow-up F1.

### Results summary
- Successes: `index` 5,5 vezes mais rápido com resultados idênticos.
- Failures: nenhuma.
- Warnings: o `check` continua inviável no pre-commit do harne8 até o F1.

### Requirement trace
- R1 [satisfied] test:TestCurrentMaterialIsReadOncePerCommit test:TestCurrentMaterialFollowsAMovedHead test:TestCurrentMaterialWithoutAHeadIsNotKept test:TestEvidenceIsJudgedAgainstTheCurrentMaterial
- R2 [satisfied] test:TestAReleaseSnapshotIsReadOncePerCommit test:TestAKeptReleaseSnapshotIsCopiedToEachReader test:TestAMovedReleaseTagIsReadAtItsNewCommit test:TestAFailedReleaseSnapshotReadIsNotKept
- R3 [satisfied] test:TestAReleaseTagPeelIsKeptPerObject test:TestAListedReleaseTagResolvesAsAloneOne
- R4 [satisfied] test:TestABundleHeaderIsParsedOncePerContent
- R6 [satisfied] test:TestAKeptValueIsComputedOnceForConcurrentMisses test:TestADeclinedKeptValueIsComputedAgain test:TestARecoveredReadIsComputedOnceAndKept test:TestABoundCrossingKeepsAPendingKey test:TestAnUnreadableMaterialIsAskedAgain test:TestASnapshotWithGapsIsReadAgain test:TestAFailedReleaseTagPeelIsAskedAgain
- R5 [satisfied] test:TestCurrentMaterialKeptDigestsAreBounded test:TestAConcurrentBurstKeepsNoMoreThanTheBound check:federated-proof-reads-each-input-once-integration

### Known gaps
- O limite do cache de cabeçalhos (R5) não tem teste próprio; é o mesmo mecanismo do de material, que tem.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

- [open] F1: o `check` recalcula o material federado inteiro para cada escopo (`checkReviewCloseout` → `ReadIndex` → `checkIndexInputs` → `BuildIndexFederatedMaterialIn`), e cada worker retém a própria read view: no harne8, 37 min com 4 workers e mais de 11 GB com 16. Calcular o material uma vez por processo, com chave no fechamento de entradas da read view, e limitar a memória por worker. (owner:@pose-maintainers crit:high review:2026-11-22)

### Review

Rodada 1 (run rrn-4a532feec7405caf, attestation rva-923a9017fb572ebd, changes-requested): 4 bloqueadores, tratados. F1 o cache de material guardava o marcador `unreadable` e não conferia o clone raso: o estado de clone raso passou a ser conferido a cada resolução do HEAD (um repositório raso lê sem cache), e um material ilegível não é guardado. F2 o cache de snapshots guardava snapshots com gaps, que podem registrar uma leitura do git que falhou: só um snapshot íntegro, ou um commit sem manifesto, é guardado. F3 uma resolução de tag que falhou era guardada como "não é commit": só uma resolução que chegou a um commit é guardada. F4 duas chamadas que perdiam a mesma chave calculavam as duas: os quatro caches passaram a um `keptCache` que coordena o trabalho em andamento por chave (quem chega espera o primeiro), e um resultado que não é guardado é recalculado por quem esperava. Um teste por achado, cada um reprova com o comportamento anterior.

Rodada 2 (run rrn-a37f896b2e3bf62f, attestation rva-87d5bfabc01ff84e, changes-requested): F1 a F3 corrigidos, F4 parcial, e 2 bloqueadores, tratados. N1 quem esperava um resultado não guardado calculava por fora, sem publicar um sucesso depois da recuperação: volta a adquirir a chave pelo cache, e a leitura recuperada é calculada uma vez e guardada. N2 o despejo por capacidade trocava o mapa inteiro, inclusive chaves ainda em cálculo: só os valores concluídos saem, e uma chave em cálculo mantém a entrada. Um teste para cada, e os dois reprovam com o helper da rodada 1.

Rodada 3 (run rrn-06a1e0155a43be30, attestation rva-4095164cf39f081d, changes-requested): F4, N1 e N2 corrigidos, e 1 bloqueador, tratado. N3 o limite só era aplicado ao admitir uma chave nova, e uma rajada de chaves distintas calculadas juntas ficava inteira guardada: o limite passou a ser aplicado quando um valor é guardado, contando os valores concluídos, e as chaves em cálculo continuam preservadas. O teste da rajada reprova sem a aplicação na conclusão (5 valores guardados com limite 1).

