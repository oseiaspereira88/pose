---
slug: pose-closeout-review-policy-by-cost-benefit
status: in-progress
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
4. Preservar associações explícitas já usadas no repositório: links Markdown locais do roadmap para um arquivo canônico em `.pose/adr/` e links locais de um ADR para o roadmap também contam após resolução real. O roadmap pose-shared-runtime hoje usa um link `../adr/<arquivo>.md`; não perder essa associação ao adotar a política nova. Definir cobertura para links inline, reference-style, destinos entre angle brackets e fragmentos, normalizados relativamente ao arquivo de origem e contidos no namespace canônico do projeto. Títulos, palavras soltas no corpo, nomes parecidos e URLs externas não inferem associação local. Metadata explícita malformada ou um destino de associação local que não pode ser resolvido produz unknown, não ausência. Não exigir status Accepted da ADR para contar: o critério aprovado é associação resolvida, não aprovação da associação.
5. União das duas direções: uma associação resolvida basta para o predicado ADR. Declaração inválida, alvo ausente/ambíguo, leitura incompleta ou inventário indisponível impede provar ausência e bloqueia dispensa por `when-justified`; não escolher o primeiro arquivo legível. Membros desconhecidos nunca contam como zero. Aplicar budgets e autorização existentes; desconhecimento é diagnóstico com refs e condição de satisfação, não fallback para never. `never` e `always` não precisam inventariar ADRs para decidir, mas não dispensam resolução dos membros/gates.

A escolha mínima é manter links locais explícitos já existentes e acrescentar metadata de associação; não ampliar a gramática global ArtifactRef com ADR nem inferir associações por palavras em prose. Backlinks remotos por busca global não fazem parte deste contrato. Se o mantenedor exigir esse alcance, obter decisão arquitetural antes do parser; a ADR aceita não define esse inventário. Não há essa exigência nesta base.

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
- Criar receipt schema 1 em `.pose/review-closeouts/milestone/<roadmap>/<milestone>.json` e `.pose/review-closeouts/roadmap/<roadmap>.json`, com escopo canônico, instante, subject/input digests, snapshot da policy e decisão do avaliador, membros/associações resolvidos e refs de revisão opcional/obrigatória. Reutilizar a classe administrativa existente `.pose/review-*` para JSON de evidência derivada, sem ampliar a lista de exclusões materiais. Para dispensa persistir literalmente `review: not-required`, chave, valor, versão e razão com counts (`members=1, threshold>2`, por exemplo); uma revisão voluntária aprovada fica referenciada à parte. Para revisão exigida registrar `review: required` e evidência verificada, nunca um revisor inventado.
- Publicar receipt via writer guard/atomic publication existentes, com rechecagem dos inputs e dos gates na região protegida. Fechar roadmap ainda altera status; receipt é o commit marker terminal e só é publicado após status. Falha entre essas publicações fica explicitamente incompleta e repetição verifica os mesmos inputs antes de concluir, sem fabricar sucesso. Recibo existente igual torna repetição somente verificação; diferente/corrompido não pode ser sobrescrito para limpar um veto. Não incluir o próprio receipt/status na identidade semântica que ele atesta.
- Receipt não substitui evidência de entrega ou revisão de membro. Preservar exit gates mesmo quando não há revisão agregada; resultado negativo de gate continua bloqueante, sem auto-attest de juízo humano. Mudança de composição/policy antes de close recalcula elegibilidade; aprovação de bundle aberto com inputs relevantes alterados precisa supersessão como hoje.
- Sem receipt, história terminal anterior à adoção é verificada pelo contrato/plano selado existente, inclusive milestones herdados de roadmap selado. Nunca inferir adoção pela data ou preencher receipts retroativos. Um mero milestone reached sem prova terminal prévia continua aberto na adoção. Done não permite esconder corrupção ou veto histórico. Fechamento novo usa receipt selado; mudanças posteriores da policy não reavaliam a decisão histórica, mas mudanças semânticas e integridade continuam verificadas pelas regras existentes.
- `doctor` mostra escopos abertos que agora devem revisão, por política ou composição: motivo atual, count/ADR, regra anterior quando demonstrável por inputs registrados e ação necessária. Sem snapshot anterior, declarar comparação desconhecida; não inventar transição. Close continua sendo o gate autoritativo.

### CLI, MCP and federation composition

