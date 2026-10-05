---
slug: pose-abm-causality-attestation
status: done
created_at: 2026-09-18
completed_at: 2026-09-27
supersedes:
depends_on: pose-abm-contract-nodes, pose-abm-atomic-start, pose-abm-progressive-review
priority: 2
components: pose-dist/pose-mcp
task_type: feature
delivers: governance:abm-causality-closeout
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
- created: .pose/specs/2026-09-27-pose-abm-causality-attestation.md
- created: pose-mcp/internal/pose/causality_closeout.go
- created: pose-mcp/internal/pose/causality_closeout_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/changelogs/unreleased/pose-abm-causality-attestation.md

Reconciliado em 2026-09-27: a causalidade estrutural básica (mapping de fato
material para base que alcança R/C, disposições com rationale, unknown não
cobrado) já foi entregue pelo R7 de `pose-abm-progressive-review` sob o
contrato `structural-causality`. Esta spec a completa sem tocar o contrato
existente; `review_plan.go` não muda e os schemas seguem aditivos em v1
(campos opcionais), sem `v2/`.

### Delivery targets
- governance:abm-causality-closeout module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

Produtor integration: check `abm-causality-closeout-integration` (`go test
./internal/pose ./internal/cli -run 'ABMCausalityCloseout' -count=1`).
Nenhuma instância adota a capability nesta entrega.

### Desenho
- Contrato `causality-closeout`, carimbado no bundle somente quando a policy
  adota `causality_closeout_version: 1`. Bundles sem o carimbo mantêm o
  veredito (R7). Com ele, o bundle sela a base (digest da projeção de
  contract nodes por spec do escopo) e a banda do plano (R1), e mudança
  material de R/A/D torna a review obsoleta (R4).
- Regras adicionais, só para bundles com o carimbo, sobre o contrato
  `structural-causality` existente: basis em premissa invalidada ou retirada,
  ou decisão retirada, bloqueia (R2/R4); mapping `mapped` exige rationale
  próprio, e a mesma justificativa repetida para três ou mais fatos na mesma
  base bloqueia como falta de proporcionalidade (R3); `accepted-risk` exige
  owner e prazo e não dispensa fato de contrato público ou de governança
  (R5); `not-applicable` com cobertura desconhecida bloqueia, porque unknown
  não é ausência (R5); banda elevated ou critical com fato material e nenhum
  critério que responda por estrutura bloqueia, sem exigir revisor humano, e
  a fixture trivial não ganha gate (R6).

### Mudanças de API/contrato
Mappings pertencem ao julgamento do bundle; observações não são editáveis pelo reviewer. Usar criterion de proporcionalidade para contestar cinco abstrações ligadas ao mesmo R sem razão suficiente. Nenhum pass por presença de ID.

### Dados e operação
Persistir somente o que o contrato exige; projeções são reconstruíveis. Documentar versionamento, idempotência, limites de execução, diagnóstico e proveniência de qualquer novo campo/evento.

### Riscos técnicos
Um grafo formalmente completo pode justificar uma decisão ruim. Corpus mantém o caso mapeado-mas-injustificado e depende de avaliação semântica, não de linter teatral.

### Rollout e reversão
Shadow comparativo seguido de enforcement por adoption. Manter baseline simples e registrar qualquer regressão de cerimônia como blocker do rollout.

## 4. Tasks

- [x] Confirmar dependências e revisar o desenho contra a release que será implementada.
- [x] Registrar delivery profile/target/entrypoint e inventário exato antes de iniciar código.
- [x] Implementar os requisitos em incrementos coesos e incluir os casos negativos de Validation.
- [x] Atualizar interfaces, docs e scaffolds que consomem o contrato, sem drift de vendor.
- [x] Executar matriz aplicável, obter review explícito e anexar evidência por R-ID.
- [x] Fechar somente pelo gate POSE, com riscos/follow-ups dispostos e resultado de composição atual.

## 5. Decisions

- Date: 2026-09-27. Implementado sem rollout, como contract-nodes e
  atomic-start: o contrato só é carimbado sob `causality_closeout_version`,
  que nenhuma instância adota; o shadow comparativo e o enforcement ficam
  para depois do stop/go do piloto.

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

