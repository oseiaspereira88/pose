# Frente 5 — dívida técnica, defeitos e trabalho incompleto acumulado

Data: 2026-10-12. Baseline: `61c2909949fe50d2d3ca47c3c380aae688cba078`, após o merge do PR #150; engine distribuído 7.4.1. Público: mantenedores do POSE, implementadores do motor e operadores de projetos consumidores.

[Índice do programa](2026-10-11-pose-analysis-index.md) · [Inventário, fontes e observações estruturadas](../results/2026-10-12-pose-front-05-residual-work.json) · [Script de reprodução](../results/2026-10-12-pose-front-05-residual-probes.py).

## 1. Conclusão e limites

O POSE já administra disposições, dono, criticidade, prazo de triagem e candidatos de reconciliação. Não é necessário construir outro backlog paralelo. O problema atual é que esses mecanismos respondem a perguntas diferentes e ainda não preservam todas as distinções necessárias: uma pendência consultiva não equivale a defeito, obrigação de adoção, implementação prometida ou condição de release. A projeção de follow-ups transforma todo `[open]` em `residual-debt`, enquanto o assessment chama de cobertura qualquer menção ao caminho de um arquivo numa spec ativa. São simplificações com consequências operacionais. [E01–E09]

A recomendação é acrescentar semântica e fortalecer identidade, cobertura e reconciliação nos mecanismos existentes. Manter os gates derivados de compromissos e evidências; classificar trabalho residual não deve bloquear projetos por padrão. Tampouco deve permitir que um requisito obrigatório desapareça apenas porque foi encaminhado a outra spec. A frente 6 definirá o vínculo exato com escopo e prontidão de releases.

Esta análise inspecionou código, testes existentes e artefatos históricos; leu os 108 textos atuais e executou oito cenários sintéticos no binário oficial disponível. O binário não foi recompilado deste checkout. A suíte Go não foi reexecutada: não há executável Go disponível. Fixtures pequenas demonstram comportamentos, sem provar efeito em todos os consumidores nem bypass completo de closeout. O JSON conserva os resultados, inclusive as limitações de cobertura do Attention das fixtures: snapshots coerentes, mas producer `start` unsupported porque atomic start não foi adotado. Essa limitação não foi ocultada nem tratada como cobertura global completa.

A classificação dos 108 itens é proposta editorial, não adjudicação de satisfação. Para 92 itens houve correspondência exata com a reconciliação histórica; sua classe foi preservada como hipótese, não revalidada contra toda implementação. Os 16 restantes receberam classe pela leitura da intenção. Nenhuma spec, disposition, policy ou release foi alterada.

## 2. O que já existe e deve ser preservado

| Mecanismo | Garantia presente | Limite atual | Evidência |
|---|---|---|---|
| Follow-ups canônicos | Parser compartilhado lê layouts flat e folder, bullets com wrapping e metadata final | Sem tipo de trabalho ou identificador autoritativo estável | E01, E03 |
| Disposições | `open`, `spawned`, `covered`, `duplicate`, `done`, `wont-do`; lint valida formato e referências | Encaminhamento não prova atendimento; não há monitoração geral do destino | E04, E24, E25 |
| Ownership/SLA | `owner`, `crit`, `review`, `by`; vencimento e filtro por dono | SLA de triagem, não prazo de implementação; calendário validado apenas por formato | E01, E03, E16, E26 |
| Reconciliação | Destino terminal, teste existente, revisão vencida e similaridade lexical levantam candidatos | Candidatos Go/textuais e principalmente itens explicitamente `open`; não verificam adequação | E02, E15, E20 |
| Attention | Follow-ups de specs `done` permanecem consultivos | Perde bullets sem disposition e não separa naturezas do trabalho | E07, E08 |
| Assessment | Scanner lexical distingue comentários, strings e construções executáveis; IDs resistem a deslocamento de linha | Cobertura por substring de arquivo; identidade repetida; perda silenciosa de scan | E05, E06, E14, E23 |
| Review findings | Findings abertos/changes-requested bloqueiam verificação; risco aceito depende da policy, dono, razão e data | Não é o mesmo contrato de follow-up; formatos de review têm diferenças | E11, E12 |
| Reconciliação do projeto | Relatório e JSON históricos classificaram 120 itens e requisitos de 12 specs | Snapshot histórico, não inventário vivo reconciliado automaticamente | E17–E19 |

