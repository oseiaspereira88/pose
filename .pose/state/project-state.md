---
schema_version: 1
generated_at: 2026-11-02T23:47:37Z
baseline_commit: 1bf72b3d03b229b4d0240bd0dca0a07ad97b2ef1
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
<!-- state:derived hash:77c5843a44cb -->

- specs: total=481 draft=18 in-progress=17 blocked=0 done=444 superseded=0 abandoned=2
- roadmaps: total=23 active=3 done=16
- últimos closeouts:
  - spec:pose-roadmap-reviews-findings (2026-11-02)
  - spec:pose-evidence-gate-means-validation (2026-11-02)
  - spec:pose-engine-reads-raw-history (2026-11-02)
  - spec:pose-effects-reported-for-what-they-did (2026-11-02)
  - spec:pose-milestone-review-proportional (2026-11-02)
  - ... e mais 439 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:c293f3f955e1 -->

- abertos: 218
- por criticidade: high=5 medium=40 low=154 sem-classificação=19
- vencidos (review < hoje): 34
- cobertura: INCOMPLETA — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md; invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md; os números acima são o que pôde ser lido
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 24 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:e20a9224b717 -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (59 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:ada13748516c -->

- ADRs: total=93
  - adr:2026-11-02-a-milestone-of-one-member-is-that-member.md
  - adr:2026-11-01-a-milestone-answers-to-its-own-edges.md
  - adr:2026-10-31-review-reads-raw-history-and-roots-diff-from-the-empty-tree.md
  - adr:2026-10-29-make-delegated-review-adoption-explicit-and-recoverable.md
  - adr:2026-10-26-delegation-is-a-first-class-execution-surface.md
- knowledge: total=18 ativo=18 expirado=0

## Validação & Evidência
<!-- state:derived hash:de665bd9ff23 -->

- último registro: task=validate-native outcome=pass (2026-10-24T18:00:55Z)
- últimos 30 dias: total=41 outcome_ok=27 outcome_outro=14
- reports revisados (.md): total=285
  - report:2026-11-02-pose-v7-legacy-cleanup-both-corpora.md
  - report:2026-10-29-launcher-parity-failing-first.md
  - report:delegation-adoption-review-round4.md
  - report:delegation-adoption-review-round3.md
  - report:delegation-adoption-review-round2.md

## Arquitetura
<!-- state:derived hash:e6be3f58e285 status:active -->

- componentes: total=3 verificados=3 completude=99.8%
- linhas_de_codigo: producao=102055 testes=94666 total=196721
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=70 ativos=6 gaps=64
- divida_tecnica: total=1 coberta=1 descoberta=0 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
