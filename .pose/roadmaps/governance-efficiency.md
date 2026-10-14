---
slug: governance-efficiency
status: draft
created_at: 2026-10-04
depends_on:
---

# Roadmap: Proportional and measured governance

**Program:** POSE agency and readiness, wave 3 of 4.
**Outcome:** governing a delivery costs less operator work without losing integrity — closeout is a recoverable plan, valid observations are reused by material equivalence, assessments and spec surface follow materiality, residual backlog drift is visible, waiting and rework are measured — and every adoption is decided by a pilot.

Source: [third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md),
items POSE-20 to POSE-27, POSE-29 and POSE-30.

## Milestone: recoverable-closeout
- after: spec:pose-phase-scoped-readiness, spec:pose-governed-effect-enforcement
- target_start:
- target_due:
- specs: pose-recoverable-closeout-plan, pose-material-equivalence-reuse

**Exit gate:** an interrupted closeout resumes without duplicate bundles or attestations; stale `results_path` is regenerated before sealing; judgment is never filled; reuse cites origin and input bindings.

## Milestone: proportional-surface
- after: recoverable-closeout
- target_start:
- target_due:
- specs: pose-adaptive-assessment-freshness, pose-progressive-spec-surface, pose-followup-reconciliation-candidates, pose-action-request-presentation

**Exit gate:** a trivial change keeps the minimal path with adopted gates intact; factual sections are generated, never intent or rationale; candidates are raised without auto-close; grouped requests keep independent answers.

## Milestone: measured-governance
- after: recoverable-closeout
- target_start:
- target_due:
- specs: pose-governance-wait-rework-observability

**Exit gate:** pending age, attributed waiting and known blocking are separate; unknown stays unknown; rework causes are classified only with evidence; benchmarks report time and bytes on fixed corpora.

## Milestone: agency-pilot
- after: measured-governance
- target_start:
- target_due:
- specs: pose-agency-readiness-pilot

**Exit gate:** stop/go recorded with baseline comparison, rollback conditions and delimited adoption scope.

## Milestone: abm-adoption
- after: spec:pose-open-backlog-reconciliation, spec:pose-flat-spec-amendments, spec:pose-effective-governance-projection
- target_start:
- target_due:
- specs: pose-abm-capability-adoption

**Exit gate:** contract nodes, atomic start and causality closeout each have an explicit adopt-or-defer decision backed by pilot and shadow evidence; no second implementation.

## Continuação prospectiva — lote 4

Preservar o programa histórico e seus gates. Consultar o
[lote 4](../reports/2026-10-13-pose-batch-4-authoring.md).

## Milestone: spec-facts-binding
- after: spec:pose-progressive-spec-surface
- target_start:
- target_due:
- specs: pose-spec-facts-evidence-binding

**Exit gate:** B30 distingue associated evidence de candidato não vinculado,
divulga binding/freshness e preserva reuso adequado; facts não geram judgment,
rationale ou conclusão semântica. B30 não depende artificialmente de B12/B13.

## Continuação prospectiva — lote 5

Consultar o [lote 5](../reports/2026-10-13-pose-batch-5-authoring.md). Preservar
R-IDs/status/baselines das duas specs abertas reutilizadas; o relatório formaliza
os planos prospectivos, sem alterar seus contratos históricos.

## Milestone: amendment-history-preclose
- after: spec:pose-action-target-and-node-scoping
- target_start:
- target_due:
- specs: pose-amendment-history-holds-before-close

**Exit gate:** continuar R1–R3 na spec existente: baseline da primeira emenda
derivada do start, warning in-progress e recusa antes de close. Reconciliar
artefatos/dependências antes de start; manter append-only.

## Milestone: terminal-closeout-verification
- after: amendment-history-preclose
- target_start:
- target_due:
- specs: pose-closeout-terminal-verification

**Exit gate:** B11 distingue done/no-op de verified/terminal; plano e closeout
compartilham verificação aplicável, sem gate duplicado de amendments ou
enforcement retrospectivo de legacy/cutoffs.

## Milestone: update-dry-run-qualification
- after:
- target_start:
- target_due:
- specs: pose-update-dry-run-reports-the-whole-update

**Exit gate:** B29 confronta implementação/revisões e a decisão registrada de
refazer a abordagem; compara preview/update em fixtures, incluindo source
discovery, symlinks/permissões, e reconcilia trace/evidência antes do closeout.
Código presente e trace histórico satisfeito não encerram o gap residual.
B29 é independente de B11; a escrita não executa update desta instância.


## Continuação prospectiva — lote 8

Consultar o [lote 8](../reports/2026-10-13-pose-batch-8-authoring.md).
B22/B23 estendem mecanismos existentes após B06; não alteram taxonomia,
compromissos ou dispositions das unidades ainda planejadas B24–B27/B42.

