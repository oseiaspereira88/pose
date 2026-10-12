# POSE — frente 1: estado real e reconciliação das análises recentes

**Data:** 2026-10-11, America/Recife. As execuções locais atravessaram 2026-10-12 em UTC.  
**Repositório:** `oseiaspereira88/pose`.  
**Baseline:** `8d26872587496a3c14c705568ac0f37096be82a4`, main observada no início da auditoria.  
**Referência anterior:** `392aaa5a007f745cdf1804f65631ea2792567fd9`, análise consolidada de 2026-10-03.  
**Natureza:** diagnóstico e reconciliação; não implementa correções, altera políticas, aceita riscos nem fecha specs.  
**Companion:** [reconciliação dos 33 itens e snapshot estruturado](../results/2026-10-11-pose-front-01-reconciliation.json).  
**Programa:** [índice das dez frentes](2026-10-11-pose-analysis-index.md).

## 1. Parecer executivo

O POSE já implementou grande parte da fundação que as primeiras análises propunham. Uma nova release não deve começar recriando uma central de pendências, um modelo de solicitações ou um plano de closeout: essas peças existem, estão conectadas e possuem testes dirigidos. A próxima análise precisa investigar sua suficiência, precisão e custo nos fluxos reais.

A diferença mais relevante entre a referência anterior e a atual é a passagem de proposta para implementação e adoção. O repositório publica a 7.4.1; a discussão centrada em preparar a 6.3.0 é histórica. Entre as duas baselines há 425 commits e 856 arquivos alterados. A dimensão do diff inclui artefatos derivados e registros de governança: não representa, por si, volume de melhoria funcional ou produtividade.

Três conclusões orientam o restante do programa:

1. **Agência e pendências têm uma arquitetura existente para preservar.** Obligations projetam fontes autoritativas; ActionRequests persistem solicitações materiais; readiness interpreta efeitos por fase; Attention apresenta o resultado; os gates de domínio continuam responsáveis pela autorização das transições.
2. **Revisão delegada ainda não fecha o ciclo autônomo.** Brief e dispatch foram entregues. Proveniência, ledger confiável, adoção da capability e fulfillment por runners permanecem em specs draft. O dispatch atual não produz a attestation; uma execução concluída não equivale a uma revisão aprovada.
3. **Performance ganhou otimizações locais, mas não um runtime residente comum.** Há snapshot de listagem por projeção, cache de paths de commits e paralelismo limitado. A CLI continua efêmera; parte do MCP invoca subprocessos; snapshots atuais não garantem uma leitura imutável de todos os insumos.

O backlog futuro deve privilegiar correção de leituras silenciosas, fechamento da confiança na delegação, classificação do trabalho residual e escopo verificável de releases. A arquitetura residente deve ser decidida com medições e equivalência de resultados, preservando os mecanismos existentes.

## 2. Método e limites das evidências

Foram confrontados a análise e o backlog versionados de 2026-10-03, os relatos posteriores recuperados das conversas, o histórico Git, a árvore atual, políticas, specs não terminais, roadmaps e registros de adoção. A leitura do código foi dirigida às temáticas da pauta. A existência de arquivos ou de uma spec `done` não foi tomada isoladamente como prova de suficiência.

As conclusões usam quatro níveis, que devem continuar nas próximas frentes:

| Nível | O que demonstra | O que não demonstra |
|---|---|---|
| Código inspecionado | O caminho implementado e seus limites observáveis | Que todo cenário de produção funciona |
| Teste inspecionado | Um cenário e a expectativa expressa pelo projeto | Uma nova execução aprovada nesta auditoria |
| Registro retido / CI consultada | O resultado registrado para um commit ou execução | Revalidação local de toda a suíte |
| Comando executado nesta auditoria | O resultado atual naquele corpus e ambiente | Benchmark generalizável ou eficácia humana |

Foram executados `pose version`, `pose specs --json`, `pose state --attention --json` e `pose followups --all --json`, usando o binário publicado 7.4.1. O commit da tag é `7ef3be2a44f0d0c2daceaf3f5c63e6840197023a`; entre ele e a baseline auditada mudam apenas seis registros de publicação/verificação, sem diferença no código executável. O código declara `7.4.1-dev`; o binário foi estampado `7.4.1`. Essa diferença importa para freshness de assessments.

**A suíte Go não foi reexecutada localmente:** não havia toolchain Go disponível. A CI do HEAD auditado foi consultada e registrava sucesso; seu workflow configura Go 1.26.9 e executa `go -C pose-mcp test ./... -count=1`. Essa é evidência externa retida, não uma execução nossa. Não foram executados agentes de fornecedores, testes de concorrência, novos benchmarks, um rollout em projetos de terceiros ou uma sessão de uso por uma pessoa.