Encadear `cmd/pose/main.go -> internal/cli/review_closeout.go -> Store/evaluator/receipt verifier`; `internal/mcpserver/server.go -> Store` para `pose_review_plan`, `pose_closeout_state`, review verification e aceitação federada. Mesmos modos, counts, veto, resolução e diagnóstico em JSON/texto/MCP. MCP permanece nas permissões atuais; não adicionar ferramenta de configuração ou close write. CLI faz a configuração e o close; MCP observa resultado e próxima ação.

Na federação, avaliar o escopo na sua autoridade, com bindings/autorização/pins/revisão selecionada existentes. Não usar default never do consumidor para dispensar review do produtor. Manifesto novo vincula receipt e decisão; produtores legados continuam no verificador legado. Unknown/unauthorized/transfer-in-progress/stale source e rejeição remota bloqueiam; aliases de redirects verificados deduplicam. Usar somente projetos sintéticos temporários nos testes, nunca outros checkouts reais.

### Artifacts

Lista do change set completo da spec, incluindo a ADR criada e aprovada nos commits de planejamento. Os demais caminhos são previstos para implementação futura; não recriar a ADR nem adotar `.pose/policy/review.json` local. Artefatos de runtime dos testes ficam em `t.TempDir`, não no repositório.

- modified: .pose/specs/2026-11-11-pose-closeout-review-policy-by-cost-benefit.md
- created: .pose/adr/2026-11-12-closeout-review-policy-by-cost-and-benefit.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/roadmaps.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/spec_transfer.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/capability_catalog.go
- modified: pose-mcp/internal/pose/federated_acceptance.go
- created: pose-mcp/internal/pose/closeout_review_policy.go
- created: pose-mcp/internal/pose/closeout_receipt.go
- created: pose-mcp/internal/pose/closeout_adr_associations.go
- created: pose-mcp/internal/pose/closeout_review_policy_test.go
- modified: pose-mcp/internal/pose/diagnostic_codes.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/capability_catalog_test.go
- modified: pose-mcp/internal/cli/adopt_test.go
- modified: pose-mcp/internal/cli/adopt_catalog_test.go
- modified: pose-mcp/internal/cli/progressive_review_test.go
- modified: pose-mcp/internal/cli/release_compatibility_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/setup.go
- modified: pose-mcp/internal/cli/adopt.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/surface_check.go
- created: pose-mcp/internal/cli/closeout_review_policy_test.go
- created: pose-mcp/internal/mcpserver/closeout_review_policy_test.go
- modified: pose-mcp/schemas/v1/review-plan.schema.json
- created: pose-mcp/schemas/v1/closeout-receipt.schema.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/templates/roadmap.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/roadmap.md
- created: .pose/starts/pose-closeout-review-policy-by-cost-benefit.json
- created: .pose/changelogs/unreleased/pose-closeout-review-policy-by-cost-benefit.md

Reconciliação na implementação (2026-11-13): a lista acima é o change set real. Saíram do plano `review_policy_baseline.go` (a proteção reusa `resolveReviewGovernanceGuard` e `parseReviewPolicyAt` a partir do avaliador, sem mudar o arquivo), `state.go` e `mcpserver/server.go` (a projeção MCP é a do `CloseoutState`, que já carrega a decisão e o recibo), `cli/index.go` e `cli/help.go` (nenhuma projeção de índice nova; o uso do `pose adopt` vive em `adopt.go`), `distpolicy.go` (instalação nova adota pelo catálogo, `DefaultForNew`, com o mesmo produtor de `pose adopt`) e os arquivos de teste separados de recibo e associações, unidos em `closeout_review_policy_test.go`. Entraram `federated_acceptance.go` (prova de origem atrelada ao recibo de um fechamento sem revisão exigida, achado F7 da revisão), `cli/surface_check.go` (o gate do milestone normaliza `spec:<slug>`, F13) e `spec_transfer.go`, cujo gate de capacidade recusava schema 5 com transferência adotada. Entraram também `diagnostic_codes.go` (três códigos novos catalogados) e os testes existentes que fixavam o catálogo, o schema após `adopt` e a compatibilidade com a release anterior: uma instalação nova agora adota o par, logo schema 5, que a v7.4.0 recusa por decisão da ADR; o teste passa a exigir que essa recusa seja a do schema e que uma instância legada continue legível pela release anterior depois do `pose update`, que não adota o par.

### Delivery targets

