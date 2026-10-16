---
slug: pose-journal-checkpoint-and-lock-recovery
status: done
created_at: 2026-10-13
completed_at: 2026-10-16
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

Paths reconciliados na main b8c33dad antes do start. A escrita original
pertence ao lote documental 4; este delta implementa somente B13. Os leitores e
escritores afetados são o journal de action requests (`parseActionJournal`,
`LoadActionRequest`, `appendActionEvent`), o checkpoint do plano de closeout
(`InspectCloseoutCheckpoint`, escrita e limpeza) e a trava por spec do start
(`lockStart`, `RecoverStartLock`). O consumidor CLI é `action show|list`,
`close --plan|--apply|--resume` e `start --apply|--status|--cancel|--recover-lock`;
o MCP é `pose_action_requests` e `pose_start_status`. Não há rota MCP de close
(declarado no B12).

- modified: .pose/specs/2026-10-13-pose-journal-checkpoint-and-lock-recovery.md
- created: .pose/starts/pose-journal-checkpoint-and-lock-recovery.json
- created: .pose/changelogs/unreleased/pose-journal-checkpoint-and-lock-recovery.md
- created: .pose/adr/2026-10-16-recovery-reads-evidence-not-presence.md
- created: pose-mcp/internal/pose/journal_integrity.go
- created: pose-mcp/internal/pose/start_lock.go
- created: pose-mcp/internal/pose/start_lock_linux.go
- created: pose-mcp/internal/pose/start_lock_other.go
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/pose/start.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/cli/closeout_plan.go
- modified: pose-mcp/internal/cli/start.go
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/mcpserver/server.go
- created: pose-mcp/internal/pose/recovery_integrity_test.go
- created: pose-mcp/internal/pose/recovery_lock_linux_test.go
- created: pose-mcp/internal/cli/recovery_integrity_test.go
- created: pose-mcp/internal/cli/recovery_lock_linux_test.go
- created: pose-mcp/internal/mcpserver/recovery_integrity_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: .gitignore
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/results/b13-front09-probes.py
- created: .pose/results/b13-front09-probes.json
- created: .pose/results/b13-implementation-validation.json
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/roadmaps/pose-state-consistency-and-recovery.md

### Delivery targets

- capability:journal-checkpoint-and-lock-recovery module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/closeout_plan.go

### Approach and execution gates

Contrato no ADR `2026-10-16-recovery-reads-evidence-not-presence`. O journal
é lido por inteiro: cada linha termina em newline e traz exatamente um objeto
JSON, sem campo duplicado (chaves que diferem só na caixa contam como
duplicadas) ou desconhecido, com tipo de evento conhecido, o schema suportado
e a próxima sequência; caracteres de controle escapados continuam válidos. O primeiro
trecho que não lê vira `*ActionJournalError` com linha, offset, tamanho e
SHA-256, `interrupted-tail` (última linha sem terminador, com proposta de
truncamento para revisão humana) ou `corrupt`; o motor nunca trunca e recusa
append sobre ele. O checkpoint é `absent`, `valid`, `corrupt`, `unsupported`
ou `unavailable`; é progresso, não artefato autoritativo, então é trocado por
temporário exclusivo e rename, sem sync; a remoção ao fim é verificada. Apply
sobre checkpoint que não lê é `blocked` antes de qualquer passo; passo de efeito
registrado e não refletido pelo repositório é `conflict` até `--apply --digest`
do plano atual. A trava do start é publicada com o dono (link de temporário
exclusivo) e classificada como `owner-active`, `owner-unknown` ou `owner-dead`
pela identidade (host, digests de máquina e boot, namespace, PID e início do
processo), nunca pela idade; `--recover-lock <owner digest>` remove só a trava
morta inspecionada, sob flock do diretório, sem tocar start record nem spec.
Compatibilidade: recusas novas e campos opcionais; classificado como minor.

B13 não generaliza para geração atômica de índices (B15), publisher/CAS entre
writers (B14), supersession de requests (B10) ou daemon (B35). Prova de vida só
em Linux e na mesma máquina; outras plataformas reportam `owner-unknown`.

## 4. Tasks

- [x] Context, readiness, assessment freshness e mapa de leitores, escritores e travas contra a main b8c33dad.
- [x] ADR do contrato antes do código estrutural.
- [x] Native start com R1–R9/D1–D2 preservados.
- [x] Journal estrito com diagnóstico tipado e proposta de reparo sem reparo automático; CLI e MCP.
- [x] Checkpoint com estados, escrita por rename, remoção verificada, bloqueio e conflito de progresso.
- [x] Trava do start com dono, classificação por identidade, recuperação verificada e status em CLI/MCP.
- [x] V1–V8 nos consumidores domínio, CLI e MCP, com race detector; probes P01–P17 antes e depois (controles inalterados); 29 mutações reprovando; validação estrita do módulo.
- [x] Remediar a primeira revisão independente: chaves que diferem só na caixa contam como duplicadas no journal e no checkpoint; tipo de evento desconhecido é corrupção; `--recover-lock` vazio é recusado pelo parser; o teste de linha grande usa um evento válido, sensível só ao limite.
- [x] Revisão por ator diferente e closeout nativo no PR #187; merge permanece com o maintainer.

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
Testes de domínio, CLI e MCP executados, com vet, suíte completa, race e builds Windows/Darwin; 31 mutações independentes detectadas por asserção. A validação estrita e os gates nativos estão registrados no parecer das duas rodadas.

