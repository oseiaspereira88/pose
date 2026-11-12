---
slug: pose-closeout-review-policy-by-cost-benefit
status: draft
created_at: 2026-11-11
completed_at:
supersedes:
depends_on:
remediates:
priority: 2
components: pose-mcp
task_type: feature
delivers: capability:closeout-review-policy-by-cost-benefit
---

# Spec: Política de revisão do fechamento de milestone e de roadmap por custo e ganho

## 1. Intent

### Goal

A instância escolhe, por tipo de escopo, se o fechamento de um milestone e o de um roadmap exigem revisão independente: `never`, `when-justified` ou `always`. O `pose close` do milestone e o do roadmap continuam obrigatórios em todos os casos; muda só a exigência da revisão do escopo.

### Business value

Dados do pose-dist em 2026-11-11: 7 runs de revisão de milestone e 6 de roadmap, todas de `pose-shared-runtime`. As revisões acharam coisa de substância (a chave do memo ignorava a configuração global do Git, um pedido do runtime sem identidade, a medição que o exit gate exige não existia como descrita) e deriva entre ADR, roadmap e specs, mas só em um roadmap com ADR e vários membros. Num milestone de um só membro a revisão repete a da spec, e o loop do B31 foi parado por isso em 2026-11-08. Hoje a revisão é sempre exigida; o custo (runs, rodadas, bundle próprio) não se justifica em escopos pequenos e o ganho existe em escopos compostos.

### Constraints

- O padrão da instância é `never` para os dois tipos (decisão do mantenedor); `always` exige revisão própria inclusive para milestone de um membro (não equivale à herança legada).
- A revisão de cada membro (spec) continua exigida como hoje; a dispensa só vale para a revisão do escopo agregado.
- Fechamentos já selados continuam válidos pela regra com que foram selados: a política vale para escopos fechados depois dela.
- Dispensar nunca é silencioso: o fechamento registra que a revisão não foi exigida, por qual política e por qual regra.

### Non-goals

- Mudar a revisão de spec.
- Remover o `pose close` de milestone e de roadmap.

## 2. Requirements

- R1: ADR primeiro: onde a política vive (arquivo de política versionado da instância), a semântica dos três valores, o padrão `never`, a compatibilidade com escopos já selados, e por que `when-justified` é uma regra determinística e não um juízo do agente.
- R2: Chaves `milestone_review_required` e `roadmap_review_required`, com os valores `never`, `when-justified` e `always`, lidas pelo motor; valor desconhecido é recusado.
- R3: `when-justified` para milestone: a revisão é exigida quando o milestone tem mais de duas specs canônicas distintas. Para roadmap: exigida quando contém mais de uma spec canônica distinta e está associado a pelo menos uma ADR (campo do roadmap ou ADR que o cita). Caso contrário a revisão do escopo é dispensável e a dos membros vale.
- R4: Quando a revisão não é exigida (`never`, ou `when-justified` sem os critérios), `pose close` do milestone e do roadmap continua exigido e passa sem a revisão do escopo, e o resultado registra `review: not-required`, a chave, o valor e o motivo (por exemplo `members=1, threshold>2`).
- R5: Uma revisão feita mesmo quando não exigida continua válida e é verificada como hoje (a instância pode revisar por escolha).
- R6: `pose setup` e o comando de configuração mostram e permitem mudar as duas chaves; `pose doctor` avisa quando um escopo aberto passa a exigir revisão por causa de uma mudança de política.
- R7: Os fechamentos existentes do pose-dist e das instâncias não mudam de estado ao instalar a política (nenhum `pose check` novo falha).

## 3. Technical Plan

### Authority, baseline and scope

Planejamento local somente, em 2026-11-12. `pose context --task spec:pose-closeout-review-policy-by-cost-benefit --json` resolveu a autoridade existente `xref:proj.pose-dist/spec:pose-closeout-review-policy-by-cost-benefit`, status `draft`, revisão `f6fa91fee8aa7f2abbc314bb077bb0aa1d824a1b`; nenhuma autoridade nova ou escrita qualificada. Não iniciar nem adotar a política nesta execução. O estado agregado consultado está stale (`commits_exceeded`); usar fontes atuais, sem refresh.

