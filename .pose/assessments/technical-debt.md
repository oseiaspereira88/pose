# Technical Debt Assessment: pose-dist

> **Gerado por**: POSE Technical Debt Engine (`pose assess tech-debt`)
> **Data de Avaliação**: 2026-11-12T08:27:24Z
> **Baseline Commit**: 30cd2b1749d1

## 1. Resumo Executivo

- **Total de Marcadores**: 2
- **TODOs**: 1 | **FIXMEs**: 0 | **HACKs**: 0 | **Panics**: 1 | **Stubs**: 0
- **Cobertos por Backlog Ativo**: 1
- **Não Cobertos**: 1
- **Recomendações**: 1 Follow-ups | 0 Specs | 0 Roadmaps
- **Escopo do scan**: 355 arquivo(s) de código lidos; **INCOMPLETO** — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown). Zero marcadores não prova ausência de dívida nos arquivos não lidos.
- **Excluídos (generated-header)**: 1 arquivo(s) — files that declare themselves generated are not scanned
- **Excluídos (skipped-directory)**: 40 arquivo(s) — directories that hold dependencies, build output or tool state are not scanned

## 2. Ocorrências

| ID | Marcador | Componente | Arquivo e Linha | Cobertura | Evidência | Recomendação |
|---|---|---|---|---|---|---|
| DEBT-057f3bf2 | `PANIC` | `pose-mcp` | [bounded_cache.go:334](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/bounded_cache.go#L334) | `uncovered` | `` | `create_followup` |
| DEBT-8dcc4523 | `TODO` | `pose-mcp` | [techdebt.go:329](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L329) | `covered_by_spec` | `spec:pose-residual-material-coverage-and-identity` | `none` |

## 3. Política de Ação

- Registre dívida local não coberta como follow-up.
- Abra uma spec quando um componente concentrar dívida recorrente.
- Estenda o roadmap quando a dívida atravessar vários arquivos e exigir coordenação sistêmica.
- Não crie recomendação nova quando uma spec, roadmap ou follow-up ativo já cobrir a ocorrência.
