# Revisão independente — b28 / PR #196

Decisão: **changes requested**. Há **8 bloqueantes de merge (medium)** e **3 follow-ups aceitáveis (low)**. Nenhum selo, atestação, closeout, commit ou push foi realizado por esta revisão.

## Identidade, escopo e estabilidade

- Revisor declarado: `agent:independent-gpt-6.1-sol-review`, execução Codex/OpenAI; implementador Claude/Anthropic. Separação de vendor declarada, sem prova criptográfica de identidade ou independência cognitiva.
- HEAD revisado: `cc590314f844db9074a3414d13df0a3ac6ae144f`; merge-base contra main: `1e0a313ba1d87cebd4a355eeb3b83c0bd080e5e8`; delta b28: dois commits, 28 arquivos. Ambos os commits carregam `POSE-Spec: pose-distributed-instructions-parity` em bloco contíguo.
- O slug solicitado `pose-cli-suggestion-and-dry-run-accuracy` não existe nesse HEAD. Spec, trailers e corpo da PR #196 identificam `pose-distributed-instructions-parity`. Nenhum closeout foi tentado para qualquer slug.
- O workspace compartilhado começou com conflito em generation.json e avançou por ações externas durante a revisão. Os checks relevantes usaram cópia isolada fixada em cc590314 e binário compilado desse HEAD. Este parecer **não aprova os commits posteriores**. Nenhuma alteração externa foi revertida.
- Policy e issuer pins permaneceram intactos. No pose-dist a identidade é declarada e as atestações são sem envelope; como há bloqueantes, nenhuma foi escrita.
- Evidência compacta: [b28-independent-review.json](../results/b28-independent-review.json). Saídas integrais temporárias têm caminhos e SHA-256 nessa evidência; dados das reproduções são sintéticos.

## Rules applied during review

- Tipo: bugfix misto com documentação e produtor de evidência.
- Skills: `.agents/skills/pose-review/SKILL.md`, `.agents/skills/pose-spec-closeout/SKILL.md`; workflow `.pose/workflows/review.md`; manual `POSE.md`.
- `.pose/rules/security.md`: inputs, identidade de regra, paths e integridade da evidência.
- `.pose/rules/backend-go.md`: erros de I/O, consumidores Go e contrato CLI/MCP.
- `.pose/rules/documentation-style.md`: instruções, mirrors EN/pt-BR e mensagens correntes.
- `.pose/rules/delivery-evidence.md`: diferenciar resultado mecânico de correção e impedir fechamento sem evidência válida.
- `.pose/rules/knowledge-governance.md`: consultada para decisões anteriores; nenhum risco bloqueante foi aceito, nem handoff de aprovação criado.
- Regras frontend/Kubernetes de implementação não se aplicam: esta PR não altera essas aplicações ou manifests. A descoberta da regra Kubernetes faz parte do contrato revisado.
- Conhecimento consultado: `module-metadata-discovery-invalidates-review-provenance` e `review-assurance-requires-cryptographic-proof`. Escopo e semântica das atestações anteriores rvb-6167567bf3fa6ac4 / rva-5358c584e2a8bd4f conferidos, sem reutilizar seus julgamentos.

## Plano e execução dos tools

- Plano efetivo: `spec-closeout@1`, backend-review, engineering-judgment, high-criticality-review e structural-materiality; 15 critérios, 12 entradas de tool; banda critical e separação different-actor.
- Plan digest: `sha256:4483bc4b7473f4cb3f1bc520ebbf74f7c3efd49ae41d8dcfdda836ee194ec207`.
- Bundle somente preparado: `rvb-dd3eb3e34d53cc45`, digest `sha256:dd3eb3e34d53cc455fafaf9f71b2fe7699ecf8d4239ac31a4d7cfa06cee8c7b7`. Não selado e não gravado como aprovação. Avisos de owner ausente em pose-mcp e paths documentais não mapeados foram mantidos.
- Required ativos: validate strict nos dois módulos Go (cobertura da matriz por módulo; não alegar execução do comando global), artifact-check strict, skills-check strict e surface-check strict executados. Surface-check concluiu com zero findings após disponibilizar resultados novos na cópia isolada.
- Recommended executados: suggest review para pose-mcp, assess design, assess discover --if-stale --component pose-mcp, assess integrate e assess tech-debt. Nenhum recommended omitido.
- Completion-only: review-check, closeout-check, auto-attest, attest, verify e close deliberadamente não executados, pois o usuário proíbe selar/atestar com bloqueantes. Não alterar policy para contornar o resultado.