Base arquitetural aceita: [ADR de custo e ganho](../adr/2026-11-12-closeout-review-policy-by-cost-and-benefit.md), aprovação explícita do mantenedor em 2026-11-12. Sua aceitação não satisfaz implementação de R1-R7. Consultados `knowledge:adr-component-aware-review-plans-review` e `knowledge:adr-sealed-review-bundles-review`; TTL terminou em 2026-11-11, portanto são contexto histórico, conferido no código atual, não evidência vigente.

Módulo `pose-mcp`: criticality `high`, domain `go`, profile `baseline` em module-metadata. Aplicar feature workflow, backend-go, security, documentation-style e delivery-evidence. A matriz usa `moduleOverrides.pose-mcp`, substitui checks padrão e exige build, unit e integrações; vet é opcional na matriz, mas obrigatório pela regra Go. Não há necessidade de dependência externa nova.

Hoje `ReviewPolicy`/loader e `reviewRequiredForScope` estão em `review_closeout.go`; `roadmaps.go` e `review_bundle.go` implementam herança e reached; `review_plan.go` transporta exit gates; `review_policy_baseline.go` restaura proteção. `cmdCloseLocalWithGuardedStore` apenas verifica milestone, sem persistir uma transição; roadmap escreve status. `artifact_ref.go` aceita spec/roadmap/milestone, não ADR. `pointers.go` já aceita `adr:<filename.md>` local. Não tratar nenhum desses pontos como suporte já existente à decisão nova.

### Policy and compatibility (R1, R2, R6, R7)

- Planejar schema de review policy **5** (o maior atual é 4); reservar esse número novamente antes de implementar. O loader antigo recusa 5. Schema 5 exige ambas as novas chaves e preserva os contratos previamente adotados: qualified refs e transfer continuam opcionais, com versões 0/ausente ou 1 e transfer 1 exigindo qualified refs 1. Não habilitar refs/transfer ou overlays por adotar o par; schema 5 desacopla reconhecimento da versão da policy da adoção dessas capacidades. Manter os pré-requisitos existentes de component-aware/bundles e validar sua combinação sem os ativar silenciosamente. Atualizar também os probes manuais de schema do resolver; não mudar schema de profiles, planos ou bundles históricos por consequência automática.
- Schemas 1-4 sem ambas as chaves, e instâncias sem policy, conservam exatamente a regra legada, inclusive enabled/cutoffs, singleton e `unreviewed_milestones`. Chave nova em schema antigo, uma chave só, null, vazio, tipo errado, duplicação JSON e enum desconhecido são inválidos antes de planejar ou escrever. Schema 5 sem ambas as chaves é inválido.
- Instalação nova e adoção explícita recebem ambas `never`. Update entrega o motor e a oferta de adoção; não converte policy existente. Não tocar a policy desta instância como parte da feature. Preservar configurações independentes, datas e pins existentes; toggles antigos de outras capacidades não podem baixar schema 5 ou remover o par. A adoção aqui não altera a autoridade das operações federadas já configuradas.
- Reutilizar catálogo de capacidades e `pose adopt` como comando de configuração, sem criar outro comando genérico de settings. Planejar capability `closeout-review-policy`, com tratamento próprio do par (o registro simples VersionKey não representa dois enums). `pose adopt closeout-review-policy --apply` adota com `never/never`; preview sem apply mostra legado efetivo, novos valores e impactos. Planejar flags `--milestone-review-required` e `--roadmap-review-required` nesse caminho: exigir as duas quando qualquer uma for informada, tanto na adoção quanto em mudanças posteriores. Recusar essas flags com outra capability, --off, --decline ou --defer. Não oferecer rollback por remoção de chaves/schema.
- `pose setup` e `--json` mostram ambos os valores, a condição legada e a diferença proposta; a interação escolhe o par e chama a mesma operação de configuração. Sem entrada/confirmação não adotar. Instalação nova usa o mesmo produtor de defaults, sem datas copiadas do distribuidor.