A migração anterior já corrigiu o caso em que apenas mencionar o componente cobria todos seus marcadores. O código atual exige arquivo ou ID, e há teste dessa garantia. O achado novo é mais estreito: mencionar o arquivo continua cobrindo todos seus marcadores sem compromisso específico. Preservar a melhoria e completar a precisão. [E05, E14, E21]

Também existe sinalização de revisão vencida para findings aceitos ou encaminhados no último review record de um escopo. Ela é consultiva para release. A verificação de attestation aceita `resolved`, `wont-fix` e `accepted-risk` com regras próprias. Não unificar esses vocabulários por simples substituição textual: mapear a intenção e declarar qual registro tem autoridade. [E11, E12]

## 3. Inventário atual e reconciliação histórica

As consultas atuais retornaram 309 follow-ups no total: 108 `open`, 98 `done`, 48 `covered`, 44 `spawned`, nove `wont-do` e dois `duplicate`. Dos abertos, 107 pertencem a specs `done` e um a spec `in-progress`; 11 têm `review` vencido e 42 são `unowned`. Há quatro `high`, 22 `medium`, 63 `low` e 19 sem criticidade. Ausência de dono e ausência de criticidade são dimensões diferentes, portanto as contagens não precisam coincidir.

`followups --candidates` produziu 16 candidatos: 11 com revisão vencida, cinco com destino terminal e um com teste existente; um item tem mais de uma razão. Não converter essa quantidade em “16 resolvidos”. Candidato com destino abandonado significa necessidade de reconsiderar a intenção, não evidência de entrega. Teste existente não informa se foi executado, passou ou cobre o comportamento devido. [E02]

O relatório histórico mediu 120 abertos em `4133b56`, com 11 vencidos e 52 sem dono. A análise anterior citava 123 em outra baseline. Não calcular produtividade ou taxa de resolução subtraindo snapshots distintos. Houve novas entregas, novos follow-ups e triagens; o que importa é a reconciliação por intenção e identidade. [E18, E19, E28]

### Classes propostas para os 108 itens

| Classe operacional herdada/proposta | Quantidade | Como usar |
|---|---:|---|
| Residual aceito ou adiado | 39 | Confirmar o gatilho de reconsideração e a decisão original; texto condicional sozinho não prova aceite formal |
| Implementação material | 27 | Conferir comportamento atual e se há compromisso ativo, defeito ou melhoria opcional |
| Adoção, piloto ou rollout | 16 | Verificar o consumidor competente; implementação no engine não comprova adoção |
| Operação externa ou humana | 13 | Verificar se há intenção vigente e autorização; criar request apenas quando uma ação material for necessária |
| Revisão/aceitação de evidência | 12 | Buscar resultados atuais, critérios e autoridade; evitar reimplementar trabalho existente |
| Candidato à satisfação | 1 | Revalidar adequação antes de escrever disposition |

O JSON lista cada texto, origem, metadata, classe proposta, vínculo histórico e limite. Os números são locais ao snapshot e às classes editoriais; não são métricas do engine nem certificados de dívida aceita.

Exemplos que orientam a próxima triagem:

- `pose-check-spawns-fewer-git-processes` pede batching de leituras Git remanescentes. É implementação de performance; a frente 8 deve medir e decidir a arquitetura.
- `pose-recurrence-flapping-signal` pede investigar alternância de `validate-native`. É investigação de confiabilidade, sem defeito causal já provado pelo texto.
- `pose-v7-1-0-version-alignment` registra limpeza de temporários após interrupção e impacto em quota. A frente 9 deve verificar recuperação e lifecycle dos fixtures.
- Os quatro itens `high` pedem re-pin, validação ou reconciliação no Harne8. Sua severidade não torna o POSE inteiro bloqueado; são obrigações de adoção no consumidor competente.
- `pose-attention-projects-every-source` pede evidência de publicação/verificação de v7.0.0. Registro de publicação faltante não prova release defeituosa ou não publicada.
- Pedidos de Homebrew, tradução e catálogo misturam decisões de produto, manutenção e capacidades adicionais. Não chamar toda funcionalidade desejada de dívida técnica.

Há um item histórico de satisfação ainda aberto com referência a `pose-onboarding-context-extraction`. A análise anterior é pista; o estado atual exige comprovar intenção atendida e autoridade, sem auto-close.

## 4. Observações reproduzidas

Execute o script com `python3 .pose/results/2026-10-12-pose-front-05-residual-probes.py --binary /caminho/pose --output /tmp/front05.json`. Ele cria repositórios sintéticos, usa somente arquivos locais e mantém fixtures para inspeção. Não invoca agentes, provedores ou operações em repositórios reais. Os IDs são do snapshot; novas correções podem mudar os resultados.

| Probe | Preparação | Resultado observado | Interpretação |
|---|---|---|---|
| P01 | Spec `done` com duas pendências vencidas, uma sem disposition e outra `[open]` | CLI contabiliza duas abertas; candidatos e residual de Attention incluem apenas a explícita | Paridade de leitura incompleta; lint pode denunciar o formato, mas as consultas ainda divergem |
| P02 | Arquivo com TODO, FIXME e panic; spec draft cita o arquivo apenas como contexto | Três marcadores `covered_by_spec`, zero recomendações | Menção de localização não comprova compromisso sobre cada ocorrência |
| P03 | Dois comentários TODO idênticos no mesmo arquivo | Duas ocorrências com o mesmo ID | Colisão determinística da identidade; não é colisão criptográfica aleatória |
| P04 | Linha de comentário com 70 mil caracteres antes de um TODO | Sucesso com zero marcadores | `bufio.Scanner` para em token longo; `scanner.Err()` não é consumido |
| P05 | Metadata com `review:2026-99-99` | Campo aceito sem `meta_error`, não vencido, sem candidato | Regex verifica formato, não calendário |
| P06 | Follow-up `spawned: beta`, destino `abandoned` | Zero abertos e nenhum candidato de reconciliação | Falta de acompanhamento do encaminhamento; abandono pode ser legítimo, mas intenção originária precisa de resultado explícito |
| P07 | Inserir nova bullet antes da pendência existente | ID da pendência anterior muda sem mudança de texto | Ordinal entra no NodeRef e no hash; identidade não sobrevive à inserção |
| P08 | `[open]` cita spec done e teste existente | Duas razões de candidato, spec permanece byte a byte igual | Controle positivo: candidato não muda disposition |

P06 demonstra ausência de sinal, não obrigação de reabrir automaticamente nem bypass de gate. O destino pode ter sido abandonado por decisão válida registrada em outro lugar. P01 usa documento inválido para o fechamento; não demonstra que um closeout correto o aceitou. P04 usa arquivo abaixo do limite geral de tamanho e uma única linha longa: a perda não decorre da exclusão documentada de arquivos gigantes.

P07 é particularmente relevante porque a limitation atual descreve apenas rewording como origem de um novo ID. O código inclui `f<ordinal>` no source; portanto inserir, excluir ou reordenar bullets também muda identidade. Uma referência `spec#ordinal` deve ser apresentada como localização no snapshot, não chave durável. [E07]

## 5. Modelo recomendado: separar natureza, compromisso e efeito

Não ampliar `status` da spec com todos os tipos de problema. Manter o lifecycle pequeno e registrar dimensões independentes do trabalho residual.