2026-09-27: implementado sem rollout. Testes novos: mapping justificado e
vivo aceito; basis em premissa invalidada e em decisão retirada recusados;
mapping sem rationale e motivo colado em três fatos recusados, dois motivos
iguais aceitos; risco aceito sobre contrato público e sem owner/prazo
recusados; not-applicable sob cobertura desconhecida recusado; banda
elevated/critical sem critério recusada, baseline e fixture trivial sem
gate, nenhuma exigência humana; carimbo só com adoção, base selada e
bundle obsoleto após mudança material; as regras rodam no caminho de
atestação só para bundle carimbado; `--mapping` com owner e prazo. Com as
regras desligadas, 6 testes reprovaram. `go test ./...`, `go vet` e o
`-race` da seleção passam.
Em `afe2c5a` a matriz completa passou 32/32, com o novo check
`abm-causality-closeout-integration`; `artifact-check` com 13 claims e sem
erros, `surface-check` sem achados. Bundle `rvb-e561ea73a2ab6915` aprovado
pela atestação `rva-aceac3a813e2aaa7`, registrada pelo agente com
autorização explícita do usuário para autoatestar. As reviews seladas
invalidadas pela matriz foram renovadas.

### Requirement trace
- R1 [satisfied] test:TestABMCausalityCloseoutIsStampedOnlyOnAdoptionAndSealsTheBasis
- R2 [satisfied] test:TestABMCausalityCloseoutRefusesADeadBasis test:TestABMCausalityCloseoutAcceptsAJustifiedLiveMapping
- R3 [satisfied] test:TestABMCausalityCloseoutSeparatesCompletenessFromAcceptance
- R4 [satisfied] test:TestABMCausalityCloseoutRefusesADeadBasis test:TestABMCausalityCloseoutIsStampedOnlyOnAdoptionAndSealsTheBasis
- R5 [satisfied] test:TestABMCausalityCloseoutBoundsAcceptedRisk test:TestABMCausalityCloseoutUnknownIsNotAbsence test:TestABMCausalityCloseoutCLIMappingCarriesRiskOwnerAndDate
- R6 [satisfied] test:TestABMCausalityCloseoutRaisesElevatedObligationsOnly
- R7 [satisfied] test:TestABMCausalityCloseoutRunsOnlyForStampedBundles

### Gaps conhecidos
Infraestrutura, amostra/usuários ou credenciais necessárias ao aceite deverão ser disponíveis na execução; ausência será reportada sem simulação de sucesso.

## 7. Final Report

### Escopo entregue
Contrato `causality-closeout`, carimbado só sob `causality_closeout_version`,
que sela banda e base R/A/D e acrescenta ao `structural-causality` as regras
de base viva, justificativa própria e proporcional, risco aceito com limites,
unknown não tratado como ausência e obrigação elevada sem gate trivial.
Nenhuma instância adota a capability.

### Riscos residuais
O limite de proporcionalidade (três fatos com o mesmo motivo) é uma
heurística objetiva, não avaliação semântica; um grafo completo e bem
redigido ainda pode justificar uma decisão ruim, e isso continua sendo
julgamento do revisor. O shadow comparativo previsto no rollout não foi
executado, porque a capability não está adotada.

### Follow-ups
- [done] Adotado no pose-dist em 2026-10-05 por decisão do maintainer (action request act-08a9fd2d9a0e50bb): duas medições (sem e com `structural-materiality@1`), quatro remediações no motor (pose-validation-check-additions-are-not-material, pose-causality-closeout-adoption-cutoff, pose-attest-refuses-what-verify-rejects, pose-governed-capabilities-default-on-new-instances) e então `pose adopt causality-closeout --date 2026-10-06 --apply`. Instâncias novas adotam na instalação. Item original: rodar o shadow comparativo do causality-closeout sobre os corpora e adotar `causality_closeout_version` depois do stop/go do piloto. (owner:@pose-maintainers crit:medium review:2026-11-01)
