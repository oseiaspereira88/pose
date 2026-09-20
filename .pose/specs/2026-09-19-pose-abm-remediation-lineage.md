---
slug: pose-abm-remediation-lineage
status: in-progress
created_at: 2026-09-19
completed_at:
supersedes:
depends_on: pose-abm-governance-outcomes
priority: 2
components: pose-mcp
task_type: feature
delivers: capability:remediation-lineage, surface:remediation-lineage-lint
---

# Spec: Lineage explícita de remediação

## 1. Intent

### Objetivo
Registrar e validar um vínculo `remediates` explícito entre entregas, e calcular
remediação numa janela madura apenas quando a população observável e a censura
forem conhecidas.

### Valor de negócio
Distinguir uma entrega que conserta outra de uma que apenas veio depois. Sem
isso, qualquer taxa de remediação mede ordem cronológica, não causa.

### Restrições
- Esta spec recebe R4 e R5 de [governance-outcomes](2026-09-19-pose-abm-governance-outcomes.md),
  que fechou com os dois diferidos para cá. O primeiro incremento implementa R1;
  a projeção da janela (R2/R3) permanece pendente.
- O vínculo é declarado, nunca inferido por similaridade lexical. Depender de
  parecença textual entre títulos foi rejeitado no desenho do programa.
- `depends_on` não é `remediates`: dependência não implica conserto.
- Janela incompleta é inconclusiva, e a censura precisa ser reportada junto com
  a taxa; ausência de dado não vira zero.

### Não-objetivos
Não inferir causalidade a partir de proximidade temporal, não ranquear pessoas,
agentes ou modelos, e não transformar a taxa observada num score agregado.

## 2. Requirements

### Funcionais e critérios de aceite
- R1: Registrar `remediates` apontando para spec ou finding, com categoria de um
  conjunto fechado — defect-fix, simplification, revert, requirement-change,
  planned-evolution — recusando refs órfãs e ciclos.
- R2: Calcular remediação numa janela de 30 dias apenas sobre a população que
  completou a janela, reportando censura e entregas excluídas por imaturidade.
- R3: Somente as categorias pertinentes entram na taxa; mudança de requisito não
  é defeito, e aceitação de risco é categoria própria, não defeito corrigido.

### Não-funcionais
Determinístico e offline. Nenhum serviço, índice ou banco novo.

### Segurança
Conteúdo do repositório é input não confiável. A projeção agrega por projeto,
processo ou banda, e nunca retorna principal, unidade de negócio ou identidade.

### Compatibilidade
Aditivo e opt-in. Specs sem `remediates` continuam válidas e fora da população.

## 3. Technical Plan

### Áreas afetadas
`pose-mcp/internal/pose` e a projeção de `pose stats governance` entregue por
governance-outcomes.

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-remediation-lineage.md
- created: .pose/adr/2026-09-20-explicit-remediation-links-in-spec-frontmatter.md
- created: pose-mcp/internal/pose/remediation_lineage.go
- created: pose-mcp/internal/pose/remediation_lineage_test.go
- created: pose-mcp/internal/cli/remediation_lineage_test.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/schema_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/schemas/v1/spec.schema.json
- modified: pose-mcp/schemas/v1/review-bundle.schema.json
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/assessments/integrations.md
- modified: .pose/state/components/pose-mcp.json
- modified: .pose/state/integrations.json

- modified: pose-mcp/internal/pose/governance_outcomes.go
- modified: pose-mcp/internal/cli/governance_stats.go
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json

Inventário inicial ampliado conforme o trabalho aconteceu, em vez de declarar
antes arquivos que ninguém havia escrito.

### Delivery targets
- capability:remediation-lineage module:pose-mcp/internal/pose profile:composed-capability entrypoint:pose-mcp/internal/pose/review_bundle.go
- surface:remediation-lineage-lint module:pose-mcp/internal/cli profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Mudanças de API/contrato
Um campo `remediates` no contrato de spec e a extensão da projeção existente.
Sem DSL, sem motor paralelo.

