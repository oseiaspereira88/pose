# Frente 6 — escopo, compromissos e suficiência de releases

Data: 2026-10-12. Revisão fonte: `b983245b6e4aef12e4ffc429b0b1b71a4e8f638d`, após o merge do PR #151. O código de `pose-mcp` está idêntico ao tag v7.4.1; os relatórios recentes acrescentam contexto, não alteram o engine.

[Índice](2026-10-11-pose-analysis-index.md) · [Evidências estruturadas e fontes](../results/2026-10-12-pose-front-06-release-sufficiency.json) · [Probes reproduzíveis](../results/2026-10-12-pose-front-06-release-probes.py) · [Frente 5](2026-10-12-pose-front-05-residual-work.md).

## 1. Parecer

O POSE tem um lifecycle de release concreto: congela fragments e notes, registra manifest, distingue tag de publicação, exige vínculo com evidência de publicação para projetar verificação e preserva os artefatos históricos. O workflow deste repositório executa a CI do commit publicado, testes, compatibilidade e segurança; a verificação independente autentica assets antes de executá-los. Essas entregas devem ser preservadas. [E01, E02, E19, E21, E29–E33]

A lacuna central é a composição das garantias. A integridade de um snapshot de fragments não demonstra que o produto atende ao escopo prometido. O gate de roadmap avalia aceitação e critérios; o de release prepara fragments existentes. O manifest não liga explicitamente o cut a um escopo de compromisso, outcomes ou critérios aceitos. Os probes mostraram que um roadmap com spec draft e check ausente é recusado, enquanto a mesma spec pode compor um candidato aceito por `release prepare` e `release check --strict`. [E01–E07]

Há ainda lacunas internas de integridade: campos semânticos do manifest podem divergir dos fragments sem reprovação; assets de verificação podem diferir da publicação; o replay confia em digests persistidos sem recomputá-los. Critérios de delivery podem passar por existência da declaração enquanto `surface-check` reprova sua suficiência. A correção deve ligar os avaliadores existentes e tornar explícito o alcance de cada resultado, sem criar outro motor de gates.

Recomendar duas entregas: **fortalecer integridade e suficiência dos gates atuais**; depois **compor um cut explicitamente ligado aos compromissos selecionados e à evidência operacional**. A numeração das versões fica para a frente 10. Não bloquear releases pelo histórico inteiro do projeto, nem chamar dívida opcional de compromisso obrigatório.

## 2. Método e limites

Inspecionar fontes E01–E41: CLI, Store, roadmap/delivery graph, closeout, aceitação federada, ledger, políticas, ADRs, specs, workflow e testes. Executar 14 probes com o binário oficial 7.4.1 disponível, em repositórios Git descartáveis. Tags criadas são locais e sintéticas; evidências têm providers/repositories genéricos. Nenhuma publicação, assinatura real, workflow externo ou alteração de projeto consumidor foi executada.

A suíte Go não foi reexecutada: não há executável Go disponível. O código local não foi recompilado. Foram inspecionados os testes existentes, e os probes observam o binário cuja árvore fonte do engine coincide com a baseline. Isso sustenta os comportamentos descritos, sem validar integralmente os contratos de releases históricas.

Os fixtures usam review policy desabilitada para separar o comportamento dos avaliadores. Essa condição é explícita: não demonstram bypass de uma attestation obrigatória com assurance forte nem sucesso do pipeline real. P13 inclui declaration e arquivos commitados, elimina o mismatch de source revision observado na primeira preparação e ainda reproduz a divergência entre roadmap e surface. P12 registra negativos que funcionam. Resultados, código do harness e hashes de fontes estão versionados.

## 3. Contratos existentes e significado dos resultados

