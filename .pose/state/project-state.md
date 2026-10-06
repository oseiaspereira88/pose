---
schema_version: 1
generated_at: 2026-10-06T03:46:24Z
baseline_commit: 4d053a23723cf309a4f754512dc746293e3feadf
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
<!-- state:derived hash:6e4336f1c0f8 -->

- specs: total=343 draft=2 in-progress=32 blocked=0 done=309 superseded=0 abandoned=0
- roadmaps: total=17 active=1 done=11
- últimos closeouts:
  - spec:pose-validation-check-additions-are-not-material (2026-10-06)
  - spec:pose-followup-reconciliation-candidates (2026-10-05)
  - spec:pose-attention-federated-parity (2026-10-05)
  - spec:pose-review-attribution-roles (2026-10-05)
  - spec:pose-v7-legacy-cleanup-plan (2026-10-05)
  - ... e mais 304 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:e9b044be7c74 -->

- abertos: 103
- por criticidade: high=4 medium=24 low=52 sem-classificação=23
- vencidos (review < hoje): 8
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)

## Capabilities
<!-- state:derived hash:7db5fb52757a -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (0 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:b8f47d8fd574 -->

- ADRs: total=44
  - adr:2026-08-15-retired-machinery-files-stay-on-disk-never-auto-migrated-by-pose-update.md
  - adr:2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type.md
  - adr:2026-08-14-unified-review-convergence-and-auto-attestation.md
  - adr:2026-08-13-sealed-review-bundles-and-attestations.md
  - adr:2026-08-12-component-aware-effective-review-plans.md
- knowledge: total=10 ativo=10 expirado=0

## Validação & Evidência
<!-- state:derived hash:d3076e2addb6 -->

- último registro: task=validate-native outcome=pass (2026-10-02T11:43:40Z)
- últimos 30 dias: total=139 outcome_ok=117 outcome_outro=22
- reports revisados (.md): total=166
  - report:2026-10-06-v7-validation-check-materiality-review.md
  - report:2026-10-pose-open-backlog-reconciliation.md
  - report:pose-abm-capability-adoption.md
  - report:pose-agency-readiness-pilot.md
  - report:pose-v7-legacy-cleanup-plan.md

## Arquitetura
<!-- state:derived hash:19a4164022b2 status:active -->

- componentes: total=3 verificados=3 completude=100.0%
- linhas_de_codigo: producao=54413 testes=43811 total=98224
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=58 ativos=1 gaps=57
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
