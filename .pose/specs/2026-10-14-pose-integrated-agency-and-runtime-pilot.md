---
slug: pose-integrated-agency-and-runtime-pilot
status: draft
created_at: 2026-10-14
completed_at:
depends_on: pose-delegated-review-capability, pose-residual-triage-and-risk-acceptance, pose-condition-response-and-work-metrics, pose-proportional-next-action, pose-release-shared-sufficiency-contract, pose-release-scope-selection-and-exclusivity, pose-agency-readiness-pilot, pose-mechanization-adversarial-corpus
priority: 2
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:integrated-agency-and-runtime-pilot
---

# Spec: Qualificar UX, contratos adversariais e runtime em piloto integrado

## 1. Intent

### Goal

Demonstrar utilidade e integridade nos oito percursos operados por pessoa e agente,
com baseline comparável, runtime opcional qualificado e decisão delimitada.

### Business value

B42 / F07-10. Consultar a [frente 7, seção 9](../reports/2026-10-12-pose-front-07-operational-ux.md#9-pilotos-métricas-e-evidência-de-excelência)
e a [matriz autoritativa](../results/pose-implementation-plan.json).
Reusar o [piloto histórico](2026-10-04-pose-agency-readiness-pilot.md) e o
[corpus adversarial](2026-10-04-pose-mechanization-adversarial-corpus.md), ambos done;
não repetir sua entrega, alterar os bytes ou reinterpretar o rehearsal como pessoa.

### Constraints

Esta escrita cria somente o contrato draft. Não executar piloto, chamar agentes,
operar projetos externos, alterar policy, iniciar runtime, migrar dados ou preparar
release. Execução funcional requer escopo separado, assessment freshness, readiness
e artefatos reconciliados antes de start. Manter outputs/evidência sem segredos ou
dados pessoais; usar fixtures sintéticas e identificação de papéis suficiente.
Preservar adoções históricas e separar snapshots, observações e estado corrente.
Este corpus pertence à qualificação do POSE; não é checklist obrigatório para cada
feature do adotante. Defeitos pertencem às suas specs, não a um novo engine do piloto.

## 2. Requirements

- R1: Pré-registrar os oito percursos J1–J8 da frente 7, corpus/fixtures imutáveis, revisões baseline/candidate, engine/build, configuração/policy já vigente, módulos e ambiente. Fixar denominadores, repetições, limites de tempo/recursos e critérios de comparação antes das medições; não selecionar apenas casos felizes ou mudar corpus para obter ganho.
- R2: Executar cada percurso em trilhas distintas de pessoa e agente, identificando quem operou, quem preparou/confirmou conclusões, execution/run e assurance realmente verificada. Medir a pessoa sem script responder em seu nome; registrar assistência/conhecimento interno, tentativas/loops do agente e limitações de amostra. Sem participante humano real, declarar essa trilha pendente; principal human:* em fixture e rehearsal não contam como experiência humana ou review independente.
- R3: Qualificar mudança trivial já autorizada, bug com regressão e contrato em dois módulos pelas instruções distribuídas EN/pt-BR/scaffold, descoberta de regras e rotas CLI/MCP pertinentes. Reusar spec/authority/evidência adequadas, sem pedido redundante, nova spec por typo, instalação automática de extensão ou enfraquecimento de gates. Comandos sugeridos devem ser executáveis ou revelar handoff/limitação, e não só existir no catálogo.
- R4: Qualificar judgment por agente independente, exceção humana de risco e operação externa com os contratos existentes de B21/B25/B27. Preservar source/target/fase, papel de holder/executor, assurance, autoridade, conclusão material e fulfillment verificado. Prefixo de identidade, run terminado, JSON assinado ou campos preenchidos não provam independência cognitiva, aprovação semântica ou operação externa executada. Pessoa atua quando o contrato exigir, sem confirmação universal nova.
- R5: Qualificar retomada após alteração material de source e tarefa cross-project: revalidar inputs/digest/generation e autorização no projeto correto, conservar efeitos/receipts/checkpoints, repetir apenas evidência invalidada e não duplicar writes. Cobrir transferência de autoridade com request aberto, resposta antiga/stale e seleção de raiz não autorizada. Não assumir que histórico de outro snapshot seja estado corrente nem repetir uma decisão já válida.
- R6: Estender o corpus adversarial existente com a satisfação formal mais barata de cada percurso/gate novo, controles positivos e recusas/recuperação. Cobrir instruções, flags/filtros inválidos, stdout/stderr/JSON em success/wait/failure, query success versus coverage/readiness, done/no-op versus verified, producer indisponível, erro operacional versus resultado negativo e evidence/authority stale. Known-gap deve permanecer explícito até ser fechado pelo dono do mecanismo; não inverter expectativa ou abrandar gate para aprovar o piloto.
- R7: Medir separadamente tempo de comando e de operador, calls/trocas CLI-MCP, trabalho útil entregue, fatos gerados/reusados com adequação, recusas úteis/artificiais, perguntas materiais/redundantes, invalidações/rework por causa e retomada sem reconstrução manual. Usar unidades, clock, denominadores/coverage B26 e métricas comparáveis; unknown não vira zero, volume de TODOs/residuals/campos/bytes não vira produtividade. Publicar distribuições/amostras/limitações, sem score agregado ou p95 de três observações.
- R8: Consumir B37/B40 para qualificar suficiência/seleção do cut nos percursos pertinentes, sem preparar/publicar release ou adotar policy por inferência. Baseline e candidate usam o mesmo trabalho/contrato e evidence scope; resultado entregue com coverage é requisito de comparação. Registrar falhas/limitações e encaminhá-las à spec proprietária ou delta autorizado, sem implementar correções de engine escondidas sob o piloto.
- R9: Preparar o piloto somente após os seis pré-requisitos técnicos B21/B25/B26/B27/B37/B40 e reusar as duas fundações done. Qualificação final exige separadamente B31/B33/B35 entregues e prova atual: domínio/handoff CLI-MCP, vistas/caches profundamente isolados e bounded, runtime opcional por worktree com attach/identity/lifecycle/fallback e QoS. Comparar standalone/CI e runtime opt-in cold/warm/churn/restart sob carga no mesmo corpus, sem tomar stdio do cliente, cache/watch hint ou UID como autoridade. Não exigir B35 para escrever ou preparar fixtures; não encerrar B42 antes de qualificar esse escopo final.
- R10: Pré-registrar stop/go/rollback por instância e escopo: zero falso sucesso/coverage/authority, zero bypass/duplicação/perda de efeito e todos os oito percursos nas duas trilhas com resultado ou limitação explícita que impeça qualificação integral. Ganhos/trade-offs e budgets devem atingir limites fixados antes da coleta; ausência de baseline, prova de runtime, pessoa real ou threshold resolvido mantém decisão inconclusiva/defer. Registrar decisão pelo ator competente, razões e evidence binding; go delimitado não prova eficácia universal nem altera policy automaticamente. Rollback interrompe ampliação/uso experimental e retorna ao caminho previamente qualificado, preservando história/receipts; mudança de policy requer autorização própria.

## 3. Technical Plan

### Artifacts

Paths prospectivos, a reconciliar antes do start. Somente esta spec é criada no
lote 15 sob `pose-consolidation-batch-15-authoring`; scripts, fixtures, resultados
e casos de teste novos abaixo não existem e não foram executados. Preservar
reports/results históricos; o piloto integrado terá registros próprios.

- created: .pose/specs/2026-10-14-pose-integrated-agency-and-runtime-pilot.md
- created: .pose/starts/pose-integrated-agency-and-runtime-pilot.json
- created: .pose/changelogs/unreleased/pose-integrated-agency-and-runtime-pilot.md
- created: .pose/results/pose-integrated-agency-and-runtime-pilot.json
- created: .pose/reports/pose-integrated-agency-and-runtime-pilot.md
- created: tests/pilot/integrated-agency-and-runtime.sh
- created: tests/pilot/integrated-agency-and-runtime-corpus.json
- created: pose-mcp/internal/cli/integrated_agency_runtime_pilot_test.go
- modified: pose-mcp/internal/cli/adversarial_corpus_test.go
- modified: pose-mcp/internal/cli/adversarial_corpus_gates_test.go
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:integrated-agency-and-runtime-pilot module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Approach and execution gates

1. Reusar harness de [quickstart](../../tests/quickstart/first-governed-loop.sh),
   [medição limpa](../../tests/quickstart/measure-clean-environment.sh),
   [probes de UX](../results/2026-10-12-pose-front-07-ux-probes.py) e corpus Go.
   Inventariar comandos/atores/fixtures e binding de evidence antes de escrever
   runner novo; não copiar uma segunda implementação de gates ou agendar agentes
   no core. A execução usa operações admitidas e Harness externo já governado.
2. Os seis Bxx em depends_on são pré-requisitos técnicos. As duas specs históricas
   são fundações concluídas. B21 só é aceito depois das fases B19 → B20 → B21;
   intermediate acceptance não é done da spec compartilhada. A escrita não muda
   dependências/status/R-IDs/start baselines dessas reutilizadas.
3. B31/B33/B35 são condições adicionais de qualificação final, não novos
   pré-requisitos técnicos para preparar o piloto. O milestone proprietário
   `governance-efficiency/integrated-agency-runtime-pilot` aguarda seus resultados
   para fechamento. A distinção consta na matriz; não alegar suporte nativo a
   gates de fase que ainda não existe. R9/trace/review devem verificar essas provas
   antes do closeout da spec; preparação parcial não permite qualificá-la integralmente.
4. Fixar baseline/candidate, quotas/plataformas, amostras e thresholds por percurso
   e modo antes da coleta. Executor automatizado conserva exit/stdout/stderr,
   eventos/inputs e resultados; sessões de pessoa/agente têm protocolos separados.
   Não preencher judgment, respostas humanas ou stop/go por script.
5. Resolver compatibilidade/formato do registro antes do código; usar ADRs
   existentes, incluindo [obligations](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md)
   e as propostas de [release](../adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md)
   e [domínio/runtime](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md).
   Não reabrir decisões aceitas nem adotar propostas por referência. Nova extensão
   estrutural, se necessária, deve ser explicitamente decidida antes do código.

### Percursos pré-registrados

Executar cada linha nas trilhas pessoa e agente, preservando dados separados.

| ID | Percurso | Resultado a qualificar |
|---|---|---|
| J1 | Mudança trivial autorizada | Reusar autoridade e fluxo mínimo; nenhuma pergunta artificial ou spec obrigatória por typo. |
| J2 | Bug com regressão | Reusar spec adequada, demonstrar regressão e correção com coverage pertinente. |
| J3 | Contrato em dois módulos | Spec/ADR e integração proporcionais; instruções/callers coerentes, sem checks alheios como prova. |
| J4 | Judgment por agente independente | Conclusão, provenance/assurance e fulfillment reais; execução separada quando exigida. |
| J5 | Exceção humana de risco | Pessoa responde apenas a decisão material; risco/validade/restrição seguem o contrato. |
| J6 | Operação externa | Harness admitido e receipt verificado; ausência/partial effect não viram sucesso. |
| J7 | Retomada após source mudar | Revalidar só inputs afetados; autoridade transferida/stale e retry sem duplicação. |
| J8 | Tarefa cross-project | Contexto/raízes autorizados e handoff explícito, sem vazamento ou retarget silencioso. |

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8, R9, R10
- Decision: Qualificar percursos completos e modos standalone/runtime sobre corpus fixo, distinguindo pessoa, agente, rehearsal e provas atuais; manter seis dependências técnicas e três de qualificação final.
- Rationale: Campos presentes, testes de schema e duração de comandos não demonstram utilidade, experiência humana ou runtime seguro. Preparação pode ocorrer antes da residência, mas closeout integrado exige seu aceite.

## 6. Validation

### Strategy

O aceite F07-10 cobre R1–R10. Os casos abaixo são prospectivos, sem requisito
satisfeito por lint, merge ou execução do pipeline documental.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — J1–J8 por pessoa e agente, baseline/candidate fixos | Dezesseis combinações identificadas; pessoa real distinta de fixture, assistência/limitações e trabalho útil com coverage. |
| V2 — Instruções EN/pt-BR/scaffold, CLI/MCP e erro/wait/recusa | Comandos sugeridos executáveis, JSON/streams/next action coerentes; controles positivos e negativos. |
| V3 — Judgment, risco, operação externa e atalhos formais | Autoridade/conclusão/fulfillment verificados; identidade/run/schema não substituem prova, sem resposta fabricada. |
| V4 — Source material versus alteração irrelevante, autoridade transferida, replay | Invalidation pertinente, autorização válida reutilizada e nenhum efeito duplicado; history/receipts preservados. |
| V5 — Cross-project, raiz não autorizada, producer ausente e evidence stale | Contexto/disclosure/coverage honestos; nenhum falso completo, retarget ou vazamento. |
| V6 — Métricas de trabalho/reuso, waiting/rework e denominadores | Dimensões/unknown explícitos e comparação reproduzível; sem produtividade por redução de backlog. |
| V7 — Domínio/cache/runtime cold/warm/churn/restart, CI fallback e carga | Provas B31/B33/B35 atuais, isolamento/budgets/QoS e resultados/authority equivalentes; runtime permanece opcional. |
| V8 — Falha de integrity/budget, amostra/participante ausente, stop/go/rollback | Stop/defer ou decisão delimitada com evidência; rollback preserva efeitos/história, sem policy automática. |

### Deterministic checks

Na execução autorizada, incluir regressões na matriz, executar `cd pose-mcp && go test ./...`
e `go vet ./...`, integração CLI/MCP/quickstart/adversarial e race quando pertinente.
Executar harness sobre fixtures isoladas e sessões reais separadas conforme protocolo,
reter resultados com hashes/engine/configuração/coverage e comparação verificável.
Conferir lint/check, artifact/surface e review independente atual; roadmap-check
deve recusar fechamento antes das dependências e evidências finais. Passes de CI
não substituem experiência humana, medição de runtime ou stop/go do ator competente.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V6, V7 |
| R2 | V1, V3, V8 |
| R3 | V1, V2 |
| R4 | V1, V3 |
| R5 | V1, V4, V5 |
| R6 | V2, V3, V4, V5 |
| R7 | V1, V6, V7 |
| R8 | V1, V5, V6, V8 |
| R9 | V7, V8 |
| R10 | V1, V3, V7, V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 15; consultar [relatório documental](../reports/2026-10-14-pose-batch-15-authoring.md).
Piloto, fixtures/testes novos, métricas atuais, revisão e closeout não executados.

### Residual risks

O piloto histórico é rehearsal automatizado e não exerceu transferência de autoridade;
adoções existentes não demonstram o escopo ampliado. Thresholds/amostras/plataformas e
participantes reais ainda precisam ser fixados. Provas de runtime e reviewer independente
permanecem futuras. Sem essas evidências não há qualificação integral ou eficácia universal.

### Follow-ups

Manter execução funcional separadamente autorizada e B43 condicional na matriz.
Não criar migração, ampliar adoção ou executar stop/go durante esta escrita.
