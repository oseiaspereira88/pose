# Review independente — B15 — rodada 2

Data: 2026-10-17. **Decisão: changes-requested.** Reexaminado o HEAD `d1e1347061508aeb1336b3ec8acaba00e2cf0812`, PR #193, branch `impl/pose-b15-index-generation`. Há **6 findings ativos: 3 high e 3 medium**, detalhados juntos abaixo. Nenhum código foi corrigido, nenhum selo/atestação/close aplicado e nenhum commit/push feito nesta rodada.

## Escopo, autoridade e método

Revisor: `agent:independent-gpt-6.1-sol-review`, autorizado pelo maintainer. Implementador informado: `agent:claude-sonnet-5-5`, sessão `296d1fcb-e989-40ae-9362-cf3514db8b26`. Julgamento `scope` / `assumption-integrity`: o prefixo `agent:independent-` é uma declaração exigida pelo motor até a spec `pose-delegated-review-run-provenance` substituí-lo. Esta execução usa Codex/OpenAI, outro vendor em relação ao implementador Claude/Anthropic. Identidade declarada; no pose-dist as atestações não são assinadas.

A rodada 1 está preservada no histórico Git e em cópias externas do relatório/JSON presentes no commit revisado; seus hashes são registrados na evidência compacta. Esta rodada revê todo o diff `18e4342b..d1e13470`, com atenção adicional ao delta desde `299fa696`, à trava, marcador, pin, escritores diretos, retenção e resultados com efeito parcial. A CLI foi compilada desta branch por `go -C pose-mcp build -o ../pose-review ./cmd/pose` e o resultado movido da raiz para `../pose-review`, usado nos comandos. Não foram exportados POSE_PROJECT_ROOT, POSE_PROJECT_ROOTS ou POSE_DEFAULT_PROJECT_ID. Nenhum subagente foi usado.

Foram relidos/consultados AGENTS.md, POSE.md nas seções de review/closeout, workflow review, skills pose-review/pose-spec-closeout, spec R1–R8/V1–V6/D1, ADR B15, ADR B14, CONS-05 da frente 09 e knowledge pertinente. A árvore começou limpa. Só este relatório e o JSON compacto serão atualizados; efeitos derivados de assess/artifact-check ficam arquivados externamente e são revertidos. Requirements, Technical Plan, testes, produção, policy, capabilities, adoção, plano B15 e roadmaps não foram alterados.

## Rules applied during review

- Change type: bugfix com contrato de arquivos, concorrência e documentação.
- Workflow consulted: `.pose/workflows/review.md`; skills `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`.
- `.pose/rules/backend-go.md`: propriedade da trava, commit, concorrência, leitores e efeitos parciais.
- `.pose/rules/security.md`: confinamento de retenção e integridade dos arquivos; nenhum segredo foi incorporado.
- `.pose/rules/documentation-style.md`: ADR, trace e quatro manuais root/dist EN/pt-BR.
- `.pose/rules/delivery-evidence.md`: testes verdes não substituem contraexemplos nem a reconciliação de artefatos.
- `.pose/rules/knowledge-governance.md`: decisões anteriores consultadas; nenhum risco bloqueador aceito nem handoff de aprovação criado.
- Regras de frontend, Kubernetes e stacks não alteradas: não aplicáveis.

## Disposição dos findings da rodada 1

| Finding | Disposição nesta rodada |
|---|---|
| F01 | Fechado para publishers cooperantes: aquisição exclusiva cobre predecessor, commit e retenção; contenção é Busy, sem espera. |
| F02 | Parcial: writers cooperantes são excluídos, mas a substituição ainda não commita sobre o descritor da view que leu; F12. |
| F03 | Parcial: o plano sobre geração current usa um pin, mas legacy e preparação do bundle ainda misturam; F11/F15. |
| F04 | Parcial: Store fresco recusa pending, mas um pin legacy e o fallback do escritor escapam à transição; F11/F14. |
| F05 | Fechado no caminho cooperante: fast path usa ID da própria view sob trava. |
| F06 | Fechado no caminho cooperante: retenção serializada e proteção do descritor vivo além do argumento. |
| F07 | Fechado: publicação e retenção recusam symlink na base generations, além dos links filhos. |
| F08 | Fechado: parser exige EOF após um único valor JSON; tails inválidos são recusados. |
| F09 | Parcial: cmdIndex informa Committed, mas refreshDeliveryIndex ainda nega o efeito publicado; F13. |
| F10 | Fechado: refresh torna CleanupError observável sem converter commit em falha. |

