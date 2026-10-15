---
schema_version: 1
generated_at: 2026-10-14T22:53:36Z
baseline_commit: 319cd0a3200eafc845e9eeba3e0600c893e44e89
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
<!-- state:derived hash:aedf92e12ea1 -->

- specs: total=438 draft=40 in-progress=20 blocked=0 done=377 superseded=0 abandoned=1
- roadmaps: total=22 active=3 done=11
- últimos closeouts:
  - spec:pose-historical-trust-and-review-reuse (2026-10-14)
  - spec:pose-delegated-review-brief (2026-10-10)
  - spec:pose-delegated-review-dispatch (2026-10-10)
  - spec:pose-v7-4-1-version-alignment (2026-10-10)
  - spec:pose-v7-4-0-version-alignment (2026-10-10)
  - ... e mais 372 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:b971be01bdeb -->

- abertos: 110
- por criticidade: high=4 medium=24 low=63 sem-classificação=19
- vencidos (review < hoje): 12
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 2 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:7db5fb52757a -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (0 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:f78c15dbd473 -->

- ADRs: total=63
  - adr:2026-10-04-obligations-are-projected-action-requests-are-persisted.md
  - adr:2026-10-04-mcp-governance-write-risk-class.md
  - adr:2026-10-02-trusted-dependabot-runtime-repairs.md
  - adr:2026-10-02-release-version-source-is-declared-by-the-project.md
  - adr:2026-10-01-scoped-git-batch-reader-for-structural-assessment.md
- knowledge: total=7 ativo=7 expirado=0

## Validação & Evidência
<!-- state:derived hash:1c9983a8de53 -->

- último registro: task=validate-native outcome=fail (2026-10-13T04:23:26Z)
- últimos 30 dias: total=100 outcome_ok=76 outcome_outro=24
- reports revisados (.md): total=198
  - report:2026-10-14-pose-implementation-b04.md
  - report:2026-10-14-b04-review-brief.md
  - report:pose-abm-capability-adoption.md
  - report:pose-agency-readiness-pilot.md
  - report:pose-v6-2-0-autonomous-work.md

## Arquitetura
<!-- state:derived hash:751375eda7d7 status:active -->

- componentes: total=3 verificados=3 completude=100.0%
- linhas_de_codigo: producao=70779 testes=56168 total=126947
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=64 ativos=1 gaps=63
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
