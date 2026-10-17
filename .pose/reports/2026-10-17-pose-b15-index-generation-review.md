# Review independente — B15 / pose-index-generation-integrity

Data: 2026-10-17. Decisão: **changes-requested**. Há 5 findings high, 4 medium e 1 low; nenhum foi corrigido nesta revisão. Não selar, atestar como approved, fechar, commitar closeout ou publicar a branch neste estado.

## Escopo e independência

Revisor autorizado pelo maintainer: `agent:independent-gpt-6.1-sol-review` (Codex/OpenAI). Implementador informado: `agent:claude-sonnet-5-5`, sessão `296d1fcb-e989-40ae-9362-cf3514db8b26` (Claude/Anthropic). PR #193, branch `impl/pose-b15-index-generation`, base `18e4342b`, HEAD `299fa696e20c3fbc996d4cf022a7d29b6ebc9060`.

Julgamento `scope` / `assumption-integrity`: O prefixo agent:independent- é uma declaração exigida pelo motor até a spec pose-delegated-review-run-provenance substituí-lo. Esta execução usa Codex/OpenAI, outro vendor em relação ao implementador Claude/Anthropic. Identidade declarada; no pose-dist as atestações não são assinadas.

Foi usada a CLI compilada desta branch com `go -C pose-mcp build -o ../pose-review ./cmd/pose`; o binário resultante na raiz foi movido para `../pose-review`, conforme o caminho operacional solicitado. Não foram exportados POSE_PROJECT_ROOT, POSE_PROJECT_ROOTS ou POSE_DEFAULT_PROJECT_ID. Nenhum agente auxiliar foi usado. A árvore começou limpa. Só relatório e evidência compacta serão deixados no clone; os efeitos derivados dos comandos de avaliação/indexação foram preservados externamente e revertidos, sem alterar produção, testes, policy, capabilities, adoção, spec ou roadmaps.

## Rules applied during review

- Change type: bugfix com mudança de contrato de arquivos e documentação.
- Workflow consulted: `.pose/workflows/review.md`; skills `.agents/skills/pose-review/SKILL.md` e `.agents/skills/pose-spec-closeout/SKILL.md`; seções de review e closeout do POSE.md.
- `.pose/rules/backend-go.md`: publicação, concorrência, persistência e propagação de erros.
- `.pose/rules/security.md`: confinamento da retenção e exposição de dados; F07 viola o confinamento.
- `.pose/rules/documentation-style.md`: contrato, ADR e quatro manuais EN/pt-BR root/dist.
- `.pose/rules/delivery-evidence.md`: distinguir gates verdes de correção efetiva; não declarar closeout.
- `.pose/rules/knowledge-governance.md`: leitura de decisões anteriores; nenhum risco aceito nem handoff criado.
- Regras frontend, Kubernetes e demais stacks: não aplicáveis à alteração.

Foram consultados a spec R1–R8/V1–V6/D1, o ADR B15 de 2026-10-17, o ADR B14 de 2026-10-16, CONS-05 do relatório da frente 09, knowledge e o diff integral `18e4342b..HEAD`. Assessment tech-debt: zero markers; integrate: inventário de 64 integrações, 63 consumers não observados, sem atribuir esses gaps genéricos ao delta B15. Recurrence: zero chaves sinalizadas em 14 dias. Os oito commits atribuídos têm POSE-Spec; o artifact-check nativo resolve corretamente o range, sem depender do parser de trailers do Git.

## Findings completos

### B15-F01 — high — pose-mcp/internal/pose/index_generation.go:460

Dois PublishIndexGeneration leem o mesmo descritor e chegam juntos ao commit. Ambos retornam sucesso para IDs diferentes e ambos declaram o mesmo predecessor. PublishFile faz read/compare e rename separados; B15 não toma a trava por recurso que o contrato B14 exige para coordenar esses passos.

Requisitos afetados: R2, R4, R8. Evidência: TestB15IndependentDescriptorCASIsNotExclusive; barreira no PublishFault existente, antes dos dois CAS; dupla aceitação observada na tentativa 0 da execução final.

Ação esperada: Coordenar leitura do predecessor, validação e commit de geração; provar que escritores concorrentes não podem ambos aceitar o mesmo predecessor. O teste atual só intercala um rival que termina antes do CAS, sem exercitar a janela compare/rename.

