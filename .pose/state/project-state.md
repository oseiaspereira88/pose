---
schema_version: 1
generated_at: 2026-10-22T12:43:23Z
baseline_commit: e58ca80ef26e3ed0220e9b588cece5576d8c511f
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
<!-- state:derived hash:9e792935ab49 -->

- specs: total=446 draft=34 in-progress=18 blocked=0 done=393 superseded=0 abandoned=1
- roadmaps: total=22 active=3 done=11
- últimos closeouts:
  - spec:pose-action-target-and-node-scoping (2026-10-22)
  - spec:pose-git-and-dispatch-snapshot-integrity (2026-10-21)
  - spec:pose-retire-unused-first-party-action-exception (2026-10-21)
  - spec:pose-ci-security-refresh (2026-10-21)
  - spec:pose-release-manifest-tag-integrity (2026-10-19)
  - ... e mais 388 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:9fc3c59e7c15 -->

- abertos: 133
- por criticidade: high=4 medium=23 low=87 sem-classificação=19
- vencidos (review < hoje): 25
- cobertura: INCOMPLETA — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md; invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md; os números acima são o que pôde ser lido
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 15 vencidos (ver `pose followups --open`)

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
<!-- state:derived hash:ddef542325a8 -->

- último registro: task=validate-native outcome=pass (2026-10-22T11:06:06Z)
- últimos 30 dias: total=86 outcome_ok=63 outcome_outro=23
- reports revisados (.md): total=241
  - report:2026-10-22-b38-pending-reconciliation.md
  - report:2026-10-22-b38-pending-review-r1.md
  - report:2026-10-22-standard-validate-native.md
  - report:2026-10-20-standard-b07-snapshot-integrity.md
  - report:2026-10-21-b08-implementation.md

## Arquitetura
<!-- state:derived hash:9717eb5ab73b status:active -->

- componentes: total=3 verificados=3 completude=99.8%
- linhas_de_codigo: producao=79299 testes=69389 total=148688
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=64 ativos=1 gaps=63
- divida_tecnica: total=1 coberta=1 descoberta=0 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