## Todos os findings ativos

### B15-F11 — high — pose-mcp/internal/pose/index_generation.go:402

PinIndexes tomado enquanto o repositório é legacy guarda apenas o status, não os arquivos. ReadIndex do pin legacy lê nomes vivos e ignora o marcador e o descritor. Uma primeira publicação interrompida depois de services deixa esse pin ler arquivos de A e B sem erro, embora um Store fresco retorne unavailable. Um review context real resultou em owner/criticality de B e observed paths de A sem warning/blocker.

Requisitos/gates afetados: R3, R5, R6, R8.

Evidência: TestR2LegacyPinBypassesPendingMarker e TestR2LegacyReviewContextMixesDespitePending, usando apenas hooks existentes na cópia; nenhum hook novo de produção.

Ação esperada: Fixar uma view legacy consistente ou detectar a transição para pending/descriptor e recusar/repetir a operação inteira. Não permitir que o pin legacy contorne a proteção do primeiro commit. Revalidar também a janela do leitor legacy sem pin entre a checagem do marcador e a leitura do arquivo.

### B15-F12 — high — pose-mcp/internal/pose/index_generation.go:583

ReplaceIndex lê a view A na linha 561, mas publishGenerationLocked relê o predecessor na linha 475. Um writer que não toma a trava pode publicar B e terminar entre essas leituras. A substituição aceita B como predecessor e publica seis arquivos/bindings antigos de A com sucesso, revertendo B. O CAS não está vinculado ao descritor que forneceu os arquivos, contrariando a correção declarada e o ADR. Não é a janela compare/rename residual da B14: B termina antes da nova leitura do helper.

Requisitos/gates afetados: R2, R4, R6.

Evidência: TestR2ReplaceIndexDoesNotCASTheViewItRead; replace-after-read existente publica um conjunto B completo e um descritor válido pelo publisher B14, sem tomar a trava; receipt.Previous=B e arquivos não substituídos=A.

Ação esperada: Transmitir/verificar no commit os bytes exatos do descritor da view usada na substituição. Se mudarem, recusar ou reconstruir a operação, mesmo quando o writer externo ignora a trava.

### B15-F13 — medium — pose-mcp/internal/cli/index.go:360

A CLI index agora usa Receipt.Committed, mas refreshDeliveryIndex ainda ignora esse campo quando err != nil. Se o dir sync do descritor falha após rename, a nova geração já está corrente, mas o refresh informa que o índice não foi atualizado. Afeta o refresh de artifact-check e o bootstrap de review bundle e preserva a afirmação de ausência de efeito que F09 pretendia eliminar.

Requisitos/gates afetados: R4, R6, R8.

Evidência: TestR2RefreshDeniesItsCommittedEffect; PublishSyncHook existente falha somente no sync de diretório de generation.json. Geração corrente muda, mas stderr contém the delivery index was not refreshed.

Ação esperada: Usar o receipt também na via de erro do refresh e distinguir publicado com durabilidade não confirmada de falha sem efeito, nomeando a geração efetiva.

### B15-F14 — medium — pose-mcp/internal/cli/index.go:358

ReplaceIndex encontra legacy, devolve ErrIndexGenerationLegacy e solta a trava; o fallback os.WriteFile ocorre fora dela. Nesse intervalo um index cooperante pode adquirir a mesma trava e publicar uma geração completa. O refresh então sobrescreve o delivery legado com o grafo anterior e ambos os comandos têm sucesso, deixando Diverged=delivery-integrity.json. Um clone sem cópias imutáveis não consegue consumir o conjunto descrito. A trava nova não coordena esse escritor direto durante a migração.

