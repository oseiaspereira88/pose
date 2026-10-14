---
slug: pose-release-publication-evidence-replay
status: draft
created_at: 2026-10-14
completed_at:
depends_on: pose-historical-trust-and-review-reuse, pose-release-manifest-tag-integrity, pose-release-lifecycle-closure, pose-reproducible-release-verification
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:release-publication-evidence-replay, governance:release-publication-evidence-replay
---

# Spec: Revalidar evidência temporal e cobertura de publicação no replay

## 1. Intent

### Goal

Impedir que digest declarado, asset divergente ou provider sem vínculo elevem uma projeção a verified ou autorizem open-next.

### Business value

B39 / F06-05, F06-09 e F06-10. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de release](../reports/2026-10-12-pose-front-06-release-sufficiency.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Revalidar no replay schema, identidade version/tag/commit, evidence digest recomputado e transição dos eventos, além da validação de entrada. P09 com assets alterados e digest antigo deve produzir gap/invalid/unknown adequado, nunca verified corrente; preservar bytes para diagnóstico e não corrigir journals silenciosamente.
- R2: P07 deve comparar publicação e verificação por identidade de asset e digest, incluindo publication binding. Verificação parcial declara subconjunto, assets extras têm papel/regra explícitos e missing/mismatched assets não passam como cobertura total. Não aceitar só equality do publication_digest declarado.
- R3: P08 deve vincular repository/tag/commit e identidade segundo o papel: tag provider git, publisher github e verifier github-actions podem diferir legitimamente. Origem de publicação segue a policy aplicável e verifier segue contrato/prova pertinentes; presença de string provider ou URL não autentica origem.
- R4: Consumir trust/reuso material de B04 e o verifier independente existente, sem reinventar assinatura/provenance no ledger. Divulgar publisher, verifier, run e assurance realmente observados; import de relatório tipado sozinho não prova origem, julgamento independente ou execução em todas as plataformas.
- R5: Divulgar matriz de assets/plataformas/canais e forma de verificação: execução nativa, inspeção de digest/assinatura/provenance, unavailable/unknown e limitações. Linux executado não vira teste nativo multiplataforma nem disponibilidade simultânea em todos os canais; prepared snapshot, publicação observada e verificação têm resultados distintos em CLI/script/MCP.
- R6: P05 deve distinguir candidato corrente stale de snapshot histórico íntegro. Revalidar inputs atuais para candidato conforme policy; história usa bindings e contrato do cut, sem ficar inválida só pelo avanço de versão/policy ou abertura do ciclo seguinte. Eventos adulterados e prova inadequada continuam detectados também na leitura histórica; o modo histórico não dispensa verificação pertinente de trust.
- R7: Formalizar adoção/versionamento prospectivos e migration preview de contratos temporais antes do código, conforme ADR proposto. Knobs tag_pattern/require_verification expõem supported/configured/adopted/effective e limites, sem anunciar enforcement que não existe ou impor novo gate retrospectivo indiscriminadamente.
- R8: Preservar lineage de tentativas/retries/conflitos e invalidação da prova que referencia publicação superseded, sem sobrescrever publicação/tag/manifests ou herdar verified de outra tentativa. Reconciliar o contrato atual de um evento por state com a proposta de lineage; sintaxe/schema e tratamento de correções exigem decisão explícita antes de start. B41 fornece persistência/coordenação, sem dependência artificial B41 para a validação do reader.
- R9: Open-next e demais consumidores devem usar o resultado revalidado no mesmo recorte, com proof binding e coverage exigidos pela policy aplicável; nunca somente State=verified da projeção antiga. Registro factual de publicação externa validamente observado não é impedido por falta de autorização retrospectiva para o cut; divulgar não conformidade e limitar assurance sem fabricar aprovação.
- R10: Qualificar P05/P07/P08/P09, controles P12 e history/next-cycle em fixtures sintéticas e paridade CLI/script/MCP. Reusar lifecycle e verificação done, completar inventário dos schemas/produtores/consumers antes de start; leitura não publica release, aplica migração ou modifica policy.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-13-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-14-pose-release-publication-evidence-replay.md
- created: .pose/starts/pose-release-publication-evidence-replay.json
- created: .pose/changelogs/unreleased/pose-release-publication-evidence-replay.md
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/pose/release_lifecycle_test.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/mcpserver/release_status_tool_test.go
- modified: scripts/release.sh
- modified: tests/release/independent-verify.sh
- modified: .github/workflows/verify-release.yml
- modified: .pose/adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md
- created: pose-mcp/internal/pose/release_evidence_replay_test.go
- created: pose-mcp/internal/cli/release_temporal_evidence_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:release-publication-evidence-replay module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/release_lifecycle.go

- governance:release-publication-evidence-replay module:. profile:release-governance entrypoint:.github/workflows/verify-release.yml

### Approach and execution gates

Executar após B04/B38. Reusar ProjectRelease/GetReleaseStatus/LoadReleaseEvents, snapshot/tag integrity e independent verification. Confrontar os decoders atuais, contratos de identity/assets e consumidores open-next/scripts; completar paths dos schemas e report producers antes de start. O [ADR temporal proposto](../adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md) e knowledge:adr-release-temporal-evidence-review registram a proposta, não adoção; reconciliar/aceitar decisão de compatibilidade e schema antes do código. B39 define validade/lineage, B41 a persistência recuperável e B40 a seleção integrada, evitando ciclo nativo entre B39/B41.

## 5. Decisions

### Decision D1
- Proposal: [Temporal evidence and attempt lineage](../adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md), status Proposed; knowledge:adr-release-temporal-evidence-review.
- Basis: R1, R2, R3, R4, R5, R6, R7, R8, R9, R10
- Decision: Revalidar prova e coverage no replay, separando candidato corrente, história do cut e observação factual de publicação.
- Rationale: Digest declarado e estado projetado não substituem recomputação, trust e coverage; avanço de ciclo não reescreve história íntegra.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P09: payload adulterado com digest antigo e evento íntegro | Recomputação denuncia adulteração; positivo válido preservado. |
| V2 — P07: assets ausentes/diferentes, subconjunto e relatório extra | Coverage parcial explícita; total somente com conjunto/digests adequados. |
| V3 — P08: git/github/github-actions legítimos e repo/tag/commit errados | Papéis preservados; identidade inválida diagnosticada sem igualar providers. |
| V4 — Relatório importado sem origem comprovada e verifier autenticado | Assurance observada; nenhum judgment/origem inventado. |
| V5 — Linux nativo, outras plataformas inspecionadas e canal indisponível | Matriz qualificada, sem claim multiplataforma/simultâneo indevido. |
| V6 — P05: candidato stale, história íntegra e história adulterada após próximo ciclo | Modos temporais explícitos; advancement não staleia história correta, tampering é detectado. |
| V7 — Policy knobs, legacy/cutoff e migration preview | Suporte/compatibilidade/efeito observados, sem enforcement retrospectivo silencioso. |
| V8 — Retry/conflito, publicação superseded e open-next | Lineage preservada, prova antiga não autoriza consumidor corrente. |
| V9 — Import factual de publicação não autorizada previamente e controles P12 | Fato validado divulgado com não conformidade; não inventa approval ou verified. |
| V10 — CLI/script/MCP e consultas repetidas | Mesmo resultado/coverage; zero writes e ausência de release/migração. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V1, V3, V6 |
| R2 | V2, V8 |
| R3 | V3, V4 |
| R4 | V4, V5 |
| R5 | V5, V10 |
| R6 | V1, V6 |
| R7 | V7 |
| R8 | V8 |
| R9 | V8, V9, V10 |
| R10 | V1–V10 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 13. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-14-pose-batch-13-authoring.md).

### Residual risks

Contratos temporais e lineage requerem aceitação prospectiva do ADR e compatibilidade/schema antes de start. Trust ausente e cobertura parcial limitam claims. Estes casos não foram executados no lote documental; história não é reescrita nem publicação mutável autorizada.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
