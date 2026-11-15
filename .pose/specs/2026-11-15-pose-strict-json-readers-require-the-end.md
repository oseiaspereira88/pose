---
slug: pose-strict-json-readers-require-the-end
status: in-progress
created_at: 2026-11-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates: spec:pose-hooks-delegate-to-the-instance-runner@defect-fix
priority:            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: pose-mcp
task_type: bugfix
surface:             # optional: minimal | standard | full (default) — minimal drops Tasks and derivable report sections; `pose specs facts <slug>` derives them
delivers: capability:strict-json-readers-require-the-end
---

# Spec: pose-strict-json-readers-require-the-end

## 1. Intent

### Goal
Fazer todo leitor estrito de JSON do motor recusar conteúdo depois do valor que lê: um objeto válido seguido de lixo ou de um segundo valor é malformado, não lido como sua primeira parte.

### Business value
A revisão da spec `pose-hooks-delegate-to-the-instance-runner` (rodada 1, F1) achou que a policy de hooks, lida com `DisallowUnknownFields`, aceitava um objeto válido seguido de lixo e rodava o executor declarado. A mesma forma estava em outros dois leitores: a evidência de `pose release record` e o envelope de `pose action fulfil`. Os dois aceitavam conteúdo depois do objeto, e a evidência chegava a ser gravada como evento da release. Um leitor que se declara estrito e lê só o começo do arquivo aceita entradas que um leitor correto recusaria, e cada leitor novo podia repetir o erro, porque nada media a propriedade.

### Constraints
- Leitores que já exigem o fim (um segundo `Decode` comparado com `io.EOF`, `Token`, ou um `json.Unmarshal` dos mesmos bytes) não mudam.

### Non-goals
- Não tornar estrito um leitor deliberadamente permissivo (a policy de revisão aceita campos de motores mais novos).

---

## 2. Requirements

### Functional
- R1: The evidence of `pose release record`, the envelope of `pose action fulfil` and the hooks policy shall be refused when anything but white space follows their JSON value, and nothing shall be recorded or run.
- R2: Every engine function that refuses unknown JSON fields shall do so through `DecodeStrictJSON`, or be named in the guard's reviewed list with how it refuses content after the value.

### Non-functional
- Nenhuma.

### Security
- Fecha uma aceitação silenciosa de entrada malformada num leitor de evidência de release e num de envelope de ação.

### Compatibility
- Uma evidência, um envelope ou uma policy válidos, com ou sem espaço em volta, são lidos como antes.

---

## 3. Technical Plan

### Affected areas
- `pose/strict_json.go`: `DecodeStrictJSON`, um valor e nada depois.
- `cli/release_lifecycle.go`, `cli/review_request.go`, `cli/maintenance.go`: os três leitores passam a usá-lo.
- `pose/strict_json_test.go`: o teste do helper e o guard que percorre o fonte.

### Artifacts
- created: pose-mcp/internal/pose/strict_json.go
- created: pose-mcp/internal/pose/strict_json_test.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/review_request.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/delegated_review_request_test.go
- modified: pose-mcp/internal/cli/release_review_regression_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:strict-json-readers-require-the-end module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Composto em produção por `pose release record`, `pose action fulfil` e pelos git hooks do motor. Check permanente `strict-json-readers-require-the-end-integration`, evidenceClass `integration`, required: `go test ./internal/pose ./internal/cli -run 'TestDecodeStrictJSON|TestStrictJSONReadersRequireTheEnd|TestReviewRequestIsFulfilledOverTheCLI|TestF05RecordRefusesATagWhoseTreeIsNotIntact|TestAnInvalidHookRunnerIsRefused' -count=1`.

### API/contract changes
- Nenhuma: entradas válidas são lidas como antes.

### Data/storage changes
- Nenhuma.

### Technical risks
- Um arquivo que alguém escreveu com conteúdo extra depois do objeto passa a ser recusado; é o comportamento pretendido, e a mensagem diz o motivo.

---

## 4. Tasks

### Implementation
- [x] `DecodeStrictJSON` e os três leitores (R1).
- [x] Guard pelo predicado real: toda função com `DisallowUnknownFields` fora do helper está na lista revisada, com o mecanismo do fim (R2).

### Validation
- [x] Regressões pela CLI para a evidência e o envelope, e o teste da policy de hooks.
- [x] Prova de que o guard aponta os dois leitores originais sem a correção.

---

## 5. Decisions

