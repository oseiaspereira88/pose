# B01 — qualificação de respostas persistidas

Estado: in-progress; implementação mergeada e regressões de superfícies
qualificadas; revisão independente e closeout permanecem pendentes.
Data da execução: 2026-10-15 UTC (retomada solicitada em 2026-10-14 no Brasil).
Base: a5016ce5eb0991db4e07ff096923a01b3a76741c, merge de #178.
Spec deste PR: pose-action-answers-reverify-on-read, R1–R5/D1 preservados.

## Reconciliação e escopo

A matriz distingue 44 unidades, 36 propostas obrigatórias materializadas e
B43 condicional. #159–#175 entregaram planejamento; seus closeouts
administrativos não representam implementação funcional. B04 tem spec done,
review aprovada, aceite e closeout nativo publicados por #178. Seu merge
remove a espera de publicação, sem criar prova retrospectiva de autorização.

B01 depende de B04 e reúne duas specs reutilizadas. #157 implementou a leitura
de respostas; #158 implementou disclosure de review. Ambas estão in-progress.
Este PR qualifica apenas a primeira, mantendo uma spec por PR. B01 não será
delivery_complete enquanto a segunda não tiver qualificação/closeout. B02
continua dependente de B01; não foi iniciado. Nenhuma policy ou release é adotada.

Context resolveu proj.pose-dist. Assessment discover --if-stale --component
pose-mcp reutilizou a avaliação por conteúdo/motor/matriz inalterados antes dos
testes novos. Start em preview informou que a spec já está in-progress; o start
original e sua baseline sha256:e36228291ce22452a542c23105cd3aa8ccffa1f266e4ace07d805de6c4ddc608
foram preservados. A dependência de execução B04 está satisfeita; não se edita a
baseline histórica para acrescentá-la retrospectivamente.

## Comportamento e cobertura

Os testes de domínio existentes recusam label verified sem proof e adulteração
SSH/issuer de answer, actor, digest, reason e metadata; verificam tempo fornecido,
expiry histórico, precisão de segundo, declared legado e SSH após rotação.
Os testes novos exercitam Main da CLI e o transporte HTTP JSON-RPC do MCP:
uma resposta declared válida é legível; adulterar apenas seu label para verified
causa action-claim-not-verified em show JSON/text, list, request por ID, list e
presentação. Attention e pose_obligations expõem cobertura unavailable/complete
false. A verificação continua centralizada no reader; nenhum runtime foi alterado.

O percurso mínimo para satisfazer o gate só na forma seria escrever verified no
journal sem assinatura. O corpus TestActionReadRefusesProoflessVerifiedAndDegradesConsumers
e os dois testes de superfícies recusam esse percurso. A assinatura autentica
bytes; não comprova julgamento semântico nem autorização histórica imutável.

## Validação

Go 1.26.9 alinhado ao CI. A bateria direcionada de domínio, CLI e MCP passou:
TestActionRead*, TestActionClaimRefuses*, TestActionCLIRejectsPersisted* e
TestToolsCall_ActionRequestsRejects*. A primeira execução do novo teste MCP
identificou Type vazio na fixture; corrigido para answered e reexecutado com pass.
Não houve alteração do runtime para satisfazer o teste.

Targeted -race also passed: pose 1.165s, CLI 1.076s, MCP 1.050s; no skips.
Assess integrate retained pre-existing unobserved-consumer diagnostics; tech-debt
reported zero markers and zero uncovered debt. Recurrence-check found no failure.

Acceptance request act-3ee9344e0652cc60 restricts closeout only; no acceptance
or human adoption has been recorded in this execution.

Evidência global nativa atual será registrada no results_path antes de selar o
bundle. Não substituir essa execução pelos resultados anteriores de B04.

## Rules applied during review

- Change type: qualificação de bugfix, testes de integração e documentação.
- Workflows: .pose/workflows/bugfix.md e .pose/workflows/review.md.
- security: prova adulterada, fronteira de autoridade e fixtures sintéticas.
- documentation-style: separar resultado funcional de revisão/closeout.
- knowledge-governance: consumir knowledge:action-answer-proof-and-historical-time.
- backend-go: regras do plano efetivo para o componente Go.
- Frontend/infra: nenhum contrato ou implementação desses domínios foi alterado.

## Revisão e limites

O plano efetivo exige different-actor pelo overlay high-criticality-review@1.
Esta execução prepara evidências e conclusões, mas não representa esse outro
ator nem pode registrar adoção humana ainda não dada. O bundle deve ser selado
com evidência atual; a preparação mecânica não satisfaz judgment.
Manter spec in-progress e delivery_complete false até review verify e closeout.

Próximo recorte após merge: qualificar pose-review-disclosure-requires-proof,
também em B01. B02 só é elegível depois do fechamento das duas specs.