Requisitos/gates afetados: R2, R5, R6, R8.

Evidência: TestR2LegacyRefreshEscapesGenerationLock; um único hook de agendamento inserido apenas na cópia antes do fallback legacy executa cmdIndex completo, depois libera a escrita antiga. O refresh não emite erro e a geração fica divergente.

Ação esperada: Coordenar a decisão e a escrita legacy na mesma região protegida, reavaliando o modo caso uma geração tenha sido publicada. Cobrir artifact-check/review bundle disputando a primeira publicação com index.

### B15-F15 — high — pose-mcp/internal/pose/review_bundle.go:487

O pin está limitado à cópia local de Store dentro de ReviewPlan. prepareReviewBundle usa o Store original: guarda o plano A da linha 469 e lê delivery-integrity novamente na linha 487. Uma publicação cooperante de B nesse intervalo produz bundle com component owner @old de A e resultado de integração check-only-in-b de B, sem blockers. Fixar só o plano não fixa os sete índices consumidos pela operação de preparação do bundle.

Requisitos/gates afetados: R3, R6, R8.

Evidência: TestR2BundleCombinesPlanAWithEvidenceB; fixture de bundle já usada pelo projeto e IndexReadFault existente; B é publicado na segunda leitura, sem qualquer hook novo. Bundle preparado mistura plano A e evidência B com zero blockers.

Ação esperada: Fixar a geração na fronteira da operação de bundle e compartilhá-la em todos os seus leitores, fazendo ReviewPlan respeitar um pin recebido. Isto não exige fixar Git, policies ou uma read view global B34/B35.

### B15-F16 — medium — pose-mcp/internal/pose/spec.go:58

A rodada adiciona indexPin ao Store em spec.go, mas esse caminho não foi declarado nos Artifacts da spec. O artifact-check nativo resolve corretamente 14 commits, 33 claims e 49 observações e falha com error undeclared pose-mcp/internal/pose/spec.go. Há um blocker obrigatório de reconciliação de escopo apesar dos 138 checks de validate passarem.

Requisitos/gates afetados: AGENTS.md: mandatory artifacts and applicable checks, review criterion scope, native artifact-check --strict.

Evidência: ../pose-review artifact-check --spec pose-index-generation-integrity --strict, exit 1; artifact-check.log. Os 14 commits carregam o slug, portanto não é um falso negativo do parser de trailers.

Ação esperada: Reconciliar o artefato de produção pelo mecanismo de plano/amendment aplicável e reexecutar artifact-check. O revisor não altera o Technical Plan nem reduz o change set para ocultar o caminho.

## Trava, recuperação, publicação e leitores

A aquisição usa publishStartLock: arquivo temporário exclusivo com owner preenchido antes do hard link para .generation.lock, fallback O_EXCL onde links não funcionam. A trava existente não é rompida por idade; owner active/unknown produz Busy. Um owner provadamente morto é revalidado sob flock não bloqueante do diretório de índices antes da remoção. Disputa pela recuperação termina em Busy; a nova aquisição continua exclusiva. A liberação confere os bytes do próprio owner antes de remover o arquivo. Não há espera nem inversão de ordem com a trava de lifecycle de specs: a geração é uma região local de escrita; os testes concorrentes e os probes de kill real qualificam isso no Linux.

Os probes adicionais mataram processos reais em after-legacy-file:services.json, before-commit e após o rename do descritor (antes do dir sync), aguardaram a saída e repetiram a mesma intenção. A trava morta foi recuperada, o descritor convergiu e a trava foi liberada. Após kill pós-commit, o marker pode permanecer no fast path Unchanged; com descritor válido ele é inerte e não altera current. Isso foi registrado como resíduo de crash, sem inventar uma certificação de power loss ou exigir pin interprocesso. Fora do Linux o motor não prova owner morto automaticamente; os builds Windows/Darwin qualificam compilação, não recuperação de processo nessas plataformas.