CI consultada: [run 38086886171](https://github.com/oseiaspereira88/pose/actions/runs/38086886171), no commit auditado, com conclusão `success`. O SHA-256 local do binário usado foi `9ae6e0855c8b82599cab1d9999ae3a20644e091a5e0490b2f85d593294104d66`, igual ao SHA registrado no relatório de verificação da release.

O estado atual de Harne8 não foi inspecionado. Seus resultados citados abaixo pertencem exclusivamente aos registros retidos no POSE. Os 108 follow-ups foram enumerados, mas **não foram adjudicados individualmente** nesta frente. A classificação de dívida, defeito, compromisso e pendência pertence à frente 5.

## 3. Baseline de publicação, adoção e inventário

### 3.1. Versão publicada e código auditado

A consulta ao GitHub retornou [v7.4.1](https://github.com/oseiaspereira88/pose/releases/tag/v7.4.1), publicada em 2026-10-10T21:13:09Z. O repositório retém manifesto, eventos, publicação e verificação em `.pose/releases/v7.4.1/`. O [relatório de verificação](../releases/v7.4.1/verification-report.md) registra verificação de assinaturas, provenance, checksums, SBOM e reconstrução independente.

O achado antigo sobre confusão entre versão candidata e publicada tem implementação corretiva: `resolvePublicVersionProvenance` discrimina candidato, preparado e publicado. A validade de uma release futura ainda exigirá examinar o escopo e a evidência pertinentes, não apenas reproduzir a existência desses campos.

### 3.2. Configuração atual da instância

| Dimensão | Observado | Implicação |
|---|---|---|
| Identidade do projeto | `proj.pose-dist` em `.pose/project.json` | O nome da pasta do checkout não define a identidade |
| Agency/readiness | `agency_readiness_version: 1` | Requests materiais podem impedir transições nos gates de escrita |
| Contract nodes | `contract_nodes_version: 1` | Não é mais capacidade apenas disponível e não adotada |
| Atomic start | versão 1, cutoff 2026-10-06 | Leitura histórica precisa preservar esse cutoff |
| Causality closeout | versão 1, cutoff 2026-10-06 | Adoção exige analisar também o overlay aplicável |
| Definition of Ready | cutoff 2026-10-09 | A referência anterior com data vazia não descreve a instância atual |
| Resolução de requests | política de atores em modo `declared` | Uma resposta autorizada pelo role map não é automaticamente autenticada |
| Reviewer adapters | `.pose/policy/reviewers.json` ausente | A instância auditada não configura dispatch por esse arquivo |
| Stamp da instância | `pose version` informa última atualização 7.1.0 | Versão do motor e atualização da maquinaria são fatos distintos |

Fontes: [review policy](../policy/review.json), [DoR](../policy/dor.json), [actions](../policy/actions.json), [identidade](../project.json). A ausência de adapters e de runs neste checkout não prova ausência de uso local não versionado em outros projetos.

Os [registros de adoção ABM](pose-abm-capability-adoption.md) documentam problemas descobertos no shadow e remediações antes da adoção. Já o [piloto de agência/readiness](pose-agency-readiness-pilot.md) explicita que o ensaio era automatizado e que a fricção de uma pessoa continuava desconhecida. Decisão de adoção delimitada e validação humana de UX devem continuar separadas.

### 3.3. Inventário atual, a partir da CLI e dos arquivos fonte

| Artefato / dimensão | Quantidade |
|---|---:|
| Specs listadas | 383 |
| Specs `done` | 376 |
| Specs `draft` | 5 |
| Specs `in-progress` | 1 |
| Specs `abandoned` | 1 |
| Follow-ups totais | 309 |
| Follow-ups abertos | 108 |
| Follow-ups vencidos, segundo a data do comando | 11 |
| Follow-ups sem owner, segundo a CLI | 42 |
| Follow-ups abertos em specs `done` | 107 |
| Follow-ups abertos em spec `in-progress` | 1 |

Entre os follow-ups abertos: 4 high, 22 medium, 63 low e 19 sem criticidade explícita. Não confundir esses números com uma adjudicação de severidade de defeitos. O owner conta todos os itens expostos pela CLI conforme sua regra; não inferir o denominador a partir de outra tabela.

O armazenamento auditado contém 3.164.207 bytes em 392 arquivos sob specs, 57.750.045 bytes em 1.416 arquivos sob review-bundles e 8.259.695 bytes em 11 arquivos sob indexes. O número de arquivos sob specs inclui journals e auxiliares, não apenas as 383 specs. Tamanho em bytes não mede parsing, chamadas Git, CPU ou latência.

As seis specs não terminais são:

| Spec | Estado | Trabalho a esclarecer |
|---|---|---|
| [run provenance](../specs/2026-10-09-pose-delegated-review-run-provenance.md) | draft | Assinatura e vínculo run → attestation → independência |
| [attempt ledger](../specs/2026-10-09-pose-delegated-review-attempt-ledger.md) | draft | Histórico verificável, rejeições e razões de rerun |
| [delegated-review capability](../specs/2026-10-09-pose-delegated-review-capability.md) | draft | Adoção, registro automático de conclusão válida e escalonamento |
| [action fulfillment](../specs/2026-10-09-pose-delegated-review-action-fulfillment.md) | draft | Mesmo contrato para adapter local e runner remoto |
| [update dry-run](../specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.md) | in-progress | Código e revisões existem; validar fidelidade residual e closeout |
| [amendment history before close](../specs/2026-10-10-pose-amendment-history-holds-before-close.md) | draft | Baseline da primeira emenda e recusa antes de fechar história inválida |

O update dry-run é importante para a frente 7 mesmo sem ter sido a origem da pauta: o próprio projeto registra adoções declaradas com base em uma prévia que não mostrava o que a atualização real faria. Não presumir que a implementação existente já satisfaz todos os requisitos dessa spec aberta.

## 4. Reconciliação dos 33 itens de 2026-10-03

“Implementação presente” significa caminho de código e testes pertinentes encontrados; não significa que cada critério histórico foi reexecutado ou que não existem bugs residuais. A spec associada é um local de rastreabilidade, não o fundamento único do parecer. O JSON companion inclui fontes imutáveis e seus hashes.

| POSE-01 | Corrigir a leitura dos cutoffs legados de judgment e structural causality | Implementação presente | [`review_closeout.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/review_closeout.go) · [`pose-legacy-contract-cutoffs`](../specs/2026-10-04-pose-legacy-contract-cutoffs.md) | A matriz de cutoffs foi implementada; preservar precedência e contratos selados. |
| POSE-02 | Distinguir versão candidata, preparada e publicada em public claims | Implementação presente | [`publicclaims.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/cli/publicclaims.go) · [`pose-public-claims-publication-provenance`](../specs/2026-10-04-pose-public-claims-publication-provenance.md) | Candidato, preparado e publicado agora têm proveniência própria. |
| POSE-03 | Expor garantias reais de identidade e separação de review | Implementação presente | [`review_assurance.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/review_assurance.go) · [`pose-review-assurance-disclosure`](../specs/2026-10-04-pose-review-assurance-disclosure.md) | Declared e verified são discriminados; isso não autentica a execução delegada. |
| POSE-04 | Versionar a atribuição de autoria, conclusão, confirmação e aplicação de review | Implementação presente | [`review_attribution.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/review_attribution.go) · [`pose-review-attribution-roles`](../specs/2026-10-04-pose-review-attribution-roles.md) | Papéis e suplementos existem; não converter atribuição legada em confirmação humana. |
| POSE-05 | Alinhar blocked entre lifecycle, transfer, readiness e métricas | Implementação presente | [`readiness.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/readiness.go) · [`pose-blocked-semantics-alignment`](../specs/2026-10-04-pose-blocked-semantics-alignment.md) | Blocked é operacional; adoption metrics conserva v1 e acrescenta v2. |
| POSE-06 | Reconciliar specs abertas e follow-ups com requisitos e evidência atual | Parcial | [`followups_parse.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/followups_parse.go) · [`pose-open-backlog-reconciliation`](../specs/2026-10-04-pose-open-backlog-reconciliation.md) | Houve duas reconciliações; 108 follow-ups atuais ainda exigem classificação individual. |
| POSE-07 | Dar ao layout flat de spec a mesma integridade de amendments do layout em pasta | Implementação presente | [`amendments.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/amendments.go) · [`pose-flat-spec-amendments`](../specs/2026-10-04-pose-flat-spec-amendments.md) | Journals flat são separados; o gate anterior ao close tem uma nova spec aberta. |
| POSE-08 | Projetar governança suportada, configurada, aplicável e efetiva | Implementação presente | [`effective_governance.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/effective_governance.go) · [`pose-effective-governance-projection`](../specs/2026-10-04-pose-effective-governance-projection.md) | A introspecção existe; eficácia em uso não se deduz só de policy flags. |
| POSE-09 | Definir o contrato comum de obrigação e suas referências | Implementação presente | [`obligation.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/obligation.go) · [`pose-obligation-contract`](../specs/2026-10-04-pose-obligation-contract.md) | Read model normalizado, sem segundo ledger de resolução. |
| POSE-10 | Adicionar reason codes e referências estruturadas aos produtores | Implementação presente | [`diagnostic_codes.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/diagnostic_codes.go) · [`pose-typed-producer-diagnostics`](../specs/2026-10-04-pose-typed-producer-diagnostics.md) | Diagnósticos tipados existem, mas há fallback legacy-opaque para certos blockers. |
| POSE-11 | Implementar agregação de obligations somente leitura | Implementação presente | [`obligation_projection.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/obligation_projection.go) · [`pose-obligation-projection`](../specs/2026-10-04-pose-obligation-projection.md) | 13 produtores são declarados e responderam current nesta execução. |
| POSE-12 | Garantir snapshot, freshness e cobertura nas respostas operacionais | Parcial | [`obligation_projection.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/obligation_projection.go) · [`pose-obligation-projection`](../specs/2026-10-04-pose-obligation-projection.md) | Binding e cobertura existem; snapshot não é isolamento dos arquivos durante toda a operação. |
| POSE-13 | Expor Attention e obligations com uma lógica comum para CLI e MCP | Implementação presente | [`attention.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/attention.go) · [`pose-state-attention`](../specs/2026-10-04-pose-state-attention.md) | CLI e MCP têm projeção comum; wrappers de outros gates ainda usam subprocessos. |
| POSE-14 | Persistir ActionRequest material com identidade e ciclo de vida mínimos | Implementação presente | [`action_request.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/action_request.go) · [`pose-action-requests`](../specs/2026-10-04-pose-action-requests.md) | Solicitações persistentes com assunto, efeitos e identidade já existem. |
| POSE-15 | Resolver solicitações com atribuição e autoridade adequadas | Implementação presente | [`action_resolution.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/action_resolution.go) · [`pose-action-request-resolution`](../specs/2026-10-04-pose-action-request-resolution.md) | Resolução checa papel, projeto, digest e provas quando exigidas. |
| POSE-16 | Proteger resoluções contra replay, conflito, retries e mudança do assunto | Implementação presente | [`action_resolution.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/action_resolution.go) · [`pose-action-request-resolution`](../specs/2026-10-04-pose-action-request-resolution.md) | Revision e idempotency são exigidas; auditaremos concorrência e falhas na frente 9. |
| POSE-17 | Projetar readiness por fase e escopo com continuidade conservadora | Implementação presente | [`readiness_phases.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/readiness_phases.go) · [`pose-phase-scoped-readiness`](../specs/2026-10-04-pose-phase-scoped-readiness.md) | Estados por fase preservam unknown e restrição parcial. |
| POSE-18 | Aplicar efeitos governados nos gates de escrita de domínio, CLI e MCP | Implementação presente | [`governed_effects.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/governed_effects.go) · [`pose-governed-effect-enforcement`](../specs/2026-10-04-pose-governed-effect-enforcement.md) | Requests são enforced nos pontos de escrita; outros domínios conservam seus gates. |
| POSE-19 | Preservar solicitações e obrigações durante transfer e reconciliação de autoridade | Implementação presente | [`spec_transfer.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/spec_transfer.go) · [`pose-transfer-preserves-obligations`](../specs/2026-10-04-pose-transfer-preserves-obligations.md) | Transfer já trata requests; integração externa atual não foi reexecutada. |
| POSE-20 | Agrupar pedidos relacionados e apresentar perguntas concretas ao usuário | Implementação presente | [`action_presentation.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/action_presentation.go) · [`pose-action-request-presentation`](../specs/2026-10-04-pose-action-request-presentation.md) | Agrupamento e perguntas concretas existem; satisfação humana de UX ainda precisa observação. |
| POSE-21 | Evoluir closeout para um plano mecânico recuperável | Implementação presente | [`closeout_plan.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/closeout_plan.go) · [`pose-recoverable-closeout-plan`](../specs/2026-10-04-pose-recoverable-closeout-plan.md) | Plano e checkpoint existem; falha do Git ainda pode aparentar contexto limpo. |
| POSE-22 | Reusar evidência e critérios por equivalência material e proveniência | Implementação presente | [`reuse_equivalence.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/reuse_equivalence.go) · [`pose-material-equivalence-reuse`](../specs/2026-10-04-pose-material-equivalence-reuse.md) | Reuso material é explicado; não equivale a cache residente compartilhado. |
| POSE-23 | Aplicar freshness e materialidade aos assessments | Implementação presente | [`assessment_freshness.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/assessment_freshness.go) · [`pose-adaptive-assessment-freshness`](../specs/2026-10-04-pose-adaptive-assessment-freshness.md) | Freshness depende de conteúdo, versão e matrix, não de TTL arbitrário. |
| POSE-24 | Oferecer spec progressiva e gerar conteúdo factual derivável | Implementação presente | [`spec_facts.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/spec_facts.go) · [`pose-progressive-spec-surface`](../specs/2026-10-04-pose-progressive-spec-surface.md) | Superfície progressiva e fatos derivados existem. |
| POSE-25 | Sinalizar candidatos a reconciliação de follow-ups sem auto-close | Implementação presente | [`followup_candidates.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/followup_candidates.go) · [`pose-followup-reconciliation-candidates`](../specs/2026-10-04-pose-followup-reconciliation-candidates.md) | Candidatos são apresentados sem auto-close. |
| POSE-26 | Observar idade de pendências e intervalos de espera por causa | Implementação presente | [`governance_waits.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/governance_waits.go) · [`pose-governance-wait-rework-observability`](../specs/2026-10-04-pose-governance-wait-rework-observability.md) | Idade, espera atribuída e bloqueio conhecido são separados. |
| POSE-27 | Explicar retrabalho de governança e medir custo computacional | Parcial | [`governance_waits.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/governance_waits.go) · [`pose-governance-wait-rework-observability`](../specs/2026-10-04-pose-governance-wait-rework-observability.md) | Há métricas e script; benchmark ignora exit codes e restaura árvores com git checkout. |
| POSE-28 | Manter corpus adversarial contra cumprimento formal sem valor | Implementação presente | [`abm_golden_test.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/abm_golden_test.go) · [`pose-mechanization-adversarial-corpus`](../specs/2026-10-04-pose-mechanization-adversarial-corpus.md) | Corpus existe; precisa incorporar os novos riscos de delegação, snapshot e release. |
| POSE-29 | Executar piloto do corte vertical de agência e readiness antes do enforcement amplo | Adoção; UX humana pendente | [`governed_effects.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/governed_effects.go) · [`pose-agency-readiness-pilot`](../specs/2026-10-04-pose-agency-readiness-pilot.md) | Piloto automatizado e adoção delimitada registrados; fricção de uma pessoa segue não medida. |
| POSE-30 | Concluir piloto e decisão de adoção das capacidades ABM já implementadas | Adoção registrada | [`effective_governance.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/effective_governance.go) · [`pose-abm-capability-adoption`](../specs/2026-10-04-pose-abm-capability-adoption.md) | Policy atual adota contract-nodes, atomic-start e causality-closeout; preservar cutoffs. |
| POSE-31 | Representar staleness material de premissas e seus scopes | Implementação presente | [`premise_validity.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/premise_validity.go) · [`pose-assumption-validity-scope`](../specs/2026-10-04-pose-assumption-validity-scope.md) | Premissas usam triggers materiais e podem permanecer unknown/unsupported. |
| POSE-32 | Fechar o ciclo de falsifiers e efeitos esperados com reconsideração explícita | Implementação presente | [`falsifier_observation.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/falsifier_observation.go) · [`pose-falsifier-reconsideration`](../specs/2026-10-04-pose-falsifier-reconsideration.md) | Contradição observada gera candidato a julgamento, não verdict automático de arquitetura. |
| POSE-33 | Planejar limpeza de schema, campos legados e blocked numa major apropriada | Inventário, sem apply | [`legacy_inventory.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/legacy_inventory.go) · [`pose-v7-legacy-cleanup-plan`](../specs/2026-10-04-pose-v7-legacy-cleanup-plan.md) | Inventário/dry-run existem; não há apply de migração e a mensagem ainda fala em 6.x. |

Não manter os 33 itens como uma lista indiferenciada de funcionalidades por construir. Preservar o documento antigo como registro de sua baseline e usar esta reconciliação para abrir somente deltas materiais. O plano de cleanup, em particular, não deve ser interpretado como migração executada só porque a major 7 já foi publicada.

## 5. Reavaliação das temáticas posteriores

### 5.1. Revisão secundária: o adapter existe; a confiança completa está pendente

`RenderReviewBrief`, `PlanReviewDispatch` e `RunReviewDispatch` implementam o começo do caminho: brief do bundle, adapter configurável, worktree no commit selado, timeout, grupo de processos e registro de run/transcript. `RunReviewDispatch` verifica mudanças fora de paths considerados derivados e o HEAD final. O comando expõe explicitamente que não registrou uma attestation.

Isso já substitui parte dos scripts pessoais, mas ainda não atende o resultado final do [ADR aceito](../adr/2026-10-09-delegated-review-is-an-adapter.md): o motor deve registrar a conclusão de uma execução que passe suas verificações, e chamar uma pessoa apenas nas exceções definidas pela policy. Essas decisões do mantenedor **não devem ser reabertas como uma exigência de aplicação manual universal**.

O [roadmap existente](../roadmaps/delegated-review.md) contém a decomposição trust/adoption. Há drift documental verificável: o roadmap permanece `draft`, e algumas specs draft ainda descrevem o ADR como aguardando aceitação, embora o ADR esteja aceito e a spec da capability reflita as decisões finais. Corrigir rastreabilidade e dependências será parte da frente 3.

Riscos dirigidos que essa frente precisa investigar:

- `ListReviewRuns` ignora arquivos ilegíveis ou JSON inválido. Não comprova integridade dos registros que usou para contar tentativas.
- O orçamento usa `PriorRuns` calculado no plan. Não há reserva atômica visível nesse caminho; plans simultâneos podem contar o mesmo saldo.
- Não há evento persistido de início antes da execução nesse caminho. Falha de processo, de criação do worktree ou de persistência exige política de recuperação.
- Os registros usam criação exclusiva, mas isso não prova proteção contra exclusão ou reescrita posterior. A spec do ledger permanece aberta.
- Worktree compartilha metadados Git e o subprocesso herda ambiente. É isolamento de checkout com verificação posterior; não é sandbox completa imposta pelo motor.
- `git status` pós-run descarta o erro. Falha pode impedir a observação de modificações; a comparação posterior de HEAD é uma proteção diferente, não equivalente.
- O transcript é acumulado em memória, sem cap explícito nesse caminho. A classificação de paths derivados precisa preservar precisamente o conteúdo revisado.

Esses pontos são achados de leitura e hipóteses de cenários adversariais; não foram reproduzidos com fornecedores nesta frente. A frente 3 deverá estabelecer testes e soluções, vinculando-os às quatro specs existentes.

### 5.2. Atestações: assinatura, autoria e independência são garantias diferentes

Há disclosure de assurance, atribuição de papéis, confirmação vinculada ao conteúdo, issuer nativo e ledger assinado para atestações legadas. Portanto, “faltam campos de proveniência em todo o motor” é uma descrição antiga e ampla demais.

O gap específico é a cadeia de confiança do **run delegado** até o julgamento aceito. Um envelope assinado autentica o que seu emissor afirmou; não demonstra automaticamente que houve outro modelo, outro vendor, uma execução isolada ou julgamento adequado. O caminho declarado ainda admite convenções textuais de identidade; o disclosure explicita seus limites. A substituição dessa convenção para instâncias que adotarem delegated-review é escopo já previsto.

Preservar o histórico: o ledger de atestações legadas não se confunde com o ledger de tentativas delegadas. Não usar a implementação do primeiro para declarar entregue o segundo.

### 5.3. Pendências: integração existente, completude ainda precisa ser demonstrada

`ProjectObligations` declara 13 produtores e nesta execução todos responderam `current`, com `complete: true`: readiness, closeout, review, start, followups, action-requests, assessments, premises, falsifiers, docs-review, capability-trigger, release e findings.

A execução apresentou 108 residual-debt, duas dependencies, uma evidence advisory e uma judgment bloqueando closeout. A evidence stale decorre, nesta chamada, da diferença entre assessment em 7.4.1-dev e binário em 7.4.1; não deve ser apresentada como regressão funcional comprovada. Há duas dependências da capability aguardando suas specs draft. A pendência de judgment pertence ao update dry-run.

O modelo faz a separação correta entre satisfação, conhecimento, espera e efeitos. Entretanto, `current` é uma declaração do produtor: se um parser ignora um erro e retorna uma lista incompleta, a camada agregadora não recupera o fato perdido. A frente 4 deve avaliar cobertura real, identidades, correlação, propagação e alcance por fase, preservando fontes autoritativas e os gates existentes.

### 5.4. Dívida e release: existem resíduos e gates, mas falta avaliar sua suficiência

`followupObligations` classifica follow-ups abertos como residual-debt advisory por padrão. Isso protege contra bloquear qualquer entrega por todo o passado. Ao mesmo tempo, não diferencia automaticamente dívida aceita, defeito ativo, promessa incompleta ou fratura da release.

O lifecycle de release tem manifestos, fragments, evidências e projeções de estado; `ReleaseGovernedRefusal` impede a preparação quando requests aplicáveis restringem release. O problema futuro não é simplesmente “não há release governance”: é estabelecer se o motor consegue confrontar **o que a release prometeu** com **o que ela demonstra entregar**, incluindo adiamentos explícitos e defeitos incompatíveis com esse compromisso.

A frente 5 classifica trabalho residual; a frente 6 define sua consequência para uma release concreta. Nem todo follow-up medium é blocker, e nem todo defeito pode ser adiado porque foi registrado como follow-up.

### 5.5. Processo residente: preservar otimizações e revalidar os gargalos

`withSpecSnapshot` já evita reler a listagem de specs a cada produtor de uma projeção. Também há cache de scopes de bundles nessa operação, cópias de slices e execução concorrente com resultados ordenados. `commitPaths` possui limite de 20.000 entradas; `takeWorkingTreeSnapshot` reduz leituras repetidas de status no caminho que o usa. Logo, o diagnóstico “toda projeção sempre relê todas as specs N vezes” não descreve integralmente o código atual.

Ainda assim, `GetSpec` busca em uma listagem e lê o corpo novamente; o snapshot não fornece catálogo persistente por slug nem corpos versionados compartilhados. `FollowupCandidates` e a projeção de follow-ups chamam o parser separadamente. O MCP usa `runGate`/`runCLIExit` em gates como check e lint, enquanto outras consultas são diretas. Paginação e `--recent` podem cortar a resposta depois da listagem, sem reduzir o trabalho necessário para produzi-la.

A frente 8 deve medir esses caminhos na baseline exata e comparar alternativas. O runtime por worktree, catálogo híbrido, conteúdo lazy, índices e fallback standalone continuam propostas, não implementação comprovada. Uma sessão lógica iniciada não demonstra compartilhamento de memória entre processos.

Não transformar snapshots atuais em cache de longa duração sem corrigir sua identidade e invalidação. `CurrentObligationSnapshot` vincula HEAD, dirty boolean, policies, contratos, autoridade e índice. Não fixa o conteúdo de cada arquivo nem valida, ao final da projeção, que todos pertenciam à mesma geração. Duas edições sucessivas de uma spec com dirty já true podem preservar essa identidade quando não alteram os outros insumos. `git status` ignora untracked e seu erro não marca incoerência. Esse binding é útil, mas não é uma transação de leitura.

## 6. Deltas prioritários para as próximas análises

Não são issues adjudicadas nem compromissos de release. São candidatos cuja evidência já permite investigação dirigida.

| ID | Delta | Evidência / natureza | Frente responsável | Tratamento preliminar |
|---|---|---|---|---|
| D01 | Erros do parser de follow-ups desaparecem | `ParseSpecFollowups` retorna apenas slice e dá continue em falha | 4, com confiabilidade na 9 | Prioridade de integridade; retornar cobertura/erro útil |
| D02 | Falha de Git aparenta contexto limpo no plano | `ungeneratedChanges` retorna nil em erro | 9 | Investigar recusa/unknown sem autorizar continuidade silenciosa |
| D03 | Snapshot insuficiente para cache duradouro | dirty boolean, corpos relidos, sem validação final de geração | 8, com invariantes na 9 | Pré-condição para reutilização entre operações |
| D04 | Run não comprova independência nem produz conclusão governada | dispatch e quatro specs draft | 3 | Completar o roadmap trust/adoption existente |
| D05 | Tentativas concorrentes e incompletas não têm reserva/recuperação nesse caminho | orçamento do plan; persistência no fim | 3, com invariantes na 9 | Repro adversarial antes de ampliar delegação |
| D06 | Isolamento e transcript precisam de contrato explícito | worktree, ambiente herdado, buffer sem cap | 3 | Delimitar garantias e limites do adapter |
| D07 | Trabalho residual sem distinção suficiente de compromisso | follow-ups projetados como advisory residual-debt | 5 | Taxonomia e reconciliação; não promover tudo a blocker |
| D08 | Suficiência de release não se deduz do ledger de publicação | manifest/fragments/evidence existentes | 6 | Investigar compromisso de escopo e blockers derivados |
| D09 | Fechamento pode revelar história inválida depois | gate de amendments condicionado a done; spec draft existente | 9 | Trabalhar na spec aberta; evitar duplicação |
| D10 | Preview/update/adoption podem divergir na percepção do operador | spec update dry-run ainda in-progress | 7 | Examinar código e revisões antes de declarar resolvido |
| D11 | Mensagens de migração ainda dizem 6.x/7.0 futuro | `cmdMigrate` rejeita apply com esse texto em motor 7.4.1 | 7 e 10 | Corrigir sem inferir que apply já existe |
| D12 | Benchmark atual pode ocultar falhas e desfazer mudanças locais | `measure` usa `|| true`; script termina com `git checkout -- .pose/indexes .pose/state` | 8 | Medir em cópia descartável; registrar verdict e ambiente |
| D13 | Uma obligation inválida descartada não torna o report incompleto | `ValidateObligation` acrescenta limitation e dá continue; Complete depende de snapshot/coverage | 4, com invariantes na 9 | Reproduzir e definir o significado de completude com perdas explícitas |

Não foi executado o script de benchmark no checkout do usuário; a linha de restauração justifica preferir um corpus descartável quando a frente 8 medir index. Não alterar o gate de produção para melhorar o número medido.

**Reprodução dirigida de D01:** em diretório descartável, criar `.pose/specs/2026-10-11-visible.md` com uma spec e um follow-up open válido; criar `.pose/specs/2026-10-11-unreadable.md` como link para `absent-target.md`, que não existe; executar `pose followups --all --json` nesse diretório. Observado: exit 0, stderr vazio, um item e um open, sem campos de erro, limitation ou coverage. Isso demonstra a omissão de uma entrada ilegível nesse cenário. Não demonstra quantas pendências existiriam em um arquivo cujo conteúdo não pode ser lido, nem reproduz todos os tipos de erro de filesystem.

## 7. Composição preliminar das entregas futuras

Definir números de releases agora seria antecipar compatibilidade e dependências ainda não analisadas. Há, contudo, quatro conjuntos de resultado que devem orientar a composição:

1. **Correção e confiabilidade:** leituras/Git com erro explícito, preflight de amendments, precisão de preview e mensagens. Um fechamento `done` não pode ser surpreendido por um gate que só aplica a regra depois da transição.
2. **Delegação completa:** proveniência e ledger antes da capability que produz attestations automaticamente; escalonamento por policy e fulfillment no mesmo contrato local/remoto. Esse conjunto tem roadmap e decisões aceitas para reaproveitar.
3. **Compromissos e release:** classificação do residual, adiamento explícito e suficiência do escopo, construídos sobre obligations e release lifecycle existentes.
4. **Eficiência e runtime:** primeiro medições e eliminação de trabalho repetido; depois compartilhamento residente, se o benefício e a consistência justificarem. Daemon obrigatório não é uma premissa desta análise.

A ordem não obriga quatro releases separadas. Correções podem anteceder ou acompanhar uma entrega maior; confiança do run e recuperação de tentativas são dependências da delegação autônoma. As frentes 2–9 determinarão os critérios, riscos, migrações e agrupamentos definitivos; a frente 10 consolidará o plano.

## 8. Invariantes e contrato das próximas frentes

- Preservar filesystem/Git e registros de domínio como fontes canônicas; cache e Attention são projeções.
- Diferenciar autorização recebida, identidade declarada, confirmação autenticada e conclusão produzida por execução.
- Preservar contratos selados, cutoffs e histórico; não aplicar regras novas retroativamente por acidente.
- Tratar ausência de evidência e erro de leitura como estados distintos de “não há pendência”.
- Automatizar passos mecânicos; exigir julgamento onde o resultado depende dele, com pessoas somente nas exceções previstas.
- Preservar funcionamento do POSE sozinho e compatibilidade de contrato com runners externos.
- Não usar paginação de apresentação para reduzir silenciosamente a cobertura de gates.
- Avaliar esforço, custo e eficácia separadamente; quantitativos de artefatos não são métricas de valor.

Cada próxima análise deve citar baseline, cenário concreto, caminho no código, teste existente ou repro necessário, implicação operacional, alternativas, recomendação e critérios de aceite. Um achado transversal terá uma frente responsável, com referência nas demais. Reconciliar as specs existentes antes de propor novas.

## 9. Fontes e validação

O companion retém hashes e links imutáveis do código e testes usados na reconciliação. As fontes adicionais deste documento estão relacionadas abaixo no commit auditado. Documentos operacionais e specs foram usados para identificar intenções e registros, nunca para substituir comportamento do código.

- [`pose-mcp/internal/pose/spec.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/spec.go)
- [`pose-mcp/internal/pose/cli.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/cli.go)
- [`pose-mcp/internal/pose/review_dispatch.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/review_dispatch.go)
- [`pose-mcp/internal/pose/review_dispatch_unix.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/review_dispatch_unix.go)
- [`pose-mcp/internal/pose/review_brief.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/review_brief.go)
- [`pose-mcp/internal/cli/review_dispatch.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/cli/review_dispatch.go)
- [`pose-mcp/internal/pose/release_lifecycle.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/release_lifecycle.go)
- [`pose-mcp/internal/pose/legacy_ledger.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/legacy_ledger.go)
- [`pose-mcp/internal/pose/issuer_key.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/issuer_key.go)
- [`pose-mcp/internal/pose/git_spawn_cache.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/pose/git_spawn_cache.go)
- [`pose-mcp/internal/cli/specs_cmd.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/cli/specs_cmd.go)
- [`pose-mcp/internal/cli/lintspec.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/cli/lintspec.go)
- [`pose-mcp/internal/cli/migrate_dryrun.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/cli/migrate_dryrun.go)
- [`pose-mcp/internal/cli/update_dryrun.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/cli/update_dryrun.go)
- [`pose-mcp/internal/cli/adoption_metrics.go`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/pose-mcp/internal/cli/adoption_metrics.go)
- [`scripts/bench-governance.sh`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/scripts/bench-governance.sh)
- [`.github/workflows/ci.yml`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/.github/workflows/ci.yml)
- [`.pose/reports/2026-10-pose-open-backlog-reconciliation.md`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/.pose/reports/2026-10-pose-open-backlog-reconciliation.md)
- [`.pose/reports/2026-10-03-pose-consolidated-analysis.md`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/.pose/reports/2026-10-03-pose-consolidated-analysis.md)
- [`.pose/reports/pose-agency-readiness-pilot.md`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/.pose/reports/pose-agency-readiness-pilot.md)
- [`.pose/reports/pose-abm-capability-adoption.md`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/.pose/reports/pose-abm-capability-adoption.md)
- [`.pose/roadmaps/delegated-review.md`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/.pose/roadmaps/delegated-review.md)
- [`.pose/adr/2026-10-09-delegated-review-is-an-adapter.md`](https://github.com/oseiaspereira88/pose/blob/8d26872587496a3c14c705568ac0f37096be82a4/.pose/adr/2026-10-09-delegated-review-is-an-adapter.md)

**Checks deste documento:** `pose check --strict` passou na baseline e após a inclusão dos documentos, com os mesmos 30 avisos (27 de changelog, dois de capability assessment e um de stamp da instância). O JSON foi parseado; os 33 IDs são únicos; os 66 hashes de evidência e os links relativos foram conferidos. O diff foi validado quanto a whitespace. A reprodução de D01 foi executada no fixture descartável e está registrada no companion. A suíte Go permanece apenas com evidência da CI citada, sem reexecução local.

**Limite final:** esta frente estabelece a base atual e os deltas para investigação. Não certifica o motor, a suficiência das releases históricas, independência cognitiva de agentes ou eficácia humana em produção.
