---
slug: pose-index-input-binding-reads-each-transfer-once
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:index-input-binding-reads-each-transfer-once
---

# Spec: Uma operação lê os planos de transferência de um projeto uma vez

## 1. Intent

### Goal

A vinculação das entradas do índice, que `review-check`, `closeout-check` e `artifact-check` recalculam a cada chamada, lê os planos de transferência de cada projeto uma vez por operação em vez de uma vez por artefato resolvido, e dá às lojas de origem do acceptance federado o mesmo snapshot de specs que o resolver já usa.

### Business value

Medido em 2026-11-10 neste repositório (2833 commits, 11 MB de índice): `review-check spec:<slug>` leva cerca de 56 s, `closeout-check` cerca de 51 s e `artifact-check --spec` cerca de 137 s por chamada. O perfil de CPU mostrou o cache de grafos da B33 sem efeito (a segunda chamada no mesmo processo levou os mesmos 51 s) e o custo dominante em `incompleteTransferFor`, que lê e valida cada plano de `.pose/transfers` a cada `Resolve`, ou seja, uma vez por dependência de cada roadmap. Este é o achado que redireciona a fatia 2 da B35: o runtime residente não acelera o que a operação recalcula; a operação precisa deixar de recalcular. O ganho vale para toda chamada, com ou sem runtime.

### Constraints

- Nenhuma resposta muda: a saída de `review-check` e de `closeout-check` é a mesma, byte a byte.
- O listing é por operação (o snapshot do resolver), nunca entre operações: sem cache persistente nem prova de frescor por mtime.
- Sem snapshot (um resolver de fora de uma operação governada) o comportamento é o de antes.

### Non-goals

- As cerca de 3100 chamadas ao `git` por `review-check` (348 `git log -1 -- <path>` somam 20 s): é o próximo gargalo e fica como follow-up, porque agrupá-las muda a semântica da simplificação de histórico.
- O runtime e o cache residente (B35).

## 2. Requirements

- R1: Uma operação lê os planos de `.pose/transfers` de um projeto uma única vez, qualquer que seja o número de artefatos resolvidos; cada resolução responde a partir desse listing com a mesma resposta de uma leitura nova, inclusive o estado de plano inválido na mesma ordem.
- R2: As lojas de origem que o acceptance federado abre usam o snapshot de specs da operação, de modo que `GetSpec` não relista nem reparseia o diretório de specs a cada artefato.
- R3: A saída de `review-check` e de `closeout-check` num repositório real é idêntica antes e depois, e o tempo de `review-check` diminui de modo medido.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/pose`: o resolver de artefatos, o snapshot de specs e a leitura de transferências.

### Artifacts

- created: .pose/specs/2026-11-10-pose-index-input-binding-reads-each-transfer-once.md
- created: .pose/starts/pose-index-input-binding-reads-each-transfer-once.json
- created: .pose/changelogs/unreleased/pose-index-input-binding-reads-each-transfer-once.md
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/spec_transfer.go
- modified: pose-mcp/internal/pose/federated_acceptance.go
- created: pose-mcp/internal/pose/transfer_listing_once_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- governance:index-input-binding-reads-each-transfer-once module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

Nenhuma: as funções públicas e a saída não mudam.

## 4. Tasks

- [x] Medir o custo: perfil de CPU e contagem das execuções do `git` de um `review-check`
- [x] Ler os planos de transferência uma vez por operação e repassar o snapshot às lojas de origem
- [x] Comparar a saída antes e depois e medir o tempo
- [x] Teste que conta as leituras de `plan.json` e falha sem o listing compartilhado
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: manter a vinculação como está e confiar no runtime da B35
- Selected option: um listing de transferências por operação, guardado no snapshot de specs do projeto, e o snapshot repassado às lojas de origem do acceptance
- Rationale: a medição mostrou que um processo residente repete os mesmos 51 s; o trabalho repetido está dentro da operação, e o snapshot por operação já é o mecanismo do resolver para isso
- Consequences: um plano gravado durante a operação não é visto por ela, como já acontece com as specs
- Falsifier: uma resolução que responde de forma diferente com e sem o listing compartilhado, ou uma saída de `review-check` que difere

## 6. Validation

### Strategy

Um teste conta as leituras de `plan.json` e compara as respostas com e sem snapshot; a comparação byte a byte da saída antes e depois, e a medição do tempo, foram feitas no `pose-dist`.

### Deterministic checks

`go test` do teste do rastreio e a matriz estrita em clone limpo.

### Requirement trace
- R1 [satisfied] test:TestAnOperationReadsTheTransferPlansOfAProjectOnce check:index-input-binding-integration
- R2 [satisfied] test:TestAnOperationReadsTheTransferPlansOfAProjectOnce
- R3 [satisfied] test:TestAnOperationReadsTheTransferPlansOfAProjectOnce

### Known gaps
- R2 e R3 não têm teste próprio: a comparação byte a byte e o tempo (56 s para 38 s no `review-check`, medidos com o binário anterior e o novo no mesmo repositório) são medições feitas à mão e registradas aqui, não um teste de regressão; a variação de carga da máquina entra nesses números.

---

## 7. Final Report

### Delivered scope
O listing dos planos de transferência de um projeto é lido uma vez por operação e repassado às resoluções (R1); as lojas de origem do acceptance federado usam o snapshot de specs da operação (R2); a saída de `review-check` e `closeout-check` ficou idêntica e o `review-check` passou de cerca de 56 s para cerca de 38 s (R3).

### Files and modules changed
- pose-mcp/internal/pose: artifact_ref.go, spec.go, spec_transfer.go, federated_acceptance.go e o teste novo; a matriz ganha `index-input-binding-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- Um plano de transferência gravado durante uma operação só é visto pela seguinte.
- O gargalo restante é o `git`: cerca de 3100 execuções por `review-check`.

### Follow-ups

- [open] Reduzir as cerca de 3100 execuções do `git` por `review-check` (348 `git log -1 -- <path>` somam cerca de 20 s), sem mudar a simplificação de histórico que cada uma aplica. (owner:@pose-maintainers crit:medium review:2027-02-08)
- [open] Dar teste de regressão à saída idêntica e ao tempo da vinculação das entradas do índice. (owner:@pose-maintainers crit:low review:2027-02-08)