| Camada | Fonte/avaliação presente | Afirmação sustentada | Limite |
|---|---|---|---|
| Spec | Lifecycle, review e closeout | Spec encerrada conforme os contratos aplicáveis | Não certifica uma composição inteira nem adoção no consumidor |
| Delivery | Artifact claims, Git, target, entrypoint, profile e resultados atuais | Proveniência e evidência do target no escopo avaliado | Declaração de target sozinha não comprova resultado |
| Milestone | Membros, predecessores e aceitação federada do milestone | Aceitação daquele subconjunto | Não aplica critérios globais do roadmap por padrão |
| Roadmap | Aceitação de membros e `Cut criteria` | Gate do programa conforme critérios reconhecidos | Não seleciona automaticamente o escopo de uma release |
| Manifest | Fragments, notes, policy e version evidence | Snapshot preparado dos insumos congelados | Não registra contrato explícito de outcomes ou revisão da composição |
| Tag | Git ref/commit | Tag local existente | Não comprova conteúdo preparado no tag nem publicação |
| Publicação | Evento com evidência tipada | Publicação registrada localmente | Import não consulta provider nem autentica automaticamente o relatório |
| Verificação | Evento ligado ao digest de publicação | Vínculo declarado com a publicação | Ainda não compara adequadamente os assets nem verifica todos os inputs no replay |
| Workflow real | CI, testes, segurança, assinatura/proveniência e verifier | Execução das etapas e seus resultados retidos | Não substituir esses resultados por `check: valid` do snapshot |

Separar `valid prepared snapshot`, `ready for cut`, `published` e `verified`. O primeiro não deve funcionar como certificado dos demais. Preservar os nomes atuais por compatibilidade, mas acrescentar disclosure de escopo e condições avaliadas.

`roadmap-check` retorna `terminal` como resultado do gate; não muda o `status` do roadmap. Na baseline real, `agency-readiness` continua `draft` e retorna `terminal: true` porque os membros consultados foram aceitos e nenhum critério estruturado foi reconhecido. O fechamento efetivo via closeout tem outro contrato, incluindo lifecycle e review. Não inferir que o roadmap foi automaticamente encerrado. [E03, E05, E34]

A aceitação federada é uma capacidade relevante já existente: preserva autoridade local, exige provas do source e composição no coordenador, distingue implantação de adoção e não escreve no repositório remoto durante uma consulta. O gate de milestone já foi corrigido para não bloquear um milestone anterior pelos membros de um posterior. Manter essa separação e reutilizar os mesmos avaliadores no contrato de cut. [E06, E16, E23, E28]

## 4. Baseline operacional atual

`pose release status --json` retornou 75 releases locais e zero fragments pendentes: 55 projetadas `verified`, 12 `tagged`, quatro `failed`, três `published` e uma `prepared`. As seis mais recentes, v7.0.0 a v7.4.1, aparecem `verified`; v6.3.0 permanece `prepared`. Esses números descrevem o ledger local consultado, não uma nova certificação de disponibilidade ou integridade no provider.

`pose release check --version v7.4.1 --strict --json` retornou `valid: true`, zero gaps e tag commit `7ef3be2a44f0d0c2daceaf3f5c63e6840197023a`. A leitura de v7.4.1 mostrou manifest com uma spec/fragment e eventos tagged, published e verified. Os nomes e digests dos 36 assets em publicação e verificação coincidem; o evento tagged usa provider `git`, o published `github` e o verified `github-actions`. Portanto, corrigir validação de provider não significa exigir a mesma string em todas as fases. Provider de tag, publisher e verifier são papéis distintos; a origem da publicação deve corresponder à policy, e a origem da verificação deve seguir um contrato declarado. [E37–E40]

Entre os 18 roadmaps versionados, dois têm o heading canônico `## Cut criteria`, três têm `## Release cut criteria` e 13 não têm nenhum desses headings. Os três alternativos são `agency-readiness`, `community-launch` e `pose-v7-onboarding-and-consolidation`. O texto desses programas descreve intenção de cut em prosa, mas não está convertido no contrato estruturado que o parser executa. Não contar automaticamente prosa como gate, nem declarar todos esses programas aptos para release só porque as specs estão `done`. [E04, E20, E34–E36]