### B15-F02 — high — pose-mcp/internal/pose/index_generation.go:488

ReplaceIndex lê A; outro index publica B antes de ele chamar PublishIndexGeneration. Este relê B como predecessor, mas recebe seis arquivos e bindings copiados de A: publica com sucesso a substituição e reverte silenciosamente os demais índices de B para A. Atinge refresh de artifact-check e bootstrap de review bundle.

Requisitos afetados: R4, R6. Evidência: TestB15IndependentReplaceIndexLosesConcurrentGeneration; um hook de agendamento acrescentado SOMENTE na cópia descartável, imediatamente antes da chamada da linha 488, permite B terminar nesse intervalo.

Ação esperada: Vincular o commit da substituição ao descritor que forneceu os arquivos; recusar ou reconstruir a substituição quando esse descritor mudar.

### B15-F03 — high — pose-mcp/internal/pose/review_plan.go:691

Uma única resolveReviewPlanContext lê repo-map de A na linha 387/559, uma publicação troca para B, e loadObservedReviewPaths lê delivery-integrity de B na linha 453/691. Cada ReadIndex verifica seu próprio descritor, mas a operação não fixa uma geração. Probe retorna owner antigo/criticality low junto com observed:work:svc/new.go exclusivo de B, sem warning nem blocker.

Requisitos afetados: R3, R6, R8. Evidência: TestB15IndependentReviewPlanMixesGenerations; IndexReadFault existente publica B na segunda leitura; nenhuma alteração de código de produção necessária ao probe.

Ação esperada: Compartilhar a mesma view de geração entre os leitores da operação, ou conferir e repetir a operação inteira quando o descritor mudar. Isto é coerência dos sete índices dentro de uma operação B15, não a read view global futura B34/B35.

### B15-F04 — high — pose-mcp/internal/pose/index_generation.go:449

Na primeira migração de uma instância legacy, não existe generation.json. A publicação troca repo-map e services, depois falha/crasha. O descritor continua ausente e ReadIndex devolve repo-map novo e delivery-integrity antigo como legacy-unversioned, sem erro. GetDeliveryIntegrity descarta o status na linha 438 e os consumers continuam usando a mistura. A mesma janela existe enquanto a primeira publicação ainda está em curso.

Requisitos afetados: R2, R5, R8. Evidência: TestB15IndependentLegacyFirstFailureMixes; sete arquivos legacy antigos e falha em after-legacy-file:services.json, por hook existente; repo=new e delivery=old retornados sem erro.

Ação esperada: Tornar a transição legacy para generation-aware recuperável e distinguível antes de alterar nomes legados: preservar um conjunto anterior verificável ou marcar explicitamente a indisponibilidade. Manter a compatibilidade de instâncias realmente não migradas.

### B15-F05 — medium — pose-mcp/internal/pose/index_generation.go:422

O predecessor lido é A, que já tem nome legado divergente. Durante ReadIndexGeneration, B é publicado e a releitura retorna B íntegro. O fast path compara o ID desejado apenas com prev=A e usa status/Diverged de B, retornando Unchanged=true para A sem reparar a divergência inicial nem publicar A. Nenhum instante da chamada tinha A com nomes legados íntegros.

Requisitos afetados: R4, R6. Evidência: TestB15IndependentUnchangedCanNameAnotherGeneration; arquivo legado editado antes da chamada e IndexReadFault existente publicando B; receipt nomeia A/Unchanged enquanto o descritor corrente nomeia B.

Ação esperada: Exigir que a view usada pelo fast path corresponda ao mesmo descritor/predecessor e à geração pretendida; tratar troca concorrente como conflito/retry coerente.

### B15-F06 — medium — pose-mcp/internal/pose/index_generation.go:530

A publica e para antes de sua retenção; B publica depois. Quando A retoma, pruneIndexGenerations protege o argumento A, sem ler a geração viva B. Se B já passou da carência e há dois diretórios mais novos (inclusive staging de outros publishers), A remove a cópia imutável da geração que o descritor ainda nomeia. O fallback legado mascara a perda até uma publicação posterior modificar esses nomes.

Requisitos afetados: R4, R7, R8. Evidência: TestB15IndependentCleanupRemovesRivalsLiveGeneration; rival publicado no PublishSyncHook de diretório; os.Chtimes simula a suspensão além da carência, como os testes de retenção da branch; contagem e carência padrão preservadas.

