# GraphForge Cross-Component Integration & Gap Analysis Report

> **Gerado por**: POSE Integration Engine (`pose assess integrate`)
> **Data de Avaliação**: 2026-08-02T17:09:55Z
> **Baseline Commit**: 3eaa7db815cf

---

## 1. Resumo Executivo das Integrações

- **Total de Contratos Mapeados**: 11
- **Contratos Ativos**: 10
- **Gaps de Integração Identificados**: 8

---

## 2. Matriz de Contratos Inter-Componentes

| Nome do Contrato | Protocolo | Provedor | Consumidor | Status |
|---|---|---|---|---|
| GraphMutationEvent (Protobuf) | `protobuf` | `ast-engine, indexers, collector, enricher, planner` | `graph-core` | `active` |
| Kafka Mutation Topics (ast/git/infra/test/runtime/semantic/plan.mutations) | `kafka` | `ast-engine, indexers, collector, enricher, planner` | `graph-core` | `active` |
| NodeKinds (74) & EdgeKinds (87) | `library` | `graphforge/pkg/models` | `ast-engine, graph-core, mcp-server, indexers, collector, enricher, planner` | `active` |
| MCP Protocol (52 Tools) | `mcp` | `graphforge/mcp-server` | `cli, graphforge-web, ai-agents` | `active` |
| Curation HTTP API (Port 8090) | `rest` | `graphforge/semantic-enricher` | `graphforge-web` | `active` |
| Planning HTTP API (Port 8095) | `rest` | `graphforge/planning-engine` | `cli, graphforge-web` | `degraded` |
| Conductor Run Reporter & Fleet State (38 MCP Tools) | `mcp/http` | `conductor` | `harne8-pose-mcp, harness, portal` | `active` |
| Harness Execution Engine Sandbox (gRPC/REST) | `grpc` | `harness` | `conductor` | `active` |
| Portal Session Bridge & Edge Gateway | `rest/jwt` | `workers/app` | `portal, conductor, site` | `active` |
| OPA Policy Enforcement Engine (egress-policy.json) | `opa` | `pose-dist/mcp-enforce` | `harne8-pose-mcp` | `active` |
| Harne8 Open API & Protobuf Shared Contracts | `openapi/protobuf` | `contracts` | `workers/app, conductor, harness, site` | `active` |


---

## 3. Detalhamento dos Gaps de Integração Identificados

### ⚠️ [GAP-01] pkg/componentshit Subaproveitado no mcp-server
- **Severidade**: medium
- **Provedor**: `graphforge/pkg/componentshit`
- **Consumidor**: `graphforge/mcp-server`
- **Descrição**: Nenhuma tool MCP exporta a travessia transitiva de impacto de capabilities POSE. Apenas o CLI Go possui o comando 'components-hit'.

### ⚠️ [GAP-02] runtime-collector Anomalias/Impacto vs. Dashboard no graphforge-web
- **Severidade**: high
- **Provedor**: `graphforge/runtime-collector`
- **Consumidor**: `graphforge/graphforge-web`
- **Descrição**: Os dashboards de saude utilizam mocks/simulacoes no Zustand store em vez de assinar em tempo real a stream de anomalias/latencia OTel.

### ⚠️ [GAP-03] planning-engine Shadow Graph (SAGA & Approval) vs. cli e graphforge-web
- **Severidade**: high
- **Provedor**: `graphforge/planning-engine`
- **Consumidor**: `graphforge/cli, graphforge/graphforge-web`
- **Descrição**: A CLI nao possui subcomandos para aprovacao de planos (Lead Signoff / Emergency) ou acionamento de rollback SAGA. O Web nao renderiza o Shadow Graph.

### ⚠️ [GAP-04] Desconexao do semantic-enricher em Ambientes POSE Remotos/Multi-Projeto
- **Severidade**: medium
- **Provedor**: `harne8-pose-mcp / .pose`
- **Consumidor**: `graphforge/semantic-enricher`
- **Descrição**: O semantic-enricher le arquivos .pose/ do disco local via os.ReadFile em vez de consumir a API do pose-mcp, impedindo uso distribuido.

### ⚠️ [GAP-05] Bytecode JVM no ast-engine vs. Mapeamento de Atributos no graph-core
- **Severidade**: low
- **Provedor**: `graphforge/ast-engine (bytecode parser)`
- **Consumidor**: `graphforge/graph-core`
- **Descrição**: O graph-core aceita nos com node_type=Class, mas as tabelas relacionais do Postgres nao possuem colunas para instrucoes bytecode JVM.

### ⚠️ [GAP-06] Conductor Tenancy Bridge vs. Harness Sandbox Execution Isolation
- **Severidade**: critical
- **Provedor**: `conductor`
- **Consumidor**: `harness`
- **Descrição**: O runner de sandbox do Harness no modo dev (docker-compose.dev.yaml) nao propaga o tenant-id isolado do Conductor em execucoes concorrentes de agentes.

### ⚠️ [GAP-07] Portal Session Bridge em workers/app vs. Autenticacao GraphForge Web
- **Severidade**: high
- **Provedor**: `workers/app (Cloudflare Worker)`
- **Consumidor**: `graphforge/graphforge-web`
- **Descrição**: O graphforge-web utiliza sessoes de streaming anonimas com Neo4j, enquanto o workers/app exige JWT Bearer tokens na borda, exigindo um header mapping no gateway.

### ⚠️ [GAP-08] mcp-enforce Sincronizacao de Politicas em Tempo Real
- **Severidade**: medium
- **Provedor**: `pose-dist/mcp-enforce`
- **Consumidor**: `harne8-pose-mcp`
- **Descrição**: As politicas OPA sao carregadas na inicializacao do processo, porem falta um file-watcher para recarregar alteracoes em .pose/policy/*.json em tempo real.