A baseline reconcilia achados anteriores: tag não equivale a publicação; manifest preparado não prova release externa; follow-ups consultivos não bloqueiam por si mesmos. A análise de 2026-10-03 usava uma baseline anterior e v6.3.0 preparada. Hoje há outras releases e evidências, então preservar esse relatório como histórico, sem repetir sua versão como estado atual. [E11, E41]

## 5. Fluxo real e distribuição das responsabilidades

`releaseInputs` exige policy adotada, fragments válidos, specs existentes, version source válida e target mais novo que releases/tags anteriores. Não exige que as specs estejam `done`, revisadas ou com todos os targets entregues. `plan` emite a seleção derivada da fila, recomendação SemVer e blockers de foreign fragment claims. `prepare` consulta requests de release se a capability está adotada, recusa foreign claims salvo exceção explícita e arquiva os fragments. [E01, E08, E10]

A seleção atual é **todos os fragments de `unreleased`**; não há argumento de seleção de specs/milestone/roadmap nos comandos inspecionados. O ADR descreve selected fragments, mas essa seleção é implícita pela fila atual. Isso funciona para cuts pequenos sob controle editorial; é insuficiente como expressão de um cut parcial em projeto com muitos trabalhos concorrentes. Não presumir que cada fragment pendente pertence ao mesmo compromisso de produto. [E01, E21]

`prepare` não modifica specs. A política atual de archival usa o manifest como testemunho do movimento de fragment e preserva a declaração e o subject selado da spec. Isso corrige a abordagem histórica que reescrevia artifact claims no cut e invalidava reviews. Qualquer nova ligação entre cut e specs deve preservar essa decisão. [E09, E22]

`checkRelease` compara policy digest, archive, notes, quantidade/digests de fragments, input digest reconstruído, presença duplicada pending/archive e gaps de lifecycle. `release check` acrescenta foreign fragment claims e informação da tag. Não recompõe readiness de cada spec/roadmap, nem comprova que o tree da tag contém o snapshot consultado. A CI deste repositório executa outros gates, e o workflow de release depende dela. Portanto, o gap no comando local não elimina os controles operacionais que já existem. [E01, E29, E30]

`scripts/release.sh` requer worktree limpo, recusa tag preexistente, roda snapshot check, suíte Go e compatibilidade e então cria/pusha uma nova tag. Seu sucesso de polling é existência de release no GitHub, não verificação independente completa. Não executar esse script nesta auditoria. Melhorar o texto/resultado final para distinguir publicação observada de readiness e verificação. [E32]

A importação de eventos foi desenhada para registrar fatos externos depois de ocorridos. Preservar a decisão da frente 4: uma pendência de execução não deve impedir registrar que o provider publicou. Validar a evidência e a associação, mas não ocultar o fato porque o cut violou uma precondição. Representar publicação observada com não conformidades, em vez de confundir auditoria com autorização retrospectiva.

## 6. Probes e resultados

Execute `python3 .pose/results/2026-10-12-pose-front-06-release-probes.py --binary /caminho/pose --output /tmp/front06.json`. O harness conserva fixtures e stdout/JSON suficientes para inspeção; o companion elimina duplicação do JSON em stdout.

