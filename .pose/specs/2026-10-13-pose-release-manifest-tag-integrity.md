---
slug: pose-release-manifest-tag-integrity
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on: pose-release-lifecycle-closure
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:release-manifest-tag-integrity
---

# Spec: Verificar manifest completo e snapshot no tree da tag

## 1. Intent

### Goal

Validar schema e projeções semânticas do manifest e demonstrar que a tag aponta para o snapshot de release correto, preservando história legítima.

### Business value

B38 / F06-04 e F06-06. P04 altera schema/specs/categories/breaking sem recusa; P06 usa tag sem manifest no tree e worktree com snapshot. Reutilizar lifecycle/arquivo de release existentes. Consultar o [plano autoritativo](../results/pose-implementation-plan.json)
e a [análise fonte](../reports/2026-10-12-pose-front-06-release-sufficiency.md).

### Constraints

Esta interação escreve a spec; não implementa o comportamento. Preservar as
fundações done, contratos selados, evidências e autorizações já existentes.
Não adotar policy/capability automaticamente. Classificar compatibilidade antes
de iniciar: recusa mais estrita ou mudança de semântica pública não é patch por
definição. Reconciliar decisões estruturais com os ADRs existentes; registrar
somente decisões adicionais necessárias, antes da implementação correspondente.

## 2. Requirements

- R1: Manifest deve validar schema suportado, shapes/tipos, versão, paths e campos desconhecidos conforme contrato versionado, com diagnóstico localizado; objeto JSON parseável não deve ser tratado como manifest válido/complete.
- R2: Recompor projeções redundantes de specs/categories/breaking e demais campos semânticos derivados a partir dos fragments/inputs autoritativos do cut; divergência, duplicação inválida ou alteração relevante deve ser detectada, sem normalização silenciosa de adulteração.
- R3: Verificar notes digest, release input digest, policy/version evidence e snapshot/archive pertinentes conforme schema e contrato aplicáveis; rebuild/check deve cobrir cada mudança semanticamente relevante, distinguindo fonte ausente/ilegível de ausência legítima.
- R4: Resolver tag e seu commit/tree (incluindo peeling de tag anotada) e ler manifest/archive/notes/version_source no snapshot da tag, sem usar worktree atual como substituto. Tag sem snapshot devido deve ser recusada com diagnóstico, mesmo quando o worktree contém um manifest válido.
- R5: Binding deve distinguir target head/preparação de commit que contém o snapshot e tag: não exigir igualdade impossível antes da publicação do próprio manifest. Validar vínculo semântico pertinente e integridade de conteúdo selado, sem aceitar coincidência de nome/version como prova.
- R6: Commits posteriores de provider evidence ou início do próximo ciclo não devem staleiar cut/tag historicamente corretos; dados históricos devem ser verificados sob seu contrato, com limites legacy/unsupported explícitos, sem aplicar critérios futuros retrospectivamente.
- R7: Checker/status/CLI/MCP devem compartilhar verificação e divulgar preparado/tagged/published/verified e coverage separadamente. Checker é read-only: não mover/recriar tag, reescrever manifest/notes/archive ou backfill de prova para corrigir o resultado.
- R8: Qualificar P04/P06, schema/shapes/version/path/unknown fields, cada projeção/digest adulterado, tag leve/anotada e snapshot ausente, com controles de cut correto seguido por evidence/novo ciclo; preservar composição/federação existentes.

## 3. Technical Plan

### Artifacts

Os paths abaixo são declarações prospectivas para a implementação. Neste lote
somente esta spec é criada, sob a atribuição documental de
`pose-consolidation-batch-6-authoring`. Reconciliar os paths exatos antes de start;
não tratar a declaração de um teste futuro como teste existente ou executado.

- modified: .pose/specs/2026-10-13-pose-release-manifest-tag-integrity.md
- created: .pose/starts/pose-release-manifest-tag-integrity.json
- created: .pose/changelogs/unreleased/pose-release-manifest-tag-integrity.md
- created: pose-mcp/internal/pose/release_manifest_integrity.go
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- created: pose-mcp/internal/pose/release_manifest_tag_integrity_test.go
- created: pose-mcp/internal/pose/release_manifest_fixture_test.go
- created: pose-mcp/internal/cli/release_manifest_tag_integrity_test.go
- created: pose-mcp/internal/pose/release_review_regression_test.go
- created: pose-mcp/internal/cli/release_review_regression_test.go
- created: pose-mcp/internal/pose/release_review_r2_test.go
- modified: pose-mcp/internal/pose/release_archive.go
- modified: pose-mcp/internal/pose/review_release_archive_test.go
- modified: pose-mcp/internal/pose/release_version_source_test.go
- modified: pose-mcp/internal/pose/release_lifecycle_test.go
- created: .pose/reports/2026-10-18-b38r2-independent-review.md
- created: .pose/results/b38r2-independent-review.json
- created: .pose/results/b38r2-independent-validation.json
- created: .pose/results/b38r2-global-validation.json
- created: .pose/results/b38r2-mcp-enforce-validation.json
- modified: pose-mcp/internal/pose/obligation_sources.go
- modified: pose-mcp/internal/cli/publicclaims.go
- created: .pose/reports/2026-10-18-b38-independent-review.md
- created: .pose/results/b38-independent-review.json
- modified: pose-mcp/internal/pose/obligation_sources_test.go
- modified: pose-mcp/internal/pose/release_archive_test.go
- modified: pose-mcp/internal/cli/publicclaims_provenance_test.go
- created: .pose/results/b38-mutations.py
- created: .pose/results/b38-mutations.json
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:release-manifest-tag-integrity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/release_lifecycle.go