| Natureza proposta | Pergunta respondida | Evidência mínima para classificá-la |
|---|---|---|
| Dívida técnica | Há decisão técnica com custo futuro conhecido? | Decisão/limitação, custo ou risco e condição de reconsideração |
| Defeito | Qual comportamento viola o esperado? | Esperado versus observado, reprodução ou evidência dirigida |
| Compromisso incompleto | O que foi prometido e ainda falta? | Requirement/escopo autoritativo e parcela não atendida |
| Lacuna de evidência | O trabalho existe, mas falta prova adequada? | Critério e evidência ausente/insuficiente |
| Adoção ou operação | Qual consumidor/canal ainda precisa executar algo? | Destino, ação e responsabilidade competente |
| Melhoria opcional | Qual novo valor está sendo proposto? | Justificativa e gatilho, sem inventar promessa anterior |
| Não classificado | Falta contexto suficiente? | Limite explícito, sem inferir dívida ou satisfação |

“Fratura de release” deve ser relação com uma entrega específica, não natureza exclusiva: um defeito pode afetar uma release publicada; uma evidência faltante pode impedir demonstrar prontidão sem significar defeito no produto. “Duplicado” e “obsoleto” são resultados de reconciliação com justificativa, não explicações causais do trabalho.

Distinguir em paralelo: criticidade do impacto, urgência, responsável pela triagem, responsável pela implementação, prazo de triagem, compromisso de entrega, destino e efeito por fase. Um item `high` pode estar aceito para um consumidor futuro; um item `low` pode ser requisito obrigatório do release cut atual. O bloqueio nasce do compromisso, não da etiqueta.

A forma mínima pode ser metadata opcional na fonte canônica. Não decidir aqui a sintaxe exata; avaliar extensão do formato de follow-up contra referências a findings/requirements existentes. Exigir ADR para mudança estrutural, schema/versionamento e migração prospectiva. Não criar um arquivo por TODO nem uma spec por pendência.

## 6. Encaminhamento, envelhecimento e fechamento

`spawned` significa que existe um destino para a intenção; `covered` declara cobertura; `duplicate` declara equivalência. Nenhum dos três, isoladamente, demonstra satisfação. A consulta de trabalho residual precisa mostrar a cadeia de encaminhamento e seu resultado, sem contar origem e destino como duas implementações devidas.

Quando o destino está `done`, levantar candidato de adequação. Quando está `abandoned` ou `superseded`, verificar razão e sucessor. Se o compromisso original continua vigente, expor perda de cobertura; se foi retirado validamente, mostrar a decisão. Detectar referências órfãs e ciclos. Não usar o estado terminal genérico como sinônimo de entrega. [E02, E04, E13]

Para dívida adiada, exigir condição de reconsideração quando material: data, mudança de escala, incidente ou entrada em novo escopo. A data vencida abre triagem; não determina implementação imediata. Renewal precisa guardar razão e evidência, sem simplesmente empurrar a data e apagar histórico. Repetidas postergações podem gerar sinal consultivo e levar a decisão de produto.

Aceitar risco em review já possui contrato com severity permitida, owner, rationale e review date. Aproveitar esse modelo para vínculos de risco, sem equiparar `wont-do: reason` a aceitação autorizada de risco. `done` em follow-up deve ter adequação e evidência; `[wont-do]` deve ter razão e autoridade proporcional ao compromisso abandonado. Não exigir interação humana para toda correção ou revisão de triagem: autonomia deve seguir policy, com pessoas nas exceções.

Reconciliar em lote com plano revisável, snapshot e diff. Aplicar somente decisões sustentadas e autorizadas, com detecção de mudanças concorrentes. Quando o resultado ainda é candidato, persistir a análise de triagem ou apenas projetá-la; não forçar disposition definitiva para baixar contagem. A fonte canônica permanece o follow-up ou finding original, e o relatório não se torna outro backlog concorrente.

## 7. Assessment: evidência de localização não é cobertura de intenção