| Probe | Cenário | Observação | Alcance |
|---|---|---|---|
| P01 | Fragment de spec draft; roadmap com `check:never-produced` | Roadmap exit 1; plan sem blockers; prepare exit 0; check strict `valid: true` | Avaliadores desconectados; não prova aprovação por CI real |
| P02 | Critério sob `Release cut criteria` | Zero critérios, terminal true, warning de ausência | Descoberta de intenção não executada |
| P03 | `manual-review:` aponta para arquivo vazio | Critério e roadmap passam | O contrato comprova resolução de path, não aprovação semântica |
| P04 | Alterar schema, specs, categories e breaking do manifest | Check strict continua válido; status expõe campos alterados | Projeções redundantes sem recomposição e validação de schema |
| P05 | Version source muda após freeze de candidato não tagged | Check continua válido; plan do mesmo target recusa versão | Falta declaração de escopo temporal; histórico deve permanecer verificável após abrir próximo ciclo |
| P06 | Tag aponta para commit que não contém manifest; worktree contém snapshot | Check strict válido com tagged true | Falta vínculo entre snapshot consultado e tree da tag |
| P07 | Verified liga publication digest correto, mas apresenta assets diferentes | Record aceita; status verified; check válido | Vínculo por digest da publicação não comprova assets verificados |
| P08 | Provider/repository do evento difere da policy | Tagged e published aceitos, check válido | Identidade presente não equivale a associação ao release correto |
| P09 | Alterar assets no evento published mantendo evidence digest antigo | Status verified, check válido, open-next aceita próximo ciclo | Replay não revalida hash; efeito chega a consumidor concreto |
| P10 | Recriar fragment da mesma spec para segunda release | Novo plan/prepare/check passam | Exclusividade global não está no preflight local do cut |
| P11 | Dois fragments pendentes, um de spec draft | Ambos entram no mesmo freeze | Seleção implícita pela fila inteira |
| P12 | Archive alterado; verified com vínculo errado | Check recusa archive; record recusa vínculo | Controles positivos de detecção preservados |
| P13 | Surface declarada sem proveniência/reachability/e2e, arquivos commitados | Roadmap passa C1 e terminal; surface-check strict exit 1 | Declaração não implica suficiência do delivery |
| P14 | Critério isolado `evidence:e2e`, sem resultado de evidência | Critério e roadmap passam | Ref é parseada, mas não tem avaliação correspondente no loop |

P03 não demonstra fraude criptográfica: os testes atuais tratam `manual-review:` como arquivo resolvível, e não como assinatura. A questão é explicitar essa garantia e fornecer mecanismo de judgment quando o critério exige aprovação. P05 não justifica comparar toda release histórica com a versão corrente: avanço de versão é normal. A recomendação é separar revalidação de candidato e integridade histórica. P10 é detectável pelo check estrutural global que enumera assignments; o probe não demonstra que a CI global aceitaria a duplicação. [E01, E13–E18]

P13 não desliga o scanner de delivery: ele está ativo e encontra os erros. A divergência nasce da avaliação de critérios, que reconhece o target e a deferral, mas não consome necessariamente seus findings e classes requeridas; o gate final filtra findings do roadmap por path do próprio slug. `evidence:` é reconhecida pelo parser, mas não possui ramo de validação nesse loop. Corrigir a composição desses resultados; não criar mais uma declaração que replique o profile. [E03, E04]

P09 não é ataque remoto nem prova de alteração do provider. Um journal local foi modificado no fixture para testar o reader. O hash declarado deveria denunciar a divergência antes de `verified` alcançar `open-next`; isso complementa a frente 2, que é responsável pela propagação geral de assurance. Aqui o efeito e a correção são específicos do lifecycle de release.

## 7. Contrato recomendado para uma release suficiente

Uma release suficiente precisa dizer **o que entrega, a quem, em qual estado e com quais limites**. O contract deve vincular cut, compromissos e evidências usando fontes existentes, não copiar requisitos completos para outro arquivo.

Registrar no plano e no snapshot: specs/milestones/outcomes selecionados, critérios aplicáveis, targets de delivery, requisitos ou relações residuais que afetam o cut, versões dos contratos e referências/digests da aceitação e validação. Separar os excluídos por intenção de cut parcial dos adiados por decisão explícita. Excluir trabalho opcional fora do escopo não é waiver de requisito comprometido.

| Condição | Efeito recomendado sobre o cut |
|---|---|
| Spec selecionada não encerrada segundo policy aplicável | Restringir ou exigir tipo de candidato explicitamente admitido; não apresentar como entrega completa |
| Critério de outcome obrigatório não atendido | Restringir o cut daquele compromisso |
| Declaração de delivery existe, mas prova obrigatória falta/stale | Unknown/not-ready, com fonte e ação necessária |
| Requirement obrigatório encaminhado a outra spec | Seguir a intenção até destino; encaminhamento sozinho não satisfaz |
| Defeito viola promessa do cut | Restringir conforme severity/policy ou exigir exceção válida |
| Dívida técnica aceita, fora do compromisso atual | Apresentar residual e aceitação; não criar bloqueio global |
| Validação/adoção no consumidor ainda pendente | Limitar a afirmação a engine implementado; restringir promessa de capacidade composta |
| Operação externa futura em outro canal | Declarar disponibilidade por canal; não inventar publicação simultânea |
| Fonte necessária inacessível ou contrato não reconhecido | Declarar cobertura incompleta, sem afirmar readiness completa |