### Canonical references and ADR associations (R3)

Contrato novo proposto aqui, ainda não implementado; fixar fixtures antes do parser:

1. Aceitar em `- specs:` os formatos existentes `slug`, `spec:slug` e `xref:project-id/spec:slug`. Vincular local ao project_id declarado do proprietário do roadmap; resolver por `ArtifactResolver`, seguir somente redirects verificados e deduplicar pela identidade final `xref:<project-id>/spec:<slug>`. Ordenar lexicalmente essas strings. Não deduplicar por slug sem projeto, nome de arquivo ou checkout. Duplicatas entre milestones entram uma só vez na união do roadmap. `depends_on`, `after` e `consumes` continuam gates, não viram membros automaticamente.
2. Adicionar frontmatter flat opcional `adrs: [adr:2026-11-12-closeout-review-policy-by-cost-and-benefit.md]` no roadmap. O valor de cada pointer é exatamente um basename `.md` dentro de `.pose/adr/` do projeto proprietário. Reusar contenção do resolver de pointers, acrescentando leitura real e validação de arquivo regular; rejeitar subdiretórios, traversal, symlinks externos, URI e qualificação `xref:.../adr:...` (não suportada). A identidade normalizada da associação é a tupla `(project_id, adr basename)`; ordenar/deduplicar pela tupla, sem inventar ArtifactRef ADR.
3. Reconhecer também frontmatter flat opcional no ADR: `roadmaps: [roadmap:program]` ou `roadmaps: [xref:proj.pose-dist/roadmap:program]`. Resolver o roadmap explicitamente no projeto do ADR e comparar identidade canônica exata. Fazer inventário limitado e completo de `.pose/adr/*.md` do proprietário do roadmap no mesmo read view; não varrer registros globais nem procurar backlinks em outros projetos. Referência cross-project declarada num ADR local só associa esse ADR a esse alvo exato, sem autorizar leitura/escrita adicional.
4. Apenas metadata estruturada conta; links Markdown, título, palavras no corpo, `source_refs` e nome parecido não contam. A ADR aceita usa corpo livre; sua aceitação não cria associação com algum roadmap. Não exigir status Accepted da ADR para contar: o critério aprovado é associação resolvida, não aprovação da associação.
5. União das duas direções: uma associação resolvida basta para o predicado ADR. Declaração inválida, alvo ausente/ambíguo, leitura incompleta ou inventário indisponível impede provar ausência e bloqueia dispensa por `when-justified`; não escolher o primeiro arquivo legível. Membros desconhecidos nunca contam como zero. Aplicar budgets e autorização existentes; desconhecimento é diagnóstico com refs e condição de satisfação, não fallback para never. `never` e `always` não precisam inventariar ADRs para decidir, mas não dispensam resolução dos membros/gates.

A escolha mínima é estender somente metadata de associação; não ampliar a gramática global ArtifactRef com ADR nem inferir associações em prose. Backlinks remotos por busca global não fazem parte deste contrato. Se o mantenedor exigir esse alcance, obter decisão arquitetural antes do parser; a ADR aceita não define esse inventário. Não há essa exigência nesta base.

### One evaluator and precedence (R2-R5)

Criar avaliador de domínio versionado `closeout-review@1`, compartilhado por review-plan, ReviewCheck/VerifyReviewBundle, closeout state/plan, reached/next action, doctor e federation. Produzir `required`, `policy_key`, `policy_value`, `evaluator_version`, razão estável, contagem e refs canônicas, associações resolvidas, overrides de proteção e diagnósticos. Projeções só renderizam a resposta; não reimplementam limiares.

| Modo adotado | Milestone | Roadmap |
| --- | --- | --- |
| never | Não exige revisão própria | Não exige revisão própria |
| when-justified | Exige se distinct_specs > 2 | Exige se distinct_specs > 1 AND resolved_adrs > 0 |
| always | Exige revisão própria mesmo com 1 membro | Exige revisão própria mesmo com 1 membro |

