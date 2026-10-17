# Review independente — B15 — rodada 3

Data: 2026-10-17. **Decisão: approved.** HEAD revisado `ff50b4eba9f1df07e0ca3f49b88162647b65c78d`, PR #193, branch `impl/pose-b15-index-generation`. F01–F16 tratados na causa; nenhum finding novo. Closeout nativo aplicado. Merge permanece com o maintainer.

## Escopo, autoridade e método

Revisor autorizado: `agent:independent-gpt-6.1-sol-review`. Implementador informado: `agent:claude-sonnet-5-5`, sessão `296d1fcb-e989-40ae-9362-cf3514db8b26`. O prefixo `agent:independent-` é uma declaração exigida pelo motor até `pose-delegated-review-run-provenance` substituí-lo. Esta execução é Codex/OpenAI, outro vendor em relação ao implementador Claude/Anthropic. Identidade declarada; no pose-dist as atestações não são assinadas. Nenhum subagente usado.

Review do diff integral `18e4342b..ff50b4eb`, incluindo os deltas corretivos das duas rodadas anteriores. Leitura de AGENTS.md, POSE.md (review/closeout), workflow review, skills pose-review/pose-spec-closeout, R1–R8/V1–V6/D1, ADR B15, ADR B14, CONS-05 e knowledge pertinente. Rodadas 1 e 2 preservadas no histórico e referenciadas por hash na evidência compacta. A CLI desta branch foi compilada pelo comando solicitado e usada como `../pose-review`; nenhum seletor POSE_PROJECT_ROOT/ROOTS/DEFAULT_PROJECT_ID exportado. A árvore começou limpa. Nenhum código de produção, teste, policy, capability ou adoção alterado pelo revisor.

## Rules applied during review

- Change type: bugfix de integridade/concorrência, com contrato de arquivos e documentação.
- Workflow: `.pose/workflows/review.md`; skills `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`.
- `.pose/rules/backend-go.md`: protocolo de publicação, concorrência, propriedade da trava, recuperação e efeitos parciais.
- `.pose/rules/security.md`: confinamento da retenção, digests, symlinks e ausência de segredos.
- `.pose/rules/documentation-style.md`: spec, ADR e quatro manuais root/dist EN/pt-BR.
- `.pose/rules/delivery-evidence.md`: evidência executada, provenance atual e reconciliação de artefatos.
- `.pose/rules/knowledge-governance.md`: preservação das rodadas anteriores e limites já documentados, sem criar backlog ou aceitar defeito bloqueador.
- Frontend, Kubernetes e demais stacks: não aplicáveis ao delta.

## Findings e reconciliação causal

Nenhum finding novo nesta rodada. F01–F16 encerrados no protocolo e escopo declarados; nenhum risco de severidade média ou maior foi aceito para obter aprovação.

| Findings anteriores | Correção reavaliada e evidência |
|---|---|
| F01, F05, F06 | Trava exclusiva do predecessor até retenção, fast path com a própria view, contenção sem espera; testes de publishers concorrentes, geração viva e recuperação de dono morto. |
| F02, F12 | `DescriptorBytes` conserva os bytes exatos da view; `basedOn` recusa antes das escritas e vincula o CAS final. Probe independente publica B válido fora da trava após leitura de A e obtém conflito sem reversão de B. |
| F03, F11, F15 | Pin na entrada do bundle, review-check e start; plano respeita pin recebido. Pin legacy recusa marcador/descritor e leitura legacy é revalidada após ler. Probes independentes cobrem transição legacy interrompida e concluída, janela pós-leitura e bundle com publicação entre plano e evidência. |
| F04, F14 | Pending antes de mudar qualquer nome legado na primeira publicação; `ReplaceIndexOrLegacy` mantém decisão e escrita sob a mesma trava. Probe de refresh versus primeira publicação obtém Busy e o retry fica current sem Diverged. |
| F07 | Publicação/retenção recusam `generations/` como link; filhos precisam ser diretórios diretos `gen-<16 hex>`, sem links; testes de entradas alheias e symlinks. |
| F08 | Decoder exige EOF após o único valor; teste e mutação de trailing bytes. |
| F09, F13 | Receipt.Committed é consumido pela CLI e refresh inclusive na via de erro; probe de sync após rename confirma geração efetiva nomeada, sem alegar ausência de efeito. |
| F10 | Falha de retenção é warning no refresh e não desfaz o commit; regressão e mutação específicas. |
| F16 | Quatro arquivos do pin declarados nos Artifacts; `artifact-check --strict` passa. Não houve redução artificial de range ou bypass de atribuição. |

