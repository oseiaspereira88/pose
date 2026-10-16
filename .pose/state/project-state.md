---
schema_version: 1
generated_at: 2026-10-16T04:41:54Z
baseline_commit: e08138c7e71872463e363dcf064ad555872fb6b9
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
<!-- state:derived hash:ed7d135ad502 -->

- specs: total=439 draft=39 in-progress=18 blocked=0 done=381 superseded=0 abandoned=1
- roadmaps: total=22 active=3 done=11
- últimos closeouts:
  - spec:pose-review-approval-domain-preflight (2026-10-16)
  - spec:pose-dist-instance-follows-engine (2026-10-16)
  - spec:pose-review-disclosure-requires-proof (2026-10-15)
  - spec:pose-action-answers-reverify-on-read (2026-10-15)
  - spec:pose-historical-trust-and-review-reuse (2026-10-14)
  - ... e mais 376 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:3d469f56370a -->

- abertos: 108
- por criticidade: high=4 medium=22 low=63 sem-classificação=19
- vencidos (review < hoje): 14
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 4 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:e20a9224b717 -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (59 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:6b7e7f699ecb -->

- ADRs: total=69
  - adr:2026-10-14-shared-domain-and-optional-worktree-runtime.md
  - adr:2026-10-14-review-time-context-and-reuse-provenance.md
  - adr:2026-10-14-release-temporal-evidence-and-attempt-lineage.md
  - adr:2026-10-09-obligations-may-originate-on-the-project.md
  - adr:2026-10-09-delegated-review-is-an-adapter.md
- knowledge: total=11 ativo=11 expirado=0

## Validação & Evidência
<!-- state:derived hash:78529f3eb571 -->

- último registro: task=validate-native outcome=fail (2026-10-13T04:23:26Z)
- últimos 30 dias: total=99 outcome_ok=75 outcome_outro=24
- reports revisados (.md): total=203
  - report:pose-abm-capability-adoption.md
  - report:pose-agency-readiness-pilot.md
  - report:pose-v6-2-0-autonomous-work.md
  - report:pose-v7-legacy-cleanup-plan.md
  - report:2026-10-16-pose-b02-domain-preflight-review.md

## Arquitetura
<!-- state:derived hash:1a6ee06cee0b status:active -->

- componentes: total=3 verificados=3 completude=100.0%
- linhas_de_codigo: producao=70779 testes=56339 total=127118
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=64 ativos=1 gaps=63
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
