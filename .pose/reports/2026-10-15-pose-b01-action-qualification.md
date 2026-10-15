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
TestActionRead*, TestActionClaimRefuses*, TestActionReadCLI* e
TestActionReadMCP*. A primeira execução do novo teste MCP
identificou Type vazio na fixture; corrigido para answered e reexecutado com pass.
Não houve alteração do runtime para satisfazer o teste.

Targeted -race also passed: pose 1.165s, CLI 1.076s, MCP 1.050s; no skips.
Assess integrate retained pre-existing unobserved-consumer diagnostics; tech-debt
reported zero markers and zero uncovered debt. Recurrence-check flagged zero keys; one historical flapping key remains diagnostic.

Acceptance request act-3ee9344e0652cc60 restricts closeout only; no acceptance
or human adoption has been recorded in this execution.

Validação global nativa strict passou 132/132 em 265.6s no sujeito remoto
4fe2e2a6e814273cc51ebf0fdb02f97552c1d5bb, sem falhas/skips. A suíte Go
completa também passou (CLI 60.942s, domínio 36.109s). Check strict passou
com os 30 warnings históricos; lint da spec passou 5/5 traces, zero follow-ups
abertos e nenhuma alteração de R/D/baseline. Closeout-check de B04 confirmou
lifecycle_done/terminal, review válida e zero blockers na retomada.

Os nomes dos testes novos agora incluem ActionRead, selecionando-os no check
historical-trust-reuse-integration já registrado, sem acrescentar outra policy
ou duplicar a matriz. O results_path .pose/results/delivery-validation.json é
a fonte da qualificação atual; não substituir por evidência histórica de B04.
Readiness MCP nativa informou ready=true, waiting_on vazio; execution/review
clear, closeout restricted por action-open/review-not-approved.

O índice de entrega foi regenerado nativamente e seu snapshot é publicado
com os resultados e o bundle: o leitor da revisão o utiliza para reconstruir
o sujeito atribuído. A conferência após a primeira publicação detectou o
índice anterior no checkout remoto; regenerar/publicar o índice e conferir
novamente a frescura do bundle antes da revisão. Não promover outras specs.

## Rules applied during review

- Change type: qualificação de bugfix, testes de integração e documentação.
- Workflows: .pose/workflows/bugfix.md e .pose/workflows/review.md.
- security: prova adulterada, fronteira de autoridade e fixtures sintéticas.
- documentation-style: separar resultado funcional de revisão/closeout.
- knowledge-governance: consumir knowledge:action-answer-proof-and-historical-time.
- backend-go: regras do plano efetivo para o componente Go.
- Frontend/infra: nenhum contrato ou implementação desses domínios foi alterado.

## Conclusões propostas para julgamento

Estas conclusões foram preparadas por agent:codex-b01. Não são uma attestation
independente nem adoção pelo maintainer.

- Scope: o delta acrescenta provas dos consumidores aos requisitos existentes;
  nenhuma alteração de runtime, R/D ou baseline foi necessária.
- Assumption-integrity: horário de aceitação e pin/grant atuais são fronteiras
  explícitas; assinatura não comprova autorização histórica imutável.
- Design-causality: reverificar no reader protege todos os consumidores;
  verificar só no renderer permite que JSON/MCP/gates consumam o label forjado.
- Solution-proportionality: reutilizar o reader e o check ActionRead existente
  é suficiente; não adicionar outro motor, cache, policy ou migração.
- Security: o caso adversarial troca apenas declared por verified no journal;
  CLI/MCP recusam e a projeção degrada cobertura. Fixtures são sintéticas.
- Compatibility: declared permanece legível; SSH após rotação e expiry histórico
  preservam os contratos testados. A identidade não implica validade semântica.
- Operability: action-claim-not-verified e unavailable/complete false tornam a
  falha explícita em consulta, lista e Attention; nenhum record é reescrito.
- Documentation: matriz/roadmaps distinguem código mergeado, qualificação,
  revisão e fechamento; B01/B02 não são promovidos por este PR.

## Revisão e limites

O plano efetivo exige different-actor pelo overlay high-criticality-review@1.
Esta execução prepara evidências e conclusões, mas não representa esse outro
ator nem pode registrar adoção humana ainda não dada. O bundle deve ser selado
com evidência atual; a preparação mecânica não satisfaz judgment.
Manter spec in-progress e delivery_complete false até review verify e closeout.

Próximo recorte após merge: qualificar pose-review-disclosure-requires-proof,
também em B01. B02 só é elegível depois do fechamento das duas specs.

## Conferência do pacote publicado

O pacote inicial 822e663 omitiu o índice que reconstrói o sujeito da revisão.
A regeneração posterior detectou também que a publicação mudou a proveniência
atribuída pela matriz. O bundle rvb-45c65bb585cb4f18 conserva sua execução ce077834;
a tentativa rvb-4a69fb210314a116 não possui as classes funcionais exigidas e não
pode aprovar nem fechar a spec. Ambos são preservados como histórico.

ScopedDeliveryProvenanceDigest vincula claims e changesets. O commit 822e663
atribuía outputs derivados à spec e assim mudava o digest da própria coleta.
Registrar esta correção documental com POSE-Spec, coletar no último sujeito
atribuído e publicar somente outputs derivados em commit operacional sem esse
trailer: eles não implementam/modificam artefatos declarados desta spec. Não
transportar labels nem editar digests para fazer o gate passar. Gerar índice,
bundle e preparação mecânica juntos e conferir o checkout publicado. Nenhum código, R/D, baseline, policy
ou requisito foi alterado por essa correção de proveniência. O snapshot nativo
final e seus blockers estão nos arquivos b01-action-*; julgamento e aceite
continuam pendentes.
