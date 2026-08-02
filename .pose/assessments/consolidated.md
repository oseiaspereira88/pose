# Harne8 Platform Macro Assessment & Monorepo Consolidation

> **Gerado por**: POSE Discovery Engine (`pose assess discover`)
> **Data de Avaliação**: 2026-08-02T17:09:55Z
> **Baseline Commit**: 3eaa7db815cf

---

## 1. Resumo Executivo da Plataforma Harne8

- **Total de Componentes Auditados**: 2
- **Linhas de Código de Produção**: 27441
- **Linhas de Código de Testes**: 14955
- **Total Geral de Linhas de Código**: 42396
- **Total de Arquivos Auditados**: 200
- **Completude Dinâmica da Plataforma**: 33.2%
- **Dívidas Técnicas em Aberto**: 38 TODOs | 30 FIXMEs | 4 Panics | 27 Stubs
- **Especificações (Specs) em Aberto**: 3
- **Gaps de Integração Identificados**: 8

---

## 2. Inventário e Métricas dos Componentes Harne8

| # | Componente Slug | Caminho do Módulo | Linguagens | LOC Produção | LOC Testes | Arquivos | TODOs | Completude | Status |
|---|---|---|---|---|---|---|---|---|---|
| 01 | `mcp-enforce` | `mcp-enforce` | `go` | 1036 | 943 | 24 | 0 | 86% | `in_progress` |
| 02 | `pose-mcp` | `pose-mcp` | `go` | 26405 | 14012 | 176 | 38 | 16% | `needs_attention` |

---

## 3. Arquitetura dos Subsistemas Harne8

1. **Conductor & Harness Control Plane (`conductor`, `harness`)**:
   - Orquestração de frota de agentes de IA, acompanhamento de execuções de runs e execução de sandboxes com suporte a SAGAs.
2. **GraphForge Knowledge Subsystem (`graphforge/*`)**:
   - Compilação de código AST (Rust/Go), indexadores (Git/Infra/Test), correlation engine OTel, enricher semântico LLM, motor de planejamento shadow graph e interface tri-engine Canvas 2D/3D (React 19).
3. **Edge Gateway & Portal (`workers/app`, `site`)**:
   - Gateway Cloudflare Workers, autenticação de sessão Portal, distribuição de magic links e site oficial Harne8.
4. **Governança POSE & Enforce (`pose-dist/pose-mcp`, `pose-dist/mcp-enforce`, `contracts`)**:
   - Servidor nativo `harne8-pose-mcp`, motor de aplicação de política OPA `mcp-enforce` e esquemas compartilhados Protobuf/OpenAPI.