## Checks e limites

- `validate --strict --module pose-mcp`: **136/136 pass**, zero falhas/erros/skips; inclui build, suíte Go completa, vet, testes de integração, race exigido pela matriz e gate novo da b28.
- `validate --strict --module mcp-enforce`: **3/3 pass**. Total por união das matrizes: **139/139**.
- `surface-check --spec pose-distributed-instructions-parity --strict`: **pass, zero findings**, 136 resultados atuais, após copiar o run exato para o `results_path` da policy na cópia isolada. A primeira execução usou o arquivo antigo e recusou stale-evidence/unconsumed-capability; esse arquivo não foi substituído no workspace original.
- Testes direcionados de CLI/domínio: suggest, migrate, help, instruções distribuídas e install passaram.
- `check --strict`: SUCCESS com 29 warnings existentes; `skills-check --strict`: 22 skills, zero erros/warnings; `lint-spec --strict`: zero specs falhas, aviso de seção opcional Tasks.
- `artifact-check --spec pose-distributed-instructions-parity --strict`: reconcilia 23 claims/28 observações, sem erro bloqueante. Os cinco paths extras são índices derivados; 635 warnings globais de orphan não são 635 defeitos novos desta PR.
- `assess design`: 20/20 arquivos semânticos observados, três deltas (produtor de validação e duas instruções governadas); decisão D1 é proporcional, mas suas garantias são incompletas pelos casos abaixo.
- `assess integrate`: 64 contratos, 63 consumidores MCP não observados; não confundir observação estática ausente com regressão nova. Store.Suggest/handler MCP e GetRule foram conferidos diretamente.
- `assess tech-debt`: zero marcadores; recurrence-check tolerant, 14 dias: 393 registros, zero flags; nenhuma escalação por threshold.
- Install **e update reais**, EN e pt-BR, passaram; 36 arquivos Markdown por instância sem discovery incondicional. Nenhuma policy foi adotada pelo revisor.
- 12/12 mutações do implementador reexecutadas em TMPDIR normal; B8 mostra por que esse número sozinho não prova morte por asserção.
- Nenhuma dependência foi adicionada. Revisão manual do diff não encontrou segredo. Scanners dedicados govulncheck/gitleaks/golangci-lint não estavam disponíveis; não se alega execução deles.

## Achados classificados — lista consolidada

A classificação bloqueante inclui lacunas preexistentes que o escopo R3/R4/R6 promete corrigir; cada item explicita origem e consumidores vizinhos. Não se exige tratar toda dívida histórica do repositório como regressão desta PR.

### B1 — Módulo raiz perde regras de domínio

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:188`; requisitos: R3/R6/R8.
- Reprodução/evidência: Após install, update e index reais em EN e pt-BR, repo-map declara language=go com path vazio. suggest feature --path main.go sai 0 e omite backend-go. A condição p != vazio exclui o módulo raiz; path ponto e prefixos não normalizados também não casam.
- Ação: Normalizar ambos os caminhos; reconhecer raiz como fallback de menor especificidade; preservar fronteiras e módulo mais específico.
- Varredura de classe nos vizinhos/origem: Predicado preexistente, não resolvido pelo escopo que promete inferência correta. Vizinhos: produtor scanModules converte ponto para vazio; review_plan também exclui raiz como componente e documenta a limitação. Controle svc-other não casa com svc; svc/web vence svc.

### B2 — Kubernetes existente no catálogo é anunciado sem extensão conhecida

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:250`; requisitos: R4/R6/R8.
- Reprodução/evidência: Sem kubernetes.md, --domain kubernetes produz unavailable, sem next_step; humano afirma no extension known. extensions/pose-rule-kubernetes/extension.json existe e AGENTS documenta seu install.
- Ação: Reconciliar catálogo completo de arquivos/extensões, incluindo Kubernetes, sem instalação automática.
- Varredura de classe nos vizinhos/origem: Novo consumidor inverte mapa do doctor incompleto. Conferidos todos os 12 pares existentes, manifests de extensions e resolução/install; nenhum segundo arquivo duplicado no mapa.