Ordem: verificar história terminal selada primeiro; para escopo aberto validar schema/resolução, calcular a regra, unir obrigações do baseline protegido e mandatory-human, verificar qualquer revisão voluntária e seu veto, validar membros/arestas/exit gates, então permitir close explícito. `enabled:false`, cutoff antigo, singleton ou `unreviewed_milestones` não podem anular `always` adotado ou proteção mais forte. Nunca usar um early return de dispensa para ocultar rejeição autêntica, história ilegível ou revisão voluntária inválida. Revisão voluntária deve passar o mesmo verificador, freshness, assinatura e independência da obrigatória; requests-changes/rejected só saem pela remediação/supersessão existente. Não criar uma aprovação de revisor para representar dispensa.

### Explicit close and historical state (R4, R5, R7)

- Em escopo adotado aberto, membros terminais significam reached/eligible, não terminal. `pose close milestone:program/core` e `pose close roadmap:program` seguem sintaxe existente sem --apply/--resume; não ampliar os modos de close de spec. Next actions, readiness de dependentes, attention e aceitação federada exigem receipt para esses novos fechamentos, inclusive quando revisão própria aprovada ou dispensada.
- Criar receipt schema 1 em `.pose/closeouts/milestone/<roadmap>/<milestone>.json` e `.pose/closeouts/roadmap/<roadmap>.json`, com escopo canônico, instante, subject/input digests, snapshot da policy e decisão do avaliador, membros/associações resolvidos e refs de revisão opcional/obrigatória. Para dispensa persistir literalmente `review: not-required`, chave, valor, versão e razão com counts (`members=1, threshold>2`, por exemplo); uma revisão voluntária aprovada fica referenciada à parte. Para revisão exigida registrar `review: required` e evidência verificada, nunca um revisor inventado.
- Publicar receipt via writer guard/atomic publication existentes, com rechecagem dos inputs e dos gates na região protegida. Fechar roadmap ainda altera status; receipt é o commit marker terminal e só é publicado após status. Falha entre essas publicações fica explicitamente incompleta e repetição verifica os mesmos inputs antes de concluir, sem fabricar sucesso. Recibo existente igual torna repetição somente verificação; diferente/corrompido não pode ser sobrescrito para limpar um veto. Não incluir o próprio receipt/status na identidade semântica que ele atesta.
- Receipt não substitui evidência de entrega ou revisão de membro. Preservar exit gates mesmo quando não há revisão agregada; resultado negativo de gate continua bloqueante, sem auto-attest de juízo humano. Mudança de composição/policy antes de close recalcula elegibilidade; aprovação de bundle aberto com inputs relevantes alterados precisa supersessão como hoje.
- Sem receipt, história terminal anterior à adoção é verificada pelo contrato/plano selado existente, inclusive milestones herdados de roadmap selado. Nunca inferir adoção pela data ou preencher receipts retroativos. Um mero milestone reached sem prova terminal prévia continua aberto na adoção. Done não permite esconder corrupção ou veto histórico. Fechamento novo usa receipt selado; mudanças posteriores da policy não reavaliam a decisão histórica, mas mudanças semânticas e integridade continuam verificadas pelas regras existentes.
- `doctor` mostra escopos abertos que agora devem revisão, por política ou composição: motivo atual, count/ADR, regra anterior quando demonstrável por inputs registrados e ação necessária. Sem snapshot anterior, declarar comparação desconhecida; não inventar transição. Close continua sendo o gate autoritativo.

### CLI, MCP and federation composition

Encadear `cmd/pose/main.go -> internal/cli/review_closeout.go -> Store/evaluator/receipt verifier`; `internal/mcpserver/server.go -> Store` para `pose_review_plan`, `pose_closeout_state`, review verification e aceitação federada. Mesmos modos, counts, veto, resolução e diagnóstico em JSON/texto/MCP. MCP permanece nas permissões atuais; não adicionar ferramenta de configuração ou close write. CLI faz a configuração e o close; MCP observa resultado e próxima ação.

