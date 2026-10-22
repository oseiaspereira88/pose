# Technical Debt Assessment: pose-pending-b38

> **Gerado por**: POSE Technical Debt Engine (`pose assess tech-debt`)
> **Data de Avaliação**: 2026-10-22T06:02:41Z
> **Baseline Commit**: e27f1f75593c

## 1. Resumo Executivo

- **Total de Marcadores**: 1
- **TODOs**: 1 | **FIXMEs**: 0 | **HACKs**: 0 | **Panics**: 0 | **Stubs**: 0
- **Cobertos por Backlog Ativo**: 1
- **Não Cobertos**: 0
- **Recomendações**: 0 Follow-ups | 0 Specs | 0 Roadmaps
- **Escopo do scan**: 269 arquivo(s) de código lidos; **INCOMPLETO** — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown). Zero marcadores não prova ausência de dívida nos arquivos não lidos.
- **Excluídos (generated-header)**: 1 arquivo(s) — files that declare themselves generated are not scanned
- **Excluídos (skipped-directory)**: 36 arquivo(s) — directories that hold dependencies, build output or tool state are not scanned

## 2. Ocorrências

| ID | Marcador | Componente | Arquivo e Linha | Cobertura | Evidência | Recomendação |
|---|---|---|---|---|---|---|
| DEBT-8dcc4523 | `TODO` | `pose-mcp` | [techdebt.go:329](file:///tmp/pose-pending-b38/pose-mcp/internal/pose/techdebt.go#L329) | `covered_by_spec` | `spec:pose-residual-material-coverage-and-identity` | `none` |

## 3. Política de Ação

- Registre dívida local não coberta como follow-up.
- Abra uma spec quando um componente concentrar dívida recorrente.
- Estenda o roadmap quando a dívida atravessar vários arquivos e exigir coordenação sistêmica.
- Não crie recomendação nova quando uma spec, roadmap ou follow-up ativo já cobrir a ocorrência.
