---
schema_version: 1
generated_at: 2026-09-28T01:32:03Z
baseline_commit: 06c2c6098ae236cd88e512205a56c832c2816783
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
<!-- state:derived hash:eb8e5b1ce99b -->

- specs: total=251 draft=5 in-progress=65 blocked=0 done=181 superseded=0 abandoned=0
- roadmaps: total=12 active=1 done=11
- últimos closeouts:
  - spec:pose-governance-outcomes-v2 (2026-09-28)
  - spec:pose-federated-milestone-scoped-acceptance (2026-09-27)
  - spec:pose-review-bundle-classifies-transfer-records (2026-09-27)
  - spec:pose-check-dor-accepts-qualified-refs (2026-09-27)
  - spec:pose-federated-external-milestone-members (2026-09-27)
  - ... e mais 176 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:370b6c6971e1 -->

- abertos: 119
- por criticidade: high=7 medium=28 low=60 sem-classificação=24
- vencidos (review < hoje): 2
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)

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
<!-- state:derived hash:3d31225f71f2 -->

- último registro: task=validate-native outcome=pass (2026-09-27T21:05:43Z)
- últimos 30 dias: total=116 outcome_ok=104 outcome_outro=12
- reports revisados (.md): total=148
  - report:2026-09-27-standard-validate-native.md
  - report:2026-09-26-standard-validate-native.md
  - report:2026-09-26-federated-carried-forward-proof.md
  - report:2026-09-25-standard-validate-native.md
  - report:2026-09-24-multirepo-autonomous-review.md

## Arquitetura
<!-- state:derived hash:f5e07bfd7492 status:active -->

- componentes: total=3 verificados=3 completude=100.0%
- linhas_de_codigo: producao=52968 testes=40879 total=93847
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=58 ativos=1 gaps=57
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