A release pode entregar um subconjunto coerente de um programa ainda aberto. Reusar o escopo de milestone já existente; não exigir concluir roadmaps inteiros para patch que resolve uma única spec. Ao mesmo tempo, não liberar uma promessa composta apenas porque todas as implementações locais estão `done`: o coordenador continua responsável pela composição e adoção. [E16, E23]

Definir query de suficiência como avaliação por condição: atendida, não atendida, dispensada por exceção válida, não aplicável e desconhecida. O lifecycle de release continua prepared/tagged/published/verified/failed/yanked. Uma publicação pode existir e ainda violar o compromisso; integridade de assets e suficiência funcional são dimensões independentes.

Reutilizar artifact graph, delivery profiles, closeout, aceitação federada e requests por exceção. Executar validação cara separadamente e consumir resultados atuais; não rodar a suíte inteira em toda listagem. Não usar um booleano `ready` sem coverage e sem discriminar a fase: prontidão para freeze, tag, publicar e afirmar verified têm requisitos diferentes.

## 8. Integridade da cadeia e evolução temporal

Validar o schema e recompor todos os campos derivados do manifest. O digest atual liga version/previous/fragments/policy/evidence, mas não recompõe os campos specs, categories e breaking armazenados. Projeções redundantes precisam bater com inputs ou ser derivadas no reader. Validar também identidade de version/path e shapes obrigatórios, com diagnóstico tipado. [E01, E02]

Vincular tag ao conteúdo congelado. Não exigir `prepared_at commit == tag commit`, pois o manifest só passa a existir no commit de freeze; armazenar o próprio SHA do commit que contém manifest também cria ciclo. Comparar o tree/tag e os hashes dos artefatos selecionados, incluindo version source quando aplicável, e permitir commits de registro de evidência posteriores sem alterar a identidade do candidato. Não resolver divergência movendo ou forçando tag. [E21, E22]

Para eventos, revalidar schema, identidade, evidence digest e transição na leitura, além da entrada. Comparar assets publicados e verificados segundo contrato de cobertura: mesmo asset precisa ter mesmo digest; verificação parcial deve declarar subconjunto e não afirmar verificação total. Assets extras, como o próprio relatório de publicação, precisam de regra explícita. Diferenciar publisher e verifier, preservando os providers legítimos observados em v7.4.1. [E02, E37–E40]

Importar evidência tipada de arquivo não autentica a origem do arquivo. Conectar a proof/assurance do verifier à cadeia de confiança da frente 2, sem tentar reinventar Sigstore no ledger. O verifier existente já autentica checksum, assinatura e provenance antes de executar assets; o registro deve expor qual relatório/run produziu o julgamento, quais assets e plataformas foram realmente executados e quais receberam somente inspeção. Não chamar verificação Linux de teste nativo em todas as plataformas. [E31, E33]

A validade temporal tem dois contratos: avaliar **candidato corrente** com fontes atuais e avaliar **snapshot histórico** com contratos que governavam aquele cut. Alterar policy ou versão do projeto não deve transformar toda release antiga em inválida por padrão. O modo deve ser explícito, com compatibilidade e migration preview. Os knobs `tag_pattern` e `require_verification` estão presentes na policy, mas não são lidos como decisões operacionais nos caminhos inspecionados; esclarecer suporte ou implementação, sem fazer opções declaradas parecerem garantias efetivas. [E01, E02, E24, E37]

