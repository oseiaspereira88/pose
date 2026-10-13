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

## Risk controls

- No aggregate efficiency or quality score; dimensions only.
- Automation is preview-first with explicit apply.
- Dogfooding is necessary but does not prove ergonomics for external adopters; record it as a limitation.