## Protocolo, leitores e escritores

O ID é derivado dos digests/tamanhos dos sete outputs e dos cinco bindings declarados, sem timestamp. Publicação usa cópias imutáveis `ExpectAbsent`, verifica leitura de volta, publica nomes legados `ExpectAny` e por último o descritor autoritativo sobre os bytes lidos. Antes do commit, o leitor conserva A pelas cópias; sem elas, uma divergência é unavailable. Na primeira publicação o marcador impede que mistura pareça legacy. Depois do commit, o leitor valida B; uma mudança de descritor durante a leitura causa nova tentativa limitada. O pin mantém os bytes já copiados durante a operação. Retry da mesma intenção converge; nomes legados divergentes são reparados, e Unchanged só ocorre com bytes correntes também nos legados.

No caminho com hard link, a trava registra dono antes de tornar-se visível, nunca espera nem toma posse por idade; dono desconhecido ou possivelmente vivo é recusado. Recuperação revalida o mesmo dono morto sob flock do diretório; liberação só remove os bytes da própria posse. Publish/Replace e fallback legacy partilham essa exclusão. O CAS B14 continua obrigatório contra escritores que ignorem a trava, com os limites herdados de compare/rename e filesystem declarados, sem alegar transação contra um escritor hostil arbitrário.

Busca dos sete nomes e `.pose/indexes` incluiu Go, scripts, workflows, skills, docs-site e CI. Leitores funcionais do motor usam ReadIndex ou um Store pinado. `doctor` usa existência para saúde de instalação; medições/arquivamento de scripts e CI não dão verdict de coerência do conjunto. Não foi encontrado consumidor funcional interno omitido que produza mistura. Código externo de Harne8/check-pose-mirror-parity não está neste clone e não foi certificado. O limite de leitores externos que abrem nomes legados diretamente é explícito.

Refresh e bootstrap de review bundle preservam modo legacy sob trava, recusam unavailable e substituem current como nova geração. Install/update sincronizam conteúdo distribuído e placeholders antes de `cmdIndex`; descritor, cópias e arquivos ocultos de runtime nunca são semeados/sincronizados. Fixtures atuais usam substituição de geração quando apropriado. A jornada independente de instalação/onboarding e upgrade de 6.2.0 passou, inclusive check estrito da instância atualizada. Compatibilidade **minor**: índices gerados editados à mão ficam sombreados pela cópia imutável e verify os aponta; clone sem cópias verifica os legados e recusa divergência. Manuais EN/pt-BR root/dist concordam; a seção Contributor Mode é propriedade da instância EN.

## Qualificação e limites

Vet, suíte completa sem cache de testes, race em pose/cli/mcpserver, builds Windows/Darwin e jornada real passaram. As 54 mutações versionadas foram reexecutadas em cópias e mortas. Um segundo passe registrou os testes individuais que falham; a cobertura de contrafactuais e sua limitação estão no mapa externo. Alguns mutants provocam falha no setup, portanto a contagem não prova todos os escalonamentos. Um contrafactual adicional de remoção destrutiva do descritor fez falhar a asserção de preservação do registro anterior. Seis probes independentes das causas F11–F15 passaram. Dois hooks de agendamento adicionais existiram somente na cópia descartável; fontes e patches foram arquivados externamente.

P01–P03 reproduzem o `after` versionado, normalizado apenas o sufixo aleatório de staging; cinco de cinco descritores anteriores preservados no output bloqueado. Integração da matriz testa V1–V6 com race. `artifact-check` reconcilia 37 claims, 52 observações e 17 commits atribuídos antes do closeout; warnings globais de órfãos não são bloqueadores da B15. O gate não é satisfeito só pela presença do descritor: corrupção, tails JSON, alteração de identidade, falta de arquivo, pending e transição legacy têm casos adversariais e mutações que derrubam a aceitação formal vazia.

