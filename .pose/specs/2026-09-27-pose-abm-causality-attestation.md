---
slug: pose-abm-causality-attestation
status: draft
created_at: 2026-09-18
completed_at:
supersedes:
depends_on: xref:proj.pose-dist/spec:pose-abm-contract-nodes, xref:proj.pose-dist/spec:pose-abm-atomic-start, xref:proj.pose-dist/spec:pose-abm-progressive-review
priority: 2
components: pose-dist/pose-mcp
task_type: feature
delivers:
---

# Spec: Causalidade estruturada e closeout proporcional

## 1. Intent

### Objetivo
Consolidar mappings SD→D/R no plano/bundle/attestation e fechar somente com disposições exigidas.

### Valor de negócio
Vincular custo estrutural, base e julgamento sem pontuar arquitetura.

### Restrições
- Owner proposto: @pose-maintainers. Linha: 6.0 candidato.
- Evidência de origem: [review ABM](../reports/2026-09-18-pose-abm-review.md), achados Design causality e judgment effectiveness.
- Contrato transversal: [desenho técnico/UX](../../docs/architecture/pose-abm-design.md).
- Usar uma fonte de verdade; manter offline, privacidade, adoption explícita e compatibilidade.
- Esta é spec de implementação futura; nenhum requisito está declarado satisfeito.

### Não-objetivos
Não criar juiz LLM no core, score de qualidade, ranking individual, infraestrutura paralela ou mudança de escopo fora dos requisitos abaixo.

## 2. Requirements

### Funcionais e critérios de aceite
- R1: Versionar plano/attestation quando necessário e selar structural context, base e obligations por capability, mantendo DAG sem autorreferência.
- R2: Exigir mappings para SD material sob policy adotada; refs resolvem no subject exato e D alcança R/constraint que justifica a mudança.
- R3: Separar completude do mapping de aceitação semântica; ligação arbitrária sintaticamente válida exige julgamento explícito e pode gerar finding.
- R4: Assumption invalidated ou mudança material de R/A/D/SD invalida critérios dependentes; reusar apenas slices idênticos e aprovações autorizadas.
- R5: Unknown/unsupported não vira ausência de estrutura; dispensa ou risco aceito obedece limites, owner/prazo e não pode dispensar integridade.
- R6: Closeout eleva obrigações de elevated/critical sem adicionar gate à fixture trivial; high/critical não implica mandatory-human universal.
- R7: Preservar histórico legado e expor delta de revisão com próximo passo; métricas e registro da attestation não invalidam o bundle.

### Segurança e compatibilidade
Aplicar os negativos da seção Validation. Conteúdo do repositório é input não confiável; ausência de capability/evidência não pode enfraquecer autorização. Dados legados permanecem auditáveis, sem rationale fabricado.

## 3. Technical Plan

### Áreas afetadas
pose-dist/pose-mcp. Reaproveitar os contratos e produtores existentes; registrar alteração material da base antes de ampliar o diff.

### Artifacts
- created: .pose/specs/2026-09-18-pose-abm-causality-attestation.md
- modified: pose-dist/pose-mcp/internal/pose/review_plan.go
- modified: pose-dist/pose-mcp/internal/pose/review_bundle.go
- modified: pose-dist/pose-mcp/internal/pose/review_closeout.go
- created: pose-dist/pose-mcp/internal/pose/design_causality_test.go
- created: pose-dist/pose-mcp/schemas/v2/review-plan.schema.json
- created: pose-dist/pose-mcp/schemas/v2/review-attestation.schema.json

Inventário inicial de implementação. Antes de codar, reconciliar paths e acrescentar por amendment os schemas, fixtures, documentação e arquivos gerados realmente afetados. Paths de artefatos criados por dependências só são modificados após essas entregas.

### Delivery targets
Alvo planejado: `governance:abm-causality-closeout`. Registrar o delivery profile e o produtor de evidência correspondente na matriz e declarar `delivers`/target/entrypoint antes de promover esta spec para `in-progress`; não inventar perfil já instalado. Exigir integration para contrato/capability/governance e reachability + integration/e2e para surface. Este planejamento não ativa policy nem declara entrega composta.

### Mudanças de API/contrato
Mappings pertencem ao julgamento do bundle; observações não são editáveis pelo reviewer. Usar criterion de proporcionalidade para contestar cinco abstrações ligadas ao mesmo R sem razão suficiente. Nenhum pass por presença de ID.

### Dados e operação
Persistir somente o que o contrato exige; projeções são reconstruíveis. Documentar versionamento, idempotência, limites de execução, diagnóstico e proveniência de qualquer novo campo/evento.

### Riscos técnicos
Um grafo formalmente completo pode justificar uma decisão ruim. Corpus mantém o caso mapeado-mas-injustificado e depende de avaliação semântica, não de linter teatral.

### Rollout e reversão
Shadow comparativo seguido de enforcement por adoption. Manter baseline simples e registrar qualquer regressão de cerimônia como blocker do rollout.

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
| SD inexistente, D órfã, ciclo e outro subject; R1/R2/R5 | `cd pose-dist/pose-mcp && go test ./internal/pose -run 'ABMDesignCausality' -count=1` | Referências ilegítimas recusadas |
| Mapping presente mas judgment pendente; R3/R6 | `cd pose-dist/pose-mcp && go test ./internal/pose ./internal/cli -run 'ABMCausalityCloseout' -count=1` | Sem aprovação por simples preenchimento |
| Invalidation/reuse/legacy; R4/R7 | `cd pose-dist/pose-mcp && go test ./internal/pose -run 'ABMCausalityReuse\|ReviewBundle' -count=1` | Revisão só do slice afetado; histórico preservado |

### Checks de governança
- `pose lint-spec pose-abm-causality-attestation --ready-check` antes de iniciar.
- `pose validate --strict --module pose-dist/pose-mcp` no escopo de código aplicável; executar separadamente os demais módulos afetados.
- `pose artifact-check --spec pose-abm-causality-attestation --strict` após atribuição Git correta.
- `pose surface-check --spec pose-abm-causality-attestation --strict` após declaração e composição dos targets.
- `pose review verify spec:pose-abm-causality-attestation` antes da transição terminal.

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