- capability:closeout-review-policy-by-cost-benefit module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

Produção composta exige integração atravessando CLI e MCP até o avaliador e receipt verificado, mais teste de federação e instalação sintética. Registrar check permanente `closeout-review-policy-integration`, evidenceClass `integration`, required, em moduleOverrides.pose-mcp; comando no módulo: `go test ./internal/pose ./internal/cli ./internal/mcpserver -run '^TestCloseoutReviewPolicy' -count=1`. Os novos testes usam esse prefixo e subtestes da tabela abaixo. Build/unit isolados não provam esse alvo.

## 4. Tasks

- [x] Registrar a ADR (R1) e a aprovação explícita do mantenedor em 2026-11-12; sem afirmar implementação.
- [x] Antes de implementar, resolver novamente contexto local, confirmar schema 5 disponível, executar discovery exigido e start autorizado em outra execução; manter baseline de história legada em fixtures.
- [x] Incremento 1 — R1/R2/R6/R7: fixtures de loader, schema 5, catálogo/adopt/setup e defaults distribuídos; validar Schema/Adoption; nenhum update silencioso de policy.
- [x] Incremento 2 — R3: fixtures de aliases, redirects e metadata ADR, avaliador comum e integração com plano/proteção; validar Members/Associations/Thresholds/Protection.
- [x] Incremento 3 — R4/R5/R7: receipt guardado, close explícito, reached/attention/state e verificação terminal histórica; validar Receipt/Veto/History/Publication. Não deixar membro final fechar agregado implicitamente.
- [x] Incremento 4 — R2-R7: projeções CLI/MCP/federação, doctor e schema JSON; validar Parity/Federation/Doctor e smoke de instalação temporária.
- [x] Incremento 5 — Atualizar manuais/templates e matriz; executar checks completos proporcionais ao risco, artifact/surface-check e trace real antes de solicitar revisão independente da implementação.
- [ ] Fazer revisão e closeout com evidência vigente, follow-ups e changelog; só então mudar status/completed_at.

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
| Required unit/contract R3 — adrs forward, roadmaps back, links Markdown locais existentes, união/dedup; palavras soltas, traversal externo, symlink, faltante/ilegível, metadata malformada | `go test ./internal/pose -run '^TestCloseoutReviewPolicyAssociations$' -count=1 -v` | Metadata e links explícitos resolvem a mesma identidade; associação no formato usado pelo roadmap atual não se perde; leitura incompleta não prova ausência; erro não concede dispensa; sem lookup global. |
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
| Required existing regressions — herança, reached, proteção, história e aceitação federada | `LC_ALL=C LANGUAGE=C go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1 -timeout=30m` | Fixtures legadas seguem verdes; fixtures adotadas testam semântica nova sem enfraquecer asserts legados. |
| Required module baseline / build/security/contract | `LC_ALL=C LANGUAGE=C pose validate --strict --module pose-mcp --report --report-task test-plan-baseline-pose-closeout-review-policy-by-cost-benefit` | Todas as checks required da matriz passam, inclusive integração permanente; relatório contém comandos/outcomes/classes vigentes. |
| Required dependency security | `govulncheck ./...` | Sem caminhos vulneráveis alcançáveis; registrar separadamente avisos em módulos requeridos não chamados. |
| Required Go rule | `go vet ./...` | Sem achados bloqueantes; executar mesmo sendo optional na matriz. |
| Required delivery/structure (raiz) | `pose check --strict` | Estrutura e matriz válidas; sem falha nova nos fechamentos históricos. |
| Required contract assessment futuro (raiz) | `pose assess integrate` | Impacto CLI/MCP/federação documentado, sem contrato divergente. |
| Required artifact gate futuro (raiz) | `pose artifact-check --spec pose-closeout-review-policy-by-cost-benefit --strict` | Claims reconciliadas com observações dos commits atribuídos; emendar lista se houver divergência real. |
| Required composition futuro (raiz) | `pose surface-check --spec pose-closeout-review-policy-by-cost-benefit --strict` | Alvo capability ligado ao entrypoint e integração atual; sem alegar entrega com unit/build apenas. |

Não criar check permanente para cada subteste: uma integração registrada com o prefixo novo é suficiente; preservar checks existentes de milestone-review-proportional, milestone-closes-when-reached, review-policy-adoption, policy-scope-validation, federated-roadmap-acceptance e done-spec-keeps-its-sealed-plan. Nenhuma atualização da matriz nesta tarefa de planejamento.