D1 e ADR são adequados aos requisitos neste escopo: há commit de um conjunto validado e indisponibilidade explícita quando não pode ser lido. Não prometem atomicidade dos sete nomes legados, snapshot Git/source global, reader registry entre processos ou durabilidade arbitrária de power loss. Retenção combina contagem e carência, podendo manter mais que três gerações recentes. Depois de kill pós-commit, um pending residual pode ficar inerte com descritor válido; não passa a certificar mistura. Recuperação de dono morto foi exercitada em Linux; cross builds não qualificam recuperação real nas outras plataformas. B34/B35 e B41 continuam suas responsabilidades já declaradas, sem nova transferência silenciosa de backlog.

A falha global de surface-check foi reproduzida em worktree da main `18e4342b`: action-mismatch dos três `release_sufficiency_test.go` da spec `pose-release-shared-sufficiency-contract`, além dos warnings globais. A B15 inicialmente tinha evidence stale e unconsumed-capability com os resultados antigos; o validate canônico regenerou a provenance antes do selo. A aprovação exige o surface-check **da spec B15** verde, como o plano requer; não houve waiver de blocker desse escopo.

## Plano selado e closeout nativo

Bundle `rvb-68d1469c0679927b`, digest
`sha256:68d1469c0679927b040adb743d039e86ad0e8297236194419694193e543d4b57`;
plano `sha256:9668b168b34560575933d73cb6256845c755ff9a0b14591f47da13911eb01eb1`.
Atestação `rva-ce3732f9185f01c8`. O plano tem 15 critérios (7 mecânicos e
8 julgamentos), 11 entradas de ferramentas e um delta material, ADR
`SD-7d3d8212`, mapeado a D1. O assessment também observa o check da matriz,
que não é material no plano selado.

| Ferramenta do plano | Execução/disposição |
|---|---|
| artifact-check | strict, exit 0 |
| assess-design | subject selado, exit 0 |
| assess-discover / pose-mcp | if-stale, exit 0; byproducts externos/restaurados |
| assess-integrate | exit 0; providers/consumers inspecionados |
| assess-tech-debt | exit 0; sem dívida descoberta não coberta |
| suggest-review / pose-mcp | exit 0; regras aplicadas acima |
| validate / componente vazio | strict canônico 138/138, exit 0 |
| validate / pose-mcp | strict explícito 135/135, exit 0 |
| surface-check | strict da spec, 1 alvo, 138 resultados, 0 findings, exit 0 |
| review-check | deferred na atestação; executado após apply, fresh/approved, exit 0 |
| closeout-check | deferred na atestação; terminal após close, exit 0 |

Auto-attest foi somente preview e listou oito julgamentos pendentes; cada um foi
respondido explicitamente com conclusão específica desta spec e evidência selada.
A primeira prévia manual recusou o mapeamento não material da matriz; o plano
foi relido e apenas o delta material ADR foi mapeado. Não houve alteração de
policy nem contorno de blocker. Prévia explícita final e apply saíram com 0.
Attribution preparada/concluída/aplicada pelo revisor declarado; nenhuma
confirmação humana ou assinatura criptográfica foi inventada.

Review verify pré-close: ready-to-close, fresh=true, approved=true, exit 0.
Close nativo exit 0; spec done/completed_at 2026-10-17. Closeout-check pré-close
apontou apenas lifecycle ainda não done; depois confirmou terminal=true e
review_approved=true. Lint-spec e check estritos pós-close passaram. Warnings de
metadata owner ausente e paths documentais não mapeados permanecem visíveis, sem
elevar assurance declarada a independência criptograficamente verificada.

Tasks, Requirement trace com produtores reais e Final Report foram reconciliados;
Requirements e Technical Plan permaneceram byte a byte iguais. B15 delivery_complete
foi registrado no plano e nos dois roadmaps, sem encerrar o programa nem selecionar
nova implementação. O discovery obrigatório --update-state foi executado com seus
efeitos arquivados e restaurados; esses resultados não integram o commit.

O registro compacto [b15-independent-review.json](../results/b15-independent-review.json)
contém exits, fatos e hashes; saídas completas, scripts e patches estão fora do
repositório em `/home/go/.cache/b15-independent-review-r3`. Temporários grandes
foram removidos. A verificação final após index/documentação é registrada ali;
o hash do commit transportador e o resultado do push são entregues na resposta final.

## Follow-ups

Não há finding ativo nem novo follow-up a transferir. Permanecem os limites já
declarados para B34/B35 e B41; nenhuma disposição spawned, covered ou duplicate
foi escrita. Aguardar merge de #193 pelo maintainer e reavaliar readiness nativa
sem escolher automaticamente outra implementação.
