---
schema_version: 1
generated_at: 2026-10-29T18:28:04Z
baseline_commit: 56dbe4dcbec2c97f5aab50b744bba7c1361316ba
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
<!-- state:derived hash:ae1b59020f8f -->

- specs: total=456 draft=19 in-progress=18 blocked=0 done=418 superseded=0 abandoned=1
- roadmaps: total=23 active=3 done=11
- últimos closeouts:
  - spec:pose-delegated-review-capability (2026-10-29)
  - spec:pose-delegated-review-action-fulfillment (2026-10-28)
  - spec:pose-delegated-verdict-destinations (2026-10-28)
  - spec:pose-delegated-review-conclusions (2026-10-28)
  - spec:pose-delegated-review-round-protocol (2026-10-27)
  - ... e mais 413 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:f13b2fabc372 -->

- abertos: 179
- por criticidade: high=4 medium=29 low=127 sem-classificação=19
- vencidos (review < hoje): 31
- cobertura: INCOMPLETA — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md; invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md; os números acima são o que pôde ser lido
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 21 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:e20a9224b717 -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (59 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:9a109e7387b1 -->

- ADRs: total=90
  - adr:2026-10-29-make-delegated-review-adoption-explicit-and-recoverable.md
  - adr:2026-10-26-delegation-is-a-first-class-execution-surface.md
  - adr:2026-10-25-reserve-delegated-attempts-before-launch-and-witness-immutable-runs.md
  - adr:2026-10-25-continue-delegated-runs-under-one-immutable-attempt.md
  - adr:2026-10-25-bound-trusted-reviewer-execution-and-disclose-unavailable-isolation.md
- knowledge: total=18 ativo=18 expirado=0

## Validação & Evidência
<!-- state:derived hash:9b28ebc983c6 -->

- último registro: task=validate-native outcome=pass (2026-10-24T18:00:55Z)
- últimos 30 dias: total=53 outcome_ok=33 outcome_outro=20
- reports revisados (.md): total=283
  - report:delegation-adoption-review-round4.md
  - report:delegation-adoption-review-round3.md
  - report:delegation-adoption-review-round2.md
  - report:delegation-adoption-review-round1.md
  - report:pose-abm-capability-adoption.md

## Arquitetura
<!-- state:derived hash:b6c9fab3f931 status:active -->

- componentes: total=3 verificados=3 completude=99.8%
- linhas_de_codigo: producao=99081 testes=85213 total=184294
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=64 ativos=1 gaps=63
- divida_tecnica: total=1 coberta=1 descoberta=0 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
