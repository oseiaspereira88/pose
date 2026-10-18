# Revisão independente B06 — rodada 8

Revisão do HEAD `67f25e791eb5be71e4ef1c87a379a74b24f0570a`, PR #194, branch `impl/pose-b06-source-coverage`, em 2026-10-18. Parecer técnico favorável: nenhum novo bloqueante de código encontrado. O maintainer esclareceu que o pose-dist usa atestações sem envelope. A atestação `rva-eb2a3da043e02b89` foi registrada com identidade e separação declaradas; review verify confirmou fresh/approved e o close nativo concluiu a spec. Nenhuma policy foi alterada.

Revisor: `agent:independent-gpt-6.1-sol-review`, autorizado pelo maintainer. O prefixo `agent:independent-` é uma declaração exigida pelo motor até a spec `pose-delegated-review-run-provenance` substituí-lo. Esta execução usa Codex/OpenAI, outro vendor em relação ao implementador Claude/Anthropic, sessão `296d1fcb-e989-40ae-9362-cf3514db8b26`. A identidade é declarada; uma assinatura do payload não autentica a independência cognitiva do revisor.

## Revalidação e achados

| Item anterior | Resultado desta rodada | Classificação atual |
| --- | --- | --- |
| R7-F01 | Os 11 campos consumidos do header passam por detecção de duplicatas, inclusive schema_version, profile, reviewer, supersedes e evidence_refs. Controles negativos e ablações passam. | Corrigido |
| R7-F02 | A atribuição global só aceita scope sem repetição. Scope ambíguo continua perda sem atribuição e não certifica ausência de finding no escopo original. | Corrigido |
| R7-F03 | A publicação usa os resultados observados e incorpora somente estados consolidados ilegíveis como perdas conservadoras. O teste existente de componente removido e a suíte completa passam. | Corrigido |
| R7-F04 | O resumo de perda de estado ainda omite arquivo/tipo, mas conserva unavailable e não publica métricas como certificadas. | Follow-up aceitável, low |
| R7-F05 | Mutante equivalente removido; corpus atual 115/115 morto por asserção. Ablação suplementar retira realmente a validação do loader e faz o teste reprovar. | Resolvido |
| R6-F04 | Link quebrado na raiz de estados ainda perde diagnóstico; estado permanece unavailable e discover falha antes de publicar. | Follow-up aceitável, low |

Não há novos findings nesta rodada. R6-F04 e R7-F04 estão registrados na spec com owner `@core`, criticidade low e revisão em 2026-12-01. O cenário de R6-F04 envolve `updateProjectStateArchitecture`; o de R7-F04 envolve a redução de falhas do loader a contagem em `pose-mcp/internal/cli/assess.go:732`. Ambos afetam precisão de diagnóstico (R1/R8), sem caminho demonstrado para aprovação indevida. A evidência compacta conserva as adjudicações anteriores, inclusive os 11 achados da primeira rodada que haviam sido parcialmente corrigidos.

## Superfície e contrato

Foram revistos o diff integral desde `0b2a726f`, o delta da rodada, R1–R9, V1–V6, D1, ADR e relatórios front01/front02/front05, além dos leitores e consumidores de fontes governadas em pose-mcp. A busca abrangeu specs, roadmaps, knowledge, reports, reviews, bundles, attestations, journals, ledgers, docs-review, capability, releases, markers, tech-debt, assessments, state, Attention, doctor, check, suggest, stats e MCP. As listas de leitores e consumidores estão fora do repositório, com hashes na evidência.

CompleteFor/LossesFor preservam perdas sem escopo como potencialmente relevantes e isolam perdas comprovadamente alheias. Caminho, timestamp do nome, instantes com offset e prefixos estruturais foram confrontados com duplicatas, JSON truncado e conteúdo de distração. Um controle adicional confirma que um registro inválido com scope único de outro alvo não bloqueia a aprovação válida local. As perdas relevantes impedem retenção indevida de Approved/Fresh/Current, seleção de histórico e closeout; perdas comprovadamente antigas apenas avisam. Os diagnósticos orientam restaurar ou remover o artefato perdido.

O scan distingue perdas de exclusões declaradas (tamanho, binário, gerado, diretórios e links). O scanner sobre Content preserva linhas/ocorrências em CRLF, última linha sem quebra e Rust, inclusive após linhas longas. A projeção de incompletude chega a discover, dívida, MCP, estado, Attention e obligations; o agregador recebe os slots dos produtores após sua conclusão. Não foi observado consumidor alterado que promovesse incomplete a current.