### B3 — Stat não prova disponibilidade e confunde ausência com falha de acesso

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:258`; requisitos: R3/R4/R6.
- Reprodução/evidência: Diretório backend-go.md e arquivo 0000 são installed; GetRule não os lê. Diretório .pose/rules sem permissão transforma regra existente em extension-needed. Diretório svc/go.mod também converte backend Python em Go.
- Ação: Separar inexistência de erro de I/O; verificar arquivo regular e legibilidade; não oferecer instalação como remediação de perda de acesso. Aplicar a mesma disciplina ao sinal go.mod.
- Varredura de classe nos vizinhos/origem: Helper compartilhado ruleExtensionInstalled já usava apenas Stat no doctor; nova chamada o amplia a todas as regras. Vizinhos GetRule/ListRules recusam diretórios; requireTaskRef também apenas Stat; novo sinal go.mod repete o problema.

### B4 — Configuração de regras malformada é descartada com sucesso

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:86`; requisitos: R4/R6/R8.
- Reprodução/evidência: rules_by_domain.backend-go igual a string ou lista [17] resulta em domínio conhecido, exit 0, apenas regras base e nenhum diagnóstico; rules com elemento numérico é filtrado e task null também sai 0.
- Ação: Validar shape e tipos antes de montar a trilha; recusar configuração inválida com diagnóstico explícito, sem converter falha em conjunto vazio.
- Varredura de classe nos vizinhos/origem: Coerção stringSlice preexistente herdada pelos novos statuses. Vizinhos checkTaskMap tipa rules base, mas não declara/valida rules_by_domain; Store.Suggest e MCP retransmitem o resultado.

### B5 — Identidade de regra sem validação anuncia caminho recusado pelo consumidor

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:254`; requisitos: R4/R6/R7.
- Reprodução/evidência: Com rules=[../../outside] e outside.md na raiz, sai 0 com availability installed e path .pose/rules/../../outside.md; GetRule recusa o mesmo identificador por ValidateName.
- Ação: Validar identificadores de regras antes de resolver paths; preservar confinamento e a mesma identidade aceita por GetRule. Não tratar arquivo arbitrário como regra instalada.
- Varredura de classe nos vizinhos/origem: Nova classificação installed reutiliza string bruta. Vizinhos Store.GetRule/GetWorkflow/GetSkill validam nome; checkTaskMap verifica confinamento no projeto para regras base, mas não identidade de regra nem rules_by_domain. Não houve leitura/exfiltração de conteúdo pela sonda.

### B6 — Índice perdido ou inválido vira ausência silenciosa de domínio

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/suggest.go:173`; requisitos: R3/R6/R7.
- Reprodução/evidência: Mesmo path e task: mapa ausente, JSON malformado, geração inválida e mapa válido sem match retornam JSON idêntico, exit 0 e stderr vazio, sem domain_source/status. Um mapa inválido que continha módulo Go perde suas regras como se não houvesse módulo.
- Ação: Preservar source/estado e distinguir no-match de índice indisponível; propagar erro ou diagnóstico inconclusivo acionável antes de oferecer trilha parcial.
- Varredura de classe nos vizinhos/origem: Fallback preexistente não resolve o contrato de diagnóstico da b28. Vizinhos ReadIndex preserva erro tipado e task-map inválido já falha; MCP herda o silêncio. InventoryLegacy também suprime alguns erros de policy/supplements: fora do delta semântico R5 e sem atribuir resolução a esta PR.

