---
slug: pose-abm-atomic-start
status: done
created_at: 2026-09-18
completed_at: 2026-09-27
supersedes:
depends_on: pose-abm-contract-nodes
priority: 2
components: pose-dist/pose-mcp
task_type: feature
delivers: governance:abm-atomic-start
---

# Spec: Início governado e baseline transacional

## 1. Intent

### Objetivo
Aplicar readiness, baseline de R/A/D e início como uma operação recuperável e idempotente.

### Valor de negócio
Tornar visível o que estava registrado antes da execução gerenciada.

A promessa é essa e apenas essa: `pose start` fornece a baseline contra a qual um amendment de R/A/D significa alguma coisa. Não prova que a decisão foi pensada antes de codar, e não deve ser apresentada como defesa contra racionalização posterior — um snapshot registra precedência observável, não cronologia cognitiva. Vendida como prova de pensamento é cerimônia; vendida como origem do diff de decisão é barata e útil. Por isso `start` não é exigido na banda `baseline`. Ver [corroboração](../reports/2026-09-18-pose-abm-corroboration.md), seção 4.

### Restrições
- Owner proposto: @pose-maintainers. Linha: consolidação 6.0.0.
- Evidência de origem: [review ABM](../reports/2026-09-18-pose-abm-review.md), achados Cronologia de base e integridade de transição.
- Contrato transversal: [desenho técnico/UX](../../docs/architecture/pose-abm-design.md).
- Usar uma fonte de verdade; manter offline, privacidade, adoption explícita e compatibilidade.
- Esta é spec de implementação futura; nenhum requisito está declarado satisfeito.

### Não-objetivos
Não criar juiz LLM no core, score de qualidade, ranking individual, infraestrutura paralela ou mudança de escopo fora dos requisitos abaixo.

## 2. Requirements

### Funcionais e critérios de aceite
- R1: Oferecer pose start spec:slug com preview/dry-run e apply explícito, resolvendo readiness, dependências e obligations pré-implementação.
- R2: Gravar baseline, registro de execução e transição draft→in-progress com lock por spec, revision esperada e journal/recuperação de falha.
- R3: Repetição com mesma intenção é idempotente; duas execuções concorrentes não produzem baselines divergentes nem sobrescrita.
- R4: Classificar recorded-before-managed-execution, introduced-during-execution e legacy-unbaselined; não alegar precedência cognitiva.
- R5: Detectar edição manual de status/base sob contrato adotado e pedir reconciliação; não impedir leitura do projeto nem fabricar histórico.
- R6: Não exigir structural delta futuro: utilizar base/previsão declarada e recalcular obligations quando existir subject observado.
- R7: Reutilizar o motor em integração Harne8 por capability; preservar modo offline, erros acionáveis e cancelamento sem meia transição.

### Segurança e compatibilidade
Aplicar os negativos da seção Validation. Conteúdo do repositório é input não confiável; ausência de capability/evidência não pode enfraquecer autorização. Dados legados permanecem auditáveis, sem rationale fabricado.

## 3. Technical Plan

### Áreas afetadas
pose-dist/pose-mcp. Reaproveitar os contratos e produtores existentes; registrar alteração material da base antes de ampliar o diff.

### Artifacts
- created: .pose/specs/2026-09-27-pose-abm-atomic-start.md
- created: pose-mcp/internal/pose/start.go
- created: pose-mcp/internal/pose/start_test.go
- created: pose-mcp/internal/cli/start.go
- created: pose-mcp/internal/cli/start_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- created: pose-mcp/internal/mcpserver/start_test.go
- modified: docs-site/docs/mcp.md
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/changelogs/unreleased/pose-abm-atomic-start.md

Reconciliado em 2026-09-27: paths sem o prefixo `pose-dist/`; `readiness.go`
e `amendments.go` não mudam, porque `SpecReadiness` e a projeção de contract
nodes já entregam readiness e baseline.

### Delivery targets
- governance:abm-atomic-start module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

Produtor integration: check `abm-atomic-start-integration` (`go test
./internal/pose ./internal/cli ./internal/mcpserver -run 'ABMAtomicStart'
-count=1`). Nenhuma instância adota a capability nesta entrega.