D1 e o ADR descrevem honestamente a atribuição de melhor esforço, sem prometer reconstrução de conteúdo perdido. O tratamento de bundle ilegível divulga a perda e impede retenção quando a ordem/relevância não pode ser provada. Journals de action requests permanecem na B13; a migração histórica do ledger e o resíduo B22–B25/B34 não equivalem a certificar leituras perdidas. A varredura não encontrou outro leitor dentro do escopo declarado com perda silenciosa que justificasse novo bloqueante.

Campos JSON são aditivos; a projeção de bullets sem disposição como followup-undisposed altera contagens/Attention/pose state para consumidores como Harne8, Conductor e docs-site. A recusa de --fail-overdue quando a leitura está incompleta também é observável. A classificação minor é adequada; não se declara compatibilidade comportamental absoluta. Os quatro espelhos POSE foram comparados: os pt-BR coincidem; a diferença EN corresponde ao Contributor Mode local.

## Verificação independente

No HEAD exato, passaram vet, suíte completa sem cache, race dos três pacotes exigidos, builds Windows/Darwin, artifact-check, assess integrate, assess tech-debt, assess design, discover, surface-check e jornada install-and-upgrade. Os probes front05 coincidem com os versionados, descontado o hash do binário.

Os 83 testes selecionados dos oito arquivos de cobertura passaram sem skips. As 115 mutações únicas morreram por asserção, sem contar erro de build/setup/panic como sucesso. Cinco ablações suplementares cobrem lacunas de sensibilidade, incluindo retirada real da validação do loader. Cada um dos 83 testes tem uma falha por asserção correspondente no mapa de sensibilidade. O driver original também foi executado diretamente em um controle; a cópia instrumentada apenas registra stdout e nome do mutante, sem mudar mutações, seleção ou veredito.

`validate --strict`: **139/139**, nenhum skip/falha, resultado canônico em `.pose/results/delivery-validation.json`. O validate específico de pose-mcp exigido pelo plano também passou: **136/136**. `lint-spec --strict`: exit 0. `check --strict`: exit 0, 29 avisos. As saídas integrais ficam em `/home/go/.cache/b06-independent-review-r8`; a evidência compacta registra comandos, resultados e SHA-256, sem copiar stdout volumoso.

## Closeout nativo concluído

Bundle selado: **`rvb-d4e49da25f48b9ae`**. Digest `sha256:d4e49da25f48b9aeb8556a64b5067f6ba1c028a272506e490a0bb31c01ba831b`; plano `sha256:f6db4d3e39dbd3fca9afbadd1c49312998bcec9d1c303bab22c4cbdc8b783425`. Foram executadas as ferramentas exigidas, inclusive surface-check. Há disposições propostas para 15 critérios e 11 entradas de ferramenta, com validate global e por componente; review-check e closeout-check foram executados e ficam deferred por ausência da própria atestação final. A causalidade de design relaciona SD-b48e04cc ao requisito D1.

A prévia completa sem envelope passou (exit 0) e foi aplicada por instrução expressa
do maintainer, que corrigiu a exigência anterior de assinatura: harne8-agents vale
somente nos closeouts do próprio Harne8. A recusa assinada anterior permanece
registrada nas saídas históricas; ela não foi contornada e não exigiu mudar policy.
Atestação `rva-eb2a3da043e02b89`, sem envelope, com identidade e separação de vendors
declaradas; não se alega autenticação de independência cognitiva.

`review verify` e `review-check` passaram; `close` nativo concluiu status done.
Tasks e Requirement trace foram atualizados com produtores realmente executados,
Final Report registra revisão e riscos; B06 delivery_complete e SHA-256 da spec
foram registrados no implementation-plan e nos dois roadmaps. Requirements e
Technical Plan permaneceram intactos. R6-F04 e R7-F04 continuam open.

A indexação final passou. Após as edições, review verify permaneceu closed,
fresh=true e approved=true; closeout-check confirmou terminal=true e zero blockers.
Lint estrito da spec e check estrito finais passaram (exit 0). Este conjunto de
artefatos constitui o commit de closeout para impl/pose-b06-source-coverage; a
publicação não faz merge. Os resultados e hashes estão na evidência compacta.

Handoff dos riscos aceitos: [B06](../knowledge/2026-10-18-handoff-pose-b06-source-coverage-closeout.md).

A agregação global de follow-ups divulga perdas em duas specs antigas de outro escopo;
essa incompletude não foi convertida em ausência nem em blocker global desta aprovação.
Os dois follow-ups B06 foram confirmados como open mesmo após status done.