### B7 — Inventário de instruções aprova varredura incompleta

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `pose-mcp/internal/cli/distributed_instructions_parity_test.go:33`; requisitos: R1/R2/R8.
- Reprodução/evidência: Em cópia exata do HEAD, arquivo com pose assess discover sem if-stale sob subdiretório 0000 faz TestDistributedInstructionsAskForDiscoveryOnlyWhenStale passar; chmod 0700 faz o mesmo teste falhar na instrução.
- Ação: Falhar em erros de WalkDir/ReadFile e em raízes esperadas ausentes; não substituir cobertura por contagem mínima global. Manter exclusões explícitas quando necessárias.
- Varredura de classe nos vizinhos/origem: Código novo. Vizinhos teste de install ignora erros tanto de walk quanto de leitura; AGENTS ausente causa Skip, e os limiares 20/10 não detectam perda parcial de inventário. Teste de igualdade dos quatro mirrors usa leitura fatal corretamente.

### B8 — Executor de mutações registra falha de shell como mutação morta

- Severidade: **medium**; classe: **bloqueante de merge**.
- Local: `.pose/results/b28-mutations.py:48`; requisitos: R8.
- Reprodução/evidência: TMPDIR com espaços gera cd: too many arguments, exit 2 e nenhum go test executado; o script registra killed=1,total=1. Captura observacional de subprocess.run preservou execução real e comprovou o erro.
- Ação: Executar argv com cwd, sem shell interpolado; distinguir infraestrutura/build de falha da asserção esperada, preferindo go test -json e eventos do teste-alvo; não contar erro desconhecido como killed.
- Varredura de classe nos vizinhos/origem: Código novo. Vizinhos b14-mutations.py e b15-mutations.py repetem shell=True e a mesma heurística de duas substrings. Executor nativo validate usa program+args+cwd e falha em exit não-zero. As 12 mutações em TMPDIR normal foram reexecutadas e morreram, mas esse resultado não cobre este controle adversarial.

### F1 — Saída humana não divulga alias/origem explícitos

- Severidade: **low**; classe: **follow-up aceitável**.
- Local: `pose-mcp/internal/cli/suggest.go:125`; requisitos: R3/R6.
- Reprodução/evidência: --domain Go tem domain_input/domain_source no JSON, mas sem --path a saída humana não mostra domínio/alias; mostra as mesmas regras e disponibilidade.
- Ação: Mostrar domínio efetivo, input e source também em texto.
- Varredura de classe nos vizinhos/origem: Sem relaxar regras ou aprovação: diferença de explicação; ownership owner:@core crit:low review:2026-12-01.

### F2 — MCP recusa alias de caixa aceito pela CLI

- Severidade: **low**; classe: **follow-up aceitável**.
- Local: `pose-mcp/internal/pose/cli.go:34`; requisitos: R3/R6.
- Reprodução/evidência: CLI --domain Go normaliza; Store.Suggest/pose_suggest rejeita Go por ValidateName antes de delegar. Chamada real do adaptador reproduziu o erro.
- Ação: Normalizar o alias aceito antes de validar, mantendo bloqueio de travessia; testar ambas as superfícies.
- Varredura de classe nos vizinhos/origem: Recusa explícita e conservadora, sem produzir resultado indevidamente aprovado; aliases minúsculos funcionam. ownership owner:@core crit:low review:2026-12-01.

### F3 — Limitação do dry-run ainda fala em 6.x

- Severidade: **low**; classe: **follow-up aceitável**.
- Local: `pose-mcp/internal/pose/legacy_inventory.go:49`; requisitos: R5.
- Reprodução/evidência: migrate v7 --dry-run --json em fixture retorna no apply exists in 6.x, enquanto help e recusa de --apply foram atualizados e motor é 7.4.1-dev.
- Ação: Alinhar limitations e comentários correntes ao fato sem versão: este motor não oferece apply.
- Varredura de classe nos vizinhos/origem: O comando continua dry-run, writes vazio e apply recusado; texto não concede capacidade nem aprovação. ownership owner:@core crit:low review:2026-12-01.