Incremento 1: frontmatter inline `remediates: spec:original@defect-fix` ou
`finding:rva-<16hex>/F1@simplification`, com múltiplos vínculos separados por
vírgula. As cinco categorias de R1 são fechadas. Identificar finding pela
atestação imutável, nunca pelo ID local sozinho. Validar refs e ciclos antes de
lint/DoR ou seal; incluir o campo no subject semântico e no digest legado apenas
quando não vazio. Ausência do campo preserva bytes e comportamento históricos.
Limitar o grafo a 256 specs/1024 vínculos, confinar leituras ao projeto e recusar
IDs ambíguos ou referências repetidas. Não inferir nada de `depends_on`.

### Dados e operação
Nenhuma persistência nova; a projeção continua reconstruível a partir dos
artefatos existentes.

### Riscos técnicos
A tentação de inferir o vínculo quando ele não é declarado. Um vínculo ausente
é desconhecido, e a spec precisa manter esse estado visível em vez de preenchê-lo.

### Rollout e reversão
Advisory primeiro, com a taxa publicada ao lado da cobertura. Reverter é deixar
de exigir o campo, sem reescrever dado histórico.

## 4. Tasks

- [x] Definir o contrato de `remediates` e seus casos negativos antes de codar.
- [x] Implementar registro, validação e projeção.
- [x] Cobrir refs órfãs, ciclos, categorias inválidas e janela imatura.
- [ ] Declarar alvo de entrega e fechar pelo gate POSE.

## 5. Decisions

- Contrato do incremento: [ADR](../adr/2026-09-20-explicit-remediation-links-in-spec-frontmatter.md).
- Contexto consumido: knowledge:adr-sealed-review-bundles-review; não omitir o
  novo vínculo da projeção semântica nem invalidar sujeitos sem opt-in.

- Data: 2026-09-19.
- Contexto: governance-outcomes precisava fechar com R4 e R5 sem evidência, e a
  própria spec instruía manter-se aberta até eles ou abrir sucessora.
- Opções consideradas: manter governance-outcomes aberta indefinidamente;
  retirar R4/R5 dela; abrir esta sucessora.
- Decisão: abrir esta sucessora e apontar o trace de lá para cá.
- Racional: manter a spec aberta parava um incremento pronto por causa de outro
  que nem começou, e retirar os requisitos apagaria a intenção registrada.
- Consequências: a taxa de remediação continua indisponível até esta spec ser
  entregue, e isso fica visível em vez de implícito.

## 6. Validation

### Estratégia
Planejamento anterior à implementação. Os comandos abaixo são entregas a criar.

| Cenário/requisitos | Comando planejado | Evidência esperada |
| --- | --- | --- |
| Refs órfãs, ciclos e categorias; R1 | `cd pose-mcp && go test ./internal/pose -run 'RemediationLineage' -count=1` | Cada caso negativo com blocker estável |
| Janela madura e censura; R2/R3 | `pose stats governance --json` | População, censura e exclusões reportadas |

Plano obrigatório do incremento 1, anterior ao código (módulo high):

| Cenário | Comando | Evidência esperada |
| --- | --- | --- |
| Parse, categorias, órfãos, ciclos diretos/indiretos, finding ausente, tamper e symlink; R1 | `cd pose-mcp && go test ./internal/pose -run RemediationLineage -count=1` | Recusas estáveis; fixtures válidas resolvem sem escrever |
| CLI real, ready-check, legado e ausência de vínculo | `cd pose-mcp && go test ./internal/cli -run RemediationLineage -count=1` | Lint recusa vínculo inválido; specs sem campo mantêm resultado |
| Campo alterado após seal e ordem dos vínculos | `cd pose-mcp && go test ./internal/pose -run RemediationLineage -count=1` | Subject fica stale por mudança material; ordem não altera o selo |
| Regressão e paridade gerada | `pose validate --module pose-mcp --strict` | Build/test/vet e produtor dedicado passam; nenhuma taxa fabricada |

### Checks de governança
- `pose lint-spec pose-abm-remediation-lineage --ready-check` antes de iniciar.
- `pose validate --strict --module pose-mcp` no escopo aplicável.
- `pose review verify spec:pose-abm-remediation-lineage` antes da transição.

| Projeção: censura, categorias, indisponibilidade, amostra; R2/R3 | `cd pose-mcp && go test ./internal/pose -run RemediationProjection -count=1` | Imatura censurada; categoria excluída visível e fora da taxa; ausência de contrato é indisponibilidade |

### Log de execução
2026-09-19: spec criada para receber R4 e R5 de governance-outcomes no
fechamento daquela. Nenhum requisito foi implementado ou declarado satisfeito.

