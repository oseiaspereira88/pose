---
slug: pose-abm-contract-nodes
status: in-progress
created_at: 2026-09-18
completed_at:
supersedes:
depends_on:
priority: 2
components: pose-dist/pose-mcp
task_type: feature
delivers: contract:abm-contract-nodes
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
- modified: .pose/specs/2026-09-27-pose-abm-contract-nodes.md
- created: pose-mcp/internal/pose/contract_nodes.go
- modified: pose-mcp/internal/pose/amendments.go
- modified: pose-mcp/internal/pose/design_basis.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/amend.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- created: pose-mcp/internal/pose/contract_nodes_test.go
- created: pose-mcp/internal/pose/testdata/contract-nodes/v1-amendments.jsonl
- created: pose-mcp/internal/pose/testdata/contract-nodes/v1-spec.md
- created: pose-mcp/internal/cli/contract_nodes_cli_test.go
- created: pose-mcp/internal/mcpserver/contract_nodes_test.go
- modified: pose-mcp/internal/pose/schema_test.go
- created: pose-mcp/schemas/v1/contract-nodes.schema.json
- modified: pose-mcp/schemas/README.md
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/changelogs/unreleased/pose-abm-contract-nodes.md

Reconciliado em 2026-09-27 contra o motor fixado: os paths perderam o prefixo
`pose-dist/` (esta spec vive no próprio pose-dist); `review_bundle.go` não
muda, porque Decisions já é seção semântica do bundle (R4 vira teste de
invariância); o schema entra em `schemas/v1/` como contrato novo e aditivo,
conforme o README de schemas, e não em `v2/`.

### Delivery targets
- contract:abm-contract-nodes module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

Produtor integration: check `abm-contract-nodes-integration` da matriz
(`go test ./internal/pose ./internal/cli ./internal/mcpserver -run
'ABMContractNodes|ABMNodeTransitions|ABMNodeDigest|Amend' -count=1`); unit pelo
`go test ./...` do módulo. Nenhuma instância adota a capability nesta entrega.

### Desenho
- Projeção `contract-nodes` (schema 1, independente da versão comercial): nós
  R (texto do requisito), A (claim, scope, affects, evidence; estado
  unverified|verified|invalidated|withdrawn) e D (basis, opções, rationale,
  consequências, falsifier; estado active|withdrawn, novo campo opcional
  `Status` na decisão), com namespace `spec:<slug>`, hash normalizado de
  conteúdo separado do estado, relações e digest sem números de linha.
- Eventos de amendment schema 2: IDs R/A/D, `before`/`after` por nó (hash e
  estado), mudança `transition` para estado, origem (author/reviewer) e
  `assurance: declared`. O reader continua lendo schema 1 como histórico só de
  R, sem rationale inventado.
- Capability `contract_nodes_version: 1` na policy de review. Sem ela, o gate
  segue só R (comportamento atual) e um log com evento schema 2 é recusado,
  não interpretado. Com ela, o gate cobre R/A/D, transições e as regras de R5.
- Editorial não mascara semântica: um reconhecimento `editorial` é recusado
  quando o nó mudou relações (affects/basis) ou estado.
- R5: D ativa com basis em A `invalidated` ou `withdrawn` é erro até a
  decisão mudar de basis ou ser retirada; refs órfãs e ciclos seguem do parser
  de design basis.

### Mudanças de API/contrato
Promover o parser advisory validado no piloto. Event v2 pode referenciar nodes tipados; manter adaptação v1 sem reescrita. Contrato de serialização e normalização deve ter golden bytes antes de novos digests públicos.

### Dados e operação
Persistir somente o que o contrato exige; projeções são reconstruíveis. Documentar versionamento, idempotência, limites de execução, diagnóstico e proveniência de qualquer novo campo/evento.

### Riscos técnicos
Normalização excessiva pode esconder mudança semântica; exigir casos contrastivos. Históricos antigos incompletos permanecem assim, com diagnóstico.

### Rollout e reversão
Somente após stop/go do piloto; dual reader e capability explícita. Done legado auditável; novas obrigações limitadas à adoção.

## 4. Tasks

- [x] Confirmar dependências e revisar o desenho contra a release que será implementada.
- [x] Registrar delivery profile/target/entrypoint e inventário exato antes de iniciar código.
- [x] Implementar os requisitos em incrementos coesos e incluir os casos negativos de Validation.
- [x] Atualizar interfaces, docs e scaffolds que consomem o contrato, sem drift de vendor.
- [ ] Executar matriz aplicável, obter review explícito e anexar evidência por R-ID.
- [ ] Fechar somente pelo gate POSE, com riscos/follow-ups dispostos e resultado de composição atual.

## 5. Decisions

- Date: 2026-09-27. Decisão do usuário: implementar agora, sem rollout. O
  gate do piloto (D1) passa a travar adoção da capability, rollout e closeout
  do milestone `contract-nodes`, não a escrita do código; nenhuma instância
  adota `contract_nodes_version` nesta entrega.

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
2026-09-27: implementado sem rollout. Testes novos: projeção R/A/D e digest,
leitura de histórico v1 (inclusive um log real do repositório em
`testdata`), recusa de schema 2 sem capability e de linhas malformadas,
transições, basis invalidada ou retirada, refs órfãs e ciclos, editorial que
não mascara semântica, invariância a Tasks/Final Report, CLI com e sem
capability, gate do lint e a tool MCP. Com o gate ignorando a capability, 5
testes reprovaram. `go test ./...`, `go vet ./...` e o `-race` da seleção
passam. `help_catalog.go` saiu dos artefatos: `amend` não tem entrada no
catálogo; o uso vem do próprio comando.

### Requirement trace
- R1 [satisfied] test:TestABMContractNodesProjectionCoversRADWithStableDigest test:TestABMContractNodesMigratesARealV1Log
- R2 [satisfied] test:TestABMContractNodesProjectionCoversRADWithStableDigest test:TestABMNodeTransitionsRequireAcknowledgement
- R3 [satisfied] test:TestABMNodeTransitionsEditorialCannotMaskSemantic test:TestABMContractNodesCLIAdoptedRecordsNodesAndRefusesMaskedChanges
- R4 [satisfied] test:TestABMNodeDigestIgnoresDerivedSections
- R5 [satisfied] test:TestABMNodeTransitionsInvalidatedAssumptionBlocksActiveDecision
- R6 [satisfied] test:TestABMContractNodesCLIAdoptedRecordsNodesAndRefusesMaskedChanges test:TestABMContractNodesMigratesARealV1Log
- R7 [satisfied] test:TestABMContractNodesRefusesSchema2WithoutCapability test:TestABMContractNodesCLILintRefusesSchema2WithoutCapability test:TestABMContractNodesMCPReturnsProjectionAndCapability

### Gaps conhecidos
Infraestrutura, amostra/usuários ou credenciais necessárias ao aceite deverão ser disponíveis na execução; ausência será reportada sem simulação de sucesso.

## 7. Final Report

### Escopo entregue
Somente especificação de implementação. Runtime, rollout e requisitos permanecem pendentes.

### Riscos residuais
Os riscos de implementação da seção Technical Plan não foram aceitos como entrega.

### Follow-ups
Nenhum desdobramento adicional nesta fase: o escopo pendente está nos requisitos desta spec. Se a implementação revelar trabalho fora deles, registrar owner, criticidade, prazo de triagem e disposição antes do closeout.