Na federação, avaliar o escopo na sua autoridade, com bindings/autorização/pins/revisão selecionada existentes. Não usar default never do consumidor para dispensar review do produtor. Manifesto novo vincula receipt e decisão; produtores legados continuam no verificador legado. Unknown/unauthorized/transfer-in-progress/stale source e rejeição remota bloqueiam; aliases de redirects verificados deduplicam. Usar somente projetos sintéticos temporários nos testes, nunca outros checkouts reais.

### Artifacts

Lista prevista para implementação futura; somente a primeira linha pode ser escrita nesta tarefa. A ADR aceita já existe e é referência, não um arquivo a recriar. Não adotar `.pose/policy/review.json` local. Artefatos de runtime dos testes ficam em `t.TempDir`, não no repositório.

- modified: .pose/specs/2026-11-11-pose-closeout-review-policy-by-cost-benefit.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_policy_baseline.go
- modified: pose-mcp/internal/pose/roadmaps.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/federated_acceptance.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/state.go
- modified: pose-mcp/internal/pose/capability_catalog.go
- created: pose-mcp/internal/pose/closeout_review_policy.go
- created: pose-mcp/internal/pose/closeout_receipt.go
- created: pose-mcp/internal/pose/closeout_adr_associations.go
- created: pose-mcp/internal/pose/closeout_review_policy_test.go
- created: pose-mcp/internal/pose/closeout_receipt_test.go
- created: pose-mcp/internal/pose/closeout_adr_associations_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/setup.go
- modified: pose-mcp/internal/cli/adopt.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/help.go
- created: pose-mcp/internal/cli/closeout_review_policy_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- created: pose-mcp/internal/mcpserver/closeout_review_policy_test.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/schemas/v1/review-plan.schema.json
- created: pose-mcp/schemas/v1/closeout-receipt.schema.json
- modified: POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/templates/roadmap.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/roadmap.md
- created: .pose/starts/pose-closeout-review-policy-by-cost-benefit.json
- created: .pose/changelogs/unreleased/pose-closeout-review-policy-by-cost-benefit.md

Atualizar index.go apenas para projetar metadata nova de roadmap, não regenerar indexes nesta tarefa. Atualizar review-plan schema com a decisão adicional e closeout-receipt schema novo; omitir campos novos em payload histórico sem reserializar bundle selado. Validar o par local/template distribuído no fluxo normal de implementação. Se reconciliação identificar outro caminho necessário, emendar esta lista antes de alterá-lo.

### Delivery targets

- capability:closeout-review-policy-by-cost-benefit module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Produção composta exige integração atravessando CLI e MCP até o avaliador e receipt verificado, mais teste de federação e instalação sintética. Registrar check permanente `closeout-review-policy-integration`, evidenceClass `integration`, required, em moduleOverrides.pose-mcp; comando no módulo: `go test ./internal/pose ./internal/cli ./internal/mcpserver -run '^TestCloseoutReviewPolicy' -count=1`. Os novos testes usam esse prefixo e subtestes da tabela abaixo. Build/unit isolados não provam esse alvo.

## 4. Tasks