### Requirement trace

- R1 [satisfied] test:TestCloseoutReviewPolicySchema test:TestCloseoutReviewPolicyThresholds check:closeout-review-policy-integration
- R2 [satisfied] test:TestCloseoutReviewPolicySchema test:TestCloseoutReviewPolicyAdoption test:TestCloseoutReviewPolicyUnavailable check:closeout-review-policy-integration
- R3 [satisfied] test:TestCloseoutReviewPolicyThresholds test:TestCloseoutReviewPolicyMembers test:TestCloseoutReviewPolicyAssociations test:TestCloseoutReviewPolicyFederation check:closeout-review-policy-integration
- R4 [satisfied] test:TestCloseoutReviewPolicyReceipt test:TestCloseoutReviewPolicyPublication test:TestCloseoutReviewPolicySmoke test:TestCloseoutReviewPolicyParity check:closeout-review-policy-integration
- R5 [satisfied] test:TestCloseoutReviewPolicyVeto test:TestCloseoutReviewPolicyProtection test:TestCloseoutReviewPolicyAlwaysRefusesWithoutReview check:closeout-review-policy-integration
- R6 [satisfied] test:TestCloseoutReviewPolicyAdoption test:TestCloseoutReviewPolicySetupShowsTheLegacyRule test:TestCloseoutReviewPolicyDoctor check:closeout-review-policy-integration
- R7 [satisfied] test:TestCloseoutReviewPolicyHistory test:TestPreviousReleaseReadsThisEnginesReviewPolicy check:closeout-review-policy-integration check:milestone-review-proportional-integration check:milestone-closes-when-reached-integration

### Execution results

Depois das rodadas 1 a 4 de revisão (2026-11-13): `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1` verde (pose 359s, cli 249s, mcpserver 6s), incluindo os testes novos de cada achado; `go vet ./...` exit 0. A matriz completa é reexecutada pelo pipeline de fechamento sobre o commit final.

2026-11-13, sobre o código de `36fdc959` e índices de `8fc1e9a6`: `go test ./internal/pose ./internal/cli ./internal/mcpserver -run '^TestCloseoutReviewPolicy' -count=1` passa com 17 testes de topo e todos os subtestes (61 em `internal/pose`), nenhum skip. A regressão completa dos três pacotes reprovou primeiro 11 testes existentes, todos corrigidos e reexecutados: inventários (catálogo de capacidades, códigos de diagnóstico, classe de `.pose/review-closeouts/`, limite de prints diretos), `POSE.md` citando um diretório ausente (o gate pós-instalação reprovava, arrastando quickstart e preview do update), fixtures que forçavam schema 2 numa policy com o par, e a compatibilidade com a release anterior, que agora exige recusa pelo schema e leitura da instância legada após `pose update`. `pose validate --strict --module pose-mcp --json-out .pose/results/delivery-validation.json`: 210/210 pass em 404,5s (a execução anterior, concorrente com `pose check`, perdeu só o `test` por timeout de 600s). `go vet ./...` exit 0. `govulncheck ./...`: nenhuma vulnerabilidade alcançável; dois avisos em módulos requeridos não chamados. `pose artifact-check --spec … --strict` e `pose surface-check --spec … --strict` exit 0 (surface: 1 alvo, 210 resultados, 0 findings). `pose lint-spec … --strict` SUCESSO. `pose check --strict` lista 19 erros de contrato de entrega, nenhum desta spec: são de `pose-optional-worktree-runtime`, `pose-worktree-runtime-qos-and-measurement`, `pose-worktree-runtime-resident-cache`, `pose-common-domain-and-cli-mcp-handoff`, `pose-governance-operation-performance` e `pose-bounded-isolated-graph-caches`. O mesmo `pose check --strict` no commit base `956153f1`, com o binário daquele commit, lista os mesmos 19 erros, linha a linha: nenhum é introduzido por esta entrega (R7).

### Planning checks

