---
slug: pose-historical-trust-and-review-reuse
status: draft
created_at: 2026-10-13
completed_at:
depends_on: pose-material-equivalence-reuse, pose-native-attestation-issuer
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:historical-trust-and-review-reuse
---

# Spec: Separar validade histórica, autorização atual e proveniência de reuso

## 1. Intent

### Goal

Verificar tempo e fonte de reuso com resultados distintos para autenticidade, validade temporal histórica e admissibilidade de uma operação atual.

### Business value

B04 / T2-06 e T2-07. #157 corrigiu apenas o tempo de action answers; #158 fez disclosure consumir prova criptográfica. Reutilizar knowledge:action-answer-proof-and-historical-time e knowledge:review-assurance-requires-cryptographic-proof; completar o delta de reviews e reuso. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise de confiança](../reports/2026-10-11-pose-front-02-trust-chain.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: A verificação temporal deve receber um relógio/contexto explícito, recusar emissão futura fora da tolerância admitida, expiração anterior à emissão e expiração anterior à aceitação aplicável, com limites de igualdade e precisão documentados e testes determinísticos.
- R2: O resultado deve distinguir validade histórica do claim, confiança/autorização atual e base temporal usada. Timestamp afirmado por issuer, timestamp observado e dado legado editável não devem ser apresentados como a mesma evidência; não inferir autorização histórica imutável de data ou label sem prova.
- R3: Uma aceitação histórica legitimamente comprovada não deve desaparecer somente por expiração posterior; uma nova operação deve ser avaliada no seu próprio tempo e nas exigências atuais aplicáveis. Remoção de pin/grant e evidência histórica insuficiente devem ter diagnóstico próprio, sem inventar receipt retrospectivo.
- R4: O reuso apresentado como passagem anterior comprovada deve validar os bindings, prova e autoridade exigidas da fonte, sua decisão/findings, o critério passed e seu contrato/inputs selados; assinatura alterada, fonte negativa, critério inválido ou indisponibilidade não devem satisfazer essa modalidade.
- R5: Uma conclusão atual que reassume explicitamente uma referência histórica deve registrar a autoria e o julgamento atuais, conservar a referência e distinguir essa modalidade de reuso de passagem comprovada; a referência não deve dispensar os critérios da aprovação atual.
- R6: A travessia de fontes deve detectar ciclos e limitar profundidade, nós e custo; memoização deve ser por operação e por identidade/digest/contexto de confiança relevantes. Limite ou falha deve produzir diagnóstico inconclusivo, nunca sucesso ou fonte ausente silenciosamente.
- R7: Domínio, verify, disclosure, review-check, closeout, CLI JSON/humano e MCP devem usar o mesmo resultado temporal/de proveniência, preservando a distinção entre identidade criptográfica e aprovação semântica.
- R8: O delta deve preservar declared legado, rotação com pin histórico mantido, claim sem expiry quando admitido pelo contrato, precisão histórica e regressões de action answers do #157; não impor TTL universal, novo critério retrospectivo ou bloqueio de escopo alheio.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-1-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- created: .pose/specs/2026-10-13-pose-historical-trust-and-review-reuse.md
- created: .pose/starts/pose-historical-trust-and-review-reuse.json
- created: .pose/changelogs/unreleased/pose-historical-trust-and-review-reuse.md
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_trust_time.go
- created: pose-mcp/internal/pose/review_trust_time_test.go
- created: pose-mcp/internal/pose/review_reuse_provenance.go
- created: pose-mcp/internal/pose/review_reuse_provenance_test.go
- created: pose-mcp/internal/cli/review_trust_reuse_cli_test.go
- created: pose-mcp/internal/mcpserver/review_trust_reuse_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:historical-trust-and-review-reuse module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_bundle.go

### Approach and execution gates

Extrair avaliação temporal contextual e verificação limitada de fonte, reutilizando os verificadores criptográficos de B01. Definir tolerância explícita, conservadora por padrão, sem tolerância escondida ou credibilidade de relógio fabricada. Manter revogação/pins/grants atuais como fronteira explícita enquanto não existir prova histórica suficiente. Registrar a base de confiança limitada do legado; não fabricar histórico de policy. Qualificar os contratos de histórico e operação nova antes de mudar o verdict de registros existentes.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R2, R3, R4, R5
- Decision: Separar passagem histórica comprovada de reassunção atual; exigir diagnóstico explícito para prova temporal/autorização histórica insuficiente.
- Rationale: Uma data editável ou uma fonte referenciada não é um receipt; nenhum dos caminhos pode dispensar o julgamento atual devido.

### Decision D2
- Basis: R6
- Decision: Limitar travessia e memoização por operação antes de ampliar o reuso.
- Rationale: Verificação recursiva sem orçamento pode trocar um gap de confiança por ciclo ou custo incontrolável.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Tempo futuro/reverso/expirado e igualdade | Recusa ou aceite conforme contexto/tolerância explícitos; teste não usa wall clock real. |
| V2 — Aceitação válida seguida de expiry; ausência de expiry | Histórico preservado quando comprovado; operação nova avaliada separadamente. |
| V3 — Pin/grant removido e tempo legado alterado | Diagnóstico de confiança/base temporal insuficiente; sem promoção de autorização histórica. |
| V4 — Fonte com assinatura/bindings/decision/criterion adulterados | Reuso comprovado recusado com diagnóstico localizado. |
| V5 — Reassunção atual de referência histórica | Julgamento/autoria atuais explícitos; sem rotular a referência como prova antiga. |
| V6 — Ciclo, cadeia profunda, fan-out e fonte ilegível | Budget/ciclo/inconclusivo explícitos; sem loop, sucesso parcial ou custo ilimitado. |
| V7 — Paridade domínio/CLI/MCP e regressões #157/#158 | Mesmos resultados; declared, precisão e rotação legítima preservados. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

Mapear os R-IDs aos casos acima e aos produtores reais durante a implementação.
Não declarar satisfeitos os requisitos funcionais pelo lint desta spec.

## 7. Final Report

### Delivered scope

Spec em draft, escrita no lote 1. Implementação, integração, revisão e closeout
funcionais pendentes; qualificação da escrita está no relatório do lote.

### Residual risks

Um payload assinado prova a declaração do issuer, não a verdade de um relógio ou a existência de uma autorização histórica não registrada. A definição de histórico versus nova operação precisa de decisão de compatibilidade/contrato antes do código. Cobertura de inventário ilegível é B06; o reuso deve expor suas próprias falhas sem assumir cobertura global completa.

### Follow-ups

Nenhum item residual encerrado nesta escrita. As dependências e unidades restantes
permanecem na matriz do programa, sem disposition automática.