### Desenho
- `pose start spec:<slug>` é preview por padrão, somente leitura e sempre
  disponível: readiness e dependências (`SpecReadiness`), obrigações
  pré-implementação a partir do plano de review declarado (sem structural
  delta), baseline R/A/D (projeção de contract nodes), revisão Git e digest do
  arquivo, num plano vinculado a digest.
- `--apply --digest` exige `atomic_start_version: 1` na policy de review.
  Lock exclusivo por spec, compare-and-swap no plano recalculado, registro em
  `.pose/starts/<slug>.json` com fase `baseline-recorded` e, depois, a
  transição `draft → in-progress` e a fase `started`. Repetir com o mesmo
  plano é idempotente e retoma uma execução interrompida; outro plano para
  spec já iniciada é recusado. `--cancel` desfaz um registro sem transição.
- `--status` classifica cada nó como `recorded-before-managed-execution`,
  `introduced-during-execution` ou `legacy-unbaselined`, recalcula as
  obrigações e, com a capability, pede reconciliação para status alterado à
  mão ou registro editado. Nunca afirma precedência cognitiva nem data retroativa.

### Mudanças de API/contrato
Ponto de entrada simétrico a close; journal pequeno por transação, recuperação definida para cada crash point. Snapshot do worktree/revision é fato observado, não prova de quando o agente decidiu.

### Dados e operação
Persistir somente o que o contrato exige; projeções são reconstruíveis. Documentar versionamento, idempotência, limites de execução, diagnóstico e proveniência de qualquer novo campo/evento.

### Riscos técnicos
Lock obsoleto, crash e discos diferentes podem impedir atomicidade por rename; testar layout real, fsync/recuperação e mensagens de reconciliação. Evitar snapshots globais caros.

### Rollout e reversão
Disponibilizar preview antes de adoption; active legacy permanece unbaselined até reconciliação explícita. Nunca backdate.

## 4. Tasks

- [x] Confirmar dependências e revisar o desenho contra a release que será implementada.
- [x] Registrar delivery profile/target/entrypoint e inventário exato antes de iniciar código.
- [x] Implementar os requisitos em incrementos coesos e incluir os casos negativos de Validation.
- [x] Atualizar interfaces, docs e scaffolds que consomem o contrato, sem drift de vendor.
- [x] Executar matriz aplicável, obter review explícito e anexar evidência por R-ID.
- [x] Fechar somente pelo gate POSE, com riscos/follow-ups dispostos e resultado de composição atual.

## 5. Decisions

- Date: 2026-09-27. Implementado sem rollout, como o contract-nodes: preview
  disponível a todos, apply atrás de `atomic_start_version`, sem adoção.

- Data: 2026-09-18.
- Contexto: a análise inicial propõe novos controles; o review confirmou limites que exigem implementação proporcional.
- Opções consideradas: novo subsistema ABM; extensão dos contratos existentes.
- Decisão proposta: extensão limitada descrita nesta spec e no desenho compartilhado.
- Racional: preservar autoridade única e custo proporcional sem transformar uma referência em prova de julgamento.
- Consequências: novos contratos precisam de casos negativos, migração e validação do consumidor.
- ADR: [limites de julgamento e evidência](../adr/2026-09-18-pose-abm-judgment-and-evidence-boundaries.md); [autoridade Harne8](../adr/2026-09-18-harne8-abm-single-governance-authority.md).

## 6. Validation

### Estratégia
Plano anterior à implementação. Os nomes de testes ABM e comandos novos abaixo são entregas a criar pelas respectivas specs; não foram executados como aceite desta feature. Checks existentes da matriz permanecem obrigatórios.

| Cenário/requisitos | Comando planejado | Evidência esperada |
| --- | --- | --- |
| Start/replay/conflito; R1/R2/R3 | `cd pose-dist/pose-mcp && go test -race ./internal/pose ./internal/cli -run 'ABMStart' -count=1` | Uma baseline e transição por operação |
| Crash após cada write e manual status; R2/R4/R5 | `cd pose-dist/pose-mcp && go test ./internal/pose -run 'ABMStartRecovery' -count=1` | Recuperação íntegra ou bloqueio explícito |
| Offline e preflight; R6/R7 | `pose validate --strict --module pose-dist/pose-mcp` | Sem dependência de rede/delta futuro |