### Approach and execution gates

Definir schema/compatibilidade e recomposição a partir dos inputs selados, reutilizando lifecycle e loaders. Ler blobs do tree da tag por interfaces Git read-only com erros tipados. Reconciliar paths/helpers reais de check/tag antes de start; separar integridade de suficiência (B37), replay temporal (B39), seleção/exclusividade antes do freeze (B40) e recuperação/publicação (B41). Não atualizar tags/releases reais nesta qualificação.

Antes de executar, aceitar o escopo de implementação, avaliar readiness nativa,
executar assessment freshness no componente, completar o plano de artefatos,
registrar produtores de evidência na matriz e criar a baseline por start.
Não contornar dependência não terminal apenas porque o código parece presente.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6
- Decision: Recompor projeções pelo cut selado e verificar snapshot diretamente no tree da tag.
- Rationale: Campos redundantes e árvore corrente não são prova independente do conteúdo publicado.

## 6. Validation

### Strategy

Usar fixtures sintéticas e relógio controlado. Executar os casos abaixo no domínio
e nos consumidores indicados, sem simular aprovação pela presença de um campo.

| Caso planejado | Resultado esperado |
|---|---|
| V1 — P04 schema/shapes/version/paths/unknown fields | Recusa/unsupported localizado, sem falso manifest válido. |
| V2 — Specs/categories/breaking e digests adulterados | Recomposição detecta cada alteração relevante. |
| V3 — P06 tag sem snapshot e worktree com manifest | Recusa por tree da tag, sem fallback ao worktree. |
| V4 — Tag leve/anotada, archive/notes/version source ausentes ou alterados | Binding e conteúdo correto verificados ou diagnóstico específico. |
| V5 — Target head versus commit do snapshot | Sem igualdade impossível; vínculo pertinente comprovado. |
| V6 — Evidence posterior e próximo ciclo | Cut correto preservado historicamente, sem staleness artificial. |
| V7 — CLI/MCP/read-only e controles legacy/federados | Paridade e limites explícitos; nenhuma tag movida ou história reescrita. |

### Deterministic checks

Planejar suíte Go do módulo, integração CLI/MCP, checks estruturais e de segurança;
usar race detector para travessia/memoização ou concorrência quando houver mudança
nessas áreas. Verificar artifact-check e delivery/surface evidence na implementação.
Testes de execução ainda não foram criados ou executados para este delta.

### Requirement trace

- R1 [satisfied] check:release-manifest-tag-integrity-integration test:TestAManifestIsValidOnlyIfItsShapeAndProjectionsAgree test:TestF01ManifestIsStrictJSON test:TestF04VersionEvidenceMustBeTheReleaseAndItsSourceMustBeThere test:TestF10VersionPrecedenceIsSemanticAndPreviousIsOlder test:TestR2NestedFragmentObjectsAreStrict test:TestR2VersionEvidenceFollowsThePolicyOfTheCut test:TestR2PrepareDoesNotOverwriteAManifestItCannotRead
- R2 [satisfied] check:release-manifest-tag-integrity-integration test:TestAManifestIsValidOnlyIfItsShapeAndProjectionsAgree test:TestF02ProjectionsAreRecomposedFromTheFragmentBlobs test:TestR2ArchivalWitnessIsTheSpecOfTheBlob test:TestR2NotesAreTheRenderingOfTheFragments
- R3 [satisfied] check:release-manifest-tag-integrity-integration test:TestTheSnapshotIsVerifiedAgainstTheTaggedTreeNotTheWorktree test:TestAVersionSourceThatCannotBeRecomputedIsALimit test:TestF03PolicyDigestFollowsThePolicyOfTheTag test:TestF06AbsenceIsNotALoss test:TestR2PoliciesOfTheCutAreValidated test:TestR2GitFailuresAreNotAbsence test:TestR2PreparedEvidenceNamesTheSourceOfThePolicy
- R4 [satisfied] check:release-manifest-tag-integrity-integration test:TestATagWithoutTheSnapshotInItsTreeIsRefusedWhateverTheWorktreeHolds test:TestLightweightAndAnnotatedTagsResolveToTheirCommit test:TestReleaseCheckReadsTheTagsOwnTree test:TestF07ABranchWithTheNameOfATagIsNotATag test:TestF05RecordRefusesATagWhoseTreeIsNotIntact
- R5 [satisfied] check:release-manifest-tag-integrity-integration test:TestACorrectCutStaysIntactAfterEvidenceAndTheNextCycle test:TestF09TheWorkingTreeMustBeTheCutOfTheTag
- R6 [satisfied] check:release-manifest-tag-integrity-integration test:TestACorrectCutStaysIntactAfterEvidenceAndTheNextCycle test:TestAManifestIsValidOnlyIfItsShapeAndProjectionsAgree test:TestF08ALaterPolicyDoesNotInvalidateATaggedCut
- R7 [satisfied] check:release-manifest-tag-integrity-integration test:TestReleaseStatusReportsTheSnapshotOfTheTag test:TestReleaseCheckReadsTheTagsOwnTree test:TestF05StatusDoesNotTrustASnapshotThatIsNotIntact test:TestR2StatusSeesTheFilesOfTheCutThatWereLost
- R8 [satisfied] check:release-manifest-tag-integrity-integration report:b38-mutations.json — adulteração de cada projeção e digest, schema, versão, caminhos, campo desconhecido, tag leve e anotada, tag sem snapshot e cut correto seguido de evidência e novo ciclo; as mutações de produção morrem por asserção

