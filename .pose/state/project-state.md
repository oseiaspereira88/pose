---
schema_version: 1
generated_at: 2026-09-26T00:52:30Z
baseline_commit: f1258058deec3f4ff1883d1bc9dd162de89a367c
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
<!-- state:derived hash:5305477e2a74 -->

- specs: total=233 draft=4 in-progress=65 blocked=0 done=164 superseded=0 abandoned=0
- roadmaps: total=12 active=1 done=11
- últimos closeouts:
  - spec:pose-review-root-binding-parity (2026-09-26)
  - spec:pose-agent-project-context (2026-09-25)
  - spec:pose-spec-authority-transfer (2026-09-24)
  - spec:pose-federated-roadmap-acceptance (2026-09-24)
  - spec:check-worker-count-is-the-machines (2026-09-21)
  - ... e mais 159 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:211b98f49a1c -->

- abertos: 111
- por criticidade: high=2 medium=25 low=60 sem-classificação=24
- vencidos (review < hoje): 1
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)

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
<!-- state:derived hash:dbacef1ac3f7 -->

- último registro: task=validate-native outcome=pass (2026-09-26T00:51:26Z)
- últimos 30 dias: total=91 outcome_ok=80 outcome_outro=11
- reports revisados (.md): total=146
  - report:2026-09-26-standard-validate-native.md
  - report:2026-09-25-standard-validate-native.md
  - report:2026-09-24-multirepo-autonomous-review.md
  - report:2026-09-24-standard-validate-native.md
  - report:2026-09-21-standard-validate-native.md

## Arquitetura
<!-- state:derived hash:ca0af4a1a681 status:active -->

- componentes: total=3 verificados=3 completude=100.0%
- linhas_de_codigo: producao=50787 testes=38806 total=89593
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=57 ativos=1 gaps=56
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