Ação esperada: Coordenar retenção com publicadores e proteger o descritor efetivamente vivo no momento da remoção; cobrir a retomada de um cleanup antigo depois de outro commit.

### B15-F07 — high — pose-mcp/internal/pose/index_generation.go:517

O Lstat protege links filhos, mas os.ReadDir(base) segue um symlink no próprio generations/. Apontando generations para um diretório externo com quatro gen-<16 hex> antigos e dados, prune remove recursivamente dois desses diretórios externos. O reconhecimento de nome e o Lstat do filho não confinam a operação ao diretório real do projeto.

Requisitos afetados: R7. Evidência: TestB15IndependentCleanupFollowsBaseSymlink; quatro diretórios externos sintéticos, dois removidos; nenhuma alteração de código de produção no probe.

Ação esperada: Recusar ancestral/base symlink e confinar a enumeração/remoção à árvore de gerações pertencente ao projeto; testar o link na base, além dos links filhos já cobertos.

### B15-F08 — medium — pose-mcp/internal/pose/index_generation.go:273

Acrescentar ] após um descritor válido produz JSON inválido, mas dec.More retorna false para um delimitador de fechamento e o descritor é aceito como current. O parser não exige EOF após o valor e a classe corrupt deixa de produzir unavailable.

Requisitos afetados: R1, R3, R8. Evidência: TestB15IndependentDescriptorAcceptsTrailingCloseBracket; append de um único byte ] a generation.json; ReadIndexGeneration retorna current.

Ação esperada: Exigir EOF real depois de exatamente um valor JSON e cobrir delimitadores soltos e outros tails inválidos.

### B15-F09 — medium — pose-mcp/internal/cli/index.go:159

Quando o sync do diretório falha depois do rename do descritor, PublishFile retorna ErrPublishUnsynced com efeito já publicado. A CLI devolve publish-unsynced, mas acrescenta incondicionalmente que a geração anterior continua em vigor; também sai antes de registrar os campos de geração do receipt. O probe mostra o novo descritor corrente enquanto stderr afirma o anterior.

Requisitos afetados: R4, R6, R8. Evidência: TestB15IndependentUnsyncedCommitClaimsPreviousStillCurrent; PublishSyncHook existente falha somente no dir sync de generation.json após uma primeira geração válida.

Ação esperada: Distinguir falha sem commit de efeito publicado com durabilidade não confirmada e expor a geração efetiva no resultado; não prometer rollback para ErrPublishUnsynced.

### B15-F10 — low — pose-mcp/internal/cli/index.go:339

refreshDeliveryIndex descarta o receipt inteiro de ReplaceIndex. Se a publicação termina mas a retenção não consegue remover uma geração antiga, CleanupError é perdido e nada é emitido em stderr, diferentemente do caminho pose index. Atinge artifact-check e bootstrap do bundle.

Requisitos afetados: R4, R6. Evidência: TestB15IndependentRefreshDropsCleanupWarning; diretório antigo com descendente sem permissão de unlink, publicação concluída, arquivo protegido remanescente e stderr vazio.

Ação esperada: Propagar CleanupError como warning sem converter a publicação efetiva em falha.

## Protocolo, leitores e compatibilidade

O ID é gen- mais 16 hex do SHA-256 da lista ordenada de digests/tamanhos dos sete outputs e dos bindings. As fases usam ExpectAbsent para cópias imutáveis, ExpectAny para nomes legados e ExpectDigest/ExpectAbsent authoritative para o descritor. Com uma geração prévia disponível, o leitor de conjunto normalmente vê seus bytes imutáveis durante as duas primeiras fases; sem cópias, só aceita nomes legados que conferem, ou retorna unavailable. Após o commit pode ler a nova geração. A releitura limitada do descritor evita que uma única chamada ReadIndexGeneration devolva seus arquivos misturados. Isso não estende a fixação a duas chamadas ReadIndex feitas por uma operação (F03), nem torna o compare/rename exclusivo (F01).

Os casos normais de retry e de saída bloqueada passam; a primeira migração legacy (F04), os efeitos pós-commit (F09) e os schedules concorrentes F01/F02/F05/F06 não estão resolvidos. Unchanged não pode usar o estado/Diverged de outra geração. A preferência da cópia imutável tem o custo documentado: editar o arquivo gerado legado não afeta o motor enquanto a cópia existir; --verify acusa divergência. Um clone sem generations confere o arquivo legado e recusa o digest divergente.