- [x] Registrar a ADR (R1) e a aprovação explícita do mantenedor em 2026-11-12; sem afirmar implementação.
- [ ] Antes de implementar, resolver novamente contexto local, confirmar schema 5 disponível, executar discovery exigido e start autorizado em outra execução; manter baseline de história legada em fixtures.
- [ ] Incremento 1 — R1/R2/R6/R7: fixtures de loader, schema 5, catálogo/adopt/setup e defaults distribuídos; validar Schema/Adoption; nenhum update silencioso de policy.
- [ ] Incremento 2 — R3: fixtures de aliases, redirects e metadata ADR, avaliador comum e integração com plano/proteção; validar Members/Associations/Thresholds/Protection.
- [ ] Incremento 3 — R4/R5/R7: receipt guardado, close explícito, reached/attention/state e verificação terminal histórica; validar Receipt/Veto/History/Publication. Não deixar membro final fechar agregado implicitamente.
- [ ] Incremento 4 — R2-R7: projeções CLI/MCP/federação, doctor, indexer e schema JSON; validar Parity/Federation/Doctor e smoke de instalação temporária.
- [ ] Incremento 5 — Atualizar manuais/templates e matriz; executar checks completos proporcionais ao risco, artifact/surface-check e trace real antes de solicitar revisão independente da implementação.
- [ ] Fazer revisão e closeout futuros com evidência vigente, follow-ups e changelog; só então mudar status/completed_at. Este planejamento não executa essa etapa.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7
- Selected option: ADR [Closeout review policy by cost and benefit](../adr/2026-11-12-closeout-review-policy-by-cost-and-benefit.md), Accepted, aprovação explícita em 2026-11-12.
- Rationale: separar fechamento obrigatório de revisão agregada com limiares determinísticos, adoção explícita e preservação do histórico selado.
- Consequences: schemas sem chaves continuam legados; adopting/new default never; always inclui singleton; veto e proteção não são dispensados.

### Decision D2
- Basis: R2, R3, R4, R6, R7
- Selected option: planejar schema 5, metadata flat `adrs`/`roadmaps`, identidade canônica do resolver existente e receipt separado. Usar catálogo/adopt/setup existentes para o par de enums.
- Rationale: ArtifactRef hoje não resolve ADR e milestone close não persiste transição; esses contratos precisam implementação explícita. Evitar novo tipo global de identidade, heurística de prose e framework novo de configuração.
- Consequences: os formatos e caminhos acima são propostas de implementação dentro da ADR aceita, não fatos entregues. Confirmar disponibilidade do número de schema antes do start. Se houver demanda por descoberta de backlinks remotos ou se outra mudança já consumir schema 5, resolver essa escolha antes da implementação, emendar a spec e, se alterar o contrato aceito, obter decisão do mantenedor. Nenhuma aprovação adicional da ADR base está pendente.

## 6. Validation

### Risk and execution boundary

Risco alto: dispensa indevida de autoridade, reinterpretação de aprovação histórica e divergência CLI/MCP/federação. Nesta execução executar somente lint da spec e leituras limitadas. **Todos os comandos abaixo são planejados, não executados**, exceto os registrados em Planning checks. Testes novos ainda não existem: cada comando filtrado só é evidência quando todos os testes/subtestes previstos existirem, rodarem e não houver `no tests to run`. Comandos Go partem de `pose-mcp/`; comandos `pose` partem da raiz. Fixtures criam instâncias sintéticas com relógio fixo, sem rede, credenciais reais ou escrita fora de t.TempDir.

### Risk-based validation table