Executado em 2026-11-12: `/tmp/pose-redaction-cli context --task spec:pose-closeout-review-policy-by-cost-benefit --json` resolveu a autoridade local existente. `/tmp/pose-redaction-cli lint-spec pose-closeout-review-policy-by-cost-benefit --strict` passou (7 requisitos, zero seções vazias, zero falhas; trace sem satisfação de implementação). `/tmp/pose-redaction-cli lint-spec pose-closeout-review-policy-by-cost-benefit --ready-check` passou (`spec.ready=true`, zero falhas). `git diff --check` passou. Esses checks validam o planejamento, não a entrega. Não executar discovery, assessments, full validation, indexação, start, revisão independente ou close como parte deste planejamento.

---

## 7. Final Report

### Delivered scope

Review policy schema 5 com `milestone_review_required` e `roadmap_review_required` (`never`, `when-justified`, `always`), recusado por engines anteriores e validado antes de qualquer plano ou escrita (par incompleto, null, tipo, enum, chave duplicada, chave em schema antigo). Capability `closeout-review-policy` no catálogo: instalação nova adota `never`/`never` pelo mesmo produtor de `pose adopt`; `pose adopt closeout-review-policy` mostra a regra atual e a proposta, aceita o par só junto e não se desliga; nenhuma outra capability rebaixa o schema 5; `pose update` não adota. Avaliador único `closeout-review@1` (membros canônicos `xref:<projeto>/spec:<slug>` com aliases e redirects verificados, associações de ADR nas duas direções por metadata ou link local, desconhecido nunca dispensa, proteção pelo baseline quando o escopo muda o contrato de revisão) consumido por review check, verificação de bundle, review plan, closeout state, reached, attention, doctor e MCP. `pose close` de milestone e roadmap publica um recibo imutável em `review-closeouts/`; o escopo só é terminal com ele; recibo corrompido bloqueia e não é sobrescrito; roadmap done sem recibo é completado por novo `pose close`. `always` anula a herança do membro único e `unreviewed_milestones`; rejeição registrada vale sob `never`; roadmaps fechados antes da adoção mantêm a leitura selada. Membros escritos `spec:<slug>` passaram a resolver no closeout state, onde viravam `spec:spec:<slug>`.

### Residual risks and next owner

A política desta instância não foi adotada (fora do escopo); a adoção do pose-dist e do Harne8 é decisão separada e destrava o fechamento de B31/B35 sem revisão agregada quando o valor escolhido dispensar. Uma instalação nova fica ilegível para a v7.4.0 por decisão da ADR. O recibo grava o motivo, não um snapshot da regra anterior, então o `doctor` declara a regra anterior como desconhecida. A comparação de decisão no momento da publicação não cobre corrida com um escritor que ignore os locks do motor.

### Review

Rodada 1 (run rrn-7c2315202e585212, Codex gpt-6.1-sol, bundle rvb-08c2ad291964aeda): a run foi recusada pelo motor porque o revisor rodou `pose assess integrate`/`tech-debt`, que escrevem no worktree selado; sua conclusão, decisão changes-requested, trouxe 15 achados bloqueadores, todos tratados no código e cobertos por testes: F1 mandatory-human nunca é dispensado (`mandatoryHumanReview`); F2 repetir o close com recibo exige estado terminal atual; F3 a recuperação de um roadmap done sem recibo segue as regras de um escopo aberto e não fecha sobre revisão registrada; F4 o recibo só vale para a mesma composição canônica e `always` não pode gravar not-required; F5 o fechamento grava uma intenção antes do status, então uma interrupção fica incompleta e só um roadmap done sem recibo nem intenção é histórico; F6 sob adoção o gate vem antes de `enabled=false` e do cutoff legado; F7 prova federada por recibo; F8 preflight e publicação usam a mesma decisão protegida (`CurrentCloseoutDecision`); F9 dispensa exige estado sem bloqueios e a publicação revalida decisão, membros, ADRs e estado na região protegida, antes do status no roadmap; F10 membro transferido é seguido pelo resolvedor até a identidade final; F11 ADR associada é lida e validada como as demais; F12 o review plan falha com recibo corrompido ou divergente; F13 `spec:<slug>` fecha pelo gate do milestone; F14 desconhecido nunca dispensa nem aprova; F15 o doctor só aceita aprovação fresca para a obrigação atual. F8–F10 e F15 têm cobertura por construção, não por teste dedicado.

