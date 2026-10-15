# B01 — qualificação de respostas persistidas

**Atualização de closeout — 2026-10-15:** spec encerrada pelo fluxo nativo (`done`, `completed_at: 2026-10-15`), após aceite explícito do maintainer. A ação está answered/satisfied, revision 2, com assurance declared. Admissão: bundle `rvb-cfe993ee0dbf209f`, attestation `rva-370e6fda5e127a19`; validation estrita 132/132 no sujeito publicado `9d89a2a0`, verify/review-check aprovados, closeout-check de admissão terminal=true e sem blockers. Os registros anteriores abaixo são históricos. O pacote será reindexado e reverificado após a publicação do lifecycle; os IDs correntes são os de `.pose/results/b01-action-review-verification.json`. B01 continua parcial/delivery_complete false, B04 terminal e B02 aguardando a spec irmã.


Estado: in-progress; implementação e qualificação revisadas nesta execução independente.
Closeout aguarda acceptance do maintainer; consumir os resultados nativos finais abaixo.
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

## Revisão e limites da execução implementadora (histórico)

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


## Revisão independente — 2026-10-15

Revisor: `agent:independent-codex-pr179`, o assistente OpenAI Codex desta sessão,
distinta da execução implementadora `agent:codex-b01`. O identificador descreve
esta execução real; não representa outra pessoa, outro modelo contratado ou um
runner externo. A revisão foi feita por leitura própria dos requisitos, do diff
atribuído de #157, dos testes de domínio e dos transportes adicionados em #179,
e das evidências seladas. As propostas do implementador foram tratadas como
hipóteses a conferir, não como conclusões automaticamente adotadas.

A independência é declarada: não há claim assinado de separação, nem demonstração
de independência cognitiva (não observável). Nenhuma confirmação ou adoção humana
é atribuída ao maintainer. Não foi instalado adapter nem alterada qualquer policy.
O bundle inicial `rvb-f9020ebb6c7dff19` estava fresco no head `6ef5db75`;
a publicação das correções documentais atribuídas exige nova coleta e novo bundle.
Os IDs definitivos, digests, atribuição e resultado de verify são registrados em
`.pose/results/b01-action-review-verification.json`, não inferidos deste texto.

### Critérios de julgamento e conclusões próprias

