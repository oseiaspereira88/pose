---
slug: pose-distributed-instructions-parity
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-adaptive-assessment-freshness, pose-v7-legacy-cleanup-plan
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:distributed-instructions-parity
---

# Spec: Tornar instruções distribuídas coerentes com as capacidades existentes

## 1. Intent

### Goal

Alinhar workflows/skills/scaffold/pt-BR, descoberta de regras e mensagens de migrate ao comportamento efetivamente disponível, preservando freshness e decisões de instalação.

### Business value

B28 / D11 e F07-06. Workflows ainda podem pedir discovery incondicional; suggest não resolve consistentemente aliases/disponibilidade; migrate fala em futura 7.0 apesar de apply indisponível. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise fonte](../reports/2026-10-12-pose-front-07-operational-ux.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Workflows de feature/closeout, skills, AGENTS/POSE e cópias EN/pt-BR/scaffold pertinentes devem usar a mesma regra de freshness entregue: discovery --if-stale nos pontos de execução previstos, sem rescan obrigatório por sessão ou fechamento quando inputs não mudaram.
- R2: Regressões devem cobrir arquivos efetivamente distribuídos/instalados, não só uma lista parcial de fontes root; inventariar mirrors/locales antes da edição e validar jornada gerada por install/update sem adotar policy incidentalmente.
- R3: Suggest deve normalizar aliases/domain/path segundo taxonomia compartilhada e preservar source da inferência; go/backend-go e outros aliases documentados devem produzir trilha consistente, sem converter todo backend indiscriminadamente em Go.
- R4: Regra declarada deve ser distinguida de instalada/disponível/aplicável. Regra ausente deve produzir diagnóstico e próximo passo extension-needed quando houver extensão correspondente; não anunciar rules_effective como arquivo existente nem instalar automaticamente.
- R5: Migrate deve explicar dry-run/inventário e ausência atual de apply com comando/limitação corretos, sem promessa de futura 7.0 ou inferência de que apply já existe. A migração condicionada B43 mantém seu próprio escopo/decisão.
- R6: Comandos sugeridos, opções, links e mensagens devem funcionar na superfície suportada ou recusar explicitamente o que falta; CLI humano/JSON e consumers de suggest devem divulgar a mesma disponibilidade, sem sugerir execução impossível.
- R7: Preservar contratos done, procedimentos de diagnóstico úteis, autorização vigente e opt-in de extensões/adoção; manter uma fonte de verdade e mirrors coerentes sem replicar sequências divergentes ou pedir confirmações já cobertas.
- R8: Qualificar aliases/path, regra presente/ausente/unsupported e jornada distribuída EN/pt-BR, além de migrate dry-run/recusa de apply. Não atribuir eficácia funcional a correção textual sem exercitar o comando pertinente.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-6-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-distributed-instructions-parity.md
- created: .pose/starts/pose-distributed-instructions-parity.json
- created: .pose/changelogs/unreleased/pose-distributed-instructions-parity.md
- modified: .pose/workflows/feature.md
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/cli/suggest.go
- modified: pose-mcp/internal/cli/migrate_dryrun.go
- created: pose-mcp/internal/cli/distributed_instructions_parity_test.go
- created: pose-mcp/internal/scaffold/distributed_instructions_parity_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:distributed-instructions-parity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/suggest.go

### Approach and execution gates

Inventariar arquivos root/scaffold/locales e acrescentar os paths exatos ao plano antes de start. Reutilizar freshness entregue e catálogo de extensões, corrigindo instruções/descoberta em vez de novo scheduler. Classificar compatibilidade dos campos de suggest e mensagens; registrar ADR adicional somente quando houver decisão estrutural nova. Instalação/update de fixtures devem ser isolados e não alterar esta instância. Manter ownership distinta da reconciliação narrativa B44.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5
- Decision: Reutilizar freshness e taxonomia existentes, divulgando disponibilidade real e limitação de apply.
- Rationale: Texto e descoberta devem descrever o motor atual, sem instalar ou prometer capacidade futura.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Discovery duas vezes com inputs estáveis/alterados | Reuso/refresco segundo contrato, mesmas instruções em arquivos distribuídos. |
| V2 — Install/update em fixtures EN/pt-BR/scaffold | Jornada e mirrors coerentes; nenhuma adoção incidental. |
| V3 — go/backend-go, aliases válidos e domains desconhecidos | Normalização com source e limitação explícitos, sem inferência errada de stack. |
| V4 — Regra presente, ausente e extensão unsupported | Disponibilidade real e next step; nenhuma instalação automática. |
| V5 — Migrate dry-run e apply indisponível | Mensagem atual e recusa correta; sem data/versão futura prometida. |
| V6 — CLI/JSON e comandos sugeridos | Mesma trilha, links/opções executáveis ou erro explícito. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Trace planejado; vincular casos aos produtores reais na execução.
Não declarar requisitos satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1,V2 |
| R2 | V2 |
| R3 | V3 |
| R4 | V4 |
| R5 | V5 |
| R6 | V3–V6 |
| R7 | V1,V2,V4 |
| R8 | V1–V6 |

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 6. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Instrução distribuída pode continuar divergente se o inventário excluir um locale/mirror. Disponibilidade de extensão e adoção são decisões separadas da descoberta. Não mudar enforcement para fazer o roteiro parecer executável.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
