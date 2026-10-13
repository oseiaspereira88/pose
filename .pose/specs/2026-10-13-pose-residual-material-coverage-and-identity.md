---
slug: pose-residual-material-coverage-and-identity
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-source-coverage-integrity, pose-debt-marker-lexical-precision, pose-covered-followup-anchor-verification
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:residual-material-coverage-and-identity
---

# Spec: Distinguir cobertura material, associação e identidade rastreada

## 1. Intent

### Goal

Impedir que uma menção a arquivo cubra automaticamente todos os markers e manter vínculos de residual quando sua apresentação muda.

### Business value

B22 / F05-02 e F05-04. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de residual](../reports/2026-10-12-pose-front-05-residual-work.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Menção a arquivo/componente deve produzir no máximo associação candidata; coverage efetiva deve exigir vínculo material verificável à ocorrência e ao compromisso pertinente. Duas ocorrências no mesmo arquivo podem ter cobertura distinta; draft ou existência de spec não provam entrega.
- R2: Matching deve delimitar tokens/paths/IDs e recorte: prefixo de path, comentário de exemplo, citação negativa ou incidental não devem afirmar compromisso. Se semântica não for decidível deterministicamente, apresentar candidato com motivo/limitação, sem promover a cobertura.
- R3: O scanner deve distinguir ocorrências idênticas no mesmo arquivo e vários markers na mesma linha com IDs sem colisão no snapshot; posição de apresentação não deve ser identidade rastreada. Não perder a precisão lexical ou a coverage de B06.
- R4: Follow-up rastreado deve preservar identidade e vínculos quando outra bullet é inserida/excluída, reordenada ou rewrapped, em layouts flat/folder. Texto idêntico em ocorrências distintas não deve mesclar intenções; ordinal continua localização no snapshot, nunca chave durável.
- R5: Definir prospectivamente identidade de ocorrência, identidade rastreada e localização. Avaliar ID explícito somente para itens rastreados, com unicidade/namespacing e diagnóstico de duplicação/ambiguidade; não criar arquivo por TODO ou novo ledger universal.
- R6: Renomeação, mudança de layout ou edição material deve declarar aliases/proveniência e limites de continuidade. Alias deve resolver univocamente, detectar ciclo/conflito e ter custo limitado; sem mapeamento confiável, apresentar candidato de reconciliação, sem retarget automático.
- R7: Legado deve continuar consultável; IDs históricos não devem ser reinterpretados silenciosamente como identidade durável. Versionar projeções alteradas e expor aliases/limitations; resolver referências antigas somente com evidência adequada. Consultas não escrevem IDs, aliases ou migrations.
- R8: CLI, MCP, assessment, obligations e Attention devem consumir a mesma distinção de identidade/associação/cobertura no mesmo snapshot e preservar diagnósticos de B06. Qualificar P02/P03/P07 e controles positivos sem mudar dispositions ou declarar itens resolvidos.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-8-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-residual-material-coverage-and-identity.md
- created: .pose/starts/pose-residual-material-coverage-and-identity.json
- created: .pose/changelogs/unreleased/pose-residual-material-coverage-and-identity.md
- modified: pose-mcp/internal/pose/techdebt.go
- modified: pose-mcp/internal/pose/followups_parse.go
- modified: pose-mcp/internal/pose/followup_candidates.go
- modified: pose-mcp/internal/pose/obligation_sources.go
- modified: pose-mcp/internal/pose/techdebt_coverage_test.go
- created: pose-mcp/internal/pose/residual_identity_test.go
- created: pose-mcp/internal/cli/residual_identity_parity_test.go
- created: pose-mcp/internal/mcpserver/residual_identity_parity_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:residual-material-coverage-and-identity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/techdebt.go

### Approach and execution gates

B06 deve fornecer perdas/coverage antes deste delta. Estender scanner e parser compartilhados; inventariar os consumidores reais e limites de compatibilidade antes de start. Separar associação candidata de compromisso adequado e identidade de locator. Decidir sintaxe/versionamento de ID rastreado e aliases por ADR se houver contrato estrutural novo. Não ampliar natureza/compromisso/efeito de B24 nem métricas/triagem de B25/B26. Não alegar estabilidade de duas ocorrências indistinguíveis sem âncora confiável.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Projetar cobertura somente a partir de vínculo material e preservar identidade rastreada com continuidade explícita.
- Rationale: Associação e localização não provam intenção, compromisso ou resolução; preservar incerteza evita certificar ausência de trabalho sem evidência.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P02: spec menciona arquivo apenas como contexto | Associação candidata; nenhuma cobertura automática; markers no mesmo arquivo independentes. |
| V2 — Prefixo, menção negativa/incidental e compromisso explícito delimitado | Negativos não cobrem; controle positivo cobre somente a ocorrência pertinente. |
| V3 — P03: TODOs idênticos e markers na mesma linha | IDs de ocorrência distintos; duplicação de ID rastreado diagnosticada. |
| V4 — P07: inserir/remover/reordenar/rewrap bullets flat/folder | Vínculos rastreados preservados; localização pode mudar; duplicatas não mescladas. |
| V5 — Rename/migração, aliases cíclicos/conflitantes e edição material | Continuidade explícita quando demonstrada, caso ambíguo diagnosticado e custo limitado. |
| V6 — Legado, fonte incompleta e consultas em CLI/MCP/Attention | Paridade no snapshot; zero writes/auto-dispositions; uncertainty preservada. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V2 |
| R2 | V2 |
| R3 | V3, V6 |
| R4 | V4 |
| R5 | V3, V4 |
| R6 | V5 |
| R7 | V5, V6 |
| R8 | V1–V6 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 8. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-8-authoring.md).

### Residual risks

Formato de identidade ainda requer decisão prospectiva; hash de snippet/ordinal atual não satisfaz todos os invariantes. Aliases sem prova podem ligar intenções diferentes. B24/B25 continuam responsáveis por natureza/compromisso/efeito e resposta a condições, sem duplicação neste delta.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