O ID continua sendo o digest da lista fixa de sete arquivos e bindings. As fases continuam ExpectAbsent para cópias imutáveis, ExpectAny para nomes legados, CAS authoritative do descritor por último. A trava impede duas aceitações concorrentes do predecessor por escritores cooperantes e protege a retenção. O CAS B14 ainda tem sua janela documentada entre compare e rename para um escritor externo; não se promete excluí-la. F12 ocorre antes dessa janela: a view A é esquecida quando o helper relê B. O ADR deve ser lido com esse limite, não como garantia de que qualquer writer que ignore a trava será recusado em todo schedule.

A preferência imutável, leitura de clones sem generations, status Diverged, Unchanged, schema/EOF e compaction continuam qualificados nos testes. O marcador torna unavailable uma primeira publicação interrompida para leitores frescos. Um pin current mantém os bytes daquela geração. F11 mostra que um pin legacy não mantém bytes nem revalida a transição; F15 mostra que a preparação do bundle não compartilha o pin do plano. São operações sobre os mesmos sete índices B15, não uma exigência de snapshot global Git/policy ou do daemon B34/B35.

A busca integral por .pose/indexes, joins e os sete nomes inclui doctor, check, state, report, release, MCP, scripts, workflows, skills, docs-site e CI. Não foi encontrado outro leitor funcional que contorne ReadIndex. O escritor direto do fallback legacy permanece, conforme F14. Os consumidores de Store podem, contudo, compor leituras de gerações distintas: F15 mostra por que uma busca textual por ReadIndex não basta. doctor usa existência para diagnóstico de instalação, não para certificar a geração. scripts/bench-governance.sh mede bytes e restaura índices; CI arquiva projeções e docs-site descreve os caches. Nenhum deles foi tratado como prova de coerência. A implementação de check-pose-mirror-parity/Harne8 não está neste clone; não foi executada nem certificada.

Scaffold não sincroniza/semeia generation.json, generations ou arquivos ocultos de runtime. O install/update continua semeando placeholders e chamando index, qualificado pela jornada real. Os quatro manuais root/dist EN/pt-BR documentam preferência imutável, edição manual e clone sem cópias. A classificação de compatibilidade continua minor, condicionada às correções: índices são projeções geradas; editar o legado não muda o motor enquanto a cópia existe, e --verify acusa a diferença. O fallback concorrente F14 é um defeito dessa migração, não uma limitação aceitável de leitores externos.

D1 e o desenho de descritor mais cópias e trava são proporcionais. Não é necessário daemon nem transação entre todos os arquivos para corrigir os contraexemplos. Continuam honestos os limites declarados de Git não fixado, cinco bindings governados (sem completude global de dependências), leitores externos dos nomes legados, carência/contagem sem registro de readers e power loss além do fsync. Esses limites não autorizam misturar índices dentro de um plano/bundle nem trocar o predecessor da view de uma substituição.

## Validação e força da evidência

- go vet ./..., go test ./... -count=1, go test -race em internal/pose/internal/cli/internal/mcpserver e builds GOOS windows/darwin: exit 0.
- Driver original de mutações reexecutado: **49/49 killed**, sem build-error/survived. A segunda execução ampliou a seleção aos quatro pacotes focados e preservou saídas JSON individuais.
- Mapa de contrafactuais: 43 testes novos passam no código revisado; 42 falham sob pelo menos uma das 49 mutações. O teste de preservação do descritor restante falha na asserção com uma mutação adicional de cleanup destrutivo. Algumas falhas de mutantes são falhas de setup (segundo CAS), e o controle de bytes já copiados não prova retenção de reader em andamento; esses limites estão no mapa externo. Nenhuma contagem foi usada como prova de cobertura de todos os interleavings.
- Probes P01–P03: mesmos exits/estados funcionais do after versionado; cinco descritores anteriores preservados nos cinco bloqueios, divergência/repair manual e controle corrente equivalentes. Paths temporários e stdout incidental não foram usados para igualdade byte a byte.
- Jornada direta com a CLI revisada: instalação, onboarding, upgrade de 6.2.0 e resposta assinada à adoção passaram; a release anterior foi escolhida pela mesma regra usando gh API autenticada. A validação usou GH_TOKEN somente no próprio comando, sem imprimir o token.
- Validate nativo --strict: **138/138 pass**, zero failed/errored/skipped. Resumo derivado por jq de JSON integral externo. B15 integration executou com race.
- Artifact-check --strict: **exit 1**, 33 claims / 49 observações / 14 commits; erro undeclared spec.go. Os 14 commits carregam POSE-Spec pelo scanner do motor, confirmado sem usar o parser Git de trailers.
- Surface-check --strict com os 138 resultados desta rodada: **exit 1**, um alvo, um finding undeclared spec.go. É o mesmo blocker F16, não uma segunda contagem.
- Lint-spec --strict e check --strict: exit 0 (check mantém warnings globais). Review verify: exit 1, ready-to-seal, fresh=false, approved=false.
- Assess tech-debt, integrate, discover --if-stale, design, suggest review e recurrence-check foram executados e suas saídas/hashes preservados. Assessment genérico de consumers não observados não foi convertido em finding B15. Não foi feito assess discover --update-state de closeout.

