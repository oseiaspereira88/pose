---
schema_version: 1
generated_at: 2026-08-08T03:39:37Z
baseline_commit: 9a92252825dede38fd79913e7dc779fc96523944
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
<!-- state:derived hash:f9b385b8600d -->

- specs: total=71 draft=1 in-progress=0 blocked=0 done=70 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-sbom-license-coverage-gate (2026-08-08)
  - spec:pose-docs-asset-parity (2026-08-08)
  - spec:pose-compat-gate-candidate-integrity (2026-08-07)
  - spec:pose-installer-local-binary-precedence (2026-08-07)
  - spec:pose-verifier-assets-variable-fix (2026-08-07)
  - ... e mais 65 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:373d72e02e89 -->

- abertos: 33
- por criticidade: high=0 medium=14 low=19 sem-classificação=0
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:78115c60373b -->

- assessment: presente, baseline_commit=commit:c9a08fa, assessed_at=2026-07-22 (17 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:4709d372b486 -->

- ADRs: total=37
  - adr:2026-08-06-mcp-active-context-authorized-discovery.md
  - adr:2026-08-03-immutable-release-ledger.md
  - adr:2026-08-02-immutable-hierarchical-review-and-closeout-evidence.md
  - adr:2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
  - adr:2026-07-19-versioned-validation-result-contract.md
- knowledge: total=2 ativo=2 expirado=0

## Validação & Evidência
<!-- state:derived hash:421c63be46ed -->

- último registro: task=closeout-pose-sbom-license-coverage-gate outcome=unknown (2026-08-08T02:55:33Z)
- últimos 30 dias: total=126 outcome_ok=100 outcome_outro=26
- reports revisados (.md): total=63
  - report:2026-08-08-standard-closeout-pose-sbom-license-coverage-gate.md
  - report:2026-08-08-standard-closeout-pose-docs-asset-parity.md
  - report:2026-08-08-standard-closeout-pose-dependency-digest-pinning-docs-lock.md
  - report:2026-08-08-standard-closeout-pose-dependency-digest-pinning.md
  - report:2026-08-07-standard-closeout-pose-package-channel-workflow-safety.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
