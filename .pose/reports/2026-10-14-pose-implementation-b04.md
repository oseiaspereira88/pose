# Implementação B04 — tempo de confiança e proveniência de reuso

Base reconciliada: `6c3b6e1fe3481a998110226550e0e622d59ce79c` (merge #175).
Escopo: uma spec, `pose-historical-trust-and-review-reuse`, R1–R8 preservados.

## Reconciliação e decisão de retomada

O plano contém 44 unidades/96 candidatos. As 36 propostas obrigatórias estão
materializadas; B43 permanece condicionado e não é implementação entregue.
#159–#175 materializaram planejamento e preservam closeouts administrativos
pendentes. #157/#158 entregaram correções parciais de B01, cujas duas specs ainda
estão in-progress e devem ser qualificadas depois de B04. Nenhuma dessas
situações foi promovida a entrega completa por merge ou presença de código.

B04 é raiz da primeira camada e da cadeia preferencial B04 → B01 → B02. As
fundações `pose-material-equivalence-reuse` e `pose-native-attestation-issuer`
estão done na main. Seus requisitos/baselines não foram alterados. Os closeouts
administrativos não são dependências nativas funcionais desta spec; conservam
seu inventário próprio, sem bloquear artificialmente raízes elegíveis. B43 não
foi materializado ou executado nesta interação.

Context resolveu a autoridade local proj.pose-dist, readiness passou com zero
falhas e assessment --if-stale do pose-mcp foi atualizado antes do código. Start
nativo registrou a baseline `sha256:13047cc2e47c03bb826433e3157d262f1e7f97570e5ea3b3fe0f17146d292215`.
As alterações posteriores refinam paths/plano/evidências, preservando R1–R8 e D1/D2.

## Comportamento implementado e limites

O domínio separa claim histórico autenticado, contexto de nova operação e pins
atuais. Uma data assinada é issuer-asserted; historical_authorization continua
not-established. Expiry posterior não destrói um claim válido no contexto
histórico assinado, mas nova aprovação expirada é recusada antes da escrita.
Issued futuro, expiry reverso e attested_at futuro são recusados. Igualdade é
válida; sem expiry não ganha TTL. Precisão de segundo descreve um intervalo
limitado e não configura clock skew. Pins/grants continuam fronteira atual.

Verified-source valida assinatura, bundle, bindings, critérios selados,
findings/decisão e fontes recursivas. Current-reaffirmation exige julgamento
atual passed com conclusão. O modo legado é divulgado como reafirmação,
sem prova retrospectiva. Traversal tem 16 níveis, 32 nós, 128 edges e 16 MiB;
falha/ciclo/budget é inconclusivo. Arquivos JSON têm leitura limitada antes da
alocação integral; caches pertencem à operação e ao digest/contexto de confiança.

CLI verify/review-check/closeout e MCP pose_review_bundle consomem o domínio.
Fixtures sintéticas cobrem legado, histórico assinado, expiry, adulteração,
revogação, fonte negativa, binding, reafirmação e uma cadeia real profunda.
O contrato é aditivo e classificado para minor; nenhuma policy/adoption, tag,
release, manifest ou migração foi aplicado. ADR temporal registra os limites.

## Validação

Resultados são registrados após execução; ver seção Validation/Requirement
trace da spec e `.pose/results/delivery-validation.json` para evidência estruturada.
Regressões ampliadas passaram: pose 55.336s, CLI 8.921s, scaffold 6.116s,
com source futuro (inclusive declared assinado), whitespace de reafirmação,
corpus ABM, fixtures de issuer, action answers e paridade dos manuais embutidos.
Race detector de temporal/proveniência passou (35.498s). A otimização do caso
sem fontes passou em nova bateria (pose 6.149s, CLI 1.127s).

CI #752 (38600639877) e Security #754 (38600639619) passaram no head
2ff049b2c1676018124e07272160c56cb216cc75. Isso inclui suíte Go inteira,
Shellcheck, instalação, import/quickstart, upgrade, demo, imagens, runtimes,
CodeQL, govulncheck, dependency review e secrets. A validação Security usa
--tolerant/SARIF e não comprova por si só validate --strict sem findings.

Check --strict passou com 30 warnings históricos: 27 fragments ausentes,
assessment antigo/baseline distante e instância 7.1.0 versus motor 7.4.1.
Lint-spec --strict passou com zero findings; artifact-check reconciliou 29 paths
no primeiro commit sem erros. Os paths adicionais são qualificados depois.

A suíte ampla local com Go 1.26.0 encontrou timeout de 10 minutos em testes de
CLI/federação; isso não foi computado como pass. O CI com Go 1.26.9 executou a
suíte completa com sucesso. O toolchain local foi alinhado a 1.26.9 para gerar
evidência nativa strict. GOFLAGS=-buildvcs=false desabilita somente stamping do
binário local; o CI conserva o toolchain/stamping do repositório.

## Revisão e closeout

O plano nativo exige different-actor, derivado do overlay high-criticality-review@1.
Esta execução implementadora não representa um ator independente. O bundle será
selado com evidências atuais; judgment/verify/aceitação e closeout ficam pendentes
até review independente real. Não registrar attestations com identidade fictícia,
não editar status done manualmente e não desligar o overlay. A implementação pode
ser revisada no PR; delivery_complete permanece false até gates e closeout.

Próxima unidade na cadeia preferida: B01, qualificar as duas specs reutilizadas,
somente após merge e closeout de B04. As demais raízes da camada 1 continuam
tecnicamente elegíveis, mas esta interação não inicia outro lote.

## Handoff de qualificação nativa

Head documental e48b2f9bc94b84158021ccfd00233efc0b76ba41: CI #754
(38604918357) e Security #756 (38604918342) passaram integralmente. O head de
código 4708933 também passou ambos os pipelines; a mudança posterior é somente
narrativa/matriz/spec. O último head desta atualização deve ter seus próprios
pipelines conferidos antes da revisão.

Validate --strict --module pose-mcp executou 129 checks: 128 pass e um fail no
install-and-upgrade-journeys-integration, exclusivamente porque tar tentou chown
para uid/gid 1001, indisponível neste sandbox. O percurso passou no CI. Repetição
com TAR_OPTIONS=--no-same-owner preservou os checks, mas o executor desconectou
e perdeu os processos/checkpoints antes da conclusão. Não computar esse rerun
como pass e não selar o results_path histórico como prova atual.

A coleta nativa deve ser retomada no fluxo normal: validate successful current
results_path, assess --if-stale --update-state, index, bundle --seal, preparação
mecânica, julgamento different-actor, review verify, resposta de aceitação e close.
O arquivo de results e o índice derivados não são source-tree entregues nesta
interação; o PR declara somente seus paths efetivamente atribuídos. R1–R8 e
a baseline permanecem intactos. O pedido act-fa20ae0029b6ad14 restringe closeout.
B01 não é iniciado nem considerado elegível na cadeia preferida antes de
qualificação/closeout de B04 e merge. Os closeouts documentais permanecem separados.

## Retomada após o merge #176

Base atual: d55b8d41fd283ff1bba3e794fae1cd8c6e7707f7. O head final do PR #176,
77142f28e6045dfeeb204e07c86d8dab7c3a0fbe, passou CI #755 (38609619410) e
Security #757 (38609619438). O merge entregou implementação; não encerrou a spec.
Nenhum código, requisito, baseline, policy ou autorização foi alterado nesta
retomada. O lote é a qualificação de B04, ainda uma única spec.

`validate --strict --module pose-mcp` passou 129/129 checks em 365.9s na main
mergeada. Go 1.26.9; GOFLAGS=-buildvcs=false somente no binário local;
TAR_OPTIONS=--no-same-owner evita chown na extração, mantendo todos os checks.
O percurso real de instalação/upgrade passou em 31.5s. A evidência estruturada
é regenerada no results_path canônico antes de index e seal do pacote final.
Assessment discover --if-stale, integrate, tech-debt, suggest review e design
foram executados. Tech-debt retornou zero marcadores; a detecção é uma observação
limitada do scanner, não uma garantia de ausência de todo débito. Recurrence
identificou um flapping histórico de validate-native, sem cluster não resolvido.

## Preparação dos julgamentos de revisão

Preparador: agent:codex-b04-qualification. Este relatório contém a análise e a
proposta de conclusões; não é uma attestation different-actor. Nenhuma aprovação
foi registrada em nome de um bot, de um agente independente ou de human:oseias.
A revisão do GitHub em 77142f28 também não é uma resposta aos critérios nativos.
O pacote mecânico preparado fica em
`.pose/reports/2026-10-14-b04-review-preparation.json`, com o bundle exato e seus
critérios pendentes; o brief nativo acompanha o mesmo sujeito selado.

| Critério | Conclusão proposta e fundamento |
|---|---|
| scope | O delta implementa B04; R1–R8 e D1/D2 permanecem íntegros. B01 e B02 não foram iniciados. |
| security | `loadReuseAttestation` e `loadReuseBundle` aceitam somente prefixo e 16 dígitos hexadecimais antes de construir paths; as duas chamadas de `reserveReuseArtifact` usam diretórios constantes. O carregador recusa symlinks nos diretórios, valida digest e limita JSON a 4 MiB; assinatura/pins/bindings são verificados antes de aceitar verified-source. |
| compatibility | `temporal`, `reuse` e `mode` são aditivos. `mode` ausente conserva legacy-reaffirmation; não há TTL universal, migração de registros, adoção de policy nem emissão de release. A semântica mais estrita para nova operação é documentada como minor no ADR. |
| operability | Diagnósticos distinguem fonte indisponível, proof/binding inválidos, revogação, tempo incompatível e budget inconclusivo. `historical_authorization: not-established` impede apresentar data assinada como recibo observado. |
| documentation | POSE.md, pt-BR e os dois manuais embutidos descrevem tempo, modos, budgets e limites de confiança iguais aos do domínio. As fixtures de paridade passaram. |
| assumption-integrity | Uma assinatura autentica a declaração do issuer; não comprova relógio real, autorização histórica observada ou independência cognitiva. Pins/grants atuais continuam a fronteira explícita. Os testes separam essas leituras. |
| design-causality | O contexto temporal atende R1–R3; a avaliação compartilhada atende R4/R5/R7; budgets e memoização por operação atendem R6. O ADR observado SD-4abee72b é a decisão estrutural para D1/D2 e R2/R3/R4/R5/R6. |
| solution-proportionality | Um helper temporal e uma travessia local limitam o delta. Não criar serviço persistente, cache global ou receipt retroativo é suficiente para este escopo. A escolha deve ser revista se um contrato futuro exigir autorização histórica observada ou budgets configuráveis. |

Os critérios mecânicos devem citar somente IDs de evidência que o bundle final
selou, nunca o texto deste relatório como substituto de integração. A proposta
acima precisa do julgamento do outro ator exigido por high-criticality-review@1.

## Findings do PR #176 e disposições propostas

- A revisão GitHub 5522348432 apontou falta de POSE-Spec no commit c14dd1b8,
  que não pertence ao histórico mergeado. Os cinco commits efetivos 9646f0a9,
  2ff049b2, 4708933f, e48b2f9b e 77142f28 têm o trailer literal correto.
  `artifact-check --spec pose-historical-trust-and-review-reuse --strict`
  confirmou o change set cs-ab3e90650b05 sem erros. Não reescrever histórico
  para corrigir uma leitura que não descreve os commits efetivos.
- CodeQL alert 129 sinalizou path controlado por input em
  review_reuse_provenance.go:49. Na implementação mergeada, ambos os callers
  validam IDs hexadecimais e usam diretórios constantes antes da chamada;
  nenhum separador de path passa por esses callers. A análise propõe
  não considerar esse fluxo explorável como traversal; o alerta não foi
  automaticamente dispensado, apagado ou tratado como prova de ausência
  universal de vulnerabilidades. O outro ator deve confirmar a disposição.

## Gates restantes e próxima unidade

Independent judgment, review verify/review-check e aceite da ação
act-fa20ae0029b6ad14 continuam necessários antes de close. A policy actions.json
reserva o papel maintainer a human:oseias; não alterar essa lista nem resolver
o pedido por um agente. O merge #176 não responde ao pedido de aceitar somente
após verify aprovado. Delivery_complete permanece false.

Depois da aprovação nativa, resposta válida à ação e closeout de B04, B01 é o
próximo lote da cadeia: qualificar as duas specs existentes de #157/#158. Aguardar
o merge deste pacote antes de iniciar esse lote. As outras raízes da primeira
camada permanecem elegíveis pelo DAG, sem dependência artificial em B01.