## Varredura adicional sem achado

Ordenação foi conferida em suggest, catálogo invertido, seleção de módulos e inventário migrate: tipos são ordenados para apresentação, os pares arquivo/extensão existentes são únicos, prefixos aninhados usam comprimento e os detalhes de migrate usam ordenação textual apenas para exibição. Não há critério semântico de versão/data sendo decidido por essa ordenação no delta. Fronteiras svc/svc-other e seleção de svc/web foram exercitadas como controles. Um arquivo .go dentro de backend híbrido, isoladamente, não foi considerado prova de classificação incorreta; B3 usa o sinal falso inequívoco de go.mod ser um diretório.

## Critérios, contratos e conclusão operacional

| Critério | Resultado desta revisão |
|---|---|
| assumption-integrity | Não demonstrado: presença de path/Stat e exit não-zero são usados como provas mais fortes do que fornecem (B3/B7/B8). |
| backend-concurrency | Sem alteração concorrente de produção; checks race registrados passaram. |
| backend-contracts / compatibility | Incompleto: CLI, identidade GetRule, paths raiz e diagnóstico de configuração divergem (B1–B6). |
| backend-input-errors / backend-observability | Findings B3–B6: perda de acesso/configuração vira resultado parcial bem-sucedido. |
| backend-integration-impact | MCP herda resultados da CLI e ainda valida aliases de forma diferente (F2); catálogo Kubernetes diverge (B2). |
| correctness / requirements | Findings B1–B8; checks verdes não bastam para afirmar R2/R3/R4/R6/R8 satisfeitos. |
| design-causality | D1 corretamente reutiliza taxonomia/freshness, mas não sustenta as garantias nos casos adversariais. Nenhuma aprovação/mapping gravado. |
| documentation | Mirrors e freshness verificados; F1/F3 pendentes e alegações de disponibilidade exigem corrigir B1–B6. |
| operability | Findings B2/B3/B6: próximos passos ausentes ou incorretos e erro sem causa. |
| scope | Delta da b28 conferido; slug do pedido divergente explicitado e não fechado. |
| security | B5 anuncia regra que a API recusa; nenhum segredo ou mudança de authority/policy introduzido. |
| solution-proportionality | Correções locais em validação de inputs, resolução de regra/path e produtor de mutações são suficientes; não é necessário novo mecanismo de governança. |

O delta de instruções não acrescenta gate de aprovação, mas acrescenta um produtor determinístico usado como prova de entrega. O atalho mais barato para satisfazê-lo em forma é esconder parte do inventário (B7) ou provocar falha de shell antes de qualquer teste (B8). As sondagens adversariais reproduzem ambos; o corpus existente não impede esses atalhos. Corrigir e incorporar os controles antes de considerar o gate evidência suficiente.

Não há mudança de API HTTP, eventos, schema público ou pin de issuer. Os campos novos de suggest são aditivos no JSON, mas seus estados e a paridade com consumidores precisam das correções listadas. Migrate continua recusando apply; nenhuma migração irreversível foi tentada. Rollback operacional é retornar ao binário anterior, preservando os artefatos; esta revisão não executou deploy.

Follow-ups F1/F2/F3 são conservadores: diferença de explicação, recusa explícita e texto residual; não permitem registrar uma aprovação inválida. Não foram escritos na spec porque há bloqueantes e o pedido autoriza apenas parecer/evidência neste ramo do fluxo. Caso sejam mantidos após remediação, registrar cada um `[open]` com `(owner:@core crit:low review:2026-12-01)`.

Não foi staged contribuição upstream: Contributor Mode exige consentimento explícito do usuário, que não foi concedido para essa ação. B8 registra a recorrência adjacente sem criar draft nem issue.

**Próxima revisão:** corrigir B1–B8, repetir as sondagens e checks em HEAD fixado, reconciliar o slug e os commits posteriores do workspace e somente então preparar evidência fresca no results_path da policy, selar, atestar sem envelope, verificar e fechar. Nenhuma aprovação desta execução pode ser reutilizada: ela não existe.