A recuperação precisa preservar tentativas e permitir rejeitar prova conflitante sem apagar história. O writer atual recusa segundo evento do mesmo state com evidence digest diferente, e aceita replay idêntico. Isso é útil para idempotência, mas precisa ser reconciliado com a regra de que publicação diferente invalida verificação anterior: decidir se mudança de assets gera tentativa/revisão explícita ou exige novo release. Não recomendar publicação mutável nem sobrescrita de tags. Esse cenário de retry não foi reproduzido; é questão de contrato confirmada por leitura. [E01, E02, E25]

## 9. Backlog proposto e critérios de aceite

Prioridades são propostas de auditoria, não compromisso de versão nem criação automática de specs. Conservar estados históricos; abrir continuação corretiva quando necessário. Reaproveitar as specs/ADRs existentes como origem dos requisitos.

| ID | Prioridade | Entrega | Critérios de aceite |
|---|---|---|---|
| F06-01 | P1 | Suficiência compartilhada no preflight | P01 expõe blockers de spec/escopo selecionado antes de freeze; plan e apply avaliam o mesmo contrato; nenhum roadmap alheio cria bloqueio global |
| F06-02 | P1 | Adequação de critérios tipados | P13 recusa delivery sem prova exigida; P14 recusa ou declara inválida ref sem evaluator; profile e provenance atuais governam; deferral continua não afirmando entrega |
| F06-03 | P2 | Descoberta e declaração dos critérios | P02 denuncia heading/intenção não reconhecida; template oferece sintaxe canônica; ausência intencional é distinta de parse incompleto; P03 distingue path existente de approval |
| F06-04 | P1 | Validação completa do manifest | P04 recusa schema inválido e projeções divergentes; campo desconhecido/shapes/version/path têm diagnóstico; rebuild deve detectar cada alteração semanticamente relevante |
| F06-05 | P1 | Replay e vínculo da evidência de release | P07–P09 não produzem verified total; recomputar hashes; identidade respeita papéis publisher/verifier; cobertura parcial explícita; open-next usa resultado validado |
| F06-06 | P1 | Snapshot ligado ao tree da tag | P06 recusa tag sem snapshot; conferir archive/notes/version source no tag; commits posteriores de evidência não staleiam cut correto; tag nunca é movida pelo checker |
| F06-07 | P1 | Exclusividade antes da mutação | P10 é recusado no plan/apply, sem mover fila; compartilhar descoberta com check global; leitura incompleta não vira ausência de assignment |
| F06-08 | P2 | Seleção explícita e contrato do cut | P11 admite selecionar subconjunto coerente; manifest inclui scope/refs/digests; exclusões e adiamentos têm significado distinto; requisição/policy decide exceções |
| F06-09 | P2 | Contratos temporais e recuperação | P05 diferencia candidato stale de história íntegra; prospectividade e migration preview; knobs de policy têm status de suporte; retries/conflitos conservam lineage e invalidam prova superseded |
| F06-10 | P2 | Assurance operacional e canais | Resultado nomeia publisher/verifier/run, matriz de assets/plataformas/canais e limitações; CLI/script distinguem snapshot válido, publicação e verificação; import factual não é bloqueado por autorização retrospectiva |

F06-01 e F06-08 continuam o lifecycle de release e a composição existente no roadmap; não substituir aceitação federada. F06-02–03 continuam delivery integrity/roadmap gate. F06-04–07 completam R4/R7/R10/R12–R14/R18 do lifecycle já entregue, com probes específicos. F06-09 precisa ADR prospectivo para suportar mudanças sem reclassificar automaticamente história. F06-10 reusa independent verification e integra disclosure da frente 2.

## 10. Entregas coesas e alternativas

**Entrega A — confiabilidade dos contratos atuais.** F06-02, F06-04–07 e a parte mínima de F06-03 que revela perda de cobertura. Corrigir divergências reproduzidas primeiro: critérios com evidência ausente, campos semânticos do manifest, replay, assets, vínculo com tag e fragment já atribuído. Acrescentar negativos sem transformar metadados opcionais de legacy em falha retrospectiva indiscriminada. Preservar P12 e os testes de scope/federation.