O scanner tem valor como detector de sinais. Um `panic` pode ser validação intencional de invariantes; TODO pode registrar evolução opcional. Nem todo marcador é dívida, e muitos defeitos não carregam marcador. Manter `marker-detected` separado de `confirmed-debt` e `confirmed-defect`.

Substituir cobertura binária por estados explícitos: associação lexical candidata, compromisso identificado, evidência validada, ausência de cobertura e cobertura desconhecida/incompleta. Exigir ligação ao marcador específico ou conjunto explicitamente enumerado. Path por substring também admite prefixos e menções negativas; resolver referências estruturadas quando houver, e declarar limites da heurística quando não houver. [E05]

Hoje a consulta pode usar a seção `Follow-ups` de ProjectState, que é derivada e pode estar retida. Sua presença não comprova intenção vigente. O provider produz contagens e referências, sem copiar texto completo de todo item. Preferir parser canônico para cobertura ativa, usando o state como apresentação/cache validado; declarar revisão e freshness. O caso de cobertura por state obsoleto não foi reproduzido nesta frente. [E05, E09]

A recomendação de spec/roadmap deriva de volume por componente: mais de 15 marcadores sugere spec; mais de 50 em mais de cinco arquivos sugere roadmap. Isso não estima esforço, risco ou coesão. Manter sugestões como heurísticas transparentes e avaliar amostragem/agrupamento por intenção. Dezesseis panics legítimos não justificam automaticamente nova spec.

Para IDs, distinguir identidade de intenção e ocorrência: o hash atual de path, marker e snippet mantém estabilidade quando a linha se move, mas duas linhas iguais colidem; renomear arquivo ou editar texto muda chave. Avaliar ID explícito apenas quando o item passa a ser rastreado, com occurrence IDs e alias para migração. Não penalizar alterações de código normais com criação automática de dívida nova.

## 8. Backlog proposto e critérios de aceite

Estes itens são recomendações desta auditoria, não novas specs adjudicadas. Reutilizar os contratos existentes e registrar continuação com requisitos próprios, sem reabrir indiscriminadamente specs históricas `done`.

| ID | Prioridade proposta | Entrega | Critérios de aceite verificáveis |
|---|---|---|---|
| F05-01 | P1 | Paridade e cobertura das consultas | P01 retorna o mesmo conjunto não resolvido em CLI, MCP e Attention; formato inválido aparece com diagnóstico; fonte ilegível declara cobertura incompleta |
| F05-02 | P1 | Cobertura de marker por compromisso | P02 produz associação candidata, sem afirmar compromisso; dois markers no arquivo podem ter cobertura distinta; menção negativa e prefixo de path não cobrem por engano |
| F05-03 | P1 | Scan completo ou limitação explícita | P04 encontra o TODO ou informa scan incompleto; testar arquivo ilegível, scanner error e exclusões; nenhum consumidor apresenta zero como cobertura completa |
| F05-04 | P1 | Identidade de follow-up e ocorrência | P03 distingue ocorrências; P07 preserva identidade rastreada; rewrap e reordenação não perdem vínculos; renomeação/migração declara aliases e limitações |
| F05-05 | P2 | Datas e metadata válidas | P05 rejeita calendário inválido; cobrir ano bissexto, campos duplicados e metadata incompleta; SLA não vira prazo de implementação |
| F05-06 | P1 | Reconciliação dos encaminhamentos | P06 sinaliza destino abandonado/superseded/órfão; resolve sucessor e detecta ciclo; done exige adequação; nenhum caso aplica disposition automática |
| F05-07 | P2 | Natureza e compromisso no residual | Projetar dívida, defeito, incompletude, evidência e adoção sem mudar lifecycle; legacy fica não classificado; nenhum rótulo cria gate sozinho |
| F05-08 | P2 | Triagem atualizável e revisável | Reconciliar os 108 por snapshot, diff e fonte; decisões com evidência/autoridade; detectar mudança concorrente e aplicação repetida; manter candidatos abertos |
| F05-09 | P2 | Aceite, adiamento e reavaliação | Distinguir risco aceito de wont-do; preservar razão, policy, validade e sucessão; vencimento gera triagem; exceptions seguem ActionRequest existente |
| F05-10 | P2 | Métricas e recomendações úteis | Separar markers, intenções, cobertura, encaminhamento e implementações devidas; mostrar limites das heurísticas de volume; medir idade desde fatos autoritativos, não mtime |

