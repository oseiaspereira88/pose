---
slug: pose-journal-checkpoint-and-lock-recovery
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-cli-mcp-operation-results, pose-recoverable-closeout-plan, pose-abm-atomic-start
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:journal-checkpoint-and-lock-recovery
---

# Spec: Recuperar journals, checkpoints e locks pela evidência

## 1. Intent

### Goal

Recusar leitura parcial/corrupta como estado válido, revelar persistência falha e permitir retomada segura sem quebrar lock de owner vivo.

### Business value

B13 / CONS-02, CONS-07 e CONS-12. Journal aceita objeto extra após primeiro JSON na linha; checkpoint ilegível/corrupto vira absent; writes de checkpoint podem ser ignorados; O_EXCL pode sobreviver ao processo. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de UX](../reports/2026-10-12-pose-front-07-operational-ux.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Parser de journal deve consumir integralmente cada linha, exigir EOF após o objeto, schema suportado e seq integral/contígua conforme contrato, com limites explícitos e política deliberada de chaves duplicadas. Truncamento, objeto extra e scanner error não devem fornecer view aparentemente válida.
- R2: Diagnóstico de journal inválido deve preservar identificação, bytes/posição/hash pertinentes e distinguir tail interrompido de corrupção em qualquer posição. Repair deve ser proposta revisável e autorizada, sem completar answer/waiver, apagar evento válido ou truncar automaticamente para satisfazer gate.
- R3: Checkpoint deve distinguir absent legítimo, corrupt, unsupported e unavailable; erro de parse/leitura não deve iniciar operação nova silenciosamente. Validar schema/scope/plan digest/progresso e revelar o que é desconhecido, sem tratar marcador completed como receipt de efeito não comprovado.
- R4: Erro de persistência/Sync/clear de checkpoint e progresso deve ser observado e retornar outcome de B12 com fase/efeitos confirmados/limites de durabilidade; não anunciar resume seguro ou complete quando a gravação pertinente falhou. Distinguir artefato autoritativo de cache descartável antes de impor custo de durabilidade.
- R5: Resume deve manter intenção/ID e revalidar inputs, policy/autoridade, digests e postimagens/progresso pertinentes antes de executar só efeitos pendentes comprovados. Plano stale ou checkpoint adulterado deve gerar conflito/reconcile, não replay ou completion por lista de steps.
- R6: Recovery de lock persistente deve identificar recurso/worktree, operação/fase e owner verificável com contexto host/boot/processo suficiente para distinguir PID reutilizado. Idade/mtime/TTL não deve autorizar remoção; owner vivo ou liveness desconhecida deve impedir limpeza automática.
- R7: Lock órfão comprovado pode ser reconciliado sob exclusão mútua e comparação da identidade atual, preservando intenção/progresso e recusando corrida com aquisição nova; OS advisory lock liberado e sentinel O_EXCL não devem ser tratados como o mesmo mecanismo. Declarar limites de plataformas, filesystem e writers não cooperantes.
- R8: Contrato deve conservar geração/intenção e continuidade após restart/fallback do executor existente. Futuro daemon/runtime deve consumir esse contrato na sua própria qualificação; esta spec não cria daemon nem presume que processo reiniciado seja o owner anterior.
- R9: CLI/MCP/status/resume devem revelar corrupt/unavailable/owner-active/owner-unknown/stale e próximo passo suportado, sem apagar arquivos ou mudar policy por inferência. Qualificar crash/retry/concorrência e controles íntegros por fixtures sintéticas e preservar história.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-4-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-journal-checkpoint-and-lock-recovery.md
- created: .pose/starts/pose-journal-checkpoint-and-lock-recovery.json
- created: .pose/changelogs/unreleased/pose-journal-checkpoint-and-lock-recovery.md
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/pose/start.go
- modified: pose-mcp/internal/cli/closeout_plan.go
- created: pose-mcp/internal/pose/recovery_integrity_test.go
- created: pose-mcp/internal/cli/recovery_integrity_test.go
- created: pose-mcp/internal/mcpserver/recovery_integrity_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:journal-checkpoint-and-lock-recovery module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/closeout_plan.go

### Approach and execution gates

Reutilizar parser estrito, atomic-start e resultados de B12; mapear readers/writers/locks pertinentes e registrar protocolo antes do código. Definir receipt/fase suficiente para o closeout atual e recovery por pre/postimages, sem transação universal. Não generalizar para geração atômica de índices (B15), publisher/CAS (B14), supersession específica de requests (B10) ou daemon (B35). Qualificar o contrato de fallback atual e deixar integração com runtime futuro no lote correspondente.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5
- Decision: Distinguir corrupção/indisponibilidade de ausência e retomar por intenção e efeito comprovados.
- Rationale: Um checkpoint legível ou uma lista completed não prova o estado publicado.

### Decision D2
- Basis: R6, R7, R8
- Decision: Recuperar lock somente com owner/identidade comprovados e exclusão mútua pertinente.
- Rationale: Lock antigo pode continuar vivo; TTL e PID isolado não provam abandono.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — JSON extra, seq/schema inválidos, truncamento e chaves duplicadas | Recusa tipada; consumo integral; bytes/posição preservados e sem resposta inventada. |
| V2 — Checkpoint absent/corrupt/unsupported/unavailable | Estados distintos; não iniciar execução nova pelo erro. |
| V3 — Falha de write/Sync/clear após efeito | Fase/receipt e durabilidade limitada expostos pelo outcome de B12. |
| V4 — Resume com plan/input/authority stale e progresso adulterado | Revalidação recusa/reconcile; efeito comprovado não duplicado. |
| V5 — Lock vivo antigo, owner desconhecido e PID reutilizado | Sem limpeza por idade; identidade e liveness verificadas. |
| V6 — Órfão comprovado versus aquisição concorrente | Recovery protegido; não remover lock novo nem perder intenção. |
| V7 — Crash/restart/fallback e plataformas de lock diferentes | Geração/intenção conservadas ou conflito explícito; sem daemon implementado. |
| V8 — CLI/MCP e controles de journal/checkpoint íntegros | Paridade e retry seguro, sem repair destrutivo automático. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Trace planejado; vincular os casos aos produtores reais na implementação.
Não declarar requisitos funcionais satisfeitos pelo lint.

| Requisito | Casos planejados |
|---|---|
| R1 | V1 |
| R2 | V1 |
| R3 | V2 |
| R4 | V3 |
| R5 | V4 |
| R6 | V5 |
| R7 | V6 |
| R8 | V7 |
| R9 | V1–V8 |

Consultar também [consistência/recovery](../reports/2026-10-12-pose-front-09-consistency-recovery.md) e [performance](../reports/2026-10-12-pose-front-08-resident-performance.md).

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 4. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Liveness pode ser impossível de provar em host remoto, boot desconhecido ou filesystem diferente; manter owner-unknown e caminho de reconciliação. Receipt não deve confiar apenas no checkpoint editável. Garantias de Sync e coordenação precisam de contrato por classe de artefato/plataforma, sem prometer proteção contra editor não cooperante.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