### Decision D1
- Date: 2026-11-15
- Context: sete leitores estritos já exigiam o fim de quatro formas diferentes, e três não exigiam.
- Options considered: (a) corrigir só os três; (b) um helper comum e um guard que mede a propriedade no fonte.
- Decision: (b).
- Rationale: (a) deixava o próximo leitor livre para repetir o erro; o guard mede o predicado da regra (a mesma função que recusa campos desconhecidos exige o fim) e aceita as formas corretas que já existem, sem reescrevê-las.
- Consequences: um leitor novo com `DisallowUnknownFields` usa o helper, confere o fim, ou declara no guard por que não precisa.

---

### Decision D2
- Date: 2026-11-15
- Context: as rodadas 1 e 2 da revisão acharam, cada uma, uma forma sintática que o predicado aceitava sem conferir o fim (`Decode(new(T))` e `More()`; depois uma comparação com `io.EOF` no primeiro decode e um `errors.Is` descartado).
- Options considered: (a) refinar a análise sintática do fim; (b) exigir o helper e listar, revisados, os leitores que conferem o fim de outro modo.
- Decision: (b).
- Rationale: (a) aproxima uma propriedade semântica e cada refinamento abre a próxima variante, como o extrator de Markdown da spec de política de revisão; (b) é um predicado que o guard mede exatamente (a função está fora do helper e da lista), e cada exceção tem uma justificativa que um revisor confere no código.
- Consequences: um leitor estrito novo usa `DecodeStrictJSON` ou entra na lista com o mecanismo do fim; a lista tem 16 entradas, uma delas o `dispatch` do MCP, cujos argumentos o transporte já leu como um valor.

---

## 6. Validation

### Strategy
Regressão pela CLI de cada leitor corrigido, com lixo e com um segundo valor depois do objeto; teste do helper; guard sobre o fonte do motor, provado contra a árvore com o defeito.

### Deterministic checks

#### Test
- Command: `go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold -count=1`
- Scope: pose-mcp
- Expected: verde

### Execution log
- Date: 2026-11-15
- Environment: Linux, Go 1.26.
- Notes: sem a correção, `pose action fulfil --envelope` aceitou um envelope seguido de lixo (`action.fulfil.preview=true`) e `pose release record` gravou a evidência seguida de lixo (`Recorded tagged for v1.0.0.`); o guard apontou `cmdReleaseRecord` e `cmdActionFulfil`. Com a correção, os três passam. A primeira versão do guard, que contava `io.EOF` por arquivo, deixou de ver os leitores que conferem o fim com um segundo `Decode`; o predicado foi corrigido para a função e para as formas que já existiam.

### Results summary
- Successes: as regressões reprovam sem a correção e passam com ela; o guard aponta os dois leitores originais e aceita os demais.
- Failures: nenhuma.
- Warnings: nenhum.

### Requirement trace
- R1 [satisfied] test:TestReviewRequestIsFulfilledOverTheCLI test:TestF05RecordRefusesATagWhoseTreeIsNotIntact test:TestAnInvalidHookRunnerIsRefused test:TestDecodeStrictJSONReadsOneValueAndNothingElse
- R2 [satisfied] test:TestStrictJSONReadersRequireTheEnd check:strict-json-readers-require-the-end-integration

### Known gaps
- A lista revisada descreve o mecanismo de cada leitor; o guard prova que nenhum leitor estrito fica fora dela, não que cada descrição é verdadeira, o que cabe à revisão de quem a altera.

---

## 7. Final Report

### Delivered scope

Não concluída.

### Residual risks

### Follow-ups

### Review

Rodada 1 (run rrn-964929504dce2647, attestation rva-0757736fb2ddb7e1, changes-requested): 1 bloqueador, tratado. F1 o predicado do guard aceitava formas que não conferem o fim: `Decode(new(T))` sozinho e `More()`, que devolve falso diante de um `}` ou `]` sobrando. O predicado passou a exigir uma condição de fim de fato conferida (comparação com `io.EOF` por `==`, `!=` ou `errors.Is`, `DecodeStrictJSON`, ou `json.Unmarshal` da entrada), e fixtures positivas e negativas o provam; as negativas reprovam com os atalhos de volta.

Rodada 2 (run rrn-d6a9f64a55d3094f, attestation rva-717f9999c83d0013, changes-requested): F1 parcial e 1 bloqueador, tratado. N1 o predicado aceitava qualquer comparação com `io.EOF`, inclusive no erro do primeiro decode, e um `errors.Is` cujo resultado era descartado. Pela decisão D2, o guard deixou de julgar a forma da conferência: toda função com `DisallowUnknownFields` fora de `DecodeStrictJSON` precisa estar na lista revisada, que diz como cada uma confere o fim (16 entradas, conferidas no código); o teste de fixtures sintáticas saiu. Sem a correção dos leitores, o guard aponta `cmdReleaseRecord` e `cmdActionFulfil`.