### Requirement trace

Produtores executados nesta revisão independente; lint valida a estrutura, não
substitui estes testes e probes. Evidência compacta das duas rodadas em
[review](../results/b13-independent-review.json).

- R1 [satisfied] test:TestAJournalIsReadWholeOrNotAtAll test:TestAJournalKeepsTextWithEscapedControls — probes P01/P03/P04/P05/P06/P14 e limites exatos; mutações de schema, sequência, duplicatas, folding, tipo e tamanho detectadas.
- R2 [satisfied] test:TestAnInterruptedTailIsProposedNeverRepaired test:TestAJournalThatDoesNotReadIsLocatedInBothRenderers test:TestMCPLocatesAJournalThatDoesNotRead — diagnósticos byte/hash exatos e arquivo preservado.
- R3 [satisfied] test:TestACheckpointSaysWhatItIs test:TestACheckpointThatDoesNotReadStopsTheApplyBeforeAnyStep — probe unavailable por chmod 000 e aliases corrupt sem passos.
- R4 [satisfied] test:TestACheckpointIsReplacedWholeAndItsRemovalIsReported check:cli-mcp-operation-results-integration — escrita por rename e falhas observáveis em unconfirmed, checkpoint não autoritativo sem garantia de power loss conforme ADR.
- R5 [satisfied] test:TestACheckpointIsProgressNeverAReceipt check:quickstart-real-lifecycle-integration — conflito sem replay e substituição somente por apply com digest atual, replanejamento do repositório.
- R6 [satisfied] test:TestAStartLockIsJudgedByItsOwnersIdentity test:TestALockWithoutAReadableOwnerIsUnknown — identidade host/máquina/boot/namespace/PID/início e idade irrelevante.
- R7 [satisfied] test:TestOnlyAProvenDeadLockIsRecoveredAndOnlyTheOneInspected test:TestConcurrentRecoveriesRemoveTheDeadLockOnce test:TestAReleaseRemovesOnlyItsOwnLock — 20 corridas adicionais com aquisição concorrente.
- R8 [satisfied] test:TestARecoveredLockLeavesTheInterruptedStartToResume check:abm-atomic-start-integration — intenção/baseline preservadas, sem presumir owner após restart nem implementar daemon.
- R9 [satisfied] test:TestAJournalThatDoesNotReadIsLocatedInBothRenderers test:TestTheStartCommandReportsAndRecoversOnlyADeadLock test:TestADeadStartLockIsRecoveredByItsDigestInBothRenderers test:TestMCPLocatesAJournalThatDoesNotRead test:TestMCPStartStatusShowsAHeldLock check:journal-checkpoint-and-lock-recovery-integration — probes P01–P17 e controles preservados.

Consultar também [consistência/recovery](../reports/2026-10-12-pose-front-09-consistency-recovery.md) e [performance](../reports/2026-10-12-pose-front-08-resident-performance.md).

## 7. Final Report

### Delivered scope

B13 entregue com journal integral e diagnóstico tipado sem reparo automático,
checkpoint com cinco estados e retomada por efeitos observados, e trava de start
com owner verificável e recuperação explícita de dono morto sob exclusão mútua.
CLI e MCP qualificados sem mudar policy, capabilities ou adoção. D1/D2 e o ADR
foram julgados na revisão independente Codex/OpenAI, frente ao implementador
Claude/Anthropic, por `agent:independent-gpt-6.1-sol-review`, autorizado pelo
maintainer. Identidade e separação declaradas; atestações não assinadas.

### Review and delivery evidence

O [parecer, Second round](../reports/2026-10-16-pose-b13-journal-checkpoint-lock-recovery-review.md)
e a [evidência compacta das duas rodadas](../results/b13-independent-review.json)
preservam F1–F4 e suas contraprovas. Validação estrita global 135/135 e módulo
132/132 em 8e8b1307; vet, suíte completa, race, builds Windows/Darwin e 31
mutações independentes detectadas por asserção. P01–P17, controles e jornada
quickstart preservados. Bundle `rvb-bbed7a49533bf1e3`, atestação `rva-dfa2006f7d36184d` e closeout nativo
identificados no parecer; lint e check qualificam estrutura, não requisitos
funcionais.

### Residual risks

Prova de liveness limita-se a Linux e à mesma máquina com identidade disponível;
contexto insuficiente permanece owner-unknown. Writers devem cooperar com os
protocolos de exclusão. Checkpoint não autoritativo usa rename sem sync e não
promete durabilidade contra power loss. Fontes externas do Harne8 não foram
inspecionadas nesta revisão limitada a este clone. Os limites e a separação de
B10/B14/B15/B35 permanecem os do ADR; nenhum novo risco foi aceito por finding.

### Follow-ups

Não surgiu follow-up residual desta revisão. O trabalho fora do escopo continua
no programa já governado, sem criar, encerrar, duplicar ou dispositionar outra
spec por inferência. Aguardar merge de #187 pelo maintainer e reavaliar a
readiness nativa sem escolher automaticamente nova implementação.