A verificação é de leitura: o snapshot lê blobs do commit da tag e usa apenas um diretório temporário para reler a fonte de versão; não move tag, não reescreve manifest, notas nem archive.

## 7. Final Report

### Delivered scope

B38 entregue: o manifest passa por um contrato estrito (schema suportado, campos desconhecidos recusados, versão e caminhos limpos, projeções `specs`/`categories`/`breaking` e `release_input_digest` recompostas a partir dos fragments congelados); `release check` e `release status` verificam a tag pela sua própria árvore (tag leve ou anotada resolvida ao commit; manifest, notas, fragments congelados e fonte de versão lidos como blobs desse commit), recusam tag sem snapshot qualquer que seja o worktree, não tornam obsoleto um cut correto por commit, evidência ou ciclo posteriores, e reportam como limite a fonte de versão que não pode ser recomputada. Revisão independente e closeout pendentes.

### Residual risks

Schema estrito precisa de política deliberada para versões legacy e extensões futuras. Tag existir não prova snapshot correto, e manifest do worktree não pode completar o tree da tag. Sem inputs autoritativos suficientes, divulgar incomplete/unknown em vez de reconstruir prova retrospectiva.

### Follow-ups

Primeira revisão independente (changes-requested, dez bloqueantes) tratada: o manifest é lido como
um único documento JSON de nomes exatos, campos obrigatórios e sem chave repetida; os campos que
ele repete são recompostos dos blobs de fragment e da policy da árvore da tag; a evidência de
versão tem de ser a versão da release e sua fonte tem de estar na árvore; o status e os
consumidores só tratam como sólida (`Sound`) uma release cujo snapshot é íntegro, e uma release
cuja pasta sumiu do worktree continua no registro pela tag; ausência, falha de leitura e perda de
inventário são respostas distintas; a tag é só `refs/tags`; uma release com tag é julgada sob a
policy do seu corte e o worktree tem de ser o mesmo corte da tag; a precedência de versões é a do
SemVer 2.0 em todos os consumidores, e `previous_version` tem de ser mais antigo. Segunda revisão (oito bloqueantes) tratada: os objetos aninhados do manifest seguem o contrato da raiz; o testemunho de archival confere a spec que o blob nomeia; o status confere o worktree com o corte como o check; a evidência de versão segue a policy do corte (JSON lido com a chave, arquivo do motor só pelo caminho exato) e o candidato confere fonte e tipo; falhas do Git e tags sem snapshot são lacunas e não ausência; as policies do corte (release e changelog) são validadas pelo próprio contrato; as notas são a renderização dos fragments congelados; e `prepare` não sobrescreve manifest existente que não lê. Duas guardas não têm mutante próprio porque ele seria equivalente: o fim do documento (um objeto ou colchete depois do manifest já é recusado pela leitura de chaves repetidas e caracteres de controle) e a solidez por snapshot não íntegro (um snapshot não íntegro sempre deixa gaps na projeção, que a tornam não sólida por si). Item aberto,
de baixo risco:

- [open] Reconciliar o texto de authoring da spec com a implementação e a validação efetivamente executadas, mantendo separado o closeout pendente. (owner:@core crit:low review:2026-12-01)

As dependências e unidades restantes permanecem na matriz do programa, sem disposição automática.