| Critério | Conclusão e referência verificável |
| --- | --- |
| assumption-integrity | Passa. R3/D1 limitam a alegação ao instante registrado e à precisão disponível. `verifyActionClaimWithin` compara emissão/expiry com o intervalo de aceitação e consulta pins/grants atuais. O teste `TestActionReadKeepsHistoricalExpiryAndSecondPrecision` prova o caso subsegundo, sem provar autorização histórica imutável. A hipótese de integridade do instante registrado continua explícita. |
| compatibility | Passa. R4 preserva declared sem prova e deriva verified somente após verificação. `TestActionReadDerivesProofAndPreservesDeclaredAndRotatedSSH` cobre legado, remoção da chave registrada e alteração do label para declared. Os testes CLI/MCP exercitam primeiro uma resposta declared válida; o erro posterior é o endurecimento intencional de R1/R5, sem migração de schema. |
| design-causality | Passa. R1/R5 exigem proteger consumidores além do texto. `LoadActionRequest` chama `verifyActionAnswers` antes de `deriveActionView`; `appendActionEvent` também recusa histórico inválido antes de idempotência/append. A verificação apenas no renderer deixaria listas, JSON, projeções e gates expostos. D1 reutiliza a verificação com relógio explícito para atender R3 sem expirar história válida. |
| documentation | Passa após correção editorial. Spec, decision log e changelog descrevem limites reais; as atualizações desta revisão distinguem parecer técnico de acceptance e closeout. Foram corrigidas narrativas atuais dos roadmaps que ainda diziam B01 não iniciado/aguardando merge de B04; registros históricos e baseline permanecem preservados. |
| operability | Passa. R5 falha com `action-claim-not-verified`; `TestActionReadRefusesProoflessVerifiedAndDegradesConsumers` verifica listagem/gate e cobertura unavailable. Os testes de transporte verificam JSON/text/list/presentation e Attention/obligations incomplete, sem reescrever o journal. Falhar a lista inteira preserva visibilidade da incompletude, em vez de ocultar a ação adulterada. |
| scope | Passa. O diff original implementa R1–R5 nos três arquivos de domínio; #179 acrescenta dois testes de transporte e qualificação. O intervalo Git contém outras specs, então a revisão usa os segmentos atribuídos do bundle, não atribui todo base..head a este trabalho. Nenhum runtime, R/D, start, baseline, policy ou B04 é alterado nesta revisão. |
| security | Passa. R2 é conferido em `VerifyRecordedActionSignature` (statement, chave, namespace, fingerprint e flags) e `verifyActionClaimWithin` (assinatura Ed25519, issuer, audience/project, request, principal, answer/reason). A presença de provas mistas exige validar ambas. Os subtestes de adulteração e o caso proofless negam o caminho barato de gravar apenas verified no journal. Fixtures não contêm credenciais reais. Assinatura SSH continua prova dos bytes, não recibo imutável de autorização histórica. |
| solution-proportionality | Passa. Verificar no reader comum é a menor correção que cobre R1/R5; só verificar no renderer é insuficiente. R3 reutiliza o verificador e R4 conserva o formato legado. Não há cache, daemon, segundo motor, novo runner, policy ou migração especulativa. A necessidade futura de provar autorização histórica ou cargas de leitura muito maiores exigiria novo escopo e medições, não uma ampliação silenciosa desta correção. |

Parecer técnico: aprovado para o escopo desta spec; nenhum finding funcional
aberto. Correção documental de baixa severidade aplicada nos roadmaps para
remover estado operacional defasado. Não constitui aceite do maintainer.
Revalidar o novo sujeito atribuído antes de selar e registrar a attestation;
resultados mecânicos atuais estão nos arquivos nativos `b01-action-*`.

Rules applied: security (prova e autoridade), backend-go (reader/erros/testes),
documentation-style e delivery-evidence (estado comprovável), knowledge-governance
(decision log vigente), `_base-recurrence`. Frontend e infraestrutura não se aplicam.
Assessment design/discover/integrate/tech-debt e suggest-review foram executados;
os diagnósticos preexistentes de consumidor não observado não são regressões desta
spec. Recurrence não encontrou chaves sinalizadas; manteve um flapping histórico.
Não houve aceitação de risco novo nem follow-up que exija novo handoff.
Review-check e closeout-check são ferramentas de conclusão, executadas após a
attestation; a restrição de acceptance não deve ser declarada como sucesso.

### Acceptance exata pendente

Registro histórico da entrega anterior: essa pendência foi satisfeita pelo aceite explícito documentado ao final deste relatório.

`act-3ee9344e0652cc60` permanece open, revision 1, destinada ao papel maintainer.
A policy `.pose/policy/actions.json` atribui esse papel apenas a `human:oseias`.
A pergunta solicita examinar a qualificação e a revisão independente ou adotar
expressamente as conclusões; delegar a operação não manifesta esse aceite.
O maintainer deve decidir sobre as oito conclusões acima, especialmente R5:
CLI/MCP/gates recusam prova inválida e a projeção expõe cobertura incompleta;
declared legado e aceitação histórica válida continuam preservados, dentro dos
limites de confiança documentados. B01 continua parcial e B02 não é liberado.

Somente após manifestação explícita do maintainer, conferir novamente `pose action
show act-3ee9344e0652cc60 --json` e, se digest/revision ainda forem estes, registrar:

```bash
pose action resolve act-3ee9344e0652cc60 --actor human:oseias --answer accept \
  --request-digest sha256:a2525c04b212fe11fee18ecdb04f32d47875a62dedbbc6d2ceaed5c9f941e716 \
  --expected-revision 1 --idempotency-key pr179-maintainer-acceptance --apply
```

