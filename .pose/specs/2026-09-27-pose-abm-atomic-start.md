---
slug: pose-abm-atomic-start
status: draft
created_at: 2026-09-18
completed_at:
supersedes:
depends_on: pose-abm-contract-nodes
priority: 2
components: pose-dist/pose-mcp
task_type: feature
delivers:
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
- created: .pose/specs/2026-09-18-pose-abm-atomic-start.md
- created: pose-dist/pose-mcp/internal/pose/start.go
- created: pose-dist/pose-mcp/internal/pose/start_test.go
- created: pose-dist/pose-mcp/internal/cli/start.go
- modified: pose-dist/pose-mcp/internal/pose/readiness.go
- modified: pose-dist/pose-mcp/internal/pose/amendments.go

Inventário inicial de implementação. Antes de codar, reconciliar paths e acrescentar por amendment os schemas, fixtures, documentação e arquivos gerados realmente afetados. Paths de artefatos criados por dependências só são modificados após essas entregas.

### Delivery targets
Alvo planejado: `governance:abm-atomic-start`. Registrar o delivery profile e o produtor de evidência correspondente na matriz e declarar `delivers`/target/entrypoint antes de promover esta spec para `in-progress`; não inventar perfil já instalado. Exigir integration para contrato/capability/governance e reachability + integration/e2e para surface. Este planejamento não ativa policy nem declara entrega composta.

### Mudanças de API/contrato
Ponto de entrada simétrico a close; journal pequeno por transação, recuperação definida para cada crash point. Snapshot do worktree/revision é fato observado, não prova de quando o agente decidiu.

### Dados e operação
Persistir somente o que o contrato exige; projeções são reconstruíveis. Documentar versionamento, idempotência, limites de execução, diagnóstico e proveniência de qualquer novo campo/evento.

### Riscos técnicos
Lock obsoleto, crash e discos diferentes podem impedir atomicidade por rename; testar layout real, fsync/recuperação e mensagens de reconciliação. Evitar snapshots globais caros.

### Rollout e reversão
Disponibilizar preview antes de adoption; active legacy permanece unbaselined até reconciliação explícita. Nunca backdate.

## 4. Tasks

- [ ] Confirmar dependências e revisar o desenho contra a release que será implementada.
- [ ] Registrar delivery profile/target/entrypoint e inventário exato antes de iniciar código.
- [ ] Implementar os requisitos em incrementos coesos e incluir os casos negativos de Validation.
- [ ] Atualizar interfaces, docs e scaffolds que consomem o contrato, sem drift de vendor.
- [ ] Executar matriz aplicável, obter review explícito e anexar evidência por R-ID.
- [ ] Fechar somente pelo gate POSE, com riscos/follow-ups dispostos e resultado de composição atual.

## 5. Decisions

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

### Requirement trace
Pendente de implementação: cada R-ID acima exige evidência nominal dos cenários e do delivery target no closeout. Não usar preenchimento documental como satisfied.

### Gaps conhecidos
Infraestrutura, amostra/usuários ou credenciais necessárias ao aceite deverão ser disponíveis na execução; ausência será reportada sem simulação de sucesso.

## 7. Final Report

### Escopo entregue
Somente especificação de implementação. Runtime, rollout e requisitos permanecem pendentes.

### Riscos residuais
Os riscos de implementação da seção Technical Plan não foram aceitos como entrega.

### Follow-ups
Nenhum desdobramento adicional nesta fase: o escopo pendente está nos requisitos desta spec. Se a implementação revelar trabalho fora deles, registrar owner, criticidade, prazo de triagem e disposição antes do closeout.

