---
schema_version: 1
generated_at: 2026-11-07T07:38:50Z
baseline_commit: b6ef1e54f36e5b7cd5c4946ed42933d28949782b
staleness_policy: max_age_days=7,max_commits=20
refresh_pending: 
---

# Project State

## Resumo executivo
<!-- state:curated -->

`pose-dist` é a distribuição standalone do POSE (Project Operating Standard for
Engineering): o binário Go nativo (`pose`), servidor MCP, docs-site e todo o
material publicável do produto POSE em si — trabalha em paralelo ao repositório
`harne8`, que consome esta distribuição via `pose-dist/pose-mcp` como
dependência do próprio ecossistema.

## Direção atual
<!-- state:curated -->

Trabalho corrente puxado pelo repositório `harne8` (roadmap
`harne8-semantic-state-governance`): este ciclo adicionou o artefato nativo de
`project-state` (`pose state`/`pose_project_state`) ao motor Go. Backlog nativo
próprio deste repositório (produtização v2, DX) segue disponível conforme
capacidade.

## Specs & Roadmaps
<!-- state:derived hash:4447b87a344c -->

- specs: total=482 draft=16 in-progress=0 blocked=0 done=464 superseded=0 abandoned=2
- roadmaps: total=23 active=3 done=16
- últimos closeouts:
  - spec:pose-governance-read-view-generation (2026-11-06)
  - spec:pose-consolidation-batch-14-authoring (2026-11-03)
  - spec:pose-consolidation-batch-4-authoring (2026-11-03)
  - spec:pose-governance-operation-performance (2026-11-03)
  - spec:pose-consolidation-batch-7-authoring (2026-11-03)
  - ... e mais 459 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:b1d19f90cd74 -->

- abertos: 221
- por criticidade: high=5 medium=43 low=154 sem-classificação=19
- vencidos (review < hoje): 42
- cobertura: INCOMPLETA — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md; invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md; os números acima são o que pôde ser lido
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 32 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:e20a9224b717 -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (59 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:6d92b4c709dd -->

- ADRs: total=94
  - adr:2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
  - adr:2026-11-02-a-milestone-of-one-member-is-that-member.md
  - adr:2026-11-01-a-milestone-answers-to-its-own-edges.md
  - adr:2026-10-31-review-reads-raw-history-and-roots-diff-from-the-empty-tree.md
  - adr:2026-10-29-make-delegated-review-adoption-explicit-and-recoverable.md
- knowledge: total=21 ativo=21 expirado=0

## Validação & Evidência
<!-- state:derived hash:21ed16dff8a2 -->

- último registro: task=validate-native outcome=pass (2026-10-24T18:00:55Z)
- últimos 30 dias: total=41 outcome_ok=27 outcome_outro=14
- reports revisados (.md): total=290
  - report:2026-11-06-pose-b34-global-review-view-budget.md
  - report:2026-11-05-pose-b34-read-view-capacity-calibration.md
  - report:2026-11-05-pose-b34-dependency-budget-measurement.md
  - report:pose-abm-capability-adoption.md
  - report:pose-agency-readiness-pilot.md

## Arquitetura
<!-- state:derived hash:00ac0f3186cc status:active -->

- componentes: total=3 verificados=3 completude=99.8%
- linhas_de_codigo: producao=110745 testes=103751 total=214496
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=70 ativos=6 gaps=64
- divida_tecnica: total=1 coberta=1 descoberta=0 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
