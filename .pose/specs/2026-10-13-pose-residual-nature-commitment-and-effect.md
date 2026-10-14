---
slug: pose-residual-nature-commitment-and-effect
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-residual-material-coverage-and-identity, pose-followup-metadata-and-routing-integrity, pose-obligation-contract, pose-followup-reconciliation-candidates
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:residual-nature-commitment-and-effect
---

# Spec: Separar natureza, compromisso e efeito do trabalho residual

## 1. Intent

### Goal

Projetar o que falta, seu compromisso autoritativo e efeito por fase sem chamar todo open de dívida ou inventar novos blockers pelo rótulo.

### Business value

B24 / D07 e F05-07. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de residual](../reports/2026-10-12-pose-front-05-residual-work.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Projetar naturezas distinguíveis de dívida técnica, defeito, compromisso incompleto, lacuna de evidência, adoção/operação e melhoria opcional, com não classificado/unknown quando não há contexto. Definir evidência e fonte adequadas para cada classificação; TODO/open/status/severity/path citado não devem confirmar natureza automaticamente.
- R2: Manter natureza, compromisso, efeito por fase/escopo, criticidade/urgência, dono de triagem/implementação, SLA de triagem e destino como dimensões independentes. Não ampliar lifecycle da spec ou converter prazo de triagem em prazo de implementação; fonte sem metadata continua legada/não classificada, sem classificação retrospectiva fabricada.
- R3: Compromisso deve referenciar requirement/escopo/decisão autoritativa e parcela pertinente, usando IDs/aliases/adequação de B22/B23. Encaminhamento spawned/covered/duplicate ou destino done isolados não demonstram satisfação nem apagam intenção vigente; cadeia origem/destino deve conservar vínculo e resultado sem duplicar implementação devida.
- R4: Efeito bloqueante deve derivar de regra/policy/compromisso vigente, autoridade e escopo explícitos, nunca apenas de categoria, high severity, review vencido ou advisory. Item opcional/adiado não cria gate sozinho, enquanto obrigação vigente não vira opcional só por renomeação/classificação. Unknown conserva limitações e efeitos conhecidos, sem inferir ausência de restrição ou blocker global.
- R5: Representar fratura de release como relação com uma entrega/consumidor específica, não natureza exclusiva ou promessa nova de cut. B37 continua contrato de suficiência/escopo; esta projeção expõe referências/limites e não prepara release, aceita risco ou decide imediatamente quando implementar.
- R6: Estender fontes/projeções canônicas existentes (follow-ups/findings/requirements/obligations) com identidade estável e proveniência de classificação. Candidato editorial/heurístico deve permanecer candidato; classificação factual precisa de fundamento observado/declaração admitida. Não criar backlog paralelo, arquivo por TODO, spec por pendência ou estado resolved próprio no agregador.
- R7: Reusar ADR de obligations e preservar schema/versionamento/compatibilidade das fontes e consumers. Avaliar metadata opcional na fonte versus refs estruturadas e registrar extensão estrutural necessária antes do código; não fixar sintaxe ou versão concreta por inferência. Queries são read-only e não escrevem classificação, dispositions, risco aceito ou migração de legado automaticamente.
- R8: CLI/MCP/Attention/assessment devem consumir o mesmo recorte/identidade, natureza/compromisso/efeito e coverage pertinente. Expor agrupamentos por intenção, fonte/destino/limites e contagens sem misturar candidatos, itens devidos e ocorrências; sem score agregado ou métricas de produtividade por baixar residual.
- R9: Qualificar matriz das naturezas e controles de legado, compromisso obrigatório low, dívida high adiada, marker opcional, destino done inadequado, links órfãos/cíclicos e fonte ilegível. Demonstrar ausência de blockers por rótulo e preservação de restrições autoritativas, sem triagem/apply/aceitação de risco de B25 ou condições/métricas de B26.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-11-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-residual-nature-commitment-and-effect.md
- created: .pose/starts/pose-residual-nature-commitment-and-effect.json
- created: .pose/changelogs/unreleased/pose-residual-nature-commitment-and-effect.md
- modified: pose-mcp/internal/pose/obligation.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/obligation_sources.go
- modified: pose-mcp/internal/pose/followups_parse.go
- modified: pose-mcp/internal/pose/followup_candidates.go
- modified: pose-mcp/internal/cli/state_attention.go
- modified: pose-mcp/schemas/v1/obligation.schema.json
- created: pose-mcp/internal/pose/residual_nature_commitment_test.go
- created: pose-mcp/internal/cli/residual_dimensions_test.go
- created: pose-mcp/internal/mcpserver/residual_dimensions_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:residual-nature-commitment-and-effect module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/obligation_projection.go

### Approach and execution gates

Executar depois de B22/B23. Reusar ADR 2026-10-04-obligations-are-projected-action-requests-are-persisted: read model, fontes autoritativas, dimensões separadas e gates por efeito/escopo. Decidir sintaxe/schema/compatibilidade da extensão e registrá-la em ADR prospectivo apenas onde a decisão estrutural ainda não estiver coberta; o path schema v1 é ponto de inventário atual, não autorização para redefinir sua semântica silenciosamente (versionar/criar novo path se necessário antes de start). Manter taxonomia de fonte/categoria antiga como legacy quando cabível e acrescentar dimensões de projeção com limites explícitos. Classificação não resolve compromisso; B25 adjudica triagem/risco, B26 mede condições e B37 integra release.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Projetar natureza, compromisso e efeito separadamente, com fundamento e fontes canônicas preservados.
- Rationale: Categoria, apresentação e estado de processo não provam autorização ou satisfação; preservar fonte e efeitos pertinentes evita bloqueios e confirmações artificiais.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Cada natureza com fundamento pertinente e classificação sem fonte | Naturezas distinguíveis; candidato/unknown não promovidos a fato. |
| V2 — Legado/open/TODO/panic e metadata ausente | Não classificado/marker detectado; lifecycle e autoridade preservados. |
| V3 — Obrigação low vigente versus dívida high adiada/advisory | Gate deriva de compromisso/efeito, nunca de rótulo/severity/SLA. |
| V4 — Spawned/covered/duplicate, done adequado/inadequado e aliases | Vínculo/adequação preservados; sem duplicar devida ou auto-disposition. |
| V5 — Release específica, lacuna de evidência e consumidor de adoção | Relação e limites explícitos; sem preparar cut ou declarar produto defeituoso por falta de prova. |
| V6 — Fontes ilegíveis, refs órfãs/cíclicas e coverage incompleta | Unknown localizado com restrições conhecidas; sem zero completo ou blocker universal. |
| V7 — CLI/MCP/Attention/assessment e fonte canônica intacta | Paridade de dimensões/contagens/IDs; zero writes ou backlog paralelo. |
| V8 — Compatibilidade/schema/aliases e queries repetidas | Legado útil, projeção versionada e estável; nenhuma classificação retrospectiva automática. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V2 |
| R2 | V2, V3 |
| R3 | V4, V6 |
| R4 | V3, V6 |
| R5 | V5 |
| R6 | V1, V4, V7 |
| R7 | V2, V7, V8 |
| R8 | V5–V8 |
| R9 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 11. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-11-authoring.md).

### Residual risks

Extensão de metadata e schema exige decisão prospectiva e classificação de compatibilidade. Fonte ilegível não permite inferir compromisso ou satisfação. Categoria de residual não substitui obligations/effects, e sem decisão/autoridade adequada não há aceitação de risco ou retirada de compromisso.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
