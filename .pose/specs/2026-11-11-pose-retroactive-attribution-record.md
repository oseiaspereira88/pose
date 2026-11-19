---
slug: pose-retroactive-attribution-record
status: in-progress
created_at: 2026-11-11
completed_at:
supersedes:
depends_on: 
remediates:
priority: 2
components: pose-mcp
task_type: feature
surface: minimal
changelog: none
---

# Spec: Registro de atribuição retroativa para histórico selado

## 1. Intent

### Goal

O motor aceita um registro assinado e versionado que reatribui caminhos de commits já publicados (compartilhados por vários trailers ou sem spec declarante) a uma spec, sem editar a seção Artifacts selada das specs fechadas.

### Business value

O `pose check --strict` do pose-dist falha com 19 erros "undeclared governed path" desde o fechamento do roadmap `pose-shared-runtime` (2026-11-10): o commit de congelamento `80b49143` cita seis specs e altera `resident_git_memo.go` e seu teste, que nenhuma spec fechada declara; `57e20957` (um trailer) alterou arquivos declarados por outras duas; `pose-project-agnostic-assessment-engines` tem dois casos próprios. Uma regra de motor para caminhos de commit compartilhado reduziu os erros de 19 para 14 mas não chega a verde (testado e descartado em 2026-11-11), e a seção Artifacts das specs fechadas é selada. O gap é de governança: o histórico publicado não pode ser reescrito, então a correção é um registro explícito, auditável e que não afrouxe o gate para commits futuros.

### Constraints

Preserve committed Artifacts sections and sealed provenance. Accept only a signature from a human principal explicitly registered as a maintainer.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: ADR primeiro: forma do registro (arquivo, assinatura, quem pode escrever), escopo (caminho + commit + spec declarante) e por que não uma regra implícita.
- R2: `pose index` e `pose artifact-check` leem o registro e deixam de reportar `undeclared` para o par (commit, caminho) reatribuído, com a razão e o autor visíveis no relatório.
- R3: O registro só cobre commits já existentes no momento da escrita; um commit novo com o mesmo problema continua reprovando.
- R4: Um registro que referencia commit ou spec inexistentes, ou um caminho que o commit não alterou, é recusado.
- R5: Os 19 erros do pose-dist são resolvidos por registros, e o `pose check --strict` do pose-dist passa.

## 3. Technical Plan

### Artifacts

- created: pose-mcp/internal/cli/artifact_attributions.go
- created: pose-mcp/internal/cli/artifact_attributions_test.go
- created: pose-mcp/internal/pose/artifact_attribution_index_test.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/pose/index_generation.go

### Tasks

- Verify the signed manifest and historical witnesses.
- Apply dispositions in the live delivery graph and scope checks.
- Register the production records following maintainer signature.
- Validate the full repository and request independent review.

## 5. Decisions

See [the attribution ADR](../adr/2026-11-18-bind-retroactive-delivery-attribution-to-signed-git-ranges.md).

## 6. Validation

| Scenario | Required command | Expected evidence |
| --- | --- | --- |
| Verified record and unchanged provenance | `go -C pose-mcp test ./internal/cli -run '^TestRetroactiveAttribution' -count=1` | Only the exact undeclared finding is replaced with the signer and reason; the sealed provenance digest is unchanged. |
| Unknown path, missing or wrong key, bad signature, mismatched project or baseline, new range | Same command | Fail closed or leave the new finding in force as appropriate. |
| Index freshness | `go -C pose-mcp test ./internal/pose -run '^TestRetroactiveAttributionIndexBinding' -count=1` | Manifest appearance, changes and removal affect the binding; absent manifests preserve compatibility. |
| Repository integration | `pose index && pose index --verify && pose check --strict` | The 19 historical paths carry verified attribution records; the strict gate passes. |
| Affected module | `pose validate --strict --module pose-mcp --report` | Registered checks pass. |

---

## 7. Final Report

### Delivered scope
Implementation and validation are in progress.