Esse comando não foi executado. Em caso de rejeição, usar `--answer reject`,
registrar os motivos e corrigir antes de repetir a revisão. Após a decisão,
consultar `pose close spec:pose-action-answers-reverify-on-read --plan --json`;
usar `--apply`/`--resume --reviewer agent:independent-codex-pr179` conforme o plano,
revalidando e obtendo nova attestation se os inputs do bundle mudarem. Confirmar
`review verify`, `review-check` e `closeout-check` antes da transição. Esta sessão
prepara o closeout até esse gate e mantém a spec in-progress.


## Aceite explícito do maintainer — 2026-10-15

Após a entrega da revisão independente e das evidências do head cf4c40ca, o
maintainer declarou nesta conversa: “eu aceito. prossiga”. O fluxo nativo
registrou `act-3ee9344e0652cc60` como answered/satisfied, revision 2, answer accept,
actor human:oseias, com assurance declared. A operação foi aplicada por esta
sessão Codex; não há assinatura ou confirmação criptográfica. O journal preserva
a pergunta, o digest, a revisão original e o evento aberto; acrescenta somente
a resposta autorizada, vinculada às conclusões de rvb-aa88dec0a71ddbdc /
rva-df0fb2c18df35afc. A attestation técnica anterior não foi reescrita.

Essa alteração supersede o bundle anterior. A retomada renova evidências e
reafirma os oito julgamentos antes do gate final. Os IDs e o resultado terminal
definitivos são os dos snapshots nativos `b01-action-review-verification.json`,
`b01-action-closeout-check.json` e `b01-action-closeout-plan.json`. Esta nota
supersede a pendência de acceptance acima, que permanece como histórico.
B01 continua parcial/delivery_complete false, a spec irmã exige outro PR,
B04 permanece terminal e B02 não é liberado.


## Closeout nativo após o aceite — 2026-10-15

O fluxo registrou a resposta autorizada sem editar o evento aberto ou a policy.
A renovação estrita passou em 132/132, sem falhas/erros/skips. Artifact-check
estrito passou; surface-check teve target=1, results=132, findings=0. O preview
continuou com sete critérios mecânicos e oito julgamentos pendentes; estes
foram explicitamente reafirmados nesta sessão, não aprovados pelo preview.
A revisão renovada foi selada em rvb-cfe993ee0dbf209f e registrada em
rva-370e6fda5e127a19, preservando a identidade e a separação declaradas.

O gate de contexto inicialmente recusou três assessments gerados não commitados
(apenas datas e referências de baseline). Eles foram publicados como resultados
gerados, sem trailer artificial de implementação, em 2aca0f4f. O plano voltou a
ready-to-close e close --resume aplicou a transição. O resultado nativo de
admissão está em b01-action-closeout-admission.json: terminal=true, lifecycle_done=true,
blockers=[]. O log de closeout registra step.transition=done. Nenhum status,
digest ou aceite foi alterado manualmente.

A publicação do lifecycle e de seus resultados modifica artefatos declarados e
recebe o trailer exigido pelo próprio fluxo de closeout. Não há mudança em
R1–R5, D1 ou nos seis arquivos Go de implementação/testes: seus digests seguem
iguais aos da revisão anterior. A coleta funcional executada em 9d89a2a0 é
preservada como evidência do sujeito funcional; a metadata publicada é
reindexada e novamente revisada/verificada antes da entrega final. Resultados
gerados posteriores são publicados separadamente, sem atribuição como nova
implementação. Os bundles e attestations anteriores permanecem preservados.

B01 continua parcial/delivery_complete false, com action-answer done e disclosure
in-progress para outro PR. B04 não é reaberto e B02 não é liberado. Nenhum merge,
runner de review, adoção de policy ou próxima spec foi iniciado.