F05-01 referencia parser/obligation projection e a garantia R4 de `pose-followup-reconciliation-candidates`; a perda de leitura foi encontrada na frente 1. F05-02–03 continuam precisão do assessment e ativação de governança. F05-04 estende os contratos de ownership/projeção com identidade real. F05-06 e F05-08 continuam reconciliação/candidatos, aproveitando relatório existente. F05-07 e F05-09 exigem decisão estrutural prospectiva. Não criar dez specs automaticamente: agrupar por causa e release coesa.

## 9. Composição de entregas e alternativas

**Entrega A: confiabilidade das consultas existentes.** F05-01–06, com subentrega inicial de scan/cobertura e paridade. Preservar formato existente; acrescentar disclosure/schema de consulta quando necessário. Incluir negativos para P01–P07 e controle positivo P08. Essa entrega reduz omissões e afirmações indevidas antes de ampliar taxonomia.

**Entrega B: gestão explícita do residual.** F05-07–10 após identidade e coverage estáveis, com piloto sobre subconjunto dos 108 e relatório de triagem. Reusar obligations, ActionRequests e findings; integrar compromisso de release somente após a frente 6. A numeração das versões será definida na frente 10, após reconciliar dependências das demais análises.

Alternativas avaliadas:

1. Apenas triagem manual periódica: compatível e útil imediatamente, mas não corrige perda de scan nem identidade; usar como ponte.
2. Extensão mínima de metadata e projeções: recomendada por preservar fontes e ergonomia; requer migração prospectiva e política clara para legacy.
3. Registro central de todos os resíduos: pode oferecer relações fortes, mas duplica intenção entre specs, findings e journal; só considerar se o piloto provar que a extensão não atende.
4. Bloquear tudo vencido ou converter todos os `[open]` em defects: rejeitar, porque mistura SLA, risco e compromisso e aumenta mecanização.
5. Auto-disposition por teste/spec terminal: rejeitar; controle P08 e contrato atual preservam julgamento de adequação.

A UX deve oferecer filas por intenção: corrigir comportamento, produzir evidência, adotar no consumidor, decidir exceção e revisar dívida. Mostrar ação, fonte, destino e limite antes da contagem. As consultas podem ordenar por urgência/impacto declarado sem afirmar prioridade derivada de ID; Attention atualmente ordena por ID. Evitar questionários repetidos para residual opcional e permitir triagem em lote com preview.

## 10. Fronteiras com outras análises

| Frente | Responsabilidade mantida lá | Uso nesta frente |
|---|---|---|
| 1 | Estado real e leitura incompleta de follow-ups | Inventário atualizado e caso de paridade |
| 2 | Verificação/autoridade até consumidores | Validade de decisões de risco e dispositions |
| 3 | Autonomia e revisão delegada | Triagem/review pode ser agêntica conforme policy |
| 4 | Requests, satisfação, efeito e escopo | Residual classificado não vira request/gate automaticamente |
| 6 | Compromissos e prontidão de releases | Definir quando o residual afeta uma entrega específica |
| 7 | UX e redução de cerimônias | Filas acionáveis, lote e mensagens |
| 8 | Performance/runtime | Escalabilidade das consultas e candidatos |
| 9 | Concorrência/recuperação | Aplicação de plano, scan incompleto e estado derivado |
| 10 | Backlog único e releases | Deduplicação e sequenciamento final |

## 11. Validação desta entrega documental

