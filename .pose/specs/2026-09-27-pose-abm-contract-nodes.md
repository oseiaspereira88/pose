---
slug: pose-abm-contract-nodes
status: draft
created_at: 2026-09-18
completed_at:
supersedes:
depends_on:
priority: 2
components: pose-dist/pose-mcp
task_type: feature
delivers:
---

# Spec: Contrato versionado de nós R/A/D e amendments

## 1. Intent

### Objetivo
Consolidar identidade, relações e histórico de requisitos, premissas e decisões materiais.

### Valor de negócio
Permitir auditoria de mudanças de base sem criar log paralelo.

### Restrições
- Owner proposto: @pose-maintainers. Linha: 6.0 candidato.
- Evidência de origem: [review ABM](../reports/2026-09-18-pose-abm-review.md), achados Premissas sem identidade e racionalização posterior.
- Contrato transversal: [desenho técnico/UX](../../docs/architecture/pose-abm-design.md).
- Usar uma fonte de verdade; manter offline, privacidade, adoption explícita e compatibilidade.
- Esta é spec de implementação futura; nenhum requisito está declarado satisfeito.

### Não-objetivos
Não criar juiz LLM no core, score de qualidade, ranking individual, infraestrutura paralela ou mudança de escopo fora dos requisitos abaixo.

## 2. Requirements

### Funcionais e critérios de aceite
- R1: Versionar projeção de contract nodes e amendment events preservando leitura de R-only v1; não ligar schema automaticamente à versão comercial.
- R2: Generalizar baseline/hash/eventos append-only para R/A/D, com IDs estáveis, namespace, estado e transições explícitas.
- R3: Exigir reconhecimento de mudanças materiais após baseline conforme policy adotada; reconhecer mudança editorial sem mascarar alteração semântica.
- R4: Preservar Decisions como fonte canônica e invalidação já existente do bundle; Tasks, Final Report e métricas derivadas não alteram o subject semântico.
- R5: Detectar refs órfãs, nós retirados ainda usados e cycles; invalidated assumption não sustenta decisão ativa até reconciliação.
- R6: Expor before/after, origem e assurance do registro sem transcrição cognitiva; legado não recebe rationale/data inventados.
- R7: Atualizar schema registry, CLI/MCP, scaffold/locales e fixtures de migração; reader sem capability obrigatória recusa interpretação permissiva.

### Segurança e compatibilidade
Aplicar os negativos da seção Validation. Conteúdo do repositório é input não confiável; ausência de capability/evidência não pode enfraquecer autorização. Dados legados permanecem auditáveis, sem rationale fabricado.

## 3. Technical Plan

### Áreas afetadas
pose-dist/pose-mcp. Reaproveitar os contratos e produtores existentes; registrar alteração material da base antes de ampliar o diff.

### Artifacts
- created: .pose/specs/2026-09-18-pose-abm-contract-nodes.md
- modified: pose-dist/pose-mcp/internal/pose/amendments.go
- modified: pose-dist/pose-mcp/internal/cli/amend.go
- modified: pose-dist/pose-mcp/internal/pose/design_basis.go
- modified: pose-dist/pose-mcp/internal/pose/review_bundle.go
- created: pose-dist/pose-mcp/internal/pose/contract_nodes_test.go
- created: pose-dist/pose-mcp/schemas/v2/contract-nodes.schema.json

Inventário inicial de implementação. Antes de codar, reconciliar paths e acrescentar por amendment os schemas, fixtures, documentação e arquivos gerados realmente afetados. Paths de artefatos criados por dependências só são modificados após essas entregas.

### Delivery targets
Alvo planejado: `contract:abm-contract-nodes`. Registrar o delivery profile e o produtor de evidência correspondente na matriz e declarar `delivers`/target/entrypoint antes de promover esta spec para `in-progress`; não inventar perfil já instalado. Exigir integration para contrato/capability/governance e reachability + integration/e2e para surface. Este planejamento não ativa policy nem declara entrega composta.

### Mudanças de API/contrato
Promover o parser advisory validado no piloto. Event v2 pode referenciar nodes tipados; manter adaptação v1 sem reescrita. Contrato de serialização e normalização deve ter golden bytes antes de novos digests públicos.

### Dados e operação
Persistir somente o que o contrato exige; projeções são reconstruíveis. Documentar versionamento, idempotência, limites de execução, diagnóstico e proveniência de qualquer novo campo/evento.

### Riscos técnicos
Normalização excessiva pode esconder mudança semântica; exigir casos contrastivos. Históricos antigos incompletos permanecem assim, com diagnóstico.

### Rollout e reversão
Somente após stop/go do piloto; dual reader e capability explícita. Done legado auditável; novas obrigações limitadas à adoção.

## 4. Tasks

- [ ] Confirmar dependências e revisar o desenho contra a release que será implementada.
- [ ] Registrar delivery profile/target/entrypoint e inventário exato antes de iniciar código.
- [ ] Implementar os requisitos em incrementos coesos e incluir os casos negativos de Validation.
- [ ] Atualizar interfaces, docs e scaffolds que consomem o contrato, sem drift de vendor.
- [ ] Executar matriz aplicável, obter review explícito e anexar evidência por R-ID.
- [ ] Fechar somente pelo gate POSE, com riscos/follow-ups dispostos e resultado de composição atual.

## 5. Decisions

- Date: 2026-09-27. Após a transferência do Harne8, o gate do piloto de campo
  (decisão D1 do plano 6.0.0) deixa de ser `depends_on` desta spec e passa a
  `after: spec:pose-abm-field-pilot` do milestone `contract-nodes` no roadmap
  coordenador `pose-abm-contract-consolidation` (proj.harne8). O motor não
  depende de um escopo do consumidor: o checkout isolado do pose-dist não o
  resolveria, e o ADR de reconciliação proíbe essa aresta.

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
| R-only versus R/A/D; R1/R2/R7 | `cd pose-dist/pose-mcp && go test ./internal/pose ./internal/cli -run 'ABMContractNodes\|Amend' -count=1` | Roundtrip e compatibilidade sem perda |
| Invalidated, withdrawn e semantic drift; R3/R5/R6 | `cd pose-dist/pose-mcp && go test ./internal/pose -run 'ABMNodeTransitions' -count=1` | Mudança exige reconhecimento; sem rationale inventado |
| Digest/staleness; R4 | `cd pose-dist/pose-mcp && go test ./internal/pose -run 'ABMNodeDigest\|ReviewBundleSemantic' -count=1` | Sem mudança de comportamento para conteúdo derivado |

### Checks de governança
- `pose lint-spec pose-abm-contract-nodes --ready-check` antes de iniciar.
- `pose validate --strict --module pose-dist/pose-mcp` no escopo de código aplicável; executar separadamente os demais módulos afetados.
- `pose artifact-check --spec pose-abm-contract-nodes --strict` após atribuição Git correta.
- `pose surface-check --spec pose-abm-contract-nodes --strict` após declaração e composição dos targets.
- `pose review verify spec:pose-abm-contract-nodes` antes da transição terminal.

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