## Milestone: residual-material-identity
- after: spec:pose-source-coverage-integrity, spec:pose-debt-marker-lexical-precision, spec:pose-covered-followup-anchor-verification
- target_start:
- target_due:
- specs: pose-residual-material-coverage-and-identity

**Exit gate:** B22 trata menção a arquivo como associação candidata; compromisso
material e ocorrência delimitam cobertura. Ocorrências idênticas têm IDs distintos;
identidade rastreada sobrevive a rewrap/reordenação/inserção, e rename/migração
declara aliases/limites sem retarget automático. Qualificar P02/P03/P07 e paridade.

## Milestone: followup-metadata-routing
- after: residual-material-identity, spec:pose-source-coverage-integrity, spec:pose-followup-ownership-sla, spec:pose-followup-reconciliation-candidates
- target_start:
- target_due:
- specs: pose-followup-metadata-and-routing-integrity

**Exit gate:** B23 valida calendário, duplicação e metadata incompleta; SLA permanece
triagem. Destino abandoned/órfão/superseded/cíclico tem diagnóstico, resolução limitada
e adequação explícita; done isolado não resolve intenção. Qualificar P05/P06, sem
aplicar disposition automaticamente ou certificar complete após perdas de leitura.


## Continuação prospectiva — lote 11

Consultar o [lote 11](../reports/2026-10-13-pose-batch-11-authoring.md). Reusar o
ADR de obligations; schema/sintaxe novos exigem decisão prospectiva antes do código.
B25/B26/B42 permanecem futuros, sem triagem automática ou eficácia presumida.

## Milestone: proportional-next-action
- after: spec:pose-action-target-and-node-scoping, spec:pose-action-request-recovery-and-preflight, spec:pose-cli-mcp-operation-results, spec:pose-action-request-presentation, spec:pose-progressive-spec-surface
- target_start:
- target_due:
- specs: pose-proportional-next-action

**Exit gate:** B27 distingue holder/executor/pessoa/advisory e semântica de actor
query, reaproveita autorização suficiente e preserva request/digest/efeito na
retomada/handoff. Rotas trivial/bug/contract/decision são condicionais; triagem
em lote preserva answers e dedup é candidato. Quatro kinds retomam fase/target
correto com prova e negatives rastreáveis; apresentação não implementa delegação.

## Milestone: residual-nature-commitment
- after: residual-material-identity, followup-metadata-routing, spec:pose-obligation-contract, spec:pose-followup-reconciliation-candidates
- target_start:
- target_due:
- specs: pose-residual-nature-commitment-and-effect

**Exit gate:** B24 projeta dívida/defeito/incompletude/evidência/adoção/opcional com
natureza, compromisso e efeito separados; legado fica não classificado. Rótulo,
criticidade ou SLA não criam gate sozinhos, e compromisso vigente não desaparece
por encaminhamento ou renomeação. Sources/lifecycle/coverage preservados, sem
backlog paralelo, migração, triagem ou risco aceito automaticamente.


## Continuação prospectiva — lote 12

Consultar o [lote 12](../reports/2026-10-13-pose-batch-12-authoring.md).
B25/B26 estão materializados em draft; B42 continua piloto futuro. Não aplicar
triagem/risco ou métricas nesta escrita. Preservar fundações e ADR de obligations.

## Milestone: residual-triage-risk
- after: residual-nature-commitment, spec:pose-action-answers-reverify-on-read, spec:pose-review-disclosure-requires-proof, spec:pose-writer-coordination-and-atomic-publication, spec:pose-open-backlog-reconciliation
- target_start:
- target_due:
- specs: pose-residual-triage-and-risk-acceptance

**Exit gate:** B25 reconcilia corpus histórico 108 por snapshot/diff/fonte,
separado do inventário corrente; candidatos sem evidência/autoridade ficam abertos.
Preview/apply revalidam inputs, detectam concorrência e replay sem duplicação.
Risco aceito difere de wont-do, com razão/policy/validade/sucessão e gatilho de
triagem no vencimento; exceção material usa ActionRequest, sem disposition automática.

## Milestone: condition-work-metrics
- after: residual-nature-commitment, spec:pose-action-request-recovery-and-preflight, spec:pose-governance-wait-rework-observability
- target_start:
- target_due:
- specs: pose-condition-response-and-work-metrics

**Exit gate:** B26 separa resposta, condição e restrição; decline não encerra
restrição vigente. Reopening sem timestamp observado fica unknown e intervalos
são unidos por scope sem duplicação. Markers, intenções, cobertura, encaminhamento
e implementações devidas têm denominadores/limites próprios; idade vem de fatos,
nunca mtime. Não inferir produtividade ou eficácia do piloto por volume.

## Risk controls

- No aggregate efficiency or quality score; dimensions only.
- Automation is preview-first with explicit apply.
- Dogfooding is necessary but does not prove ergonomics for external adopters; record it as a limitation.