Foi feita busca em todo o clone, incluindo arquivos ocultos, pelos sete nomes, .pose/indexes e joins de indexes. Os leitores funcionais encontrados são GetDeliveryIntegrity/SpecFacts/MCP, review-plan (mapa e caminhos), snapshot de obrigações e suggest, migrados para ReadIndex. O problema de composição F03 permanece apesar de não haver bypass textual. doctor usa existência de arquivos como diagnóstico de instalação; check/state/report/release consomem fontes canônicas ou Store, sem outro parser legado dos sete arquivos identificado. review_bundle.go classifica caminhos, sem ler os sete como uma operação coerente. O bootstrap ainda usa Stat do nome legado para decidir se refresca; o refresh se recusa a construir sobre unavailable. Em legacy escreve o arquivo como antes e em current usa ReplaceIndex, sujeito a F02 e F10.

Scaffold IsIncluded e NeutralIndexTemplates excluem generation.json e generations; install/update semeiam placeholders ausentes e chamam cmdIndex conforme os testes e as jornadas. Os quatro manuais explicam a preferência imutável, divergência e clone sem cópias; o upgrade real de 6.2.0 passa check --strict. Não foi encontrada divergência adicional entre os quatro trechos B15.

scripts/bench-governance.sh mede bytes do nome legado e restaura os índices com Git; não produz verdict de coerência dos sete. CI arquiva spec-graph e roadmaps como evidência; docs-site descreve projeções, sem parser do conjunto encontrado. check-pose-mirror-parity/Harne8 não têm implementação neste clone, portanto não houve execução ou alegação de certificação desses consumidores externos. Quem lê os nomes legados diretamente continua exposto à janela de mistura declarada no ADR; essa limitação externa não justifica F03/F04 em consumidores internos.

Classificação de compatibilidade: **minor**, condicionada à correção dos findings. Os índices são projeções geradas, os schemas de compaction/provenance existentes são preservados e a mudança de edição manual foi explicitamente documentada e migrada nas fixtures. Nenhuma compatibilidade externa não testada foi presumida. Não foi alterada a classificação breaking nem qualquer policy.

D1 e o ADR escolhem uma estrutura proporcional: descritor mais cópias, sem daemon, permite fixar somente a geração de índices. Os limites de Git não fixado, nomes legados externos, carência/contagem sem pin interprocesso e power loss além do fsync são declarados e aceitáveis como limites, sem prometer uma read view global. A contagem não é um teto absoluto durante a carência. A checagem de freshness cobre os cinco bindings explicitados; adicionar um go.mod pode mudar o mapa sem tornar esses bindings stale (probe binding-probe.log). Cobertura de todas as dependências fonte pertence à read view/dependency-completeness futura; não foi inventado um requisito de snapshot global para B15. Em contrapartida, declarar limites não autoriza remover a geração viva ou seguir links, e as afirmações de nenhum leitor interno misturar e de commits concorrentes respeitarem o predecessor são contraditas pelos probes desta revisão. D1 ainda não está demonstrada.

## Evidência e cobertura

`go vet ./...`, `go test ./... -count=1`, `go test -race` nos três pacotes solicitados e builds GOOS windows/darwin passaram. O driver original de 33 mutações foi reexecutado sem modificar o clone: 33/33 killed, zero build-error/survived. Uma segunda passagem executou os quatro pacotes focados com saída JSON para mapear falhas por teste: 29 dos 30 testes novos falham sob pelo menos uma dessas mutações. Uma mutação adicional de cleanup destrutivo do descritor faz o trigésimo falhar na asserção de preservação do descritor. Todos passam na implementação original.

Esse mapa não certifica cada propriedade isoladamente: a mutação que antecipa o commit também provoca um segundo CAS, e alguns testes falham no setup de publicação. O teste ReaderThatFixedAGenerationKeepsItsBytes é um controle de bytes já copiados, não um teste de reader ainda copiando durante retenção. Esses limites estão registrados em test-counterfactual-map.json; não se infere cobertura exaustiva de 33/33 killed. Os dez probes adicionais demonstram os dez findings. Nove usam os hooks já existentes/fixtures; F02 usa um hook adicional apenas de agendamento na cópia descartável, com patch integral preservado externamente. Os probes são assertivas de reprodução do defeito: PASS significa que o defeito descrito foi observado. Não foram adicionados ao código nem aos testes do PR.