2026-09-20, incremento 1: contrato `remediates` no frontmatter, com parse,
categorias fechadas, refs órfãs, ciclos, limites de grafo, confinamento de leitura
e identificação de finding pela atestação imutável. Coberto por
`RemediationLineage` em `internal/pose` e `internal/cli`.

2026-09-20, incremento 2: a projeção. O placeholder que governance-outcomes deixou
— `available: false`, `reason: no explicit remediation event contract is adopted` —
foi substituído por um cálculo sobre a população que completou a janela de 30 dias.

Três propriedades foram implementadas como recusas, porque cada uma era uma forma
de publicar um número lisonjeiro:

- Entrega recente demais para ter sido remediada é **censurada**, não contada como
  limpa. Sem isso a taxa melhora só por entregar.
- Entrega madura sem vínculo de entrada é **desconhecida**, não limpa. Nada no
  repositório afirma que ela não tinha defeito, então `unlinked_unknown` fica ao
  lado da taxa em vez de ser dobrado dentro dela.
- Categorias fora do conjunto contado continuam **reportadas** por categoria:
  `requirement-change` e `planned-evolution` aparecem, e não viram defeito.

A cobertura só pode ser `partial`, nunca completa, pela mesma razão: vínculo
ausente é desconhecido. E ausência do contrato é `unavailable` com motivo, não
taxa zero.

Injeção de defeito, quatro casos, cada um reprovando exatamente o caso que o
afirma: imaturas no denominador, toda categoria contando como defeito, ausência de
vínculo deixando de ser desconhecida, e ausência do contrato publicada como taxa
zero. A primeira injeção não compilou na primeira tentativa e foi refeita — uma
injeção que não compila não prova nada.

Medido em vez de assumido: o producer registrado nomeava apenas
`RemediationLineage`, então o corpus da projeção rodava no passo `test` do módulo e
não no check que é a evidência de integração declarada desta entrega. O `-run`
agora nomeia `RemediationProjection` também, e `validate --strict --module pose-mcp`
passa 9/9.

Neste repositório a projeção reporta `available=false` com motivo, porque nenhuma
entrega declara vínculo. Isso é o estado honesto e não um defeito: inferir o
vínculo a partir de `depends_on` ou de proximidade temporal é exatamente o risco
técnico que esta spec nomeou, e nenhum vínculo foi inventado para produzir número.

### Requirement trace

- R1 [satisfied] surface:remediation-lineage-lint evidence:integration
  check:abm-remediation-lineage-integration
  test:TestRemediationLineageCLI
  test:TestRemediationLineageReferencesAndCategories
  test:TestRemediationLineageCycleBoundsAndConfinement
  test:TestRemediationLineageFindingUsesImmutableAttestation
  test:TestRemediationLineageIsSemanticAndOrderIndependent — categorias fechadas,
  refs órfãs e ciclos recusados, finding identificado pela atestação imutável
- R2 [satisfied] capability:remediation-lineage evidence:integration
  check:abm-remediation-lineage-integration
  test:TestRemediationProjectionCensorsImmatureDeliveries
  test:TestRemediationProjectionCoverageIsNeverComplete — janela de 30 dias sobre a
  população madura, censura reportada, amostra insuficiente declarada
- R3 [satisfied] capability:remediation-lineage evidence:integration
  check:abm-remediation-lineage-integration
  test:TestRemediationProjectionCountsOnlyPertinentCategories
  test:TestRemediationProjectionWithoutLinksIsUnavailable — só as categorias
  pertinentes entram na taxa, as excluídas seguem visíveis, e ausência de contrato
  é indisponibilidade
O alvo de superfície é traçado por R1, cujo contrato é o que o lint recusa, e o de
capacidade por R2 e R3, que são a projeção.

### Gaps conhecidos
A população observável de remediação ainda não existe: o corpus histórico das
duas instâncias não contém nenhum finding, então não há classe positiva a medir
até que o contrato de julgamento explícito produza uma.

---

## 7. Final Report

### Escopo entregue
Nada. Esta spec é o destino declarado de dois requisitos diferidos.

### Riscos residuais
Os riscos da seção Technical Plan não foram aceitos como entrega.

### Follow-ups

Nenhum desdobramento adicional nesta fase: o escopo pendente são os próprios
requisitos desta spec.