| Classe / requisito / cenário | Comando exato planejado | Evidência esperada |
| --- | --- | --- |
| Required unit/contract R2/R7 — schemas 1-4 sem keys, 5 com par; parcial, null, duplicado, enum/tipo inválido, versão velha com key nova, downgrade | `go test ./internal/pose -run '^TestCloseoutReviewPolicySchema$' -count=1 -v` | Legado idêntico; entradas inválidas falham antes de plano/write; fixture do loader antigo recusa 5. |
| Required unit R3 — milestone 0/1/2/3 e roadmap 0/1/2 distinct specs, com/sem ADR; todos os três modos | `go test ./internal/pose -run '^TestCloseoutReviewPolicyThresholds$' -count=1 -v` | Limites estritos >2 e >1 AND ADR; always não herda singleton; never só dispensa agregado. |
| Required unit/contract R3 — bare/typed/qualified local, mesmo slug em dois projetos, duplicates/redirects entre milestones | `go test ./internal/pose -run '^TestCloseoutReviewPolicyMembers$' -count=1 -v` | União ordenada de identidades finais; aliases contam uma vez, projetos distintos duas; ciclo/transfer pendente/identidade ambígua bloqueiam. |
| Required unit/contract R3 — adrs forward, roadmaps back, união/dedup; prose, traversal, symlink, faltante/ilegível, metadata malformada | `go test ./internal/pose -run '^TestCloseoutReviewPolicyAssociations$' -count=1 -v` | Só syntax declarada resolve; leitura incompleta não prova ausência; erro não concede dispensa; sem lookup global. |
| Required security R2/R5 — baseline mais forte, enabled/cutoff/declaration, mandatory-human, revisão opcional negativa/expirada/ilegível | `go test ./internal/pose -run '^TestCloseoutReviewPolicy(Protection|Veto)$' -count=1 -v` | Política fraca não autoaprova; negativo autêntico mantém veto até remediação existente; humano não vira máquina. |
| Required integration R4 — reached sem receipt, close explícito em todos os modos, repetição, optional approval | `go test ./internal/pose ./internal/cli -run '^TestCloseoutReviewPolicyReceipt$' -count=1 -v` | Elegível não é terminal; receipt contém policy/version/count/reason e not-required; approved optional preservado; close repete sem fabricar revisão. |
| Required negative/publication R4/R7 — mudança de input após gate, corrida, permissão/lock negado, I/O falha e falha entre status e receipt | `go test -race ./internal/pose ./internal/cli -run '^TestCloseoutReviewPolicyPublication$' -count=1 -v` | Sem terminal falso, sem overwrite de história; erro tipado/incompleto, recuperação determinística com rechecagem. |
| Required regression R7 — legacy sealed singleton/multi/roadmap, policy/composição mudada; história corrompida | `go test ./internal/pose ./internal/cli -run '^TestCloseoutReviewPolicyHistory$' -count=1 -v` | Verificação usa plano/policy selados, bytes históricos intactos; corrupção continua bloqueante e não pede backfill. |
| Required integration R6/R7 — install, update, preview/adopt/config/setup, par inválido e sem confirmação | `go test ./internal/pose ./internal/cli -run '^TestCloseoutReviewPolicyAdoption$' -count=1 -v` | New/adopting never/never; update legado sem mudança; preview/read-only não escreve; par/schema publicados juntos sob guard. |
| Required integration R6 — aumento de obrigação por policy ou composição e comparação sem snapshot | `go test ./internal/cli -run '^TestCloseoutReviewPolicyDoctor$' -count=1 -v` | Escopo aberto aparece com motivo/ação; sealed done não reabre; anterior desconhecido é exposto. |
| Required contract R2-R7 — CLI text/JSON e MCP review-plan/state/verifier sobre fixtures iguais | `go test ./internal/pose ./internal/cli ./internal/mcpserver -run '^TestCloseoutReviewPolicyParity$' -count=1 -v` | required/counts/reasons/diagnostics e next close iguais; nenhum write por leitura MCP; schema JSON confere. |
| Required federation/security R3-R7 — produtor legado/adotado e consumidor com policy diferente; unauthorized/unavailable/stale/transfer pendente | `go test ./internal/pose ./internal/cli ./internal/mcpserver -run '^TestCloseoutReviewPolicyFederation$' -count=1 -v` | Fonte decide revisão; receipt/manifest ligados à revisão selecionada; pins/authorization/arestas preservados; unknown e veto remoto bloqueiam. |
| Required negative R2-R7 — cancelamento/timeout/dependência indisponível e inventário limitado excedido | `go test ./internal/pose ./internal/cli ./internal/mcpserver -run '^TestCloseoutReviewPolicyUnavailable$' -count=1 -v` | Erro/unknown, nunca not-required por falta de dados; nenhuma receipt ou adoção parcial. Sem provider remoto necessário; fallback só o legado explicitamente não adotado. |
| Required e2e/smoke R2-R7 — binário e instalação temporária, configurar cada modo, fechar membros e agregado, verificar estado | `go test ./internal/cli -run '^TestCloseoutReviewPolicySmoke$' -count=1 -v` | Harness constrói CLI em tempdir, instala fixture, exerce flags/close reais e receipt; prova entrada de produção até verificador, sem projetos reais. |
| Required permanent integration — todo prefixo novo | `go test ./internal/pose ./internal/cli ./internal/mcpserver -run '^TestCloseoutReviewPolicy' -count=1` | Check closeout-review-policy-integration verde e todas as famílias acima presentes; não aceitar filtro vazio. |
| Required existing regressions — herança, reached, proteção, história e aceitação federada | `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1 -timeout=30m` | Fixtures legadas seguem verdes; fixtures adotadas testam semântica nova sem enfraquecer asserts legados. |
| Required module baseline / build/security/contract | `pose validate --strict --module pose-mcp --report --report-task test-plan-baseline-pose-closeout-review-policy-by-cost-benefit` | Todas as checks required da matriz passam, inclusive integração permanente; relatório contém comandos/outcomes/classes vigentes. |
| Required Go rule | `go vet ./...` | Sem achados bloqueantes; executar mesmo sendo optional na matriz. |
| Required delivery/structure (raiz) | `pose check --strict` | Estrutura e matriz válidas; sem falha nova nos fechamentos históricos. |
| Required contract assessment futuro (raiz) | `pose assess integrate` | Impacto CLI/MCP/federação documentado, sem contrato divergente. |
| Required artifact gate futuro (raiz) | `pose artifact-check --spec pose-closeout-review-policy-by-cost-benefit --strict` | Claims reconciliadas com observações dos commits atribuídos; emendar lista se houver divergência real. |
| Required composition futuro (raiz) | `pose surface-check --spec pose-closeout-review-policy-by-cost-benefit --strict` | Alvo capability ligado ao entrypoint e integração atual; sem alegar entrega com unit/build apenas. |

