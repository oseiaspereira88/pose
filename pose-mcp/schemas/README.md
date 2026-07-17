# pose-mcp — Response Schemas

Schemas versionados das respostas estruturadas (`structuredContent`) das tools
do `pose-mcp`, conforme a política da
[ADR-014](../../decisions/ADR-014-schema-evolution-e-serializacao.md):

- **Versionamento por diretório** (`v1/`, `v2/`, …). Dentro de uma versão a
  evolução é **additive-only**: nunca remover nem repropositar campo; campo
  novo é sempre opcional. Breaking change ⇒ novo diretório de versão + plano
  de migração.
- `additionalProperties: true` por design — consumidores devem tolerar campos
  novos (forward compatibility).
- O teste `internal/pose/schema_test.go` guarda o **drift** entre os structs
  Go e estes schemas: campo serializado que não conste do schema quebra o CI.

## Cobertura

| Schema | Tools |
|---|---|
| `v1/spec.schema.json` | `pose_get_spec`; itens de `pose_list_specs` (envelope `{specs: [...], count}`) |
| `v1/artifact.schema.json` | `pose_get_workflow`, `pose_get_rules`; itens dos envelopes de listagem (`{workflows|rules: [...], count}`) |
| `v1/gate-result.schema.json` | `pose_check`, `pose_lint_spec` |

## Pass-through (contrato da CLI)

`pose_suggest` e `pose_get_followups` repassam o JSON emitido pela própria CLI
(`./pose suggest --json`, `./pose followups --json`). **O dono desses contratos
é a CLI do POSE** (fonte de verdade determinística — ADR-003: adapter, não
fork); o adapter não os redefine nem os re-schematiza.