O script dos oito probes foi executado duas vezes, com o mesmo comportamento material; a segunda execução inicializa commits nas fixtures e é a retenção oficial. JSON, hashes das 28 fontes, referências locais e compilação sintática do script foram conferidos. `pose check --strict` passou na baseline e após os documentos: `Result: SUCCESS — (strict mode) with 30 warning(s).` Os 30 avisos preexistentes foram mantidos. O JSON registra o comando, resultado e hash do log final. Não foi executada a suíte Go, nem implementada correção no motor. O diff contém somente este relatório, evidências, script e índice.

## 12. Fontes vinculadas à baseline

- **E01** — [pose-mcp/internal/pose/followups_parse.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/followups_parse.go)
- **E02** — [pose-mcp/internal/pose/followup_candidates.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/followup_candidates.go)
- **E03** — [pose-mcp/internal/cli/followups.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/cli/followups.go)
- **E04** — [pose-mcp/internal/cli/lintspec.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/cli/lintspec.go)
- **E05** — [pose-mcp/internal/pose/techdebt.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/techdebt.go)
- **E06** — [pose-mcp/internal/pose/assessment_scan.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/assessment_scan.go)
- **E07** — [pose-mcp/internal/pose/obligation_projection.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/obligation_projection.go)
- **E08** — [pose-mcp/internal/pose/attention.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/attention.go)
- **E09** — [pose-mcp/internal/cli/state_providers.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/cli/state_providers.go)
- **E10** — [pose-mcp/internal/pose/cli.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/cli.go)
- **E11** — [pose-mcp/internal/pose/obligation_sources.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/obligation_sources.go)
- **E12** — [pose-mcp/internal/pose/review_closeout.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/review_closeout.go)
- **E13** — [pose-mcp/internal/pose/readiness.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/readiness.go)
- **E14** — [pose-mcp/internal/pose/techdebt_coverage_test.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/techdebt_coverage_test.go)
- **E15** — [pose-mcp/internal/pose/followup_candidates_test.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/pose/followup_candidates_test.go)
- **E16** — [pose-mcp/internal/cli/followups_owner_test.go](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/pose-mcp/internal/cli/followups_owner_test.go)
- **E17** — [.pose/specs/2026-10-04-pose-open-backlog-reconciliation.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/specs/2026-10-04-pose-open-backlog-reconciliation.md)
- **E18** — [.pose/reports/2026-10-pose-open-backlog-reconciliation.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/reports/2026-10-pose-open-backlog-reconciliation.md)
- **E19** — [.pose/results/pose-open-backlog-reconciliation.json](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/results/pose-open-backlog-reconciliation.json)
- **E20** — [.pose/specs/2026-10-04-pose-followup-reconciliation-candidates.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/specs/2026-10-04-pose-followup-reconciliation-candidates.md)
- **E21** — [.pose/specs/2026-08-07-pose-governance-gate-activation/spec.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/specs/2026-08-07-pose-governance-gate-activation/spec.md)
- **E22** — [.pose/specs/2026-08-07-pose-release-cycle-debt-closure/spec.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/specs/2026-08-07-pose-release-cycle-debt-closure/spec.md)
- **E23** — [.pose/specs/2026-08-03-pose-debt-marker-lexical-precision.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/specs/2026-08-03-pose-debt-marker-lexical-precision.md)
- **E24** — [.pose/specs/2026-08-23-pose-covered-followup-anchor-verification.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/specs/2026-08-23-pose-covered-followup-anchor-verification.md)
- **E25** — [.pose/specs/2026-10-05-pose-followup-dispositions-accept-qualified-refs.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/specs/2026-10-05-pose-followup-dispositions-accept-qualified-refs.md)
- **E26** — [.pose/specs/2026-07-18-pose-followup-ownership-sla.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/specs/2026-07-18-pose-followup-ownership-sla.md)
- **E27** — [.pose/adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md)
- **E28** — [.pose/reports/2026-10-03-pose-consolidated-analysis.md](https://github.com/oseiaspereira88/pose/blob/61c2909949fe50d2d3ca47c3c380aae688cba078/.pose/reports/2026-10-03-pose-consolidated-analysis.md)