**Entrega B — cut governado por compromisso.** F06-01, F06-08–10 e restante de F06-03, após contrato de identidade/coverage consolidado. Pilotar seleção pequena de specs/milestone, preview consistente, prova de composição e disponibilidade por canal. Não exigir um roadmap novo para cada patch nem uma decisão humana para cada read; policy controla exceções materiais.

| Alternativa | Benefício | Limite/decisão |
|---|---|---|
| Manter somente workflow e triagem manual | Baixo custo, serve como ponte | Comandos locais continuam podendo sugerir garantia maior que a observada |
| Compor avaliadores existentes em release preflight | Reusa contratos e preserva offline | Recomendado; exige snapshot, coverage e seleção explícita |
| Fazer release check rodar todos os checks do projeto | Aparente centralização | Alto custo, ruído e bloqueio por trabalho fora do escopo; consumir resultados adequados em vez disso |
| Novo ledger de compromissos completo | Relações fortes | Duplica requirements/findings/follow-ups; preferir refs e testemunhos |
| Usar fragments como único contrato de produto | Simples | Notes são descrição editorial, não critério verificável de suficiência |

A versão correta pode ser patch, minor ou major segundo política e contrato público. A recomendação atual de SemVer usa `breaking` e categoria `added`, sem decidir o bump por autoridade própria; manter target explícito. As novas falhas obrigatórias e mudanças de schema precisam de adoção prospectiva e análise de compatibilidade, não de imposição indiscriminada a toda instância.

## 11. Dependências entre frentes

A frente 2 governa confiança, prova e propagação de assurance; a 3 governa review delegado e conclusão pelo engine; a 4 governa escopo/efeitos das requests; a 5 classifica residual e compromisso encaminhado. Nesta frente, essas capacidades determinam o alcance do cut, sem duplicar seu backlog.

A frente 7 detalhará mensagens e UX das fases; a 8 medirá custos e reuso de snapshots; a 9 aprofundará concorrência, rollback e append seguro dos eventos. `prepare` possui rollback local, mas não constitui transação robusta contra crash; o writer de eventos usa append sem lock/fsync demonstrado nesses caminhos. Esses temas pertencem à frente 9, e não foram reproduzidos aqui como novos bugs independentes. A frente 10 consolidará as dependências, deduplicará itens e definirá versões completas.

## 12. Validação da entrega documental

O harness final contém 14 probes, executados em fixtures commitadas; P13 inclui uma rodada exploratória que foi ajustada para eliminar source revision mismatch antes da retenção oficial. O JSON conserva a rodada final, que sustenta as conclusões. Validar sintaxe Python, referências locais, hashes das 41 fontes e coerência dos resultados. `pose check --strict` passou na baseline e após os documentos: `Result: SUCCESS — (strict mode) with 30 warning(s).` Os 30 avisos preexistentes foram mantidos; comando, resultado e hash do log constam no JSON. Não implementar engine, alterar policy, publicar release ou mudar disposition nesta entrega.

## 13. Fontes vinculadas à revisão auditada

