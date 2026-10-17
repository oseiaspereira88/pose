---
slug: pose-index-generation-integrity
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-writer-coordination-and-atomic-publication, pose-delivery-integrity-index-compaction
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:index-generation-integrity
---

# Spec: Publicar e consumir uma geração completa de índices

## 1. Intent

### Goal

Fazer leitores receberem conjunto coerente identificado ou unavailable, preservando compatibilidade dos consumers e sem misturar gerações após falha/crash.

### Business value

B15 / CONS-05. Index publica arquivos separadamente em ordem variável; falha em services/packages pode deixar outros cinco outputs atualizados. Reutilizar compaction sem confundir arquivo atômico com geração atômica. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise fonte](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Definir conjunto de índices pertencente à geração, identity/schema/input bindings e descriptor/manifest de commit ou mecanismo equivalente; arquivos individuais não devem certificar completion do conjunto.
- R2: Produzir staging exclusivo via B14, validar todos os outputs/digests e publicar um commit de geração que só referencia conjunto completo. Falha antes do commit não deve tornar mistura corrente; manter geração anterior válida ou unavailable explicitamente.
- R3: Leitor deve fixar geração/descriptor para a operação e verificar conjunto/bindings pertinentes; missing/corrupt/incompatible ou mistura de generation IDs deve gerar incomplete/unavailable e nunca zero completo ou verdict de coerência fabricado.
- R4: Crash/retry/concorrência em qualquer fase deve conservar intenção, identificar commit efetivo e convergir idempotentemente, sem publicar descriptor parcial ou confundir falha de cleanup com falha de publicação; receipts/outcomes devem usar B12/B13 via B14.
- R5: Migrar paths e consumers atuais com compatibilidade explícita; fallback legacy não deve combinar arquivos de gerações diversas silenciosamente. Preservar schema de compaction e provenance/digests equivalentes quando os inputs não mudaram.
- R6: Readers e writers CLI/MCP devem usar o mesmo contrato de geração, com invalidação mínima pertinente e status conhecido/unknown; não tratar HEAD/mtime ou existência de um arquivo como commit integral.
- R7: Cleanup/retenção deve ser limitado e respeitar leitores que fixaram a geração e processos concorrentes; não apagar geração viva ou diretórios arbitrários após falha. Declarar limites de filesystem/plataformas e opções de migração antes do código.
- R8: Qualificar o probe de output bloqueado, crash/fault injection antes/depois do descriptor, readers concorrentes, retry e legacy, garantindo geração completa ou indisponível; não implementar daemon/read views globais de B34/B35 neste delta.

## 3. Technical Plan

### Artifacts

Paths reconciliados na main 18e4342b antes do start. A escrita original pertence ao
lote documental 7; este delta implementa somente B15. Os sete outputs de `pose index`
passam a ser uma geração identificada por um descritor (`.pose/indexes/generation.json`)
publicado por último; os leitores (`GetDeliveryIntegrity`, plano de review, projeção de
obrigações, `pose suggest`) leem pelo mesmo contrato e os dois escritores diretos de
`delivery-integrity.json` (refresh do artifact-check e bootstrap do review bundle)
passam a substituir o arquivo dentro de uma nova geração. O MCP lê o conjunto pelo
mesmo `Store`; seu teste cobre a ferramenta `pose_delivery_integrity`.

- modified: .pose/specs/2026-10-13-pose-index-generation-integrity.md
- created: .pose/starts/pose-index-generation-integrity.json
- created: .pose/changelogs/unreleased/pose-index-generation-integrity.md
- created: .pose/adr/2026-10-17-index-sets-are-published-as-verified-generations.md
- created: pose-mcp/internal/pose/index_generation.go
- created: pose-mcp/internal/pose/index_generation_test.go
- created: pose-mcp/internal/cli/index_generation_test.go
- created: pose-mcp/internal/mcpserver/index_generation_test.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/suggest.go
- modified: pose-mcp/internal/cli/progressive_review_test.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go
- modified: .gitignore
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/results/b15-front09-probes.py
- created: .pose/results/b15-front09-probes.json
- created: .pose/results/b15-implementation-validation.json
- created: .pose/results/b15-mutations.py
- created: .pose/results/b15-mutations.json
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/roadmaps/pose-state-consistency-and-recovery.md

### Delivery targets

- capability:index-generation-integrity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/index.go

### Approach and execution gates

Definir descriptor/generation e estratégia de compatibilidade em ADR prospectivo. Inventariar os sete outputs atuais e todos os consumers pertinentes antes de start; ampliar o plano exato onde necessário. Reutilizar B14 e compaction, publicar o ponteiro somente após validar conjunto. Operação de leitura fixa geração com limite de retenção; não substituir isso por snapshot stale universal ou pelo futuro daemon.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R7
- Decision: Certificar geração pelo commit integral e bindings, preservando consumers com migração explícita.
- Rationale: Atomicidade por arquivo e sucesso de parte dos outputs não certificam o conjunto.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Output services/packages bloqueado no staging | Nenhum conjunto misto corrente; anterior válido ou unavailable explícito. |
| V2 — Leitor concorrente com publicação | Uma geração completa fixada e verificada, nunca combinação de versões. |
| V3 — Crash antes/depois de descriptor e retry mesma intenção | Commit efetivo identificado; convergência uma vez, sem descriptor parcial. |
| V4 — Missing/corrupt/schema/input binding divergente | Diagnóstico e unavailable/incomplete; não zero completo. |
| V5 — Legacy paths, compaction e consumers CLI/MCP | Migração compatível, equivalência de provenance e mesma classificação. |
| V6 — Cleanup/retenção enquanto reader usa geração | Geração viva preservada; limites e fase/erro observáveis. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Trace planejado; vincular os casos aos produtores reais na execução.
Não declarar requisitos satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1,V2 |
| R2 | V1,V3 |
| R3 | V2,V4 |
| R4 | V3 |
| R5 | V5 |
| R6 | V2,V5 |
| R7 | V6 |
| R8 | V1–V6 |

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 7. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Compatibilidade com paths históricos e readers antigos pode exigir migração em fases; nenhum fallback pode alegar coerência que não comprova. Um conjunto completo ainda depende de cobertura/freshness dos inputs, a qualificar nos deltas pertinentes. Retenção não pode ser ilimitada nem ignorar readers vivos.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