### Checks de governança
- `pose lint-spec pose-abm-atomic-start --ready-check` antes de iniciar.
- `pose validate --strict --module pose-dist/pose-mcp` no escopo de código aplicável; executar separadamente os demais módulos afetados.
- `pose artifact-check --spec pose-abm-atomic-start --strict` após atribuição Git correta.
- `pose surface-check --spec pose-abm-atomic-start --strict` após declaração e composição dos targets.
- `pose review verify spec:pose-abm-atomic-start` antes da transição terminal.

### Log de execução
2026-09-18: planejamento criado. Validação editorial deste documento não satisfaz os requisitos de runtime.

2026-09-27: implementado sem rollout. Testes novos cobrem preview somente
leitura e vinculado a digest, capability obrigatória no apply, baseline e
transição única, idempotência, retomada após interrupção, cancelamento sem
meia transição, plano obsoleto, digest errado, spec não pronta, oito applies
concorrentes gravando uma baseline, classificação de origem dos nós e
reconciliação sem histórico fabricado, além de CLI e da tool MCP
`pose_start_status`. Com o apply ignorando capability e digest, 3 testes
reprovaram. A primeira suíte completa pegou 58 falhas de instalação: o
manual citava `.pose/starts/`, caminho inexistente numa instalação nova; o
texto passou a descrever o registro sem o caminho. `go test ./...`, `go vet`
e o `-race` da seleção passam. A linha de `pose_spec_amendments` em
`docs-site/docs/mcp.md` também foi atualizada para os campos do contract-nodes.
Em `f20ceee` a matriz completa passou 31/31, com o novo check
`abm-atomic-start-integration`; `artifact-check` com 20 claims e sem erros,
`surface-check` sem achados. Bundle `rvb-a00394e596550afd` aprovado pela
atestação `rva-18ed561feeef7eca`, registrada pelo agente com autorização
explícita do usuário para autoatestar; `validate` de docs-site foi
dispensado porque a matriz não declara produtor para esse componente. As
reviews seladas invalidadas pela matriz foram renovadas.

### Requirement trace
- R1 [satisfied] test:TestABMAtomicStartPreviewIsReadOnlyAndDigestBound test:TestABMAtomicStartCLIPreviewApplyStatus
- R2 [satisfied] test:TestABMAtomicStartApplyRecordsBaselineAndTransitionsOnce test:TestABMAtomicStartResumesAfterInterruptionAndCancels
- R3 [satisfied] test:TestABMAtomicStartApplyRecordsBaselineAndTransitionsOnce test:TestABMAtomicStartConcurrentAppliesRecordOneBaseline
- R4 [satisfied] test:TestABMAtomicStartApplyRecordsBaselineAndTransitionsOnce test:TestABMAtomicStartReconciliationNeverFabricatesHistory
- R5 [satisfied] test:TestABMAtomicStartReconciliationNeverFabricatesHistory
- R6 [satisfied] test:TestABMAtomicStartPreviewIsReadOnlyAndDigestBound
- R7 [satisfied] test:TestABMAtomicStartCLIApplyNeedsCapability test:TestABMAtomicStartMCPStatusIsReadOnly test:TestABMAtomicStartResumesAfterInterruptionAndCancels

### Gaps conhecidos
Infraestrutura, amostra/usuários ou credenciais necessárias ao aceite deverão ser disponíveis na execução; ausência será reportada sem simulação de sucesso.

## 7. Final Report

### Escopo entregue
`pose start` com preview somente leitura, apply atômico, idempotente e
retomável atrás de `atomic_start_version`, cancelamento sem meia transição,
classificação de origem dos nós e reconciliação sem histórico fabricado, em
CLI e na tool MCP somente leitura `pose_start_status`. Nenhuma instância
adota a capability.

### Riscos residuais
A atomicidade usa lock exclusivo e escrita com compare-and-swap no mesmo
diretório; discos de rede sem `O_EXCL` confiável não foram testados. Um lock
deixado por queda exige remoção manual após confirmação.

### Follow-ups
- [open] Adotar `atomic_start_version` depois do stop/go do piloto, junto com `contract_nodes_version`. (owner:@pose-maintainers crit:medium review:2026-11-01)