Rodada 2 (run rrn-d94a50c1b4d6e56f, bundle rvb-4732b51bcadc81d9): recusada pelo motor por limpeza do adaptador não confirmada e por citar achados da rodada 1, cuja run nunca virou rodada aberta. O revisor deu 9 dos 15 por resolvidos e 6 por parciais, e trouxe 6 bloqueadores e 2 follow-ups baixos, todos tratados: F1 overlays adotados com independência mandatory-human também impedem dispensa; F2 o recibo vale só para os mesmos membros canônicos e o mesmo exit gate (`exit_gate_digest`), lidos sem a política atual; F3 histórico exige evidência positiva do fechamento legado (aprovação registrada no roadmap, ou a regra legada o dispensando), não só o status; F4 roadmap `when-justified` resolve as associações de ADR mesmo abaixo do limiar; F5 todo candidato do inventário de ADR é lido, inclusive um README associado, e um ilegível é desconhecido; F6 reached, attention e doctor perguntam se o recibo ainda fecha o escopo (`closedByReceipt`), não só se existe; F7 `--request` recusa as flags do par; F8 a instalação indica como mudar os valores, não um `--off` recusado. Discordância registrada em F3: um milestone aprovado antes da adoção passar a dever o recibo é o contrato da spec (no legado o close de milestone nunca foi uma transição), não reabertura de história.

Rodada 3 (run rrn-057e10b4445fd53d, attestation rva-f27e990acb2ad3be, round 2 do motor, changes-requested): 8 bloqueadores, todos tratados: N1 e N7 a exigência humana vem do plano efetivo (overlays que o plano seleciona, com cutoffs, e o perfil base), montado sem o gate, e o baseline protegido acrescenta a sua exigência humana e não descarta um `always` adotado com `enabled=false`; N2 reached, attention, roadmap reached e doctor perguntam ao verificador aplicável (`reviewCheck`), não só pela ausência de recusa; N3 o schema do recibo declara `exit_gate_digest` e um teste confere os campos emitidos contra o schema; N4 o close de um roadmap fechado antes da adoção é verificação somente leitura, sem recibo (a recuperação só existe com intenção pendente); N5 uma declaração reversa para roadmap inexistente do projeto é desconhecida; N6 o README do inventário de ADR é lido como as demais; N8 o recibo de roadmap vincula os critérios de corte (`ParseRoadmapCriteria`) e repetir o close reexecuta o gate do milestone ou o roadmap check antes de responder.

Rodada 4 (run rrn-2ad3c228c124a4f4, attestation rva-15b77929c77babfe, changes-requested): 8 bloqueadores, todos tratados: N9 um escopo que remove o perfil do seu tipo enquanto muda o contrato não se dispensa, e o baseline protegido conta os overlays humanos que adotou; N10 um milestone de mais de um membro com exit gate não dispensa a revisão própria, única que julga o gate (o de um membro é julgado na revisão do membro); N11 sob a política adotada nenhum bloqueio é removido por texto, então o de um filho fica; N12 um link reverso para dentro de `.pose/roadmaps/` que não resolve deixa a resposta desconhecida; N13 reached, attention e doctor perguntam o estado terminal completo (`getCloseoutState`); N14 o caminho terminal de um roadmap com recibo reavalia os critérios de corte pelo grafo de entrega indexado (`closeout-receipt-gate-failed`); N15 `spec.go` declarado (o artifact-check passa a rodar antes de selar); N16 um milestone com aprovação própria selada antes da adoção continua fechado, provado pelo bundle: o plano selado agora carrega a decisão de fechamento (`ReviewBundlePlan.CloseoutReview`), ausente em todo bundle anterior à adoção, sem mudar o digest dos legados.

### Follow-ups

- [open] F1: adotar a closeout review policy no pose-dist e no Harne8 e fechar B31 e B35 sob ela. (owner:@pose-maintainers crit:medium review:2026-11-20)
- [open] F2: registrar no recibo um snapshot da regra efetiva anterior, para o doctor comparar a transição em vez de declará-la desconhecida. (owner:@pose-maintainers crit:low review:2026-12-13)
- [open] F4: reconciliar os 19 erros de contrato de entrega que `pose check --strict` já reportava antes desta entrega (specs do runtime de worktree, `pose-common-domain-and-cli-mcp-handoff`, `pose-governance-operation-performance`, `pose-bounded-isolated-graph-caches`). (owner:@pose-maintainers crit:medium review:2026-11-20)
- [open] F3: projetar `adrs:` de roadmap no índice e no MCP `pose_get_roadmap`, hoje lido só pelo avaliador. (owner:@pose-maintainers crit:low review:2026-12-13)