P01–P03 têm os mesmos resultados funcionais do after versionado: cinco descritores anteriores preservados em cinco bloqueios, divergência dos legados detectada, edição manual detectada/reparada e controle corrente. A comparação ignora nomes temporários/host e não exige igualdade de saída incidental. Saídas integrais, fontes dos probes, patch do hook e mapa de testes estão fora do repositório, sob /home/go/.cache/b15-independent-review; os hashes constam na evidência compacta e no manifesto externo.

Validação nativa global: **138/138 pass**, zero failed/errored/skipped; JSON integral externo, resumo derivado por jq em validation-summary.json. O check index-generation-integrity-integration também rodou com race. A jornada direta usou o binário compilado e a release anterior 6.2.0, escolhida pela mesma regra da jornada a partir da API autenticada via gh: install, onboarding e upgrade passaram. A primeira tentativa absoluta havia falhado por rate limit anônimo; a execução final fixa --previous e o validate usou GH_TOKEN restrito ao próprio comando, sem registrar o token.

Native artifact-check --strict: exit 0 (33 claims, 46 observações; warnings globais de orphan não equivalem a blockers B15). surface-check --strict: exit 0. lint-spec --strict: exit 0. check --strict: exit 0. review verify: exit 1, ready-to-seal, fresh=false, approved=false. A validação global inclui pose-mcp; não foi repetido um validate --module isolado após a decisão negativa. review-check e closeout-check são ferramentas de completion e permanecem diferidas, assim como close.

## Estado do fluxo nativo

Bundle apenas preparado: `rvb-c3e35a09032444eb`; digest `sha256:c3e35a09032444ebc119e4d64863691b856f52d04be3922315fc1ceeac89a248`; plan digest `sha256:1405f6e3840bfb5df903994d732efc8d6bb73402ffbc35b108903b36abcf1ce9`. Não existe selo desta revisão nem atestação aplicada. `close` não foi executado. A spec permanece in-progress; task de review/closeout, Requirement trace, Final Report, plano B15 e roadmaps não foram indevidamente marcados completos. Nenhum commit ou push foi feito; HEAD permanece o implementado `299fa696e20c3fbc996d4cf022a7d29b6ebc9060`.

## Julgamentos e trace R/V

R1: identidade/conjunto definidos, parser integral falha em F08. R2: fases normais qualificadas, concorrência e migração inicial falham em F01/F04. R3: uma chamada de leitura de conjunto qualificada, operação review-plan falha em F03 e corrupção trailing em F08. R4: F01/F02/F05/F06/F09/F10 impedem concluir coordenação e receipts. R5: compaction e jornada de upgrade passam, mas F04 viola o fallback sem mistura silenciosa. R6: rotas migradas, mas composição/atualização/observabilidade falham em F02/F03/F05/F09/F10. R7: regras de nome, filhos e carência passam; geração viva e base symlink falham em F06/F07. R8: corpus verde não cobre os schedules reproduzidos. V1/V3 precisam incluir primeira publicação legacy e falha pós-commit; V2 precisa operação composta e dois writers na janela CAS; V4 precisa EOF real; V5 precisa migração interrompida; V6 precisa cleanup atrasado e base link.

Os julgamentos de correção, requisitos, concorrência, integração, segurança, operabilidade e documentação não podem ser aprovados contra esses contraexemplos. O menor modo de satisfazer o gate só em forma seria citar a suíte verde/33 mutações e declarar identidade independente; o motor registra evidência e identidade declarada, mas não substitui os probes de interleaving nem uma revisão semântica. Não se alegou que o corpus adversarial de governança cobria estes defeitos de índices. Escopo e arquitetura são proporcionais; o efeito causal de D1 não está provado enquanto persistirem os findings.

## Próximos passos

O implementador deve corrigir os findings, acrescentar regressões que falhem pelos comportamentos específicos descritos e solicitar nova execução independente sobre o HEAD corrigido. Só depois cabem validate/index/seal/attest/verify/close e a atualização dos artefatos operacionais. Não houve aceitação de risco para contornar esses blockers. Nenhum draft de contribuição, issue, novo PR ou comentário externo foi criado.