Os testes/probes independentes foram executados em cópia descartável. F11/F12/F13/F15 usam hooks já existentes; F14 usa um único hook adicional de agendamento antes da escrita legacy, somente na cópia. PASS nesses probes significa que o defeito foi reproduzido. Fontes, patch do hook, resultados integrais, diff, mapa de testes e manifesto SHA-256 ficam em /home/go/.cache/b15-independent-review-r2. As cópias e temporários grandes são removidos; nenhum conteúdo de saída integral é copiado para o JSON do repositório.

## Julgamentos R1–R8 e V1–V6

R1 está definido e o parser foi corrigido. R2/R4 ainda falham na vinculação da substituição e na fronteira do fallback (F12/F14), e o resultado pós-commit do refresh é incorreto (F13). R3/R5/R6 não estão demonstrados pelas leituras legacy e pela composição do bundle (F11/F15). R7 ficou qualificado no protocolo cooperante e na base sem links, respeitados os limites de plataforma/retenção declarados. R8 continua incompleto contra os schedules novos apesar da suíte verde. V1/V3/V5 precisam abranger leitor já pinado e escritor fallback na primeira geração; V2 precisa abranger bundle inteiro e substituição sob writer externo antes do CAS; V3 precisa resultado correto do refresh pós-commit. V4 EOF e V6 recuperação/retencão ganharam evidência relevante. A reconciliação de artefatos ainda bloqueia a entrega (F16).

Julgamentos de correctness, requirements, backend-concurrency/integration, operability, compatibility e scope não podem ser aprovados com esses contraexemplos. O menor modo de passar só em forma seria citar 138/138 e 49/49 sem mapear os schedules e sem executar artifact-check/surface-check. Esses dois gates recusaram F16; os probes desta revisão detectaram o que a suíte não cobria. Não se alegou que o corpus adversarial de governança demonstrava propriedades de geração que ele não testa.

## Fluxo nativo e próximos passos

Bundle apenas preparado: `rvb-3d4fac8e420a843c`; digest `sha256:3d4fac8e420a843c70f21d57321feea0e69a82ff0ac0f75b8434624a36759bd8`; plan digest `sha256:1405f6e3840bfb5df903994d732efc8d6bb73402ffbc35b108903b36abcf1ce9`. Nenhum selo nem atestação aplicada. Review-check/closeout-check são ferramentas de completion e ficam diferidas; close não foi executado. Validate do componente é coberto pela execução global; não houve repetição isolada de --module depois da decisão negativa.

A spec permanece in-progress e não foi alterada nesta rodada. Task de revisão/closeout, Requirement trace, Final Report da spec, plano B15 e roadmaps não foram marcados completos. HEAD permanece d1e1347061508aeb1336b3ec8acaba00e2cf0812; nenhum commit foi enviado. O implementador deve corrigir os seis findings, acrescentar regressões dos cenários específicos e reconciliar spec.go pelo fluxo aplicável. Só depois cabe nova revisão independente e closeout nativo. Nenhum risco bloqueador foi aceito; nenhuma contribuição/issue/PR/comentário externo foi criado.