- **E01** — [pose-mcp/internal/cli/release_lifecycle.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/cli/release_lifecycle.go)
- **E02** — [pose-mcp/internal/pose/release_lifecycle.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/release_lifecycle.go)
- **E03** — [pose-mcp/internal/cli/surface_check.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/cli/surface_check.go)
- **E04** — [pose-mcp/internal/pose/delivery_surface.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/delivery_surface.go)
- **E05** — [pose-mcp/internal/pose/review_closeout.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/review_closeout.go)
- **E06** — [pose-mcp/internal/pose/federated_acceptance.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/federated_acceptance.go)
- **E07** — [pose-mcp/internal/pose/roadmaps.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/roadmaps.go)
- **E08** — [pose-mcp/internal/pose/release_version_source.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/release_version_source.go)
- **E09** — [pose-mcp/internal/pose/release_archive.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/release_archive.go)
- **E10** — [pose-mcp/internal/pose/release_claimed_fragments.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/release_claimed_fragments.go)
- **E11** — [pose-mcp/internal/pose/obligation_sources.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/obligation_sources.go)
- **E12** — [pose-mcp/internal/pose/readiness_phases.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/readiness_phases.go)
- **E13** — [pose-mcp/internal/cli/release_lifecycle_test.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/cli/release_lifecycle_test.go)
- **E14** — [pose-mcp/internal/pose/release_lifecycle_test.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/release_lifecycle_test.go)
- **E15** — [pose-mcp/internal/cli/roadmap_check_gate_test.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/cli/roadmap_check_gate_test.go)
- **E16** — [pose-mcp/internal/cli/roadmap_gate_scope_test.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/cli/roadmap_gate_scope_test.go)
- **E17** — [pose-mcp/internal/cli/surface_check_test.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/cli/surface_check_test.go)
- **E18** — [pose-mcp/internal/pose/delivery_surface_test.go](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/pose-mcp/internal/pose/delivery_surface_test.go)
- **E19** — [.pose/workflows/release.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/workflows/release.md)
- **E20** — [.pose/templates/roadmap.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/templates/roadmap.md)
- **E21** — [.pose/adr/2026-08-03-immutable-release-ledger.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/adr/2026-08-03-immutable-release-ledger.md)
- **E22** — [.pose/adr/2026-09-11-release-archival-is-attested-by-the-ledger-never-written-into-specs.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/adr/2026-09-11-release-archival-is-attested-by-the-ledger-never-written-into-specs.md)
- **E23** — [.pose/adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md)
- **E24** — [.pose/adr/2026-10-02-release-version-source-is-declared-by-the-project.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/adr/2026-10-02-release-version-source-is-declared-by-the-project.md)
- **E25** — [.pose/specs/2026-08-02-pose-release-lifecycle-closure.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/specs/2026-08-02-pose-release-lifecycle-closure.md)
- **E26** — [.pose/specs/2026-09-09-pose-roadmap-check-reaches-its-gate.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/specs/2026-09-09-pose-roadmap-check-reaches-its-gate.md)
- **E27** — [.pose/specs/2026-10-09-pose-release-prepare-guards-claimed-fragments.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/specs/2026-10-09-pose-release-prepare-guards-claimed-fragments.md)
- **E28** — [.pose/specs/2026-09-26-pose-roadmap-gate-scopes-milestones-and-external-members.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/specs/2026-09-26-pose-roadmap-gate-scopes-milestones-and-external-members.md)
- **E29** — [.github/workflows/release.yml](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.github/workflows/release.yml)
- **E30** — [.github/workflows/ci.yml](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.github/workflows/ci.yml)
- **E31** — [.github/workflows/verify-release.yml](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.github/workflows/verify-release.yml)
- **E32** — [scripts/release.sh](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/scripts/release.sh)
- **E33** — [tests/release/independent-verify.sh](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/tests/release/independent-verify.sh)
- **E34** — [.pose/roadmaps/agency-readiness.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/roadmaps/agency-readiness.md)
- **E35** — [.pose/roadmaps/community-launch.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/roadmaps/community-launch.md)
- **E36** — [.pose/roadmaps/pose-v7-onboarding-and-consolidation.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/roadmaps/pose-v7-onboarding-and-consolidation.md)
- **E37** — [.pose/policy/release.json](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/policy/release.json)
- **E38** — [.pose/releases/v7.4.1/manifest.json](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/releases/v7.4.1/manifest.json)
- **E39** — [.pose/releases/v7.4.1/events.jsonl](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/releases/v7.4.1/events.jsonl)
- **E40** — [.pose/releases/v7.4.1/verified-evidence.json](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/releases/v7.4.1/verified-evidence.json)
- **E41** — [.pose/reports/2026-10-03-pose-consolidated-analysis.md](https://github.com/oseiaspereira88/pose/blob/b983245b6e4aef12e4ffc429b0b1b71a4e8f638d/.pose/reports/2026-10-03-pose-consolidated-analysis.md)
