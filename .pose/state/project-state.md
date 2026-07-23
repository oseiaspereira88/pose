---
schema_version: 1
generated_at: 2026-07-23T23:35:15Z
baseline_commit: 3c1c9fb86abe134aede19b3c5a9653e03fdd3fbf
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
<!-- state:derived hash:b31925459089 -->

- specs: total=36 draft=0 in-progress=0 blocked=0 done=36 superseded=0 abandoned=0
- roadmaps: total=7 active=0 done=7
- últimos closeouts:
  - spec:pose-capability-mechanism (2026-07-21)
  - spec:pose-otel-observability (2026-07-19)
  - spec:pose-agent-skills-conformance (2026-07-19)
  - spec:pose-changed-scope-validation (2026-07-19)
  - spec:pose-cross-repo-portfolio (2026-07-19)
  - ... e mais 31 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:7af64d34d2f1 -->

- abertos: 33
- por criticidade: high=5 medium=12 low=15 sem-classificação=1
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:c6059a141abf -->

- assessment: presente, baseline_commit=commit:c9a08fa, assessed_at=2026-07-22 (1 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:96e441203205 -->

- ADRs: total=33
  - adr:2026-07-19-versioned-validation-result-contract.md
  - adr:2026-07-19-verified-public-install-contract.md
  - adr:2026-07-19-validation-runtime-guardrails-and-harness-delegation.md
  - adr:2026-07-19-upgrade-compatibility-lab-populated-fixtures.md
  - adr:2026-07-19-slsa-build-l2-provenance-claim.md
- knowledge: total=1 ativo=1 expirado=0

## Validação & Evidência
<!-- state:derived hash:fe9ec417d781 -->

- último registro: task=validate-native outcome=pass (2026-07-19T20:11:07Z)
- últimos 30 dias: total=26 outcome_ok=25 outcome_outro=1
- reports revisados (.md): total=3
  - report:2026-07-19-standard-validate-native.md
  - report:2026-07-18-doc-audit-product-roadmap-portfolio.md
  - report:README.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).