Não criar check permanente para cada subteste: uma integração registrada com o prefixo novo é suficiente; preservar checks existentes de milestone-review-proportional, milestone-closes-when-reached, review-policy-adoption, policy-scope-validation, federated-roadmap-acceptance e done-spec-keeps-its-sealed-plan. Nenhuma atualização da matriz nesta tarefa de planejamento.

### Requirement trace (planned only)

R1: base ADR aceita; implementação/documentação pendentes. R2: Schema/Adoption/Parity planejados. R3: Thresholds/Members/Associations/Federation planejados. R4: Receipt/Publication/Smoke planejados. R5: Protection/Veto planejados. R6: Adoption/Doctor planejados. R7: History/regressões planejados. Nenhum requisito marcado satisfied; inserir referências reais e evidência de execução somente após implementação.

### Planning checks

Executado em 2026-11-12: `/tmp/pose-redaction-cli context --task spec:pose-closeout-review-policy-by-cost-benefit --json` resolveu a autoridade local existente. `/tmp/pose-redaction-cli lint-spec pose-closeout-review-policy-by-cost-benefit --strict` passou (7 requisitos, zero seções vazias, zero falhas; trace sem satisfação de implementação). `/tmp/pose-redaction-cli lint-spec pose-closeout-review-policy-by-cost-benefit --ready-check` passou (`spec.ready=true`, zero falhas). `git diff --check` passou. Esses checks validam o planejamento, não a entrega. Não executar discovery, assessments, full validation, indexação, start, revisão independente ou close como parte deste planejamento.

---

## 7. Final Report

### Delivered scope

**Planned only.** Plano pré-implementação; status permanece draft e completed_at vazio. A ADR está Accepted, mas nenhum código, policy adotada, requisito implementado, revisão independente ou evidência de entrega é afirmado. Somente este arquivo é alterado pela tarefa atual.

### Residual risks and next owner

Implementação futura por mantenedor/implementador de proj.pose-dist: conferir reserva de schema, preservar história/proteção e executar a tabela de risco antes da revisão. Metadata ADR e receipt descritos são contratos novos planejados. Não há decisão da ADR base a reapresentar ao mantenedor; só solicitar input se surgir colisão de schema ou necessidade de backlink remoto fora do escopo local explicitado. Nenhum follow-up ou contribuição é criado nesta tarefa.
